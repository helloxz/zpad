package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helloxz/zpad/locales"
)

// pathWithin reports whether path is dir or somewhere inside it.
func pathWithin(path, dir string) bool {
	if path == dir {
		return true
	}
	rel, err := filepath.Rel(dir, path)
	return err == nil && filepath.IsLocal(rel)
}

// rebasePath moves path from under oldDir to the same place under newDir.
func rebasePath(path, oldDir, newDir string) string {
	rel, err := filepath.Rel(oldDir, path)
	if err != nil {
		return path
	}
	return filepath.Join(newDir, rel)
}

// selectTreePath marks the row of path chosen in tp's tree.
func (a *App) selectTreePath(tp treePane, path string) {
	for i := 0; i < tp.state.Rows(); i++ {
		if tp.state.Item(i) == path {
			*tp.selected = i
			return
		}
	}
}

// checkName reports what is wrong with a typed file or folder name itself,
// without looking at the directory: an empty name or a path separator.
func checkName(name string) string {
	switch {
	case name == "":
		return locales.NameEmpty()
	case name == "." || name == ".." || strings.ContainsAny(name, `/\`):
		return locales.NameInvalid()
	}
	return ""
}

// beginRename opens the rename dialog for path.
func (a *App) beginRename(path string) {
	a.renameOpen = true
	a.renamePath = path
	a.renameValue = filepath.Base(path)
	a.renameErr = ""
	a.renameFocus = true
	a.update()
}

// applyRename checks the typed name and starts the rename.
func (a *App) applyRename() {
	name := strings.TrimSpace(a.renameValue)
	old := a.renamePath
	if msg := checkName(name); msg != "" {
		a.renameErr = msg
		a.update()
		return
	}
	newPath := filepath.Join(filepath.Dir(old), name)
	if newPath == old {
		a.renameOpen = false
		return
	}
	if !strings.EqualFold(newPath, old) {
		if _, err := os.Stat(newPath); err == nil {
			a.renameErr = locales.NameExists()
			a.update()
			return
		} else if !errors.Is(err, os.ErrNotExist) {
			a.renameErr = err.Error()
			a.update()
			return
		}
		if a.findDocument(newPath) != nil {
			a.renameErr = locales.NameExists()
			a.update()
			return
		}
	}
	a.renameOpen = false
	a.runFileWork(func() { a.renameOnDisk(old, newPath) })
}

// renameOnDisk moves a file or folder and updates the app.
func (a *App) renameOnDisk(old, newPath string) {
	if err := os.Rename(old, newPath); err != nil {
		a.fail(locales.RenameFailed(filepath.Base(old)), err)
		return
	}
	a.onUI(func() { a.finishRename(old, newPath) })
}

// finishRename updates open documents, the tree and the session after a
// rename.
func (a *App) finishRename(old, newPath string) {
	for _, p := range []*Pane{&a.left, &a.right} {
		for _, tab := range p.tabs {
			if tab.doc.Path == "" || !pathWithin(tab.doc.Path, old) {
				continue
			}
			oldID := tab.doc.ID
			tab.doc.Path = rebasePath(tab.doc.Path, old, newPath)
			tab.doc.ID = tab.doc.Path
			tab.doc.setLanguage(tab.doc.Path)
			a.clearPreviewIfNotMarkdown(tab.doc)
			a.draftMoved(tab.doc)
			if a.revealDoc == oldID {
				a.revealDoc = tab.doc.ID
			}
		}
	}
	a.rebaseTreeStates(old, newPath)
	a.refreshTrees()
	a.treeSelected = -1
	a.dirSelected = -1
	if a.activeTab() != nil {
		a.focusEditor = true
	}
	a.updateTitle()
	a.saveSession()
	a.update()
}

// beginCreate asks for the name of a new file or folder in dir.
func (a *App) beginCreate(dir string, isDir bool) {
	a.createOpen = true
	a.createDir = dir
	a.createIsDir = isDir
	a.createValue = ""
	a.createErr = ""
	a.createFocus = true
	a.update()
}

// runFileWork runs a file operation off the UI thread, where it lands back
// with onUI. Without a window, as in the view tests, it runs at once rather
// than racing the frames the tester builds.
func (a *App) runFileWork(fn func()) {
	if a.win == nil {
		fn()
		return
	}
	go fn()
}

// createNameError returns what is wrong with creating name in dir, or "".
func (a *App) createNameError(dir, name string) string {
	if msg := checkName(name); msg != "" {
		return msg
	}
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err == nil {
		return locales.NameExists()
	} else if !errors.Is(err, os.ErrNotExist) {
		return err.Error()
	}
	if a.findDocument(path) != nil {
		return locales.NameExists()
	}
	return ""
}

// applyCreate checks the typed name and starts creating the new entry.
func (a *App) applyCreate() {
	name := strings.TrimSpace(a.createValue)
	if msg := a.createNameError(a.createDir, name); msg != "" {
		a.createErr = msg
		a.update()
		return
	}
	path := filepath.Join(a.createDir, name)
	isDir := a.createIsDir
	a.createOpen = false
	a.runFileWork(func() { a.createPath(path, isDir) })
}

// createPath makes an empty file, or a folder, and shows it in the tree.
func (a *App) createPath(path string, isDir bool) {
	var err error
	if isDir {
		err = os.Mkdir(path, 0o755)
	} else {
		var f *os.File
		if f, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644); err == nil {
			err = f.Close()
		}
	}
	if err != nil {
		a.fail(locales.CreateFailed(filepath.Base(path)), err)
		return
	}
	a.onUI(func() { a.finishCreate(path, isDir) })
}

// finishCreate opens a new file in the active pane and shows the new entry
// in the tree, expanding the folder it sits in.
func (a *App) finishCreate(path string, isDir bool) {
	// A new entry shows at once: its folder is open in the tree that lists
	// it, and both trees read their folders again.
	a.expandTreeParents(filepath.Dir(path))
	a.refreshTrees()
	if !isDir {
		a.OpenPaths([]string{path})
	}
	a.update()
}

// expandTreeParents opens dir in the trees that list it, so that an entry
// created inside a closed folder shows at once.
func (a *App) expandTreeParents(dir string) {
	if a.tree != nil && pathWithin(dir, a.tree.Root()) {
		a.treeState.Open.Add(dir)
	}
	if a.dirTree != nil && pathWithin(dir, a.dirTree.Root()) {
		a.dirState.Open.Add(dir)
	}
}

// rebaseTreeStates moves the folders the trees show open when a folder
// above them is renamed.
func (a *App) rebaseTreeStates(old, newPath string) {
	rebase := func(state *ui.OutlineState[string]) {
		var open []string
		for p := range state.Open.All() {
			if pathWithin(p, old) {
				open = append(open, rebasePath(p, old, newPath))
			} else {
				open = append(open, p)
			}
		}
		state.Open.Clear()
		for _, p := range open {
			state.Open.Add(p)
		}
	}
	rebase(&a.treeState)
	rebase(&a.dirState)
}

// dirtyWithin reports whether a document at path, or under a folder, has
// unsaved changes.
func (a *App) dirtyWithin(path string) bool {
	for _, p := range []*Pane{&a.left, &a.right} {
		for _, tab := range p.tabs {
			if tab.doc.Path != "" && pathWithin(tab.doc.Path, path) && tab.doc.Buffer.Modified() {
				return true
			}
		}
	}
	return false
}

// confirmDelete asks before deleting a file or folder.
func (a *App) confirmDelete(path string, isDir bool) {
	name := filepath.Base(path)
	if a.dirtyWithin(path) {
		go func() {
			_, _ = mygo.Dialog.Message(mygo.MessageOptions{
				Parent:  a.win,
				Type:    mygo.MessageWarning,
				Message: locales.DeleteUnsavedTitle(),
				Detail:  locales.DeleteUnsavedDetail(name),
			})
		}()
		return
	}
	go func() {
		question := locales.DeleteFileQuestion(name)
		if isDir {
			question = locales.DeleteFolderQuestion(name)
		}
		res, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:             a.win,
			Type:               mygo.MessageWarning,
			Message:            question,
			Detail:             path + "\n" + locales.DeletePermanent(),
			Buttons:            []string{locales.Cancel(), locales.Delete()},
			DefaultButton:      0,
			CancelButton:       0,
			DestructiveButtons: []int{1},
		})
		if err != nil || res.Button != 1 {
			return
		}
		a.deletePath(path, isDir)
	}()
}

// deletePath removes a file or folder and updates the app.
func (a *App) deletePath(path string, isDir bool) {
	var err error
	if isDir {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}
	if err != nil {
		a.fail(locales.DeleteFailed(filepath.Base(path)), err)
		return
	}
	a.onUI(func() { a.finishDelete(path) })
}

// finishDelete closes the tabs of a deleted file or folder and refreshes
// the trees.
func (a *App) finishDelete(path string) {
	a.closeTabsWithin(path)
	a.forgetTreePaths(path)
	a.refreshTrees()
	a.updateTitle()
	a.saveSession()
	a.SetToast(locales.Deleted(filepath.Base(path)))
	a.update()
}

// closeTabsWithin closes the tabs of path, or of the documents under it
// when it is a folder.
func (a *App) closeTabsWithin(path string) {
	var targets []*Tab
	for _, p := range []*Pane{&a.left, &a.right} {
		for _, tab := range p.tabs {
			if tab.doc.Path != "" && pathWithin(tab.doc.Path, path) {
				targets = append(targets, tab)
			}
		}
	}
	for _, target := range targets {
		for paneIdx := 0; paneIdx < 2; paneIdx++ {
			p := a.pane(paneIdx)
			for i, tab := range p.tabs {
				if tab == target {
					a.removeTab(paneIdx, i)
					break
				}
			}
		}
	}
}

// forgetTreePaths closes any tree folder within path in the outline.
func (a *App) forgetTreePaths(path string) {
	forget := func(state *ui.OutlineState[string]) {
		var keep []string
		for p := range state.Open.All() {
			if !pathWithin(p, path) {
				keep = append(keep, p)
			}
		}
		state.Open.Clear()
		for _, p := range keep {
			state.Open.Add(p)
		}
	}
	forget(&a.treeState)
	forget(&a.dirState)
	a.treeSelected = -1
	a.dirSelected = -1
}

// openInOtherPane opens path in the pane that is not active, creating the
// split with just that file when there is none.
func (a *App) openInOtherPane(path string) {
	target := 1 - a.activePane
	if !a.split {
		a.ensureSplit()
		a.activePane = 1
		a.focusEditor = true
		target = 1
	}
	a.openInPane(path, target)
	a.saveSession()
	a.update()
}
