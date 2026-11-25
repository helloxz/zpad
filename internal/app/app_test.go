package app

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helloxz/zpad/internal/settings"
	"github.com/helloxz/zpad/internal/theme"
	"github.com/helloxz/zpad/locales"
)

func TestViewShowsSkeleton(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	tt := ui.NewTester(a.View, 1200, 800)

	if !tt.HasText("ZPad") {
		t.Fatalf("view does not show the app name; texts %q", tt.Texts())
	}
	if !tt.HasText(locales.AppTagline()) {
		t.Fatalf("view does not show the empty state; texts %q", tt.Texts())
	}
	if !tt.HasText("ZPad " + Version) {
		t.Fatalf("status bar does not show the version; texts %q", tt.Texts())
	}
}

// main sets Version from mygo.json, which mygo build links in: the app has
// no version of its own to miss.
func TestSetVersionReachesTheView(t *testing.T) {
	old := Version
	t.Cleanup(func() { Version = old })

	SetVersion("9.9.9")
	a := New(settings.Default(), t.TempDir())
	tt := ui.NewTester(a.View, 1200, 800)
	if !tt.HasText("ZPad 9.9.9") {
		t.Fatalf("the status bar does not show the version set; texts %q", tt.Texts())
	}

	// A plain go build links none: the default stays, also over an empty one.
	SetVersion("")
	if Version != "9.9.9" {
		t.Fatalf("an empty version left %q, want 9.9.9", Version)
	}
}

func TestEditorView(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	src := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := loadDocument(path, 20)
	if err != nil {
		t.Fatal(err)
	}
	a := New(settings.Default(), dir)
	a.addDocument(doc)

	tt := ui.NewTester(a.View, 1200, 800)
	for _, want := range []string{"main.go", "Go", "UTF-8", locales.LineColumn(1, 1)} {
		if !tt.HasText(want) {
			t.Errorf("view does not show %q; texts %q", want, tt.Texts())
		}
	}
}

func TestWhitespaceMode(t *testing.T) {
	for _, tc := range []struct {
		setting string
		want    editor.Whitespace
	}{
		{"selection", editor.WhitespaceSelection},
		{"none", editor.WhitespaceNone},
		{"all", editor.WhitespaceAll},
		{"trailing", editor.WhitespaceTrailing},
		{"", editor.WhitespaceSelection},
	} {
		if got := whitespaceMode(tc.setting); got != tc.want {
			t.Errorf("whitespaceMode(%q) = %v, want %v", tc.setting, got, tc.want)
		}
	}
}

// The editor takes the whitespace setting on every frame, so changing it
// needs no reopened tab.
func TestViewAppliesWhitespaceSetting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(path, []byte("a b\tc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := loadDocument(path, 20)
	if err != nil {
		t.Fatal(err)
	}
	a := New(settings.Default(), dir)
	a.addDocument(doc)
	a.Settings.Whitespace = "all"

	ui.NewTester(a.View, 1200, 800)
	tab := a.pane(0).activeTab()
	if tab == nil {
		t.Fatal("no active tab after opening a document")
	}
	if want := editor.WhitespaceAll; tab.state.Whitespace != want {
		t.Fatalf("editor whitespace %v, want %v", tab.state.Whitespace, want)
	}
}

func TestTabs(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.NewFile()
	a.NewFile()
	if len(a.left.tabs) != 2 || a.left.active != 1 {
		t.Fatalf("tabs %d, active %d", len(a.left.tabs), a.left.active)
	}
	a.removeTab(0, 1)
	if len(a.left.tabs) != 1 || a.left.active != 0 {
		t.Fatalf("after remove: tabs %d, active %d", len(a.left.tabs), a.left.active)
	}
	if a.left.tabs[0].doc.Name() != locales.UntitledN(1) {
		t.Fatalf("tab name %q", a.left.tabs[0].doc.Name())
	}
}

// New untitled documents are numbered, and the number stays as the tabs
// around it close: a prompt that names one must keep pointing at it.
func TestUntitledNamesKeepTheirNumbers(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.NewFile()
	a.NewFile()
	a.NewFile()
	a.removeTab(0, 1) // 未命名 2
	a.NewFile()

	var names []string
	for _, tab := range a.left.tabs {
		names = append(names, tab.doc.Name())
	}
	want := []string{locales.UntitledN(1), locales.UntitledN(3), locales.UntitledN(4)}
	if !slices.Equal(names, want) {
		t.Fatalf("tab names %q, want %q", names, want)
	}
}

func TestSaveDialogNamesAnUntitledDocument(t *testing.T) {
	workspace := t.TempDir()
	a := New(settings.Default(), t.TempDir())
	a.setWorkspace(workspace)
	doc := a.newUntitledDocument()
	if got, want := a.saveDialogPath(doc), filepath.Join(workspace, "untitled-1.txt"); got != want {
		t.Fatalf("untitled save path %q, want %q", got, want)
	}
	second := a.newUntitledDocument()
	if got, want := a.saveDialogPath(second), filepath.Join(workspace, "untitled-2.txt"); got != want {
		t.Fatalf("second save path %q, want %q", got, want)
	}
	doc.Path = filepath.Join(workspace, "note.md")
	if got := a.saveDialogPath(doc); got != doc.Path {
		t.Fatalf("saved path %q, want %q", got, doc.Path)
	}

	if home, err := os.UserHomeDir(); err == nil {
		bare := New(settings.Default(), t.TempDir())
		want := filepath.Join(home, "untitled-1.txt")
		if got := bare.saveDialogPath(bare.newUntitledDocument()); got != want {
			t.Fatalf("home save path %q, want %q", got, want)
		}
	}
}

func TestWorkspaceOpensSidebar(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := New(settings.Default(), t.TempDir())
	a.setWorkspace(workspace)
	if !a.Settings.SidebarOpen {
		t.Fatal("sidebar did not open with the workspace")
	}
	if a.tree == nil {
		t.Fatal("no tree")
	}
	if roots := a.tree.Roots(); len(roots) != 1 || filepath.Base(roots[0]) != "a.txt" {
		t.Fatalf("roots %v", roots)
	}
}

func TestThemeSwitchPersists(t *testing.T) {
	dir := t.TempDir()
	a := New(settings.Default(), dir)

	a.setThemeID(theme.DarkID)
	if a.Settings.Theme != theme.DarkID {
		t.Fatalf("settings theme %q, want dark", a.Settings.Theme)
	}
	if !a.Dark {
		t.Fatal("app is not dark after choosing the dark theme")
	}

	if _, err := os.Stat(filepath.Join(dir, "settings.json")); err != nil {
		t.Fatalf("settings were not saved: %v", err)
	}

	a.setThemeID("dracula")
	if a.Settings.Theme != "dracula" || !a.Dark {
		t.Fatalf("named theme %q dark %v, want dracula and dark", a.Settings.Theme, a.Dark)
	}
	loaded, err := settings.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Theme != "dracula" {
		t.Fatalf("saved theme %q, want dracula", loaded.Theme)
	}
}

func TestUnknownThemeFallsBackToSystem(t *testing.T) {
	s := settings.Default()
	s.Theme = "neon"
	a := New(s, t.TempDir())
	if a.Settings.Theme != theme.SystemID {
		t.Fatalf("theme %q, want %q", a.Settings.Theme, theme.SystemID)
	}
}
