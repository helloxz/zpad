package app

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helloxz/zpad/internal/session"
	"github.com/helloxz/zpad/internal/settings"
	"github.com/helloxz/zpad/locales"
)

// homeTestApp opens dir as the Home tab's folder, with the sidebar showing
// it.
func homeTestApp(t *testing.T, dir string) (*App, *ui.Tester) {
	t.Helper()
	a := New(settings.Default(), t.TempDir())
	a.setHomeRoot(dir)
	a.Settings.SidebarOpen = true
	a.sidebarTab = sidebarHome
	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()
	return a, tt
}

// The Home tab lists the user's folder, with the three tabs and the menu
// button all still on the bar.
func TestHomeTabShowsTheHomeFolder(t *testing.T) {
	dir := t.TempDir()
	treeFile(t, dir, "a.txt")
	_, tt := homeTestApp(t, dir)

	for _, text := range []string{locales.OpenFiles(), locales.Workspace(), locales.Home(), locales.SidebarMenu()} {
		if !tt.HasText(text) {
			t.Fatalf("the sidebar tab bar does not show %q; texts %q", text, tt.Texts())
		}
	}
	if !tt.HasText("a.txt") {
		t.Fatalf("the Home tab does not list a.txt; texts %q", tt.Texts())
	}
}

// A file clicked in the Home tab opens as a preview tab, as one clicked in
// the workspace does.
func TestHomeClickOpensPreviewTab(t *testing.T) {
	dir := t.TempDir()
	path := treeFile(t, dir, "a.txt")
	a, tt := homeTestApp(t, dir)

	if err := tt.Click("a.txt"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "a.txt to open", func() bool { return len(a.left.tabs) == 1 })
	if tab := a.left.tabs[0]; !tab.temporary || tab.doc.Path != path {
		t.Fatalf("the Home tab opened %q temporary=%v, want a preview of %q",
			tab.doc.Path, tab.temporary, path)
	}
}

// The Home tab deletes single files but never a folder: losing a whole
// folder under the home directory is too much to take back.
func TestHomeTreeMenuDeletesFilesOnly(t *testing.T) {
	dir := t.TempDir()
	treeFile(t, dir, "a.txt")
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, tt := homeTestApp(t, dir)

	if err := tt.RightClick("a.txt"); err != nil {
		t.Fatal(err)
	}
	menu := tt.Menu()
	for _, want := range []string{locales.Rename(), locales.Delete()} {
		if !menuHas(menu, want) {
			t.Fatalf("the Home file menu %q does not offer %q", menu, want)
		}
	}

	tt.Frame()
	if err := tt.RightClick("docs"); err != nil {
		t.Fatal(err)
	}
	menu = tt.Menu()
	for _, want := range []string{locales.NewFile(), locales.NewFolder(), locales.OpenAsWorkspace(), locales.Rename()} {
		if !menuHas(menu, want) {
			t.Fatalf("the Home folder menu %q does not offer %q", menu, want)
		}
	}
	if menuHas(menu, locales.Delete()) {
		t.Fatalf("the Home folder menu %q offers Delete", menu)
	}
}

// Deleting a file from the Home tab takes it off the disk, closes its tab
// and out of the tree.
func TestHomeDeletesAFile(t *testing.T) {
	dir := t.TempDir()
	path := treeFile(t, dir, "a.txt")
	a, tt := homeTestApp(t, dir)

	if err := tt.Click("a.txt"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "a.txt to open", func() bool { return len(a.left.tabs) == 1 })

	a.deletePath(path, false)
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the file is still on the disk: %v", err)
	}
	if len(a.left.tabs) != 0 {
		t.Fatalf("%d tabs left, want the deleted file's closed", len(a.left.tabs))
	}
	if slices.Contains(a.dirTree.Children(dir), path) {
		t.Fatalf("the Home tree still lists %q", path)
	}
}

// The workspace tree keeps its Delete: only the Home tree leaves it out.
func TestWorkspaceTreeMenuKeepsDelete(t *testing.T) {
	dir := t.TempDir()
	treeFile(t, dir, "a.txt")
	a := New(settings.Default(), t.TempDir())
	a.setWorkspace(dir)
	a.Settings.SidebarOpen = true
	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()

	if err := tt.RightClick("a.txt"); err != nil {
		t.Fatal(err)
	}
	if !menuHas(tt.Menu(), locales.Delete()) {
		t.Fatalf("the workspace menu %q lost Delete", tt.Menu())
	}
}

// Creating a file from the Home tab shows it in that tree, and opens it.
func TestHomeCreateShowsTheFile(t *testing.T) {
	dir := t.TempDir()
	a, _ := homeTestApp(t, dir)

	a.beginCreate(dir, false)
	a.createValue = "new.txt"
	a.applyCreate()
	path := filepath.Join(dir, "new.txt")
	eventually(t, "the new file to open", func() bool {
		return a.findDocument(path) != nil
	})
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the file was not created: %v", err)
	}
	if !slices.Contains(a.dirTree.Children(dir), path) {
		t.Fatalf("the Home tree does not list the new file: %v", a.dirTree.Children(dir))
	}
}

// Renaming a file from the Home tab moves the open tab's path with it and
// refreshes the tree.
func TestHomeRenameRefreshesTheTree(t *testing.T) {
	dir := t.TempDir()
	oldPath := treeFile(t, dir, "a.txt")
	a, tt := homeTestApp(t, dir)

	if err := tt.Click("a.txt"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "a.txt to open", func() bool { return len(a.left.tabs) == 1 })

	a.beginRename(oldPath)
	a.renameValue = "b.txt"
	a.applyRename()
	newPath := filepath.Join(dir, "b.txt")
	eventually(t, "the rename to finish", func() bool {
		_, err := os.Stat(newPath)
		return err == nil
	})
	if got := a.left.tabs[0].doc.Path; got != newPath {
		t.Fatalf("the open tab still shows %q, want %q", got, newPath)
	}
	children := a.dirTree.Children(dir)
	if !slices.Contains(children, newPath) || slices.Contains(children, oldPath) {
		t.Fatalf("the Home tree shows %v, want %q only", children, newPath)
	}
	if a.dirSelected != -1 {
		t.Fatalf("the tree still chooses row %d after the rename", a.dirSelected)
	}
}

// A folder opened from the Home tab becomes the workspace, and the sidebar
// switches to that tab.
func TestHomeOpensFolderAsWorkspace(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "docs")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	a, tt := homeTestApp(t, dir)

	if err := tt.RightClick("docs"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(locales.OpenAsWorkspace()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.root != sub || a.tree == nil || a.tree.Root() != sub {
		t.Fatalf("workspace %q tree %v, want %q", a.root, a.tree, sub)
	}
	if a.sidebarTab != sidebarWorkspace {
		t.Fatalf("sidebar tab %d, want the workspace", a.sidebarTab)
	}
}

// Hidden files follow the same setting in both trees.
func TestHomeFollowsShowHidden(t *testing.T) {
	dir := t.TempDir()
	treeFile(t, dir, ".hidden")
	a, _ := homeTestApp(t, dir)

	hidden := filepath.Join(dir, ".hidden")
	if slices.Contains(a.dirTree.Children(dir), hidden) {
		t.Fatal("the Home tree lists a hidden file before Show Hidden is on")
	}
	a.toggleHidden()
	if !slices.Contains(a.dirTree.Children(dir), hidden) {
		t.Fatal("Show Hidden did not reach the Home tree")
	}
}

// Coming back to the Home tab reads the folder again, so files another
// program made show without a manual refresh.
func TestHomeTabRefreshesWhenShown(t *testing.T) {
	dir := t.TempDir()
	treeFile(t, dir, "a.txt")
	a, tt := homeTestApp(t, dir)

	if err := tt.Click(locales.OpenFiles()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	treeFile(t, dir, "late.txt")
	if err := tt.Click(locales.Home()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("late.txt") {
		t.Fatalf("the Home tab did not re-read the folder; texts %q", tt.Texts())
	}
	if a.sidebarTab != sidebarHome {
		t.Fatalf("sidebar tab %d, want Home", a.sidebarTab)
	}
}

// The sidebar menu points at the Home tree while its tab is chosen: it
// offers a refresh and the hidden files switch, and no workspace close.
func TestHomeSidebarMenu(t *testing.T) {
	dir := t.TempDir()
	_, tt := homeTestApp(t, dir)

	if err := tt.Click(locales.SidebarMenu()); err != nil {
		t.Fatal(err)
	}
	want := []string{locales.Refresh(), locales.ShowHidden(), "-", locales.CloseSidebar()}
	if menu := tt.Menu(); !slices.Equal(menu, want) {
		t.Fatalf("menu %q, want %q", menu, want)
	}
}

// Renaming the workspace's own folder from the Home tab moves the
// workspace with it: its tree reads the new path instead of going empty.
func TestHomeRenameWorkspaceRoot(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "proj")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	file := treeFile(t, root, "a.txt")

	a := New(settings.Default(), t.TempDir())
	a.setHomeRoot(dir)
	a.Settings.SidebarOpen = true
	a.setWorkspace(root)
	a.addDocument(docAt(file, "x"))

	renamed := filepath.Join(dir, "proj2")
	a.beginRename(root)
	a.renameValue = "proj2"
	a.applyRename()

	if a.root != renamed {
		t.Fatalf("workspace root %q, want %q", a.root, renamed)
	}
	if a.tree == nil || a.tree.Root() != renamed || a.tree.Name() != "proj2" {
		t.Fatalf("workspace tree %v, want its root at %q", a.tree, renamed)
	}
	roots := a.tree.Roots()
	if len(roots) != 1 || roots[0] != filepath.Join(renamed, "a.txt") {
		t.Fatalf("workspace list %v, want the renamed folder's file", roots)
	}
	if got := a.left.tabs[0].doc.Path; got != filepath.Join(renamed, "a.txt") {
		t.Fatalf("the open tab still shows %q", got)
	}
	// The tab is still the workspace's, and its list is back under the new
	// name: the header no longer shows the old folder.
	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()
	if !tt.HasText("proj2") {
		t.Fatalf("the workspace tab does not show its new name; texts %q", tt.Texts())
	}
	st, err := session.Load(a.SettingsDir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Root != renamed {
		t.Fatalf("the session keeps root %q, want %q", st.Root, renamed)
	}
}

// Renaming a folder above the workspace moves the workspace with it too.
func TestHomeRenameParentOfWorkspaceRoot(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "parent")
	root := filepath.Join(parent, "app")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	treeFile(t, root, "a.txt")

	a := New(settings.Default(), t.TempDir())
	a.setHomeRoot(dir)
	a.Settings.SidebarOpen = true
	a.setWorkspace(root)

	a.beginRename(parent)
	a.renameValue = "moved"
	a.applyRename()

	moved := filepath.Join(dir, "moved")
	newRoot := filepath.Join(moved, "app")
	if a.root != newRoot {
		t.Fatalf("workspace root %q, want %q", a.root, newRoot)
	}
	if a.tree == nil || a.tree.Root() != newRoot {
		t.Fatalf("workspace tree %v, want its root at %q", a.tree, newRoot)
	}
	roots := a.tree.Roots()
	if len(roots) != 1 || roots[0] != filepath.Join(newRoot, "a.txt") {
		t.Fatalf("workspace list %v, want the file under the new root", roots)
	}
	children := a.dirTree.Children(dir)
	if slices.Contains(children, parent) || !slices.Contains(children, moved) {
		t.Fatalf("the Home tree shows %v, want %q in place of %q", children, moved, parent)
	}
}
