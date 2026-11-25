// Package app holds ZPad's state and builds its window.
package app

import (
	"log"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helloxz/zpad/internal/draft"
	"github.com/helloxz/zpad/internal/files"
	"github.com/helloxz/zpad/internal/settings"
	"github.com/helloxz/zpad/internal/snippets"
	"github.com/helloxz/zpad/internal/theme"
	"github.com/helloxz/zpad/internal/vim"
	"github.com/helloxz/zpad/locales"
)

// Tab is one open document and the editor state showing it.
type Tab struct {
	doc           *Document
	state         *editor.State
	pendingScroll *float64
	// vim is the tab's modal state while Vim mode is on; nil when it is off
	// or the tab has not been shown since it was turned on.
	vim *vim.Session
	// preview shows the document as rendered Markdown rather than in the
	// editor; previewScroll keeps its place across that switch.
	preview       bool
	previewScroll ui.ScrollState
}

// Pane is one side of the window: its tabs and the one chosen.
type Pane struct {
	tabs   []*Tab
	active int
}

// dragKind tells what a paneDrag carries.
type dragKind uint8

const (
	dragTab dragKind = iota
	dragFile
)

// paneDrag is something dropped on a pane: a tab to move there, or a file
// to open there.
type paneDrag struct {
	kind dragKind
	pane int
	id   string
	path string
}

// wantsPane reports whether d can land on a pane: a file on any, a tab
// only on one it does not come from.
func (d paneDrag) wantsPane(pane int) bool {
	return d.kind == dragFile || d.pane != pane
}

func (p *Pane) activeTab() *Tab {
	if p.active < 0 || p.active >= len(p.tabs) {
		return nil
	}
	return p.tabs[p.active]
}

// The sidebar's tabs.
const (
	sidebarFiles = iota
	sidebarWorkspace
)

// Version is the version the status bar and the About dialog show. main
// sets it from mygo.json, which mygo build links in: that file is the app's
// only version, so a release bumps one file. It is "dev" until then, as
// under go test or go run.
var Version = "dev"

// SetVersion records the version mygo build or mygo dev linked in; an empty
// one, as from a plain go build, keeps the default.
func SetVersion(v string) {
	if v != "" {
		Version = v
	}
}

// App is the state the window shows.
type App struct {
	Settings    settings.Settings
	SettingsDir string
	// LogPath is where SetupLogging writes, "" when logging is off.
	LogPath string
	// Dark reports whether the current theme is built on a dark palette.
	// Settings.Theme says which theme it is.
	Dark bool

	win     *mygo.Window
	running bool
	errMsg  string
	// customMenuBar draws the application menu's top-level menus in the
	// window's title bar, as Windows does: the window is created without a
	// title bar there, and the menus open in native popups. menuBarNames
	// are their labels, in order, rebuilt with the menu on a language
	// change.
	customMenuBar bool
	menuBarNames  []string
	// checkUpdates runs the update check, from the menu and the toolbar.
	// It is a field because updater.CheckForUpdates panics when the plugin
	// is not registered, as in the view tests.
	checkUpdates func()

	root           string
	tree           *files.Tree
	treeState      ui.OutlineState[string]
	treeSelected   int
	treeGeneration uint64
	sidebarTab     int

	left       Pane
	right      Pane
	split      bool
	splitWidth float32
	// splitVertical divides the content top and bottom rather than left
	// and right; splitHeight is the top pane's height then.
	splitVertical bool
	splitHeight   float32
	centerSplit   bool
	activePane    int // 0 left/top, 1 right/bottom

	// tabOverflow says whether a pane's tab bar is wider than the pane,
	// which shows the tab list button; revealPane and revealDoc ask the
	// tab about to build to scroll into view.
	tabOverflow [2]bool
	revealPane  int
	revealDoc   string

	// focusEditor asks the next frame to give the keyboard to the active
	// pane's editor. editorRef finds that editor: its element outlives a
	// tab, so AutoFocus would not take it once one is open, and a handle
	// focuses it in the frame the flag is set.
	focusEditor bool
	editorRef   ui.Handle
	untitledSeq int
	busy        bool
	loading     int
	// loadingPaths holds the files being read, so that one asked for twice,
	// as a second instance and the system may, is read and opened once.
	loadingMu        sync.Mutex
	loadingPaths     map[string]bool
	externalChecking bool
	lastDirty        bool
	pendingOpen      []string

	// The recovery copies of unsaved content, while KeepUnsaved is on: the
	// store, the loop that writes it, and the copies found at launch.
	drafts       *draft.Store
	draftStop    chan struct{}
	draftWriting bool
	draftWrites  sync.WaitGroup
	draftEntries map[string]draft.Entry
	draftByPath  map[string]string
	draftUsed    map[string]bool
	draftLogged  string

	showSettings bool
	aboutOpen    bool
	aboutIcon    *ui.Bitmap
	// appIconPNG is the icon the About dialog decodes, embedded by main.
	appIconPNG   []byte
	fontSize     float64
	lineHeight   float64
	fontFamilies []string
	// pendingToasts are the toasts the next frame shows, in order.
	pendingToasts []ui.Toast

	renameOpen  bool
	renamePath  string
	renameValue string
	renameErr   string
	renameFocus bool

	// createOpen asks for the name of a new file or folder in createDir;
	// createIsDir says which of the two it makes.
	createOpen  bool
	createDir   string
	createIsDir bool
	createValue string
	createErr   string
	createFocus bool

	// The search across open documents: the panel, its query and toggles,
	// the rows it shows, and the scan on its way. searchGen says which scan
	// is current, so an older one's results are ignored; searchCancel stops
	// one early.
	searchOpen      bool
	searchFocus     bool
	searchQuery     string
	searchCase      bool
	searchWord      bool
	searchRegex     bool
	searchRows      []searchRow
	searchFiles     int
	searchSel       int
	searchList      ui.ListState
	searchBusy      bool
	searchErr       string
	searchTruncated bool
	searchGen       uint64
	searchCancel    chan struct{}
	// searchRanFor and searchRanOptions are what the rows stand for: a
	// query and toggles edited since then need Enter (or the search
	// button) before the panel can jump to a result.
	searchRanFor     string
	searchRanOptions searchOptions

	// The user's code snippets: the store read when the panel opens, the
	// panel's query, the rows it lists, the choice among them, and the
	// group on view, -1 for all of them. snippetCounts is how many rows
	// each group matched, for the column on the left. The editor dialog
	// edits a snippet, the group dialog a group's name; both sit over the
	// panel, which is not drawn then.
	snippets      *snippets.Store
	snippetsOpen  bool
	snippetsFocus bool
	snippetsQuery string
	snippetsRows  []snippetRow
	snippetCounts []int
	snippetsSel   int
	snippetsList  ui.ListState
	snippetsGroup int

	snippetEditOpen    bool
	snippetEditGroupID string
	snippetEditID      string
	snippetEditName    string
	snippetEditBody    string
	snippetEditErr     string
	snippetEditFocus   bool

	groupEditOpen  bool
	groupEditID    string
	groupEditName  string
	groupEditErr   string
	groupEditFocus bool

	// vimCmdOpen is the : or / command line at the bottom of the window,
	// which vimCmdKind says is which and vimCmdValue holds; vimCmdFocus
	// gives its field the keyboard once.
	vimCmdOpen  bool
	vimCmdKind  rune
	vimCmdValue string
	vimCmdFocus bool
	vimCmdTab   *Tab
	// frame is the context of the frame being built, for the clipboard the
	// Vim session's yanks and deletes use from an input handler.
	frame *ui.Context

	closePromptOpen bool
	allowClose      bool
	allowQuit       bool
	// viewLogged says the window size was logged, once, on the first frame.
	viewLogged bool
}

// New returns an app with saved settings and the directory to save them
// in.
func New(s settings.Settings, settingsDir string) *App {
	s.Theme = theme.Normalize(s.Theme)
	def := theme.Resolve(s.Theme, false)
	return &App{
		Settings:      s,
		SettingsDir:   settingsDir,
		LogPath:       filepath.Join(settingsDir, LogName),
		Dark:          def.Dark,
		treeSelected:  -1,
		left:          Pane{active: -1},
		right:         Pane{active: -1},
		splitWidth:    600,
		splitHeight:   500,
		snippetsGroup: -1,
		snippetsSel:   -1,
		fontSize:      float64(s.FontSize),
		lineHeight:    float64(s.LineHeight),
		loadingPaths:  map[string]bool{},
		checkUpdates:  updater.CheckForUpdates,
	}
}

// SetThemeSource applies a theme's appearance to the native window
// chrome before App.Run and when the theme changes: a named theme keeps
// the light or dark title bar it is built on, "system" follows the
// desktop.
func SetThemeSource(id string) {
	if id == "" || id == theme.SystemID {
		mygo.Theme.SetSource(mygo.ThemeSystem)
		return
	}
	def, ok := theme.Lookup(id)
	if !ok {
		mygo.Theme.SetSource(mygo.ThemeSystem)
		return
	}
	if def.Dark {
		mygo.Theme.SetSource(mygo.ThemeDark)
	} else {
		mygo.Theme.SetSource(mygo.ThemeLight)
	}
}

// Prepare marks the app as running and resolves the system appearance.
// Call it on the main thread once the app is ready, before the window.
func (a *App) Prepare() {
	a.running = true
	if a.Settings.Theme == "system" {
		a.Dark = mygo.Theme.IsDark()
	}
}

// Attach remembers the window, installs the menu and takes dropped files.
func (a *App) Attach(win *mygo.Window) {
	a.win = win
	// Windows hides its title bar and draws the menu bar in it; the view
	// tests build the window without Attach and draw no menu bar unless
	// they ask for one.
	a.customMenuBar = runtime.GOOS == "windows"
	win.OnClose(a.onWindowClose)
	win.OnFocus(func() { a.checkExternalChanges() })
	win.OnFileDrop(func(e *mygo.FileDropEvent) {
		a.OpenPaths(e.Paths)
	})
	mygo.Theme.OnUpdated(func() {
		if a.Settings.Theme == "system" {
			a.Dark = mygo.Theme.IsDark()
			a.update()
		}
	})
	mygo.App.OnLocaleChanged(func(string) {
		a.InstallMenu()
		a.updateTitle()
		a.update()
	})
	a.InstallMenu()
	a.updateTitle()
	if a.Settings.KeepUnsaved {
		if a.drafts == nil {
			if err := a.openDraftsStore(); err != nil {
				a.draftLog(err)
			}
		}
		a.startDraftLoop()
	}
	if len(a.pendingOpen) > 0 {
		paths := a.pendingOpen
		a.pendingOpen = nil
		go a.OpenPaths(paths)
	}
}

// HandleOpenFile opens a file the system asks the app to show. Before the
// window exists, it keeps the path for Attach.
func (a *App) HandleOpenFile(path string) {
	if a.win == nil {
		a.pendingOpen = append(a.pendingOpen, path)
		return
	}
	a.OpenPaths([]string{path})
}

// SetError shows an error in the status bar, for a failure that has no
// better home: one the user must act on while its document is open.
//
// Actions that end, as opening a file does, tell with SetErrorToast
// instead, so that a message the user cannot act on does not stay.
func (a *App) SetError(msg string) {
	a.errMsg = msg
}

// errorToastTime is how long an error toast shows: enough to read a reason
// under its title. The pointer resting on it holds it, and Escape closes
// it.
const errorToastTime = 6 * time.Second

// SetErrorToast tells near the bottom of the window that something the
// user asked for did not happen: title says what, detail why, and it goes
// away on its own.
func (a *App) SetErrorToast(title, detail string) {
	a.pendingToasts = append(a.pendingToasts, ui.Toast{
		ID:          title,
		Title:       title,
		Description: detail,
		Type:        "error",
		Timeout:     errorToastTime,
	})
	a.update()
}

// SetToast shows a message near the bottom of the window once.
func (a *App) SetToast(msg string) {
	a.pendingToasts = append(a.pendingToasts, ui.Toast{ID: msg, Title: msg})
	a.update()
}

// SetAppIcon gives the About dialog the icon the app embeds. mygo build
// reads the same PNG for the app's own icon, and mygo.json's
// resourcesExclude keeps it out of the app, so that no file beside the app
// carries it: the binary does.
func (a *App) SetAppIcon(data []byte) {
	a.appIconPNG = data
}

// onUI runs fn on the main thread, or at once before a window exists.
func (a *App) onUI(fn func()) {
	if a.win == nil {
		fn()
		return
	}
	a.win.Update(fn)
}

// askFrame asks a window for a frame built anew. It is a variable so tests
// can see that an action taken outside the view, as a menu item's, asked
// for the frame showing what it changed.
var askFrame = func(w *mygo.Window) {
	if w != nil {
		w.Update(func() {})
	}
}

func (a *App) update() {
	askFrame(a.win)
}

func (a *App) pane(i int) *Pane {
	if i == 1 {
		return &a.right
	}
	return &a.left
}

func (a *App) activePaneObj() *Pane { return a.pane(a.activePane) }

func (a *App) activeTab() *Tab { return a.activePaneObj().activeTab() }

func (a *App) setActivePane(i int) {
	if i != 0 && (i != 1 || !a.split) {
		return
	}
	if a.activePane == i {
		return
	}
	a.activePane = i
	a.focusEditor = true
	a.updateTitle()
	a.update()
}

// activateTab chooses tab i of a pane, as a click on it does, and sees it
// into view once the tab bar is laid out.
func (a *App) activateTab(paneIdx, i int) {
	p := a.pane(paneIdx)
	if i < 0 || i >= len(p.tabs) {
		return
	}
	p.active = i
	a.activePane = paneIdx
	a.focusEditor = true
	a.revealPane, a.revealDoc = paneIdx, p.tabs[i].doc.ID
	a.updateTitle()
	a.update()
}

func (a *App) setDocReadOnly(doc *Document, readOnly bool) {
	doc.ReadOnly = readOnly
	for _, p := range []*Pane{&a.left, &a.right} {
		for _, tab := range p.tabs {
			if tab.doc == doc {
				tab.state.ReadOnly = readOnly
			}
		}
	}
}

func (a *App) findDocument(path string) *Document {
	for _, p := range []*Pane{&a.left, &a.right} {
		for _, tab := range p.tabs {
			if tab.doc.Path == path {
				return tab.doc
			}
		}
	}
	return nil
}

// openDocuments returns the documents open in either pane, once each, in
// tab order: the left pane first, then the right.
func (a *App) openDocuments() []*Document {
	n := len(a.left.tabs) + len(a.right.tabs)
	docs := make([]*Document, 0, n)
	seen := make(map[*Document]bool, n)
	for _, p := range []*Pane{&a.left, &a.right} {
		for _, tab := range p.tabs {
			if !seen[tab.doc] {
				seen[tab.doc] = true
				docs = append(docs, tab.doc)
			}
		}
	}
	return docs
}

// activateDocument shows doc, in the active pane when it is open there and
// in the other pane otherwise.
func (a *App) activateDocument(doc *Document) {
	for _, paneIdx := range []int{a.activePane, 1 - a.activePane} {
		if paneIdx == 1 && !a.split {
			continue
		}
		p := a.pane(paneIdx)
		for i, tab := range p.tabs {
			if tab.doc == doc {
				a.activateTab(paneIdx, i)
				return
			}
		}
	}
}

func (a *App) addDocument(doc *Document) {
	a.addDocumentAt(-1, doc)
}

// addDocumentAt adds doc to a pane, -1 for the active one, sharing the
// document already open elsewhere.
func (a *App) addDocumentAt(paneIdx int, doc *Document) {
	a.errMsg = ""
	if doc.Path != "" {
		if existing := a.findDocument(doc.Path); existing != nil {
			doc = existing
		}
	}
	if (paneIdx != 0 && paneIdx != 1) || (paneIdx == 1 && !a.split) {
		paneIdx = a.activePane
	}
	p := a.pane(paneIdx)
	for i, tab := range p.tabs {
		if tab.doc == doc {
			p.active = i
			a.activePane = paneIdx
			a.focusEditor = true
			a.updateTitle()
			return
		}
	}
	st := &editor.State{Buffer: doc.Buffer, ReadOnly: doc.ReadOnly}
	p.tabs = append(p.tabs, &Tab{doc: doc, state: st})
	p.active = len(p.tabs) - 1
	a.activePane = paneIdx
	a.revealPane, a.revealDoc = paneIdx, doc.ID
	a.focusEditor = true
	a.updateTitle()
}

func (a *App) newUntitledDocument() *Document {
	a.untitledSeq++
	doc := newUntitled(a.untitledSeq, a.Settings.DefaultEncoding)
	doc.Buffer.SetLineEnding(a.Settings.DefaultEOL)
	return doc
}

func (a *App) removeTab(paneIdx, i int) {
	p := a.pane(paneIdx)
	if i < 0 || i >= len(p.tabs) {
		return
	}
	doc := p.tabs[i].doc
	p.tabs = append(p.tabs[:i], p.tabs[i+1:]...)
	switch {
	case len(p.tabs) == 0:
		p.active = -1
	case p.active > i:
		p.active--
	case p.active == i && p.active >= len(p.tabs):
		p.active = len(p.tabs) - 1
	}

	if len(p.tabs) == 0 && a.split {
		a.collapseSplit(paneIdx)
	}
	if paneIdx == a.activePane {
		a.focusEditor = true
	}
	a.pruneSearchDocument(doc)
	a.dropPreviewOptions(doc)
	if a.findTab(doc) == nil {
		// The last tab of the document: its recovery copy goes with it.
		a.forgetDraft(doc)
	}
	a.updateTitle()
}

// collapseSplit closes the split when one of its panes has no tabs left,
// keeping the other pane's tabs, states and choice.
func (a *App) collapseSplit(emptyPane int) {
	if !a.split {
		return
	}
	if emptyPane == 0 {
		a.left = a.right
	}
	a.right = Pane{active: -1}
	a.split = false
	a.activePane = 0
	a.revealPane, a.revealDoc = 0, ""
	if tab := a.left.activeTab(); tab != nil {
		a.revealDoc = tab.doc.ID
	}
	a.focusEditor = true
}

// dropOnPane takes what was dropped on a pane: it moves a tab there, or
// opens a file from the sidebar there.
func (a *App) dropOnPane(d paneDrag, toPane int) {
	if d.kind == dragFile {
		a.openInPane(d.path, toPane)
		return
	}
	a.moveTab(d, toPane)
}

// ensureSplit opens the split with an empty right pane.
func (a *App) ensureSplit() {
	if a.split {
		return
	}
	a.split = true
	a.centerSplit = true
	a.right = Pane{active: -1}
}

// openInPane opens path as a tab in a pane, sharing the document already
// open elsewhere.
func (a *App) openInPane(path string, paneIdx int) {
	a.openPaths([]string{path}, paneIdx)
}

// moveTab moves the tab dragged from one pane into the other, at the end,
// or shows the copy the target pane already has. The source pane's split
// closes when its last tab leaves.
func (a *App) moveTab(from paneDrag, toPane int) {
	if from.kind != dragTab {
		return
	}
	if from.pane == toPane || from.pane < 0 || from.pane > 1 || toPane < 0 || toPane > 1 {
		return
	}
	a.dropTabAt(from, toPane, -1)
}

// moveTabIndex moves a pane's tab into the other pane, at the end, or
// shows the copy the target pane already has. The source pane's split
// closes when its last tab leaves.
func (a *App) moveTabIndex(srcPane, idx, dstPane int) {
	a.moveTabAt(srcPane, idx, dstPane, -1)
}

// dropTabAt moves the tab a paneDrag carries to index to of pane, -1 for
// the end, finding it in its own pane first.
func (a *App) dropTabAt(d paneDrag, pane, to int) {
	for i, tab := range a.pane(d.pane).tabs {
		if tab.doc.ID == d.id {
			a.moveTabAt(d.pane, i, pane, to)
			return
		}
	}
}

// moveTabAt moves a pane's tab to index to of a pane, -1 for the end:
// within one pane that reorders its tabs, which needs to, and across the
// two that moves the tab there, or shows the copy the target already has.
// The source pane's split closes when its last tab leaves.
func (a *App) moveTabAt(srcPane, idx, dstPane, to int) {
	if srcPane < 0 || srcPane > 1 || dstPane < 0 || dstPane > 1 {
		return
	}
	if dstPane == 1 && !a.split {
		return
	}
	src, dst := a.pane(srcPane), a.pane(dstPane)
	if idx < 0 || idx >= len(src.tabs) {
		return
	}
	if srcPane == dstPane {
		// Dropping on the pane alone draws no place to reorder to.
		if to < 0 {
			return
		}
		a.reorderTab(srcPane, idx, to)
		return
	}
	tab := src.tabs[idx]
	if to < 0 || to > len(dst.tabs) {
		to = len(dst.tabs)
	}
	target := -1
	for i, other := range dst.tabs {
		if other.doc == tab.doc {
			target = i
			break
		}
	}

	src.tabs = append(src.tabs[:idx], src.tabs[idx+1:]...)
	switch {
	case len(src.tabs) == 0:
		src.active = -1
	case src.active > idx:
		src.active--
	case src.active == idx && src.active >= len(src.tabs):
		src.active = len(src.tabs) - 1
	}

	if target < 0 {
		dst.tabs = slices.Insert(dst.tabs, to, tab)
		target = to
	}
	dst.active = target
	a.activePane = dstPane

	if a.split && len(src.tabs) == 0 {
		a.collapseSplit(srcPane)
	}
	a.focusEditor = true
	a.revealPane, a.revealDoc = a.activePane, tab.doc.ID
	a.updateTitle()
	a.saveSession()
	a.update()
}

// reorderTab moves a pane's tab to index to, counted before the tab
// leaves its place, keeping the tab chosen chosen.
func (a *App) reorderTab(paneIdx, idx, to int) {
	p := a.pane(paneIdx)
	if idx < 0 || idx >= len(p.tabs) {
		return
	}
	if to < 0 {
		return
	}
	if to > len(p.tabs) {
		to = len(p.tabs)
	}
	// Dropped where it is, or right after itself, the tab stays.
	if to == idx || to == idx+1 {
		return
	}
	tab, chosen := p.tabs[idx], p.activeTab()
	tabs := append(p.tabs[:idx], p.tabs[idx+1:]...)
	if to > idx {
		to--
	}
	p.tabs = slices.Insert(tabs, to, tab)
	if chosen != nil {
		p.active = slices.Index(p.tabs, chosen)
	}
	a.activePane = paneIdx
	a.revealPane, a.revealDoc = paneIdx, tab.doc.ID
	a.updateTitle()
	a.saveSession()
	a.update()
}

// copyTabIndex shows a pane's tab in the other pane, sharing its document,
// or activates the copy that pane already has.
func (a *App) copyTabIndex(srcPane, idx, dstPane int) {
	if srcPane == dstPane || srcPane < 0 || srcPane > 1 || dstPane < 0 || dstPane > 1 {
		return
	}
	if dstPane == 1 && !a.split {
		return
	}
	src, dst := a.pane(srcPane), a.pane(dstPane)
	if idx < 0 || idx >= len(src.tabs) {
		return
	}
	tab := src.tabs[idx]
	target := -1
	for i, other := range dst.tabs {
		if other.doc == tab.doc {
			target = i
			break
		}
	}
	if target < 0 {
		dst.tabs = append(dst.tabs, &Tab{
			doc:   tab.doc,
			state: &editor.State{Buffer: tab.doc.Buffer, ReadOnly: tab.doc.ReadOnly},
		})
		target = len(dst.tabs) - 1
	}
	dst.active = target
	a.activePane = dstPane
	a.focusEditor = true
	a.revealPane, a.revealDoc = dstPane, tab.doc.ID
	a.updateTitle()
	a.saveSession()
	a.update()
}

// openSplit opens the split in the given direction, duplicating the active
// tab into the second pane. A split that is already open only changes
// direction, keeping both panes' tabs, states and choice.
func (a *App) openSplit(vertical bool) {
	if a.split && a.splitVertical == vertical {
		return
	}
	a.splitVertical = vertical
	a.centerSplit = true
	if !a.split {
		a.split = true
		if tab := a.activeTab(); tab != nil {
			found := false
			for i, rt := range a.right.tabs {
				if rt.doc == tab.doc {
					a.right.active = i
					found = true
					break
				}
			}
			if !found {
				a.right.tabs = append(a.right.tabs, &Tab{
					doc:   tab.doc,
					state: &editor.State{Buffer: tab.doc.Buffer, ReadOnly: tab.doc.ReadOnly},
				})
				a.right.active = len(a.right.tabs) - 1
			}
		}
		a.activePane = 1
	}
	a.focusEditor = true
	a.saveSession()
	a.updateTitle()
	a.update()
}

// closeSplit merges the second pane into the first, keeping the tabs only
// it had.
func (a *App) closeSplit() {
	if !a.split {
		return
	}
	for _, rt := range a.right.tabs {
		duplicate := false
		for _, lt := range a.left.tabs {
			if lt.doc == rt.doc {
				duplicate = true
				break
			}
		}
		if !duplicate {
			a.left.tabs = append(a.left.tabs, rt)
		}
	}
	if a.left.active < 0 && len(a.left.tabs) > 0 {
		a.left.active = 0
	}
	a.right = Pane{active: -1}
	a.split = false
	a.activePane = 0
	a.focusEditor = true
	a.saveSession()
	a.updateTitle()
	a.update()
}

func (a *App) toggleSplit() {
	if a.split {
		a.closeSplit()
		return
	}
	a.openSplit(a.splitVertical)
}

func (a *App) updateTitle() {
	title := "ZPad"
	dirty := false
	tab := a.activeTab()
	if tab != nil {
		title = tab.doc.Name() + " — ZPad"
		dirty = tab.doc.Buffer.Modified()
	} else if a.root != "" {
		title = filepath.Base(a.root) + " — ZPad"
	}
	a.lastDirty = dirty
	if a.win != nil {
		a.win.SetTitle(title)
		a.win.SetDocumentEdited(dirty)
		if tab != nil {
			a.win.SetRepresentedFile(tab.doc.Path)
		} else {
			a.win.SetRepresentedFile("")
		}
	}
}

func (a *App) setWorkspace(root string) {
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	a.errMsg = ""
	a.root = root
	a.tree = files.New(root, a.Settings.ShowHidden)
	a.treeSelected = -1
	a.treeState = ui.OutlineState[string]{}
	a.treeGeneration++
	a.sidebarTab = sidebarWorkspace
	a.Settings.SidebarOpen = true
	a.saveSettings()
	a.saveSession()
	a.update()
}

func (a *App) toggleSidebar() {
	a.Settings.SidebarOpen = !a.Settings.SidebarOpen
	a.saveSettings()
	a.saveSession()
	a.update()
}

func (a *App) refreshTree() {
	if a.tree != nil {
		a.tree.Refresh()
		a.treeGeneration++
		a.update()
	}
}

func (a *App) toggleHidden() {
	a.Settings.ShowHidden = !a.Settings.ShowHidden
	if a.tree != nil {
		a.tree.SetShowHidden(a.Settings.ShowHidden)
		a.treeGeneration++
	}
	a.saveSettings()
	a.update()
}

// setThemeID switches to the theme id, saves it and redraws. An unknown id
// falls back to following the system, so a settings file cannot leave the
// app without a palette.
func (a *App) setThemeID(id string) {
	id = theme.Normalize(id)
	a.Settings.Theme = id
	def := theme.Resolve(id, a.Dark)
	a.Dark = def.Dark
	if a.running {
		SetThemeSource(id)
		if id == theme.SystemID {
			a.Dark = mygo.Theme.IsDark()
		}
	}
	a.saveSettings()
	a.update()
}

// themeDefinition is the theme the window draws with right now.
func (a *App) themeDefinition() *theme.Definition {
	return theme.Resolve(a.Settings.Theme, a.Dark)
}

func (a *App) saveSettings() {
	if a.SettingsDir == "" {
		return
	}
	if err := settings.Save(a.SettingsDir, a.Settings); err != nil {
		a.errMsg = locales.SaveSettingsFailed() + ": " + err.Error()
		log.Printf("save settings: %v", err)
		return
	}
	a.errMsg = ""
}
