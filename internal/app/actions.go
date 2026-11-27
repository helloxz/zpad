package app

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helloxz/zpad/internal/caseconv"
	"github.com/helloxz/zpad/internal/encoding"
	"github.com/helloxz/zpad/locales"
)

// textExtensions are the files the open dialog offers as text, which are
// the extensions mygo.json associates with the app: release_test keeps the
// two lists equal.
var textExtensions = []string{
	// Documents and data.
	"txt", "log", "csv",
	"md", "markdown", "mdx", "mkd",
	"html", "htm", "xhtml",
	"css", "scss", "less", "sass", "pcss",
	"xml", "json", "jsonc", "json5", "webmanifest",
	"yaml", "yml", "toml", "ini", "conf", "properties", "editorconfig",
	"diff", "patch", "rej",
	// Source code.
	"js", "mjs", "cjs", "jsx", "ts", "tsx", "mts", "cts",
	"vue", "svelte",
	"go", "py", "pyi", "pyw", "rs",
	"c", "h", "cpp", "cc", "cxx", "hpp", "hh", "hxx",
	"java", "kt", "kts", "scala", "swift", "dart", "cs",
	"rb", "rake", "lua", "php", "zig",
	"sh", "bash", "zsh", "fish", "ksh",
	"sql", "r", "proto", "graphql", "gql", "graphqls",
	"tf", "tfvars", "hcl", "nix",
	"erb", "rhtml", "tt", "j2", "jinja", "jinja2",
}

// TextExtensions returns the extensions the open dialog offers as text.
func TextExtensions() []string {
	return slices.Clone(textExtensions)
}

func textFilters() []mygo.FileFilter {
	return []mygo.FileFilter{
		{Name: locales.TextFiles(), Extensions: textExtensions},
		{Name: locales.AllFiles(), Extensions: []string{"*"}},
	}
}

// OpenFileDialog asks for files and opens them.
func (a *App) OpenFileDialog() {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:   a.win,
			Title:    locales.OpenFileTitle(),
			Filters:  textFilters(),
			Multiple: true,
		})
		if err != nil {
			a.fail(locales.OpenDialogFailed(), err)
			return
		}
		a.OpenPaths(paths)
	}()
}

// OpenFolderDialog asks for a workspace directory.
func (a *App) OpenFolderDialog() {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:    a.win,
			Title:     locales.OpenFolderTitle(),
			Directory: true,
		})
		if err != nil {
			a.fail(locales.OpenDialogFailed(), err)
			return
		}
		if len(paths) == 0 {
			return
		}
		a.onUI(func() { a.setWorkspace(paths[0]) })
	}()
}

// OpenPaths opens files and directories. It reads files off the UI thread.
func (a *App) OpenPaths(paths []string) {
	a.openPaths(paths, -1, false)
}

// openPreview opens path in the active pane as a preview tab, which the
// next single click in the workspace tree replaces. It opens for good when
// preview tabs are off.
func (a *App) openPreview(path string) {
	a.openPaths([]string{path}, -1, true)
}

// OpenCommandLine opens the files and folders this process was started
// with, skipping flags.
func (a *App) OpenCommandLine(args []string) {
	a.OpenPaths(commandLinePaths(args, ""))
}

// HandleSecondInstance opens what a later launch was asked to open and
// brings this window forward. Windows and Linux start a process per file
// opened from the file manager, whose command line arrives here; that
// process exits at once.
func (a *App) HandleSecondInstance(args []string, dir string) {
	paths := commandLinePaths(args, dir)
	if a.win != nil {
		a.win.Restore()
		a.win.Focus()
	}
	if len(paths) == 0 {
		return
	}
	if a.win == nil {
		// Still starting: Attach opens them with the first window.
		a.pendingOpen = append(a.pendingOpen, paths...)
		return
	}
	a.OpenPaths(paths)
}

// commandLinePaths returns the paths among command-line arguments, leaving
// out flags: relative ones are relative to dir, or to the working directory
// when dir is empty.
func commandLinePaths(args []string, dir string) []string {
	paths := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "" || strings.HasPrefix(arg, "-") {
			continue
		}
		if dir != "" && !filepath.IsAbs(arg) {
			arg = filepath.Join(dir, arg)
		}
		paths = append(paths, arg)
	}
	return paths
}

// beginLoading claims path for a read, and reports false when it is being
// read already.
func (a *App) beginLoading(path string) bool {
	a.loadingMu.Lock()
	defer a.loadingMu.Unlock()
	if a.loadingPaths[path] {
		return false
	}
	a.loadingPaths[path] = true
	return true
}

// endLoading releases path once it is opened, or cannot be.
func (a *App) endLoading(path string) {
	a.loadingMu.Lock()
	defer a.loadingMu.Unlock()
	delete(a.loadingPaths, path)
}

// openPaths opens files and directories in a pane, -1 for the active one.
// Documents already open go at once; the others are read off the UI
// thread. A preview open replaces the pane's preview tab instead of
// adding one.
func (a *App) openPaths(paths []string, paneIdx int, preview bool) {
	if len(paths) == 0 {
		return
	}
	a.onUI(func() {
		pending := make([]string, 0, len(paths))
		for _, path := range paths {
			if doc := a.findDocument(path); doc != nil {
				a.addDocumentAtMode(paneIdx, doc, preview)
				continue
			}
			if !a.beginLoading(path) {
				continue // a launch, a drop or another request reads it
			}
			pending = append(pending, path)
		}
		if len(pending) > 0 {
			a.loadPaths(pending, paneIdx, preview)
		}
	})
}

// openFailure is a file a batch could not open, and why.
type openFailure struct {
	name   string
	reason string
}

// reportOpenFailures tells in one toast what a batch of files could not
// open: the file and the reason for one, the count and the reasons for
// several, which a drop of a folder of images would otherwise stack.
func (a *App) reportOpenFailures(failures []openFailure) {
	if len(failures) == 0 {
		return
	}
	a.onUI(func() {
		if len(failures) == 1 {
			f := failures[0]
			a.SetErrorToast(locales.OpenFailed(f.name), f.reason)
			return
		}
		// Three reasons at most: a batch usually shares one.
		reasons := make([]string, 0, 3)
		for _, f := range failures {
			if len(reasons) == 3 {
				break
			}
			if !slices.Contains(reasons, f.reason) {
				reasons = append(reasons, f.reason)
			}
		}
		a.SetErrorToast(locales.FilesFailedToOpen(len(failures)),
			strings.Join(reasons, locales.ReasonSeparator()))
	})
}

// loadPaths reads files and opens them, in paneIdx when it is a pane. It
// runs on the UI thread, as openPaths calls it.
func (a *App) loadPaths(paths []string, paneIdx int, preview bool) {
	maxMB := a.Settings.LargeFileMB
	a.loading++
	defer func() {
		if a.loading > 0 {
			a.loading--
		}
		a.update()
	}()
	read := func() {
		var dirs, files []string
		var failures []openFailure
		for _, path := range paths {
			info, err := os.Stat(path)
			if err != nil {
				a.endLoading(path)
				failures = append(failures, openFailure{filepath.Base(path), err.Error()})
				continue
			}
			if info.IsDir() {
				dirs = append(dirs, path)
			} else {
				files = append(files, path)
			}
		}
		// A folder first, so files opened with it share its workspace.
		for _, dir := range dirs {
			a.onUI(func() {
				a.endLoading(dir)
				a.setWorkspace(dir)
			})
		}
		maxBytes := maxFileBytes(maxMB)
		for _, path := range files {
			started := time.Now()
			info, err := sniffOpenable(path)
			if err != nil {
				a.endLoading(path)
				failures = append(failures, openFailure{filepath.Base(path), err.Error()})
				continue
			}
			asked := false
			if info.Size() > maxBytes {
				if !a.confirmLargeFile(path, maxMB) {
					a.endLoading(path)
					continue
				}
				asked = true
			}
			doc, err := loadDocument(path, maxMB)
			if err != nil {
				a.endLoading(path)
				failures = append(failures, openFailure{filepath.Base(path), err.Error()})
				continue
			}
			log.Printf("open %s %s %d lines ro=%t %s", filepath.Base(path),
				formatSize(doc.Size), doc.Buffer.LineCount(), doc.ReadOnly,
				time.Since(started).Round(time.Millisecond))
			if doc.ReadOnly && !asked {
				if !a.confirmLargeFile(path, maxMB) {
					a.endLoading(path)
					continue
				}
			}
			a.onUI(func() {
				a.endLoading(path)
				a.addDocumentAtMode(paneIdx, doc, preview)
			})
		}
		a.reportOpenFailures(failures)
	}
	// A window's next frame brings the files in; without one, as in the
	// view tests, they are read at once rather than raced with the frames
	// the tester builds.
	if a.win == nil {
		read()
		return
	}
	go read()
}

func (a *App) confirmLargeFile(path string, maxMB int) bool {
	res, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent:        a.win,
		Type:          mygo.MessageInfo,
		Message:       locales.LargeFileQuestion(),
		Detail:        filepath.Base(path) + " · " + locales.FileTooLarge(maxMB),
		Buttons:       []string{locales.OpenReadOnly(), locales.Cancel()},
		DefaultButton: 0,
		CancelButton:  1,
	})
	return err == nil && res.Button == 0
}

// NewFile adds an untitled document. The menu's CmdOrCtrl+N runs it with no
// frame of the view around it, so it asks for the frame that shows the tab.
func (a *App) NewFile() {
	a.addDocument(a.newUntitledDocument())
	a.update()
}

// Undo takes back the active buffer's last edit.
func (a *App) Undo() {
	if tab := a.activeTab(); tab != nil && !tab.doc.ReadOnly && !tab.preview {
		tab.doc.Buffer.Undo()
		a.updateTitle()
		a.update()
	}
}

// Redo makes the active buffer's last undone edit again.
func (a *App) Redo() {
	if tab := a.activeTab(); tab != nil && !tab.doc.ReadOnly && !tab.preview {
		tab.doc.Buffer.Redo()
		a.updateTitle()
		a.update()
	}
}

// SaveActive saves the active tab.
func (a *App) SaveActive() {
	if tab := a.activeTab(); tab != nil {
		a.saveTab(tab, false, nil)
	}
}

// SaveAsActive saves the active tab under another name.
func (a *App) SaveAsActive() {
	if tab := a.activeTab(); tab != nil {
		a.saveTab(tab, true, nil)
	}
}

// showItemInFolder is mygo.Shell.ShowItemInFolder, a variable so tests can
// record the path without opening the file manager.
var showItemInFolder = mygo.Shell.ShowItemInFolder

// OpenLog opens the diagnostic log in a tab: what mygo said of the
// renderer, the files opened, and the frames that ran slow.
func (a *App) OpenLog() {
	if a.LogPath == "" {
		return
	}
	if _, err := os.Stat(a.LogPath); err != nil {
		a.SetToast(locales.NoLog())
		return
	}
	a.OpenPaths([]string{a.LogPath})
}

// canRevealActive reports whether the active tab has a file on disk to show
// in the file manager: an untitled document has no directory.
func (a *App) canRevealActive() bool {
	tab := a.activeTab()
	return tab != nil && tab.doc.Path != ""
}

// revealDocument shows doc's file in the system file manager: Finder and
// Explorer select it, and other systems open its folder. An untitled
// document has no file to show.
func (a *App) revealDocument(doc *Document) {
	if doc == nil || doc.Path == "" {
		return
	}
	showItemInFolder(doc.Path)
}

// ShowActiveInFileManager reveals the active file in the system file
// manager.
func (a *App) ShowActiveInFileManager() {
	if tab := a.activeTab(); tab != nil {
		a.revealDocument(tab.doc)
	}
}

// SaveWithEncoding saves the active tab in kind.
func (a *App) SaveWithEncoding(kind encoding.Kind) {
	tab := a.activeTab()
	if tab == nil {
		return
	}
	tab.doc.Encoding = kind
	a.saveTab(tab, false, nil)
}

// ReopenWithEncoding reads the active file again as kind.
func (a *App) ReopenWithEncoding(kind encoding.Kind) {
	tab := a.activeTab()
	if tab == nil || tab.doc.Path == "" {
		return
	}
	if !tab.doc.Buffer.Modified() {
		a.reloadAs(tab, kind)
		return
	}
	go func() {
		res, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:             a.win,
			Type:               mygo.MessageWarning,
			Message:            locales.DiscardChangesQuestion(),
			Detail:             tab.doc.Name(),
			Buttons:            []string{locales.DiscardAndReopen(), locales.Cancel()},
			DefaultButton:      1,
			CancelButton:       1,
			DestructiveButtons: []int{0},
		})
		if err != nil || res.Button != 0 {
			return
		}
		a.reloadAs(tab, kind)
	}()
}

func (a *App) reloadAs(tab *Tab, kind encoding.Kind) {
	path := tab.doc.Path
	go func() {
		started := time.Now()
		loaded, err := readDocumentWith(path, kind, a.Settings.LargeFileMB)
		if err != nil {
			a.fail(locales.ReopenFailed(), err)
			return
		}
		log.Printf("reload %s %s", filepath.Base(path), time.Since(started).Round(time.Millisecond))
		a.onUI(func() {
			tab.doc.Buffer.Reset(loaded.text)
			tab.doc.Encoding = kind
			tab.doc.Size = loaded.size
			tab.doc.ModTime = loaded.modTime
			readOnly := loaded.size > loaded.maxBytes || tab.doc.Buffer.LineCount() > maxLines
			a.setDocReadOnly(tab.doc, readOnly)
			tab.doc.setLanguage(path)
			tab.doc.Buffer.MarkSaved()
			a.forgetDraft(tab.doc)
			a.updateTitle()
		})
	}()
}

// SetLineEnding converts every line of the active tab to eol.
func (a *App) SetLineEnding(eol string) {
	tab := a.activeTab()
	if tab == nil || tab.doc.ReadOnly {
		return
	}
	tab.doc.Buffer.SetLineEnding(eol)
	a.maybeWakeDraftLoop()
	a.updateTitle()
	a.update()
}

// SelectLanguage gives the active untitled document the language chosen in
// the status bar, which changes its highlight, the name its save starts
// from, and whether it can be previewed as Markdown. A file that already has
// a path takes its language from that path, so renaming it is how its
// highlight changes.
func (a *App) SelectLanguage(lang *editor.Language) {
	tab := a.activeTab()
	if tab == nil || tab.doc.Path != "" {
		return
	}
	tab.doc.selectLanguage(lang)
	a.clearPreviewIfNotMarkdown(tab.doc)
	a.draftLanguageChanged(tab.doc)
	a.update()
}

// ConvertCase changes the case of the active tab's selections.
func (a *App) ConvertCase(mode caseconv.Mode) {
	tab := a.activeTab()
	if tab == nil || tab.doc.ReadOnly || tab.preview {
		return
	}
	applyCase(tab, mode)
	a.updateTitle()
	a.update()
}

// CloseActiveTab closes the active tab.
func (a *App) CloseActiveTab() {
	p := a.activePaneObj()
	a.CloseTab(a.activePane, p.active)
}

// CloseTab closes tab i of a pane, asking to save when it has changes.
func (a *App) CloseTab(paneIdx, i int) {
	p := a.pane(paneIdx)
	if i < 0 || i >= len(p.tabs) || a.busy {
		return
	}
	tab := p.tabs[i]
	if !tab.doc.Buffer.Modified() {
		// Closing from a tab's button happens during a frame; remove the
		// tab after it, not while its bar is being built.
		a.onUI(func() { a.removeTabByPointer(paneIdx, tab) })
		return
	}
	a.busy = true
	go func() {
		res, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:             a.win,
			Type:               mygo.MessageWarning,
			Message:            locales.SaveChangesQuestion(),
			Detail:             tab.doc.DisplayPath(a.root),
			Buttons:            []string{locales.Save(), locales.DontSave(), locales.Cancel()},
			DefaultButton:      0,
			CancelButton:       2,
			DestructiveButtons: []int{1},
		})
		if err != nil {
			a.onUI(func() { a.busy = false })
			a.fail(locales.CloseTabFailed(), err)
			return
		}
		switch res.Button {
		case 0:
			a.onUI(func() {
				a.busy = false
				a.saveTab(tab, false, func() { a.removeTabByPointer(paneIdx, tab) })
			})
		case 1:
			a.onUI(func() {
				a.busy = false
				a.removeTabByPointer(paneIdx, tab)
			})
		default:
			a.onUI(func() { a.busy = false })
		}
	}()
}

func (a *App) removeTabByPointer(paneIdx int, t *Tab) {
	p := a.pane(paneIdx)
	for i, tab := range p.tabs {
		if tab == t {
			a.removeTab(paneIdx, i)
			return
		}
	}
}

// saveDialogPath is where the save dialog for doc starts. A file keeps its
// path; an untitled document starts with its English default name in the
// workspace root, else in the home folder, so the dialog does not open at
// the process's working directory.
func (a *App) saveDialogPath(doc *Document) string {
	if doc.Path != "" {
		return doc.Path
	}
	name := doc.saveName()
	dir := a.root
	if dir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dir = home
		}
	}
	if dir == "" {
		return name
	}
	return filepath.Join(dir, name)
}

func (a *App) saveTab(tab *Tab, saveAs bool, then func()) {
	if a.busy {
		return
	}
	// Saving a preview tab keeps it, as editors do: the file is in use now.
	tab.temporary = false
	a.busy = true
	defaultPath := a.saveDialogPath(tab.doc)
	// An untitled document's chosen language names the file it saves: the
	// dialog starts with its extension, and a name typed without one gets
	// it. Both are read here, on the main thread.
	addExt := tab.doc.Path == "" && tab.doc.Language != nil
	ext := tab.doc.languageExtension()
	go func() {
		defer a.onUI(func() { a.busy = false })

		saveAs = saveAs || tab.doc.ReadOnly
		path := tab.doc.Path
		if saveAs || path == "" {
			p, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
				Parent:      a.win,
				Title:       locales.SaveFileTitle(),
				DefaultPath: defaultPath,
				Filters:     textFilters(),
			})
			if err != nil {
				a.fail(locales.SaveFailed(), err)
				return
			}
			if p == "" {
				return
			}
			path = p
			if addExt {
				path = withExtension(path, ext)
			}
		}

		if !saveAs && tab.doc.Path != "" {
			if _, statErr := os.Stat(path); errors.Is(statErr, os.ErrNotExist) {
				res, err := mygo.Dialog.Message(mygo.MessageOptions{
					Parent:        a.win,
					Type:          mygo.MessageWarning,
					Message:       locales.FileMissingQuestion(),
					Detail:        path,
					Buttons:       []string{locales.Recreate(), locales.Cancel()},
					DefaultButton: 0,
					CancelButton:  1,
				})
				if err != nil || res.Button != 0 {
					return
				}
			}
		}

		started := time.Now()
		written, err := writeDocument(tab.doc, path)
		if err != nil {
			// The dialog is the one to act on; a toast under it would say
			// the same twice.
			mygo.Dialog.Error(locales.SaveFailed(), path+": "+err.Error())
			return
		}
		log.Printf("save %s %d bytes %s", filepath.Base(path), written,
			time.Since(started).Round(time.Millisecond))
		var modTime time.Time
		if info, statErr := os.Stat(path); statErr == nil {
			modTime = info.ModTime()
		}
		a.onUI(func() {
			tab.doc.Path = path
			tab.doc.ID = path
			tab.doc.Size = written
			tab.doc.ModTime = modTime
			if saveAs {
				a.setDocReadOnly(tab.doc, false)
			}
			tab.doc.setLanguage(path)
			a.clearPreviewIfNotMarkdown(tab.doc)
			tab.doc.Buffer.MarkSaved()
			a.forgetDraft(tab.doc)
			a.updateTitle()
			if then != nil {
				then()
			}
		})
	}()
}

func (a *App) fail(title string, err error) {
	if err == nil {
		return
	}
	a.onUI(func() { a.SetErrorToast(title, err.Error()) })
}
