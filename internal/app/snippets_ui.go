package app

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/helloxz/zpad/internal/snippets"
	"github.com/helloxz/zpad/internal/theme"
	"github.com/helloxz/zpad/locales"
)

// The panel's column headers: both use this height, so that the two boxes
// below them start on the same line whatever an action becomes.
const snippetHeaderHeight = 26

// snippetGroupsWidth is the width of the panel's group column: enough for a
// language name and a count, not enough to crowd the snippets beside it.
const snippetGroupsWidth = 160

// snippetsModal draws the snippet panel over the window: groups on the left,
// the snippets of the one chosen on the right. A snippet is added, edited or
// removed from here; only its name and text open a dialog of their own.
func (a *App) snippetsModal(c *ui.Context, t *ui.Theme) {
	if !a.snippetsOpen || a.snippetEditOpen || a.groupEditOpen {
		return
	}
	width, height := c.Size()
	panelW := min(float32(720), width-40)
	if panelW < 360 {
		panelW = max(width-24, 300)
	}
	panelH := min(float32(600), height*0.78)
	if panelH < 240 {
		panelH = max(height-80, 220)
	}
	ui.DialogBase(c, &a.snippetsOpen, func(backdrop, panel ui.Element) {
		c.SetTheme(t)
		backdrop.Background(ui.RGBA(0, 0, 0, 0.35))
		panel.Width(panelW).Height(panelH).Padding(14).Gap(10).
			Background(t.Background).Border(1, t.Border).Radius(14).
			Shadow(0, 16, 48, 0, ui.RGBA(0, 0, 0, 0.28))
		panel.Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, locales.Snippets()).FontSize(14).Bold()
			})
			a.snippetSearchField(c, t)
			ui.Row(c).Grow(1).FillWidth().Gap(10).MinHeight(0).Children(func() {
				a.snippetGroupsColumn(c, t)
				a.snippetListColumn(c, t)
			})
			a.snippetFooter(c, t)
		})
	})
}

func (a *App) snippetSearchField(c *ui.Context, t *ui.Theme) {
	field := ui.Row(c).Height(32).FillWidth().AlignItems(ui.Center).Gap(4).
		Padding(0, 10).Radius(6).Background(t.Surface).Border(1, t.Border)
	var in ui.Element
	field.Children(func() {
		in = ui.TextInputBase(c, &a.snippetsQuery).Grow(1).
			Label(locales.SnippetSearchLabel()).
			Placeholder(locales.SnippetSearchLabel())
		if a.snippetsFocus {
			in.Focus()
			a.snippetsFocus = false
		}
		if in.Changed() {
			a.refreshSnippetRows()
			a.update()
		}
		if in.Submitted() {
			a.insertSelectedSnippet()
		}
		if in.Shortcut(0, ui.KeyDown) {
			a.moveSnippetSelection(1)
		}
		if in.Shortcut(0, ui.KeyUp) {
			a.moveSnippetSelection(-1)
		}
		if a.snippetsQuery != "" {
			b := ui.ButtonBase(c).Size(18, 18).Center().Radius(9).
				Label(locales.SearchClear()).Cursor(ui.CursorPointer).KeepFocus()
			if b.Hovered() {
				b.Background(t.SurfaceHover)
			}
			b.Children(func() {
				ui.Icon(c, iconClose).FontSize(12).TextColor(t.TextMuted)
			})
			if b.Clicked() {
				a.snippetsQuery = ""
				a.snippetsFocus = true
				a.refreshSnippetRows()
				a.update()
			}
			tooltipBelow(c, t, b, locales.SearchClear())
		}
	})
	if in.FocusWithin() {
		field.BorderColor(t.Focus)
	}
	if field.Clicked() {
		in.Focus()
	}
}

// snippetGroupsColumn lists "all" and every group, with how many snippets
// the query matches in each, and the buttons that add and organise them.
func (a *App) snippetGroupsColumn(c *ui.Context, t *ui.Theme) {
	chosen, hasGroup := a.groupAt(a.snippetsGroup)
	ui.Column(c).Width(snippetGroupsWidth).Shrink(0).FillHeight().Gap(6).Children(func() {
		ui.Row(c).Height(snippetHeaderHeight).AlignItems(ui.Center).Gap(2).Children(func() {
			ui.Text(c, locales.SnippetGroups()).FontSize(11).Bold().TextColor(t.TextMuted)
			ui.Spacer(c)
			a.snippetIconAction(c, t, iconEdit, locales.SnippetRenameGroup(), !hasGroup, func() {
				a.openGroupDialog(chosen.ID)
			})
			a.snippetIconAction(c, t, iconTrash, locales.SnippetDeleteGroup(), !hasGroup, func() {
				a.confirmDeleteGroup(chosen.ID)
			})
			a.snippetIconAction(c, t, iconPlus, locales.SnippetNewGroup(), false, func() {
				a.openGroupDialog("")
			})
		})
		ui.Scroll(c).Grow(1).FillWidth().MinHeight(0).
			Background(t.Surface).Border(1, t.Border).Radius(8).Padding(4).
			Children(func() {
				var groups []snippets.Group
				if a.snippets != nil {
					groups = a.snippets.Groups()
				}
				ui.Column(c).FillWidth().Gap(1).Children(func() {
					a.snippetGroupRow(c, t, -1, locales.SnippetAllGroups(), a.snippetMatchTotal())
					for i, g := range groups {
						a.snippetGroupRow(c, t, i, g.Name, a.snippetMatchCount(i))
					}
				})
			})
	})
}

// snippetGroupRow is one group: a click shows its snippets. Index -1 is the
// "all" row, which renames and deletes nothing.
func (a *App) snippetGroupRow(c *ui.Context, t *ui.Theme, index int, name string, count int) {
	active := index == a.snippetsGroup
	dim := !active && a.snippetsQuery != "" && count == 0
	b := ui.ButtonBase(c).FillWidth().Height(26).Padding(0, 6).Gap(8).
		Radius(5).Cursor(ui.CursorPointer).Label(name)
	if active {
		b.Background(t.Accent)
	} else if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	color := t.Text
	switch {
	case active:
		color = t.AccentText
	case dim:
		color = t.TextMuted
	}
	b.Children(func() {
		ui.Text(c, name).FontSize(12).SingleLine().TruncateMiddle().TextColor(color)
		ui.Spacer(c)
		ui.Text(c, fmt.Sprintf("%d", count)).FontSize(10).TextColor(color)
	})
	if b.Clicked() {
		a.snippetsGroup = index
		a.refreshSnippetRows()
		// The click leaves the keyboard on the group: hand it back to the
		// field, so that ↑↓ and ↵ work on the new list at once.
		a.snippetsFocus = true
		a.update()
	}
}

// snippetListColumn shows the snippets of the chosen group, or of all of
// them, with the button that adds one.
func (a *App) snippetListColumn(c *ui.Context, t *ui.Theme) {
	title := locales.SnippetAllSnippets()
	if g, ok := a.groupAt(a.snippetsGroup); ok {
		title = g.Name
	}
	ui.Column(c).Grow(1).FillHeight().MinHeight(0).Gap(6).Children(func() {
		ui.Row(c).Height(snippetHeaderHeight).AlignItems(ui.Center).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(11).Bold().TextColor(t.TextMuted).
				SingleLine().TruncateMiddle()
			ui.Spacer(c)
			a.snippetIconButton(c, t, iconPlus, locales.SnippetNewSnippet(), func() {
				a.newSnippet()
			})
		})
		a.snippetList(c, t)
	})
}

func (a *App) snippetList(c *ui.Context, t *ui.Theme) {
	box := ui.Column(c).Grow(1).FillWidth().MinHeight(0).
		Background(t.Surface).Border(1, t.Border).Radius(8).Clip()
	if len(a.snippetsRows) == 0 {
		// The empty state keeps the panel's theme: inside the list's, a
		// primary button would take its faint accent and read as disabled.
		box.Children(func() { a.snippetEmptyState(c, t) })
		return
	}
	var list ui.Element
	box.Children(func() {
		ui.Themed(c, theme.ResultList(t)).Grow(1).FillWidth().Padding(4).Children(func() {
			a.snippetsList.Selected = &a.snippetsSel
			a.snippetsList.Label = func(i int) string {
				if sn, _, ok := a.snippetAt(i); ok {
					return sn.Name
				}
				return ""
			}
			a.snippetsList.Key = func(i int) any {
				if sn, _, ok := a.snippetAt(i); ok {
					return "snippet:" + sn.ID
				}
				return fmt.Sprintf("row:%d", i)
			}
			list = ui.List(c, &a.snippetsList, len(a.snippetsRows), func(i int) {
				a.snippetRowView(c, t, i)
			}).Grow(1)
		})
	})
	if list.Submitted() {
		a.insertSelectedSnippet()
	}
}

func (a *App) snippetRowView(c *ui.Context, t *ui.Theme, i int) {
	sn, gi, ok := a.snippetAt(i)
	if !ok {
		return
	}
	ui.Row(c).FillWidth().Height(30).Gap(2).Children(func() {
		b := ui.ButtonBase(c).Grow(1).Height(28).Padding(0, 6).Gap(8).
			Radius(5).Cursor(ui.CursorPointer).Label(sn.Name)
		if b.Hovered() {
			b.Background(t.SurfaceHover)
		}
		b.Children(func() {
			ui.Text(c, sn.Name).FontSize(12).SingleLine().TruncateMiddle().MaxWidth(220)
			ui.Text(c, snippetPreview(sn.Body)).FontSize(11).TextColor(t.TextMuted).
				SingleLine().Grow(1)
			// In the "all" list a row has to say which group it is from.
			if a.snippetsGroup == -1 {
				ui.Text(c, a.groupNameAt(gi)).FontSize(10).TextColor(t.TextMuted).
					SingleLine().Shrink(0)
			}
		})
		if b.Clicked() {
			a.insertSnippet(sn)
		}
		a.snippetIconButton(c, t, iconEdit, locales.SnippetEditTitle(), func() {
			a.openSnippetDialog(a.groupIDAt(gi), sn.ID)
		})
		a.snippetIconButton(c, t, iconTrash, locales.Delete(), func() {
			a.confirmDeleteSnippet(a.groupIDAt(gi), sn.ID)
		})
	})
}

// snippetEmptyState says why the list is empty, and offers the step that
// comes next: adding a snippet, or looking in the groups the query matched
// elsewhere.
func (a *App) snippetEmptyState(c *ui.Context, t *ui.Theme) {
	msg := locales.SnippetNone()
	if a.snippetsQuery != "" {
		msg = locales.SnippetNoMatch()
	}
	others := 0
	if a.snippetsQuery != "" && a.snippetsGroup != -1 {
		others = a.snippetMatchTotal() - a.snippetMatchCount(a.snippetsGroup)
	}
	ui.Column(c).Grow(1).Center().Gap(8).Padding(16).Children(func() {
		ui.Text(c, msg).FontSize(12).TextColor(t.TextMuted)
		if others > 0 {
			ui.Text(c, locales.SnippetOtherGroups(others)).FontSize(11).
				TextColor(t.TextMuted)
			if ui.Button(c, locales.SnippetShowAll()).Cursor(ui.CursorPointer).Clicked() {
				a.snippetsGroup = -1
				a.refreshSnippetRows()
				a.update()
			}
			return
		}
		if a.snippetsQuery != "" {
			return
		}
		ui.Text(c, locales.SnippetNoneHint()).FontSize(11).TextColor(t.TextMuted)
		if ui.PrimaryButton(c, locales.SnippetNewSnippet()).Cursor(ui.CursorPointer).Clicked() {
			a.newSnippet()
		}
	})
}

func (a *App) snippetFooter(c *ui.Context, t *ui.Theme) {
	ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
		if a.Settings.VimMode {
			ui.Text(c, locales.SnippetVimHint()).FontSize(11).TextColor(t.Warning)
		} else {
			ui.Text(c, locales.SnippetPaletteHint()).FontSize(11).TextColor(t.TextMuted)
		}
		ui.Spacer(c)
		if a.snippets != nil {
			ui.Text(c, locales.SnippetCount(a.snippets.Count())).FontSize(11).
				TextColor(t.TextMuted)
		}
	})
}

// snippetIconButton is one of the small buttons the panel's rows carry.
func (a *App) snippetIconButton(c *ui.Context, t *ui.Theme, icon *ui.SVG, label string, run func()) ui.Element {
	return a.snippetIconAction(c, t, icon, label, false, run)
}

// snippetIconAction is snippetIconButton for an action that may not apply:
// a header's rename, say, while "all" is on view, which dims rather than
// disappearing and keeps the header the same width.
func (a *App) snippetIconAction(c *ui.Context, t *ui.Theme, icon *ui.SVG, label string, disabled bool, run func()) ui.Element {
	b := ui.ButtonBase(c).Size(22, 22).Center().Radius(5).Label(label).Tooltip(label)
	if disabled {
		b.Disabled(true).DimDisabled(true)
	} else {
		b.Cursor(ui.CursorPointer)
		if b.Hovered() {
			b.Background(t.SurfaceHover)
		}
	}
	b.Children(func() {
		ui.Icon(c, icon).FontSize(13).TextColor(t.TextMuted)
	})
	if !disabled && b.Clicked() {
		run()
	}
	return b
}

// snippetPreview is the first line of a snippet's text, for the row that
// lists it.
func snippetPreview(body string) string {
	line := body
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	line = strings.TrimSpace(line)
	line = strings.ReplaceAll(line, "\t", "    ")
	runes := []rune(line)
	if len(runes) > 120 {
		return string(runes[:120]) + "…"
	}
	return line
}

// snippetEditModal asks for a snippet's name and text.
func (a *App) snippetEditModal(c *ui.Context, t *ui.Theme) {
	if !a.snippetEditOpen {
		return
	}
	width, height := c.Size()
	panelW := min(float32(480), width-40)
	if panelW < 280 {
		panelW = max(width-24, 240)
	}
	panelH := min(float32(390), height-60)
	ui.DialogBase(c, &a.snippetEditOpen, func(backdrop, panel ui.Element) {
		c.SetTheme(t)
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4))
		panel.Width(panelW).Height(panelH).Padding(20).Gap(10).
			Background(t.Background).Border(1, t.Border).Radius(14).
			Shadow(0, 16, 48, 0, ui.RGBA(0, 0, 0, 0.28))
		panel.Children(func() {
			title := locales.SnippetNewSnippet()
			if a.snippetEditID != "" {
				title = locales.SnippetEditTitle()
			}
			ui.Text(c, title).FontSize(16).Bold()
			if a.snippets != nil {
				if g, ok := a.snippets.Group(a.snippetEditGroupID); ok {
					ui.Text(c, locales.SnippetGroupOf(g.Name)).FontSize(11).
						TextColor(t.TextMuted).SingleLine().TruncateMiddle()
				}
			}
			in := ui.TextInput(c, &a.snippetEditName).FillWidth().
				Label(locales.SnippetNameLabel()).
				Placeholder(locales.SnippetNamePlaceholder())
			if a.snippetEditFocus {
				in.AutoFocus()
				a.snippetEditFocus = false
			}
			submit := in.Submitted()
			ui.Text(c, locales.SnippetBodyLabel()).FontSize(11).TextColor(t.TextMuted)
			ui.TextArea(c, &a.snippetEditBody).FillWidth().Grow(1).MinHeight(120).
				Padding(8).Radius(6).Background(t.Surface).Border(1, t.Border).
				Label(locales.SnippetBodyLabel()).
				Placeholder(locales.SnippetBodyPlaceholder())
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				counts := locales.SnippetNameLabel() + " " +
					fmt.Sprintf("%d / %d", utf8.RuneCountInString(a.snippetEditName),
						snippets.MaxNameRunes) + "  ·  " +
					locales.SnippetBodyLabel() + " " +
					formatSize(int64(len(a.snippetEditBody))) + " / " +
					formatSize(snippets.MaxBodyBytes)
				ui.Text(c, counts).FontSize(11).TextColor(t.TextMuted)
				ui.Spacer(c)
				if ui.Button(c, locales.Cancel()).Cursor(ui.CursorPointer).Clicked() {
					a.snippetEditOpen = false
				}
				if ui.PrimaryButton(c, locales.Save()).Cursor(ui.CursorPointer).Clicked() {
					submit = true
				}
			})
			if a.snippetEditErr != "" {
				ui.Text(c, a.snippetEditErr).FontSize(11).TextColor(t.Danger).SingleLine()
			}
			if submit {
				a.applySnippetEdit()
			}
		})
	})
}

// groupEditModal asks for a group's name.
func (a *App) groupEditModal(c *ui.Context, t *ui.Theme) {
	if !a.groupEditOpen {
		return
	}
	title := locales.SnippetNewGroup()
	if a.groupEditID != "" {
		title = locales.SnippetRenameGroup()
	}
	ui.DialogBase(c, &a.groupEditOpen, func(backdrop, panel ui.Element) {
		c.SetTheme(t)
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4))
		panel.Width(420).Padding(20).Gap(12).
			Background(t.Background).Border(1, t.Border).Radius(14).
			Shadow(0, 16, 48, 0, ui.RGBA(0, 0, 0, 0.28))
		panel.Children(func() {
			ui.Text(c, title).FontSize(16).Bold()
			in := ui.TextInput(c, &a.groupEditName).FillWidth().
				Label(locales.SnippetNameLabel()).
				Placeholder(locales.SnippetGroupNamePlaceholder())
			if a.groupEditFocus {
				in.AutoFocus()
				a.groupEditFocus = false
			}
			submit := in.Submitted()
			if a.groupEditErr != "" {
				ui.Text(c, a.groupEditErr).FontSize(11).TextColor(t.Danger).SingleLine()
			} else {
				ui.Text(c, fmt.Sprintf("%d / %d", utf8.RuneCountInString(a.groupEditName),
					snippets.MaxNameRunes)).FontSize(11).TextColor(t.TextMuted)
			}
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Spacer(c)
				if ui.Button(c, locales.Cancel()).Cursor(ui.CursorPointer).Clicked() {
					a.groupEditOpen = false
				}
				if ui.PrimaryButton(c, locales.Save()).Cursor(ui.CursorPointer).Clicked() {
					submit = true
				}
			})
			if submit {
				a.applyGroupEdit()
			}
		})
	})
}
