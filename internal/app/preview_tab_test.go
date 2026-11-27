package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helloxz/zpad/internal/settings"
	"github.com/helloxz/zpad/locales"
)

// treeFile writes name into dir and returns its path.
func treeFile(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(name), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// previewTestApp opens dir as the workspace with the sidebar showing it.
func previewTestApp(t *testing.T, dir string) (*App, *ui.Tester) {
	t.Helper()
	a := New(settings.Default(), t.TempDir())
	a.setWorkspace(dir)
	a.Settings.SidebarOpen = true
	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()
	return a, tt
}

// A single click in the workspace tree opens a preview tab: the file shows
// at once, but the next single click may take its place.
func TestTreeClickOpensPreviewTab(t *testing.T) {
	dir := t.TempDir()
	path := treeFile(t, dir, "a.txt")
	a, tt := previewTestApp(t, dir)

	if err := tt.Click("a.txt"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "a.txt to open", func() bool { return len(a.left.tabs) == 1 })
	if tab := a.left.tabs[0]; !tab.temporary {
		t.Fatal("the clicked file did not open as a preview tab")
	}
	if tab := a.left.tabs[0]; tab.doc.Path != path {
		t.Fatalf("the tree opened %q, want %q", tab.doc.Path, path)
	}
	if a.left.active != 0 {
		t.Fatalf("active tab %d, want the opened one", a.left.active)
	}
}

// The next single click replaces the preview tab rather than adding a
// second one: browsing files leaves no stack of tabs behind.
func TestSecondTreeClickReplacesPreviewTab(t *testing.T) {
	dir := t.TempDir()
	treeFile(t, dir, "a.txt")
	bPath := treeFile(t, dir, "b.txt")
	a, tt := previewTestApp(t, dir)

	if err := tt.Click("a.txt"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "a.txt to open", func() bool { return len(a.left.tabs) == 1 })
	tt.Frame()

	if err := tt.Click("b.txt"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "b.txt to replace a.txt", func() bool {
		return len(a.left.tabs) == 1 && a.left.tabs[0].doc.Path == bPath
	})
	if !a.left.tabs[0].temporary {
		t.Fatal("the second file did not open as a preview tab")
	}
	if a.left.active != 0 {
		t.Fatalf("active tab %d, want the replaced one", a.left.active)
	}
}

// A double click keeps the tab the first click opened: it is no longer the
// next single click's to replace.
func TestTreeDoubleClickKeepsTheTab(t *testing.T) {
	dir := t.TempDir()
	treeFile(t, dir, "a.txt")
	a, tt := previewTestApp(t, dir)

	row, ok := tt.Find("a.txt")
	if !ok {
		t.Fatalf("no a.txt in the tree; texts %q", tt.Texts())
	}
	doubleClick(tt, row.X+row.W/2, row.Y+row.H/2)
	eventually(t, "a.txt to open", func() bool { return len(a.left.tabs) == 1 })
	if a.left.tabs[0].temporary {
		t.Fatal("a double click left the tab a preview")
	}
}

// A double click answered before the file arrives still keeps it: the read
// finishes on its own goroutine and opens the tab for good.
func TestDoubleClickBeforeTheFileArrivesKeepsIt(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	path := "/tmp/slow.txt"
	a.pinPath(path)
	a.addPreviewDocumentAt(-1, docAt(path, "slow"))

	if len(a.left.tabs) != 1 || a.left.tabs[0].temporary {
		t.Fatalf("a kept file opened as a preview: %+v", a.left.tabs)
	}
}

// Editing a preview tab keeps it, so that typing is never lost to the next
// single click.
func TestEditingKeepsAPreviewTab(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	doc := docAt("/tmp/a.txt", "a")
	a.addPreviewDocumentAt(-1, doc)
	if !a.left.tabs[0].temporary {
		t.Fatal("setup: the file is not a preview tab")
	}

	doc.Buffer.Set("edited")
	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()
	if a.left.tabs[0].temporary {
		t.Fatal("an edited preview tab stayed a preview")
	}
}

// A preview open replaces only a preview tab: a tab opened for good keeps
// its place.
func TestPreviewClickKeepsPermanentTabs(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "a"))
	a.addPreviewDocumentAt(-1, docAt("/tmp/b.txt", "b"))

	if len(a.left.tabs) != 2 {
		t.Fatalf("preview open left %d tabs, want 2", len(a.left.tabs))
	}
	if a.left.tabs[0].temporary || !a.left.tabs[1].temporary {
		t.Fatalf("tabs are %v and %v, want a permanent one then a preview",
			a.left.tabs[0].temporary, a.left.tabs[1].temporary)
	}
}

// Replacing a pane's only tab with a preview tab keeps the pane and its
// split: the tab changes in place rather than leaving the pane empty.
func TestReplacingTheOnlyPreviewTabKeepsTheSplit(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "a"))
	a.toggleSplit()
	if !a.split || len(a.right.tabs) != 1 {
		t.Fatalf("setup: split %v, right tabs %d", a.split, len(a.right.tabs))
	}
	a.right.tabs[0].temporary = true

	b := docAt("/tmp/b.txt", "b")
	a.addPreviewDocumentAt(1, b)
	if !a.split {
		t.Fatal("replacing the only tab of the right pane collapsed the split")
	}
	if len(a.right.tabs) != 1 || a.right.tabs[0].doc != b || !a.right.tabs[0].temporary {
		t.Fatalf("right pane holds %d tabs, want b as its one preview", len(a.right.tabs))
	}
	if len(a.left.tabs) != 1 || a.left.tabs[0].doc.Path != "/tmp/a.txt" {
		t.Fatalf("the other pane changed: %d tabs", len(a.left.tabs))
	}
}

// The tab's own menu keeps a preview tab, for users who never double click.
func TestTabMenuKeepsAPreviewTab(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addPreviewDocumentAt(-1, docAt("/tmp/a.txt", "a"))
	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()

	if err := tt.RightClick("a.txt"); err != nil {
		t.Fatal(err)
	}
	if !menuHas(tt.Menu(), locales.KeepTabOpen()) {
		t.Fatalf("tab menu %q does not offer to keep the tab", tt.Menu())
	}
	if err := tt.ChooseMenuItem(locales.KeepTabOpen()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.left.tabs[0].temporary {
		t.Fatal("the menu left the tab a preview")
	}
}

// A double click on the tab itself keeps a preview tab too, as editors
// that take the gesture do.
func TestDoubleClickOnTabKeepsAPreviewTab(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addPreviewDocumentAt(-1, docAt("/tmp/a.txt", "a"))
	tt := ui.NewTester(a.View, 900, 500)
	tt.Frame()

	r, ok := tt.Find("a.txt")
	if !ok {
		t.Fatalf("no tab for a.txt; texts %q", tt.Texts())
	}
	doubleClick(tt, r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	if len(a.left.tabs) != 1 {
		t.Fatalf("the double click left %d tabs, want 1", len(a.left.tabs))
	}
	if a.left.tabs[0].temporary {
		t.Fatal("the double click left the tab a preview")
	}
}

// With preview tabs off, a click opens the file for good: the setting
// gives back the behaviour before preview tabs.
func TestPreviewTabsOffOpensForGood(t *testing.T) {
	s := settings.Default()
	s.PreviewTabs = false
	a := New(s, t.TempDir())
	a.addPreviewDocumentAt(-1, docAt("/tmp/a.txt", "a"))
	if a.left.tabs[0].temporary {
		t.Fatal("a preview tab opened with preview tabs off")
	}

	dir := t.TempDir()
	treeFile(t, dir, "b.txt")
	b := New(s, t.TempDir())
	b.setWorkspace(dir)
	b.Settings.SidebarOpen = true
	tt := ui.NewTester(b.View, 900, 600)
	tt.Frame()
	if err := tt.Click("b.txt"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "b.txt to open", func() bool { return len(b.left.tabs) == 1 })
	if b.left.tabs[0].temporary {
		t.Fatal("the tree click opened a preview tab with preview tabs off")
	}
}
