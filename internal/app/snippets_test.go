package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helloxz/zpad/internal/settings"
	"github.com/helloxz/zpad/internal/snippets"
	"github.com/helloxz/zpad/locales"
)

// snippetApp is an app with one document and a snippet in it.
func snippetApp(t *testing.T) (*App, *Tab, []snippets.Group) {
	t.Helper()
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/s.txt", "one\ntwo"))
	tab := a.activeTab()
	if tab == nil {
		t.Fatal("no active tab")
	}
	if !a.loadSnippets() {
		t.Fatal("could not load snippets")
	}
	g, err := a.snippets.AddGroup("JavaScript")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.snippets.AddSnippet(g.ID, "two lines", "a()\nb()"); err != nil {
		t.Fatal(err)
	}
	if !a.saveSnippets() {
		t.Fatal("could not save snippets")
	}
	return a, tab, a.snippets.Groups()
}

func TestInsertSnippetAtCursor(t *testing.T) {
	a, tab, groups := snippetApp(t)
	tab.state.SetSelections(editor.Selection{Anchor: 3, Head: 3})
	a.insertSnippet(groups[0].Snippets[0])

	if got := tab.doc.Buffer.String(); got != "onea()\nb()\ntwo" {
		t.Fatalf("buffer %q", got)
	}
	if tab.state.Cursor() != len("onea()\nb()") {
		t.Fatalf("cursor %d", tab.state.Cursor())
	}
	if !tab.doc.Buffer.Modified() {
		t.Fatal("the buffer is not marked changed")
	}
}

func TestInsertSnippetReplacesSelection(t *testing.T) {
	a, tab, groups := snippetApp(t)
	// "one" is selected.
	tab.state.SetSelections(editor.Selection{Anchor: 0, Head: 3})
	a.insertSnippet(groups[0].Snippets[0])

	if got := tab.doc.Buffer.String(); got != "a()\nb()\ntwo" {
		t.Fatalf("buffer %q", got)
	}
}

func TestInsertSnippetFollowsTheLineIndent(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/s.go", "func x() {\n    \n}\n"))
	tab := a.activeTab()
	if !a.loadSnippets() {
		t.Fatal("could not load snippets")
	}
	g, err := a.snippets.AddGroup("Go")
	if err != nil {
		t.Fatal(err)
	}
	sn, err := a.snippets.AddSnippet(g.ID, "call", "a()\nb()")
	if err != nil {
		t.Fatal(err)
	}

	// The caret sits on the indented, otherwise empty line.
	tab.state.SetSelections(editor.Selection{Anchor: 15, Head: 15})
	a.insertSnippet(sn)

	want := "func x() {\n    a()\n    b()\n}\n"
	if got := tab.doc.Buffer.String(); got != want {
		t.Fatalf("buffer %q, want %q", got, want)
	}
}

func TestInsertSnippetAtEveryCaret(t *testing.T) {
	a, tab, groups := snippetApp(t)
	tab.state.SetSelections(
		editor.Selection{Anchor: 0, Head: 0},
		editor.Selection{Anchor: 7, Head: 7},
	)
	a.insertSnippet(groups[0].Snippets[0])
	if got := tab.doc.Buffer.String(); got != "a()\nb()one\ntwoa()\nb()" {
		t.Fatalf("buffer %q", got)
	}
	if n := len(tab.state.Selections()); n != 2 {
		t.Fatalf("%d selections", n)
	}
}

func TestInsertSnippetUndoesInOneStep(t *testing.T) {
	a, tab, groups := snippetApp(t)
	before := tab.doc.Buffer.String()
	tab.state.SetSelections(editor.Selection{Anchor: 3, Head: 3})
	a.insertSnippet(groups[0].Snippets[0])
	if !tab.doc.Buffer.Undo() {
		t.Fatal("the insertion is not undoable")
	}
	if got := tab.doc.Buffer.String(); got != before {
		t.Fatalf("buffer after undo %q", got)
	}
}

func TestInsertSnippetIsRefusedWhereItCannotLand(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(a *App, tab *Tab)
		why   string
	}{
		{
			name:  "vim mode",
			setup: func(a *App, _ *Tab) { a.Settings.VimMode = true },
			why:   locales.SnippetNoVim(),
		},
		{
			name:  "read-only",
			setup: func(_ *App, tab *Tab) { tab.doc.ReadOnly = true },
			why:   locales.SnippetReadOnly(),
		},
		{
			name:  "preview",
			setup: func(_ *App, tab *Tab) { tab.preview = true },
			why:   locales.SnippetNoPreview(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, tab, groups := snippetApp(t)
			tc.setup(a, tab)
			before := tab.doc.Buffer.String()
			a.insertSnippet(groups[0].Snippets[0])
			if got := tab.doc.Buffer.String(); got != before {
				t.Fatalf("the buffer changed to %q", got)
			}
			if len(a.pendingToasts) != 1 {
				t.Fatalf("%d toasts", len(a.pendingToasts))
			}
			if got := a.pendingToasts[0].Description; got != tc.why {
				t.Fatalf("toast %q, want %q", got, tc.why)
			}
		})
	}
}

func TestInsertSnippetWithoutADocument(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	if !a.loadSnippets() {
		t.Fatal("could not load snippets")
	}
	g, _ := a.snippets.AddGroup("G")
	sn, _ := a.snippets.AddSnippet(g.ID, "n", "body")
	a.insertSnippet(sn)
	if len(a.pendingToasts) != 1 || a.pendingToasts[0].Description != locales.SnippetNoTab() {
		t.Fatalf("toasts %#v", a.pendingToasts)
	}
}

func TestSnippetPanelOpensOnAllAndCloses(t *testing.T) {
	a, _, _ := snippetApp(t)
	a.openSnippets()
	if !a.snippetsOpen {
		t.Fatal("the panel did not open")
	}
	if a.snippetsGroup != -1 {
		t.Fatalf("group %d: want the whole collection first", a.snippetsGroup)
	}
	if len(a.snippetsRows) != 1 {
		t.Fatalf("rows %#v", a.snippetsRows)
	}
	if a.snippetsSel != 0 {
		t.Fatalf("choice %d: want the first snippet", a.snippetsSel)
	}
	a.closeSnippets()
	if a.snippetsOpen || len(a.snippetsRows) != 0 || len(a.snippetCounts) != 0 {
		t.Fatal("closing kept the panel, its rows or its counts")
	}
}

func TestSnippetQueryFiltersRows(t *testing.T) {
	a, _, groups := snippetApp(t)
	g := groups[0]
	if _, err := a.snippets.AddSnippet(g.ID, "console.log", "console.log(x)"); err != nil {
		t.Fatal(err)
	}
	if !a.saveSnippets() {
		t.Fatal("could not save snippets")
	}
	a.openSnippets()
	a.snippetsQuery = "CONSOLE"
	a.refreshSnippetRows()
	if len(a.snippetsRows) != 1 {
		t.Fatalf("rows %#v", a.snippetsRows)
	}
	if sn, _, ok := a.selectedSnippet(); !ok || sn.Name != "console.log" {
		t.Fatalf("chosen %#v", sn)
	}

	// A query that matches the text of one snippet shows only that one.
	a.snippetsQuery = "b()"
	a.refreshSnippetRows()
	if sn, _, ok := a.selectedSnippet(); !ok || sn.Name != "two lines" {
		t.Fatalf("chosen by body %#v", sn)
	}

	a.snippetsQuery = "nothing here"
	a.refreshSnippetRows()
	if len(a.snippetsRows) != 0 {
		t.Fatalf("rows %#v", a.snippetsRows)
	}
	if a.snippetMatchTotal() != 0 {
		t.Fatalf("matches %d", a.snippetMatchTotal())
	}
}

func TestSnippetChoiceWrapsAround(t *testing.T) {
	a, _, groups := snippetApp(t)
	g := groups[0]
	if _, err := a.snippets.AddSnippet(g.ID, "second", "x"); err != nil {
		t.Fatal(err)
	}
	if !a.saveSnippets() {
		t.Fatal("could not save snippets")
	}
	a.openSnippets()
	if len(a.snippetsRows) != 2 {
		t.Fatalf("rows %#v", a.snippetsRows)
	}
	a.snippetsSel = 0
	a.moveSnippetSelection(1)
	if a.snippetsSel != 1 {
		t.Fatalf("choice %d", a.snippetsSel)
	}
	a.moveSnippetSelection(1) // wraps to the first
	if a.snippetsSel != 0 {
		t.Fatalf("choice %d", a.snippetsSel)
	}
	a.moveSnippetSelection(-1) // and back to the last
	if a.snippetsSel != 1 {
		t.Fatalf("choice %d", a.snippetsSel)
	}
}

func TestSnippetDialogAddsAndEdits(t *testing.T) {
	a, _, groups := snippetApp(t)
	g := groups[0]

	a.openSnippetDialog(g.ID, "")
	a.snippetEditName = "  fetch  json "
	a.snippetEditBody = "await fetch(url);\r\n"
	a.applySnippetEdit()
	if a.snippetEditOpen {
		t.Fatal("the dialog stayed open")
	}
	got, _ := a.snippets.Group(g.ID)
	if len(got.Snippets) != 2 {
		t.Fatalf("snippets %#v", got.Snippets)
	}
	added := got.Snippets[1]
	if added.Name != "fetch json" || added.Body != "await fetch(url);\n" {
		t.Fatalf("snippet %#v", added)
	}

	a.openSnippetDialog(g.ID, added.ID)
	if a.snippetEditName != "fetch json" {
		t.Fatalf("the dialog did not fill in %q", a.snippetEditName)
	}
	a.snippetEditName = "fetch"
	a.snippetEditBody = "await fetch(url)"
	a.applySnippetEdit()
	got, _ = a.snippets.Group(g.ID)
	if len(got.Snippets) != 2 || got.Snippets[1].Name != "fetch" ||
		got.Snippets[1].Body != "await fetch(url)" {
		t.Fatalf("snippets %#v", got.Snippets)
	}
}

func TestSnippetDialogShowsWhyItCannotSave(t *testing.T) {
	a, _, groups := snippetApp(t)
	g := groups[0]

	a.openSnippetDialog(g.ID, "")
	a.snippetEditName = ""
	a.applySnippetEdit()
	if a.snippetEditErr != locales.SnippetNameEmpty() {
		t.Fatalf("error %q", a.snippetEditErr)
	}

	a.snippetEditName = "two lines" // taken in the group
	a.snippetEditBody = "x"
	a.applySnippetEdit()
	if a.snippetEditErr != locales.SnippetNameTaken() {
		t.Fatalf("error %q", a.snippetEditErr)
	}

	a.snippetEditName = strings.Repeat("n", snippets.MaxNameRunes+1)
	a.snippetEditBody = "x"
	a.applySnippetEdit()
	if a.snippetEditErr != locales.SnippetNameTooLong() {
		t.Fatalf("error %q", a.snippetEditErr)
	}

	a.snippetEditName = "big"
	a.snippetEditBody = strings.Repeat("a", snippets.MaxBodyBytes+1)
	a.applySnippetEdit()
	if a.snippetEditErr != locales.SnippetBodyTooLarge(snippets.MaxBodyBytes/1024) {
		t.Fatalf("error %q", a.snippetEditErr)
	}
	if !a.snippetEditOpen {
		t.Fatal("the dialog closed on an error")
	}
	if got, _ := a.snippets.Group(g.ID); len(got.Snippets) != 1 {
		t.Fatalf("a refused snippet was added: %d", len(got.Snippets))
	}
}

func TestSnippetLimitThroughTheApp(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	if !a.loadSnippets() {
		t.Fatal("could not load snippets")
	}
	g, _ := a.snippets.AddGroup("G")
	for i := 0; i < snippets.MaxSnippets; i++ {
		if _, err := a.snippets.AddSnippet(g.ID, fmt.Sprintf("n%d", i), "x"); err != nil {
			t.Fatal(err)
		}
	}
	a.openSnippetDialog(g.ID, "")
	a.snippetEditName = "one more"
	a.snippetEditBody = "x"
	a.applySnippetEdit()
	if a.snippetEditErr != locales.SnippetTooMany(snippets.MaxSnippets) {
		t.Fatalf("error %q", a.snippetEditErr)
	}
}

func TestSnippetFromSelectionFillsTheDialog(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.js", "const x = 1;\nconsole.log(x);\n"))
	tab := a.activeTab()
	tab.state.SetSelections(editor.Selection{Anchor: 0, Head: 12}) // "const x = 1;"
	if !a.loadSnippets() {
		t.Fatal("could not load snippets")
	}
	a.openSnippetFromSelection()
	if !a.snippetEditOpen {
		t.Fatal("the dialog did not open")
	}
	if a.snippetEditBody != "const x = 1;" {
		t.Fatalf("body %q", a.snippetEditBody)
	}
	if a.snippetEditName != "const x = 1;" {
		t.Fatalf("name %q", a.snippetEditName)
	}
	if !a.snippetsOpen {
		t.Fatal("the panel is not behind the dialog")
	}
	if groups := a.snippets.Groups(); len(groups) != 1 || groups[0].Name != defaultSnippetGroup {
		t.Fatalf("groups %#v: want a default group", groups)
	}
	a.applySnippetEdit()
	groups := a.snippets.Groups()
	if len(groups[0].Snippets) != 1 {
		t.Fatalf("snippets %#v", groups[0].Snippets)
	}
}

func TestOpenSnippetFromSelectionNeedsOne(t *testing.T) {
	a, tab, _ := snippetApp(t)
	tab.state.SetSelections(editor.Selection{Anchor: 0, Head: 0})
	a.openSnippetFromSelection()
	if a.snippetEditOpen {
		t.Fatal("the dialog opened with nothing selected")
	}
	if len(a.pendingToasts) != 1 || a.pendingToasts[0].Description != locales.SnippetNoSelection() {
		t.Fatalf("toasts %#v", a.pendingToasts)
	}
}

func TestGroupDialogAddsAndRenames(t *testing.T) {
	a, _, groups := snippetApp(t)
	a.openGroupDialog("")
	a.groupEditName = "Markdown"
	a.applyGroupEdit()
	if len(a.snippets.Groups()) != 2 {
		t.Fatalf("groups %#v", a.snippets.Groups())
	}
	if a.snippetsGroup != 1 {
		t.Fatalf("the view is on group %d", a.snippetsGroup)
	}

	a.openGroupDialog(groups[0].ID)
	a.groupEditName = "Markdown"
	a.applyGroupEdit()
	if a.groupEditErr != locales.SnippetGroupNameTaken() {
		t.Fatalf("error %q", a.groupEditErr)
	}

	a.groupEditName = "JS"
	a.applyGroupEdit()
	if got, _ := a.snippets.Group(groups[0].ID); got.Name != "JS" {
		t.Fatalf("group %#v", got)
	}
}

func TestDeleteGroupAndSnippet(t *testing.T) {
	a, _, groups := snippetApp(t)
	g := groups[0]
	sn := g.Snippets[0]

	a.deleteSnippet(g.ID, sn.ID)
	got, _ := a.snippets.Group(g.ID)
	if len(got.Snippets) != 0 {
		t.Fatalf("snippets %#v", got.Snippets)
	}

	a.deleteGroup(g.ID)
	if len(a.snippets.Groups()) != 0 {
		t.Fatalf("groups %#v", a.snippets.Groups())
	}

	// The file on disk says the same.
	again, err := snippets.Load(a.SettingsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Groups()) != 0 {
		t.Fatalf("the file still holds %#v", again.Groups())
	}
}

func TestSnippetsAreReadAgainWhenThePanelOpens(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	st, err := snippets.Load(a.SettingsDir)
	if err != nil {
		t.Fatal(err)
	}
	g, _ := st.AddGroup("Shell")
	if _, err := st.AddSnippet(g.ID, "list", "ls -la"); err != nil {
		t.Fatal(err)
	}
	if err := st.Save(); err != nil {
		t.Fatal(err)
	}

	a.openSnippets()
	if a.snippets == nil || len(a.snippets.Groups()) != 1 {
		t.Fatalf("snippets %#v", a.snippets)
	}
	if got := a.snippets.Groups()[0].Snippets[0].Name; got != "list" {
		t.Fatalf("name %q", got)
	}
}

func TestSnippetsFileSitsInTheUserDirectory(t *testing.T) {
	dir := t.TempDir()
	a := New(settings.Default(), dir)
	a.openSnippets()
	a.openGroupDialog("")
	a.groupEditName = "G"
	a.applyGroupEdit()
	path := filepath.Join(dir, snippets.FileName)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("no file at %s: %v", path, err)
	}
}

func TestFoldContains(t *testing.T) {
	for _, tc := range []struct {
		text, needle string
		want         bool
	}{
		{"console.log", "console", true},
		{"Console.Log", "console.log", true},
		{"console.log", "log", true},
		{"console.log", "logx", false},
		{"中文片段", "片段", true},
		{"", "x", false},
		{"x", "", true},
	} {
		if got := foldContains(tc.text, tc.needle); got != tc.want {
			t.Errorf("foldContains(%q, %q) = %v, want %v", tc.text, tc.needle, got, tc.want)
		}
	}
}

func TestSnippetPreview(t *testing.T) {
	for _, tc := range []struct{ body, want string }{
		{"a()\nb()", "a()"},
		{"\nsecond", ""},
		{"\ttab", "tab"},
		{"  spaced  ", "spaced"},
		{"", ""},
	} {
		if got := snippetPreview(tc.body); got != tc.want {
			t.Errorf("snippetPreview(%q) = %q, want %q", tc.body, got, tc.want)
		}
	}
}

func TestSnippetShortcutOpensThePanel(t *testing.T) {
	a, _, _ := snippetApp(t)
	tt := ui.NewTester(a.View, 900, 700)
	for range 2 {
		tt.Frame()
	}
	tt.Key(ui.Cmd|ui.Alt, ui.KeyS)
	tt.Frame()
	if !a.snippetsOpen {
		t.Fatal("the shortcut did not open the panel")
	}
	if !tt.Focused(locales.SnippetSearchLabel()) {
		t.Fatalf("the search field is not focused; texts %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.snippetsOpen {
		t.Fatal("Escape did not close the panel")
	}
}

func TestSnippetToolbarButtonOpensThePanel(t *testing.T) {
	a, _, _ := snippetApp(t)
	tt := ui.NewTester(a.View, 900, 700)
	for range 2 {
		tt.Frame()
	}
	if err := tt.Click(locales.Snippets()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.snippetsOpen {
		t.Fatal("the toolbar button did not open the panel")
	}
}

func TestSnippetPanelListsAndInsertsByClick(t *testing.T) {
	a, tab, _ := snippetApp(t)
	tt := ui.NewTester(a.View, 900, 700)
	for range 2 {
		tt.Frame()
	}
	tt.Key(ui.Cmd|ui.Alt, ui.KeyS)
	tt.Frame()
	if !tt.HasText("two lines") || !tt.HasText("JavaScript") {
		t.Fatalf("the panel does not list the group and its snippet; texts %q", tt.Texts())
	}
	tab.state.SetSelections(editor.Selection{Anchor: 0, Head: 0})
	if err := tt.Click("two lines"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.snippetsOpen {
		t.Fatal("the panel stayed open after inserting")
	}
	if got := tab.doc.Buffer.String(); got != "a()\nb()one\ntwo" {
		t.Fatalf("buffer %q", got)
	}
}

func TestSnippetPanelAddsAGroup(t *testing.T) {
	a, _, _ := snippetApp(t)
	tt := ui.NewTester(a.View, 900, 700)
	for range 2 {
		tt.Frame()
	}
	tt.Key(ui.Cmd|ui.Alt, ui.KeyS)
	tt.Frame()
	if err := tt.Click(locales.SnippetNewGroup()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.groupEditOpen {
		t.Fatal("the group dialog did not open")
	}
	tt.Type("Markdown")
	tt.Frame()
	if err := tt.Click(locales.Save()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.groupEditOpen {
		t.Fatal("the group dialog stayed open")
	}
	if len(a.snippets.Groups()) != 2 {
		t.Fatalf("groups %#v", a.snippets.Groups())
	}
	if a.snippetsGroup != 1 {
		t.Fatalf("the panel is on group %d, want the one just added", a.snippetsGroup)
	}
}

func TestSnippetPanelCountsGroups(t *testing.T) {
	a, _, groups := snippetApp(t)
	g := groups[0]
	if _, err := a.snippets.AddSnippet(g.ID, "console.log", "console.log(x)"); err != nil {
		t.Fatal(err)
	}
	md, err := a.snippets.AddGroup("Markdown")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.snippets.AddSnippet(md.ID, "image", "![alt](url)"); err != nil {
		t.Fatal(err)
	}
	if !a.saveSnippets() {
		t.Fatal("could not save snippets")
	}

	a.openSnippets()
	if len(a.snippetsRows) != 3 {
		t.Fatalf("rows %#v: the all view shows every snippet", a.snippetsRows)
	}
	if a.snippetMatchTotal() != 3 {
		t.Fatalf("total %d", a.snippetMatchTotal())
	}
	if a.snippetMatchCount(0) != 2 || a.snippetMatchCount(1) != 1 {
		t.Fatalf("counts %#v", a.snippetCounts)
	}
}

func TestSnippetGroupFiltersTheList(t *testing.T) {
	a, _, _ := snippetApp(t)
	md, err := a.snippets.AddGroup("Markdown")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.snippets.AddSnippet(md.ID, "image", "![alt](url)"); err != nil {
		t.Fatal(err)
	}
	if !a.saveSnippets() {
		t.Fatal("could not save snippets")
	}
	a.openSnippets()
	if len(a.snippetsRows) != 2 {
		t.Fatalf("rows %#v", a.snippetsRows)
	}

	a.snippetsGroup = 1
	a.refreshSnippetRows()
	if len(a.snippetsRows) != 1 {
		t.Fatalf("rows %#v", a.snippetsRows)
	}
	sn, gi, ok := a.selectedSnippet()
	if !ok || sn.Name != "image" || gi != 1 {
		t.Fatalf("chosen %#v in group %d", sn, gi)
	}

	// The counts stay about the whole collection, not the group on view.
	if a.snippetMatchCount(0) != 1 || a.snippetMatchTotal() != 2 {
		t.Fatalf("counts %#v", a.snippetCounts)
	}
}

func TestNewSnippetUsesTheChosenSnippetsGroup(t *testing.T) {
	a, _, _ := snippetApp(t)
	md, err := a.snippets.AddGroup("Markdown")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.snippets.AddSnippet(md.ID, "image", "![alt](url)"); err != nil {
		t.Fatal(err)
	}
	if !a.saveSnippets() {
		t.Fatal("could not save snippets")
	}
	a.openSnippets()
	// The Markdown one is the second row of the all view.
	a.snippetsSel = 1
	a.newSnippet()
	if !a.snippetEditOpen {
		t.Fatal("the dialog did not open")
	}
	if a.snippetEditGroupID != md.ID {
		t.Fatalf("group %q, want the group of the chosen snippet %q", a.snippetEditGroupID, md.ID)
	}
}

func TestNewSnippetStartsWithTheDefaultGroup(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.js", "x"))
	a.openSnippets()
	a.newSnippet()
	if !a.snippetEditOpen {
		t.Fatal("the dialog did not open")
	}
	groups := a.snippets.Groups()
	if len(groups) != 1 || groups[0].Name != defaultSnippetGroup {
		t.Fatalf("groups %#v: want one named %q", groups, defaultSnippetGroup)
	}
	if a.snippetEditGroupID != groups[0].ID {
		t.Fatalf("group %q", a.snippetEditGroupID)
	}
}

func TestNewSnippetTakesTheSelection(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.js", "const x = 1;\nconsole.log(x);\n"))
	tab := a.activeTab()
	tab.state.SetSelections(editor.Selection{Anchor: 0, Head: 12})
	a.openSnippets()
	a.newSnippet()
	if a.snippetEditBody != "const x = 1;" {
		t.Fatalf("body %q", a.snippetEditBody)
	}
	if a.snippetEditName != "const x = 1;" {
		t.Fatalf("name %q", a.snippetEditName)
	}
}

func TestNewSnippetIgnoresAHugeSelection(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	big := strings.Repeat("a", snippets.MaxBodyBytes+10)
	a.addDocument(docAt("/tmp/a.txt", big))
	tab := a.activeTab()
	tab.state.SetSelections(editor.Selection{Anchor: 0, Head: len(big)})
	a.openSnippets()
	a.newSnippet()
	if a.snippetEditBody != "" {
		t.Fatalf("body %q: a selection over the limit was taken", a.snippetEditBody)
	}
}

func TestSnippetEmptyStatePointsAtOtherGroups(t *testing.T) {
	a, _, _ := snippetApp(t)
	md, err := a.snippets.AddGroup("Markdown")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.snippets.AddSnippet(md.ID, "image", "![alt](url)"); err != nil {
		t.Fatal(err)
	}
	if !a.saveSnippets() {
		t.Fatal("could not save snippets")
	}
	a.openSnippets()
	a.snippetsGroup = 0 // JavaScript
	a.snippetsQuery = "image"
	a.refreshSnippetRows()
	if len(a.snippetsRows) != 0 {
		t.Fatalf("rows %#v", a.snippetsRows)
	}
	others := a.snippetMatchTotal() - a.snippetMatchCount(a.snippetsGroup)
	if others != 1 {
		t.Fatalf("others %d", others)
	}
}

func TestDeleteChosenGroupFallsBackToAll(t *testing.T) {
	a, _, groups := snippetApp(t)
	a.openSnippets()
	a.snippetsGroup = 0
	a.deleteGroup(groups[0].ID)
	if a.snippetsGroup != -1 {
		t.Fatalf("group %d: want the all view", a.snippetsGroup)
	}
	if len(a.snippetsRows) != 0 {
		t.Fatalf("rows %#v", a.snippetsRows)
	}
}

func TestSnippetGroupClickShowsThatGroup(t *testing.T) {
	a, tab, _ := snippetApp(t)
	tt := ui.NewTester(a.View, 900, 700)
	for range 2 {
		tt.Frame()
	}
	tt.Key(ui.Cmd|ui.Alt, ui.KeyS)
	tt.Frame()
	if err := tt.Click("JavaScript"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.snippetsGroup != 0 {
		t.Fatalf("group %d: the click did not choose the group", a.snippetsGroup)
	}
	if tab.doc.Buffer.String() != "one\ntwo" {
		t.Fatalf("buffer %q: the click inserted a snippet", tab.doc.Buffer.String())
	}
}

func TestSnippetGroupHeaderRenamesTheChosenGroup(t *testing.T) {
	a, _, _ := snippetApp(t)
	tt := ui.NewTester(a.View, 900, 700)
	for range 2 {
		tt.Frame()
	}
	tt.Key(ui.Cmd|ui.Alt, ui.KeyS)
	tt.Frame()
	// The header's actions apply to the group chosen in the list.
	if err := tt.Click("JavaScript"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click(locales.SnippetRenameGroup()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.groupEditOpen {
		t.Fatal("the header's rename did not open the dialog")
	}
	if a.groupEditName != "JavaScript" {
		t.Fatalf("dialog name %q", a.groupEditName)
	}
	if a.groupEditID == "" {
		t.Fatal("the dialog does not rename the chosen group")
	}
}

func TestSnippetGroupHeaderActionsNeedAGroup(t *testing.T) {
	a, _, _ := snippetApp(t)
	tt := ui.NewTester(a.View, 900, 700)
	for range 2 {
		tt.Frame()
	}
	tt.Key(ui.Cmd|ui.Alt, ui.KeyS)
	tt.Frame()
	if a.snippetsGroup != -1 {
		t.Fatalf("group %d: want the all view", a.snippetsGroup)
	}
	// "All" is on view: the header's rename is dimmed and does nothing.
	if err := tt.Click(locales.SnippetRenameGroup()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.groupEditOpen {
		t.Fatal("the all view offered a rename")
	}
}

func TestSnippetOtherGroupsButtonShowsAll(t *testing.T) {
	a, _, _ := snippetApp(t)
	md, err := a.snippets.AddGroup("Markdown")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.snippets.AddSnippet(md.ID, "image", "![alt](url)"); err != nil {
		t.Fatal(err)
	}
	if !a.saveSnippets() {
		t.Fatal("could not save snippets")
	}
	tt := ui.NewTester(a.View, 900, 700)
	for range 2 {
		tt.Frame()
	}
	tt.Key(ui.Cmd|ui.Alt, ui.KeyS)
	tt.Frame()
	a.snippetsGroup = 0
	a.snippetsQuery = "image"
	a.refreshSnippetRows()
	tt.Frame()
	if !tt.HasText(locales.SnippetOtherGroups(1)) {
		t.Fatalf("no hint about the other groups; texts %q", tt.Texts())
	}
	if err := tt.Click(locales.SnippetShowAll()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.snippetsGroup != -1 {
		t.Fatalf("group %d: the button did not show all", a.snippetsGroup)
	}
	if len(a.snippetsRows) != 1 {
		t.Fatalf("rows %#v", a.snippetsRows)
	}
}

func TestSnippetRowEditButtonOpensTheDialog(t *testing.T) {
	a, tab, _ := snippetApp(t)
	tt := ui.NewTester(a.View, 900, 700)
	for range 2 {
		tt.Frame()
	}
	tt.Key(ui.Cmd|ui.Alt, ui.KeyS)
	tt.Frame()
	if err := tt.Click(locales.SnippetEditTitle()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.snippetEditOpen {
		t.Fatal("the edit button did not open the dialog")
	}
	if a.snippetEditName != "two lines" {
		t.Fatalf("name %q", a.snippetEditName)
	}
	if got := tab.doc.Buffer.String(); got != "one\ntwo" {
		t.Fatalf("buffer %q: the edit button inserted a snippet", got)
	}
}

func TestInsertSnippetKeepsTheTrailingNewlineItWasGiven(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/s.go", "func x() {\n    \n}\n"))
	tab := a.activeTab()
	if !a.loadSnippets() {
		t.Fatal("could not load snippets")
	}
	g, err := a.snippets.AddGroup("Go")
	if err != nil {
		t.Fatal(err)
	}
	sn, err := a.snippets.AddSnippet(g.ID, "block", "a()\nb()\n")
	if err != nil {
		t.Fatal(err)
	}

	tab.state.SetSelections(editor.Selection{Anchor: 15, Head: 15})
	a.insertSnippet(sn)

	// The snippet brings its own break, and takes the line's, so the block
	// stands on its own lines without leaving a blank one behind.
	want := "func x() {\n    a()\n    b()\n}\n"
	if got := tab.doc.Buffer.String(); got != want {
		t.Fatalf("buffer %q, want %q", got, want)
	}
	if got, wantCur := tab.state.Cursor(), len("func x() {\n    a()\n    b()\n"); got != wantCur {
		t.Fatalf("caret %d, want %d", got, wantCur)
	}
	// No line is left with trailing spaces.
	for i, line := range strings.Split(want, "\n") {
		if line != strings.TrimRight(line, " ") {
			t.Fatalf("line %d of %q keeps trailing spaces", i, line)
		}
	}
}

func TestInsertSnippetSplitsTheLineAfterIt(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/s.txt", "one"))
	tab := a.activeTab()
	if !a.loadSnippets() {
		t.Fatal("could not load snippets")
	}
	g, err := a.snippets.AddGroup("G")
	if err != nil {
		t.Fatal(err)
	}
	sn, err := a.snippets.AddSnippet(g.ID, "block", "x()\n")
	if err != nil {
		t.Fatal(err)
	}

	// Mid-line: the rest of the line continues below the block.
	tab.state.SetSelections(editor.Selection{Anchor: 1, Head: 1})
	a.insertSnippet(sn)
	if got := tab.doc.Buffer.String(); got != "ox()\nne" {
		t.Fatalf("buffer %q", got)
	}
}

func TestSnippetNameFromCutsALongLine(t *testing.T) {
	long := strings.Repeat("a", snippets.MaxNameRunes+20)
	got := snippetNameFrom(long)
	if n := len([]rune(got)); n != snippets.MaxNameRunes {
		t.Fatalf("name of %d characters, want %d", n, snippets.MaxNameRunes)
	}
}

func TestSnippetGroupSwitchKeepsTheFirstRowChosen(t *testing.T) {
	a, _, _ := snippetApp(t)
	md, err := a.snippets.AddGroup("Markdown")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.snippets.AddSnippet(md.ID, "image", "![alt](url)"); err != nil {
		t.Fatal(err)
	}
	if !a.saveSnippets() {
		t.Fatal("could not save snippets")
	}

	tt := ui.NewTester(a.View, 900, 700)
	for range 2 {
		tt.Frame()
	}
	tt.Key(ui.Cmd|ui.Alt, ui.KeyS)
	tt.Frame()
	if len(a.snippetsRows) != 2 || a.snippetsSel != 0 {
		t.Fatalf("rows %#v, choice %d", a.snippetsRows, a.snippetsSel)
	}

	// The chosen row was the first of the all view: the same index as the
	// first row of the group switched to, which the list used to follow by
	// its old item and so clear.
	a.snippetsGroup = 1
	a.refreshSnippetRows()
	if a.snippetsSel != 0 {
		t.Fatalf("choice %d right after the switch", a.snippetsSel)
	}
	for range 2 {
		tt.Frame()
	}
	if a.snippetsSel != 0 {
		t.Fatalf("choice %d after the frames: the list dropped it", a.snippetsSel)
	}
	sn, gi, ok := a.selectedSnippet()
	if !ok || sn.Name != "image" || gi != 1 {
		t.Fatalf("chosen %#v in group %d", sn, gi)
	}

	// Enter inserts it, so the choice is a working one.
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got := a.activeTab().doc.Buffer.String(); got != "![alt](url)one\ntwo" {
		t.Fatalf("buffer %q", got)
	}
}

func TestSnippetDeleteKeepsASelection(t *testing.T) {
	a, _, groups := snippetApp(t)
	g := groups[0]
	if _, err := a.snippets.AddSnippet(g.ID, "second", "x"); err != nil {
		t.Fatal(err)
	}
	if !a.saveSnippets() {
		t.Fatal("could not save snippets")
	}
	tt := ui.NewTester(a.View, 900, 700)
	for range 2 {
		tt.Frame()
	}
	tt.Key(ui.Cmd|ui.Alt, ui.KeyS)
	tt.Frame()

	a.snippetsSel = 1
	first, _, _ := a.snippetAt(0)
	a.deleteSnippet(g.ID, first.ID)
	for range 2 {
		tt.Frame()
	}
	if a.snippetsSel != 0 {
		t.Fatalf("choice %d after the delete", a.snippetsSel)
	}
	if sn, _, ok := a.selectedSnippet(); !ok || sn.Name != "second" {
		t.Fatalf("chosen %#v", sn)
	}
}

func TestSnippetGroupClickGivesTheKeyboardBack(t *testing.T) {
	a, tab, _ := snippetApp(t)
	md, err := a.snippets.AddGroup("Markdown")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.snippets.AddSnippet(md.ID, "image", "![alt](url)"); err != nil {
		t.Fatal(err)
	}
	if !a.saveSnippets() {
		t.Fatal("could not save snippets")
	}
	tt := ui.NewTester(a.View, 900, 700)
	for range 2 {
		tt.Frame()
	}
	tt.Key(ui.Cmd|ui.Alt, ui.KeyS)
	tt.Frame()

	if err := tt.Click("Markdown"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.snippetsGroup != 1 {
		t.Fatalf("group %d: the click did not choose the group", a.snippetsGroup)
	}
	if got := tab.doc.Buffer.String(); got != "one\ntwo" {
		t.Fatalf("buffer %q: the click inserted a snippet", got)
	}
	if !tt.Focused(locales.SnippetSearchLabel()) {
		t.Fatalf("the field did not take the keyboard back; texts %q", tt.Texts())
	}
}
