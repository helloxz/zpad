package app

import (
	"errors"
	"sort"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helloxz/zpad/internal/snippets"
	"github.com/helloxz/zpad/locales"
)

// snippetRow is one row of the snippet panel: a group's heading, or one of
// its snippets. The panel shows the groups in the store's order, and each
// one only when the query matches it or something in it.
type snippetRow struct {
	header  bool
	group   int
	snippet int
}

// openSnippets shows the snippet panel, reading the file again so that
// edits made to it by hand are seen.
func (a *App) openSnippets() { a.showSnippets(false) }

// openSnippetManager shows the same panel on its management view.
func (a *App) openSnippetManager() { a.showSnippets(true) }

// showSnippets shows the panel in one of its two views, reading the file
// when the panel was closed. A panel already open only switches view.
func (a *App) showSnippets(manage bool) bool {
	if a.snippetEditOpen || a.groupEditOpen {
		return false
	}
	if a.snippetsOpen {
		a.snippetsManage = manage
		if !manage {
			a.snippetsFocus = true
		}
		a.refreshSnippetRows()
		a.update()
		return true
	}
	if a.panelOpen() {
		return false
	}
	if !a.loadSnippets() {
		return false
	}
	a.snippetsOpen = true
	a.snippetsManage = manage
	a.snippetsFocus = !manage
	a.snippetsQuery = ""
	a.snippetsList = ui.ListState{}
	a.refreshSnippetRows()
	a.update()
	return true
}

// panelOpen reports whether something else already covers the window, so
// that a second panel does not stack on it. The snippet panel itself is
// not counted: its two views replace one another.
func (a *App) panelOpen() bool {
	return a.showSettings || a.aboutOpen || a.renameOpen || a.createOpen ||
		a.searchOpen || a.snippetEditOpen || a.groupEditOpen
}

// closeSnippets hides the panel and forgets the query. The rows go with
// it: nothing about the snippets stays held while the panel is closed. The
// list's own state is left alone here, since a click that inserts a
// snippet closes the panel during the frame, and the List reads it after;
// opening the panel starts the list afresh instead.
func (a *App) closeSnippets() {
	a.snippetsOpen = false
	a.snippetsManage = false
	a.snippetsQuery = ""
	a.snippetsRows = nil
	a.update()
}

// loadSnippets reads the snippet file into the app, or reports why it
// could not.
func (a *App) loadSnippets() bool {
	st, err := snippets.Load(a.SettingsDir)
	if err != nil {
		a.SetErrorToast(locales.SnippetsLoadFailed(), err.Error())
		return false
	}
	a.snippets = st
	a.selectGroupIndex(a.snippetsGroup)
	return true
}

// saveSnippets writes the snippets and tells whether it worked. It runs on
// the main thread, on a change the user made, never while they type in the
// editor: the file is small, and an unsaved change would be a lost one.
func (a *App) saveSnippets() bool {
	if a.snippets == nil {
		return true
	}
	if err := a.snippets.Save(); err != nil {
		a.SetErrorToast(locales.SnippetsSaveFailed(), err.Error())
		return false
	}
	return true
}

// refreshSnippetRows rebuilds the panel's rows from the query and puts the
// choice on the first snippet.
func (a *App) refreshSnippetRows() {
	rows := a.snippetsRows[:0]
	a.snippetsSel = -1
	if a.snippets == nil {
		a.snippetsRows = rows
		return
	}
	q := strings.ToLower(strings.TrimSpace(a.snippetsQuery))
	for gi, g := range a.snippets.Groups() {
		at := len(rows)
		for si, sn := range g.Snippets {
			if q != "" && !foldContains(g.Name, q) &&
				!foldContains(sn.Name, q) && !foldContains(sn.Body, q) {
				continue
			}
			rows = append(rows, snippetRow{group: gi, snippet: si})
		}
		if len(rows) == at {
			continue
		}
		// The heading goes in front of the group's rows.
		rows = append(rows, snippetRow{})
		copy(rows[at+1:], rows[at:])
		rows[at] = snippetRow{header: true, group: gi}
		if a.snippetsSel < 0 {
			a.snippetsSel = at + 1
		}
	}
	a.snippetsRows = rows
}

// foldContains reports whether text holds the lowercased needle, ignoring
// case. The needle is the query, folded once; text is walked as it is, so
// filtering does not copy every snippet's body on each keystroke.
func foldContains(text, needle string) bool {
	if needle == "" {
		return true
	}
	n := len(needle)
	for i := 0; i+n <= len(text); i++ {
		if strings.EqualFold(text[i:i+n], needle) {
			return true
		}
	}
	return false
}

// selectedSnippet returns the snippet the panel's choice points at.
func (a *App) selectedSnippet() (snippets.Snippet, bool) {
	if a.snippets == nil || a.snippetsSel < 0 || a.snippetsSel >= len(a.snippetsRows) {
		return snippets.Snippet{}, false
	}
	row := a.snippetsRows[a.snippetsSel]
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

// moveSnippetSelection steps the choice by dir, round the ends, skipping
// the group headings.
func (a *App) moveSnippetSelection(dir int) {
	n := len(a.snippetsRows)
	if n == 0 {
		return
	}
	start := a.snippetsSel
	if start < 0 || start >= n {
		start = 0
	}
	for i := 1; i <= n; i++ {
		row := (start + dir*i) % n
		if row < 0 {
			row += n
		}
		if !a.snippetsRows[row].header {
			a.snippetsSel = row
			a.snippetsList.ScrollIntoView(row)
			a.update()
			return
		}
	}
}

// insertSelectedSnippet puts the chosen snippet into the active tab.
func (a *App) insertSelectedSnippet() {
	if sn, ok := a.selectedSnippet(); ok {
		a.insertSnippet(sn)
	}
}

// snippetTarget returns the active tab, when a snippet can go into it, or
// the reason it cannot.
func (a *App) snippetTarget() (*Tab, string) {
	if a.Settings.VimMode {
		// Cells in normal mode take keys, not text: an insertion made
		// from a dialog would leave the modal state behind.
		return nil, locales.SnippetNoVim()
	}
	tab := a.activeTab()
	switch {
	case tab == nil:
		return nil, locales.SnippetNoTab()
	case tab.preview:
		return nil, locales.SnippetNoPreview()
	case tab.doc.ReadOnly:
		return nil, locales.SnippetReadOnly()
	}
	return tab, ""
}

// insertSnippet puts a snippet's text at the active tab's carets, replacing
// what is selected there, and closes the panel.
func (a *App) insertSnippet(sn snippets.Snippet) {
	tab, why := a.snippetTarget()
	if tab == nil {
		a.SetErrorToast(locales.SnippetInsertFailed(), why)
		return
	}
	body := strings.TrimRight(snippets.NormalizeBody(sn.Body), "\n")
	if strings.TrimSpace(body) == "" {
		a.SetErrorToast(locales.SnippetInsertFailed(), locales.SnippetBodyEmpty())
		return
	}
	insertSnippetText(tab, body)
	a.closeSnippets()
	a.focusEditor = true
	a.updateTitle()
	a.update()
}

// insertSnippetText replaces each selection of a tab with text, in one step
// of undo, and leaves a caret after each insertion. Lines after the first
// follow the indentation of the line they land in, as the JSON tool's
// output does.
func insertSnippetText(tab *Tab, text string) {
	buf := tab.doc.Buffer
	sels := tab.state.Selections()
	if len(sels) == 0 {
		return
	}
	type insertion struct {
		start, end int
		text       string
	}
	list := make([]insertion, 0, len(sels))
	for _, s := range sels {
		start, end := s.Start(), s.End()
		list = append(list, insertion{start: start, end: end, text: alignToLine(buf, start, text)})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].start > list[j].start })

	buf.Transact(func() {
		for _, in := range list {
			buf.Edit(editor.Edit{Start: in.start, End: in.end, Text: in.text})
		}
	})

	// An earlier insertion moves what follows it by the length it added,
	// so the carets are counted from the start.
	sort.Slice(list, func(i, j int) bool { return list[i].start < list[j].start })
	out := make([]editor.Selection, len(list))
	shift := 0
	for i, in := range list {
		at := in.start + shift + len(in.text)
		out[i] = editor.Selection{Anchor: at, Head: at}
		shift += len(in.text) - (in.end - in.start)
	}
	tab.state.SetSelections(out...)
}

// snippetSelection returns the text selected in the active tab, which is
// what "new snippet from selection" saves.
func snippetSelection(tab *Tab) string {
	parts := make([]string, 0, 4)
	for _, s := range tab.state.Selections() {
		if s.Empty() {
			continue
		}
		parts = append(parts, tab.doc.Buffer.Slice(s.Start(), s.End()))
	}
	return strings.Join(parts, "\n")
}

// openSnippetFromSelection asks for the name of a snippet holding the text
// selected in the active tab.
func (a *App) openSnippetFromSelection() {
	tab := a.activeTab()
	if tab == nil || tab.preview || tab.doc.ReadOnly {
		return
	}
	text := snippetSelection(tab)
	if strings.TrimSpace(text) == "" {
		a.SetErrorToast(locales.SnippetFromSelection(), locales.SnippetNoSelection())
		return
	}
	if len(text) > snippets.MaxBodyBytes {
		a.SetErrorToast(locales.SnippetFromSelection(), locales.SnippetBodyTooLarge(snippets.MaxBodyBytes/1024))
		return
	}
	if a.panelOpen() {
		return
	}
	// The panel opens behind the dialog, so that cancelling the dialog
	// leaves the user where they were.
	if !a.showSnippets(false) {
		return
	}
	group := a.defaultGroupID()
	a.openSnippetDialog(group, "")
	a.snippetEditBody = snippets.NormalizeBody(text)
	a.snippetEditName = snippetNameFrom(text)
	a.refreshSnippetRows()
	a.update()
}

// snippetNameFrom is the first line of a text, shortened, as the name a new
// snippet starts with.
func snippetNameFrom(text string) string {
	line := text
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	line = strings.TrimSpace(line)
	runes := []rune(line)
	if len(runes) > 40 {
		return string(runes[:40])
	}
	return line
}

// defaultGroupID is the group a snippet made without choosing one goes
// into: the first group, or a new default one, named after the language of
// the active document when it has one.
func (a *App) defaultGroupID() string {
	if groups := a.snippets.Groups(); len(groups) > 0 {
		return groups[0].ID
	}
	name := locales.SnippetDefaultGroup()
	if tab := a.activeTab(); tab != nil && tab.doc.Language != nil {
		name = tab.doc.Language.Name
	}
	g, err := a.snippets.AddGroup(name)
	if err != nil {
		return ""
	}
	a.saveSnippets()
	a.selectGroup(g.ID)
	return g.ID
}

// selectGroup puts the management view on the group with id.
func (a *App) selectGroup(id string) {
	if a.snippets == nil {
		a.snippetsGroup = 0
		return
	}
	for i, g := range a.snippets.Groups() {
		if g.ID == id {
			a.snippetsGroup = i
			return
		}
	}
	a.selectGroupIndex(a.snippetsGroup)
}

// selectGroupIndex keeps the chosen group within the groups that exist.
func (a *App) selectGroupIndex(i int) {
	n := 0
	if a.snippets != nil {
		n = len(a.snippets.Groups())
	}
	if i < 0 || i >= n {
		i = 0
	}
	a.snippetsGroup = i
}

// chosenGroup returns the group the management view is on.
func (a *App) chosenGroup() (snippets.Group, int, bool) {
	if a.snippets == nil {
		return snippets.Group{}, 0, false
	}
	groups := a.snippets.Groups()
	if a.snippetsGroup < 0 || a.snippetsGroup >= len(groups) {
		return snippets.Group{}, 0, false
	}
	return groups[a.snippetsGroup], a.snippetsGroup, true
}

// openGroupDialog asks for a group's name: a new group when id is empty,
// else the one being renamed.
func (a *App) openGroupDialog(id string) {
	name := ""
	if id != "" {
		if g, ok := a.snippets.Group(id); ok {
			name = g.Name
		}
	}
	a.groupEditOpen = true
	a.groupEditID = id
	a.groupEditName = name
	a.groupEditErr = ""
	a.groupEditFocus = true
	a.update()
}

// applyGroupEdit adds or renames a group with the name in the dialog.
func (a *App) applyGroupEdit() {
	if a.snippets == nil {
		return
	}
	var id string
	var err error
	if a.groupEditID == "" {
		var g snippets.Group
		g, err = a.snippets.AddGroup(a.groupEditName)
		id = g.ID
	} else {
		id = a.groupEditID
		err = a.snippets.RenameGroup(id, a.groupEditName)
	}
	if err != nil {
		a.groupEditErr = snippetGroupErrorText(err)
		return
	}
	if !a.saveSnippets() {
		return
	}
	a.groupEditOpen = false
	a.groupEditErr = ""
	a.selectGroup(id)
	a.refreshSnippetRows()
	a.update()
}

// openSnippetDialog asks for a snippet's name and text. An empty id adds
// one to groupID.
func (a *App) openSnippetDialog(groupID, id string) {
	if a.snippets == nil {
		return
	}
	name, body := "", ""
	groupID = a.existingGroupID(groupID)
	if id != "" {
		if g, ok := a.snippets.Group(groupID); ok {
			for _, sn := range g.Snippets {
				if sn.ID == id {
					name, body = sn.Name, sn.Body
					break
				}
			}
		}
	}
	a.snippetEditOpen = true
	a.snippetEditGroupID = groupID
	a.snippetEditID = id
	a.snippetEditName = name
	a.snippetEditBody = body
	a.snippetEditErr = ""
	a.snippetEditFocus = true
	a.update()
}

// existingGroupID returns a group id that is in the store, making the
// default group when there is none.
func (a *App) existingGroupID(id string) string {
	if _, ok := a.snippets.Group(id); ok {
		return id
	}
	return a.defaultGroupID()
}

// applySnippetEdit adds or rewrites a snippet with what the dialog holds.
func (a *App) applySnippetEdit() {
	if a.snippets == nil {
		return
	}
	groupID := a.snippetEditGroupID
	if groupID == "" {
		groupID = a.defaultGroupID()
		if groupID == "" {
			a.snippetEditErr = locales.SnippetGroupNotFound()
			return
		}
	}
	var id string
	var err error
	if a.snippetEditID == "" {
		var sn snippets.Snippet
		sn, err = a.snippets.AddSnippet(groupID, a.snippetEditName, a.snippetEditBody)
		id = sn.ID
	} else {
		id = a.snippetEditID
		err = a.snippets.UpdateSnippet(groupID, id, a.snippetEditName, a.snippetEditBody)
	}
	if err != nil {
		a.snippetEditErr = snippetErrorText(err)
		return
	}
	if !a.saveSnippets() {
		return
	}
	a.snippetEditOpen = false
	a.snippetEditErr = ""
	a.selectGroup(groupID)
	a.refreshSnippetRows()
	a.selectSnippetRow(id)
	a.update()
}

// selectSnippetRow puts the panel's choice on the snippet with id, when the
// rows show it.
func (a *App) selectSnippetRow(id string) {
	groups := a.snippets.Groups()
	for i, row := range a.snippetsRows {
		if row.header {
			continue
		}
		if row.group >= 0 && row.group < len(groups) {
			snips := groups[row.group].Snippets
			if row.snippet >= 0 && row.snippet < len(snips) && snips[row.snippet].ID == id {
				a.snippetsSel = i
				a.snippetsList.ScrollIntoView(i)
				return
			}
		}
	}
}

// confirmDeleteGroup removes a group, asking first when it has snippets in
// it: the snippets go with it, and there is no undo.
func (a *App) confirmDeleteGroup(id string) {
	if a.snippets == nil {
		return
	}
	g, ok := a.snippets.Group(id)
	if !ok {
		return
	}
	if len(g.Snippets) == 0 {
		a.deleteGroup(id)
		return
	}
	go func() {
		res, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:             a.win,
			Type:               mygo.MessageWarning,
			Message:            locales.SnippetDeleteGroupQuestion(g.Name),
			Detail:             locales.SnippetDeleteGroupDetail(len(g.Snippets)),
			Buttons:            []string{locales.Cancel(), locales.Delete()},
			DefaultButton:      0,
			CancelButton:       1,
			DestructiveButtons: []int{1},
		})
		if err != nil || res.Button != 1 {
			return
		}
		a.onUI(func() { a.deleteGroup(id) })
	}()
}

// deleteGroup removes a group without asking.
func (a *App) deleteGroup(id string) {
	if a.snippets == nil {
		return
	}
	if err := a.snippets.DeleteGroup(id); err != nil {
		a.SetErrorToast(locales.SnippetsSaveFailed(), snippetGroupErrorText(err))
		return
	}
	if !a.saveSnippets() {
		return
	}
	a.selectGroupIndex(a.snippetsGroup)
	a.refreshSnippetRows()
	a.update()
}

// confirmDeleteSnippet removes a snippet, asking first: what it holds is
// the user's text, and there is no undo for it.
func (a *App) confirmDeleteSnippet(groupID, id string) {
	if a.snippets == nil {
		return
	}
	g, ok := a.snippets.Group(groupID)
	if !ok {
		return
	}
	name := ""
	for _, sn := range g.Snippets {
		if sn.ID == id {
			name = sn.Name
			break
		}
	}
	if name == "" {
		return
	}
	go func() {
		res, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:             a.win,
			Type:               mygo.MessageWarning,
			Message:            locales.SnippetDeleteSnippetQuestion(name),
			Buttons:            []string{locales.Cancel(), locales.Delete()},
			DefaultButton:      0,
			CancelButton:       1,
			DestructiveButtons: []int{1},
		})
		if err != nil || res.Button != 1 {
			return
		}
		a.onUI(func() { a.deleteSnippet(groupID, id) })
	}()
}

// deleteSnippet removes a snippet without asking.
func (a *App) deleteSnippet(groupID, id string) {
	if a.snippets == nil {
		return
	}
	if err := a.snippets.DeleteSnippet(groupID, id); err != nil {
		a.SetErrorToast(locales.SnippetsSaveFailed(), snippetErrorText(err))
		return
	}
	if !a.saveSnippets() {
		return
	}
	a.refreshSnippetRows()
	a.update()
}

// snippetErrorText says what to fix when a snippet change is refused.
func snippetErrorText(err error) string {
	switch {
	case errors.Is(err, snippets.ErrNameEmpty):
		return locales.SnippetNameEmpty()
	case errors.Is(err, snippets.ErrNameTaken):
		return locales.SnippetNameTaken()
	case errors.Is(err, snippets.ErrNameTooLong):
		return locales.SnippetNameTooLong()
	case errors.Is(err, snippets.ErrBodyEmpty):
		return locales.SnippetBodyEmpty()
	case errors.Is(err, snippets.ErrBodyTooLarge):
		return locales.SnippetBodyTooLarge(snippets.MaxBodyBytes / 1024)
	case errors.Is(err, snippets.ErrTooManySnippets):
		return locales.SnippetTooMany(snippets.MaxSnippets)
	case errors.Is(err, snippets.ErrGroupNotFound):
		return locales.SnippetGroupNotFound()
	}
	return locales.SnippetNotFound()
}

// snippetGroupErrorText is snippetErrorText for a group's name.
func snippetGroupErrorText(err error) string {
	switch {
	case errors.Is(err, snippets.ErrNameEmpty):
		return locales.SnippetNameEmpty()
	case errors.Is(err, snippets.ErrNameTaken):
		return locales.SnippetGroupNameTaken()
	case errors.Is(err, snippets.ErrNameTooLong):
		return locales.SnippetNameTooLong()
	case errors.Is(err, snippets.ErrGroupNotFound):
		return locales.SnippetGroupNotFound()
	}
	return locales.SnippetNotFound()
}
