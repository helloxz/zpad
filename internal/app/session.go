package app

import (
	"log"
	"os"

	"github.com/egoist/mygo/ui/editor"
	"github.com/helloxz/zpad/internal/files"
	"github.com/helloxz/zpad/internal/session"
	"github.com/helloxz/zpad/locales"
)

// RestoreSession loads the workspace of the last launch, when the user
// asked for it, and the unsaved text kept for this launch, whether they asked
// to restore the session or not.
func (a *App) RestoreSession() {
	if a.SettingsDir == "" {
		return
	}
	a.loadDraftsAtLaunch()
	defer func() {
		a.draftEntries = nil
		a.draftByPath = nil
		a.draftUsed = nil
	}()

	if !a.Settings.RestoreSession || !session.Exists(a.SettingsDir) {
		if a.drafts != nil {
			restored := 0
			a.restoreLeftoverDrafts(&restored)
			a.noteRestoredDrafts(restored)
		}
		return
	}
	st, err := session.Load(a.SettingsDir)
	if err != nil {
		log.Printf("load session: %v", err)
		if a.drafts != nil {
			restored := 0
			a.restoreLeftoverDrafts(&restored)
			a.noteRestoredDrafts(restored)
		}
		return
	}
	a.restoreSession(st)
}

// loadDraftsAtLaunch opens the recovery copies before the session is read, so
// that its tabs can take their text. A store that cannot be opened turns the
// setting off, as it would leave nothing to keep.
func (a *App) loadDraftsAtLaunch() {
	if !a.Settings.KeepUnsaved {
		return
	}
	if err := a.openDraftsStore(); err != nil {
		a.draftLog(err)
		a.Settings.KeepUnsaved = false
		a.saveSettings()
		a.SetErrorToast(locales.KeepUnsavedFailed(), err.Error())
		return
	}
	a.loadDraftRestoreIndex()
}

// noteRestoredDrafts tells how many documents came back with unsaved text.
func (a *App) noteRestoredDrafts(restored int) {
	if restored > 0 {
		log.Printf("restored %d unsaved documents", restored)
		a.SetToast(locales.RestoredDrafts(restored))
	}
}

func (a *App) restoreSession(st session.State) {
	if st.Root != "" {
		if info, err := os.Stat(st.Root); err == nil && info.IsDir() {
			a.root = st.Root
			a.tree = files.New(st.Root, a.Settings.ShowHidden)
			a.treeGeneration++
		}
	}
	if st.SidebarWidth >= 160 {
		a.Settings.SidebarWidth = st.SidebarWidth
	}
	a.Settings.SidebarOpen = st.SidebarOpen
	a.split = st.Split
	a.splitVertical = st.SplitVertical
	if st.SplitWidth >= 120 {
		a.splitWidth = st.SplitWidth
	}
	if st.SplitHeight >= 120 {
		a.splitHeight = st.SplitHeight
	}
	a.activePane = st.ActivePane

	docs := map[string]*Document{}
	missing := 0
	restored := 0

	// loadDoc opens the document a session tab shows, putting the text of its
	// recovery copy over the file's when there is one.
	loadDoc := func(ts session.TabState) *Document {
		if ts.Path == "" {
			if !a.Settings.KeepUnsaved {
				return nil
			}
			// An untitled tab is back with the language the status bar was
			// given for it.
			var lang *editor.Language
			if l := editor.LanguageNamed(ts.Language); l != nil {
				lang = l
			}
			if e, ok := a.draftForTab(ts); ok {
				text, err := a.drafts.Text(e.ID)
				if err != nil {
					a.draftLog(err)
				} else {
					if e.Untitled <= 0 {
						a.untitledSeq++
						e.Untitled = a.untitledSeq
					}
					if e.Untitled > a.untitledSeq {
						a.untitledSeq = e.Untitled
					}
					a.draftUsed[e.ID] = true
					restored++
					doc := newDraftDocument(e, text)
					if lang != nil {
						doc.selectLanguage(lang)
					}
					return doc
				}
			}
			seq := ts.Untitled
			if seq <= 0 {
				a.untitledSeq++
				seq = a.untitledSeq
			}
			if seq > a.untitledSeq {
				a.untitledSeq = seq
			}
			doc := newUntitled(seq, a.Settings.DefaultEncoding)
			doc.Buffer.SetLineEnding(a.Settings.DefaultEOL)
			if lang != nil {
				doc.selectLanguage(lang)
			}
			return doc
		}
		if doc := docs[ts.Path]; doc != nil {
			return doc
		}
		doc, err := loadDocument(ts.Path, a.Settings.LargeFileMB)
		if err != nil {
			e, ok := a.draftForTab(ts)
			if !ok {
				missing++
				return nil
			}
			text, terr := a.drafts.Text(e.ID)
			if terr != nil {
				a.draftLog(terr)
				missing++
				return nil
			}
			restored++
			a.draftUsed[e.ID] = true
			doc = newDraftDocument(e, text)
		} else if e, ok := a.draftForTab(ts); ok {
			if a.attachDraft(doc, e) {
				restored++
			}
			a.draftUsed[e.ID] = true
		}
		docs[ts.Path] = doc
		return doc
	}

	loadPane := func(ps session.PaneState) Pane {
		p := Pane{active: -1}
		for _, ts := range ps.Tabs {
			doc := loadDoc(ts)
			if doc == nil {
				continue
			}
			// Do not show one document twice in one pane.
			duplicate := false
			for _, tab := range p.tabs {
				if tab.doc == doc {
					duplicate = true
					break
				}
			}
			if duplicate {
				continue
			}
			y := float64(ts.ScrollY)
			p.tabs = append(p.tabs, &Tab{
				doc:           doc,
				state:         &editor.State{Buffer: doc.Buffer, ReadOnly: doc.ReadOnly},
				pendingScroll: &y,
			})
		}
		p.active = ps.Active
		if p.active < 0 || p.active >= len(p.tabs) {
			if len(p.tabs) == 0 {
				p.active = -1
			} else {
				p.active = len(p.tabs) - 1
			}
		}
		return p
	}
	a.left = loadPane(st.Left)
	a.right = loadPane(st.Right)

	if a.split && (len(a.left.tabs) == 0 || len(a.right.tabs) == 0) {
		if len(a.left.tabs) == 0 {
			a.collapseSplit(0)
		} else {
			a.collapseSplit(1)
		}
	}
	if !a.split {
		a.activePane = 0
	}
	if a.activePane == 0 && len(a.left.tabs) == 0 && len(a.right.tabs) > 0 {
		a.activePane = 1
	}
	if a.activePane == 1 && len(a.right.tabs) == 0 && len(a.left.tabs) > 0 {
		a.activePane = 0
	}
	if a.root != "" {
		a.sidebarTab = sidebarWorkspace
	}
	a.focusEditor = len(a.left.tabs)+len(a.right.tabs) > 0

	a.restoreLeftoverDrafts(&restored)

	if tab := a.pane(a.activePane).activeTab(); tab != nil {
		a.revealPane, a.revealDoc = a.activePane, tab.doc.ID
	}
	if missing > 0 {
		a.SetToast(locales.MissingFiles(missing))
	}
	a.noteRestoredDrafts(restored)
}

// saveSession stores the workspace for the next launch.
func (a *App) saveSession() {
	if a.SettingsDir == "" {
		return
	}
	st := session.State{
		Root:          a.root,
		Split:         a.split,
		SplitWidth:    a.splitWidth,
		SplitVertical: a.splitVertical,
		SplitHeight:   a.splitHeight,
		ActivePane:    a.activePane,
		SidebarOpen:   a.Settings.SidebarOpen,
		SidebarWidth:  a.Settings.SidebarWidth,
		Left:          a.paneState(a.left),
		Right:         a.paneState(a.right),
	}
	if err := session.Save(a.SettingsDir, st); err != nil {
		log.Printf("save session: %v", err)
	}
}

func (a *App) paneState(p Pane) session.PaneState {
	ps := session.PaneState{Active: p.active}
	for _, tab := range p.tabs {
		ts := session.TabState{
			Path:    tab.doc.Path,
			ScrollY: float32(tab.state.ScrollY()),
		}
		if a.Settings.KeepUnsaved {
			ts.Draft = tab.doc.DraftID
			ts.Untitled = tab.doc.UntitledSeq
			if tab.doc.Path == "" && tab.doc.Language != nil {
				ts.Language = tab.doc.Language.Name
			}
		}
		if ts.Path == "" && !a.Settings.KeepUnsaved {
			continue
		}
		ps.Tabs = append(ps.Tabs, ts)
	}
	return ps
}
