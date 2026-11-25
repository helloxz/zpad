package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helloxz/zpad/locales"
)

// diffModal draws the text comparison panel over the window: two pickers
// for tabs already open, Compare, and MyGo's diff viewer below them. The
// viewer owns the diff, its folding and its word highlights; the panel adds
// the choice, the guards and the summary.
func (a *App) diffModal(c *ui.Context, t *ui.Theme) {
	if !a.diffOpen {
		// The dialog is gone: the snapshot, its buffers, the chosen sides and
		// the viewer's state go with it.
		a.diffA, a.diffB = nil, nil
		a.diff = nil
		a.diffIdentical = false
		a.diffFocus = false
		return
	}
	width, height := c.Size()
	panelW := min(float32(1100), width-40)
	if panelW < 520 {
		panelW = max(width-24, 360)
	}
	panelH := min(float32(780), height*0.86)
	if panelH < 320 {
		panelH = max(height-80, 300)
	}
	ui.DialogBase(c, &a.diffOpen, func(backdrop, panel ui.Element) {
		c.SetTheme(t)
		backdrop.Background(ui.RGBA(0, 0, 0, 0.35))
		panel.Width(panelW).Height(panelH).Padding(14).Gap(10).
			Background(t.Background).Border(1, t.Border).Radius(14).
			Shadow(0, 16, 48, 0, ui.RGBA(0, 0, 0, 0.28))
		panel.Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Text(c, locales.CompareText()).FontSize(14).Bold()
				ui.Spacer(c)
				ui.Text(c, locales.CompareBaseHint()).FontSize(11).
					TextColor(t.TextMuted).SingleLine()
			})
			a.diffPickers(c, t)
			a.diffBody(c, t)
			a.diffFooter(c, t)
		})
	})
}

// diffPickers is the row of A, the swap, B and Compare. Either side can be
// the same tab, which the guards below refuse rather than compare a text
// with itself.
func (a *App) diffPickers(c *ui.Context, t *ui.Theme) {
	ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
		a.diffPicker(c, t, "A", &a.diffA)
		a.diffSwapButton(c, t)
		a.diffPicker(c, t, "B", &a.diffB)
		a.diffRunButton(c, t)
	})
}

// diffPicker is one side's drop-down: the chosen tab's name and a list of
// every tab open, once each. The full path stays in the tip, so that two
// files of the same name can still be told apart. A new choice forgets an
// old result.
func (a *App) diffPicker(c *ui.Context, t *ui.Theme, side string, selected **Document) {
	current := *selected
	parts := ui.SelectBase(c, selected)
	trigger := parts.Trigger
	trigger.Grow(1).MinWidth(0).Height(34).Padding(0, 10).Gap(8).
		Radius(8).Border(1, t.Border).Background(t.Surface).
		Cursor(ui.CursorPointer).Label(side)
	if trigger.Hovered() {
		trigger.Background(t.SurfaceHover)
	}
	if trigger.FocusWithin() {
		trigger.BorderColor(t.Focus)
	}
	trigger.Children(func() {
		ui.Box(c).Size(20, 20).Center().Radius(6).Background(t.SurfaceHover).
			Children(func() {
				ui.Text(c, side).FontSize(10).Bold().TextColor(t.TextMuted)
			})
		name, color := locales.CompareSelectTab(), t.TextMuted
		if current != nil {
			name, color = current.Name(), t.Text
		}
		ui.Text(c, name).Grow(1).MinWidth(0).FontSize(12).Bold().
			TextColor(color).SingleLine().TruncateMiddle()
		ui.Icon(c, iconChevron).FontSize(10).TextColor(t.TextMuted).Rotate(90)
	})
	if current != nil && current.Path != "" {
		trigger.Tooltip(current.Path)
	}
	parts.Popup(func(panel ui.Element) {
		panel.Margin(t.Space(1), 0, 0, 0).Padding(0).
			Radius(t.Radius+2).Background(t.Background).Border(1, t.Border).
			Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.18))
		ui.Scroll(c).MaxHeight(selectPopupHeight).FillWidth().
			ScrollbarAxes(false, true).Children(func() {
			ui.Column(c).FillWidth().Gap(2).Padding(t.Space(1)).Children(func() {
				docs := a.openDocuments()
				if len(docs) == 0 {
					ui.Text(c, locales.CompareNeedTwoTabs()).FontSize(11).
						TextColor(t.TextMuted).Padding(t.Space(2), t.Space(2.5))
					return
				}
				for _, doc := range docs {
					item := parts.Item(doc).Height(30).Padding(0, t.Space(2.5)).
						Radius(t.Radius).Cursor(ui.CursorPointer)
					switch {
					case item.Highlighted():
						item.Background(t.Accent)
					case doc == current:
						item.Background(t.Surface)
					}
					item.Children(func() {
						color := t.Text
						if item.Highlighted() {
							color = t.AccentText
						}
						ui.Text(c, doc.Name()).Grow(1).MinWidth(0).FontSize(12).
							TextColor(color).SingleLine().TruncateMiddle()
					})
					if doc.Path != "" {
						item.Tooltip(doc.Path)
					}
				}
			})
		})
	})
	if trigger.Changed() {
		a.diffChanged()
	}
}

func (a *App) diffSwapButton(c *ui.Context, t *ui.Theme) {
	b := ui.ButtonBase(c).Size(34, 34).Center().Radius(7).
		Label(locales.CompareSwap()).Tooltip(locales.CompareSwap()).
		Background(t.Surface).Border(1, t.Border).Cursor(ui.CursorPointer)
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		ui.Text(c, "⇄").FontSize(14).TextColor(t.TextMuted)
	})
	if b.Clicked() {
		a.swapCompare()
	}
}

func (a *App) diffRunButton(c *ui.Context, t *ui.Theme) {
	label := locales.CompareRun()
	if a.diff != nil || a.diffIdentical {
		label = locales.CompareAgain()
	}
	b := ui.PrimaryButton(c, label).Height(34).Cursor(ui.CursorPointer)
	if a.diffProblem() != "" {
		b.Disabled(true)
	}
	if b.Clicked() {
		a.runCompare()
	}
}

// diffBody shows the viewer, or what the user has to do to get one. The
// viewer is built before the footer reads its stats: DiffView is what gives
// the new buffer a window to watch, and only then do large texts compare on
// a goroutine instead of the main thread.
func (a *App) diffBody(c *ui.Context, t *ui.Theme) {
	if a.diff != nil {
		// Kept in step with the settings and the system appearance, which
		// may change while the panel is over the window.
		a.diff.Theme = a.themeDefinition().Code
		a.diff.Font = ui.Font{Family: a.Settings.FontFamily, Size: a.Settings.FontSize}
		a.diff.LineHeight = a.Settings.LineHeight
		a.diff.TabSize = a.Settings.TabSize
		a.diff.Wrap = a.Settings.SoftWrap
		ui.Box(c).Grow(1).FillWidth().MinHeight(0).
			Border(1, t.Border).Radius(8).Clip().Children(func() {
			editor.DiffView(c, a.diff).Grow(1).FillWidth()
		})
		if a.diffFocus {
			a.diff.Focus()
			a.diffFocus = false
		}
		return
	}

	msg, color := locales.CompareReady(), t.TextMuted
	switch {
	case a.diffIdentical:
		msg, color = locales.CompareIdentical(a.diffLineCount()), t.Text
	case a.diffProblem() != "":
		msg = a.diffProblem()
	}
	ui.Column(c).Grow(1).FillWidth().Center().Padding(24).Children(func() {
		ui.Text(c, msg).FontSize(12).TextColor(color).
			TextAlign(ui.Center).MaxWidth(560)
	})
}

// diffFooter says what the comparison found and moves between its changes.
// The stats wait until the viewer has compared: while a large pair is on
// the goroutine, it says so instead of showing a false "no differences".
func (a *App) diffFooter(c *ui.Context, t *ui.Theme) {
	d := a.diff
	ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
		if d != nil {
			if d.Comparing() {
				ui.Text(c, locales.CompareComparing()).FontSize(11).
					TextColor(t.TextMuted).SingleLine()
			} else {
				added, removed := d.Stats()
				hunks := d.Hunks()
				if len(hunks) > 0 {
					ui.Text(c, locales.CompareStats(added, removed)).FontSize(11).
						TextColor(t.TextMuted).SingleLine()
					ui.Text(c, "·").FontSize(11).TextColor(t.TextMuted)
					ui.Text(c, locales.CompareHunks(len(hunks))).FontSize(11).
						TextColor(t.TextMuted).SingleLine()
					i, _ := d.CurrentHunk()
					a.diffHunkButton(c, t, "↑", locales.ComparePreviousHunk(),
						i <= 0, func() {
							d.GoToHunk(i - 1)
							d.Focus()
							a.update()
						})
					a.diffHunkButton(c, t, "↓", locales.CompareNextHunk(),
						i+1 >= len(hunks), func() {
							d.GoToHunk(i + 1)
							d.Focus()
							a.update()
						})
				}
			}
		}
		ui.Spacer(c)
		ui.Text(c, locales.CompareFooter(locales.CompareLimit())).FontSize(10).
			TextColor(t.TextMuted).SingleLine()
	})
}

func (a *App) diffHunkButton(c *ui.Context, t *ui.Theme, text, label string, disabled bool, run func()) {
	b := ui.ButtonBase(c).Size(22, 22).Center().Radius(5).
		Label(label).Tooltip(label)
	if disabled {
		b.Disabled(true).DimDisabled(true)
	} else {
		b.Cursor(ui.CursorPointer)
		if b.Hovered() {
			b.Background(t.SurfaceHover)
		}
	}
	b.Children(func() {
		ui.Text(c, text).FontSize(11).TextColor(t.TextMuted)
	})
	if !disabled && b.Clicked() {
		run()
	}
}
