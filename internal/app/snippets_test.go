package app

import (
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

func TestSnippetPanelOpensAndCloses(t *testing.T) {
	a, _, _ := snippetApp(t)
	a.openSnippets()
	if !a.snippetsOpen || a.snippetsManage {
		t.Fatalf("open %v manage %v", a.snippetsOpen, a.snippetsManage)
	}
	if len(a.snippetsRows) != 2 {
		t.Fatalf("rows %#v: want a heading and one snippet", a.snippetsRows)
	}
	if a.snippetsRows[0].header != true || a.snippetsRows[1].header {
		t.Fatalf("rows %#v", a.snippetsRows)
	}
	if a.snippetsSel != 1 {
		t.Fatalf("choice %d: want the first snippet", a.snippetsSel)
	}
	a.closeSnippets()
	if a.snippetsOpen || len(a.snippetsRows) != 0 {
		t.Fatal("closing kept the panel or its rows")
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
	if len(a.snippetsRows) != 2 {
		t.Fatalf("rows %#v", a.snippetsRows)
	}
	if sn, ok := a.selectedSnippet(); !ok || sn.Name != "console.log" {
		t.Fatalf("chosen %#v", sn)
	}

	// A query that matches the text of one snippet shows only that one.
	a.snippetsQuery = "b()"
	a.refreshSnippetRows()
	if sn, ok := a.selectedSnippet(); !ok || sn.Name != "two lines" {
		t.Fatalf("chosen by body %#v", sn)
	}

	a.snippetsQuery = "nothing here"
	a.refreshSnippetRows()
	if len(a.snippetsRows) != 0 {
		t.Fatalf("rows %#v", a.snippetsRows)
	}
}

func TestSnippetRowsSkipHeadingsWhenMoving(t *testing.T) {
	a, _, groups := snippetApp(t)
	g := groups[0]
	if _, err := a.snippets.AddSnippet(g.ID, "second", "x"); err != nil {
		t.Fatal(err)
	}
	if !a.saveSnippets() {
		t.Fatal("could not save snippets")
	}
	a.openSnippets()
	// Rows: heading, "two lines", "second".
	a.snippetsSel = 1
	a.moveSnippetSelection(1)
	if a.snippetsSel != 2 {
		t.Fatalf("choice %d", a.snippetsSel)
	}
	a.moveSnippetSelection(1) // wraps over the heading
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
		if _, err := a.snippets.AddSnippet(g.ID, strings.Repeat("n", i+1), "x"); err != nil {
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
	if groups := a.snippets.Groups(); len(groups) != 1 || groups[0].Name != "JavaScript" {
		t.Fatalf("groups %#v: want a group named after the document", groups)
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

func TestSnippetManageViewAddsAGroup(t *testing.T) {
	a, _, _ := snippetApp(t)
	tt := ui.NewTester(a.View, 900, 700)
	for range 2 {
		tt.Frame()
	}
	tt.Key(ui.Cmd|ui.Alt, ui.KeyS)
	tt.Frame()
	if err := tt.Click(locales.ManageSnippets()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !a.snippetsManage {
		t.Fatal("the manager did not open")
	}
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
}
