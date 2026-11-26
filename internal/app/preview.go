package app

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/helloxz/zpad/internal/theme"
	"github.com/helloxz/zpad/locales"
)

// openExternal opens a web or mail address with the system's handler. It is
// a variable so tests can record the address without leaving the app.
var openExternal = mygo.Shell.OpenExternal

const (
	// markdownPreviewWidth is the reading width of the rendered text, which
	// stays centered in a wider pane.
	markdownPreviewWidth float32 = 820
	// markdownTableMaxPixels is how wide a table column may grow, in DIPs.
	// The Markdown view measures TableMaxColumn in ems, so the preview
	// divides this by its font size, keeping a column about this wide at
	// every font size.
	markdownTableMaxPixels float32 = 600
	// markdownPreviewMaxBytes and markdownPreviewMaxLines keep the preview
	// responsive: the Markdown view builds every block each frame, which is
	// right for a README and too much for a whole book.
	markdownPreviewMaxBytes = 1 << 20
	markdownPreviewMaxLines = 10_000
)

// isMarkdownPath reports whether path is Markdown the preview renders. MDX
// keeps the editor: its JSX would show as plain text.
func isMarkdownPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".mkd":
		return true
	}
	return false
}

// isMarkdownDoc reports whether doc is Markdown: a file whose name says so,
// or an untitled document given Markdown as its language in the status bar.
// Until it has a path, that choice is the only thing that says what the text
// is, so the preview has to go by it.
func isMarkdownDoc(doc *Document) bool {
	if doc == nil {
		return false
	}
	if doc.Path != "" {
		return isMarkdownPath(doc.Path)
	}
	return doc.Language != nil && doc.Language.Name == "Markdown"
}

// canPreviewTab reports whether tab can show a preview now. A preview that
// is already showing can always return to the source, even when the text
// has grown past the limit.
func (a *App) canPreviewTab(tab *Tab) bool {
	if tab == nil || !isMarkdownDoc(tab.doc) {
		return false
	}
	if tab.preview {
		return true
	}
	return tab.doc.Buffer.Len() <= markdownPreviewMaxBytes &&
		tab.doc.Buffer.LineCount() <= markdownPreviewMaxLines
}

// clearPreviewIfNotMarkdown returns tabs to the editor when a save-as,
// rename, or an untitled document's language choice stops their document
// being Markdown: there would be no preview control left to switch them back.
func (a *App) clearPreviewIfNotMarkdown(doc *Document) {
	if doc == nil || isMarkdownDoc(doc) {
		return
	}
	for _, p := range []*Pane{&a.left, &a.right} {
		for _, tab := range p.tabs {
			if tab.doc != doc || !tab.preview {
				continue
			}
			tab.preview = false
			if a.activeTab() == tab {
				a.focusEditor = true
			}
		}
	}
	a.dropPreviewOptions(doc)
}

// ToggleMarkdownPreview shows the active Markdown tab as rendered Markdown,
// or returns it to the editor.
func (a *App) ToggleMarkdownPreview() {
	tab := a.activeTab()
	if tab == nil || !isMarkdownDoc(tab.doc) {
		return
	}
	if !a.canPreviewTab(tab) {
		a.SetToast(locales.PreviewTooLarge())
		return
	}
	a.toggleTabPreview(tab)
}

// toggleTabPreview switches a tab between its editor and its preview.
func (a *App) toggleTabPreview(tab *Tab) {
	if !a.canPreviewTab(tab) {
		return
	}
	tab.preview = !tab.preview
	if !tab.preview {
		a.dropPreviewOptions(tab.doc)
	}
	if tab.preview && tab.vim != nil {
		tab.vim.Reset()
	}
	a.focusEditor = true
	a.updateTitle()
	a.update()
}

// dropPreviewOptions frees what a document's preview loaded, its decoded
// bitmaps included, once no tab shows it any more: editing needs none of it.
// A preview opened again builds the options anew, retrying the images that
// failed and decoding recent remote ones from temporary files on disk.
func (a *App) dropPreviewOptions(doc *Document) {
	if doc == nil {
		return
	}
	for _, p := range []*Pane{&a.left, &a.right} {
		for _, tab := range p.tabs {
			if tab.doc == doc && tab.preview {
				return
			}
		}
	}
	hadImages := doc.mdLoadedImage.Swap(false)
	doc.md = nil
	doc.mdThemeUI = nil
	doc.previewText = ""
	doc.previewLoaded = false
	if !a.hasOpenMarkdownDoc() {
		previewImages.clear()
	}
	if hadImages {
		scheduleLargeDocGC()
	}
}

func (a *App) hasOpenMarkdownDoc() bool {
	for _, p := range []*Pane{&a.left, &a.right} {
		for _, tab := range p.tabs {
			if isMarkdownDoc(tab.doc) {
				return true
			}
		}
	}
	return false
}

// PreviewText returns the buffer's text, copied only when it has changed.
func (d *Document) PreviewText() string {
	version := d.Buffer.Version()
	if !d.previewLoaded || version != d.previewVersion {
		d.previewText = d.Buffer.String()
		d.previewVersion = version
		d.previewLoaded = true
	}
	return d.previewText
}

// markdownOptions returns the document's preview options, creating them
// when the document's path changes, and gives them the window's theme.
func (a *App) markdownOptions(doc *Document, t *ui.Theme) *markdown.Options {
	if doc.md == nil || doc.mdPath != doc.Path {
		dir := a.root
		if doc.Path != "" {
			dir = filepath.Dir(doc.Path)
		}
		opts := &markdown.Options{LoadImage: func(raw string) (ui.ImageSource, error) {
			// Markdown written for GitHub often escapes spaces in image
			// paths; decode those before looking the file up. Addresses
			// keep their escapes, which mean something in a URL.
			if !strings.HasPrefix(raw, "data:") &&
				!strings.HasPrefix(raw, "http://") &&
				!strings.HasPrefix(raw, "https://") &&
				!strings.HasPrefix(raw, "//") {
				if decoded, err := url.PathUnescape(raw); err == nil {
					raw = decoded
				}
			}
			src, err := loadPreviewImage(dir, raw)
			if err == nil && src != nil {
				doc.mdLoadedImage.Store(true)
			}
			return src, err
		}}
		opts.OnLink = func(raw string) { a.openMarkdownLink(doc, raw) }
		doc.mdLoadedImage.Store(false)
		doc.md = opts
		doc.mdPath = doc.Path
		doc.mdThemeUI = nil
	}
	if doc.md.Theme == nil || doc.mdThemeUI != t ||
		doc.mdFontSize != a.Settings.FontSize || doc.mdFontFamily != a.Settings.FontFamily {
		previewTheme := markdown.DefaultTheme(t)
		previewTheme.FontSize = a.Settings.FontSize + 2
		previewTheme.LineHeight = 1.6
		previewTheme.CodeFont = a.Settings.FontFamily
		previewTheme.CodeColors = theme.EditorTheme(t)
		// The component's default is 24em, which wraps a table cell after
		// about 360 DIPs at the default size and leaves most of the reading
		// width empty.
		previewTheme.TableMaxColumn = markdownTableMaxPixels / previewTheme.FontSize
		doc.md.Theme = previewTheme
		doc.mdThemeUI = t
		doc.mdFontSize = a.Settings.FontSize
		doc.mdFontFamily = a.Settings.FontFamily
	}
	return doc.md
}

// markdownPreview builds the scrollable rendered document. focus takes the
// keyboard once, after switching to the preview. pane keys the view, so the
// same document previewed in both panes keeps two view states.
func (a *App) markdownPreview(c *ui.Context, t *ui.Theme, tab *Tab, pane int, focus bool) ui.Element {
	opts := a.markdownOptions(tab.doc, t)
	sc := ui.Scroll(c).Key(fmt.Sprintf("markdown-preview-%s-%d", tab.doc.ID, pane)).
		Grow(1).FillWidth().MinHeight(0).TrackScroll(&tab.previewScroll).Focusable()
	sc.Children(func() {
		ui.Column(c).FillWidth().MaxWidth(markdownPreviewWidth).
			AlignSelf(ui.Center).AlignItems(ui.Stretch).Padding(28, 32, 40, 32).
			Children(func() {
				markdown.View(c, tab.doc.PreviewText(), opts)
			})
	})
	if focus {
		sc.Bind(&a.editorRef)
		a.editorRef.Focus(c)
		a.focusEditor = false
	}
	return sc
}

// previewButton toggles the active Markdown tab's preview, beside the
// language it belongs to. Other files have no such action, so the status
// bar shows nothing for them.
func (a *App) previewButton(c *ui.Context, t *ui.Theme, tab *Tab) {
	if tab == nil || !isMarkdownDoc(tab.doc) {
		return
	}
	can := a.canPreviewTab(tab)
	label := locales.PreviewMarkdown()
	tip := label + " (" + markdownPreviewShortcut() + ")"
	if tab.preview {
		label = locales.BackToSource()
		tip = label + " (Esc / " + markdownPreviewShortcut() + ")"
	} else if !can {
		tip = locales.PreviewTooLarge()
	}
	b := ui.ButtonBase(c.Key("markdown-preview")).Size(20, 20).Center().Radius(4).
		Label(label).Tooltip(tip).
		KeyShortcuts(ui.Cmd|ui.Shift, ui.KeyV).
		Cursor(ui.CursorPointer).Disabled(!can)
	color := t.TextMuted
	if tab.preview {
		b.Background(t.SurfaceHover)
		color = t.Accent
	} else if b.Hovered() {
		b.Background(t.SurfaceHover)
		color = t.Text
	}
	b.Children(func() {
		ui.Icon(c, iconPreview).FontSize(13).TextColor(color)
	})
	if b.Clicked() {
		a.toggleTabPreview(tab)
	}
}

// markdownPreviewShortcut is the shortcut that toggles the preview, as its
// tooltip shows it.
func markdownPreviewShortcut() string {
	return shortcutText("CmdOrCtrl+Shift+V")
}

// openMarkdownLink follows a link clicked in a preview: headings scroll
// within it, web and mail addresses leave through the system, and files
// open in ZPad.
func (a *App) openMarkdownLink(doc *Document, raw string) {
	if strings.HasPrefix(raw, "#") {
		if doc.md != nil {
			anchor := strings.TrimPrefix(raw, "#")
			if decoded, err := url.PathUnescape(anchor); err == nil {
				anchor = decoded
			}
			doc.md.Anchor = anchor
			doc.md.AnchorVersion++
			a.update()
		}
		return
	}
	if ui.SafeToOpen(raw) {
		if err := openExternal(raw); err != nil {
			a.SetErrorToast(locales.OpenLinkFailed(), err.Error())
		}
		return
	}
	if filepath.IsAbs(raw) {
		a.OpenPaths([]string{raw})
		return
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "" && u.Scheme != "file") {
		return
	}
	path := u.Path
	if path == "" {
		return
	}
	if decoded, err := url.PathUnescape(path); err == nil {
		path = decoded
	}
	if u.Scheme == "file" {
		if len(path) >= 3 && path[0] == '/' && path[2] == ':' {
			path = path[1:] // Windows: /C:/dir/file
		}
		path = filepath.FromSlash(path)
	} else {
		base := a.root
		if doc.Path != "" {
			base = filepath.Dir(doc.Path)
		}
		path = filepath.Join(base, filepath.FromSlash(path))
	}
	a.OpenPaths([]string{path})
}
