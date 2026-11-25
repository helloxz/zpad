package app

import (
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/helloxz/zpad/locales"
)

// dirtyDocs returns every modified document once, however many panes show
// it.
func (a *App) dirtyDocs() []*Document {
	seen := map[*Document]bool{}
	var out []*Document
	for _, p := range []*Pane{&a.left, &a.right} {
		for _, tab := range p.tabs {
			if tab.doc.Buffer.Modified() && !seen[tab.doc] {
				seen[tab.doc] = true
				out = append(out, tab.doc)
			}
		}
	}
	return out
}

func (a *App) hasDirty() bool { return len(a.dirtyDocs()) > 0 }

func (a *App) onWindowClose(e *mygo.CloseEvent) {
	if a.allowClose {
		a.saveSession()
		return
	}
	if a.closePromptOpen {
		e.PreventDefault()
		return
	}
	if a.Settings.KeepUnsaved {
		// The unsaved text stays in the recovery copies, to open again on
		// the next launch: nothing to ask.
		a.flushDrafts()
		a.saveSession()
		return
	}
	if !a.Settings.ConfirmClose || !a.hasDirty() {
		a.saveSession()
		return
	}
	e.PreventDefault()
	a.requestClose("window")
}

// BeforeQuit asks to save before the application quits.
func (a *App) BeforeQuit(e *mygo.QuitEvent) {
	if e.SessionEnding {
		if a.Settings.KeepUnsaved {
			a.flushDrafts()
		} else {
			a.saveDirtySync()
		}
		a.saveSession()
		// The session end cannot be canceled, and OnClose must not ask.
		a.allowClose = true
		return
	}
	if a.allowQuit {
		a.saveSession()
		return
	}
	if a.closePromptOpen {
		e.PreventDefault()
		return
	}
	if a.Settings.KeepUnsaved {
		a.flushDrafts()
		a.saveSession()
		return
	}
	if !a.Settings.ConfirmClose || !a.hasDirty() {
		a.saveSession()
		return
	}
	e.PreventDefault()
	a.requestClose("quit")
}

func (a *App) requestClose(reason string) {
	if a.closePromptOpen || a.busy {
		return
	}
	docs := a.dirtyDocs()
	if len(docs) == 0 {
		a.executeClose(reason)
		return
	}
	a.closePromptOpen = true
	go func() {
		res, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:             a.win,
			Type:               mygo.MessageWarning,
			Message:            locales.UnsavedFilesQuestion(len(docs)),
			Detail:             dirtyDetail(docs),
			Buttons:            []string{locales.SaveAll(), locales.DontSave(), locales.Cancel()},
			DefaultButton:      0,
			CancelButton:       2,
			DestructiveButtons: []int{1},
		})
		if err != nil {
			a.onUI(func() {
				a.closePromptOpen = false
				a.SetError(locales.CloseFailed() + ": " + err.Error())
			})
			return
		}
		switch res.Button {
		case 0:
			a.onUI(func() {
				a.closePromptOpen = false
				a.saveAllForClose(reason)
			})
		case 1:
			a.onUI(func() {
				a.closePromptOpen = false
				a.executeClose(reason)
			})
		default:
			a.onUI(func() { a.closePromptOpen = false })
		}
	}()
}

func dirtyDetail(docs []*Document) string {
	names := make([]string, 0, len(docs))
	for _, doc := range docs {
		names = append(names, doc.Name())
	}
	return strings.Join(names, ", ")
}

// closeTabs closes the tabs of a pane, asking once about the documents
// among them with unsaved changes.
func (a *App) closeTabs(paneIdx int, tabs []*Tab) {
	if a.busy || len(tabs) == 0 {
		return
	}
	// Snapshot the list: callers pass a slice of the pane's tab array, and
	// removeTab shifts that array under us, which would skip tabs.
	tabs = append([]*Tab(nil), tabs...)
	docs := dirtyTabDocs(tabs)
	if len(docs) == 0 {
		// Closing from a tab's menu happens during a frame; remove the
		// tabs after it, not while the bar is being built.
		a.onUI(func() { a.removeTabsByPointer(paneIdx, tabs) })
		return
	}
	a.busy = true
	go func() {
		res, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:             a.win,
			Type:               mygo.MessageWarning,
			Message:            locales.UnsavedFilesQuestion(len(docs)),
			Detail:             dirtyDetail(docs),
			Buttons:            []string{locales.SaveAll(), locales.DontSave(), locales.Cancel()},
			DefaultButton:      0,
			CancelButton:       2,
			DestructiveButtons: []int{1},
		})
		if err != nil {
			a.onUI(func() {
				a.busy = false
				a.fail(locales.CloseTabFailed(), err)
			})
			return
		}
		switch res.Button {
		case 0:
			a.onUI(func() {
				a.busy = false
				a.saveDocsForClose(docs, func() { a.removeTabsByPointer(paneIdx, tabs) })
			})
		case 1:
			a.onUI(func() {
				a.busy = false
				a.removeTabsByPointer(paneIdx, tabs)
			})
		default:
			a.onUI(func() { a.busy = false })
		}
	}()
}

// dirtyTabDocs returns the modified documents among tabs, once each.
func dirtyTabDocs(tabs []*Tab) []*Document {
	seen := map[*Document]bool{}
	var out []*Document
	for _, tab := range tabs {
		if tab.doc.Buffer.Modified() && !seen[tab.doc] {
			seen[tab.doc] = true
			out = append(out, tab.doc)
		}
	}
	return out
}

// removeTabsByPointer closes the tabs still open by identity, since they
// may have moved while the prompt was up.
func (a *App) removeTabsByPointer(paneIdx int, tabs []*Tab) {
	for _, tab := range tabs {
		a.removeTabByPointer(paneIdx, tab)
	}
	a.saveSession()
	a.update()
}

func (a *App) saveAllForClose(reason string) {
	a.saveDocsForClose(a.dirtyDocs(), func() { a.executeClose(reason) })
}

// saveDocsForClose saves docs, asking where to save each untitled one, and
// calls then once they are all saved.
func (a *App) saveDocsForClose(docs []*Document, then func()) {
	if len(docs) == 0 {
		then()
		return
	}
	if a.busy {
		return
	}
	a.busy = true
	defaults := make([]string, len(docs))
	// The extension of the language each untitled document was given, read
	// here on the main thread; another stays empty, and the typed name is
	// kept as it is.
	exts := make([]string, len(docs))
	for i, doc := range docs {
		defaults[i] = a.saveDialogPath(doc)
		if doc.Path == "" && doc.Language != nil {
			exts[i] = doc.languageExtension()
		}
	}
	go func() {
		type saved struct {
			doc  *Document
			path string
			size int64
		}
		var results []saved
		for i, doc := range docs {
			path := doc.Path
			if path == "" {
				p, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
					Parent:      a.win,
					Title:       locales.SaveFileTitle(),
					DefaultPath: defaults[i],
					Filters:     textFilters(),
				})
				if err != nil {
					a.onUI(func() {
						a.busy = false
						a.SetError(locales.SaveFailed() + ": " + err.Error())
					})
					return
				}
				if p == "" {
					a.onUI(func() { a.busy = false })
					return
				}
				path = withExtension(p, exts[i])
			}
			written, err := writeDocument(doc, path)
			if err != nil {
				a.onUI(func() {
					a.busy = false
					a.SetError(locales.SaveFailed() + ": " + err.Error())
				})
				mygo.Dialog.Error(locales.SaveFailed(), err.Error())
				return
			}
			results = append(results, saved{doc: doc, path: path, size: written})
		}
		a.onUI(func() {
			for _, r := range results {
				r.doc.Path = r.path
				r.doc.ID = r.path
				r.doc.Size = r.size
				r.doc.setLanguage(r.path)
				r.doc.Buffer.MarkSaved()
				a.forgetDraft(r.doc)
			}
			a.busy = false
			a.updateTitle()
			then()
		})
	}()
}

func (a *App) executeClose(reason string) {
	a.saveSession()
	switch reason {
	case "quit":
		a.allowQuit = true
		a.allowClose = true
		mygo.App.Quit()
	default:
		a.allowClose = true
		if a.win != nil {
			a.win.Close()
		}
	}
}

// saveDirtySync writes what it can without dialogs, for the system ending
// the session.
func (a *App) saveDirtySync() {
	for _, doc := range a.dirtyDocs() {
		if doc.Path == "" {
			continue
		}
		if _, err := writeDocument(doc, doc.Path); err != nil {
			log.Printf("save %s on session end: %v", doc.Path, err)
			continue
		}
		doc.Buffer.MarkSaved()
		a.forgetDraft(doc)
	}
}
