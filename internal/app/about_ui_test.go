package app

import (
	"os"
	"testing"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
	"github.com/helloxz/zpad/internal/settings"
	"github.com/helloxz/zpad/locales"
)

func TestAboutDialogShowsLicenseAndRepository(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.aboutOpen = true
	tt := ui.NewTester(a.View, 900, 700)
	for _, want := range []string{
		"ZPad",
		locales.AboutTagline(),
		locales.AboutVersion(Version),
		locales.AboutLicense(),
		"AGPL-3.0",
		"github.com/helloxz/zpad",
		locales.AboutContactAuthor(),
		"blog.xiaoz.org",
		"@xiaozblog",
		"xphub.dev",
		locales.AboutWebsite(),
		locales.AboutPrivacy(),
		locales.AboutFeedback(),
	} {
		if !tt.HasText(want) {
			t.Errorf("about dialog does not show %q; texts %q", want, tt.Texts())
		}
	}
}

// The toolbar's info button is the second way into the About dialog.
func TestToolbarAboutButtonOpensDialog(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	tt := ui.NewTester(a.View, 1200, 800)
	if err := tt.Click(locales.AboutApp()); err != nil {
		t.Fatal(err)
	}
	if !a.aboutOpen {
		t.Fatal("the toolbar's About button did not open the dialog")
	}
	tt.Frame()
	if !tt.HasText(locales.AboutTagline()) {
		t.Fatalf("the About dialog did not show; texts %q", tt.Texts())
	}
}

// The license states the terms, so it is plain text: only the project, the
// author's pages and the footer links are links.
func TestAboutLinksOpenInBrowser(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.aboutOpen = true
	tt := ui.NewTester(a.View, 900, 700)
	if err := tt.Click("AGPL-3.0"); err != nil {
		t.Fatal(err)
	}
	if got := tt.OpenedURLs(); len(got) != 0 {
		t.Fatalf("the license opened %q, want no link", got)
	}

	for i, link := range []struct{ text, url string }{
		{"github.com/helloxz/zpad", aboutRepoURL},
		{"blog.xiaoz.org", aboutBlogURL},
		{"@xiaozblog", aboutXURL},
		{"xphub.dev", aboutProductsURL},
		{locales.AboutWebsite(), aboutSiteURL},
		{locales.AboutPrivacy(), aboutPrivacyURL},
		{locales.AboutFeedback(), aboutFeedbackURL},
	} {
		if err := tt.Click(link.text); err != nil {
			t.Fatalf("click %q: %v", link.text, err)
		}
		got := tt.OpenedURLs()
		if len(got) != i+1 || got[i] != link.url {
			t.Fatalf("the %q link opened %q, want %q", link.text, got, link.url)
		}
	}
}

// The products, site and privacy links follow the interface language.
func TestAboutLinksFollowLanguage(t *testing.T) {
	cases := []struct {
		locale   string
		products string
		site     string
		privacy  string
	}{
		{"zh-Hans", aboutProductsZhURL, aboutSiteZhURL, aboutPrivacyZhURL},
		{"en", aboutProductsURL, aboutSiteURL, aboutPrivacyURL},
	}
	for _, c := range cases {
		locale := c.locale
		i18n.SetLocaleFunc(func() string { return locale })
		if got := aboutProductsLink(); got != c.products {
			t.Errorf("aboutProductsLink() in %s = %q, want %q", locale, got, c.products)
		}
		if got := aboutSiteLink(); got != c.site {
			t.Errorf("aboutSiteLink() in %s = %q, want %q", locale, got, c.site)
		}
		if got := aboutPrivacyLink(); got != c.privacy {
			t.Errorf("aboutPrivacyLink() in %s = %q, want %q", locale, got, c.privacy)
		}
	}
	i18n.SetLocaleFunc(func() string { return "en" })
}

// The author section and the footer are translated like the rest of the
// dialog.
func TestAboutDialogFollowsLanguage(t *testing.T) {
	i18n.SetLocaleFunc(func() string { return "zh-Hans" })
	defer i18n.SetLocaleFunc(func() string { return "en" })

	a := New(settings.Default(), t.TempDir())
	a.aboutOpen = true
	tt := ui.NewTester(a.View, 900, 700)
	for _, want := range []string{locales.AboutContactAuthor(), "联系作者", "博客", "更多产品", "官网", "隐私政策", "问题反馈"} {
		if !tt.HasText(want) {
			t.Errorf("the Chinese about dialog does not show %q; texts %q", want, tt.Texts())
		}
	}
}

// The Close button is gone: the backdrop and Escape are the ways out.
func TestAboutDialogCloses(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.aboutOpen = true
	tt := ui.NewTester(a.View, 900, 700)
	if tt.HasText(locales.Close()) {
		t.Fatalf("the about dialog still shows a close button; texts %q", tt.Texts())
	}

	// A corner of the window is the backdrop, outside the centered panel.
	tt.ClickAt(10, 10)
	tt.Frame()
	if a.aboutOpen {
		t.Fatal("clicking the backdrop did not close the about dialog")
	}

	a.aboutOpen = true
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.aboutOpen {
		t.Fatal("Escape did not close the about dialog")
	}
}

// The About dialog shows the icon the app embeds, and keeps the letter
// mark when the binary carries none, as in these tests.
func TestAboutIconLoads(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.readAboutIcon("../../resources/icon.png")
	if a.aboutIcon == nil {
		t.Fatal("resources/icon.png did not decode")
	}
	if w, h := ui.ImageSize(a.aboutIcon); w != aboutIconMaxSide || h != aboutIconMaxSide {
		t.Fatalf("the app icon is %vx%v, want %dx%d", w, h, aboutIconMaxSide, aboutIconMaxSide)
	}

	a.aboutOpen = true
	tt := ui.NewTester(a.View, 900, 700)
	tt.Frame()
	if !tt.HasText(locales.AboutVersion(Version)) {
		t.Fatalf("the dialog with an icon does not show the version; texts %q", tt.Texts())
	}

	// A missing file leaves the fallback in place rather than failing.
	missing := New(settings.Default(), t.TempDir())
	missing.readAboutIcon("../../resources/missing.png")
	if missing.aboutIcon != nil {
		t.Fatal("a missing icon was reported as loaded")
	}
}

// main embeds resources/icon.png: the dialog decodes it through SetAppIcon.
func TestAboutIconFromTheApp(t *testing.T) {
	data, err := os.ReadFile("../../resources/icon.png")
	if err != nil {
		t.Fatal(err)
	}
	a := New(settings.Default(), t.TempDir())
	a.SetAppIcon(data)
	a.loadAboutIcon()
	if a.aboutIcon == nil {
		t.Fatal("the embedded icon did not decode")
	}
	if w, h := ui.ImageSize(a.aboutIcon); w != aboutIconMaxSide || h != aboutIconMaxSide {
		t.Fatalf("the embedded icon is %vx%v, want %dx%d", w, h, aboutIconMaxSide, aboutIconMaxSide)
	}

	// Without one, the letter mark stays.
	bare := New(settings.Default(), t.TempDir())
	bare.loadAboutIcon()
	if bare.aboutIcon != nil {
		t.Fatal("an app without an icon reported one")
	}
}
