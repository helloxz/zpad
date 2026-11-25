package app

import (
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helloxz/zpad/internal/caseconv"
	"github.com/helloxz/zpad/internal/encoding"
	"github.com/helloxz/zpad/internal/theme"
	"github.com/helloxz/zpad/locales"
)

// InstallMenu builds the application menu.
func (a *App) InstallMenu() {
	items := a.menuItems()
	// The labels the Windows menu bar draws, in the menus' order; the
	// menus themselves are rebuilt when one opens, for their current
	// state.
	a.menuBarNames = topMenuLabels(items)
	mygo.App.SetMenu(mygo.NewMenu(items))
}

// menuItems is the application menu. It is rebuilt when the language
// changes, so every label is made now.
func (a *App) menuItems() []*mygo.MenuItem {
	fileMenu := []*mygo.MenuItem{
		{Label: locales.NewFile(), Accelerator: "CmdOrCtrl+N", Click: func(*mygo.MenuItem, *mygo.Window) { a.NewFile() }},
		{Label: locales.OpenFileEllipsis(), Accelerator: "CmdOrCtrl+O", Click: func(*mygo.MenuItem, *mygo.Window) { a.OpenFileDialog() }},
		{Label: locales.OpenFolderEllipsis(), Accelerator: "CmdOrCtrl+Shift+O", Click: func(*mygo.MenuItem, *mygo.Window) { a.OpenFolderDialog() }},
		mygo.Separator(),
		{Label: locales.Save(), Accelerator: "CmdOrCtrl+S", Click: func(*mygo.MenuItem, *mygo.Window) { a.SaveActive() }},
		{Label: locales.SaveAs(), Accelerator: "CmdOrCtrl+Shift+S", Click: func(*mygo.MenuItem, *mygo.Window) { a.SaveAsActive() }},
		mygo.Separator(),
		{Label: locales.CloseTab(), Accelerator: "CmdOrCtrl+W", Click: func(*mygo.MenuItem, *mygo.Window) { a.CloseActiveTab() }},
	}

	editMenu := []*mygo.MenuItem{
		{Role: mygo.RoleUndo},
		{Role: mygo.RoleRedo},
		mygo.Separator(),
		{Role: mygo.RoleCut},
		{Role: mygo.RoleCopy},
		{Role: mygo.RolePaste},
		{Role: mygo.RoleSelectAll},
		mygo.Separator(),
		{Label: locales.CaseMenu(), Submenu: []*mygo.MenuItem{
			{Label: locales.UpperCase(), Click: func(*mygo.MenuItem, *mygo.Window) { a.ConvertCase(caseconv.Upper) }},
			{Label: locales.LowerCase(), Click: func(*mygo.MenuItem, *mygo.Window) { a.ConvertCase(caseconv.Lower) }},
			{Label: locales.TitleCase(), Click: func(*mygo.MenuItem, *mygo.Window) { a.ConvertCase(caseconv.Title) }},
			{Label: locales.InvertCase(), Click: func(*mygo.MenuItem, *mygo.Window) { a.ConvertCase(caseconv.Invert) }},
		}},
		mygo.Separator(),
		{Label: locales.Snippets(), Submenu: []*mygo.MenuItem{
			{Label: locales.InsertSnippetEllipsis(), Accelerator: "CmdOrCtrl+Alt+S", Click: func(*mygo.MenuItem, *mygo.Window) { a.openSnippets() }},
			{Label: locales.SnippetFromSelectionEllipsis(), Click: func(*mygo.MenuItem, *mygo.Window) { a.openSnippetFromSelection() }},
		}},
	}

	viewMenu := []*mygo.MenuItem{
		{Label: locales.ToggleSidebar(), Accelerator: "CmdOrCtrl+B", Click: func(*mygo.MenuItem, *mygo.Window) { a.toggleSidebar() }},
		{Label: locales.ToggleSplit(), Accelerator: "CmdOrCtrl+\\", Click: func(*mygo.MenuItem, *mygo.Window) { a.toggleSplit() }},
		{Label: locales.SplitDirection(), Submenu: []*mygo.MenuItem{
			{Label: locales.SplitLeftRight(), Click: func(*mygo.MenuItem, *mygo.Window) { a.openSplit(false) }},
			{Label: locales.SplitTopBottom(), Click: func(*mygo.MenuItem, *mygo.Window) { a.openSplit(true) }},
		}},
		{Label: locales.Refresh(), Click: func(*mygo.MenuItem, *mygo.Window) { a.refreshTree() }},
		{Label: locales.FindEllipsis(), Click: func(*mygo.MenuItem, *mygo.Window) {
			if tab := a.activeTab(); tab != nil && !tab.preview {
				tab.state.Run(editor.Find)
			}
		}},
		{Label: locales.ReplaceEllipsis(), Click: func(*mygo.MenuItem, *mygo.Window) {
			if tab := a.activeTab(); tab != nil && !tab.preview {
				tab.state.Run(editor.FindReplace)
			}
		}},
		{Label: locales.SearchInOpenFilesEllipsis(), Accelerator: "CmdOrCtrl+Shift+F", Click: func(*mygo.MenuItem, *mygo.Window) { a.openSearchPanel() }},
		{Label: locales.ShowHidden(), Type: mygo.MenuItemCheckbox, Checked: a.Settings.ShowHidden, Click: func(it *mygo.MenuItem, _ *mygo.Window) {
			a.toggleHidden()
			it.SetChecked(a.Settings.ShowHidden)
		}},
		mygo.Separator(),
		{Label: locales.Theme(), Submenu: []*mygo.MenuItem{
			{Label: locales.ThemeLight(), Click: func(*mygo.MenuItem, *mygo.Window) { a.setThemeID(theme.LightID) }},
			{Label: locales.ThemeDark(), Click: func(*mygo.MenuItem, *mygo.Window) { a.setThemeID(theme.DarkID) }},
			{Label: locales.ThemeSystem(), Click: func(*mygo.MenuItem, *mygo.Window) { a.setThemeID(theme.SystemID) }},
			mygo.Separator(),
			{Label: locales.MoreThemesEllipsis(), Click: func(*mygo.MenuItem, *mygo.Window) { a.openSettings() }},
		}},
		mygo.Separator(),
		{Label: locales.SettingsEllipsis(), Accelerator: "CmdOrCtrl+,", Click: func(*mygo.MenuItem, *mygo.Window) { a.openSettings() }},
	}

	encMenu := []*mygo.MenuItem{
		{Label: locales.ReopenWithEncoding(), Submenu: encodingItems(func(kind encoding.Kind) {
			a.ReopenWithEncoding(kind)
		})},
		{Label: locales.SaveWithEncoding(), Submenu: encodingItems(func(kind encoding.Kind) {
			a.SaveWithEncoding(kind)
		})},
		mygo.Separator(),
		{Label: locales.LineEnding(), Submenu: []*mygo.MenuItem{
			{Label: "LF", Click: func(*mygo.MenuItem, *mygo.Window) { a.SetLineEnding("\n") }},
			{Label: "CRLF", Click: func(*mygo.MenuItem, *mygo.Window) { a.SetLineEnding("\r\n") }},
			{Label: "CR", Click: func(*mygo.MenuItem, *mygo.Window) { a.SetLineEnding("\r") }},
		}},
	}

	// The app menu's About item is ours rather than RoleAbout: the system
	// panel has no room for the license and the repository, and its links
	// would not open. The rest of the menu keeps the platform roles.
	appMenu := &mygo.MenuItem{Role: mygo.RoleAppMenu, Submenu: []*mygo.MenuItem{
		{Label: locales.AboutApp(), Click: func(*mygo.MenuItem, *mygo.Window) { a.openAbout() }},
		{Label: locales.CheckForUpdates(), Click: func(*mygo.MenuItem, *mygo.Window) { a.checkUpdates() }},
		{Label: locales.ViewLog(), Disabled: a.LogPath == "", Click: func(*mygo.MenuItem, *mygo.Window) { a.OpenLog() }},
		mygo.Separator(),
		{Role: mygo.RoleServices},
		mygo.Separator(),
		{Role: mygo.RoleHide},
		{Role: mygo.RoleHideOthers},
		{Role: mygo.RoleUnhide},
		mygo.Separator(),
		{Role: mygo.RoleQuit},
	}}

	items := []*mygo.MenuItem{
		appMenu,
		{Label: locales.MenuFile(), Submenu: fileMenu},
		{Label: locales.MenuEdit(), Submenu: editMenu},
		{Label: locales.MenuView(), Submenu: viewMenu},
		{Label: locales.MenuEncoding(), Submenu: encMenu},
	}
	// The app sets its own menu bar, so macOS has no Window menu unless one
	// is added: without RoleMinimize, Cmd+M does nothing.
	if runtime.GOOS == "darwin" {
		items = append(items, &mygo.MenuItem{Label: locales.MenuWindow(), Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleMinimize, Label: locales.Minimize()},
		}})
	}
	// RoleAppMenu is macOS only, so the other platforms get their About in
	// a Help menu of their own.
	if runtime.GOOS != "darwin" {
		items = append(items, &mygo.MenuItem{Label: locales.MenuHelp(), Submenu: []*mygo.MenuItem{
			{Label: locales.AboutApp(), Click: func(*mygo.MenuItem, *mygo.Window) { a.openAbout() }},
			{Label: locales.CheckForUpdates(), Click: func(*mygo.MenuItem, *mygo.Window) { a.checkUpdates() }},
			{Label: locales.ViewLog(), Disabled: a.LogPath == "", Click: func(*mygo.MenuItem, *mygo.Window) { a.OpenLog() }},
		}})
	}
	return items
}

func encodingItems(run func(encoding.Kind)) []*mygo.MenuItem {
	items := make([]*mygo.MenuItem, 0, len(encoding.All))
	for _, kind := range encoding.All {
		k := kind
		items = append(items, &mygo.MenuItem{
			Label: encoding.Label(k),
			Click: func(*mygo.MenuItem, *mygo.Window) { run(k) },
		})
	}
	return items
}
