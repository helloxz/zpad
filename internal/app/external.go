package app

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"github.com/helloxz/zpad/locales"
)

// checkExternalChanges compares the files of the open documents with the
// disk when the window gains focus.
func (a *App) checkExternalChanges() {
	if a.externalChecking || a.busy {
		return
	}
	a.externalChecking = true
	go func() {
		defer a.onUI(func() { a.externalChecking = false })
		for _, doc := range a.allDocs() {
			if doc.Path == "" || doc.ModTime.IsZero() {
				continue
			}
			info, err := os.Stat(doc.Path)
			if err != nil {
				// A deleted file asks on save; keep the open document.
				continue
			}
			if info.Size() == doc.Size && info.ModTime().Equal(doc.ModTime) {
				continue
			}
			if !doc.Buffer.Modified() {
				a.reloadExternal(doc)
				continue
			}
			res, err := mygo.Dialog.Message(mygo.MessageOptions{
				Parent:             a.win,
				Type:               mygo.MessageWarning,
				Message:            locales.ExternalChangeQuestion(),
				Detail:             doc.DisplayPath(a.root),
				Buttons:            []string{locales.Reload(), locales.KeepMine(), locales.SaveAs()},
				DefaultButton:      0,
				CancelButton:       -1,
				DestructiveButtons: []int{0},
			})
			if err != nil {
				continue
			}
			switch res.Button {
			case 0:
				a.reloadExternal(doc)
			case 1:
				a.onUI(func() {
					doc.Size = info.Size()
					doc.ModTime = info.ModTime()
				})
			case 2:
				a.onUI(func() {
					if tab := a.findTab(doc); tab != nil {
						a.saveTab(tab, true, nil)
					}
				})
			}
		}
	}()
}

func (a *App) allDocs() []*Document {
	seen := map[*Document]bool{}
	var out []*Document
	for _, p := range []*Pane{&a.left, &a.right} {
		for _, tab := range p.tabs {
			if !seen[tab.doc] {
				seen[tab.doc] = true
				out = append(out, tab.doc)
			}
		}
	}
	return out
}

func (a *App) findTab(doc *Document) *Tab {
	for _, p := range []*Pane{&a.left, &a.right} {
		for _, tab := range p.tabs {
			if tab.doc == doc {
				return tab
			}
		}
	}
	return nil
}

func (a *App) reloadExternal(doc *Document) {
	started := time.Now()
	loaded, err := readDocumentWith(doc.Path, doc.Encoding, a.Settings.LargeFileMB)
	if err != nil {
		a.fail(locales.ReopenFailed(), err)
		return
	}
	if doc.Path != a.LogPath {
		// Reloading the log itself, which grows as it is read, is not
		// worth a line of it each focus.
		log.Printf("reload %s %s", filepath.Base(doc.Path), time.Since(started).Round(time.Millisecond))
	}
	a.onUI(func() {
		doc.Buffer.Reset(loaded.text)
		doc.Encoding = loaded.encoding
		doc.Size = loaded.size
		doc.ModTime = loaded.modTime
		readOnly := loaded.size > loaded.maxBytes || doc.Buffer.LineCount() > maxLines
		a.setDocReadOnly(doc, readOnly)
		doc.setLanguage(doc.Path)
		doc.Buffer.MarkSaved()
		a.forgetDraft(doc)
		a.updateTitle()
		if doc.Path != a.LogPath {
			a.SetToast(locales.Reloaded())
		}
	})
}
