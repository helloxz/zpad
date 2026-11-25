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

// defaultSnippetGroup names the group a snippet goes into when the user has
// none: a fixed name, not the open document's language, so that the same
// group is used whatever file is in front.
const defaultSnippetGroup = "default"

// snippetRow is one row of the panel's list: a snippet, and the group it is
// in, which the "all" view needs to tell two of the same name apart.
type snippetRow struct {
	group   int
	snippet int
}

// openSnippets shows the snippet panel, reading the file again so that
// edits made to it by hand are seen. A panel already open only takes the
// keyboard back to its field.
func (a *App) openSnippets() { a.showSnippets() }

// showSnippets shows the panel and tells whether it is there. Opening it
// starts from the whole collection: "all" groups, no query.
func (a *App) showSnippets() bool {
	if a.snippetEditOpen || a.groupEditOpen {
		return false
	}
	if a.snippetsOpen {
		a.snippetsFocus = true
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
	a.snippetsFocus = true
	a.snippetsQuery = ""
	a.snippetsGroup = -1
	a.snippetsList = ui.ListState{}
	a.refreshSnippetRows()
	a.update()
	return true
}

// panelOpen reports whether something else already covers the window, so
// that a second panel does not stack on it.
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
	a.snippetsQuery = ""
	a.snippetsRows = nil
	a.snippetCounts = nil
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

// refreshSnippetRows rebuilds the list from the query and the chosen group,
// and counts the matches of each group for the column on the left. The
// snippet chosen stays chosen while it is still listed.
func (a *App) refreshSnippetRows() {
	keep := ""
	if sn, _, ok := a.selectedSnippet(); ok {
		keep = sn.ID
	}
	rows := a.snippetsRows[:0]
	counts := a.snippetCounts[:0]
	a.snippetsSel = -1
	if a.snippets == nil {
		a.snippetsRows, a.snippetCounts = rows, counts
		return
	}
	q := strings.ToLower(strings.TrimSpace(a.snippetsQuery))
	for gi, g := range a.snippets.Groups() {
		n := 0
		for si, sn := range g.Snippets {
			if q != "" && !foldContains(g.Name, q) &&
				!foldContains(sn.Name, q) && !foldContains(sn.Body, q) {
				continue
			}
			n++
			if a.snippetsGroup == -1 || a.snippetsGroup == gi {
				rows = append(rows, snippetRow{group: gi, snippet: si})
			}
		}
		counts = append(counts, n)
	}
	a.snippetsRows, a.snippetCounts = rows, counts
	if keep != "" {
		for i, row := range rows {
			if sn, ok := a.rowSnippet(row); ok && sn.ID == keep {
				a.snippetsSel = i
				break
			}
		}
	}
	if a.snippetsSel < 0 && len(rows) > 0 {
		a.snippetsSel = 0
	}
	// The list re-finds the item it remembers by key, and drops the choice
	// when that item is not among the new rows, as it is not after a group
	// switch: it would clear the choice just made here. Choose tells it that
	// this is the app's choice, which it keeps.
	a.snippetsList.Selected = &a.snippetsSel
	a.snippetsList.Choose(a.snippetsSel)
}

// rowSnippet returns the snippet a row stands for.
func (a *App) rowSnippet(row snippetRow) (snippets.Snippet, bool) {
	if a.snippets == nil {
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

// snippetAt returns the snippet row i shows, and its group.
func (a *App) snippetAt(i int) (snippets.Snippet, int, bool) {
	if i < 0 || i >= len(a.snippetsRows) {
		return snippets.Snippet{}, 0, false
	}
	row := a.snippetsRows[i]
	sn, ok := a.rowSnippet(row)
	if !ok {
		return snippets.Snippet{}, 0, false
	}
	return sn, row.group, true
}

// selectedSnippet returns the snippet the list's choice points at.
func (a *App) selectedSnippet() (snippets.Snippet, int, bool) {
	return a.snippetAt(a.snippetsSel)
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

// moveSnippetSelection steps the choice by dir, round the ends.
func (a *App) moveSnippetSelection(dir int) {
	n := len(a.snippetsRows)
	if n == 0 {
		return
	}
	sel := a.snippetsSel
	if sel < 0 || sel >= n {
		sel = 0
		if dir < 0 {
			sel = n
		}
	}
	sel = (sel + dir) % n
	if sel < 0 {
		sel += n
	}
	a.snippetsSel = sel
	a.snippetsList.ScrollIntoView(sel)
	a.update()
}

// snippetMatchCount is how many snippets of group i the query matches.
func (a *App) snippetMatchCount(i int) int {
	if i < 0 || i >= len(a.snippetCounts) {
		return 0
	}
	return a.snippetCounts[i]
}

// snippetMatchTotal is how many snippets the query matches, in all groups.
func (a *App) snippetMatchTotal() int {
	n := 0
	for _, c := range a.snippetCounts {
		n += c
	}
	return n
}

// insertSelectedSnippet puts the chosen snippet into the active tab.
func (a *App) insertSelectedSnippet() {
	if sn, _, ok := a.selectedSnippet(); ok {
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
// what is selected there, and closes the panel. The text is inserted as it
// is: a snippet meant to stand on its own lines ends with one, and an
// inline one does not, so nothing is added at its end.
func (a *App) insertSnippet(sn snippets.Snippet) {
	tab, why := a.snippetTarget()
	if tab == nil {
		a.SetErrorToast(locales.SnippetInsertFailed(), why)
		return
	}
	body := snippets.NormalizeBody(sn.Body)
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
		text := alignSnippet(buf, start, text)
		// A collapsed caret at the end of a line where the snippet brings
		// its own break takes that line's break too, so a block does not
		// leave a blank line before what followed it.
		if start == end && end < buf.Len() && strings.HasSuffix(text, "\n") &&
			lineEndsAt(buf, start) {
			end++
		}
		list = append(list, insertion{start: start, end: end, text: text})
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

// alignSnippet indents a snippet's lines after the first to the line the
// caret is on, as the JSON tool's output does. A snippet that ends with a
// newline, so that it stands on its own lines, keeps no indentation after
// that last one: the caret comes to rest at the start of the next line.
func alignSnippet(buf *editor.Buffer, start int, text string) string {
	out := alignToLine(buf, start, text)
	if !strings.HasSuffix(text, "\n") {
		return out
	}
	if prefix := lineIndent(buf, start); prefix != "" {
		out = strings.TrimSuffix(out, prefix)
	}
	return out
}

// lineEndsAt reports whether off is at the end of its line, so that the
// byte after it, when there is one, is that line's break.
func lineEndsAt(buf *editor.Buffer, off int) bool {
	p := buf.Pos(off)
	return p.Col >= len(buf.Line(p.Line))
}

// snippetSelection returns the text selected in the active tab, which is
// what a new snippet takes as its body.
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

// selectionForSnippet returns the text selected in the active tab, when it
// can become a snippet's body: a tab that takes edits, a text that is not
// empty, and one within the size the store allows.
func (a *App) selectionForSnippet() (string, bool) {
	tab := a.activeTab()
	if tab == nil || tab.preview || tab.doc.ReadOnly {
		return "", false
	}
	text := snippetSelection(tab)
	if strings.TrimSpace(text) == "" || len(text) > snippets.MaxBodyBytes {
		return "", false
	}
	return text, true
}

// newSnippet asks for a snippet in the chosen group. The text selected in
// the active tab, if there is any, is what it starts with: the selection is
// usually what the user wants to keep.
func (a *App) newSnippet() {
	if a.snippetEditOpen || a.groupEditOpen {
		return
	}
	if !a.snippetsOpen && !a.showSnippets() {
		return
	}
	group := a.newSnippetGroupID()
	if group == "" {
		return
	}
	a.openSnippetDialog(group, "")
	if text, ok := a.selectionForSnippet(); ok {
		a.snippetEditBody = snippets.NormalizeBody(text)
		a.snippetEditName = snippetNameFrom(text)
	}
	a.update()
}

// newSnippetGroupID is where a new snippet goes: the group of the snippet
// chosen, else the group on view, else the first one, or a new "default".
func (a *App) newSnippetGroupID() string {
	if _, gi, ok := a.selectedSnippet(); ok {
		if id := a.groupIDAt(gi); id != "" {
			return id
		}
	}
	if id := a.groupIDAt(a.snippetsGroup); id != "" {
		return id
	}
	return a.defaultGroupID()
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
	a.newSnippet()
}

// snippetNameFrom is the first line of a text, shortened to the length a
// name may take, as the name a new snippet starts with.
func snippetNameFrom(text string) string {
	line := text
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	line = strings.TrimSpace(line)
	runes := []rune(line)
	if len(runes) > snippets.MaxNameRunes {
		return string(runes[:snippets.MaxNameRunes])
	}
	return line
}

// defaultGroupID is the group a snippet made without choosing one goes
// into: the first group, or a new "default" one.
func (a *App) defaultGroupID() string {
	if a.snippets == nil {
		return ""
	}
	if groups := a.snippets.Groups(); len(groups) > 0 {
		return groups[0].ID
	}
	g, err := a.snippets.AddGroup(defaultSnippetGroup)
	if err != nil {
		return ""
	}
	a.saveSnippets()
	return g.ID
}

// groupAt returns the group at index i, where -1 is the "all" view.
func (a *App) groupAt(i int) (snippets.Group, bool) {
	if a.snippets == nil {
		return snippets.Group{}, false
	}
	groups := a.snippets.Groups()
	if i < 0 || i >= len(groups) {
		return snippets.Group{}, false
	}
	return groups[i], true
}

// groupIDAt is the id of the group at index i, "" for "all".
func (a *App) groupIDAt(i int) string {
	if g, ok := a.groupAt(i); ok {
		return g.ID
	}
	return ""
}

// groupNameAt is the name of the group at index i.
func (a *App) groupNameAt(i int) string {
	if g, ok := a.groupAt(i); ok {
		return g.Name
	}
	return ""
}

// selectGroup puts the panel on the group with id.
func (a *App) selectGroup(id string) {
	if a.snippets != nil {
		for i, g := range a.snippets.Groups() {
			if g.ID == id {
				a.snippetsGroup = i
				return
			}
		}
	}
	a.selectGroupIndex(a.snippetsGroup)
}

// selectGroupIndex keeps the chosen group within the groups that exist,
// where -1 is the "all" view.
func (a *App) selectGroupIndex(i int) {
	n := 0
	if a.snippets != nil {
		n = len(a.snippets.Groups())
	}
	if i < 0 || i >= n {
		i = -1
	}
	a.snippetsGroup = i
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

// applyGroupEdit adds or renames a group with the name in the dialog. A new
// group is shown at once, so its snippets can be added next.
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
//
// The group on view stays on view: an addition made from the "all" list does
// not throw the user into the group it went to.
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
	if a.snippetsGroup != -1 {
		a.selectGroup(groupID)
	}
	a.refreshSnippetRows()
	a.selectSnippetRow(id)
	a.snippetsFocus = true
	a.update()
}

// selectSnippetRow puts the list's choice on the snippet with id, when the
// rows show it.
func (a *App) selectSnippetRow(id string) {
	for i := range a.snippetsRows {
		if sn, _, ok := a.snippetAt(i); ok && sn.ID == id {
			a.snippetsSel = i
			a.snippetsList.ScrollIntoView(i)
			return
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
	chosen := ""
	if a.snippetsGroup >= 0 && a.groupIDAt(a.snippetsGroup) != id {
		chosen = a.groupIDAt(a.snippetsGroup)
	}
	if err := a.snippets.DeleteGroup(id); err != nil {
		a.SetErrorToast(locales.SnippetsSaveFailed(), snippetGroupErrorText(err))
		return
	}
	if !a.saveSnippets() {
		return
	}
	if chosen == "" {
		a.snippetsGroup = -1
	} else {
		a.selectGroup(chosen)
	}
	a.refreshSnippetRows()
	a.snippetsFocus = true
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
	a.snippetsFocus = true
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
