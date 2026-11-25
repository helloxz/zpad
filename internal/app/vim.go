package app

import (
	"runtime"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helloxz/zpad/internal/vim"
	"github.com/helloxz/zpad/locales"
)

// vimHost is the few editor operations a vim.Session needs, over one tab.
type vimHost struct {
	a   *App
	tab *Tab
}

func (h *vimHost) Text() vim.Buffer { return h.tab.doc.Buffer.Snapshot() }

func (h *vimHost) Cursor() int { return h.tab.state.Cursor() }

func (h *vimHost) SetCursor(off int) {
	h.tab.state.SetSelections(editor.Selection{Anchor: off, Head: off})
}

func (h *vimHost) SetSelection(anchor, head int) {
	h.tab.state.SetSelections(editor.Selection{Anchor: anchor, Head: head})
}

func (h *vimHost) Replace(start, end int, text string) {
	if h.tab.doc.ReadOnly {
		return
	}
	h.tab.doc.Buffer.Edit(editor.Edit{Start: start, End: end, Text: text})
}

func (h *vimHost) Undo() bool {
	if h.tab.doc.ReadOnly {
		return false
	}
	return h.tab.state.Run(editor.Undo)
}

func (h *vimHost) Redo() bool {
	if h.tab.doc.ReadOnly {
		return false
	}
	return h.tab.state.Run(editor.Redo)
}

func (h *vimHost) MoveToBracket() bool { return h.tab.state.Run(editor.MoveToBracket) }

func (h *vimHost) Page(forward bool) bool {
	if forward {
		return h.tab.state.Run(editor.MovePageDown)
	}
	return h.tab.state.Run(editor.MovePageUp)
}

func (h *vimHost) FindNext(previous bool, currentStart, currentEnd int) (int, int, bool) {
	if currentEnd > currentStart {
		h.tab.state.SetSelections(editor.Selection{Anchor: currentStart, Head: currentEnd})
	}
	cmd := editor.FindNext
	if previous {
		cmd = editor.FindPrevious
	}
	if !h.tab.state.Run(cmd) {
		return 0, 0, false
	}
	sel := h.tab.state.Selection()
	h.tab.state.SetSelections(editor.Selection{Anchor: sel.Start(), Head: sel.Start()})
	return sel.Start(), sel.End(), true
}

func (h *vimHost) OpenCommandLine(kind rune) { h.a.openVimCommand(kind) }
func (h *vimHost) Save()                     { h.a.SaveActive() }
func (h *vimHost) Close()                    { h.a.CloseActiveTab() }
func (h *vimHost) SaveAndClose()             { h.a.saveVimAndClose(h.tab) }
func (h *vimHost) GoToLine(line int) {
	h.tab.state.GoTo(editor.Pos{Line: line - 1, Col: 0})
}

func (h *vimHost) Indent() string {
	if h.a.Settings.UseTabs {
		return "\t"
	}
	n := h.a.Settings.TabSize
	if n < 1 {
		n = 4
	}
	return strings.Repeat(" ", n)
}

func (h *vimHost) ReadOnly() bool { return h.tab.doc.ReadOnly }

func (h *vimHost) ReadClipboard() string {
	if h.a.frame == nil {
		return ""
	}
	return h.a.frame.ReadClipboard()
}

func (h *vimHost) WriteClipboard(text string) {
	if h.a.frame != nil {
		h.a.frame.WriteClipboard(text)
	}
}

// configureVim installs or removes the modal input of a tab as the setting
// says, and gives the editor the caret shape and input method of its mode.
func (a *App) configureVim(tab *Tab, st *editor.State) {
	if !a.Settings.VimMode || tab.preview {
		if st.OnInput != nil {
			st.OnInput = nil
		}
		st.NoInputMethod = false
		st.CursorShape = editor.CursorBar
		return
	}
	if tab.vim == nil {
		tab.vim = vim.New()
	}
	if st.OnInput == nil {
		st.OnInput = func(ev ui.InputEvent) bool { return a.vimInput(tab, ev) }
	}
	a.applyVimState(tab)
}

// applyVimState tells the editor what the session's mode means: a block
// caret in normal and visual modes, a bar while inserting, and input
// methods off while keys are commands.
func (a *App) applyVimState(tab *Tab) {
	if tab.vim == nil {
		return
	}
	st := tab.state
	st.NoInputMethod = tab.vim.Mode() != vim.Insert
	switch {
	case tab.vim.Mode() == vim.Insert:
		st.CursorShape = editor.CursorBar
	case tab.vim.Pending():
		st.CursorShape = editor.CursorUnderline
	default:
		st.CursorShape = editor.CursorBlock
	}
}

// vimInput is the tab editor's OnInput while Vim mode is on.
func (a *App) vimInput(tab *Tab, ev ui.InputEvent) bool {
	if !a.Settings.VimMode || tab.preview || tab.vim == nil {
		return false
	}
	v := tab.vim
	h := &vimHost{a: a, tab: tab}
	switch ev.Kind {
	case ui.InputKeyDown:
		if a.vimPassThrough(ev) {
			return false
		}
		if v.Mode() == vim.Insert {
			if ev.Key == ui.KeyEscape || ev.Mods&ui.Ctrl != 0 && ev.Key == ui.KeyBracketLeft {
				v.Key(vim.KeyEscape, h)
				a.applyVimState(tab)
				return true
			}
			return false
		}
		if ev.Key == ui.KeyBracketLeft && ev.Mods&ui.Ctrl != 0 {
			v.Key(vim.KeyEscape, h)
			a.applyVimState(tab)
			return true
		}
		if k, ok := vimSpecial(ev); ok {
			v.Key(k, h)
			a.applyVimState(tab)
			return true
		}
		if ev.Mods&ui.Ctrl != 0 {
			switch ev.Key {
			case ui.KeyR:
				v.Key(vim.KeyCtrlR, h)
			case ui.KeyF:
				h.Page(true)
			case ui.KeyB:
				h.Page(false)
			}
			a.applyVimState(tab)
			return true
		}
		if ev.Mods&ui.Alt != 0 || ev.Mods&ui.Super != 0 {
			// Alt's keys and the Windows key do nothing in a modal editor;
			// the app's shortcuts on Command were let through above.
			return true
		}
		// A key that types text comes with an InputText event after it;
		// that event is the command, so the key itself is left to it.
		return false
	case ui.InputText:
		if v.Mode() == vim.Insert {
			return false
		}
		if ev.Text == "" {
			return false
		}
		for i, r := range ev.Text {
			if v.Mode() == vim.Insert {
				off := tab.state.Cursor()
				rest := ev.Text[i:]
				h.Replace(off, off, rest)
				h.SetCursor(off + len(rest))
				break
			}
			if a.vimCmdOpen {
				a.vimCmdValue += ev.Text[i:]
				break
			}
			v.Rune(r, h)
		}
		a.applyVimState(tab)
		return true
	case ui.InputKeyUp:
		return false
	case ui.InputPointerDown:
		if v.Mode() == vim.Visual || v.Mode() == vim.VisualLine {
			// A click leaves the visual selection to the editor, as Vim's
			// mouse does.
			v.Key(vim.KeyEscape, h)
			a.applyVimState(tab)
		}
		return false
	case ui.InputCommand:
		return false
	}
	return false
}

// vimSpecial maps a key the session names to its own Key.
func vimSpecial(ev ui.InputEvent) (vim.Key, bool) {
	switch ev.Key {
	case ui.KeyEscape:
		return vim.KeyEscape, true
	case ui.KeyEnter:
		return vim.KeyEnter, true
	case ui.KeyBackspace:
		return vim.KeyBackspace, true
	case ui.KeyDelete:
		return vim.KeyDelete, true
	case ui.KeyTab:
		return vim.KeyTab, true
	case ui.KeyLeft:
		return vim.KeyLeft, true
	case ui.KeyRight:
		return vim.KeyRight, true
	case ui.KeyUp:
		return vim.KeyUp, true
	case ui.KeyDown:
		return vim.KeyDown, true
	case ui.KeyHome:
		return vim.KeyHome, true
	case ui.KeyEnd:
		return vim.KeyEnd, true
	case ui.KeyPageUp:
		return vim.KeyPageUp, true
	case ui.KeyPageDown:
		return vim.KeyPageDown, true
	}
	return vim.KeyNone, false
}

// vimPassThrough reports whether a key is the platform's application
// shortcut, which stays the app's rather than the session's.
func (a *App) vimPassThrough(ev ui.InputEvent) bool {
	if runtime.GOOS == "darwin" {
		// Command is the app's; Vim takes Control, which is not.
		return ev.Mods&ui.Super != 0
	}
	if ev.Mods&ui.Super != 0 {
		return true
	}
	if ev.Mods&ui.Ctrl == 0 {
		return false
	}
	switch ev.Key {
	case ui.KeyN, ui.KeyO, ui.KeyS, ui.KeyW, ui.KeyComma, ui.KeyB, ui.KeyBackslash,
		ui.KeyZ, ui.KeyX, ui.KeyC, ui.KeyV, ui.KeyA, ui.KeyY:
		return true
	case ui.KeyF:
		return ev.Mods&ui.Shift != 0
	}
	return false
}

// vimModeChanged turns the setting on or off for every open tab.
func (a *App) vimModeChanged() {
	for _, p := range []*Pane{&a.left, &a.right} {
		for _, tab := range p.tabs {
			if a.Settings.VimMode {
				if tab.vim == nil {
					tab.vim = vim.New()
				}
				tab.vim.Reset()
				sel := tab.state.Selection()
				tab.state.SetSelections(editor.Selection{Anchor: sel.Head, Head: sel.Head})
				a.configureVim(tab, tab.state)
				continue
			}
			tab.vim = nil
			tab.state.OnInput = nil
			tab.state.NoInputMethod = false
			tab.state.CursorShape = editor.CursorBar
		}
	}
	a.update()
}

// vimModeLabel names a mode as the status bar shows it.
func vimModeLabel(m vim.Mode) string {
	switch m {
	case vim.Insert:
		return locales.VimInsert()
	case vim.Visual:
		return locales.VimVisual()
	case vim.VisualLine:
		return locales.VimVisualLine()
	}
	return locales.VimNormal()
}

// vimModeBadge shows NORMAL, INSERT or VISUAL beside the pane's name.
func (a *App) vimModeBadge(c *ui.Context, t *ui.Theme, tab *Tab) {
	color := t.Accent
	switch tab.vim.Mode() {
	case vim.Insert:
		color = t.Success
	case vim.Visual, vim.VisualLine:
		color = t.Warning
	}
	ui.Text(c, vimModeLabel(tab.vim.Mode())).FontSize(11).TextColor(color).
		Tooltip(locales.VimMode())
}

// openVimCommand opens the : or / line at the bottom of the window.
func (a *App) openVimCommand(kind rune) {
	a.vimCmdOpen = true
	a.vimCmdKind = kind
	a.vimCmdValue = ""
	a.vimCmdFocus = true
	a.vimCmdTab = a.activeTab()
	a.update()
}

// closeVimCommand closes the command line and gives the keyboard back to
// the editor.
func (a *App) closeVimCommand() {
	a.vimCmdOpen = false
	a.vimCmdKind = 0
	a.vimCmdValue = ""
	a.vimCmdFocus = false
	a.vimCmdTab = nil
	a.focusEditor = true
	a.update()
}

// vimCommandModal draws the : or / line above the status bar.
func (a *App) vimCommandModal(c *ui.Context, t *ui.Theme) {
	if !a.vimCmdOpen {
		return
	}
	w, h := c.Size()
	width := max(w-20, 200)
	ui.Overlay(c, func() {
		bar := ui.Row(c).Absolute().Left(10).Top(h-statusBarHeight-38).Width(width).Height(30).
			AlignItems(ui.Center).Gap(8).Padding(0, 10).Radius(8).
			Background(t.Surface).Border(1, t.Border).
			Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.18))
		var in ui.Element
		bar.Children(func() {
			ui.Text(c, string(a.vimCmdKind)).FontSize(12).TextColor(t.Accent)
			in = ui.TextInputBase(c, &a.vimCmdValue).Grow(1).Label(locales.VimCommand()).
				Placeholder(locales.VimCommand())
			if a.vimCmdFocus {
				in.Focus()
				a.vimCmdFocus = false
			}
			if in.Submitted() {
				a.submitVimCommand()
			}
			if in.Shortcut(0, ui.KeyEscape) {
				a.closeVimCommand()
			}
		})
		if bar.Clicked() {
			in.Focus()
		}
		if bar.OverlayShortcut(0, ui.KeyEscape) {
			a.closeVimCommand()
		}
	})
}

// submitVimCommand runs what the : or / line holds.
func (a *App) submitVimCommand() {
	kind, value := a.vimCmdKind, a.vimCmdValue
	if kind == ':' {
		value = strings.TrimSpace(value)
	}
	tab := a.vimCmdTab
	a.vimCmdOpen = false
	a.vimCmdKind = 0
	a.vimCmdValue = ""
	a.vimCmdFocus = false
	a.vimCmdTab = nil
	a.focusEditor = true
	if tab != nil && tab.vim != nil {
		if _, _, ok := a.tabLocation(tab); ok {
			switch kind {
			case '/':
				a.runVimSearch(tab, value)
			case ':':
				a.runVimEx(tab, value)
			}
		}
	}
	a.update()
}

// runVimSearch searches this document with the editor's own finder, closes
// its bar, and leaves the caret at the start of the match.
func (a *App) runVimSearch(tab *Tab, query string) {
	if query == "" {
		return
	}
	tab.state.Find(query)
	count, _ := tab.state.FindMatches()
	if count == 0 {
		tab.state.Run(editor.Cancel)
		tab.vim.SetSearchRange(0, 0)
		a.SetToast(locales.SearchNoResults())
		return
	}
	sel := tab.state.Selection()
	tab.vim.SetSearchRange(sel.Start(), sel.End())
	tab.state.SetSelections(editor.Selection{Anchor: sel.Start(), Head: sel.Start()})
	tab.state.Run(editor.Cancel)
	tab.vim.Normal(&vimHost{a: a, tab: tab})
}

// runVimEx runs the handful of : commands the first Vim mode has.
func (a *App) runVimEx(tab *Tab, command string) {
	switch command {
	case "":
		return
	case "w":
		a.saveTab(tab, false, nil)
	case "q":
		if pane, index, ok := a.tabLocation(tab); ok {
			a.CloseTab(pane, index)
		}
	case "wq", "x":
		a.saveVimAndClose(tab)
	default:
		if n, err := strconv.Atoi(command); err == nil && n > 0 {
			tab.vim.GoToLine(&vimHost{a: a, tab: tab}, n)
		}
	}
}

// saveVimAndClose saves tab, then closes it, as :wq and :x do.
func (a *App) saveVimAndClose(tab *Tab) {
	a.saveTab(tab, false, func() {
		// The save runs with a.busy still set, so CloseTab would refuse;
		// the document is clean now, and removing it is what closing does.
		if pane, _, ok := a.tabLocation(tab); ok {
			a.removeTabByPointer(pane, tab)
			a.update()
		}
	})
}

// tabLocation finds the pane and index a tab is in.
func (a *App) tabLocation(tab *Tab) (int, int, bool) {
	for _, pane := range []int{0, 1} {
		p := a.pane(pane)
		for i, t := range p.tabs {
			if t == tab {
				return pane, i, true
			}
		}
	}
	return 0, 0, false
}
