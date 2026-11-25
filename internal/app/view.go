package app

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helloxz/zpad/internal/encoding"
	"github.com/helloxz/zpad/internal/theme"
	"github.com/helloxz/zpad/locales"
)

// The heights of the toolbar above the content and the status bar below
// it. A vertical split centers its panes in what is left between them.
const (
	toolbarHeight   = 45 // 28 DIP controls, 8 DIP padding either side, 1 DIP border
	statusBarHeight = 26
)

// View builds the window: toolbar, sidebar and content, status bar.
func (a *App) View(c *ui.Context) {
	if !a.viewLogged {
		a.viewLogged = true
		w, h := c.Size()
		log.Printf("window %.0fx%.0f", w, h)
	}
	t := a.themeDefinition().UI
	c.SetTheme(t)
	c.SetMenuPresentation(ui.PopoverMenus)
	if !a.showSettings && !a.aboutOpen && !a.renameOpen && !a.createOpen {
		if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyF) && !c.ShortcutRepeated() {
			a.openSearchPanel()
		}
		if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyV) && !c.ShortcutRepeated() {
			a.ToggleMarkdownPreview()
		}
		if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyB) && !c.ShortcutRepeated() {
			a.OpenActiveInBrowser()
		}
		// Escape leaves a Markdown preview wherever the focus is: the
		// preview keeps a scroll state rather than a cursor, so there is
		// nothing else for it to cancel. Dialogs, menus and toasts over
		// the window take the key first, as overlays.
		if !a.searchOpen && c.Shortcut(0, ui.KeyEscape) && !c.ShortcutRepeated() {
			if tab := a.activeTab(); tab != nil && tab.preview {
				a.toggleTabPreview(tab)
			}
		}
	}

	ui.Themed(c, t).Fill().Children(func() {
		if a.customMenuBar {
			a.menuBar(c, t)
		}
		a.toolbar(c, t)
		if a.Settings.SidebarOpen {
			sp := ui.Split(c, &a.Settings.SidebarWidth,
				func() { a.sidebar(c, t) },
				func() { a.content(c, t) },
			)
			sp.Grow(1).FillWidth()
			if sp.Changed() {
				a.saveSettings()
			}
		} else {
			a.content(c, t)
		}
		a.statusBar(c, t)
		a.settingsModal(c, t)
		if !a.showSettings {
			a.aboutModal(c, t)
			if !a.aboutOpen {
				a.renameModal(c, t)
				a.createModal(c, t)
				a.searchModal(c, t)
			}
		}
		for _, t := range a.pendingToasts {
			c.AddToast(t)
		}
		a.pendingToasts = nil
	})
}

func (a *App) toolbar(c *ui.Context, t *ui.Theme) {
	// On macOS the toolbar is the title bar: keep its buttons clear of the
	// traffic lights and let its empty parts move the window. Windows draws
	// a menu bar above instead, which is the title bar there, so this bar
	// keeps its own padding.
	bar := c.TitleBar()
	tb := ui.Toolbar(c, func() {
		a.iconButton(c, t, iconSidebar, locales.ToggleSidebar(), a.Settings.SidebarOpen, a.toggleSidebar).
			KeyShortcuts(ui.Cmd, ui.KeyB)
		a.iconButton(c, t, iconPlus, locales.NewFile(), false, a.NewFile).
			KeyShortcuts(ui.Cmd, ui.KeyN)
		a.iconButton(c, t, iconOpen, locales.OpenFile(), false, a.OpenFileDialog).
			KeyShortcuts(ui.Cmd, ui.KeyO)
		a.iconButton(c, t, iconFolder, locales.OpenFolder(), false, a.OpenFolderDialog).
			KeyShortcuts(ui.Cmd|ui.Shift, ui.KeyO)
		a.iconButton(c, t, iconSave, locales.Save(), false, a.SaveActive).
			KeyShortcuts(ui.Cmd, ui.KeyS)
		ui.Box(c).Width(1).Height(18).Background(t.Border).Margin(0, 4)
		a.iconButton(c, t, iconUndo, locales.Undo(), false, a.Undo).
			KeyShortcuts(ui.Cmd, ui.KeyZ)
		a.iconButton(c, t, iconRedo, locales.Redo(), false, a.Redo).
			KeyShortcuts(ui.Cmd|ui.Shift, ui.KeyZ)
		a.iconButton(c, t, iconSearch, locales.Find(), false, func() {
			if tab := a.activeTab(); tab != nil && !tab.preview {
				tab.state.Run(editor.Find)
			}
		}).KeyShortcuts(ui.Cmd, ui.KeyF)
		a.toolButton(c, t)
		a.iconButton(c, t, iconInfo, locales.AboutApp(), a.aboutOpen, a.openAbout)
		ui.Spacer(c)
		a.iconButton(c, t, iconUpdate, locales.CheckForUpdates(), false, a.checkUpdates)
		a.splitButton(c, t)
		a.iconButton(c, t, iconSettings, locales.Settings(), a.showSettings, a.openSettings).
			KeyShortcuts(ui.Cmd, ui.KeyComma)
	}).Background(t.Surface).Height(toolbarHeight).
		BorderWidth(0, 0, 1, 0).BorderColor(t.Border)
	if a.customMenuBar {
		tb = tb.Padding(8, 10)
	} else {
		tb = tb.Padding(8, 10+bar.Right, 8, 10+bar.Left)
		if bar.Height > 0 {
			tb = tb.DragWindow()
		}
	}
}

// toolButton opens the text tools: case conversion, Base64, MD5 and JSON.
func (a *App) toolButton(c *ui.Context, t *ui.Theme) {
	b := ui.ButtonBase(c).Size(28, 28).Center().Radius(6).
		Label(locales.ToolsMenu()).Tooltip(locales.ToolsMenu()).Cursor(ui.CursorPointer)
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		ui.Icon(c, iconTools).FontSize(16).TextColor(t.Text)
	})
	b.PopoverMenuButton(func(m *ui.Menu) {
		a.toolMenu(m, a.activeTab())
	}, menuButtonCursorOptions())
}

// splitButton shows the split's direction as its icon and opens the menu
// that chooses the direction or closes the split.
func (a *App) splitButton(c *ui.Context, t *ui.Theme) {
	b := ui.ButtonBase(c).Size(28, 28).Center().Radius(6).
		Label(locales.SplitMenu()).Tooltip(locales.SplitMenu()).Cursor(ui.CursorPointer).
		KeyShortcuts(ui.Cmd, ui.KeyBackslash)
	if a.split || b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		icon := iconSplit
		if a.splitVertical {
			icon = iconSplitRows
		}
		ui.Icon(c, icon).FontSize(16).TextColor(t.Text)
	})
	b.PopoverMenuButton(func(m *ui.Menu) {
		if m.Item(locales.SplitLeftRight()).Checked(a.split && !a.splitVertical).Chosen() {
			a.openSplit(false)
		}
		if m.Item(locales.SplitTopBottom()).Checked(a.split && a.splitVertical).Chosen() {
			a.openSplit(true)
		}
		m.Separator()
		if m.Item(locales.CloseSplit()).Disabled(!a.split).Chosen() {
			a.closeSplit()
		}
	}, menuButtonCursorOptions())
}

func (a *App) iconButton(c *ui.Context, t *ui.Theme, icon *ui.SVG, label string, active bool, run func()) ui.Element {
	b := ui.ButtonBase(c).Size(28, 28).Center().Radius(6).Label(label).Tooltip(label).Cursor(ui.CursorPointer)
	if active || b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		ui.Icon(c, icon).FontSize(16).TextColor(t.Text)
	})
	if b.Clicked() {
		run()
	}
	return b
}

func (a *App) sidebar(c *ui.Context, t *ui.Theme) {
	ui.Column(c).Fill().Background(theme.Sidebar(t)).Children(func() {
		ui.Row(c).Height(38).Padding(0, 8).AlignItems(ui.Center).Gap(2).Children(func() {
			a.sidebarTabButton(c, t, locales.OpenFiles(), sidebarFiles)
			a.sidebarTabButton(c, t, locales.Workspace(), sidebarWorkspace)
			ui.Spacer(c)
			a.sidebarMenuButton(c, t)
		})
		switch a.sidebarTab {
		case sidebarWorkspace:
			a.workspacePanel(c, t)
		default:
			a.openFilesPanel(c, t, theme.FileTree(t))
		}
	})
}

func (a *App) sidebarTabButton(c *ui.Context, t *ui.Theme, label string, tab int) {
	active := a.sidebarTab == tab
	b := ui.ButtonBase(c.Key(fmt.Sprintf("sidebar-tab-%d", tab))).Height(30).Padding(0, 8).
		Center().Radius(0).Label(label).Cursor(ui.CursorPointer)
	text := t.TextMuted
	switch {
	case active:
		text = t.Text
	case b.Hovered():
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(text)
	})
	if active {
		b.BorderWidth(0, 0, 2, 0).BorderColor(t.Accent)
	}
	if tab == sidebarWorkspace && a.root != "" {
		b.Tooltip(a.root)
	}
	if b.Clicked() && !active {
		a.sidebarTab = tab
		a.update()
	}
}

func (a *App) sidebarMenuButton(c *ui.Context, t *ui.Theme) {
	menu := ui.ButtonBase(c).Size(26, 24).Center().Radius(6).
		Background(theme.Sidebar(t)).Border(1, t.Border).Label(locales.SidebarMenu()).
		Cursor(ui.CursorPointer)
	if menu.Hovered() {
		menu.Background(t.SurfaceHover)
	}
	menu.Children(func() {
		ui.Text(c, "⋯").FontSize(14).TextColor(t.TextMuted)
	})
	menu.Menu(func(m *ui.Menu) {
		if a.sidebarTab == sidebarWorkspace {
			if m.Item(locales.Refresh()).Disabled(a.tree == nil).Chosen() {
				a.refreshTree()
			}
			if m.Item(locales.ShowHidden()).Checked(a.Settings.ShowHidden).Chosen() {
				a.toggleHidden()
			}
			m.Separator()
		}
		if m.Item(locales.CloseSidebar()).Chosen() {
			a.toggleSidebar()
		}
	})
}

// openFilesPanel lists the documents open in either pane, with the
// directory each one is in: the tabs alone cannot tell files of the same
// name in different folders apart.
func (a *App) openFilesPanel(c *ui.Context, t, tree *ui.Theme) {
	docs := a.openDocuments()
	if len(docs) == 0 {
		ui.Column(c).Grow(1).Center().Gap(10).Padding(16).Children(func() {
			ui.Text(c, locales.NoOpenFiles()).FontSize(12).TextColor(t.TextMuted)
			ui.Row(c).Gap(6).Children(func() {
				if ui.Button(c, locales.NewFile()).Cursor(ui.CursorPointer).Clicked() {
					a.NewFile()
				}
				if ui.Button(c, locales.OpenFile()).Cursor(ui.CursorPointer).Clicked() {
					a.OpenFileDialog()
				}
			})
		})
		return
	}
	var active *Document
	if tab := a.activeTab(); tab != nil {
		active = tab.doc
	}
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Padding(4, 0).Children(func() {
			for _, doc := range docs {
				a.openFileRow(c, t, tree, doc, doc == active)
			}
		})
	})
}

func (a *App) openFileRow(c *ui.Context, t, tree *ui.Theme, doc *Document, active bool) {
	row := ui.Row(c).Height(26).Padding(0, 8).Gap(6).AlignItems(ui.Center).
		Cursor(ui.CursorPointer).Label(doc.Name())
	switch {
	case active:
		row.Background(tree.Accent)
	case row.Hovered():
		row.Background(tree.SurfaceHover)
	}
	row.Children(func() {
		if doc.Buffer.Modified() {
			ui.Box(c).Size(6, 6).Radius(3).Background(t.Accent).Shrink(0)
		} else {
			ui.Box(c).Size(6, 6).Shrink(0)
		}
		ui.Icon(c, iconFile).FontSize(14).TextColor(t.TextMuted)
		// The name has the row to itself; the path tip tells files of
		// the same name apart.
		ui.Text(c, doc.Name()).FontSize(12).SingleLine().TruncateMiddle()
	})
	pathTooltip(c, t, row, a.documentTooltip(doc))
	if row.Clicked() {
		a.activateDocument(doc)
	}
	if doc.Path != "" {
		row.PopoverMenu(func(m *ui.Menu) {
			if m.Item(locales.OpenInOtherPane()).Chosen() {
				a.openInOtherPane(doc.Path)
			}
			if m.Item(locales.Rename()).Chosen() {
				a.beginRename(doc.Path)
			}
			m.Separator()
			if m.Item(locales.Delete()).Danger().Chosen() {
				a.confirmDelete(doc.Path, false)
			}
		}, menuCursorOptions())
	}
}

func (a *App) documentTooltip(doc *Document) string {
	if doc.Path == "" {
		return doc.Name()
	}
	return doc.Path
}

// pathTooltip shows text beside anchor's right edge, aligned with it, so a
// long path does not cover the sidebar list it belongs to. Element.Tooltip
// always goes above its anchor, centered, so this places the tip itself and
// repeats the look of the theme's tooltip, wider to hold a path.
func pathTooltip(c *ui.Context, t *ui.Theme, anchor ui.Element, text string) {
	if text == "" {
		return
	}
	fill, ink := theme.Inverse(t)
	ui.TooltipBase(c, anchor, func(tip ui.Element) {
		tip.AttachTo(anchor, ui.AnchorEnd, ui.AnchorStart).MarginStart(6).
			MaxWidth(420).Padding(t.Space(1.25), t.Space(2)).
			Radius(t.Space(1.25)).Background(fill).TextColor(ink).
			FontSize(t.FontSize-1).
			Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.2))
		tip.Children(func() {
			ui.Text(c, text)
		})
	})
	// TooltipBase leaves the description to the caller, and assistive
	// technology does not read the tip itself.
	anchor.Description(text)
}

// tooltipBelow shows text under anchor, centered, as the controls at the
// top of the search panel need: above them is the window's edge and the
// dialog's own field. Element.Tooltip goes above, centered, unless there
// is no room there, so this places the tip itself.
func tooltipBelow(c *ui.Context, t *ui.Theme, anchor ui.Element, text string) {
	if text == "" {
		return
	}
	fill, ink := theme.Inverse(t)
	ui.TooltipBase(c, anchor, func(tip ui.Element) {
		tip.AttachTo(anchor, ui.AnchorBottom, ui.AnchorTop).Margin(4, 0, 0, 0).
			MaxWidth(320).Padding(t.Space(1.25), t.Space(2)).
			Radius(t.Space(1.25)).Background(fill).TextColor(ink).
			FontSize(t.FontSize-1).
			Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.2))
		tip.Children(func() {
			ui.Text(c, text)
		})
	})
	anchor.Description(text)
}

func (a *App) workspacePanel(c *ui.Context, t *ui.Theme) {
	if a.tree == nil {
		ui.Column(c).Grow(1).Center().Gap(8).Padding(16).Children(func() {
			ui.Text(c, locales.NoFolder()).FontSize(12).TextColor(t.TextMuted)
			if ui.Button(c, locales.OpenFolder()).Cursor(ui.CursorPointer).Clicked() {
				a.OpenFolderDialog()
			}
		})
		return
	}
	ui.Row(c).Height(26).Padding(0, 8).AlignItems(ui.Center).Gap(6).Children(func() {
		ui.Icon(c, iconFolder).FontSize(13).TextColor(t.TextMuted)
		ui.Text(c, a.tree.Name()).FontSize(12).Bold().SingleLine()
	})
	root := a.tree.Root()
	tree := ui.Themed(c, theme.FileTree(t)).Grow(1).Background(ui.Transparent)
	// The rows have menus of their own, so this one opens on the space
	// below them and creates at the top of the workspace.
	tree.PopoverMenu(func(m *ui.Menu) {
		if m.Item(locales.NewFile()).Chosen() {
			a.beginCreate(root, false)
		}
		if m.Item(locales.NewFolder()).Chosen() {
			a.beginCreate(root, true)
		}
	}, sidebarMenuOptions(t))
	tree.Children(func() {
		a.treeState.List.Selected = &a.treeSelected
		a.treeState.Generation = a.treeGeneration
		// treeRow draws the indentation and the arrow itself, inside the
		// row, so that the whole row is one click target; an outline row
		// would keep them outside its content and a click there would only
		// choose the row.
		a.treeState.Prefix = func(ui.OutlineRow) {}
		ui.Outline(c, &a.treeState, a.tree.Roots(), a.tree.Children, func(path string) {
			a.treeRow(c, t, path)
		}).Grow(1)
	})
}

// treeRow draws a row of the workspace tree whole: its indentation, its
// arrow and its name all sit inside the row, so a click anywhere in it,
// from the left edge to the right, opens the file or the folder.
func (a *App) treeRow(c *ui.Context, t *ui.Theme, path string) {
	isDir := a.tree.IsDir(path)
	open := a.treeState.Open.Has(path)
	row := ui.Row(c).FillWidth().Padding(t.Space(1), t.Space(2)).Gap(t.Space(1)).
		AlignItems(ui.Center).Cursor(ui.CursorPointer)
	row.Children(func() {
		if depth := a.treeDepth(path); depth > 0 {
			ui.Box(c).Width(float32(depth) * t.Space(4)).Shrink(0)
		}
		arrow := ui.Box(c).Size(t.Space(4), t.Space(4)).Shrink(0).Center()
		arrow.Children(func() {
			if isDir {
				chevron := ui.Icon(c, iconChevron).FontSize(12).TextColor(t.TextMuted)
				if open {
					chevron.Rotate(90)
				}
			}
		})
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			if isDir {
				ui.Icon(c, iconFolder).FontSize(14).TextColor(t.TextMuted)
			} else {
				ui.Icon(c, iconFile).FontSize(14).TextColor(t.TextMuted)
			}
			ui.Text(c, filepath.Base(path)).FontSize(12).SingleLine()
		})
	})
	if !isDir {
		row.Drag(paneDrag{kind: dragFile, path: path})
	}
	pathTooltip(c, t, row, path)
	row.PopoverMenu(func(m *ui.Menu) {
		a.selectTreePath(path)
		if isDir {
			if m.Item(locales.NewFile()).Chosen() {
				a.beginCreate(path, false)
			}
			if m.Item(locales.NewFolder()).Chosen() {
				a.beginCreate(path, true)
			}
			m.Separator()
		} else if m.Item(locales.OpenInOtherPane()).Chosen() {
			a.openInOtherPane(path)
		}
		if m.Item(locales.Rename()).Chosen() {
			a.beginRename(path)
		}
		m.Separator()
		if m.Item(locales.Delete()).Danger().Chosen() {
			a.confirmDelete(path, isDir)
		}
	}, sidebarMenuOptions(t))
	if row.Clicked() {
		// The whole row is this view's own, so the outline's row sees no
		// click and the choice is made here too.
		a.selectTreePath(path)
		a.update()
		switch {
		case !isDir:
			a.OpenPaths([]string{path})
		case row.ClickModifiers()&ui.Alt != 0:
			a.toggleFolderAll(path)
		default:
			a.toggleFolder(path)
		}
	}
}

// treeDepth is how deep path sits in the workspace: 0 for an entry at the
// workspace root, matching the indentation the rows draw.
func (a *App) treeDepth(path string) int {
	rel, err := filepath.Rel(a.tree.Root(), path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator))
}

// toggleFolderAll opens path with every folder inside it, or closes them
// all, as Option-clicking a tree row does.
func (a *App) toggleFolderAll(path string) {
	open := !a.treeState.Open.Has(path)
	var walk func(dir string)
	walk = func(dir string) {
		if open {
			a.treeState.Open.Add(dir)
		} else {
			a.treeState.Open.Remove(dir)
		}
		for _, child := range a.tree.Children(dir) {
			if a.tree.IsDir(child) {
				walk(child)
			}
		}
	}
	walk(path)
	a.update()
}

// menuCursorOptions gives the pointer the hand shape over menu items.
func menuCursorOptions() ui.PopoverMenuOptions {
	return ui.PopoverMenuOptions{Item: menuItemCursor}
}

// menuButtonCursorOptions is menuCursorOptions for a menu button.
func menuButtonCursorOptions() ui.PopoverMenuButtonOptions {
	return ui.PopoverMenuButtonOptions{Item: menuItemCursor}
}

func menuItemCursor(item ui.Element) {
	item.Cursor(ui.CursorPointer)
}

// sidebarMenuOptions styles a menu drawn in the tree, whose theme keeps
// its rows square and its accent as the selection wash.
func sidebarMenuOptions(t *ui.Theme) ui.PopoverMenuOptions {
	return ui.PopoverMenuOptions{
		Panel: func(panel ui.Element) {
			panel.Radius(t.Radius)
		},
		Item: func(item ui.Element) {
			item.Cursor(ui.CursorPointer)
			item.Radius(max(t.Radius-2, 0))
			if item.Highlighted() {
				item.Background(t.Accent).TextColor(t.AccentText)
			}
		},
	}
}

func (a *App) toggleFolder(path string) {
	if a.treeState.Open.Has(path) {
		a.treeState.Open.Remove(path)
	} else {
		a.treeState.Open.Add(path)
	}
	a.update()
}

func (a *App) content(c *ui.Context, t *ui.Theme) {
	if a.split {
		if a.centerSplit {
			w, h := c.Size()
			if a.splitVertical {
				if room := h - a.chromeHeight() - statusBarHeight; room > 2 {
					a.splitHeight = halfSplit(room)
				}
			} else {
				room := w
				if a.Settings.SidebarOpen {
					room -= a.Settings.SidebarWidth + 1
				}
				if room > 2 {
					a.splitWidth = halfSplit(room)
				}
			}
			a.centerSplit = false
		}
		if a.splitVertical {
			sp := ui.SplitVertical(c, &a.splitHeight,
				func() { a.paneContent(c, t, 0) },
				func() { a.paneContent(c, t, 1) },
			)
			sp.Grow(1).FillWidth()
			return
		}
		sp := ui.Split(c, &a.splitWidth,
			func() { a.paneContent(c, t, 0) },
			func() { a.paneContent(c, t, 1) },
		)
		sp.Grow(1).FillWidth()
		return
	}
	// Grow(1) bounds the pane to the room left below the toolbar and the
	// status bar: a long preview must scroll, not push the status bar out
	// of the window.
	ui.Box(c).Grow(1).FillWidth().MinHeight(0).Children(func() {
		a.paneContent(c, t, 0)
	})
}

// halfSplit returns the size of the first pane that leaves the two panes
// equal in room DIPs, one of which is the divider between them.
func halfSplit(room float32) float32 {
	if room <= 2 {
		return room / 2
	}
	return (room - 1) / 2
}

func (a *App) paneContent(c *ui.Context, t *ui.Theme, index int) {
	p := a.pane(index)
	col := ui.Column(c).Fill().Background(t.Background)
	drag, over := ui.DragOver[paneDrag](col)
	col.Children(func() {
		if len(p.tabs) == 0 {
			if a.split {
				a.emptyPane(c, t, index)
			} else {
				a.emptyState(c, t)
			}
			return
		}
		a.tabBar(c, t, index, p)
		tab := p.activeTab()
		if tab == nil {
			return
		}
		if tab.preview && !isMarkdownDoc(tab.doc) {
			tab.preview = false
		}
		a.configureEditor(tab, t)
		if index == a.activePane {
			if dirty := tab.doc.Buffer.Modified(); dirty != a.lastDirty {
				a.lastDirty = dirty
				if a.win != nil {
					a.win.SetDocumentEdited(dirty)
				}
			}
		}
		drop := ui.Box(c).Grow(1).FillWidth().MinHeight(0).Children(func() {
			if tab.preview {
				sc := a.markdownPreview(c, t, tab, index, a.focusEditor && index == a.activePane)
				if index != a.activePane && sc.FocusWithin() {
					a.activePane = index
					a.updateTitle()
				}
				return
			}
			v := editor.View(c, tab.state).Grow(1).FillWidth()
			if tab.pendingScroll != nil {
				y := *tab.pendingScroll
				tab.pendingScroll = nil
				tab.state.SetScrollY(y)
			}
			if a.focusEditor && index == a.activePane {
				v.Bind(&a.editorRef)
				a.editorRef.Focus(c)
				a.focusEditor = false
			}
		})
		if files := drop.DroppedFiles(); len(files) > 0 {
			a.setActivePane(index)
			a.OpenPaths(files)
		}
		if drop.FileDragOver() {
			drop.Border(2, t.Accent)
		}
		if !tab.preview && tab.state.Focused() && a.activePane != index {
			a.activePane = index
			a.updateTitle()
		}
	})
	if over && drag.wantsPane(index) {
		col.DrawOver(func(p *ui.Painter, r ui.Rect) {
			p.Stroke(ui.Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: r.H - 2}, t.Accent, 0, 2)
		})
	}
	if drag, ok := ui.Drop[paneDrag](col); ok {
		a.dropOnPane(drag, index)
	}
}

func (a *App) emptyPane(c *ui.Context, t *ui.Theme, index int) {
	ui.Column(c).Fill().Center().Gap(10).Children(func() {
		ui.Text(c, locales.PaneEmpty()).FontSize(13).TextColor(t.TextMuted)
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, locales.OpenFile()).Cursor(ui.CursorPointer).Clicked() {
				a.setActivePane(index)
				a.OpenFileDialog()
			}
			if ui.Button(c, locales.NewFile()).Cursor(ui.CursorPointer).Clicked() {
				a.setActivePane(index)
				a.NewFile()
			}
		})
	})
}

func (a *App) tabBar(c *ui.Context, t *ui.Theme, index int, p *Pane) {
	bg := theme.TabBar(t)
	if a.split && index == a.activePane {
		bg = theme.ActiveTab(t)
	}
	ui.Row(c).Height(34).FillWidth().Background(bg).
		BorderWidth(0, 0, 1, 0).BorderColor(t.Border).
		Children(func() {
			area := ui.ScrollHorizontal(c).Height(34).Grow(1).Scrollbars(false)
			// The bar takes the tabs dragged over it, drawing where the
			// tab would land: a tab reorders its own pane's tabs, and
			// lands at the pointer's place in the other pane's.
			drag, over := ui.DragOver[paneDrag](area)
			var inner ui.Element
			var rects []ui.Rect
			area.Children(func() {
				inner = ui.Row(c).Height(34).Children(func() {
					for i, tab := range p.tabs {
						rects = append(rects, a.tabItem(c, t, index, p, i, tab).Bounds())
					}
					if !a.tabOverflow[index] {
						a.newTabButton(c, t, index)
					}
				})
			})
			dropAt, caret := 0, float32(0)
			x, _, in := area.PointerPosition()
			if in && len(rects) > 0 {
				dropAt, caret = tabDropAt(rects, area.Bounds().X+x)
			}
			switch {
			case in && over && drag.kind == dragTab && !a.tabStaysAt(drag, index, dropAt):
				area.DrawOver(func(paint *ui.Painter, r ui.Rect) {
					paint.Fill(ui.Rect{X: caret - 1, Y: r.Y + 5, W: 2, H: r.H - 10}, t.Accent, 1)
				})
			case in && over && drag.kind == dragFile:
				// A file dropped on the bar opens in the pane, at the
				// end: a line under the tabs says the bar takes it.
				area.DrawOver(func(paint *ui.Painter, r ui.Rect) {
					paint.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 2, W: r.W, H: 2}, t.Accent, 0)
				})
			}
			if d, ok := ui.Drop[paneDrag](area); ok {
				if d.kind == dragTab && in && len(rects) > 0 && !a.tabStaysAt(d, index, dropAt) {
					a.dropTabAt(d, index, dropAt)
				} else {
					a.dropOnPane(d, index)
				}
			}
			if a.tabOverflow[index] {
				a.newTabButton(c, t, index)
				a.tabListButton(c, index, p)
			}
			// The tabs fit when their row is no wider than the area that
			// shows them; the bounds are the last frame's, so a change
			// asks for the frame that shows or hides the button.
			if over := inner.Bounds().W > area.Bounds().W+1; over != a.tabOverflow[index] {
				a.tabOverflow[index] = over
				a.update()
			}
		})
}

// tabDropAt returns the index a tab dropped at window x takes among the
// tabs whose boxes are rects, and where the caret before it goes: that of
// the first tab whose middle is right of x, or past the last.
func tabDropAt(rects []ui.Rect, x float32) (int, float32) {
	for i, r := range rects {
		if x < r.X+r.W/2 {
			return i, r.X
		}
	}
	last := rects[len(rects)-1]
	return len(rects), last.X + last.W
}

// tabStaysAt reports whether a tab dragged from one pane to index to of
// another, or of its own, keeps its place: dropped on itself or right
// after itself there is nothing to move.
func (a *App) tabStaysAt(d paneDrag, pane, to int) bool {
	if d.pane != pane {
		return false
	}
	for i, tab := range a.pane(d.pane).tabs {
		if tab.doc.ID == d.id {
			return to == i || to == i+1
		}
	}
	return false
}

// tabListButton shows a menu of all of a pane's tabs, for the ones a full
// tab bar hides.
func (a *App) tabListButton(c *ui.Context, paneIdx int, p *Pane) {
	wrap := ui.Box(c).Size(28, 34).Label(locales.AllTabs()).Tooltip(locales.AllTabs())
	wrap.Children(func() {
		ui.PopoverMenuButton(c, "", func(m *ui.Menu) {
			for i, tab := range p.tabs {
				if m.Item(tab.doc.Name()).ID(tab.doc.ID).Checked(i == p.active).Chosen() {
					a.activateTab(paneIdx, i)
				}
			}
		}, menuButtonCursorOptions()).Fill().Center().Padding(0).Radius(0).Cursor(ui.CursorPointer)
	})
}

// tabTextWidth caps the name a tab shows so that one long file does not take
// the whole bar and hide the others. The tab keeps the name's natural width
// below it; the title tooltip and the tab list carry the rest.
const tabTextWidth float32 = 180

func (a *App) tabItem(c *ui.Context, t *ui.Theme, paneIdx int, p *Pane, i int, tab *Tab) ui.Element {
	active := i == p.active
	bg := theme.TabBar(t)
	text := t.TextMuted
	if active {
		bg = theme.ActiveTab(t)
		text = t.Text
	}
	key := fmt.Sprintf("tab-%d-%s", paneIdx, tab.doc.ID)
	b := ui.ButtonBase(c.Key(key)).Height(32).Padding(0, 10).Gap(6).
		Radius(0).Background(bg).Label(tab.doc.Name()).
		Tooltip(a.documentTooltip(tab.doc)).Cursor(ui.CursorPointer).
		Drag(paneDrag{kind: dragTab, pane: paneIdx, id: tab.doc.ID}).
		PopoverMenu(func(m *ui.Menu) { a.tabContextMenu(m, paneIdx, i) }, menuCursorOptions()).
		// A middle press closes the tab at once, as editors do, without
		// activating it: MyGo's clicks are the primary button's, so the
		// tab takes the press; other buttons go on as they would.
		HandleInput(func(ev ui.InputEvent) bool {
			if ev.Kind != ui.InputPointerDown || ev.Button != 2 {
				return false
			}
			a.CloseTab(paneIdx, i)
			return true
		})
	if !active && b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	closed := false
	b.Children(func() {
		if tab.doc.Buffer.Modified() {
			ui.Box(c).Size(6, 6).Radius(3).Background(t.Accent).Shrink(0)
		}
		if tab.preview {
			eye := ui.ButtonBase(c.Key(key+"-preview")).Size(16, 16).Center().Shrink(0).
				Radius(4).Label(locales.BackToSource() + ": " + tab.doc.Name()).
				Tooltip(locales.BackToSource()).Cursor(ui.CursorPointer)
			eye.Children(func() {
				ui.Icon(c, iconPreview).FontSize(11).TextColor(t.Accent)
			})
			if eye.Clicked() {
				a.toggleTabPreview(tab)
			}
		}
		ui.Text(c, tab.doc.Name()).FontSize(12).TextColor(text).SingleLine().
			MaxWidth(tabTextWidth).TruncateMiddle()
		close := ui.ButtonBase(c.Key(key+"-close")).Size(16, 16).Center().Shrink(0).
			Radius(4).Label(locales.Close()).Tooltip(locales.Close()).Cursor(ui.CursorPointer)
		close.Children(func() {
			ui.Icon(c, iconClose).FontSize(11).TextColor(t.TextMuted)
		})
		if close.Clicked() {
			a.CloseTab(paneIdx, i)
			closed = true
		}
	})
	if active {
		b.BorderWidth(0, 0, 2, 0).BorderColor(t.Accent)
	}
	if b.Clicked() && !closed {
		p.active = i
		a.activePane = paneIdx
		a.focusEditor = true
		a.updateTitle()
		a.update()
	}
	if a.revealDoc != "" && a.revealPane == paneIdx && a.revealDoc == tab.doc.ID {
		b.ScrollIntoView(ui.ScrollIntoViewOptions{Inline: ui.ScrollNearest})
		a.revealDoc = ""
		a.revealPane = -1
	}
	return b
}

// tabContextMenu fills a tab's context menu: closing it or the tabs
// beside it, and moving or copying it to the other pane.
func (a *App) tabContextMenu(m *ui.Menu, paneIdx, i int) {
	p := a.pane(paneIdx)
	if i < 0 || i >= len(p.tabs) {
		return
	}
	// The items below can close or move this tab while the menu is built,
	// so keep it now for the entry that opens it.
	tab := p.tabs[i]
	if m.Item(locales.CloseTab()).Chosen() {
		a.CloseTab(paneIdx, i)
	}
	if m.Item(locales.CloseTabsLeft()).Disabled(i == 0).Chosen() {
		a.closeTabs(paneIdx, p.tabs[:i])
	}
	if m.Item(locales.CloseTabsRight()).Disabled(i == len(p.tabs)-1).Chosen() {
		a.closeTabs(paneIdx, p.tabs[i+1:])
	}
	m.Separator()
	toMain := paneIdx != 0
	if m.Item(locales.MoveToMainPane()).Disabled(!toMain).Chosen() {
		a.moveTabIndex(paneIdx, i, 0)
	}
	// A pane cannot stay empty, so the only tab of a lone pane cannot
	// move into the split it would have to create.
	canMoveOther := paneIdx == 0 && (a.split || len(p.tabs) > 1)
	if m.Item(locales.MoveToOtherPane()).Disabled(!canMoveOther).Chosen() {
		a.ensureSplit()
		a.moveTabIndex(paneIdx, i, 1)
	}
	m.Separator()
	if m.Item(locales.CopyToMainPane()).Disabled(!toMain).Chosen() {
		a.copyTabIndex(paneIdx, i, 0)
	}
	if m.Item(locales.CopyToOtherPane()).Disabled(paneIdx == 1).Chosen() {
		a.ensureSplit()
		a.copyTabIndex(paneIdx, i, 1)
	}
	if tab.doc.Path != "" {
		m.Separator()
		if m.Item(locales.RevealInFileManager()).Chosen() {
			a.revealDocument(tab.doc)
		}
	}
	if isMarkdownDoc(tab.doc) {
		if tab.doc.Path == "" {
			// The saved-file block above drew the rule for a file with a
			// path; an untitled document has only this entry after copying.
			m.Separator()
		}
		can := a.canPreviewTab(tab)
		label := locales.PreviewMarkdown()
		if tab.preview {
			label = locales.BackToSource()
		} else if !can {
			label = locales.PreviewTooLarge()
		}
		if m.Item(label).Disabled(!can).Chosen() {
			a.activateTab(paneIdx, i)
			a.toggleTabPreview(tab)
		}
	}
	if tab.doc.Path != "" && isHTMLPath(tab.doc.Path) &&
		m.Item(locales.OpenInBrowser()).Chosen() {
		a.openTabInBrowser(tab)
	}
}

func (a *App) newTabButton(c *ui.Context, t *ui.Theme, paneIdx int) {
	b := ui.ButtonBase(c.Key(fmt.Sprintf("new-tab-%d", paneIdx))).Size(28, 32).Center().Radius(0).
		Label(locales.NewFile()).Tooltip(locales.NewFile()).Cursor(ui.CursorPointer)
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		ui.Icon(c, iconPlus).FontSize(14).TextColor(t.TextMuted)
	})
	if b.Clicked() {
		a.setActivePane(paneIdx)
		a.NewFile()
	}
}

func (a *App) configureEditor(tab *Tab, t *ui.Theme) {
	st := tab.state
	st.SoftWrap = a.Settings.SoftWrap
	st.ReadOnly = tab.doc.ReadOnly
	switch a.Settings.LineNumbers {
	case "relative":
		st.LineNumbers = editor.LineNumbersRelative
	case "off":
		st.LineNumbers = editor.LineNumbersOff
	default:
		st.LineNumbers = editor.LineNumbersOn
	}
	st.TabSize = a.Settings.TabSize
	if a.Settings.UseTabs {
		st.Indent = editor.IndentTabs
	} else {
		st.Indent = editor.IndentSpaces
	}
	st.NoIndentGuides = !a.Settings.ShowGuides
	st.Whitespace = whitespaceMode(a.Settings.Whitespace)
	st.NoAutoClose = !a.Settings.AutoClose
	st.Font = ui.Font{Family: a.Settings.FontFamily, Size: a.Settings.FontSize}
	st.LineHeight = a.Settings.LineHeight
	st.Theme = theme.EditorTheme(t)
	st.Label = tab.doc.Name()
	st.Placeholder = locales.EditorPlaceholder()
	st.RenderFindBar = func(c *ui.Context, bar ui.Element, f *editor.FindBar) {
		editor.DefaultFindBar(c, bar, f)
		bar.Background(t.Surface).Border(1, t.Border).Radius(10).
			Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.18)).Padding(6)
	}
	st.RenderGoToLine = func(c *ui.Context, bar ui.Element, g *editor.GoToLineBar) {
		editor.DefaultGoToLine(c, bar, g)
		bar.Background(t.Surface).Border(1, t.Border).Radius(10).
			Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.18)).Padding(6)
	}
	st.ContextMenu = func(m *ui.Menu) {
		a.editorContextMenu(tab, m)
	}
}

// whitespaceMode turns the setting into the editor's own choice of where
// spaces and tabs show as dots and arrows.
func whitespaceMode(v string) editor.Whitespace {
	switch v {
	case "none":
		return editor.WhitespaceNone
	case "all":
		return editor.WhitespaceAll
	case "trailing":
		return editor.WhitespaceTrailing
	default:
		return editor.WhitespaceSelection
	}
}

func (a *App) editorContextMenu(tab *Tab, m *ui.Menu) {
	// MyGo separates the app's items from its own: one line, not two.
	// The Markdown preview is here because the editor's own menu has no
	// place for it; the tab's menu shows the same item.
	if tab != nil && isMarkdownDoc(tab.doc) {
		can := a.canPreviewTab(tab)
		label := locales.PreviewMarkdown()
		if !can {
			label = locales.PreviewTooLarge()
		}
		if m.Item(label).Shortcut(ui.Cmd|ui.Shift, ui.KeyV).Disabled(!can).Chosen() {
			a.toggleTabPreview(tab)
		}
	}
	m.Submenu(locales.ToolsMenu(), func(mm *ui.Menu) {
		a.toolMenu(mm, tab)
	})
	m.Submenu(locales.LineEnding(), func(mm *ui.Menu) {
		if mm.Item("LF").Chosen() {
			a.SetLineEnding("\n")
		}
		if mm.Item("CRLF").Chosen() {
			a.SetLineEnding("\r\n")
		}
		if mm.Item("CR").Chosen() {
			a.SetLineEnding("\r")
		}
	})
	m.Submenu(locales.MenuEncoding(), func(mm *ui.Menu) {
		mm.Submenu(locales.ReopenWithEncoding(), func(m2 *ui.Menu) {
			for _, kind := range encoding.All {
				k := kind
				if m2.Item(encoding.Label(k)).Chosen() {
					a.ReopenWithEncoding(k)
				}
			}
		})
		mm.Submenu(locales.SaveWithEncoding(), func(m2 *ui.Menu) {
			for _, kind := range encoding.All {
				k := kind
				if m2.Item(encoding.Label(k)).Chosen() {
					a.SaveWithEncoding(k)
				}
			}
		})
	})
	if m.Item(locales.CloseTab()).Chosen() {
		a.CloseActiveTab()
	}
}

func (a *App) emptyState(c *ui.Context, t *ui.Theme) {
	ui.Column(c).Fill().Center().Gap(14).Children(func() {
		ui.Text(c, "ZPad").FontSize(30).Bold()
		ui.Text(c, locales.AppTagline()).FontSize(13).TextColor(t.TextMuted)
		ui.Row(c).Gap(8).Children(func() {
			if ui.PrimaryButton(c, locales.OpenFile()).Cursor(ui.CursorPointer).Clicked() {
				a.OpenFileDialog()
			}
			if ui.Button(c, locales.OpenFolder()).Cursor(ui.CursorPointer).Clicked() {
				a.OpenFolderDialog()
			}
			if ui.Button(c, locales.NewFile()).Cursor(ui.CursorPointer).Clicked() {
				a.NewFile()
			}
		})
		ui.Text(c, locales.ShortcutsHint(
			shortcutText("CmdOrCtrl+O"),
			shortcutText("CmdOrCtrl+Shift+O"),
			shortcutText("CmdOrCtrl+N"),
		)).FontSize(11).TextColor(t.TextMuted)
	})
}

func (a *App) statusBar(c *ui.Context, t *ui.Theme) {
	ui.Row(c).Height(statusBarHeight).FillWidth().Padding(0, 10).AlignItems(ui.Center).Gap(10).
		Background(t.Surface).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).
		Children(func() {
			if a.split {
				ui.Text(c, a.paneLabel(a.activePane)).FontSize(11).TextColor(t.Accent)
			}
			tab := a.activeTab()
			if tab == nil {
				msg, col := locales.Ready(), t.TextMuted
				if a.loading > 0 {
					msg, col = locales.Opening(), t.Accent
				} else if a.errMsg != "" {
					msg, col = a.errMsg, t.Danger
				}
				ui.Text(c, msg).FontSize(11).TextColor(col).SingleLine()
				ui.Spacer(c)
				ui.Text(c, "ZPad "+Version).FontSize(11).TextColor(t.TextMuted)
				return
			}

			doc := tab.doc
			if a.loading > 0 {
				ui.Text(c, locales.Opening()).FontSize(11).TextColor(t.Accent)
			}
			if a.errMsg != "" {
				ui.Text(c, a.errMsg).FontSize(11).TextColor(t.Danger).MaxWidth(420).SingleLine()
			} else {
				ui.Text(c, doc.DisplayPath(a.root)).FontSize(11).TextColor(t.TextMuted).
					MaxWidth(320).SingleLine()
				a.revealButton(c, t, doc)
			}
			a.languageControl(c, t, doc)
			a.previewButton(c, t, tab)
			a.browserButton(c, t, tab)
			if doc.ReadOnly {
				ui.Text(c, locales.ReadOnly()).FontSize(11).TextColor(t.Warning)
			}

			ui.PopoverMenuButton(c, encoding.Label(doc.Encoding), func(m *ui.Menu) {
				m.Submenu(locales.ReopenWithEncoding(), func(mm *ui.Menu) {
					for _, kind := range encoding.All {
						k := kind
						if mm.Item(encoding.Label(k)).Checked(doc.Encoding == k).Chosen() {
							a.ReopenWithEncoding(k)
						}
					}
				})
				m.Submenu(locales.SaveWithEncoding(), func(mm *ui.Menu) {
					for _, kind := range encoding.All {
						k := kind
						if mm.Item(encoding.Label(k)).Checked(doc.Encoding == k).Chosen() {
							a.SaveWithEncoding(k)
						}
					}
				})
			}, menuButtonCursorOptions()).FontSize(11).Padding(0, 3).Cursor(ui.CursorPointer)

			currentEOL := doc.Buffer.LineEnding()
			ui.PopoverMenuButton(c, eolLabel(currentEOL), func(m *ui.Menu) {
				for _, item := range []struct{ label, value string }{
					{"LF", "\n"}, {"CRLF", "\r\n"}, {"CR", "\r"},
				} {
					if m.Item(item.label).Checked(currentEOL == item.value).Chosen() {
						a.SetLineEnding(item.value)
					}
				}
			}, menuButtonCursorOptions()).FontSize(11).Padding(0, 3).Cursor(ui.CursorPointer)

			ui.Spacer(c)
			if tab.preview {
				ui.Text(c, locales.PreviewMode()).FontSize(11).TextColor(t.Accent)
			} else {
				ui.Text(c, cursorLabel(tab)).FontSize(11).TextColor(t.TextMuted)
				if sel := selectionLabel(tab); sel != "" {
					ui.Text(c, sel).FontSize(11).TextColor(t.TextMuted)
				}
			}
			ui.Text(c, formatSize(doc.Size)).FontSize(11).TextColor(t.TextMuted)
			if doc.Buffer.Modified() {
				ui.Box(c).Size(6, 6).Radius(3).Background(t.Accent)
			}
		})
}

// revealButton shows a saved file in the system file manager, beside the
// path it belongs to. An untitled document has no path, so the status bar
// shows nothing for it.
func (a *App) revealButton(c *ui.Context, t *ui.Theme, doc *Document) {
	if doc == nil || doc.Path == "" {
		return
	}
	b := ui.ButtonBase(c.Key("reveal-in-file-manager")).Size(20, 20).Center().Radius(4).
		Label(locales.RevealInFileManager()).
		Tooltip(locales.RevealInFileManager()).
		Cursor(ui.CursorPointer)
	color := t.TextMuted
	if b.Hovered() {
		b.Background(t.SurfaceHover)
		color = t.Text
	}
	b.Children(func() {
		ui.Icon(c, iconReveal).FontSize(13).TextColor(color)
	})
	if b.Clicked() {
		a.revealDocument(doc)
	}
}

// browserButton opens the active HTML tab in the system browser, beside
// the language it belongs to. Other files have no such action, so the
// status bar shows nothing for them.
func (a *App) browserButton(c *ui.Context, t *ui.Theme, tab *Tab) {
	if tab == nil || !isHTMLPath(tab.doc.Path) {
		return
	}
	b := ui.ButtonBase(c.Key("open-in-browser")).Size(20, 20).Center().Radius(4).
		Label(locales.OpenInBrowser()).
		Tooltip(locales.OpenInBrowser()+" ("+openInBrowserShortcut()+")").
		KeyShortcuts(ui.Cmd|ui.Shift, ui.KeyB).
		Cursor(ui.CursorPointer)
	color := t.TextMuted
	if b.Hovered() {
		b.Background(t.SurfaceHover)
		color = t.Text
	}
	b.Children(func() {
		ui.Icon(c, iconBrowser).FontSize(13).TextColor(color)
	})
	if b.Clicked() {
		a.openTabInBrowser(tab)
	}
}

func (a *App) paneLabel(i int) string {
	if a.splitVertical {
		if i == 1 {
			return locales.BottomPane()
		}
		return locales.TopPane()
	}
	if i == 1 {
		return locales.RightPane()
	}
	return locales.LeftPane()
}

// languageControl shows the language of the document, and for an untitled
// one lets it be chosen: the choice highlights the text and names the file
// its save starts with. A file that has a path keeps the language its name
// gives it, so the label is not a control there; renaming the file changes
// the highlight.
func (a *App) languageControl(c *ui.Context, t *ui.Theme, doc *Document) {
	if doc.Path != "" {
		ui.Text(c, languageLabel(doc)).FontSize(11).TextColor(t.TextMuted).
			Tooltip(locales.LanguageFromExtension())
		return
	}
	ui.PopoverMenuButton(c, languageLabel(doc), func(m *ui.Menu) {
		if m.Item(locales.PlainText()).Checked(doc.Language == nil).Chosen() {
			a.SelectLanguage(nil)
		}
		groups := untitledLanguageGroups()
		m.Separator()
		for i, group := range [][]*editor.Language{groups.common, groups.rest} {
			if i > 0 {
				// A rule keeps the common languages apart from the rest.
				m.Separator()
			}
			for _, lang := range group {
				l := lang
				if m.Item(l.Name).Checked(doc.Language == l).Chosen() {
					a.SelectLanguage(l)
				}
			}
		}
	}, languageMenuOptions()).FontSize(11).Padding(0, 3).Cursor(ui.CursorPointer).
		Tooltip(locales.ChooseLanguage())
}

// languageMenuOptions styles the language menu of the status bar: the
// languages scroll in a popup no taller than the settings drop-downs, so the
// long list covers a part of the window rather than all of it.
func languageMenuOptions() ui.PopoverMenuButtonOptions {
	o := menuButtonCursorOptions()
	o.Panel = func(panel ui.Element) {
		panel.MaxHeight(selectPopupHeight)
	}
	return o
}

// languageLabel names the document's language, noting the files too long to
// highlight, whose buffer shows as plain text.
func languageLabel(doc *Document) string {
	if doc.Language == nil {
		return locales.PlainText()
	}
	if doc.Buffer.Language() == nil {
		return locales.LanguageNoHighlight(doc.Language.Name)
	}
	return doc.Language.Name
}

func eolLabel(eol string) string {
	switch eol {
	case "\r\n":
		return "CRLF"
	case "\r":
		return "CR"
	default:
		return "LF"
	}
}

func cursorLabel(tab *Tab) string {
	p := tab.state.CursorPos()
	line := 0
	if p.Line >= 0 && p.Line < tab.doc.Buffer.LineCount() {
		line = p.Line
	}
	text := tab.doc.Buffer.Line(line)
	col := p.Col
	if col > len(text) {
		col = len(text)
	}
	return locales.LineColumn(line+1, utf8.RuneCountInString(text[:col])+1)
}

func selectionLabel(tab *Tab) string {
	n := 0
	for _, s := range tab.state.Selections() {
		if !s.Empty() {
			n += utf8.RuneCountInString(tab.doc.Buffer.Slice(s.Start(), s.End()))
		}
	}
	if n == 0 {
		return ""
	}
	return locales.SelectedCount(n)
}

func formatSize(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	}
}
