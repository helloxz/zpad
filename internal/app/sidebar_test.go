package app

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/helloxz/zpad/internal/session"
	"github.com/helloxz/zpad/internal/settings"
	"github.com/helloxz/zpad/locales"
)

func TestOpenDocumentsAreUniqueAndOrdered(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	aDoc := docAt("/tmp/a.txt", "a")
	bDoc := docAt("/tmp/b.txt", "b")
	cDoc := docAt("/tmp/c.txt", "c")
	a.addDocument(aDoc)
	a.addDocument(bDoc)
	a.toggleSplit() // right duplicates b
	a.addDocument(cDoc)

	got := a.openDocuments()
	if len(got) != 3 || got[0] != aDoc || got[1] != bDoc || got[2] != cDoc {
		t.Fatalf("open documents %v, want a, b, c once each", got)
	}
}

func TestActivateDocumentChoosesPane(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	aDoc := docAt("/tmp/a.txt", "a")
	bDoc := docAt("/tmp/b.txt", "b")
	a.addDocument(aDoc)
	a.toggleSplit() // right duplicates a; the right pane is active
	a.addDocument(bDoc)

	a.activateDocument(aDoc)
	if a.activePane != 1 || a.right.active != 0 {
		t.Fatalf("active pane %d, tab %d; want the right pane's a", a.activePane, a.right.active)
	}

	a.setActivePane(0)
	a.activateDocument(bDoc)
	if a.activePane != 1 || a.right.tabs[a.right.active].doc != bDoc {
		t.Fatalf("active pane %d, tab %d; want the right pane's b", a.activePane, a.right.active)
	}
}

func TestSidebarTabs(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.Settings.SidebarOpen = true
	a.addDocument(docAt("/tmp/a.txt", "a"))

	tt := ui.NewTester(a.View, 900, 600)
	for _, want := range []string{locales.OpenFiles(), locales.Workspace(), "a.txt"} {
		if !tt.HasText(want) {
			t.Errorf("sidebar does not show %q; texts %q", want, tt.Texts())
		}
	}
	if err := tt.Click(locales.Workspace()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText(locales.NoFolder()) {
		t.Errorf("workspace tab shows no empty state; texts %q", tt.Texts())
	}
	if err := tt.Click(locales.OpenFiles()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("a.txt") {
		t.Errorf("open files tab does not show the file; texts %q", tt.Texts())
	}
}

func TestSidebarEmptyOpenFiles(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.Settings.SidebarOpen = true
	tt := ui.NewTester(a.View, 900, 600)
	if !tt.HasText(locales.NoOpenFiles()) {
		t.Fatalf("sidebar does not show the empty state; texts %q", tt.Texts())
	}
}

func TestWorkspaceTabOnOpenFolder(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.setWorkspace(t.TempDir())
	if a.sidebarTab != sidebarWorkspace {
		t.Fatalf("sidebar tab %d, want the workspace", a.sidebarTab)
	}
	if !a.Settings.SidebarOpen {
		t.Fatal("sidebar did not open")
	}
}

func TestTreeClickSelectsRow(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "docs")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	a := New(settings.Default(), t.TempDir())
	a.setWorkspace(dir)
	a.Settings.SidebarOpen = true

	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()
	if err := tt.Click("docs"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.treeSelected < 0 || a.treeSelected >= a.treeState.Rows() || a.treeState.Item(a.treeSelected) != sub {
		t.Fatalf("tree selection %d is not the docs folder", a.treeSelected)
	}
}

func TestTreeFileClickSelectsRow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := New(settings.Default(), t.TempDir())
	a.setWorkspace(dir)
	a.Settings.SidebarOpen = true

	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()
	row, ok := tt.Find("a.txt")
	if !ok {
		t.Fatal("no a.txt in the tree")
	}
	// Open the file first, at the row's place, so the click takes the
	// fast path and changes no document off the UI thread.
	a.addDocument(docAt(path, "a"))
	a.treeSelected = -1
	tt.ClickAt(row.X+row.W/2, row.Y+row.H/2)
	tt.Frame()
	if a.treeSelected < 0 || a.treeSelected >= a.treeState.Rows() || a.treeState.Item(a.treeSelected) != path {
		t.Fatalf("tree selection %d is not a.txt", a.treeSelected)
	}
}

// TestTreeRowClickAnywhereOpens clicks a tree row left of its name, on the
// indentation, and right of it, on the row's blank end: the whole row opens
// the file, not only its text, where the outline's own row would just
// choose it.
func TestTreeRowClickAnywhereOpens(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "docs")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sub, "a.txt")
	if err := os.WriteFile(path, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := New(settings.Default(), t.TempDir())
	a.setWorkspace(dir)
	a.Settings.SidebarOpen = true
	a.treeState.Open.Add(sub) // shows a.txt one level deep
	// Both are open already, so a click takes the fast path and changes no
	// document off the UI thread.
	doc := docAt(path, "a")
	other := docAt(filepath.Join(dir, "b.txt"), "b")
	a.addDocument(doc)
	a.addDocument(other)

	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()
	row, ok := tt.Find("a.txt")
	if !ok {
		t.Fatalf("no a.txt in the tree; texts %q", tt.Texts())
	}
	y := row.Y + row.H/2
	for _, x := range []float32{4, 20, a.Settings.SidebarWidth - 12} {
		a.activateDocument(other)
		if tab := a.activeTab(); tab == nil || tab.doc != other {
			t.Fatal("setup: b is not the active tab")
		}
		tt.ClickAt(x, y)
		tt.Frame()
		if tab := a.activeTab(); tab == nil || tab.doc != doc {
			t.Fatalf("a click at x=%v did not open a.txt; the active tab is %v", x, a.activeTab())
		}
	}
}

// TestTreeFolderGutterClickToggles opens and closes a folder with clicks
// left of its name, where its arrow is drawn.
func TestTreeFolderGutterClickToggles(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "docs")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	a := New(settings.Default(), t.TempDir())
	a.setWorkspace(dir)
	a.Settings.SidebarOpen = true

	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()
	row, ok := tt.Find("docs")
	if !ok {
		t.Fatalf("no docs in the tree; texts %q", tt.Texts())
	}
	y := row.Y + row.H/2
	tt.ClickAt(4, y)
	tt.Frame()
	if !a.treeState.Open.Has(sub) {
		t.Fatal("a click left of the name did not open the folder")
	}
	tt.Advance(600 * time.Millisecond)
	tt.ClickAt(4, y)
	tt.Frame()
	if a.treeState.Open.Has(sub) {
		t.Fatal("a second click left of the name did not close the folder")
	}
}

// TestTreeOptionClickOpensAll opens and closes a folder with the folders
// inside it, as Option-clicking a row does.
func TestTreeOptionClickOpensAll(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "docs")
	deeper := filepath.Join(sub, "notes")
	if err := os.MkdirAll(deeper, 0o755); err != nil {
		t.Fatal(err)
	}
	a := New(settings.Default(), t.TempDir())
	a.setWorkspace(dir)
	a.Settings.SidebarOpen = true

	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()
	row, ok := tt.Find("docs")
	if !ok {
		t.Fatalf("no docs in the tree; texts %q", tt.Texts())
	}
	tt.ClickAtWith(ui.Alt, 4, row.Y+row.H/2)
	tt.Frame()
	if !a.treeState.Open.Has(sub) || !a.treeState.Open.Has(deeper) {
		t.Fatalf("an Option-click did not open docs and notes: %v", a.treeState.Open)
	}
	tt.Advance(600 * time.Millisecond)
	if row, ok = tt.Find("docs"); !ok {
		t.Fatal("docs left the tree")
	}
	tt.ClickAtWith(ui.Alt, 4, row.Y+row.H/2)
	tt.Frame()
	if a.treeState.Open.Has(sub) || a.treeState.Open.Has(deeper) {
		t.Fatalf("a second Option-click did not close docs and notes: %v", a.treeState.Open)
	}
}

// TestSidebarTipSitsBesideTheRow hovers a file row: the path tip shows to
// the right of the sidebar, on the row's line, so it neither covers the
// list nor the sidebar's header.
func TestSidebarTipSitsBesideTheRow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "internal", "app", "settings_ui.go")
	a := New(settings.Default(), t.TempDir())
	a.root = dir
	a.Settings.SidebarOpen = true
	a.addDocument(docAt(path, "x"))

	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()
	row, ok := tt.Find("settings_ui.go")
	if !ok {
		t.Fatalf("no open file row; texts %q", tt.Texts())
	}
	tt.Move(row.X+row.W/2, row.Y+row.H/2)
	tt.Advance(700 * time.Millisecond)
	tt.Frame()

	tip, ok := tt.Find(path)
	if !ok {
		t.Fatalf("hovering the row showed no path tip; texts %q", tt.Texts())
	}
	if tip.X < row.X+row.W {
		t.Fatalf("tip starts at x=%v, inside the sidebar (row ends at %v)", tip.X, row.X+row.W)
	}
	if tip.Y > row.Y+row.H || tip.Y+tip.H < row.Y {
		t.Fatalf("tip y=%v h=%v is off the row's line (row y=%v h=%v)", tip.Y, tip.H, row.Y, row.H)
	}
}

// TestSidebarTabTipSitsBelow hovers the workspace tab: the path tip shows
// under the tab, left edges aligned, so it covers neither the toolbar
// above nor the tabs beside it.
func TestSidebarTabTipSitsBelow(t *testing.T) {
	dir := t.TempDir()
	a := New(settings.Default(), t.TempDir())
	a.setWorkspace(dir)
	a.Settings.SidebarOpen = true
	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()

	tab, ok := tt.Find(locales.Workspace())
	if !ok {
		t.Fatalf("no workspace tab; texts %q", tt.Texts())
	}
	tt.Move(tab.X+tab.W/2, tab.Y+tab.H/2)
	tt.Advance(700 * time.Millisecond)
	tt.Frame()

	tip, ok := tt.Find(dir)
	if !ok {
		t.Fatalf("hovering the tab showed no path tip; texts %q", tt.Texts())
	}
	if tip.Y < tab.Y+tab.H {
		t.Fatalf("tip y=%v is not under the tab (tab ends at %v)", tip.Y, tab.Y+tab.H)
	}
	if tip.Y < toolbarHeight {
		t.Fatalf("tip y=%v is over the toolbar", tip.Y)
	}
	// Find returns the tip's text, inset by the tip's own padding; a tip
	// centered on the tab would start far to the left of it.
	if tip.X < tab.X || tip.X > tab.X+24 {
		t.Fatalf("tip text x=%v is not left aligned with the tab at x=%v", tip.X, tab.X)
	}
}

// TestOpenFilesShowNamesOnly opens two files of the same name from
// different folders: the list shows just the name for each, and the tip on
// each row tells them apart by path.
func TestOpenFilesShowNamesOnly(t *testing.T) {
	dir := t.TempDir()
	appPath := filepath.Join(dir, "internal", "app", "view.go")
	themePath := filepath.Join(dir, "internal", "theme", "view.go")
	a := New(settings.Default(), t.TempDir())
	a.root = dir
	a.Settings.SidebarOpen = true
	a.addDocument(docAt(appPath, "x"))
	a.addDocument(docAt(themePath, "x"))

	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()
	for _, text := range tt.Texts() {
		if text == "internal/app" || text == "internal/theme" {
			t.Fatalf("the list still shows a directory column: %q", text)
		}
	}
	row, ok := tt.Find("view.go")
	if !ok {
		t.Fatalf("no open file row; texts %q", tt.Texts())
	}
	hover := func(y float32) {
		t.Helper()
		tt.Move(row.X+row.W/2, y)
		tt.Advance(700 * time.Millisecond)
		tt.Frame()
	}
	hover(row.Y + row.H/2)
	if _, ok := tt.Find(appPath); !ok {
		t.Fatalf("the first row's tip does not name its path; texts %q", tt.Texts())
	}
	hover(row.Y + row.H*3/2)
	if _, ok := tt.Find(themePath); !ok {
		t.Fatalf("the second row's tip does not name its path; texts %q", tt.Texts())
	}
}

// TestSidebarMenuByTab keeps the workspace tree's actions on the workspace
// tab: the open files tab has nothing to refresh and no tree to filter.
func TestSidebarMenuByTab(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := New(settings.Default(), t.TempDir())
	a.setWorkspace(dir)

	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()
	a.sidebarTab = sidebarFiles
	tt.Frame()
	if err := tt.Click(locales.SidebarMenu()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if menu := tt.Menu(); len(menu) != 1 || menu[0] != locales.CloseSidebar() {
		t.Fatalf("open files menu %q, want only %q", menu, locales.CloseSidebar())
	}
	tt.CloseMenu()

	a.sidebarTab = sidebarWorkspace
	tt.Frame()
	if err := tt.Click(locales.SidebarMenu()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	want := []string{locales.Refresh(), locales.ShowHidden(), "-", locales.CloseWorkspace(), locales.CloseSidebar()}
	if menu := tt.Menu(); !slices.Equal(menu, want) {
		t.Fatalf("workspace menu %q, want %q", menu, want)
	}
}

// TestCloseWorkspaceMenu drops the folder alone: the documents stay open,
// the sidebar stays out, and the next launch starts without a workspace.
func TestCloseWorkspaceMenu(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(file, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := New(settings.Default(), t.TempDir())
	a.setWorkspace(dir)
	a.addDocument(docAt(file, "a"))

	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()
	if err := tt.Click(locales.SidebarMenu()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.ChooseMenuItem(locales.CloseWorkspace()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()

	if a.root != "" || a.tree != nil {
		t.Fatalf("workspace %q, tree %v; want none", a.root, a.tree)
	}
	if a.sidebarTab != sidebarFiles {
		t.Fatalf("sidebar tab %d, want the open files tab", a.sidebarTab)
	}
	if !a.Settings.SidebarOpen {
		t.Fatal("closing the workspace closed the sidebar")
	}
	if len(a.left.tabs) != 1 || a.left.tabs[0].doc.Path != file {
		t.Fatalf("tabs %v, want %q kept", a.left.tabs, file)
	}
	st, err := session.Load(a.SettingsDir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Root != "" {
		t.Fatalf("session root %q, want none", st.Root)
	}
	if !tt.HasText("a.txt") {
		t.Fatalf("the open files tab does not list the tab; texts %q", tt.Texts())
	}

	// Without a workspace the menu offers nothing tree-specific but the
	// hidden files switch.
	a.sidebarTab = sidebarWorkspace
	tt.Frame()
	if err := tt.Click(locales.SidebarMenu()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	want := []string{locales.Refresh(), locales.ShowHidden(), "-", locales.CloseSidebar()}
	if menu := tt.Menu(); !slices.Equal(menu, want) {
		t.Fatalf("menu without a workspace %q, want %q", menu, want)
	}
}
