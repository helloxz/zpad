// Package theme builds ZPad's palettes for the native UI and the editor.
package theme

import (
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// Theme ids of the default pair and of following the desktop.
const (
	SystemID = "system"
	LightID  = "light"
	DarkID   = "dark"
)

// Definition is one selectable theme: the UI palette, the colors of the
// window parts that ui.Theme has no fields for, and the editor's palette.
// The default light and dark themes keep the colors they always had; the
// named themes are built from a handful of seed colors.
type Definition struct {
	ID         string
	Name       string
	Dark       bool
	UI         *ui.Theme
	FileTree   *ui.Theme
	ResultList *ui.Theme
	// Chrome colors the sidebar, the tab bar and the chosen tab.
	Chrome Chrome
	Code   *editor.Theme
}

// Chrome is the palette of the window parts ui.Theme does not color.
type Chrome struct {
	Sidebar   ui.Color
	TabBar    ui.Color
	ActiveTab ui.Color
}

var (
	defaultLight *Definition
	defaultDark  *Definition
	named        []*Definition

	byID map[string]*Definition
	byUI map[*ui.Theme]*Definition
)

func init() {
	defaultLight = defaultDefinition(LightID, false)
	defaultDark = defaultDefinition(DarkID, true)
	named = builtinThemes()

	byID = make(map[string]*Definition, len(named)+2)
	byUI = make(map[*ui.Theme]*Definition, len(named)+2)
	for _, d := range append([]*Definition{defaultLight, defaultDark}, named...) {
		byID[d.ID] = d
		byUI[d.UI] = d
	}
}

// Default returns the default theme for an appearance.
func Default(dark bool) *Definition {
	if dark {
		return defaultDark
	}
	return defaultLight
}

// Lookup returns the theme with id, including the default light and dark
// themes, or false when no such theme exists.
func Lookup(id string) (*Definition, bool) {
	d, ok := byID[id]
	return d, ok
}

// Resolve returns the theme a setting stands for: the named one when it
// exists, the default theme for an appearance otherwise (also for
// "system"), or the default light theme for an unknown id.
func Resolve(id string, systemDark bool) *Definition {
	if id == SystemID {
		return Default(systemDark)
	}
	if d, ok := byID[id]; ok {
		return d
	}
	return Default(systemDark)
}

// Normalize returns id when it names a theme, the system setting for
// anything else, so that a settings file written by hand cannot leave the
// app with no palette.
func Normalize(id string) string {
	if id == SystemID {
		return id
	}
	if _, ok := byID[id]; ok {
		return id
	}
	return SystemID
}

// Named returns the named themes in display order, default light and dark
// excluded.
func Named() []*Definition { return named }

// For returns the palette for an appearance.
func For(dark bool) *ui.Theme { return Default(dark).UI }

// Light returns the light palette.
func Light() *ui.Theme { return defaultLight.UI }

// Dark returns the dark palette.
func Dark() *ui.Theme { return defaultDark.UI }

// Sidebar is the sidebar's background.
func Sidebar(t *ui.Theme) ui.Color { return definitionFor(t).Chrome.Sidebar }

// TabBar is the tab bar's background.
func TabBar(t *ui.Theme) ui.Color { return definitionFor(t).Chrome.TabBar }

// ActiveTab is the chosen tab's background.
func ActiveTab(t *ui.Theme) ui.Color { return definitionFor(t).Chrome.ActiveTab }

// Inverse is the fill of a tip and the color of text on it, as MyGo's own
// tooltip and toast choose them: the theme's Inverse and InverseText where
// it sets them, the text on the background otherwise. A tip that MyGo cannot
// place shares the look of the ones it can.
func Inverse(t *ui.Theme) (fill, text ui.Color) {
	fill, text = t.Inverse, t.InverseText
	if fill == ui.Transparent {
		fill = t.Text
	}
	if text == ui.Transparent {
		text = t.Background
	}
	return fill, text
}

// FileTree returns the palette of the sidebar's tree: the chosen row is a
// faint, full-width wash rather than a filled accent, as VS Code's and
// Zed's file trees, and its text keeps the normal color.
func FileTree(t *ui.Theme) *ui.Theme {
	if t != nil {
		if d, ok := byUI[t]; ok {
			return d.FileTree
		}
	}
	return buildFileTreeTheme(t)
}

func buildFileTreeTheme(t *ui.Theme) *ui.Theme {
	tree := *t
	if t.Dark {
		tree.Accent = ui.RGBA(255, 255, 255, 0.07)
	} else {
		tree.Accent = ui.RGBA(0, 0, 0, 0.06)
	}
	tree.InactiveAccent = tree.Accent
	tree.AccentText = t.Text
	tree.Radius = 0
	return &tree
}

// ResultList returns the palette of the search results: the chosen row is
// a faint wash too, so the highlight of the match inside it stays visible.
func ResultList(t *ui.Theme) *ui.Theme {
	if t != nil {
		if d, ok := byUI[t]; ok {
			return d.ResultList
		}
	}
	return buildResultListTheme(t)
}

func buildResultListTheme(t *ui.Theme) *ui.Theme {
	list := buildFileTreeTheme(t)
	if t.Dark {
		list.Accent = ui.RGBA(255, 255, 255, 0.1)
	} else {
		list.Accent = ui.RGBA(0, 0, 0, 0.07)
	}
	list.InactiveAccent = list.Accent
	return list
}

// EditorTheme colors the code editor to match t.
func EditorTheme(t *ui.Theme) *editor.Theme { return definitionFor(t).Code }

// definitionFor finds the theme t belongs to. A palette that did not come
// from this package, as a test's ui.LightTheme, falls back to the default
// for its appearance.
func definitionFor(t *ui.Theme) *Definition {
	if t != nil {
		if d, ok := byUI[t]; ok {
			return d
		}
		if t.Dark {
			return defaultDark
		}
	}
	return defaultLight
}

func defaultDefinition(id string, dark bool) *Definition {
	t := lightTheme()
	chrome := Chrome{
		Sidebar:   ui.Hex("#F3F4F6"),
		TabBar:    ui.Hex("#F7F8FA"),
		ActiveTab: ui.Hex("#FFFFFF"),
	}
	if dark {
		t = darkTheme()
		chrome = Chrome{
			Sidebar:   ui.Hex("#12161F"),
			TabBar:    ui.Hex("#151922"),
			ActiveTab: ui.Hex("#1B202B"),
		}
	}
	return &Definition{
		ID:         id,
		Dark:       dark,
		UI:         t,
		FileTree:   buildFileTreeTheme(t),
		ResultList: buildResultListTheme(t),
		Chrome:     chrome,
		Code:       buildEditorTheme(t),
	}
}

func lightTheme() *ui.Theme {
	t := ui.LightTheme()
	t.Background = ui.Hex("#FFFFFF")
	t.Surface = ui.Hex("#F7F8FA")
	t.SurfaceHover = ui.Hex("#EEF0F4")
	t.SurfacePressed = ui.Hex("#E2E5EA")
	t.Border = ui.Hex("#E5E7EB")
	t.Text = ui.Hex("#111827")
	t.TextMuted = ui.Hex("#6B7280")
	t.Accent = ui.Hex("#4F46E5")
	t.AccentHover = ui.Hex("#4338CA")
	t.AccentPressed = ui.Hex("#3730A3")
	t.AccentText = ui.Hex("#FFFFFF")
	t.Danger = ui.Hex("#DC2626")
	t.Warning = ui.Hex("#D97706")
	t.Success = ui.Hex("#16A34A")
	t.Selection = ui.RGBA(79, 70, 229, 0.16)
	t.Focus = ui.RGBA(79, 70, 229, 0.55)
	t.FocusRingWidth = 2
	t.FocusRingOffset = 1
	t.Scrollbar = ui.RGBA(0, 0, 0, 0.18)
	t.ScrollbarWidth = 6
	t.ScrollbarStyle = ui.ScrollbarStyle{
		TrackWidth: 10, ThumbWidth: 6, Radius: 3, MinLength: 20,
		Thumb: ui.RGBA(0, 0, 0, 0.18), Hover: ui.RGBA(0, 0, 0, 0.28), Active: ui.RGBA(0, 0, 0, 0.38),
	}
	t.Radius = 8
	t.Spacing = 4
	t.FontSize = 13
	return t
}

func darkTheme() *ui.Theme {
	t := ui.DarkTheme()
	t.Background = ui.Hex("#0F1115")
	t.Surface = ui.Hex("#151922")
	t.SurfaceHover = ui.Hex("#1F2531")
	t.SurfacePressed = ui.Hex("#252C3A")
	t.Border = ui.Hex("#262C38")
	t.Text = ui.Hex("#E6E9EF")
	t.TextMuted = ui.Hex("#9AA4B2")
	t.Accent = ui.Hex("#7C8CFF")
	t.AccentHover = ui.Hex("#93A1FF")
	t.AccentPressed = ui.Hex("#6B7AEB")
	t.AccentText = ui.Hex("#0F1115")
	t.Danger = ui.Hex("#F87171")
	t.Warning = ui.Hex("#FBBF24")
	t.Success = ui.Hex("#4ADE80")
	t.Selection = ui.Hex("#2A3350")
	t.Focus = ui.RGBA(124, 140, 255, 0.60)
	t.FocusRingWidth = 2
	t.FocusRingOffset = 1
	t.Scrollbar = ui.RGBA(255, 255, 255, 0.16)
	t.ScrollbarWidth = 6
	t.ScrollbarStyle = ui.ScrollbarStyle{
		TrackWidth: 10, ThumbWidth: 6, Radius: 3, MinLength: 20,
		Thumb: ui.RGBA(255, 255, 255, 0.16), Hover: ui.RGBA(255, 255, 255, 0.26), Active: ui.RGBA(255, 255, 255, 0.36),
	}
	t.Radius = 8
	t.Spacing = 4
	t.FontSize = 13
	return t
}

// buildEditorTheme colors the editor for one of the default palettes,
// keeping the token colors the app has always used.
func buildEditorTheme(t *ui.Theme) *editor.Theme {
	ct := editor.DefaultTheme(t)
	ct.Background = t.Background
	ct.Text = t.Text
	ct.LineNumber = t.TextMuted
	ct.ActiveLineNumber = t.Text
	if t.Dark {
		ct.ActiveLine = ui.Hex("#161B24")
	} else {
		ct.ActiveLine = ui.Hex("#F8FAFC")
	}
	ct.Selection = t.Selection
	ct.InactiveSelection = t.Selection
	ct.Cursor = t.Accent
	ct.MatchingBracket = t.Accent
	ct.IndentGuide = t.Border
	ct.ActiveIndentGuide = t.Accent
	ct.WrapGuide = t.Border
	ct.Whitespace = t.TextMuted
	ct.Error = t.Danger
	ct.Warning = t.Warning
	ct.Info = t.Accent
	ct.Hint = t.TextMuted
	ct.Added = t.Success
	ct.Deleted = t.Danger
	ct.Modified = t.Warning

	if t.Dark {
		ct.Match = ui.RGBA(92, 75, 31, 1)
		ct.CurrentMatch = ui.RGBA(183, 121, 31, 1)
		ct.Tokens[editor.TokenKeyword] = editor.TokenStyle{Color: ui.Hex("#C4A7FF"), Bold: true}
		ct.Tokens[editor.TokenString] = editor.TokenStyle{Color: ui.Hex("#7EE2B8")}
		ct.Tokens[editor.TokenComment] = editor.TokenStyle{Color: ui.Hex("#6B7280"), Italic: true}
		ct.Tokens[editor.TokenNumber] = editor.TokenStyle{Color: ui.Hex("#F5C26B")}
		ct.Tokens[editor.TokenFunction] = editor.TokenStyle{Color: ui.Hex("#8AB4FF")}
		ct.Tokens[editor.TokenType] = editor.TokenStyle{Color: ui.Hex("#7DD3FC")}
		ct.Tokens[editor.TokenVariable] = editor.TokenStyle{Color: t.Text}
		ct.Tokens[editor.TokenConstant] = editor.TokenStyle{Color: ui.Hex("#F5C26B")}
		ct.Tokens[editor.TokenOperator] = editor.TokenStyle{Color: ui.Hex("#9AA4B2")}
		ct.Tokens[editor.TokenPunctuation] = editor.TokenStyle{Color: ui.Hex("#9AA4B2")}
		ct.Tokens[editor.TokenTag] = editor.TokenStyle{Color: ui.Hex("#C4A7FF")}
		ct.Tokens[editor.TokenAttribute] = editor.TokenStyle{Color: ui.Hex("#8AB4FF")}
		ct.Tokens[editor.TokenMeta] = editor.TokenStyle{Color: ui.Hex("#6B7280"), Italic: true}
		ct.Tokens[editor.TokenHeading] = editor.TokenStyle{Color: t.Text, Bold: true}
		ct.Tokens[editor.TokenEmphasis] = editor.TokenStyle{Color: t.Text, Italic: true}
		ct.Tokens[editor.TokenStrong] = editor.TokenStyle{Color: t.Text, Bold: true}
		ct.Tokens[editor.TokenLink] = editor.TokenStyle{Color: ui.Hex("#8AB4FF")}
		ct.Tokens[editor.TokenCode] = editor.TokenStyle{Color: ui.Hex("#7EE2B8")}
		ct.Tokens[editor.TokenInserted] = editor.TokenStyle{Color: ui.Hex("#4ADE80")}
		ct.Tokens[editor.TokenDeleted] = editor.TokenStyle{Color: ui.Hex("#F87171")}
		ct.Tokens[editor.TokenInvalid] = editor.TokenStyle{Color: ui.Hex("#F87171")}
		return ct
	}

	ct.Match = ui.RGBA(253, 230, 138, 1)
	ct.CurrentMatch = ui.RGBA(245, 158, 11, 1)
	ct.Tokens[editor.TokenKeyword] = editor.TokenStyle{Color: ui.Hex("#7C3AED"), Bold: true}
	ct.Tokens[editor.TokenString] = editor.TokenStyle{Color: ui.Hex("#047857")}
	ct.Tokens[editor.TokenComment] = editor.TokenStyle{Color: ui.Hex("#9CA3AF"), Italic: true}
	ct.Tokens[editor.TokenNumber] = editor.TokenStyle{Color: ui.Hex("#B45309")}
	ct.Tokens[editor.TokenFunction] = editor.TokenStyle{Color: ui.Hex("#2563EB")}
	ct.Tokens[editor.TokenType] = editor.TokenStyle{Color: ui.Hex("#0E7490")}
	ct.Tokens[editor.TokenVariable] = editor.TokenStyle{Color: t.Text}
	ct.Tokens[editor.TokenConstant] = editor.TokenStyle{Color: ui.Hex("#B45309")}
	ct.Tokens[editor.TokenOperator] = editor.TokenStyle{Color: ui.Hex("#6B7280")}
	ct.Tokens[editor.TokenPunctuation] = editor.TokenStyle{Color: ui.Hex("#6B7280")}
	ct.Tokens[editor.TokenTag] = editor.TokenStyle{Color: ui.Hex("#7C3AED")}
	ct.Tokens[editor.TokenAttribute] = editor.TokenStyle{Color: ui.Hex("#2563EB")}
	ct.Tokens[editor.TokenMeta] = editor.TokenStyle{Color: ui.Hex("#9CA3AF"), Italic: true}
	ct.Tokens[editor.TokenHeading] = editor.TokenStyle{Color: t.Text, Bold: true}
	ct.Tokens[editor.TokenEmphasis] = editor.TokenStyle{Color: t.Text, Italic: true}
	ct.Tokens[editor.TokenStrong] = editor.TokenStyle{Color: t.Text, Bold: true}
	ct.Tokens[editor.TokenLink] = editor.TokenStyle{Color: ui.Hex("#2563EB")}
	ct.Tokens[editor.TokenCode] = editor.TokenStyle{Color: ui.Hex("#047857")}
	ct.Tokens[editor.TokenInserted] = editor.TokenStyle{Color: ui.Hex("#16A34A")}
	ct.Tokens[editor.TokenDeleted] = editor.TokenStyle{Color: ui.Hex("#DC2626")}
	ct.Tokens[editor.TokenInvalid] = editor.TokenStyle{Color: ui.Hex("#DC2626")}
	return ct
}

// contrast is the WCAG contrast ratio of two opaque colors, used to pick
// legible text on an accent.
func contrast(a, b ui.Color) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func luminance(c ui.Color) float64 {
	channel := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(c.R) + 0.7152*channel(c.G) + 0.0722*channel(c.B)
}

// inkOn picks the theme's background as text on fill when it is legible
// there, black or white otherwise, so primary buttons stay readable in
// every theme.
func inkOn(fill, background ui.Color) ui.Color {
	if contrast(fill, background) >= 4.5 {
		return background
	}
	black := ui.RGB(0, 0, 0)
	white := ui.RGB(255, 255, 255)
	if contrast(fill, black) >= contrast(fill, white) {
		return black
	}
	return white
}
