package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helloxz/zpad/internal/settings"
	"github.com/helloxz/zpad/internal/vim"
	"github.com/helloxz/zpad/locales"
)

// vimApp opens one document with Vim mode on and gives it the keyboard.
func vimApp(t *testing.T, text string) (*App, *ui.Tester, *Tab) {
	t.Helper()
	s := settings.Default()
	s.VimMode = true
	a := New(s, t.TempDir())
	a.addDocument(docAt("/tmp/vim.txt", text))
	tt := ui.NewTester(a.View, 900, 700)
	for range 3 {
		tt.Frame()
	}
	tab := a.pane(0).activeTab()
	if tab == nil {
		t.Fatal("no active tab")
	}
	if !tab.state.Focused() {
		t.Fatal("the editor does not have the keyboard")
	}
	return a, tt, tab
}

func TestVimModeIsOffByDefault(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "abc"))
	tt := ui.NewTester(a.View, 900, 700)
	tt.Frame()
	if a.Settings.VimMode {
		t.Fatal("Vim mode is on by default")
	}
	tab := a.pane(0).activeTab()
	if tab.state.OnInput != nil {
		t.Fatal("the editor has a modal input while Vim mode is off")
	}
	if tab.state.NoInputMethod {
		t.Fatal("input methods are off while Vim mode is off")
	}
}

func TestVimModeInstallsNormalMode(t *testing.T) {
	_, tt, tab := vimApp(t, "abc")
	if tab.state.OnInput == nil {
		t.Fatal("Vim mode did not install the editor's input hook")
	}
	if !tab.state.NoInputMethod {
		t.Fatal("normal mode did not turn input methods off")
	}
	if tab.state.CursorShape != editor.CursorBlock {
		t.Fatalf("cursor shape %v, want block", tab.state.CursorShape)
	}
	if !tt.HasText(locales.VimNormal()) {
		t.Fatalf("status bar does not show NORMAL; texts %q", tt.Texts())
	}
}

func TestVimNormalKeysEdit(t *testing.T) {
	_, tt, tab := vimApp(t, "one\ntwo\nthree")
	tt.Type("dd")
	tt.Frame()
	if got := tab.doc.Buffer.String(); got != "two\nthree" {
		t.Fatalf("text %q after dd", got)
	}
	tt.Type("u")
	tt.Frame()
	if got := tab.doc.Buffer.String(); got != "one\ntwo\nthree" {
		t.Fatalf("text %q after undo", got)
	}
}

func TestVimInsertAndEscape(t *testing.T) {
	_, tt, tab := vimApp(t, "ab")
	tt.Type("ihello")
	tt.Frame()
	if tab.vim.Mode() != vim.Insert {
		t.Fatal("i did not enter insert mode")
	}
	if !tt.HasText(locales.VimInsert()) {
		t.Fatalf("status bar does not show INSERT; texts %q", tt.Texts())
	}
	if got := tab.doc.Buffer.String(); got != "helloab" {
		t.Fatalf("text %q after ihello", got)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if tab.vim.Mode() != vim.Normal {
		t.Fatal("Escape did not return to normal mode")
	}
	if !tt.HasText(locales.VimNormal()) {
		t.Fatalf("status bar does not show NORMAL; texts %q", tt.Texts())
	}
	tt.Type("x")
	tt.Frame()
	if got := tab.doc.Buffer.String(); got != "hellab" {
		t.Fatalf("text %q after x", got)
	}
}

func TestVimVisualDelete(t *testing.T) {
	_, tt, tab := vimApp(t, "abcdef")
	tt.Type("vlld")
	tt.Frame()
	if got := tab.doc.Buffer.String(); got != "def" {
		t.Fatalf("text %q after visual delete", got)
	}
}

func TestVimSearchCommandLine(t *testing.T) {
	a, tt, tab := vimApp(t, "alpha\nneedle\nbeta\nneedle")
	tt.Type("/")
	tt.Frame()
	if !a.vimCmdOpen {
		t.Fatal("/ did not open the command line")
	}
	tt.Type("needle")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if tab.state.Cursor() != 6 {
		t.Fatalf("cursor %d after /needle, want 6", tab.state.Cursor())
	}
	tt.Type("n")
	tt.Frame()
	if tab.state.Cursor() != 18 {
		t.Fatalf("cursor %d after n, want 18", tab.state.Cursor())
	}
}

func TestVimExLine(t *testing.T) {
	_, tt, tab := vimApp(t, "one\ntwo\nthree")
	tt.Type(":")
	tt.Frame()
	tt.Type("2")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if tab.state.Cursor() != 4 {
		t.Fatalf("cursor %d after :2, want 4", tab.state.Cursor())
	}
}

func TestVimOffPassesText(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/plain.txt", ""))
	tt := ui.NewTester(a.View, 900, 700)
	for range 2 {
		tt.Frame()
	}
	tt.Type("dd")
	tt.Frame()
	tab := a.pane(0).activeTab()
	if got := tab.doc.Buffer.String(); got != "dd" {
		t.Fatalf("text %q with Vim mode off, want dd", got)
	}
}

func TestVimModeToggle(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "abc"))
	tt := ui.NewTester(a.View, 900, 700)
	for range 2 {
		tt.Frame()
	}
	tab := a.pane(0).activeTab()
	a.Settings.VimMode = true
	a.vimModeChanged()
	tt.Frame()
	if tab.vim == nil || tab.state.OnInput == nil {
		t.Fatal("turning Vim mode on did not install it")
	}
	a.Settings.VimMode = false
	a.vimModeChanged()
	tt.Frame()
	if tab.vim != nil || tab.state.OnInput != nil {
		t.Fatal("turning Vim mode off did not remove it")
	}
	if tab.state.CursorShape != editor.CursorBar {
		t.Fatalf("cursor shape %v after turning Vim off, want bar", tab.state.CursorShape)
	}
}

func TestVimReadOnlyDoesNotEdit(t *testing.T) {
	_, tt, tab := vimApp(t, "abc")
	tab.doc.ReadOnly = true
	tab.state.ReadOnly = true
	tt.Frame()
	tt.Type("x")
	tt.Frame()
	if got := tab.doc.Buffer.String(); got != "abc" {
		t.Fatalf("read-only text changed to %q", got)
	}
}

func TestVimEscapeCancelsOperator(t *testing.T) {
	_, tt, tab := vimApp(t, "abc")
	tt.Type("d")
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	tt.Type("x")
	tt.Frame()
	if got := tab.doc.Buffer.String(); got != "bc" {
		t.Fatalf("text %q after d, Escape, x", got)
	}
}

func TestVimCommandLineEscape(t *testing.T) {
	a, tt, tab := vimApp(t, "abc")
	tt.Type("/")
	tt.Frame()
	if !a.vimCmdOpen {
		t.Fatal("/ did not open the command line")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.vimCmdOpen {
		t.Fatal("Escape did not close the command line")
	}
	if tab.vim.Mode() != vim.Normal {
		t.Fatal("Escape from the command line left normal mode")
	}
}

func TestVimInputMethodFollowsMode(t *testing.T) {
	_, tt, tab := vimApp(t, "")
	// Normal mode turns input methods off, so a Chinese input method types
	// commands rather than composing.
	tt.Compose("に", 1)
	tt.Frame()
	if got := tab.doc.Buffer.String(); got != "" {
		t.Fatalf("normal mode composed %q", got)
	}
	tt.Type("i")
	tt.Frame()
	// Insert mode turns them back on.
	tt.Compose("に", 1)
	tt.Frame()
	if got := tab.doc.Buffer.String(); got != "に" {
		t.Fatalf("insert mode composed %q, want に", got)
	}
}

func TestVimControlBracketLeavesInsert(t *testing.T) {
	_, tt, tab := vimApp(t, "")
	tt.Type("i")
	tt.Frame()
	if tab.vim.Mode() != vim.Insert {
		t.Fatal("i did not enter insert mode")
	}
	tt.Key(ui.Ctrl, ui.KeyBracketLeft)
	tt.Frame()
	if tab.vim.Mode() != vim.Normal {
		t.Fatal("Ctrl+[ did not return to normal mode")
	}
}
