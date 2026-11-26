package theme

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// seeds are the few colors a named theme names; everything else is derived
// from them so that a theme stays a short block of data while still allowing
// faithful per-theme token overrides where upstream specs call for them.
type seeds struct {
	background, surface, border ui.Color
	text, muted, accent         ui.Color
	danger, warning, success    ui.Color

	// Optional UI & editor overrides; zero values are derived.
	selection, activeLine      ui.Color
	sidebar, tabBar, activeTab ui.Color
	cursor                     ui.Color

	// Syntax seed colors.
	keyword, str, comment, number ui.Color
	function, typ, tag, operator  ui.Color

	// Optional syntax overrides; zero values fall back gracefully.
	property, attribute         ui.Color
	escape, regexp, punctuation ui.Color
	noBoldKeyword, italicType   bool
}

func builtinThemes() []*Definition {
	return []*Definition{
		buildNamed("one-dark", "One Dark", true, seeds{
			background: ui.Hex("#282C34"), surface: ui.Hex("#2D323B"), border: ui.Hex("#3E4451"),
			text: ui.Hex("#ABB2BF"), muted: ui.Hex("#7F848E"), accent: ui.Hex("#61AFEF"),
			danger: ui.Hex("#E06C75"), warning: ui.Hex("#E5C07B"), success: ui.Hex("#98C379"),
			selection: ui.Hex("#3B4250"), activeLine: ui.Hex("#2C313C"),
			sidebar: ui.Hex("#21252B"), tabBar: ui.Hex("#242830"), activeTab: ui.Hex("#282C34"),
			keyword: ui.Hex("#C678DD"), str: ui.Hex("#98C379"), comment: ui.Hex("#69707E"),
			number: ui.Hex("#D19A66"), function: ui.Hex("#61AFEF"), typ: ui.Hex("#E5C07B"),
			property: ui.Hex("#E06C75"), attribute: ui.Hex("#D19A66"), escape: ui.Hex("#56B6C2"),
			tag: ui.Hex("#E06C75"), operator: ui.Hex("#56B6C2"), punctuation: ui.Hex("#ABB2BF"),
			noBoldKeyword: true,
		}),
		buildNamed("dracula", "Dracula", true, seeds{
			background: ui.Hex("#282A36"), surface: ui.Hex("#313442"), border: ui.Hex("#44475A"),
			text: ui.Hex("#F2F2EC"), muted: ui.Hex("#7E8AB8"), accent: ui.Hex("#BD93F9"),
			danger: ui.Hex("#FF6B6B"), warning: ui.Hex("#FFB86C"), success: ui.Hex("#55E67D"),
			selection: ui.Hex("#414457"), activeLine: ui.Hex("#2E3140"),
			sidebar: ui.Hex("#21222C"), tabBar: ui.Hex("#242631"), activeTab: ui.Hex("#282A36"),
			cursor:  ui.Hex("#F2F2EC"),
			keyword: ui.Hex("#F476BE"), str: ui.Hex("#ECEFA0"), comment: ui.Hex("#6D7DB3"),
			number: ui.Hex("#BD93F9"), function: ui.Hex("#55E67D"), typ: ui.Hex("#80DEF2"),
			property: ui.Hex("#80DEF2"), attribute: ui.Hex("#55E67D"), escape: ui.Hex("#FF92D0"),
			tag: ui.Hex("#F476BE"), operator: ui.Hex("#F476BE"), punctuation: ui.Hex("#E6E6E0"),
			noBoldKeyword: true, italicType: true,
		}),
		buildNamed("nord", "Nord", true, seeds{
			background: ui.Hex("#2E3440"), surface: ui.Hex("#373E4D"), border: ui.Hex("#434C5E"),
			text: ui.Hex("#D8DEE9"), muted: ui.Hex("#8696B0"), accent: ui.Hex("#88C0D0"),
			danger: ui.Hex("#BF616A"), warning: ui.Hex("#EBCB8B"), success: ui.Hex("#A3BE8C"),
			selection: ui.Hex("#434C5E"), activeLine: ui.Hex("#343B49"),
			sidebar: ui.Hex("#272C36"), tabBar: ui.Hex("#2A303B"), activeTab: ui.Hex("#2E3440"),
			keyword: ui.Hex("#81A1C1"), str: ui.Hex("#A3BE8C"), comment: ui.Hex("#7B88A1"),
			number: ui.Hex("#B48EAD"), function: ui.Hex("#88C0D0"), typ: ui.Hex("#8FBCBB"),
			property: ui.Hex("#8FBCBB"), attribute: ui.Hex("#8FBCBB"), escape: ui.Hex("#EBCB8B"),
			tag: ui.Hex("#81A1C1"), operator: ui.Hex("#81A1C1"), punctuation: ui.Hex("#C5CEDB"),
			noBoldKeyword: true,
		}),
		buildNamed("tokyo-night", "Tokyo Night", true, seeds{
			background: ui.Hex("#1A1B26"), surface: ui.Hex("#222433"), border: ui.Hex("#2F3549"),
			text: ui.Hex("#C0CAF5"), muted: ui.Hex("#7982A9"), accent: ui.Hex("#7AA2F7"),
			danger: ui.Hex("#F7768E"), warning: ui.Hex("#E0AF68"), success: ui.Hex("#9ECE6A"),
			selection: ui.Hex("#283457"), activeLine: ui.Hex("#1F2230"),
			sidebar: ui.Hex("#15161E"), tabBar: ui.Hex("#171822"), activeTab: ui.Hex("#1A1B26"),
			keyword: ui.Hex("#BB9AF7"), str: ui.Hex("#9ECE6A"), comment: ui.Hex("#636D9C"),
			number: ui.Hex("#FF9E64"), function: ui.Hex("#7AA2F7"), typ: ui.Hex("#2AC3DE"),
			property: ui.Hex("#73DACA"), attribute: ui.Hex("#E0AF68"), escape: ui.Hex("#89DDFF"),
			tag: ui.Hex("#F7768E"), operator: ui.Hex("#89DDFF"), punctuation: ui.Hex("#9AA5CE"),
			noBoldKeyword: true,
		}),
		buildNamed("catppuccin-mocha", "Catppuccin Mocha", true, seeds{
			background: ui.Hex("#1E1E2E"), surface: ui.Hex("#2A2B3D"), border: ui.Hex("#45475A"),
			text: ui.Hex("#CDD6F4"), muted: ui.Hex("#A6ADC8"), accent: ui.Hex("#CBA6F7"),
			danger: ui.Hex("#F38BA8"), warning: ui.Hex("#F9E2AF"), success: ui.Hex("#A6E3A1"),
			selection: ui.Hex("#3B3D52"), activeLine: ui.Hex("#252637"),
			sidebar: ui.Hex("#181825"), tabBar: ui.Hex("#1B1B29"), activeTab: ui.Hex("#1E1E2E"),
			keyword: ui.Hex("#CBA6F7"), str: ui.Hex("#A6E3A1"), comment: ui.Hex("#747991"),
			number: ui.Hex("#FAB387"), function: ui.Hex("#89B4FA"), typ: ui.Hex("#F9E2AF"),
			property: ui.Hex("#B4BEFE"), attribute: ui.Hex("#F9E2AF"), escape: ui.Hex("#F5C2E7"),
			tag: ui.Hex("#F38BA8"), operator: ui.Hex("#94E2D5"), punctuation: ui.Hex("#9399B2"),
		}),
		buildNamed("github-light", "GitHub Light", false, seeds{
			background: ui.Hex("#FFFFFF"), surface: ui.Hex("#F6F8FA"), border: ui.Hex("#D0D7DE"),
			text: ui.Hex("#1F2328"), muted: ui.Hex("#636C76"), accent: ui.Hex("#0969DA"),
			danger: ui.Hex("#CF222E"), warning: ui.Hex("#9A6700"), success: ui.Hex("#1A7F37"),
			selection: ui.Hex("#CEE3FF"), activeLine: ui.Hex("#F6F8FA"),
			sidebar: ui.Hex("#F0F3F6"), tabBar: ui.Hex("#F6F8FA"), activeTab: ui.Hex("#FFFFFF"),
			keyword: ui.Hex("#C92A35"), str: ui.Hex("#0A3069"), comment: ui.Hex("#6E7781"),
			number: ui.Hex("#0550AE"), function: ui.Hex("#7A49D4"), typ: ui.Hex("#953800"),
			property: ui.Hex("#0550AE"), attribute: ui.Hex("#0550AE"), escape: ui.Hex("#0550AE"),
			regexp: ui.Hex("#116329"), tag: ui.Hex("#116329"), operator: ui.Hex("#C92A35"),
			punctuation: ui.Hex("#24292F"), noBoldKeyword: true,
		}),
		buildNamed("one-light", "One Light", false, seeds{
			background: ui.Hex("#FAFAFA"), surface: ui.Hex("#F0F1F3"), border: ui.Hex("#D8DADF"),
			text: ui.Hex("#383A42"), muted: ui.Hex("#696C77"), accent: ui.Hex("#3267DC"),
			danger: ui.Hex("#D6453A"), warning: ui.Hex("#B07600"), success: ui.Hex("#3B873A"),
			selection: ui.Hex("#D7E3FA"), activeLine: ui.Hex("#F2F3F5"),
			sidebar: ui.Hex("#EAECEF"), tabBar: ui.Hex("#F1F2F4"), activeTab: ui.Hex("#FAFAFA"),
			keyword: ui.Hex("#982496"), str: ui.Hex("#3B873A"), comment: ui.Hex("#848791"),
			number: ui.Hex("#986801"), function: ui.Hex("#3267DC"), typ: ui.Hex("#9E6B00"),
			property: ui.Hex("#C94038"), attribute: ui.Hex("#986801"), escape: ui.Hex("#0177AA"),
			tag: ui.Hex("#C94038"), operator: ui.Hex("#0177AA"), punctuation: ui.Hex("#5C606B"),
			noBoldKeyword: true,
		}),
		buildNamed("solarized-light", "Solarized Light", false, seeds{
			background: ui.Hex("#FDF6E3"), surface: ui.Hex("#F2ECDA"), border: ui.Hex("#DDD6C1"),
			text: ui.Hex("#52676E"), muted: ui.Hex("#6F8287"), accent: ui.Hex("#1D70B0"),
			danger: ui.Hex("#C92C29"), warning: ui.Hex("#9E7600"), success: ui.Hex("#6A8000"),
			selection: ui.Hex("#E5DEC8"), activeLine: ui.Hex("#F6EFE0"),
			sidebar: ui.Hex("#EAE3CE"), tabBar: ui.Hex("#F1EBD8"), activeTab: ui.Hex("#FDF6E3"),
			keyword: ui.Hex("#6A8000"), str: ui.Hex("#1D827A"), comment: ui.Hex("#829090"),
			number: ui.Hex("#B52B6F"), function: ui.Hex("#1E73B3"), typ: ui.Hex("#966F00"),
			property: ui.Hex("#236E8E"), attribute: ui.Hex("#966F00"), escape: ui.Hex("#B84314"),
			tag: ui.Hex("#1E73B3"), operator: ui.Hex("#586E75"), punctuation: ui.Hex("#657B83"),
			noBoldKeyword: true,
		}),
		buildNamed("catppuccin-latte", "Catppuccin Latte", false, seeds{
			background: ui.Hex("#EFF1F5"), surface: ui.Hex("#E6E9EF"), border: ui.Hex("#CCD0DA"),
			text: ui.Hex("#4C4F69"), muted: ui.Hex("#6C6F85"), accent: ui.Hex("#6A44C9"),
			danger: ui.Hex("#C4143C"), warning: ui.Hex("#C47A12"), success: ui.Hex("#2E8520"),
			selection: ui.Hex("#D4D8E2"), activeLine: ui.Hex("#E8EBF1"),
			sidebar: ui.Hex("#E1E5EC"), tabBar: ui.Hex("#E7EAF0"), activeTab: ui.Hex("#EFF1F5"),
			keyword: ui.Hex("#7239D6"), str: ui.Hex("#2E8520"), comment: ui.Hex("#808599"),
			number: ui.Hex("#C84D06"), function: ui.Hex("#1D58D8"), typ: ui.Hex("#A16308"),
			property: ui.Hex("#29629E"), attribute: ui.Hex("#A16308"), escape: ui.Hex("#B83B7E"),
			tag: ui.Hex("#C4143C"), operator: ui.Hex("#147880"), punctuation: ui.Hex("#6C6F85"),
		}),
		buildNamed("gruvbox-light", "Gruvbox Light", false, seeds{
			background: ui.Hex("#FBF1C7"), surface: ui.Hex("#F2E5BC"), border: ui.Hex("#D5C4A1"),
			text: ui.Hex("#3C3836"), muted: ui.Hex("#7C6F64"), accent: ui.Hex("#AF3A03"),
			danger: ui.Hex("#9D0006"), warning: ui.Hex("#A86C0E"), success: ui.Hex("#6D690B"),
			selection: ui.Hex("#E5D5A8"), activeLine: ui.Hex("#F5E8BF"),
			sidebar: ui.Hex("#EBDBB2"), tabBar: ui.Hex("#F2E5BC"), activeTab: ui.Hex("#FBF1C7"),
			keyword: ui.Hex("#AF3A03"), str: ui.Hex("#6D690B"), comment: ui.Hex("#8C7D6E"),
			number: ui.Hex("#8F3F71"), function: ui.Hex("#076678"), typ: ui.Hex("#965F0D"),
			property: ui.Hex("#076678"), attribute: ui.Hex("#965F0D"), escape: ui.Hex("#AF3A03"),
			tag: ui.Hex("#9D0006"), operator: ui.Hex("#427B58"), punctuation: ui.Hex("#665C54"),
			noBoldKeyword: true,
		}),
		buildNamed("monokai", "Monokai", true, seeds{
			background: ui.Hex("#272822"), surface: ui.Hex("#2E2F28"), border: ui.Hex("#414339"),
			text: ui.Hex("#F2F2EB"), muted: ui.Hex("#90908A"), accent: ui.Hex("#DFBD56"),
			danger: ui.Hex("#F83A7E"), warning: ui.Hex("#E6DB74"), success: ui.Hex("#A3DC39"),
			selection: ui.Hex("#44453B"), activeLine: ui.Hex("#31322B"),
			sidebar: ui.Hex("#1E1F1C"), tabBar: ui.Hex("#22231F"), activeTab: ui.Hex("#272822"),
			cursor:  ui.Hex("#F8F8F0"),
			keyword: ui.Hex("#F83A7E"), str: ui.Hex("#E6DB74"), comment: ui.Hex("#7A7663"),
			number: ui.Hex("#AE81FF"), function: ui.Hex("#A3DC39"), typ: ui.Hex("#66D9EF"),
			property: ui.Hex("#F2F2EB"), attribute: ui.Hex("#A3DC39"), escape: ui.Hex("#AE81FF"),
			tag: ui.Hex("#F83A7E"), operator: ui.Hex("#F83A7E"), punctuation: ui.Hex("#EAEAE2"),
			noBoldKeyword: true, italicType: true,
		}),
		buildNamed("gruvbox-dark", "Gruvbox Dark", true, seeds{
			background: ui.Hex("#282828"), surface: ui.Hex("#343130"), border: ui.Hex("#504945"),
			text: ui.Hex("#EBDBB2"), muted: ui.Hex("#A89984"), accent: ui.Hex("#FABD2F"),
			danger: ui.Hex("#EA5843"), warning: ui.Hex("#FABD2F"), success: ui.Hex("#B8BB26"),
			selection: ui.Hex("#45403D"), activeLine: ui.Hex("#302E2D"),
			sidebar: ui.Hex("#1D2021"), tabBar: ui.Hex("#232323"), activeTab: ui.Hex("#282828"),
			keyword: ui.Hex("#EA5843"), str: ui.Hex("#B8BB26"), comment: ui.Hex("#928374"),
			number: ui.Hex("#D3869B"), function: ui.Hex("#8EC07C"), typ: ui.Hex("#FABD2F"),
			property: ui.Hex("#83A598"), attribute: ui.Hex("#FABD2F"), escape: ui.Hex("#FE8019"),
			tag: ui.Hex("#EA5843"), operator: ui.Hex("#FE8019"), punctuation: ui.Hex("#BDAE93"),
			noBoldKeyword: true,
		}),
		buildNamed("rose-pine-dawn", "Rosé Pine Dawn", false, seeds{
			background: ui.Hex("#FAF4ED"), surface: ui.Hex("#F2E9E1"), border: ui.Hex("#DFDAD9"),
			text: ui.Hex("#575279"), muted: ui.Hex("#797593"), accent: ui.Hex("#286983"),
			danger: ui.Hex("#B4546C"), warning: ui.Hex("#C97C18"), success: ui.Hex("#3E7C87"),
			selection: ui.Hex("#E4DDD9"), activeLine: ui.Hex("#F4ECE5"),
			sidebar: ui.Hex("#EDE3DA"), tabBar: ui.Hex("#F3EAE2"), activeTab: ui.Hex("#FAF4ED"),
			keyword: ui.Hex("#286983"), str: ui.Hex("#A86310"), comment: ui.Hex("#8A8498"),
			number: ui.Hex("#AD5654"), function: ui.Hex("#7A6196"), typ: ui.Hex("#3C7882"),
			property: ui.Hex("#487584"), attribute: ui.Hex("#7A6196"), escape: ui.Hex("#B4546C"),
			tag: ui.Hex("#B4546C"), operator: ui.Hex("#6E6A86"),
		}),
		buildNamed("ayu-light", "Ayu Light", false, seeds{
			background: ui.Hex("#FAFAFA"), surface: ui.Hex("#F2F3F5"), border: ui.Hex("#DFE2E6"),
			text: ui.Hex("#525C68"), muted: ui.Hex("#747E8A"), accent: ui.Hex("#B8510B"),
			danger: ui.Hex("#C9424A"), warning: ui.Hex("#C4740E"), success: ui.Hex("#5B8200"),
			selection: ui.Hex("#DAE6F2"), activeLine: ui.Hex("#F3F4F6"),
			sidebar: ui.Hex("#ECEEF1"), tabBar: ui.Hex("#F2F3F5"), activeTab: ui.Hex("#FAFAFA"),
			keyword: ui.Hex("#C45316"), str: ui.Hex("#5B8200"), comment: ui.Hex("#868E99"),
			number: ui.Hex("#7F52B5"), function: ui.Hex("#A66508"), typ: ui.Hex("#1E7B9C"),
			property: ui.Hex("#2E6E8E"), attribute: ui.Hex("#A66508"), escape: ui.Hex("#427C68"),
			tag: ui.Hex("#C4454D"), operator: ui.Hex("#AC5C28"), punctuation: ui.Hex("#6E7885"),
			noBoldKeyword: true,
		}),
		buildNamed("everforest-light", "Everforest Light", false, seeds{
			background: ui.Hex("#F7F3E3"), surface: ui.Hex("#EEEAD8"), border: ui.Hex("#DCD7C3"),
			text: ui.Hex("#536068"), muted: ui.Hex("#6D7970"), accent: ui.Hex("#577820"),
			danger: ui.Hex("#C44340"), warning: ui.Hex("#B37A00"), success: ui.Hex("#5B7808"),
			selection: ui.Hex("#E2DEC9"), activeLine: ui.Hex("#F1EDDC"),
			sidebar: ui.Hex("#E7E3CF"), tabBar: ui.Hex("#EFEBD9"), activeTab: ui.Hex("#F7F3E3"),
			keyword: ui.Hex("#C44340"), str: ui.Hex("#5B7808"), comment: ui.Hex("#7D8980"),
			number: ui.Hex("#9E6700"), function: ui.Hex("#237D5B"), typ: ui.Hex("#26729C"),
			property: ui.Hex("#356E68"), attribute: ui.Hex("#9E6700"), escape: ui.Hex("#B85414"),
			tag: ui.Hex("#BA5614"), operator: ui.Hex("#687368"),
		}),
		buildNamed("celadon", "Celadon", false, seeds{
			background: ui.Hex("#EAF1F1"), surface: ui.Hex("#E1EBEB"), border: ui.Hex("#CBDCDC"),
			text: ui.Hex("#2E4C4C"), muted: ui.Hex("#5B7373"), accent: ui.Hex("#2E6F73"),
			danger: ui.Hex("#AD4B42"), warning: ui.Hex("#9C7300"), success: ui.Hex("#477352"),
			selection: ui.Hex("#CDE0E2"), activeLine: ui.Hex("#E3EDED"),
			sidebar: ui.Hex("#DAE6E7"), tabBar: ui.Hex("#E2ECEC"), activeTab: ui.Hex("#EAF1F1"),
			keyword: ui.Hex("#29666A"), str: ui.Hex("#477352"), comment: ui.Hex("#6E8686"),
			number: ui.Hex("#856539"), function: ui.Hex("#336599"), typ: ui.Hex("#486C8A"),
			property: ui.Hex("#2E6478"), attribute: ui.Hex("#6D5E46"), escape: ui.Hex("#8C5B3E"),
			tag: ui.Hex("#9E4F47"), operator: ui.Hex("#566E6E"),
		}),
		buildNamed("ink-wash", "Ink Wash", false, seeds{
			background: ui.Hex("#F4EFE4"), surface: ui.Hex("#EAE3D3"), border: ui.Hex("#D8CFC0"),
			text: ui.Hex("#2B2A26"), muted: ui.Hex("#6E675C"), accent: ui.Hex("#A4423A"),
			danger: ui.Hex("#B03A2E"), warning: ui.Hex("#9C7122"), success: ui.Hex("#567352"),
			selection: ui.Hex("#DCD2BE"), activeLine: ui.Hex("#EFE8DA"),
			sidebar: ui.Hex("#E5DEC9"), tabBar: ui.Hex("#EDE6D8"), activeTab: ui.Hex("#F4EFE4"),
			keyword: ui.Hex("#8C3B32"), str: ui.Hex("#56704D"), comment: ui.Hex("#827B6F"),
			number: ui.Hex("#856335"), function: ui.Hex("#395960"), typ: ui.Hex("#665844"),
			property: ui.Hex("#4A5D6A"), attribute: ui.Hex("#785E3B"), escape: ui.Hex("#8F5334"),
			tag: ui.Hex("#A4423A"), operator: ui.Hex("#686156"),
		}),
		buildNamed("ink-night", "Ink Night", true, seeds{
			background: ui.Hex("#1F201D"), surface: ui.Hex("#292A26"), border: ui.Hex("#3A3B35"),
			text: ui.Hex("#E3DCCB"), muted: ui.Hex("#A09A8C"), accent: ui.Hex("#C97A68"),
			danger: ui.Hex("#D46A5A"), warning: ui.Hex("#C9A36A"), success: ui.Hex("#8FA37B"),
			selection: ui.Hex("#383933"), activeLine: ui.Hex("#252621"),
			sidebar: ui.Hex("#181916"), tabBar: ui.Hex("#1B1C19"), activeTab: ui.Hex("#1F201D"),
			keyword: ui.Hex("#D17A6B"), str: ui.Hex("#9DB08A"), comment: ui.Hex("#7C796E"),
			number: ui.Hex("#C9A36A"), function: ui.Hex("#8FA9B0"), typ: ui.Hex("#B7A6C4"),
			property: ui.Hex("#9BB0A8"), attribute: ui.Hex("#C9A36A"), escape: ui.Hex("#D4956A"),
			tag: ui.Hex("#C46858"), operator: ui.Hex("#A09A8C"), punctuation: ui.Hex("#B0AA9B"),
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
	t.SurfacePressed = s.surface.Mix(s.text, 0.14)
	t.Border = s.border
	t.Text = s.text
	t.TextMuted = s.muted
	t.Accent = s.accent
	t.AccentText = inkOn(s.accent, s.background, s.text)
	if dark {
		t.AccentHover = s.accent.Mix(ui.RGB(255, 255, 255), 0.12)
		t.AccentPressed = s.accent.Mix(ui.RGB(0, 0, 0), 0.15)
		t.Inverse = s.surface.Mix(s.text, 0.14)
		t.InverseText = s.text
	} else {
		t.AccentHover = s.accent.Mix(ui.RGB(0, 0, 0), 0.08)
		t.AccentPressed = s.accent.Mix(ui.RGB(0, 0, 0), 0.18)
		t.Inverse = s.text.Mix(s.background, 0.08)
		t.InverseText = s.background
	}
	t.Danger = s.danger
	t.Warning = s.warning
	t.Success = s.success
	t.Selection = s.selection
	if t.Selection == (ui.Color{}) {
		if dark {
			t.Selection = s.accent.Alpha(0.26)
		} else {
			t.Selection = s.accent.Alpha(0.20)
		}
	}
	t.Focus = s.accent.Alpha(0.55)
	t.FocusRingWidth = 2
	t.FocusRingOffset = 1

	thumb := s.text.Alpha(0.16)
	hover := s.text.Alpha(0.26)
	active := s.text.Alpha(0.36)
	if !dark {
		thumb = s.text.Alpha(0.18)
		hover = s.text.Alpha(0.30)
		active = s.text.Alpha(0.42)
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
	ct.LineNumber = s.muted.Alpha(0.78)
	ct.ActiveLineNumber = s.text
	activeLine := s.activeLine
	if activeLine == (ui.Color{}) {
		if t.Dark {
			activeLine = s.background.Mix(s.text, 0.045)
		} else {
			activeLine = s.background.Mix(s.text, 0.03)
		}
	}
	ct.ActiveLine = activeLine
	ct.Selection = t.Selection
	if t.Selection.A == 255 {
		ct.InactiveSelection = t.Selection.Mix(s.background, 0.45)
	} else {
		ct.InactiveSelection = t.Selection.Alpha(float32(t.Selection.A) / 255 * 0.55)
	}
	ct.Cursor = pick(s.cursor, s.accent)
	ct.MatchingBracket = s.accent.Alpha(0.45)
	if t.Dark {
		amber := s.warning.Mix(ui.RGB(205, 135, 28), 0.45)
		ct.Match = amber.Alpha(0.18)
		ct.CurrentMatch = amber.Alpha(0.28)
		ct.Occurrence = s.text.Alpha(0.09)
	} else {
		gold := ui.RGB(245, 184, 46).Mix(s.warning, 0.20)
		ct.Match = gold.Alpha(0.25)
		ct.CurrentMatch = ui.RGB(238, 152, 32).Mix(s.warning, 0.20).Alpha(0.34)
		ct.Occurrence = s.text.Alpha(0.07)
	}
	ct.IndentGuide = s.border.Alpha(0.58)
	ct.ActiveIndentGuide = s.muted.Mix(s.accent, 0.35).Alpha(0.58)
	ct.WrapGuide = s.border.Alpha(0.58)
	ct.Whitespace = s.muted.Alpha(0.42)
	ct.Error = s.danger
	ct.Warning = s.warning
	ct.Info = s.accent
	ct.Hint = s.muted
	ct.Added = s.success
	ct.Deleted = s.danger
	ct.Modified = s.warning

	prop := pick(s.property, s.function)
	attr := pick(s.attribute, prop)
	esc := pick(s.escape, s.number)
	rx := pick(s.regexp, s.str)
	punct := pick(s.punctuation, s.text.Mix(s.muted, 0.35))

	ct.Tokens[editor.TokenText] = editor.TokenStyle{Color: s.text}
	ct.Tokens[editor.TokenKeyword] = editor.TokenStyle{Color: s.keyword, Bold: !s.noBoldKeyword}
	ct.Tokens[editor.TokenType] = editor.TokenStyle{Color: s.typ, Italic: s.italicType}
	ct.Tokens[editor.TokenFunction] = editor.TokenStyle{Color: s.function}
	ct.Tokens[editor.TokenVariable] = editor.TokenStyle{Color: s.text}
	ct.Tokens[editor.TokenProperty] = editor.TokenStyle{Color: prop}
	ct.Tokens[editor.TokenConstant] = editor.TokenStyle{Color: s.number}
	ct.Tokens[editor.TokenNumber] = editor.TokenStyle{Color: s.number}
	ct.Tokens[editor.TokenString] = editor.TokenStyle{Color: s.str}
	ct.Tokens[editor.TokenEscape] = editor.TokenStyle{Color: esc}
	ct.Tokens[editor.TokenRegexp] = editor.TokenStyle{Color: rx}
	ct.Tokens[editor.TokenComment] = editor.TokenStyle{Color: s.comment, Italic: true}
	ct.Tokens[editor.TokenOperator] = editor.TokenStyle{Color: s.operator}
	ct.Tokens[editor.TokenPunctuation] = editor.TokenStyle{Color: punct}
	ct.Tokens[editor.TokenTag] = editor.TokenStyle{Color: s.tag}
	ct.Tokens[editor.TokenAttribute] = editor.TokenStyle{Color: attr}
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
