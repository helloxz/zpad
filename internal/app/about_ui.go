package app

import (
	"log"
	"os"
	"strings"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
	"github.com/helloxz/zpad/locales"
)

// Where the About dialog points: the project, the author's pages, and the
// site that lists the author's other apps. The license stays plain text on
// purpose: it names the terms, it does not need to open anything.
const (
	aboutRepoURL       = "https://github.com/helloxz/zpad"
	aboutBlogURL       = "https://blog.xiaoz.org/"
	aboutXURL          = "https://x.com/xiaozblog"
	aboutProductsURL   = "https://xphub.dev/"
	aboutProductsZhURL = "https://xphub.dev/zh"

	// aboutIconMaxSide caps the decoded About icon: it shows at 64×64 DIPs,
	// so 256 pixels keeps it crisp up to a 4× display without holding a
	// 1024×1024 RGBA buffer and its mipmaps.
	aboutIconMaxSide = 256
)

// aboutProductsLink is the author's product page in the interface language,
// so a Chinese reader lands on the Chinese list.
func aboutProductsLink() string {
	if strings.HasPrefix(i18n.Locale(), "zh") {
		return aboutProductsZhURL
	}
	return aboutProductsURL
}

// loadAboutIcon decodes the icon the binary embeds once, for the About
// dialog. Without one, as in tests, the dialog keeps the letter mark.
func (a *App) loadAboutIcon() {
	a.readAboutIconData(a.appIconPNG)
}

// readAboutIcon decodes the icon at path, as tests do with the one in the
// repository.
func (a *App) readAboutIcon(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	a.readAboutIconData(data)
}

func (a *App) readAboutIconData(data []byte) {
	if len(data) == 0 {
		return
	}
	icon, err := ui.DecodeBitmapScaled(data, aboutIconMaxSide)
	if err != nil {
		log.Printf("decode app icon: %v", err)
		return
	}
	a.aboutIcon = icon
}

func (a *App) openAbout() {
	a.showSettings = false
	a.renameOpen = false
	a.aboutOpen = true
	// The icon is a 1024×1024 bitmap: decode it when the dialog first
	// opens, not at every launch.
	if a.aboutIcon == nil {
		a.loadAboutIcon()
	}
	a.update()
}

// aboutModal shows the app's name, version, license, repository, and the
// author's pages. Links open in the browser; the license is plain text
// because it names the terms rather than pointing anywhere.
func (a *App) aboutModal(c *ui.Context, t *ui.Theme) {
	if !a.aboutOpen {
		a.aboutIcon = nil
		return
	}
	ui.DialogBase(c, &a.aboutOpen, func(backdrop, panel ui.Element) {
		c.SetTheme(t)
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4))
		panel.Width(440).Padding(24).Gap(16).
			Background(t.Background).Border(1, t.Border).Radius(14).
			Shadow(0, 16, 48, 0, ui.RGBA(0, 0, 0, 0.28))
		panel.Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(14).Children(func() {
				if a.aboutIcon != nil {
					// The icon art keeps its own rounded corners and
					// margin, so it takes no background or radius here.
					ui.Image(c, a.aboutIcon).Size(64, 64)
				} else {
					ui.Box(c).Size(52, 52).Center().Radius(12).Background(t.Accent).Children(func() {
						ui.Text(c, "Z").FontSize(26).Bold().TextColor(t.AccentText)
					})
				}
				ui.Column(c).Gap(2).Children(func() {
					ui.Text(c, "ZPad").FontSize(20).Bold()
					ui.Text(c, locales.AboutVersion(Version)).FontSize(12).TextColor(t.TextMuted)
				})
			})
			ui.Text(c, locales.AboutTagline()).FontSize(12).TextColor(t.TextMuted)
			ui.Divider(c)
			a.aboutValue(c, t, locales.AboutLicense(), "AGPL-3.0")
			a.aboutLink(c, t, locales.AboutRepository(), "github.com/helloxz/zpad", aboutRepoURL)
			ui.Divider(c)
			ui.Text(c, locales.AboutContactAuthor()).FontSize(12).Bold()
			a.aboutLink(c, t, locales.AboutBlog(), "blog.xiaoz.org", aboutBlogURL)
			a.aboutLink(c, t, locales.AboutX(), "@xiaozblog", aboutXURL)
			a.aboutLink(c, t, locales.AboutProducts(), "xphub.dev", aboutProductsLink())
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Spacer(c)
				if ui.PrimaryButton(c, locales.Close()).Cursor(ui.CursorPointer).Clicked() {
					a.aboutOpen = false
				}
			})
		})
	})
}

// aboutLink is one label and the link beside it.
func (a *App) aboutLink(c *ui.Context, t *ui.Theme, label, value, url string) {
	ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(t.TextMuted).Width(92)
		ui.Link(c, value, url).FontSize(12)
	})
}

// aboutValue is one label and its plain, unlinked value.
func (a *App) aboutValue(c *ui.Context, t *ui.Theme, label, value string) {
	ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(t.TextMuted).Width(92)
		ui.Text(c, value).FontSize(12)
	})
}
