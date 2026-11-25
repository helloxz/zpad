package theme

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// seeds are the few colors a named theme names; everything else is derived
// from them so that a theme stays a short block of data.
type seeds struct {
	background, surface, border ui.Color
	text, muted, accent         ui.Color
	danger, warning, success    ui.Color

	// Optional: zero values are derived.
	selection, activeLine      ui.Color
	sidebar, tabBar, activeTab ui.Color

	// Syntax seed colors.
	keyword, str, comment, number ui.Color
	function, typ, tag, operator  ui.Color
}

func builtinThemes() []*Definition {
	return []*Definition{
		buildNamed("one-dark", "One Dark", true, seeds{
			background: ui.Hex("#282C34"), surface: ui.Hex("#2C313A"), border: ui.Hex("#3E4451"),
			text: ui.Hex("#ABB2BF"), muted: ui.Hex("#7F848E"), accent: ui.Hex("#61AFEF"),
			danger: ui.Hex("#E06C75"), warning: ui.Hex("#E5C07B"), success: ui.Hex("#98C379"),
			selection: ui.Hex("#3E4451"), activeLine: ui.Hex("#2C313A"),
			sidebar: ui.Hex("#21252B"), tabBar: ui.Hex("#21252B"), activeTab: ui.Hex("#282C34"),
			keyword: ui.Hex("#C678DD"), str: ui.Hex("#98C379"), comment: ui.Hex("#5C6370"),
			number: ui.Hex("#D19A66"), function: ui.Hex("#61AFEF"), typ: ui.Hex("#E5C07B"),
			tag: ui.Hex("#E06C75"), operator: ui.Hex("#56B6C2"),
		}),
		buildNamed("dracula", "Dracula", true, seeds{
			background: ui.Hex("#282A36"), surface: ui.Hex("#343746"), border: ui.Hex("#44475A"),
			text: ui.Hex("#F8F8F2"), muted: ui.Hex("#7A86B8"), accent: ui.Hex("#BD93F9"),
			danger: ui.Hex("#FF5555"), warning: ui.Hex("#FFB86C"), success: ui.Hex("#50FA7B"),
			selection: ui.Hex("#44475A"), activeLine: ui.Hex("#343746"),
			sidebar: ui.Hex("#21222C"), tabBar: ui.Hex("#21222C"), activeTab: ui.Hex("#282A36"),
			keyword: ui.Hex("#FF79C6"), str: ui.Hex("#F1FA8C"), comment: ui.Hex("#6272A4"),
			number: ui.Hex("#BD93F9"), function: ui.Hex("#50FA7B"), typ: ui.Hex("#8BE9FD"),
			tag: ui.Hex("#FF79C6"), operator: ui.Hex("#FF79C6"),
		}),
		buildNamed("nord", "Nord", true, seeds{
			background: ui.Hex("#2E3440"), surface: ui.Hex("#3B4252"), border: ui.Hex("#434C5E"),
			text: ui.Hex("#D8DEE9"), muted: ui.Hex("#81A1C1"), accent: ui.Hex("#88C0D0"),
			danger: ui.Hex("#BF616A"), warning: ui.Hex("#EBCB8B"), success: ui.Hex("#A3BE8C"),
			selection: ui.Hex("#434C5E"), activeLine: ui.Hex("#3B4252"),
			sidebar: ui.Hex("#292E39"), tabBar: ui.Hex("#292E39"), activeTab: ui.Hex("#2E3440"),
			keyword: ui.Hex("#81A1C1"), str: ui.Hex("#A3BE8C"), comment: ui.Hex("#7A869A"),
			number: ui.Hex("#B48EAD"), function: ui.Hex("#88C0D0"), typ: ui.Hex("#8FBCBB"),
			tag: ui.Hex("#81A1C1"), operator: ui.Hex("#81A1C1"),
		}),
		buildNamed("tokyo-night", "Tokyo Night", true, seeds{
			background: ui.Hex("#1A1B26"), surface: ui.Hex("#24283B"), border: ui.Hex("#2F3549"),
			text: ui.Hex("#C0CAF5"), muted: ui.Hex("#7982A9"), accent: ui.Hex("#7AA2F7"),
			danger: ui.Hex("#F7768E"), warning: ui.Hex("#E0AF68"), success: ui.Hex("#9ECE6A"),
			selection: ui.Hex("#283457"), activeLine: ui.Hex("#1E2030"),
			sidebar: ui.Hex("#16161E"), tabBar: ui.Hex("#16161E"), activeTab: ui.Hex("#1A1B26"),
			keyword: ui.Hex("#BB9AF7"), str: ui.Hex("#9ECE6A"), comment: ui.Hex("#565F89"),
			number: ui.Hex("#FF9E64"), function: ui.Hex("#7AA2F7"), typ: ui.Hex("#2AC3DE"),
			tag: ui.Hex("#F7768E"), operator: ui.Hex("#89DDFF"),
		}),
		buildNamed("catppuccin-mocha", "Catppuccin Mocha", true, seeds{
			background: ui.Hex("#1E1E2E"), surface: ui.Hex("#313244"), border: ui.Hex("#45475A"),
			text: ui.Hex("#CDD6F4"), muted: ui.Hex("#A6ADC8"), accent: ui.Hex("#CBA6F7"),
			danger: ui.Hex("#F38BA8"), warning: ui.Hex("#F9E2AF"), success: ui.Hex("#A6E3A1"),
			selection: ui.Hex("#585B70"), activeLine: ui.Hex("#26263A"),
			sidebar: ui.Hex("#181825"), tabBar: ui.Hex("#181825"), activeTab: ui.Hex("#1E1E2E"),
			keyword: ui.Hex("#CBA6F7"), str: ui.Hex("#A6E3A1"), comment: ui.Hex("#6C7086"),
			number: ui.Hex("#FAB387"), function: ui.Hex("#89B4FA"), typ: ui.Hex("#89DCEB"),
			tag: ui.Hex("#F38BA8"), operator: ui.Hex("#94E2D5"),
		}),
		buildNamed("github-light", "GitHub Light", false, seeds{
			background: ui.Hex("#FFFFFF"), surface: ui.Hex("#F6F8FA"), border: ui.Hex("#D0D7DE"),
			text: ui.Hex("#1F2328"), muted: ui.Hex("#656D76"), accent: ui.Hex("#0969DA"),
			danger: ui.Hex("#CF222E"), warning: ui.Hex("#9A6700"), success: ui.Hex("#1A7F37"),
			selection: ui.Hex("#B6D7FF"), activeLine: ui.Hex("#F6F8FA"),
			sidebar: ui.Hex("#F6F8FA"), tabBar: ui.Hex("#F6F8FA"), activeTab: ui.Hex("#FFFFFF"),
			keyword: ui.Hex("#CF222E"), str: ui.Hex("#0A3069"), comment: ui.Hex("#6E7781"),
			number: ui.Hex("#0550AE"), function: ui.Hex("#8250DF"), typ: ui.Hex("#953800"),
			tag: ui.Hex("#116329"), operator: ui.Hex("#CF222E"),
		}),
		buildNamed("one-light", "One Light", false, seeds{
			background: ui.Hex("#FAFAFA"), surface: ui.Hex("#EAEAEB"), border: ui.Hex("#D5D5D8"),
			text: ui.Hex("#383A42"), muted: ui.Hex("#696C77"), accent: ui.Hex("#4078F2"),
			danger: ui.Hex("#E45649"), warning: ui.Hex("#C18401"), success: ui.Hex("#50A14F"),
			selection: ui.Hex("#D7E3FA"), activeLine: ui.Hex("#F0F0F1"),
			sidebar: ui.Hex("#EAEAEB"), tabBar: ui.Hex("#F0F0F1"), activeTab: ui.Hex("#FAFAFA"),
			keyword: ui.Hex("#A626A4"), str: ui.Hex("#50A14F"), comment: ui.Hex("#A0A1A7"),
			number: ui.Hex("#986801"), function: ui.Hex("#4078F2"), typ: ui.Hex("#C18401"),
			tag: ui.Hex("#E45649"), operator: ui.Hex("#0184BC"),
		}),
		buildNamed("solarized-light", "Solarized Light", false, seeds{
			background: ui.Hex("#FDF6E3"), surface: ui.Hex("#EEE8D5"), border: ui.Hex("#DDD6C1"),
			text: ui.Hex("#586E75"), muted: ui.Hex("#74878B"), accent: ui.Hex("#268BD2"),
			danger: ui.Hex("#DC322F"), warning: ui.Hex("#B58900"), success: ui.Hex("#859900"),
			selection: ui.Hex("#E4DCC6"), activeLine: ui.Hex("#F5EED9"),
			sidebar: ui.Hex("#EEE8D5"), tabBar: ui.Hex("#EEE8D5"), activeTab: ui.Hex("#FDF6E3"),
			keyword: ui.Hex("#859900"), str: ui.Hex("#2AA198"), comment: ui.Hex("#93A1A1"),
			number: ui.Hex("#D33682"), function: ui.Hex("#268BD2"), typ: ui.Hex("#B58900"),
			tag: ui.Hex("#268BD2"), operator: ui.Hex("#657B83"),
		}),
		buildNamed("catppuccin-latte", "Catppuccin Latte", false, seeds{
			background: ui.Hex("#EFF1F5"), surface: ui.Hex("#E6E9EF"), border: ui.Hex("#CCD0DA"),
			text: ui.Hex("#4C4F69"), muted: ui.Hex("#6C6F85"), accent: ui.Hex("#8839EF"),
			danger: ui.Hex("#D20F39"), warning: ui.Hex("#DF8E1D"), success: ui.Hex("#40A02B"),
			selection: ui.Hex("#D7DAE0"), activeLine: ui.Hex("#E6E9EF"),
			sidebar: ui.Hex("#E6E9EF"), tabBar: ui.Hex("#E6E9EF"), activeTab: ui.Hex("#EFF1F5"),
			keyword: ui.Hex("#8839EF"), str: ui.Hex("#40A02B"), comment: ui.Hex("#9CA0B0"),
			number: ui.Hex("#FE640B"), function: ui.Hex("#1E66F5"), typ: ui.Hex("#04A5E5"),
			tag: ui.Hex("#D20F39"), operator: ui.Hex("#179299"),
		}),
		buildNamed("gruvbox-light", "Gruvbox Light", false, seeds{
			background: ui.Hex("#FBF1C7"), surface: ui.Hex("#F2E5BC"), border: ui.Hex("#D5C4A1"),
			text: ui.Hex("#3C3836"), muted: ui.Hex("#7C6F64"), accent: ui.Hex("#AF3A03"),
			danger: ui.Hex("#9D0006"), warning: ui.Hex("#B57614"), success: ui.Hex("#79740E"),
			selection: ui.Hex("#E5D5A8"), activeLine: ui.Hex("#F6EBC0"),
			sidebar: ui.Hex("#F2E5BC"), tabBar: ui.Hex("#F2E5BC"), activeTab: ui.Hex("#FBF1C7"),
			keyword: ui.Hex("#AF3A03"), str: ui.Hex("#79740E"), comment: ui.Hex("#928374"),
			number: ui.Hex("#8F3F71"), function: ui.Hex("#076678"), typ: ui.Hex("#B57614"),
			tag: ui.Hex("#9D0006"), operator: ui.Hex("#427B58"),
		}),
		buildNamed("monokai", "Monokai", true, seeds{
			background: ui.Hex("#272822"), surface: ui.Hex("#2E2F28"), border: ui.Hex("#49483E"),
			text: ui.Hex("#F8F8F2"), muted: ui.Hex("#A59F85"), accent: ui.Hex("#66D9EF"),
			danger: ui.Hex("#F92672"), warning: ui.Hex("#E6DB74"), success: ui.Hex("#A6E22E"),
			selection: ui.Hex("#49483E"), activeLine: ui.Hex("#3E3D32"),
			sidebar: ui.Hex("#1E1F1C"), tabBar: ui.Hex("#1E1F1C"), activeTab: ui.Hex("#272822"),
			keyword: ui.Hex("#F92672"), str: ui.Hex("#E6DB74"), comment: ui.Hex("#75715E"),
			number: ui.Hex("#AE81FF"), function: ui.Hex("#A6E22E"), typ: ui.Hex("#66D9EF"),
			tag: ui.Hex("#F92672"), operator: ui.Hex("#F92672"),
		}),
		buildNamed("gruvbox-dark", "Gruvbox Dark", true, seeds{
			background: ui.Hex("#282828"), surface: ui.Hex("#3C3836"), border: ui.Hex("#504945"),
			text: ui.Hex("#EBDBB2"), muted: ui.Hex("#A89984"), accent: ui.Hex("#FABD2F"),
			danger: ui.Hex("#FB4934"), warning: ui.Hex("#FABD2F"), success: ui.Hex("#B8BB26"),
			selection: ui.Hex("#504945"), activeLine: ui.Hex("#32302F"),
			sidebar: ui.Hex("#1D2021"), tabBar: ui.Hex("#1D2021"), activeTab: ui.Hex("#282828"),
			keyword: ui.Hex("#FB4934"), str: ui.Hex("#B8BB26"), comment: ui.Hex("#928374"),
			number: ui.Hex("#D3869B"), function: ui.Hex("#8EC07C"), typ: ui.Hex("#FABD2F"),
			tag: ui.Hex("#FB4934"), operator: ui.Hex("#FE8019"),
		}),
		buildNamed("rose-pine-dawn", "Rosé Pine Dawn", false, seeds{
			background: ui.Hex("#FAF4ED"), surface: ui.Hex("#F2E9E1"), border: ui.Hex("#DFDAD9"),
			text: ui.Hex("#575279"), muted: ui.Hex("#797593"), accent: ui.Hex("#286983"),
			danger: ui.Hex("#B4637A"), warning: ui.Hex("#EA9D34"), success: ui.Hex("#56949F"),
			selection: ui.Hex("#DFDAD9"), activeLine: ui.Hex("#F4EDE8"),
			sidebar: ui.Hex("#F2E9E1"), tabBar: ui.Hex("#F2E9E1"), activeTab: ui.Hex("#FAF4ED"),
			keyword: ui.Hex("#286983"), str: ui.Hex("#EA9D34"), comment: ui.Hex("#9893A5"),
			number: ui.Hex("#D7827E"), function: ui.Hex("#907AA9"), typ: ui.Hex("#56949F"),
			tag: ui.Hex("#B4637A"), operator: ui.Hex("#797593"),
		}),
		buildNamed("ayu-light", "Ayu Light", false, seeds{
			background: ui.Hex("#FAFAFA"), surface: ui.Hex("#F0F0F0"), border: ui.Hex("#E2E2E2"),
			text: ui.Hex("#5C6773"), muted: ui.Hex("#828C99"), accent: ui.Hex("#FF9940"),
			danger: ui.Hex("#F07178"), warning: ui.Hex("#FFAA33"), success: ui.Hex("#86B300"),
			selection: ui.Hex("#DCE7F2"), activeLine: ui.Hex("#F3F3F3"),
			sidebar: ui.Hex("#F0F0F0"), tabBar: ui.Hex("#F0F0F0"), activeTab: ui.Hex("#FAFAFA"),
			keyword: ui.Hex("#FF7733"), str: ui.Hex("#86B300"), comment: ui.Hex("#ABB0B6"),
			number: ui.Hex("#A37ACC"), function: ui.Hex("#F2AE49"), typ: ui.Hex("#55B4D4"),
			tag: ui.Hex("#F07178"), operator: ui.Hex("#ED9366"),
		}),
		buildNamed("everforest-light", "Everforest Light", false, seeds{
			background: ui.Hex("#FDF6E3"), surface: ui.Hex("#F2EFDF"), border: ui.Hex("#E1DDCB"),
			text: ui.Hex("#5C6A72"), muted: ui.Hex("#6E7A70"), accent: ui.Hex("#8DA101"),
			danger: ui.Hex("#F85552"), warning: ui.Hex("#DFA000"), success: ui.Hex("#8DA101"),
			selection: ui.Hex("#E8E5D5"), activeLine: ui.Hex("#F7F1DD"),
			sidebar: ui.Hex("#F2EFDF"), tabBar: ui.Hex("#F2EFDF"), activeTab: ui.Hex("#FDF6E3"),
			keyword: ui.Hex("#F85552"), str: ui.Hex("#8DA101"), comment: ui.Hex("#859289"),
			number: ui.Hex("#DFA000"), function: ui.Hex("#35A77C"), typ: ui.Hex("#3A94C5"),
			tag: ui.Hex("#F57D26"), operator: ui.Hex("#7A8478"),
		}),
		buildNamed("celadon", "Celadon", false, seeds{
			background: ui.Hex("#E8F1F2"), surface: ui.Hex("#DFEAEB"), border: ui.Hex("#CBDCDC"),
			text: ui.Hex("#2F4F4F"), muted: ui.Hex("#5F7777"), accent: ui.Hex("#2E6F73"),
			danger: ui.Hex("#B4544A"), warning: ui.Hex("#B58900"), success: ui.Hex("#4F7A5A"),
			selection: ui.Hex("#CCE0E2"), activeLine: ui.Hex("#E1ECED"),
			sidebar: ui.Hex("#DFEAEB"), tabBar: ui.Hex("#DFEAEB"), activeTab: ui.Hex("#E8F1F2"),
			keyword: ui.Hex("#2F6F73"), str: ui.Hex("#4F7A5A"), comment: ui.Hex("#6F8888"),
			number: ui.Hex("#9A7B4F"), function: ui.Hex("#3A6EA5"), typ: ui.Hex("#5B7C99"),
			tag: ui.Hex("#A45A52"), operator: ui.Hex("#5F7777"),
		}),
		buildNamed("ink-wash", "Ink Wash", false, seeds{
			background: ui.Hex("#F4EFE4"), surface: ui.Hex("#EAE3D3"), border: ui.Hex("#D8CFC0"),
			text: ui.Hex("#2B2A26"), muted: ui.Hex("#6E675C"), accent: ui.Hex("#A4423A"),
			danger: ui.Hex("#B03A2E"), warning: ui.Hex("#A67C2E"), success: ui.Hex("#5E7A5A"),
			selection: ui.Hex("#DCD2BE"), activeLine: ui.Hex("#EFE8DA"),
			sidebar: ui.Hex("#EDE6D8"), tabBar: ui.Hex("#EDE6D8"), activeTab: ui.Hex("#F4EFE4"),
			keyword: ui.Hex("#8C3B32"), str: ui.Hex("#5F7355"), comment: ui.Hex("#8A8377"),
			number: ui.Hex("#8C6A3C"), function: ui.Hex("#3E5D63"), typ: ui.Hex("#6B5E4A"),
			tag: ui.Hex("#A4423A"), operator: ui.Hex("#6E675C"),
		}),
		buildNamed("ink-night", "Ink Night", true, seeds{
			background: ui.Hex("#1F201D"), surface: ui.Hex("#292A26"), border: ui.Hex("#3A3B35"),
			text: ui.Hex("#E3DCCB"), muted: ui.Hex("#A09A8C"), accent: ui.Hex("#C97A68"),
			danger: ui.Hex("#D46A5A"), warning: ui.Hex("#C9A36A"), success: ui.Hex("#8FA37B"),
			selection: ui.Hex("#3A3B35"), activeLine: ui.Hex("#25261F"),
			sidebar: ui.Hex("#191A17"), tabBar: ui.Hex("#191A17"), activeTab: ui.Hex("#1F201D"),
			keyword: ui.Hex("#D17A6B"), str: ui.Hex("#9DB08A"), comment: ui.Hex("#77746A"),
			number: ui.Hex("#C9A36A"), function: ui.Hex("#8FA9B0"), typ: ui.Hex("#B7A6C4"),
			tag: ui.Hex("#C0604F"), operator: ui.Hex("#A09A8C"),
		}),
	}
}

func buildNamed(id, name string, dark bool, s seeds) *Definition {
	t := buildUI(s, dark)
	return &Definition{
		ID: id, Name: name, Dark: dark,
		UI:         t,
		FileTree:   buildFileTreeTheme(t),
		ResultList: buildResultListTheme(t),
		Chrome:     s.chrome(),
		Code:       buildCodeTheme(t, s),
	}
}

func buildUI(s seeds, dark bool) *ui.Theme {
	t := ui.LightTheme()
	if dark {
		t = ui.DarkTheme()
	}
	t.Background = s.background
	t.Surface = s.surface
	t.SurfaceHover = s.surface.Mix(s.text, 0.07)
	t.SurfacePressed = s.surface.Mix(s.text, 0.15)
	t.Border = s.border
	t.Text = s.text
	t.TextMuted = s.muted
	t.Accent = s.accent
	t.AccentText = inkOn(s.accent, s.background)
	if dark {
		t.AccentHover = s.accent.Mix(ui.RGB(255, 255, 255), 0.12)
		t.AccentPressed = s.accent.Mix(ui.RGB(0, 0, 0), 0.15)
	} else {
		t.AccentHover = s.accent.Mix(ui.RGB(0, 0, 0), 0.08)
		t.AccentPressed = s.accent.Mix(ui.RGB(0, 0, 0), 0.18)
	}
	t.Danger = s.danger
	t.Warning = s.warning
	t.Success = s.success
	t.Selection = s.selection
	if t.Selection == (ui.Color{}) {
		if dark {
			t.Selection = s.accent.Alpha(0.28)
		} else {
			t.Selection = s.accent.Alpha(0.22)
		}
	}
	t.Focus = s.accent.Alpha(0.6)
	t.FocusRingWidth = 2
	t.FocusRingOffset = 1

	thumb := s.text.Alpha(0.18)
	hover := s.text.Alpha(0.28)
	active := s.text.Alpha(0.38)
	if !dark {
		thumb = s.text.Alpha(0.20)
		hover = s.text.Alpha(0.32)
		active = s.text.Alpha(0.44)
	}
	t.Scrollbar = thumb
	t.ScrollbarWidth = 6
	t.ScrollbarStyle = ui.ScrollbarStyle{
		TrackWidth: 10, ThumbWidth: 6, Radius: 3, MinLength: 20,
		Thumb: thumb, Hover: hover, Active: active,
	}
	t.Radius = 8
	t.Spacing = 4
	t.FontSize = 13
	return t
}

func (s seeds) chrome() Chrome {
	return Chrome{
		Sidebar:   pick(s.sidebar, s.surface),
		TabBar:    pick(s.tabBar, s.surface),
		ActiveTab: pick(s.activeTab, s.background),
	}
}

func pick(c, fallback ui.Color) ui.Color {
	if c == (ui.Color{}) {
		return fallback
	}
	return c
}

func buildCodeTheme(t *ui.Theme, s seeds) *editor.Theme {
	ct := editor.DefaultTheme(t)
	ct.Background = s.background
	ct.Text = s.text
	ct.LineNumber = s.muted
	ct.ActiveLineNumber = s.text
	activeLine := s.activeLine
	if activeLine == (ui.Color{}) {
		if t.Dark {
			activeLine = s.background.Mix(s.text, 0.05)
		} else {
			activeLine = s.background.Mix(s.text, 0.035)
		}
	}
	ct.ActiveLine = activeLine
	ct.Selection = t.Selection
	ct.InactiveSelection = t.Selection
	ct.Cursor = s.accent
	ct.MatchingBracket = s.accent
	ct.Match = s.warning.Alpha(0.32)
	ct.CurrentMatch = s.warning.Alpha(0.62)
	ct.Occurrence = s.text.Alpha(0.10)
	ct.IndentGuide = s.border
	ct.ActiveIndentGuide = s.accent
	ct.WrapGuide = s.border
	ct.Whitespace = s.muted.Alpha(0.45)
	ct.Error = s.danger
	ct.Warning = s.warning
	ct.Info = s.accent
	ct.Hint = s.muted
	ct.Added = s.success
	ct.Deleted = s.danger
	ct.Modified = s.warning

	ct.Tokens[editor.TokenText] = editor.TokenStyle{Color: s.text}
	ct.Tokens[editor.TokenKeyword] = editor.TokenStyle{Color: s.keyword, Bold: true}
	ct.Tokens[editor.TokenType] = editor.TokenStyle{Color: s.typ}
	ct.Tokens[editor.TokenFunction] = editor.TokenStyle{Color: s.function}
	ct.Tokens[editor.TokenVariable] = editor.TokenStyle{Color: s.text}
	ct.Tokens[editor.TokenProperty] = editor.TokenStyle{Color: s.function}
	ct.Tokens[editor.TokenConstant] = editor.TokenStyle{Color: s.number}
	ct.Tokens[editor.TokenNumber] = editor.TokenStyle{Color: s.number}
	ct.Tokens[editor.TokenString] = editor.TokenStyle{Color: s.str}
	ct.Tokens[editor.TokenEscape] = editor.TokenStyle{Color: s.number}
	ct.Tokens[editor.TokenRegexp] = editor.TokenStyle{Color: s.str}
	ct.Tokens[editor.TokenComment] = editor.TokenStyle{Color: s.comment, Italic: true}
	ct.Tokens[editor.TokenOperator] = editor.TokenStyle{Color: s.operator}
	ct.Tokens[editor.TokenPunctuation] = editor.TokenStyle{Color: s.operator.Mix(s.text, 0.25)}
	ct.Tokens[editor.TokenTag] = editor.TokenStyle{Color: s.tag}
	ct.Tokens[editor.TokenAttribute] = editor.TokenStyle{Color: s.function}
	ct.Tokens[editor.TokenMeta] = editor.TokenStyle{Color: s.comment, Italic: true}
	ct.Tokens[editor.TokenHeading] = editor.TokenStyle{Color: s.text, Bold: true}
	ct.Tokens[editor.TokenEmphasis] = editor.TokenStyle{Color: s.text, Italic: true}
	ct.Tokens[editor.TokenStrong] = editor.TokenStyle{Color: s.text, Bold: true}
	ct.Tokens[editor.TokenLink] = editor.TokenStyle{Color: s.accent}
	ct.Tokens[editor.TokenCode] = editor.TokenStyle{Color: s.str}
	ct.Tokens[editor.TokenInserted] = editor.TokenStyle{Color: s.success}
	ct.Tokens[editor.TokenDeleted] = editor.TokenStyle{Color: s.danger}
	ct.Tokens[editor.TokenInvalid] = editor.TokenStyle{Color: s.danger}
	return ct
}
