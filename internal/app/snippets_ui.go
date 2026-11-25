package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/helloxz/zpad/internal/snippets"
	"github.com/helloxz/zpad/internal/theme"
	"github.com/helloxz/zpad/locales"
)

// snippetsModal draws the snippet panel over the window: the palette that
// inserts a snippet, or the management view. The editor dialogs sit over
// it, and the panel is not drawn then.
func (a *App) snippetsModal(c *ui.Context, t *ui.Theme) {
	if !a.snippetsOpen || a.snippetEditOpen || a.groupEditOpen {
		return
	}
	width, height := c.Size()
	panelW := min(float32(640), width-40)
	panelH := min(float32(540), height*0.72)
	if a.snippetsManage {
		panelW = min(float32(760), width-40)
		panelH = min(float32(600), height*0.78)
	}
	if panelW < 280 {
		panelW = max(width-24, 240)
	}
	if panelH < 220 {
		panelH = max(height-80, 200)
	}
	ui.DialogBase(c, &a.snippetsOpen, func(backdrop, panel ui.Element) {
		c.SetTheme(t)
		backdrop.Background(ui.RGBA(0, 0, 0, 0.35))
		panel.Width(panelW).Height(panelH).Padding(14).Gap(10).
			Background(t.Background).Border(1, t.Border).Radius(14).
			Shadow(0, 16, 48, 0, ui.RGBA(0, 0, 0, 0.28))
		panel.Children(func() {
			if a.snippetsManage {
				a.snippetManager(c, t)
				return
			}
			a.snippetPalette(c, t)
		})
	})
}

// snippetPalette is the panel that inserts a snippet: a field to filter
// with, the groups and their snippets, and how to use them.
func (a *App) snippetPalette(c *ui.Context, t *ui.Theme) {
	ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
		ui.Text(c, locales.Snippets()).FontSize(14).Bold()
		ui.Spacer(c)
		if a.canSaveSelection() {
			if ui.Button(c, locales.SnippetFromSelection()).Cursor(ui.CursorPointer).Clicked() {
				a.openSnippetFromSelection()
			}
		}
		if ui.Button(c, locales.ManageSnippets()).Cursor(ui.CursorPointer).Clicked() {
			a.snippetsManage = true
			a.snippetsQuery = ""
			a.refreshSnippetRows()
			a.update()
		}
	})
	a.snippetSearchField(c, t)
	a.snippetPaletteBody(c, t)
	a.snippetPaletteFooter(c, t)
}

// canSaveSelection reports whether the active tab has text selected that
// could become a snippet.
func (a *App) canSaveSelection() bool {
	tab := a.activeTab()
	if tab == nil || tab.preview {
		return false
	}
	for _, s := range tab.state.Selections() {
		if !s.Empty() {
			return true
		}
	}
	return false
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

func (a *App) snippetPaletteBody(c *ui.Context, t *ui.Theme) {
	if len(a.snippetsRows) == 0 {
		a.snippetEmptyState(c, t)
		return
	}

	a.snippetsList.Selected = &a.snippetsSel
	a.snippetsList.Header = func(i int) bool {
		return i >= 0 && i < len(a.snippetsRows) && a.snippetsRows[i].header
	}
	a.snippetsList.Label = func(i int) string {
		if i < 0 || i >= len(a.snippetsRows) {
			return ""
		}
		if sn, ok := a.rowSnippet(i); ok {
			return sn.Name
		}
		return ""
	}
	a.snippetsList.Key = func(i int) any {
		if i < 0 || i >= len(a.snippetsRows) {
			return i
		}
		row := a.snippetsRows[i]
		if row.header {
			return fmt.Sprintf("group:%d", row.group)
		}
		if sn, ok := a.rowSnippet(i); ok {
			return "snippet:" + sn.ID
		}
		return fmt.Sprintf("row:%d", i)
	}
	var list ui.Element
	ui.Themed(c, theme.ResultList(t)).Grow(1).MinHeight(0).Background(ui.Transparent).Children(func() {
		list = ui.List(c, &a.snippetsList, len(a.snippetsRows), func(i int) {
			a.snippetRowView(c, t, i)
		}).Grow(1)
	})
	if list.Submitted() {
		a.insertSelectedSnippet()
	}
}

// rowSnippet returns the snippet a row shows.
func (a *App) rowSnippet(i int) (snippets.Snippet, bool) {
	if a.snippets == nil || i < 0 || i >= len(a.snippetsRows) {
		return snippets.Snippet{}, false
	}
	row := a.snippetsRows[i]
	if row.header {
		return snippets.Snippet{}, false
	}
	groups := a.snippets.Groups()
	if row.group < 0 || row.group >= len(groups) {
		return snippets.Snippet{}, false
	}
	snips := groups[row.group].Snippets
	if row.snippet < 0 || row.snippet >= len(snips) {
		return snippets.Snippet{}, false
	}
	return snips[row.snippet], true
}

func (a *App) snippetRowView(c *ui.Context, t *ui.Theme, i int) {
	if a.snippets == nil || i < 0 || i >= len(a.snippetsRows) {
		return
	}
	row := a.snippetsRows[i]
	groups := a.snippets.Groups()
	if row.group < 0 || row.group >= len(groups) {
		return
	}
	g := groups[row.group]
	if row.header {
		h := ui.Row(c).FillWidth().Height(26).Padding(0, 8).Gap(6).
			AlignItems(ui.Center).Background(t.Surface)
		h.Children(func() {
			ui.Text(c, g.Name).FontSize(11).Bold().TextColor(t.TextMuted).SingleLine()
			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("%d", len(g.Snippets))).FontSize(11).TextColor(t.TextMuted)
		})
		return
	}
	sn, ok := a.rowSnippet(i)
	if !ok {
		return
	}
	e := ui.Row(c).FillWidth().Height(28).Padding(0, 8).Gap(10).
		AlignItems(ui.Center).Cursor(ui.CursorPointer)
	e.Children(func() {
		ui.Text(c, sn.Name).FontSize(12).SingleLine()
		ui.Text(c, snippetPreview(sn.Body)).FontSize(11).TextColor(t.TextMuted).
			SingleLine().Grow(1)
	})
	if e.RowClicked() {
		a.insertSnippet(sn)
	}
}

// snippetEmptyState says why the panel has nothing to list, and offers the
// step that comes next.
func (a *App) snippetEmptyState(c *ui.Context, t *ui.Theme) {
	msg := locales.SnippetNone()
	if a.snippetsQuery != "" {
		msg = locales.SnippetNoMatch()
	}
	ui.Column(c).Grow(1).Center().Gap(8).Padding(16).Children(func() {
		ui.Text(c, msg).FontSize(12).TextColor(t.TextMuted)
		if a.snippetsQuery != "" {
			return
		}
		ui.Text(c, locales.SnippetNoneHint()).FontSize(11).TextColor(t.TextMuted)
		ui.Row(c).Gap(6).Children(func() {
			if ui.Button(c, locales.SnippetNewGroup()).Cursor(ui.CursorPointer).Clicked() {
				a.openGroupDialog("")
			}
			has := a.snippets != nil && len(a.snippets.Groups()) > 0
			if ui.PrimaryButton(c, locales.SnippetNewSnippet()).Disabled(!has).
				Cursor(ui.CursorPointer).Clicked() {
				if g, _, ok := a.chosenGroup(); ok {
					a.openSnippetDialog(g.ID, "")
				}
			}
		})
	})
}

func (a *App) snippetPaletteFooter(c *ui.Context, t *ui.Theme) {
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

// snippetManager is the view that adds, renames and deletes: groups on the
// left, the snippets of the chosen one on the right.
func (a *App) snippetManager(c *ui.Context, t *ui.Theme) {
	ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
		ui.Text(c, locales.SnippetManagerTitle()).FontSize(14).Bold()
		ui.Spacer(c)
		if ui.PrimaryButton(c, locales.Done()).Cursor(ui.CursorPointer).Clicked() {
			a.snippetsManage = false
			a.snippetsQuery = ""
			a.snippetsFocus = true
			a.refreshSnippetRows()
			a.update()
		}
	})
	ui.Row(c).Grow(1).FillWidth().Gap(10).MinHeight(0).Children(func() {
		a.snippetGroupColumn(c, t)
		a.snippetColumn(c, t)
	})
}

func (a *App) snippetGroupColumn(c *ui.Context, t *ui.Theme) {
	ui.Column(c).Width(190).Shrink(0).FillHeight().Gap(6).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
			ui.Text(c, locales.SnippetGroups()).FontSize(11).Bold().TextColor(t.TextMuted)
			ui.Spacer(c)
			a.snippetIconButton(c, t, iconPlus, locales.SnippetNewGroup(), func() {
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
					for i, g := range groups {
						a.snippetGroupRow(c, t, i, g)
					}
				})
			})
		ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
			g, _, ok := a.chosenGroup()
			if ui.Button(c, locales.Rename()).Disabled(!ok).Cursor(ui.CursorPointer).Clicked() {
				a.openGroupDialog(g.ID)
			}
			if ui.Button(c, locales.Delete()).Disabled(!ok).Cursor(ui.CursorPointer).Clicked() {
				a.confirmDeleteGroup(g.ID)
			}
		})
	})
}

func (a *App) snippetGroupRow(c *ui.Context, t *ui.Theme, i int, g snippets.Group) {
	active := i == a.snippetsGroup
	b := ui.ButtonBase(c).FillWidth().Height(28).Padding(0, 8).Gap(6).
		Radius(6).Cursor(ui.CursorPointer)
	switch {
	case active:
		b.Background(t.Accent)
	case b.Hovered():
		b.Background(t.SurfaceHover)
	}
	color := t.TextMuted
	if active {
		color = t.AccentText
	}
	b.Children(func() {
		ui.Text(c, g.Name).FontSize(12).SingleLine().TextColor(color)
		ui.Spacer(c)
		ui.Text(c, fmt.Sprintf("%d", len(g.Snippets))).FontSize(10).TextColor(color)
	})
	if b.Clicked() {
		a.selectGroup(g.ID)
		a.update()
	}
}

func (a *App) snippetColumn(c *ui.Context, t *ui.Theme) {
	g, _, ok := a.chosenGroup()
	ui.Column(c).Grow(1).FillHeight().MinHeight(0).Gap(6).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
			name := locales.SnippetNoGroup()
			if ok {
				name = g.Name
			}
			ui.Text(c, name).FontSize(11).Bold().TextColor(t.TextMuted).SingleLine()
			ui.Spacer(c)
			a.snippetIconButton(c, t, iconPlus, locales.SnippetNewSnippet(), func() {
				a.openSnippetDialog(g.ID, "")
			})
		})
		ui.Scroll(c).Grow(1).FillWidth().MinHeight(0).
			Background(t.Surface).Border(1, t.Border).Radius(8).Padding(4).
			Children(func() {
				if !ok || len(g.Snippets) == 0 {
					ui.Column(c).FillWidth().Padding(24, 16).Center().Gap(4).Children(func() {
						ui.Text(c, locales.SnippetNone()).FontSize(12).TextColor(t.TextMuted)
						ui.Text(c, locales.SnippetNoneHint()).FontSize(11).TextColor(t.TextMuted)
					})
					return
				}
				ui.Column(c).FillWidth().Gap(1).Children(func() {
					for _, sn := range g.Snippets {
						a.snippetManagerRow(c, t, g.ID, sn)
					}
				})
			})
	})
}

func (a *App) snippetManagerRow(c *ui.Context, t *ui.Theme, groupID string, sn snippets.Snippet) {
	ui.Row(c).FillWidth().Height(30).Gap(4).Children(func() {
		b := ui.ButtonBase(c).Grow(1).Height(28).Padding(0, 6).Gap(8).
			Radius(5).Cursor(ui.CursorPointer).Label(sn.Name)
		if b.Hovered() {
			b.Background(t.SurfaceHover)
		}
		b.Children(func() {
			ui.Text(c, sn.Name).FontSize(12).SingleLine()
			ui.Text(c, snippetPreview(sn.Body)).FontSize(11).TextColor(t.TextMuted).
				SingleLine().Grow(1)
		})
		if b.Clicked() {
			a.openSnippetDialog(groupID, sn.ID)
		}
		a.snippetIconButton(c, t, iconEdit, locales.SnippetEditTitle(), func() {
			a.openSnippetDialog(groupID, sn.ID)
		})
		a.snippetIconButton(c, t, iconTrash, locales.Delete(), func() {
			a.confirmDeleteSnippet(groupID, sn.ID)
		})
	})
}

// snippetIconButton is one of the small buttons the manager's rows carry.
func (a *App) snippetIconButton(c *ui.Context, t *ui.Theme, icon *ui.SVG, label string, run func()) ui.Element {
	b := ui.ButtonBase(c).Size(22, 22).Center().Radius(5).Label(label).
		Tooltip(label).Cursor(ui.CursorPointer)
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		ui.Icon(c, icon).FontSize(13).TextColor(t.TextMuted)
	})
	if b.Clicked() {
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
					ui.Text(c, g.Name).FontSize(11).TextColor(t.TextMuted).SingleLine()
				}
			}
			in := ui.TextInput(c, &a.snippetEditName).FillWidth().
				Label(locales.SnippetNameLabel())
			if a.snippetEditFocus {
				in.AutoFocus()
				a.snippetEditFocus = false
			}
			submit := in.Submitted()
			ui.Text(c, locales.SnippetBodyLabel()).FontSize(11).TextColor(t.TextMuted)
			ui.TextArea(c, &a.snippetEditBody).FillWidth().Grow(1).MinHeight(120).
				Padding(8).Radius(6).Background(t.Surface).Border(1, t.Border).
				Label(locales.SnippetBodyLabel())
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				size := formatSize(int64(len(a.snippetEditBody))) + " / " +
					formatSize(snippets.MaxBodyBytes)
				ui.Text(c, size).FontSize(11).TextColor(t.TextMuted)
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
				Label(locales.SnippetNameLabel())
			if a.groupEditFocus {
				in.AutoFocus()
				a.groupEditFocus = false
			}
			if a.groupEditErr != "" {
				ui.Text(c, a.groupEditErr).FontSize(11).TextColor(t.Danger).SingleLine()
			}
			submit := in.Submitted()
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
