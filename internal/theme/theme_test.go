package theme

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

func TestDefaultThemesKeepTheirColors(t *testing.T) {
	light, dark := Default(false), Default(true)
	if light.UI.Background != ui.Hex("#FCFCFD") || light.UI.Accent != ui.Hex("#3D5BC7") {
		t.Error("default light palette changed")
	}
	if light.Chrome.Sidebar != ui.Hex("#EEF1F6") ||
		light.Chrome.TabBar != ui.Hex("#F4F6F9") ||
		light.Chrome.ActiveTab != ui.Hex("#FCFCFD") {
		t.Error("default light chrome changed")
	}
	if dark.UI.Background != ui.Hex("#161A22") || dark.UI.Accent != ui.Hex("#6E82EB") {
		t.Error("default dark palette changed")
	}
	if dark.Chrome.Sidebar != ui.Hex("#12161D") ||
		dark.Chrome.TabBar != ui.Hex("#141820") ||
		dark.Chrome.ActiveTab != ui.Hex("#161A22") {
		t.Error("default dark chrome changed")
	}
	if got := light.Code.Tokens[editor.TokenKeyword].Color; got != ui.Hex("#6A44B5") {
		t.Errorf("light keyword %v, want #6A44B5", got)
	}
	if got := dark.Code.Tokens[editor.TokenKeyword].Color; got != ui.Hex("#B499ED") {
		t.Errorf("dark keyword %v, want #B499ED", got)
	}
	for name, d := range map[string]*Definition{"light": light, "dark": dark} {
		for k := editor.TokenText; k <= editor.TokenInvalid; k++ {
			if d.Code.Tokens[k].Color == (ui.Color{}) {
				t.Errorf("default %s leaves token %d uncolored", name, k)
			}
		}
		if d.Code.Whitespace.A >= 200 || d.Code.IndentGuide.A >= 200 ||
			d.Code.ActiveIndentGuide.A >= 200 || d.Code.MatchingBracket.A >= 200 {
			t.Errorf("default %s editor guides/whitespace should stay translucent", name)
		}
		if d.Code.InactiveSelection == d.Code.Selection {
			t.Errorf("default %s inactive selection does not dim", name)
		}
	}
}

func TestNamedThemesAreComplete(t *testing.T) {
	named := Named()
	if len(named) != 18 {
		t.Fatalf("%d named themes, want 18", len(named))
	}
	seen := map[string]bool{}
	darkCount := 0
	for _, d := range named {
		if d.ID == "" || d.ID == SystemID || d.ID == LightID || d.ID == DarkID {
			t.Errorf("named theme id %q collides with a reserved id", d.ID)
		}
		if seen[d.ID] {
			t.Errorf("duplicate theme id %q", d.ID)
		}
		seen[d.ID] = true
		if d.Name == "" || d.UI == nil || d.Code == nil {
			t.Errorf("theme %q is incomplete", d.ID)
		}
		if d.Dark {
			darkCount++
		}
		if Normalize(d.ID) != d.ID {
			t.Errorf("theme %q does not normalize to itself", d.ID)
		}
		if got, ok := Lookup(d.ID); !ok || got != d {
			t.Errorf("lookup %q = %v, %v; want the theme", d.ID, got, ok)
		}
		if Resolve(d.ID, !d.Dark) != d {
			t.Errorf("theme %q changes with the system appearance", d.ID)
		}
		if Sidebar(d.UI) != d.Chrome.Sidebar || TabBar(d.UI) != d.Chrome.TabBar ||
			ActiveTab(d.UI) != d.Chrome.ActiveTab {
			t.Errorf("theme %q chrome does not resolve from its palette", d.ID)
		}
		for k := editor.TokenText; k <= editor.TokenInvalid; k++ {
			if d.Code.Tokens[k].Color == (ui.Color{}) {
				t.Errorf("theme %q leaves token %d uncolored", d.ID, k)
			}
		}
	}
	if darkCount == 0 || darkCount == len(named) {
		t.Errorf("%d of %d named themes are dark", darkCount, len(named))
	}
	if Resolve(SystemID, false) != Default(false) || Resolve(SystemID, true) != Default(true) {
		t.Error("system does not follow the appearance")
	}
	if Normalize("neon") != SystemID {
		t.Error("an unknown id does not normalize to system")
	}
	if Resolve("neon", true) != Default(true) {
		t.Error("an unknown id does not resolve to the default theme")
	}
}

func TestThemeContrast(t *testing.T) {
	themes := append([]*Definition{Default(false), Default(true)}, Named()...)
	for _, d := range themes {
		if c := contrast(d.UI.Text, d.UI.Background); c < 4.5 {
			t.Errorf("theme %q text contrast %.2f, want at least 4.5", d.ID, c)
		}
		if c := contrast(d.UI.TextMuted, d.UI.Background); c < 3 {
			t.Errorf("theme %q muted text contrast %.2f, want at least 3", d.ID, c)
		}
		if c := contrast(d.UI.AccentText, d.UI.Accent); c < 4.5 {
			t.Errorf("theme %q accent text contrast %.2f, want at least 4.5", d.ID, c)
		}
	}
}

func TestFor(t *testing.T) {
	light, dark := For(false), For(true)
	if light.Dark {
		t.Fatal("light theme is dark")
	}
	if !dark.Dark {
		t.Fatal("dark theme is not dark")
	}
	if light.Background == dark.Background {
		t.Fatal("light and dark share a background")
	}
	if light.Radius != 8 || dark.Radius != 8 {
		t.Fatalf("radii %v, %v; want 8", light.Radius, dark.Radius)
	}
	ct := EditorTheme(light)
	if ct.Background != light.Background || ct.Text != light.Text {
		t.Fatal("editor theme does not follow the UI theme")
	}
	if got := ct.Tokens[editor.TokenKeyword].Color; got != ui.Hex("#6A44B5") {
		t.Fatalf("keyword color %v, want #6A44B5", got)
	}
	if got := ct.Tokens[editor.TokenString].Color; got != ui.Hex("#197250") {
		t.Fatalf("string color %v, want #197250", got)
	}
}

func TestScrollbarsAreThin(t *testing.T) {
	for name, th := range map[string]*ui.Theme{"light": Light(), "dark": Dark()} {
		s := th.ScrollbarStyle
		if s.ThumbWidth > 6 || th.ScrollbarWidth > 6 {
			t.Errorf("%s scrollbar thumb %v (width %v), want at most 6", name, s.ThumbWidth, th.ScrollbarWidth)
		}
		if s.TrackWidth <= s.ThumbWidth {
			t.Errorf("%s scrollbar track %v, thumb %v: no room to grab", name, s.TrackWidth, s.ThumbWidth)
		}
		if s.Radius <= 0 {
			t.Errorf("%s scrollbar radius %v, want rounded", name, s.Radius)
		}
		if s.Thumb.A >= 200 {
			t.Errorf("%s scrollbar thumb alpha %d, want translucent", name, s.Thumb.A)
		}
	}
}

func TestInverseColors(t *testing.T) {
	for name, th := range map[string]*ui.Theme{"light": Light(), "dark": Dark()} {
		fill, text := Inverse(th)
		if fill != th.Inverse || text != th.InverseText {
			t.Errorf("%s tip is %v on %v, want the theme's inverse colors", name, text, fill)
		}
		if c := contrast(text, fill); c < 4.5 {
			t.Errorf("%s tip contrast %.2f, want at least 4.5", name, c)
		}
	}
	th := *Light()
	th.Inverse = ui.Transparent
	th.InverseText = ui.Transparent
	if fill, text := Inverse(&th); fill != th.Text || text != th.Background {
		t.Errorf("fallback tip %v on %v, want the text on the background", text, fill)
	}
}

func TestFileTreeTheme(t *testing.T) {
	base := Light()
	tree := FileTree(base)
	if tree.Radius != 0 {
		t.Errorf("tree radius %v, want square full-width rows", tree.Radius)
	}
	if tree.Accent == base.Accent {
		t.Error("tree still fills chosen rows with the accent")
	}
	if tree.Accent.A >= 100 {
		t.Errorf("tree wash alpha %d, want faint", tree.Accent.A)
	}
	if tree.AccentText != base.Text {
		t.Error("tree chosen rows do not keep the text color")
	}
	if tree.InactiveAccent != tree.Accent {
		t.Error("tree chosen rows change in an inactive window")
	}
	if base.Radius == 0 || base.AccentText == base.Text {
		t.Error("FileTree changed the theme it copied")
	}
}
