package app

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helloxz/zpad/internal/settings"
)

func TestShortcutTextFollowsThePlatform(t *testing.T) {
	for _, c := range []struct {
		goos string
		spec string
		want string
	}{
		{"darwin", "CmdOrCtrl+O", "⌘O"},
		{"darwin", "CmdOrCtrl+Shift+O", "⌘⇧O"},
		{"darwin", "CmdOrCtrl+Alt+N", "⌘⌥N"},
		{"windows", "CmdOrCtrl+O", "Ctrl+O"},
		{"windows", "CmdOrCtrl+Shift+O", "Ctrl+Shift+O"},
		{"windows", "CmdOrCtrl+Alt+N", "Ctrl+Alt+N"},
		{"linux", "CmdOrCtrl+Shift+V", "Ctrl+Shift+V"},
		{"windows", "F5", "F5"},
	} {
		if got := shortcutTextFor(c.goos, c.spec); got != c.want {
			t.Errorf("shortcutTextFor(%q, %q) = %q, want %q", c.goos, c.spec, got, c.want)
		}
	}
}

func TestShortcutTooltipsFollowThePlatform(t *testing.T) {
	if got, want := markdownPreviewShortcut(), shortcutText("CmdOrCtrl+Shift+V"); got != want {
		t.Errorf("preview tooltip shortcut = %q, want %q", got, want)
	}
	if got, want := openInBrowserShortcut(), shortcutText("CmdOrCtrl+Shift+B"); got != want {
		t.Errorf("browser tooltip shortcut = %q, want %q", got, want)
	}
	if got, want := softWrapShortcut(), shortcutText("Alt+Z"); got != want {
		t.Errorf("word wrap tooltip shortcut = %q, want %q", got, want)
	}
	if runtime.GOOS == "darwin" {
		return
	}
	if markdownPreviewShortcut() == "⌘⇧V" || openInBrowserShortcut() == "⌘⇧B" {
		t.Errorf("tooltips write macOS symbols on %s", runtime.GOOS)
	}
}

func TestEmptyStateShowsThePlatformShortcuts(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	tt := ui.NewTester(a.View, 1200, 800)
	tt.Frame()

	want := shortcutText("CmdOrCtrl+O")
	if !tt.HasText(want) {
		t.Fatalf("empty state does not show %q; texts %q", want, tt.Texts())
	}
	if runtime.GOOS == "darwin" {
		return
	}
	if tt.HasText("⌘O") || tt.HasText("⌘⇧O") || tt.HasText("⌘N") {
		t.Errorf("empty state writes macOS symbols on %s; texts %q", runtime.GOOS, tt.Texts())
	}
}
