package app

import (
	"testing"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
	"github.com/helloxz/zpad/internal/settings"
	"github.com/helloxz/zpad/internal/theme"
	"github.com/helloxz/zpad/locales"
)

func TestSettingsModalOpens(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.showSettings = true
	tt := ui.NewTester(a.View, 900, 760)
	for _, want := range []string{
		locales.SettingsTitle(),
		locales.Appearance(),
		locales.EditorSection(),
		locales.FilesSection(),
		locales.WorkspaceSection(),
		locales.SoftWrap(),
		locales.SoftWrapDescription(),
		locales.ShowWhitespace(),
		locales.ShowWhitespaceDescription(),
		locales.VimMode(),
		locales.VimModeDescription(),
		locales.RestoreDefaults(),
		locales.Done(),
	} {
		if !tt.HasText(want) {
			t.Errorf("settings panel does not show %q; texts %q", want, tt.Texts())
		}
	}
}

func TestThemeSelectAppliesNamedTheme(t *testing.T) {
	i18n.SetLocaleFunc(func() string { return "en" })
	defer i18n.SetLocaleFunc(func() string { return "en" })

	dir := t.TempDir()
	a := New(settings.Default(), dir)
	a.setThemeID("nord")
	a.showSettings = true
	tt := ui.NewTester(a.View, 900, 760)
	tt.Frame()

	nord, ok := theme.Lookup("nord")
	if !ok {
		t.Fatal("nord theme is missing")
	}
	if !tt.HasText(nord.Name) {
		t.Fatalf("settings select does not show the chosen theme; texts %q", tt.Texts())
	}
	if err := tt.Click(nord.Name); err != nil {
		t.Fatal(err)
	}
	tt.Frame()

	dracula, ok := theme.Lookup("dracula")
	if !ok {
		t.Fatal("dracula theme is missing")
	}
	if err := tt.Click(dracula.Name); err != nil {
		t.Fatal(err)
	}
	if a.Settings.Theme != "dracula" {
		t.Fatalf("theme %q after choosing Dracula, want dracula", a.Settings.Theme)
	}
	if !a.Dark {
		t.Fatal("Dracula did not switch the app to a dark palette")
	}
	loaded, err := settings.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Theme != "dracula" {
		t.Fatalf("saved theme %q, want dracula", loaded.Theme)
	}
}

func TestWhitespaceSelectApplies(t *testing.T) {
	i18n.SetLocaleFunc(func() string { return "en" })
	defer i18n.SetLocaleFunc(func() string { return "en" })

	dir := t.TempDir()
	a := New(settings.Default(), dir)
	a.showSettings = true
	tt := ui.NewTester(a.View, 900, 760)
	tt.Frame()

	// The editor section sits below the fold of the 640 DIP panel.
	tt.Move(450, 500)
	tt.Scroll(450, 500, 0, 600)
	tt.Frame()
	if err := tt.Click(locales.WhitespaceSelection()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click(locales.WhitespaceAll()); err != nil {
		t.Fatal(err)
	}
	if a.Settings.Whitespace != "all" {
		t.Fatalf("whitespace %q after choosing All, want all", a.Settings.Whitespace)
	}
	loaded, err := settings.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Whitespace != "all" {
		t.Fatalf("saved whitespace %q, want all", loaded.Whitespace)
	}
}

func TestSelectOptionsShowHandCursor(t *testing.T) {
	i18n.SetLocaleFunc(func() string { return "en" })
	defer i18n.SetLocaleFunc(func() string { return "en" })

	a := New(settings.Default(), t.TempDir())
	a.setThemeID("nord")
	a.showSettings = true
	tt := ui.NewTester(a.View, 900, 760)
	tt.Frame()

	nord, ok := theme.Lookup("nord")
	if !ok {
		t.Fatal("nord theme is missing")
	}
	if err := tt.Click(nord.Name); err != nil {
		t.Fatal(err)
	}
	tt.Frame()

	dracula, ok := theme.Lookup("dracula")
	if !ok {
		t.Fatal("dracula theme is missing")
	}
	r, ok := tt.Find(dracula.Name)
	if !ok {
		t.Fatalf("theme options do not show %q", dracula.Name)
	}
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	if got := tt.Cursor(); got != ui.CursorPointer {
		t.Fatalf("cursor over a select option %v, want the pointing hand", got)
	}
	tt.Key(0, ui.KeyEscape)
}

func TestThemeSelectGroupsByAppearance(t *testing.T) {
	i18n.SetLocaleFunc(func() string { return "en" })
	defer i18n.SetLocaleFunc(func() string { return "en" })

	a := New(settings.Default(), t.TempDir())
	a.setThemeID("monokai")
	a.showSettings = true
	tt := ui.NewTester(a.View, 900, 760)
	tt.Frame()

	if err := tt.Click("Monokai"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	for _, want := range []string{locales.ThemeGroupDefault(), locales.ThemeGroupDark(), locales.ThemeGroupLight(), "ZPad Dark", "ZPad Light"} {
		if !tt.HasText(want) {
			t.Errorf("theme drop-down does not show %q", want)
		}
	}
}

func TestThemeSelectKeyboardReachesLastTheme(t *testing.T) {
	i18n.SetLocaleFunc(func() string { return "en" })
	defer i18n.SetLocaleFunc(func() string { return "en" })

	a := New(settings.Default(), t.TempDir())
	a.setThemeID("monokai")
	a.showSettings = true
	tt := ui.NewTester(a.View, 900, 760)
	tt.Frame()

	if err := tt.Click("Monokai"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	// The last theme is below the capped popup; End must scroll to it and
	// Enter must choose it.
	tt.Key(0, ui.KeyEnd)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	if a.Settings.Theme != "ink-wash" {
		t.Fatalf("theme %q after End and Enter, want ink-wash", a.Settings.Theme)
	}
}

func TestThemeTagsAlignRight(t *testing.T) {
	i18n.SetLocaleFunc(func() string { return "en" })
	defer i18n.SetLocaleFunc(func() string { return "en" })

	a := New(settings.Default(), t.TempDir())
	a.setThemeID("catppuccin-mocha")
	a.showSettings = true
	tt := ui.NewTester(a.View, 900, 760)
	tt.Frame()

	name, ok := tt.Find("Catppuccin Mocha")
	if !ok {
		t.Fatal("the chosen theme name is not shown")
	}
	tag, ok := tt.Find(locales.ThemeDarkTag())
	if !ok {
		t.Fatal("the chosen theme tag is not shown")
	}
	if tag.X <= name.X+name.W {
		t.Fatalf("tag at %.0f, name ends at %.0f: not right-aligned", tag.X, name.X+name.W)
	}
}

func TestEveryNamedThemeRenders(t *testing.T) {
	for _, def := range theme.Named() {
		s := settings.Default()
		s.Theme = def.ID
		a := New(s, t.TempDir())
		a.showSettings = true
		tt := ui.NewTester(a.View, 900, 760)
		tt.Frame()
		if !tt.HasText(def.Name) {
			t.Errorf("theme %q is not listed; texts %q", def.Name, tt.Texts())
		}
		if a.Dark != def.Dark {
			t.Errorf("theme %q dark = %v, want %v", def.ID, a.Dark, def.Dark)
		}
	}
}

func TestLanguageSwitchInView(t *testing.T) {
	i18n.SetLocaleFunc(func() string { return "zh-Hans" })
	defer i18n.SetLocaleFunc(func() string { return "en" })

	a := New(settings.Default(), t.TempDir())
	tt := ui.NewTester(a.View, 900, 700)
	if !tt.HasText(locales.In("zh-Hans").AppTagline()) {
		t.Fatalf("view did not switch to Chinese; texts %q", tt.Texts())
	}
}

func TestFontNamesAreMonospaced(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	allowed := map[string]bool{}
	for _, f := range ui.FontFamilies() {
		if f.Monospaced {
			allowed[f.Name] = true
		}
	}
	for _, name := range a.fontNames() {
		if !allowed[name] {
			t.Fatalf("font %q is not monospaced", name)
		}
	}
}

func TestCatalogsHaveBothLanguages(t *testing.T) {
	en := locales.In("en").AppTagline()
	zh := locales.In("zh-Hans").AppTagline()
	if en == "" || zh == "" {
		t.Fatal("empty catalog message")
	}
	if en == zh {
		t.Fatal("English and Chinese app taglines are the same")
	}
	line := locales.In("zh-Hans").LineColumn(3, 7)
	if line != "行 3, 列 7" {
		t.Fatalf("Chinese line/column = %q", line)
	}
}
