package app

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helloxz/zpad/internal/draft"
	"github.com/helloxz/zpad/internal/encoding"
	"github.com/helloxz/zpad/internal/session"
	"github.com/helloxz/zpad/locales"
)

const (
	// draftIdle is how long after the last change a copy is written, so
	// that typing is not copied a keystroke at a time.
	draftIdle = time.Second
	// draftMax is the longest a change waits while the user keeps typing.
	draftMax = 5 * time.Second
	// draftTick is how often the copies are looked at, while the feature is
	// on.
	draftTick = time.Second
)

// draftJob is one document's copy, with the text and details it had when the
// job was made on the main thread.
type draftJob struct {
	doc   *Document
	entry draft.Entry
	snap  editor.Snapshot
	epoch uint64
}

// draftResult is what writing a job did, applied on the main thread.
type draftResult struct {
	doc     *Document
	epoch   uint64
	version uint64
	written bool
	err     error
}

// KeepUnsavedChanged turns the recovery copies on or off after the setting
// changed: on opens the store, writes the unsaved text already open and
// starts the loop; off stops the loop and deletes every copy.
func (a *App) keepUnsavedChanged() {
	if a.Settings.KeepUnsaved {
		if err := a.openDraftsStore(); err != nil {
			a.draftLog(err)
			a.Settings.KeepUnsaved = false
			a.saveSettings()
			a.SetErrorToast(locales.KeepUnsavedFailed(), err.Error())
			return
		}
		a.loadDraftRestoreIndex()
		a.writeDrafts(true)
		a.maybeWakeDraftLoop()
		return
	}
	a.stopDraftLoop()
	a.draftWrites.Wait()
	a.draftWriting = false
	for _, doc := range a.allDocs() {
		a.resetDraft(doc)
	}
	if a.drafts != nil {
		if err := a.drafts.Purge(); err != nil {
			a.draftLog(err)
		}
		a.drafts = nil
	}
	a.draftEntries = nil
	a.draftByPath = nil
	a.draftUsed = nil
}

// openDraftsStore opens the directory of recovery copies in the user data
// directory.
func (a *App) openDraftsStore() error {
	if a.drafts != nil {
		return nil
	}
	if a.SettingsDir == "" {
		return fmt.Errorf("no user data directory")
	}
	s, err := draft.Open(filepath.Join(a.SettingsDir, "drafts"))
	if err != nil {
		return err
	}
	a.drafts = s
	return nil
}

// startDraftLoop looks at the open documents once a second, while keeping
// unsaved content is on. It runs on the main thread and redraws nothing.
func (a *App) startDraftLoop() {
	if a.draftStop != nil || a.drafts == nil || a.win == nil {
		return
	}
	stop := make(chan struct{})
	a.draftStop = stop
	go func() {
		t := time.NewTicker(draftTick)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				mygo.RunOnMain(func() { a.draftTick() })
			}
		}
	}()
}

func (a *App) stopDraftLoop() {
	if a.draftStop == nil {
		return
	}
	close(a.draftStop)
	a.draftStop = nil
}

func (a *App) draftTick() {
	if a.drafts == nil || !a.Settings.KeepUnsaved {
		a.stopDraftLoop()
		return
	}
	a.writeDrafts(false)
	if !a.hasPendingDraftWork() {
		a.stopDraftLoop()
	}
}

// maybeWakeDraftLoop starts the 1s draft loop only when KeepUnsaved is on and
// at least one open document has unsaved changes not yet backed up (or an
// obsolete draft to clean up). While all documents are clean or already
// backed up, no background ticker wakes the main thread.
func (a *App) maybeWakeDraftLoop() {
	if a.draftStop != nil || a.drafts == nil || a.win == nil || !a.Settings.KeepUnsaved {
		return
	}
	if a.hasPendingDraftWork() {
		a.startDraftLoop()
	}
}

func (a *App) hasPendingDraftWork() bool {
	if a.draftWriting {
		return true
	}
	for _, p := range [2]*Pane{&a.left, &a.right} {
		for _, tab := range p.tabs {
			doc := tab.doc
			if !a.draftNeeded(doc) {
				if doc.draftExists {
					return true
				}
				continue
			}
			if !doc.draftExists || doc.draftBacked != doc.Buffer.Version() {
				return true
			}
		}
	}
	return false
}

// writeDrafts writes the copies that are due: a second after the last
// change, or after five seconds of changes. force writes every copy that is
// not up to date.
func (a *App) writeDrafts(force bool) {
	if a.drafts == nil || (a.draftWriting && !force) {
		return
	}
	jobs := a.collectDraftJobs(time.Now(), force)
	if len(jobs) == 0 {
		return
	}
	a.draftWriting = true
	store := a.drafts
	if a.win == nil {
		// No event loop to report back to, as under go test: write and
		// apply here.
		a.applyDraftResults(a.saveDraftJobs(store, jobs))
		a.draftWriting = false
		return
	}
	a.draftWrites.Add(1)
	go func() {
		defer a.draftWrites.Done()
		results := a.saveDraftJobs(store, jobs)
		a.onUI(func() {
			a.draftWriting = false
			a.applyDraftResults(results)
		})
	}()
}

// flushDrafts writes every copy that is not up to date, before the window
// closes or the app quits. It runs on the main thread and waits for a write
// on its way, so that no copy is left behind.
func (a *App) flushDrafts() {
	if a.drafts == nil {
		return
	}
	a.draftWrites.Wait()
	a.draftWriting = false
	jobs := a.collectDraftJobs(time.Now(), true)
	if len(jobs) == 0 {
		return
	}
	a.applyDraftResults(a.saveDraftJobs(a.drafts, jobs))
}

// collectDraftJobs notes the changes since the last tick and returns the
// documents whose copy is due. It runs on the main thread.
func (a *App) collectDraftJobs(now time.Time, force bool) []draftJob {
	var jobs []draftJob
	for _, doc := range a.allDocs() {
		if !a.draftNeeded(doc) {
			if doc.draftExists {
				a.forgetDraft(doc)
			}
			doc.draftSeenVersion = doc.Buffer.Version()
			doc.draftEditAt = time.Time{}
			doc.draftSince = time.Time{}
			continue
		}
		version := doc.Buffer.Version()
		if version != doc.draftSeenVersion {
			doc.draftSeenVersion = version
			doc.draftEditAt = now
			if doc.draftSince.IsZero() {
				doc.draftSince = now
			}
		}
		if doc.draftExists && doc.draftBacked == version {
			doc.draftEditAt = time.Time{}
			doc.draftSince = time.Time{}
			continue
		}
		if !force {
			due := !doc.draftEditAt.IsZero() && now.Sub(doc.draftEditAt) >= draftIdle
			if !doc.draftSince.IsZero() && now.Sub(doc.draftSince) >= draftMax {
				due = true
			}
			if !due {
				continue
			}
		}
		if doc.DraftID == "" {
			doc.DraftID = draft.NewID()
		}
		jobs = append(jobs, draftJob{
			doc:   doc,
			entry: a.draftEntry(doc),
			snap:  doc.Buffer.Snapshot(),
			epoch: doc.draftEpoch.Load(),
		})
	}
	return jobs
}

// draftNeeded reports whether a document's text belongs in a recovery copy:
// an unsaved change, or the text of an untitled document. A read-only
// document never changes.
func (a *App) draftNeeded(doc *Document) bool {
	if doc == nil || doc.ReadOnly {
		return false
	}
	if doc.Path == "" {
		return doc.Buffer.Len() > 0
	}
	return doc.Buffer.Modified()
}

func (a *App) draftEntry(doc *Document) draft.Entry {
	e := draft.Entry{
		ID:       doc.DraftID,
		Path:     doc.Path,
		Untitled: doc.UntitledSeq,
		Encoding: string(doc.Encoding),
		SavedAt:  time.Now().UnixNano(),
	}
	if doc.Path == "" && doc.Language != nil {
		e.Language = doc.Language.Name
	}
	if doc.Path != "" {
		e.Size = doc.Size
		if !doc.ModTime.IsZero() {
			e.ModTime = doc.ModTime.UnixNano()
		}
	}
	return e
}

// saveDraftJobs writes the copies on a goroutine. It touches only the
// buffers, which are safe there, and the store.
func (a *App) saveDraftJobs(store *draft.Store, jobs []draftJob) []draftResult {
	results := make([]draftResult, 0, len(jobs))
	for _, job := range jobs {
		epoch := job.epoch
		doc := job.doc
		written, err := store.Save(job.entry, func(w io.Writer) error {
			_, err := job.snap.WriteTo(w)
			return err
		}, func() bool {
			return doc.draftEpoch.Load() == epoch
		})
		results = append(results, draftResult{
			doc:     doc,
			epoch:   epoch,
			version: job.snap.Version(),
			written: written,
			err:     err,
		})
	}
	return results
}

func (a *App) applyDraftResults(results []draftResult) {
	for _, r := range results {
		if r.doc == nil || r.doc.draftEpoch.Load() != r.epoch {
			// Saved, discarded or renamed while the copy was on its way:
			// the document's state has already moved on.
			continue
		}
		switch {
		case r.err != nil:
			a.draftLog(r.err)
			// Try again on the next tick rather than waiting for an edit.
			r.doc.draftEditAt = time.Now()
			r.doc.draftSince = time.Now()
		case !r.written:
			// The store no longer wanted the copy.
		default:
			a.draftLogged = ""
			r.doc.draftExists = true
			r.doc.draftBacked = r.version
			if r.doc.Buffer.Version() == r.version {
				r.doc.draftEditAt = time.Time{}
				r.doc.draftSince = time.Time{}
			} else {
				// Edited while the copy was written: copy again from the
				// current text.
				r.doc.draftEditAt = time.Now()
				r.doc.draftSince = time.Now()
			}
		}
	}
	if a.hasPendingDraftWork() {
		a.maybeWakeDraftLoop()
	} else {
		a.stopDraftLoop()
	}
}

// forgetDraft drops a document's recovery copy: it was saved, discarded or
// closed for good.
func (a *App) forgetDraft(doc *Document) {
	if doc == nil {
		return
	}
	a.resetDraft(doc)
	if a.drafts == nil || doc.DraftID == "" {
		return
	}
	if err := a.drafts.Remove(doc.DraftID); err != nil {
		a.draftLog(err)
	}
}

// resetDraft forgets the state of a copy without touching the store, for
// the paths that already removed it or are about to drop the store.
func (a *App) resetDraft(doc *Document) {
	if doc == nil {
		return
	}
	doc.draftEpoch.Add(1)
	doc.draftBacked = 0
	doc.draftExists = false
	doc.draftSeenVersion = 0
	doc.draftEditAt = time.Time{}
	doc.draftSince = time.Time{}
}

// draftMoved follows a rename of the document's file, so that the copy
// still names the file the text goes back to.
func (a *App) draftMoved(doc *Document) {
	if a.drafts == nil || doc == nil || !doc.draftExists || doc.DraftID == "" {
		return
	}
	// A copy on its way has the old path: let it be skipped, and keep the
	// index the one that names the new path.
	doc.draftEpoch.Add(1)
	if err := a.drafts.SetPath(doc.DraftID, doc.Path); err != nil {
		a.draftLog(err)
	}
}

// draftLanguageChanged makes a document's copy due again after the language
// an untitled document was given changed: the index names the language, and
// the text did not change to say so.
func (a *App) draftLanguageChanged(doc *Document) {
	if a.drafts == nil || doc == nil || !a.draftNeeded(doc) {
		return
	}
	// A copy on its way carries the old language: let its result be
	// skipped, and write the index again from the state set here.
	doc.draftEpoch.Add(1)
	doc.draftBacked = 0
	doc.draftEditAt = time.Now()
	if doc.draftSince.IsZero() {
		doc.draftSince = doc.draftEditAt
	}
	a.maybeWakeDraftLoop()
}

func (a *App) draftLog(err error) {
	if err == nil {
		return
	}
	if a.draftLogged == err.Error() {
		return
	}
	a.draftLogged = err.Error()
	log.Printf("drafts: %v", err)
}

// loadDraftRestoreIndex reads the copies found at launch, for the restore.
func (a *App) loadDraftRestoreIndex() {
	a.draftEntries = map[string]draft.Entry{}
	a.draftByPath = map[string]string{}
	a.draftUsed = map[string]bool{}
	if a.drafts == nil {
		return
	}
	for _, e := range a.drafts.Entries() {
		a.draftEntries[e.ID] = e
		if e.Path != "" {
			if _, ok := a.draftByPath[e.Path]; !ok {
				a.draftByPath[e.Path] = e.ID
			}
		}
	}
}

// draftForTab finds the copy a session tab's document has, by id first and
// by path second, so that a tab written before the copy was made still finds
// it.
func (a *App) draftForTab(ts session.TabState) (draft.Entry, bool) {
	if a.drafts == nil {
		return draft.Entry{}, false
	}
	if ts.Draft != "" {
		if e, ok := a.draftEntries[ts.Draft]; ok && !a.draftUsed[e.ID] {
			return e, true
		}
	}
	if ts.Path != "" {
		if id, ok := a.draftByPath[ts.Path]; ok && !a.draftUsed[id] {
			return a.draftEntries[id], true
		}
	}
	return draft.Entry{}, false
}

// newDraftDocument opens a document from a recovery copy alone: its file is
// gone, or it never had one.
func newDraftDocument(e draft.Entry, text string) *Document {
	doc := &Document{
		ID:          e.Path,
		Path:        e.Path,
		Buffer:      editor.NewBuffer(""),
		Encoding:    encoding.Parse(e.Encoding),
		UntitledSeq: e.Untitled,
		DraftID:     e.ID,
		Size:        e.Size,
		draftExists: true,
	}
	if e.Path == "" {
		doc.ID = fmt.Sprintf("untitled-%d", e.Untitled)
	}
	// Set, not Reset, which would make the text look saved.
	doc.Buffer.Set(text)
	if e.Path != "" {
		doc.setLanguage(e.Path)
		if e.ModTime != 0 {
			doc.ModTime = time.Unix(0, e.ModTime)
		}
	} else if l := editor.LanguageNamed(e.Language); l != nil {
		// The language the status bar was given before the copy was made.
		doc.selectLanguage(l)
	}
	doc.draftBacked = doc.Buffer.Version()
	return doc
}

// attachDraft puts the text of a copy over the text a document was loaded
// with. It reports whether the text was kept; a copy that holds what the file
// already has is stale, and is dropped.
func (a *App) attachDraft(doc *Document, e draft.Entry) bool {
	text, err := a.drafts.Text(e.ID)
	if err != nil {
		a.draftLog(err)
		return false
	}
	if sameDraftText(doc, text) {
		if err := a.drafts.Remove(e.ID); err != nil {
			a.draftLog(err)
		}
		return false
	}
	// Reset first: Set over the file's text would keep the file's line
	// endings, and the copy's are the ones the user had. A copy of an
	// emptied file has no line endings to keep, and goes over the file's
	// text so that the empty text still counts as a change.
	if text == "" {
		doc.Buffer.Set("")
	} else {
		doc.Buffer.Reset("")
		doc.Buffer.Set(text)
	}
	doc.Encoding = encoding.Parse(e.Encoding)
	doc.DraftID = e.ID
	doc.Size = e.Size
	if e.ModTime != 0 {
		doc.ModTime = time.Unix(0, e.ModTime)
	}
	doc.draftExists = true
	doc.draftBacked = doc.Buffer.Version()
	return true
}

// sameDraftText reports whether a copy holds the text a document was loaded
// with: it is then the copy of a save that happened after it was written.
// Comparing chunk-by-chunk avoids copying the buffer or normalizing the whole
// draft in memory.
func sameDraftText(doc *Document, text string) bool {
	snap := doc.Buffer.Snapshot()
	if !strings.ContainsRune(text, '\r') {
		if len(text) != snap.Len() {
			return false
		}
		off := 0
		same := true
		snap.Chunks(0, snap.Len(), func(chunk string) bool {
			if text[off:off+len(chunk)] != chunk {
				same = false
				return false
			}
			off += len(chunk)
			return true
		})
		return same
	}
	i := 0
	same := true
	snap.Chunks(0, snap.Len(), func(chunk string) bool {
		for k := 0; k < len(chunk); k++ {
			if i >= len(text) {
				same = false
				return false
			}
			b := text[i]
			i++
			if b == '\r' {
				if i < len(text) && text[i] == '\n' {
					i++
				}
				b = '\n'
			}
			if b != chunk[k] {
				same = false
				return false
			}
		}
		return true
	})
	return same && i == len(text)
}

// restoreLeftoverDrafts opens the copies no session tab took: the app was
// started with restoring the session off, or the tab was opened after the
// last session save. They go to the left pane, without taking the active tab
// from the session.
func (a *App) restoreLeftoverDrafts(restored *int) {
	if a.drafts == nil {
		return
	}
	var left []draft.Entry
	for id, e := range a.draftEntries {
		if !a.draftUsed[id] {
			left = append(left, e)
		}
	}
	sort.Slice(left, func(i, j int) bool {
		if left[i].SavedAt != left[j].SavedAt {
			return left[i].SavedAt < left[j].SavedAt
		}
		return left[i].ID < left[j].ID
	})
	for _, e := range left {
		if e.Path != "" && a.findDocument(e.Path) != nil {
			continue
		}
		if e.Path == "" && e.Untitled <= 0 {
			a.untitledSeq++
			e.Untitled = a.untitledSeq
		}
		text, err := a.drafts.Text(e.ID)
		if err != nil {
			a.draftLog(err)
			continue
		}
		doc := newDraftDocument(e, text)
		a.appendDraftTab(doc)
		a.draftUsed[e.ID] = true
		*restored++
	}
}

// appendDraftTab adds a restored document to the left pane, after its tabs,
// leaving the active tab as it is when there is one.
func (a *App) appendDraftTab(doc *Document) {
	if doc.Path == "" && doc.UntitledSeq > a.untitledSeq {
		a.untitledSeq = doc.UntitledSeq
	}
	p := &a.left
	p.tabs = append(p.tabs, &Tab{
		doc:   doc,
		state: &editor.State{Buffer: doc.Buffer, ReadOnly: doc.ReadOnly},
	})
	if p.active < 0 {
		p.active = len(p.tabs) - 1
	}
	a.focusEditor = true
}
