package app

import (
	"sort"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helloxz/zpad/internal/encoding"
	"github.com/helloxz/zpad/internal/settings"
	"github.com/helloxz/zpad/internal/theme"
	"github.com/helloxz/zpad/locales"
)

func (a *App) openSettings() {
	a.renameOpen = false
	a.aboutOpen = false
	a.fontSize = float64(a.Settings.FontSize)
	a.lineHeight = float64(a.Settings.LineHeight)
	a.showSettings = true
	a.update()
}

func (a *App) fontNames() []string {
	if a.fontFamilies != nil {
		return a.fontFamilies
	}
	var names []string
	for _, f := range ui.FontFamilies() {
		if f.Monospaced {
			names = append(names, f.Name)
		}
	}
	sort.SliceStable(names, func(i, j int) bool {
		if names[i] == "JetBrains Mono" {
			return true
		}
		if names[j] == "JetBrains Mono" {
			return false
		}
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})
	a.fontFamilies = names
	return names
}

func (a *App) settingsModal(c *ui.Context, t *ui.Theme) {
	if !a.showSettings {
		return
	}
	ui.DialogBase(c, &a.showSettings, func(backdrop, panel ui.Element) {
		c.SetTheme(t)
		w, h := c.Size()
		cw := min(float32(700), w-64)
		if cw < 320 {
			cw = w - 32
		}
		ch := min(float32(640), h-64)
		if ch < 320 {
			ch = h - 32
		}
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4))
		panel.Width(cw).Height(ch).Padding(24).Gap(16).
			Background(t.Background).Border(1, t.Border).Radius(14).
			Shadow(0, 16, 48, 0, ui.RGBA(0, 0, 0, 0.28))
		panel.Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, locales.SettingsTitle()).FontSize(18).Bold()
			})
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Gap(24).Children(func() {
					a.settingsAppearance(c, t)
					a.settingsEditor(c, t)
					a.settingsFiles(c, t)
					a.settingsWorkspace(c, t)
				})
			})
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				if ui.Button(c, locales.RestoreDefaults()).Cursor(ui.CursorPointer).Clicked() {
					go a.confirmRestoreDefaults()
				}
				ui.Spacer(c)
				if ui.Button(c, locales.Close()).Cursor(ui.CursorPointer).Clicked() {
					a.showSettings = false
				}
				if ui.PrimaryButton(c, locales.Done()).Cursor(ui.CursorPointer).Clicked() {
					a.showSettings = false
				}
			})
		})
	})
}

// settingsSection draws a group's heading and the card of its rows.
func (a *App) settingsSection(c *ui.Context, t *ui.Theme, icon *ui.SVG, title string, build func()) {
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Icon(c, icon).FontSize(17).TextColor(t.Accent)
			ui.Text(c, title).FontSize(15).Bold()
		})
		ui.Column(c).FillWidth().Clip().Background(t.Surface).
			Border(1, t.Border).Radius(12).Children(build)
	})
}

// settingRow puts a setting's name and note on the left and its control on
// the right, or the control below them when the window is narrow. Rows are
// divided by a hairline, none after the last.
func (a *App) settingRow(c *ui.Context, t *ui.Theme, title, description string, last bool, control func()) {
	narrow := settingsNarrow(c)
	var row ui.Element
	if narrow {
		row = ui.Column(c).FillWidth().Padding(t.Space(3.5), t.Space(4)).
			Gap(t.Space(2.5))
	} else {
		row = ui.Row(c).FillWidth().Padding(t.Space(4), t.Space(4.5)).
			Gap(t.Space(4)).AlignItems(ui.Center)
	}
	if !last {
		row.BorderWidth(0, 0, 1, 0).BorderColor(t.Border)
	}
	row.Children(func() {
		label := ui.Column(c).Gap(2)
		if narrow {
			label.FillWidth()
		} else {
			label.Grow(1)
		}
		label.Children(func() {
			ui.Text(c, title).FontSize(13).Bold()
			if description != "" {
				ui.Text(c, description).FontSize(12).
					TextColor(t.TextMuted).LineHeight(1.4)
			}
		})
		if narrow {
			ui.Row(c).FillWidth().Children(control)
		} else {
			ui.Row(c).Width(240).Shrink(0).Justify(ui.End).Children(control)
		}
	})
}

func settingsNarrow(c *ui.Context) bool {
	w, _ := c.Size()
	return w < 620
}

func (a *App) settingsAppearance(c *ui.Context, t *ui.Theme) {
	a.settingsSection(c, t, iconPalette, locales.Appearance(), func() {
		a.settingRow(c, t, locales.Theme(), locales.ThemeDescription(), false, func() {
			a.selectSettingGroups(c, t, locales.Theme(), themeLabel(a.Settings.Theme),
				themeGroups(), func(label string) {
					a.setThemeID(themeIDFromLabel(label))
				}, themeOptionParts)
		})
		a.settingRow(c, t, locales.Language(), locales.LanguageDescription(), true, func() {
			a.selectSetting(c, t, locales.Language(), a.languageLabel(a.Settings.Language), languageLabels(), func(label string) {
				a.setLanguage(languageTag(label))
			})
		})
	})
}

// selectGroup is a titled block of options in a settings drop-down.
type selectGroup struct {
	label   string
	options []string
}

// themeGroups lists the themes as the default trio first, then the named
// dark palettes, then the named light ones, each named palette carrying its
// light or dark tag on the right.
func themeGroups() []selectGroup {
	dark := make([]string, 0, 7)
	light := make([]string, 0, 7)
	for _, def := range theme.Named() {
		if def.Dark {
			dark = append(dark, themeOptionLabel(def))
		} else {
			light = append(light, themeOptionLabel(def))
		}
	}
	return []selectGroup{
		{
			label:   locales.ThemeGroupDefault(),
			options: []string{locales.ThemeSystem(), locales.ThemeLight(), locales.ThemeDark()},
		},
		{label: locales.ThemeGroupDark(), options: dark},
		{label: locales.ThemeGroupLight(), options: light},
	}
}

func themeOptionLabel(def *theme.Definition) string {
	tag := locales.ThemeLightTag()
	if def.Dark {
		tag = locales.ThemeDarkTag()
	}
	return def.Name + " · " + tag
}

func themeLabel(id string) string {
	switch id {
	case theme.SystemID:
		return locales.ThemeSystem()
	case theme.LightID:
		return locales.ThemeLight()
	case theme.DarkID:
		return locales.ThemeDark()
	}
	if def, ok := theme.Lookup(id); ok {
		return themeOptionLabel(def)
	}
	return locales.ThemeSystem()
}

func themeIDFromLabel(label string) string {
	switch label {
	case locales.ThemeSystem():
		return theme.SystemID
	case locales.ThemeLight():
		return theme.LightID
	case locales.ThemeDark():
		return theme.DarkID
	}
	for _, def := range theme.Named() {
		if themeOptionLabel(def) == label {
			return def.ID
		}
	}
	return theme.SystemID
}

func (a *App) settingsEditor(c *ui.Context, t *ui.Theme) {
	a.settingsSection(c, t, iconCode, locales.EditorSection(), func() {
		options := append([]string{locales.SystemMonospace()}, a.fontNames()...)
		current := a.Settings.FontFamily
		if current == "" {
			current = locales.SystemMonospace()
		}
		a.settingRow(c, t, locales.FontFamily(), locales.FontFamilyDescription(), false, func() {
			a.selectSetting(c, t, locales.FontFamily(), current, options, func(label string) {
				if label == locales.SystemMonospace() {
					a.Settings.FontFamily = ""
				} else {
					a.Settings.FontFamily = label
				}
			})
		})

		a.settingRow(c, t, locales.FontSize(), locales.FontSizeDescription(), false, func() {
			if ui.NumberInput(c, &a.fontSize, 10, 28, 1).Width(120).
				Label(locales.FontSize()).Changed() {
				a.Settings.FontSize = float32(a.fontSize)
				a.saveSettings()
				a.update()
			}
		})
		a.settingRow(c, t, locales.LineHeight(), locales.LineHeightDescription(), false, func() {
			if ui.NumberInput(c, &a.lineHeight, 1.2, 2.0, 0.1).Width(120).
				Label(locales.LineHeight()).Changed() {
				a.Settings.LineHeight = float32(a.lineHeight)
				a.saveSettings()
				a.update()
			}
		})
		a.switchSetting(c, t, locales.SoftWrap(), locales.SoftWrapDescription(),
			&a.Settings.SoftWrap, false)

		a.settingRow(c, t, locales.LineNumbers(), locales.LineNumbersDescription(), false, func() {
			a.selectSetting(c, t, locales.LineNumbers(), lineNumbersLabel(a.Settings.LineNumbers),
				[]string{locales.LineNumbersAbsolute(), locales.LineNumbersRelative(), locales.LineNumbersOff()},
				func(label string) {
					switch label {
					case locales.LineNumbersRelative():
						a.Settings.LineNumbers = "relative"
					case locales.LineNumbersOff():
						a.Settings.LineNumbers = "off"
					default:
						a.Settings.LineNumbers = "absolute"
					}
				})
		})
		a.settingRow(c, t, locales.TabSize(), locales.TabSizeDescription(), false, func() {
			a.selectSetting(c, t, locales.TabSize(), tabSizeLabel(a.Settings.TabSize),
				[]string{"2", "4", "8"}, func(label string) {
					switch label {
					case "2":
						a.Settings.TabSize = 2
					case "8":
						a.Settings.TabSize = 8
					default:
						a.Settings.TabSize = 4
					}
				})
		})
		a.settingRow(c, t, locales.Indent(), locales.IndentDescription(), false, func() {
			a.selectSetting(c, t, locales.Indent(), indentLabel(a.Settings.UseTabs),
				[]string{locales.IndentSpaces(), locales.IndentTabs()}, func(label string) {
					a.Settings.UseTabs = label == locales.IndentTabs()
				})
		})
		a.switchSetting(c, t, locales.IndentGuides(), locales.IndentGuidesDescription(),
			&a.Settings.ShowGuides, false)
		a.settingRow(c, t, locales.ShowWhitespace(), locales.ShowWhitespaceDescription(), false, func() {
			a.selectSetting(c, t, locales.ShowWhitespace(), whitespaceLabel(a.Settings.Whitespace),
				[]string{locales.WhitespaceSelection(), locales.WhitespaceNone(),
					locales.WhitespaceAll(), locales.WhitespaceTrailing()},
				func(label string) {
					switch label {
					case locales.WhitespaceNone():
						a.Settings.Whitespace = "none"
					case locales.WhitespaceAll():
						a.Settings.Whitespace = "all"
					case locales.WhitespaceTrailing():
						a.Settings.Whitespace = "trailing"
					default:
						a.Settings.Whitespace = "selection"
					}
				})
		})
		a.switchSetting(c, t, locales.AutoClose(), locales.AutoCloseDescription(),
			&a.Settings.AutoClose, false)
		a.switchSettingAction(c, t, locales.VimMode(), locales.VimModeDescription(),
			&a.Settings.VimMode, true, false, a.vimModeChanged)
	})
}

func (a *App) settingsFiles(c *ui.Context, t *ui.Theme) {
	a.settingsSection(c, t, iconFile, locales.FilesSection(), func() {
		a.settingRow(c, t, locales.DefaultEncoding(), locales.EncodingDescription(), false, func() {
			a.selectSetting(c, t, locales.DefaultEncoding(),
				encoding.Label(encoding.Parse(a.Settings.DefaultEncoding)),
				encodingLabels(), func(label string) {
					a.Settings.DefaultEncoding = string(encodingFromLabel(label))
				})
		})
		a.settingRow(c, t, locales.DefaultLineEnding(), locales.LineEndingDescription(), false, func() {
			a.selectSetting(c, t, locales.DefaultLineEnding(), eolLabel(a.Settings.DefaultEOL),
				[]string{"LF", "CRLF", "CR"}, func(label string) {
					switch label {
					case "CRLF":
						a.Settings.DefaultEOL = "\r\n"
					case "CR":
						a.Settings.DefaultEOL = "\r"
					default:
						a.Settings.DefaultEOL = "\n"
					}
				})
		})
		a.settingRow(c, t, locales.LargeFileThreshold(), locales.LargeFileDescription(), true, func() {
			a.selectSetting(c, t, locales.LargeFileThreshold(), largeFileLabel(a.Settings.LargeFileMB),
				[]string{"10 MB", "20 MB", "50 MB"}, func(label string) {
					switch label {
					case "10 MB":
						a.Settings.LargeFileMB = 10
					case "50 MB":
						a.Settings.LargeFileMB = 50
					default:
						a.Settings.LargeFileMB = 20
					}
				})
		})
	})
}

func (a *App) settingsWorkspace(c *ui.Context, t *ui.Theme) {
	a.settingsSection(c, t, iconFolder, locales.WorkspaceSection(), func() {
		a.switchSetting(c, t, locales.PreviewTabs(), locales.PreviewTabsDescription(),
			&a.Settings.PreviewTabs, false)
		a.switchSetting(c, t, locales.SidebarDefault(), locales.SidebarDefaultDescription(),
			&a.Settings.SidebarOpen, false)
		a.switchSetting(c, t, locales.RestoreSession(), locales.RestoreSessionDescription(),
			&a.Settings.RestoreSession, false)
		a.switchSettingAction(c, t, locales.KeepUnsaved(), locales.KeepUnsavedDescription(),
			&a.Settings.KeepUnsaved, false, false, a.keepUnsavedChanged)
		confirmDescription := locales.ConfirmCloseDescription()
		if a.Settings.KeepUnsaved {
			confirmDescription = locales.ConfirmCloseDisabledDescription()
		}
		a.switchSettingAction(c, t, locales.ConfirmClose(), confirmDescription,
			&a.Settings.ConfirmClose, true, a.Settings.KeepUnsaved, nil)
	})
}

// selectSetting draws a drop-down whose options show the pointing hand,
// which MyGo's Select does not style, and applies a new choice.
func (a *App) selectSetting(c *ui.Context, t *ui.Theme, label, current string, options []string, apply func(string), parts ...func(string) (string, string)) {
	a.selectSettingGroups(c, t, label, current, []selectGroup{{options: options}}, apply, parts...)
}

// selectSettingGroups is selectSetting with titled groups of options. The
// options scroll inside a popup no taller than selectPopupHeight, so a long
// list such as the themes' never covers the window.
func (a *App) selectSettingGroups(c *ui.Context, t *ui.Theme, label, current string, groups []selectGroup, apply func(string), parts ...func(string) (string, string)) {
	split := func(opt string) (left, right string) {
		if len(parts) > 0 && parts[0] != nil {
			return parts[0](opt)
		}
		return opt, ""
	}
	sel := ui.SelectBase(c, &current)
	trigger := sel.Trigger
	trigger.Width(240).Height(34).PaddingX(t.Space(3)).Gap(t.Space(2)).
		Radius(t.Radius).Border(1, t.Border).Background(t.Background).
		Cursor(ui.CursorPointer).Label(label)
	if trigger.Hovered() {
		trigger.Background(t.SurfaceHover)
	}
	trigger.Children(func() {
		left, right := split(current)
		ui.Text(c, left).FontSize(13).SingleLine()
		ui.Spacer(c)
		if right != "" {
			ui.Text(c, right).FontSize(13).TextColor(t.TextMuted).SingleLine()
		}
		ui.Icon(c, iconChevron).FontSize(12).TextColor(t.TextMuted).Rotate(90)
	})
	sel.Popup(func(panel ui.Element) {
		panel.Margin(t.Space(1), 0, 0, 0).Padding(0).
			Radius(t.Radius+2).Background(t.Background).Border(1, t.Border).
			Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.18))
		ui.Scroll(c).MaxHeight(selectPopupHeight).FillWidth().
			ScrollbarAxes(false, true).Children(func() {
			ui.Column(c).FillWidth().Gap(2).Padding(t.Space(1)).Children(func() {
				for _, group := range groups {
					if group.label != "" {
						ui.Text(c, group.label).FontSize(11).Bold().
							TextColor(t.TextMuted).
							Padding(t.Space(1.5), t.Space(2.5))
					}
					for _, opt := range group.options {
						item := sel.Item(opt).Padding(t.Space(2), t.Space(2.5)).
							Radius(t.Radius).Cursor(ui.CursorPointer)
						switch {
						case item.Highlighted():
							item.Background(t.Accent).TextColor(t.AccentText)
						case opt == current:
							item.Background(t.Surface)
						}
						item.Children(func() {
							left, right := split(opt)
							ui.Text(c, left).FontSize(13).SingleLine()
							ui.Spacer(c)
							if right != "" {
								color := t.TextMuted
								if item.Highlighted() {
									color = t.AccentText
								}
								ui.Text(c, right).FontSize(13).TextColor(color).SingleLine()
							}
						})
					}
				}
			})
		})
	})
	if trigger.Changed() {
		apply(current)
		a.saveSettings()
		a.update()
	}
}

// selectPopupHeight caps the settings drop-down; longer lists scroll.
const selectPopupHeight = 320

// themeOptionParts splits a theme option into its name and its light or
// dark tag, so the two line up on either side of the drop-down row.
func themeOptionParts(label string) (name, tag string) {
	if i := strings.LastIndex(label, " · "); i >= 0 {
		return label[:i], label[i+len(" · "):]
	}
	return label, ""
}

func (a *App) switchSetting(c *ui.Context, t *ui.Theme, title, description string, v *bool, last bool) {
	a.switchSettingAction(c, t, title, description, v, last, false, nil)
}

// switchSettingAction is switchSetting with a callback for when the switch
// changes, and a disabled state for a setting another one makes moot.
func (a *App) switchSettingAction(c *ui.Context, t *ui.Theme, title, description string, v *bool, last, disabled bool, onChange func()) {
	a.settingRow(c, t, title, description, last, func() {
		sw := ui.Switch(c, v).Label(title)
		if disabled {
			sw.Disabled(true).DimDisabled(true)
		} else {
			sw.Cursor(ui.CursorPointer)
		}
		if sw.Changed() && !disabled {
			if onChange != nil {
				onChange()
			}
			a.saveSettings()
			a.update()
		}
	})
}

func (a *App) setLanguage(tag string) {
	a.Settings.Language = tag
	a.saveSettings()
	mygo.App.SetLocale(tag)
	a.InstallMenu()
	a.updateTitle()
	a.update()
}

func (a *App) confirmRestoreDefaults() {
	res, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent:             a.win,
		Type:               mygo.MessageWarning,
		Message:            locales.RestoreDefaultsQuestion(),
		Buttons:            []string{locales.Cancel(), locales.Restore()},
		DefaultButton:      1,
		CancelButton:       0,
		DestructiveButtons: []int{1},
	})
	if err != nil || res.Button != 1 {
		return
	}
	a.onUI(func() { a.restoreDefaults() })
}

func (a *App) restoreDefaults() {
	a.Settings = settings.Default()
	a.keepUnsavedChanged()
	a.fontSize = float64(a.Settings.FontSize)
	a.lineHeight = float64(a.Settings.LineHeight)
	if a.tree != nil {
		a.tree.SetShowHidden(a.Settings.ShowHidden)
		a.treeGeneration++
	}
	if a.dirTree != nil {
		a.dirTree.SetShowHidden(a.Settings.ShowHidden)
		a.dirGeneration++
	}
	a.saveSettings()
	SetThemeSource(theme.SystemID)
	if a.running {
		a.Dark = mygo.Theme.IsDark()
	} else {
		a.Dark = false
	}
	mygo.App.SetLocale("")
	a.InstallMenu()
	a.updateTitle()
	a.SetToast(locales.SettingsReset())
	a.update()
}

func (a *App) languageLabel(tag string) string {
	switch tag {
	case "zh-Hans":
		return locales.LanguageChinese()
	case "en":
		return locales.LanguageEnglish()
	default:
		return locales.LanguageSystem()
	}
}

func languageLabels() []string {
	return []string{locales.LanguageSystem(), locales.LanguageChinese(), locales.LanguageEnglish()}
}

func languageTag(label string) string {
	switch label {
	case locales.LanguageChinese():
		return "zh-Hans"
	case locales.LanguageEnglish():
		return "en"
	default:
		return ""
	}
}

func lineNumbersLabel(v string) string {
	switch v {
	case "relative":
		return locales.LineNumbersRelative()
	case "off":
		return locales.LineNumbersOff()
	default:
		return locales.LineNumbersAbsolute()
	}
}

func tabSizeLabel(v int) string {
	switch v {
	case 2:
		return "2"
	case 8:
		return "8"
	default:
		return "4"
	}
}

func indentLabel(useTabs bool) string {
	if useTabs {
		return locales.IndentTabs()
	}
	return locales.IndentSpaces()
}

func whitespaceLabel(v string) string {
	switch v {
	case "none":
		return locales.WhitespaceNone()
	case "all":
		return locales.WhitespaceAll()
	case "trailing":
		return locales.WhitespaceTrailing()
	default:
		return locales.WhitespaceSelection()
	}
}

func encodingLabels() []string {
	labels := make([]string, 0, len(encoding.All))
	for _, kind := range encoding.All {
		labels = append(labels, encoding.Label(kind))
	}
	return labels
}

func encodingFromLabel(label string) encoding.Kind {
	for _, kind := range encoding.All {
		if encoding.Label(kind) == label {
			return kind
		}
	}
	return encoding.UTF8
}

func largeFileLabel(v int) string {
	switch v {
	case 10:
		return "10 MB"
	case 50:
		return "50 MB"
	default:
		return "20 MB"
	}
}
