package app

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helloxz/zpad/internal/settings"
	"github.com/helloxz/zpad/internal/texttools"
	"github.com/helloxz/zpad/locales"
)

func toolTab(t *testing.T, text string) (*App, *Tab) {
	t.Helper()
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", text))
	return a, a.activeTab()
}

func selectAll(tab *Tab, start, end int) {
	tab.state.SetSelections(editor.Selection{Anchor: start, Head: end})
}

func TestBase64EncodeSelection(t *testing.T) {
	a, tab := toolTab(t, "hello world")
	selectAll(tab, 0, 5)
	a.runTextTool(tab, toolBase64Encode)

	if got := tab.doc.Buffer.String(); got != "aGVsbG8= world" {
		t.Fatalf("text %q", got)
	}
	sels := tab.state.Selections()
	if len(sels) != 1 || tab.doc.Buffer.Slice(sels[0].Start(), sels[0].End()) != "aGVsbG8=" {
		t.Fatalf("selection %+v", sels)
	}
}

func TestBase64DecodeSelection(t *testing.T) {
	a, tab := toolTab(t, "aGVsbG8=")
	selectAll(tab, 0, len("aGVsbG8="))
	a.runTextTool(tab, toolBase64Decode)

	if got := tab.doc.Buffer.String(); got != "hello" {
		t.Fatalf("text %q", got)
	}
}

func TestMD5Selection(t *testing.T) {
	a, tab := toolTab(t, "abc")
	selectAll(tab, 0, 3)
	a.runTextTool(tab, toolMD5Lower32)

	if got := tab.doc.Buffer.String(); got != "900150983cd24fb0d6963f7d28e17f72" {
		t.Fatalf("text %q", got)
	}

	a2, tab2 := toolTab(t, "abc")
	selectAll(tab2, 0, 3)
	a2.runTextTool(tab2, toolMD5Upper16)
	if got := tab2.doc.Buffer.String(); got != "3CD24FB0D6963F7D" {
		t.Fatalf("text %q", got)
	}
}

func TestJSONFormatSelection(t *testing.T) {
	a, tab := toolTab(t, `{"a":1,"b":[2,3]}`)
	selectAll(tab, 0, tab.doc.Buffer.Len())
	a.runTextTool(tab, toolJSONFormat)

	want := "{\n  \"a\": 1,\n  \"b\": [\n    2,\n    3\n  ]\n}"
	if got := tab.doc.Buffer.String(); got != want {
		t.Fatalf("text %q, want %q", got, want)
	}
}

func TestJSONFormatKeepsLineIndent(t *testing.T) {
	a, tab := toolTab(t, "x\n  {\"a\":1}\ny")
	selectAll(tab, 4, 11)
	a.runTextTool(tab, toolJSONFormat)

	want := "x\n  {\n    \"a\": 1\n  }\ny"
	if got := tab.doc.Buffer.String(); got != want {
		t.Fatalf("text %q, want %q", got, want)
	}
}

func TestJSONMinifySelection(t *testing.T) {
	a, tab := toolTab(t, "{\n  \"a\": 1\n}")
	selectAll(tab, 0, tab.doc.Buffer.Len())
	a.runTextTool(tab, toolJSONMinify)

	if got := tab.doc.Buffer.String(); got != `{"a":1}` {
		t.Fatalf("text %q", got)
	}
}

func TestTextToolMultipleSelections(t *testing.T) {
	a, tab := toolTab(t, "hello world")
	tab.state.SetSelections(
		editor.Selection{Anchor: 0, Head: 5},
		editor.Selection{Anchor: 6, Head: 11},
	)
	a.runTextTool(tab, toolBase64Encode)

	if got := tab.doc.Buffer.String(); got != "aGVsbG8= d29ybGQ=" {
		t.Fatalf("text %q", got)
	}
	sels := tab.state.Selections()
	if len(sels) != 2 {
		t.Fatalf("selections %+v", sels)
	}
	for i, want := range []string{"aGVsbG8=", "d29ybGQ="} {
		if got := tab.doc.Buffer.Slice(sels[i].Start(), sels[i].End()); got != want {
			t.Fatalf("selection %d is %q, want %q", i, got, want)
		}
	}
}

func TestTextToolUndoIsOneStep(t *testing.T) {
	a, tab := toolTab(t, "hello world")
	tab.state.SetSelections(
		editor.Selection{Anchor: 0, Head: 5},
		editor.Selection{Anchor: 6, Head: 11},
	)
	a.runTextTool(tab, toolBase64Encode)
	if !tab.doc.Buffer.CanUndo() {
		t.Fatal("the tool did not make an undo step")
	}
	tab.doc.Buffer.Undo()
	if got := tab.doc.Buffer.String(); got != "hello world" {
		t.Fatalf("after one undo the text is %q", got)
	}
}

func TestTextToolWithoutSelection(t *testing.T) {
	a, tab := toolTab(t, "hello")
	tab.state.SetSelections(editor.Selection{Anchor: 2, Head: 2})
	a.runTextTool(tab, toolBase64Encode)

	if got := tab.doc.Buffer.String(); got != "hello" {
		t.Fatalf("text %q", got)
	}
	if len(a.pendingToasts) != 1 || a.pendingToasts[0].Title != locales.TextToolNoSelection() {
		t.Fatalf("toasts %+v", a.pendingToasts)
	}
}

func TestTextToolLimit(t *testing.T) {
	text := strings.Repeat("a", texttools.MaxInput+1)
	a, tab := toolTab(t, text)
	selectAll(tab, 0, len(text))
	a.runTextTool(tab, toolBase64Encode)

	if got := tab.doc.Buffer.String(); got != text {
		t.Fatal("the tool changed text over the limit")
	}
	if len(a.pendingToasts) != 1 || a.pendingToasts[0].Title != locales.TextToolTooLarge() {
		t.Fatalf("toasts %+v", a.pendingToasts)
	}
}

func TestTextToolReadOnly(t *testing.T) {
	a, tab := toolTab(t, "hello")
	tab.doc.ReadOnly = true
	tab.state.ReadOnly = true
	selectAll(tab, 0, 5)
	a.runTextTool(tab, toolBase64Encode)

	if got := tab.doc.Buffer.String(); got != "hello" {
		t.Fatalf("text %q", got)
	}
}

func TestTextToolDecodeBinaryKeepsText(t *testing.T) {
	a, tab := toolTab(t, "//4=")
	selectAll(tab, 0, 4)
	a.runTextTool(tab, toolBase64Decode)

	if got := tab.doc.Buffer.String(); got != "//4=" {
		t.Fatalf("text %q", got)
	}
	if len(a.pendingToasts) != 1 || a.pendingToasts[0].Title != locales.TextToolNotText() {
		t.Fatalf("toasts %+v", a.pendingToasts)
	}
}

func TestTextToolIsAtomicAcrossSelections(t *testing.T) {
	a, tab := toolTab(t, "aGVsbG8= ////")
	tab.state.SetSelections(
		editor.Selection{Anchor: 0, Head: 8},
		editor.Selection{Anchor: 9, Head: 13},
	)
	a.runTextTool(tab, toolBase64Decode)

	if got := tab.doc.Buffer.String(); got != "aGVsbG8= ////" {
		t.Fatalf("text %q", got)
	}
}

func TestTextToolInvalidJSON(t *testing.T) {
	a, tab := toolTab(t, `{"a":1,}`)
	selectAll(tab, 0, 8)
	a.runTextTool(tab, toolJSONFormat)

	if got := tab.doc.Buffer.String(); got != `{"a":1,}` {
		t.Fatalf("text %q", got)
	}
	if len(a.pendingToasts) != 1 || a.pendingToasts[0].Title != locales.TextToolJSONInvalid() {
		t.Fatalf("toasts %+v", a.pendingToasts)
	}
}

func TestTextToolUnchangedIsNoUndoStep(t *testing.T) {
	a, tab := toolTab(t, `{"a":1}`)
	tab.doc.Buffer.MarkSaved()
	selectAll(tab, 0, tab.doc.Buffer.Len())
	a.runTextTool(tab, toolJSONMinify)

	if tab.doc.Buffer.Modified() {
		t.Fatal("minifying an already compact value marked the document changed")
	}
	if tab.doc.Buffer.CanUndo() {
		t.Fatal("minifying an already compact value added an undo step")
	}
}

func TestLineNumbersSelection(t *testing.T) {
	a, tab := toolTab(t, "https://a\nhttps://b\n")
	selectAll(tab, 0, tab.doc.Buffer.Len())
	a.runLineNumbers(tab, ",")

	want := "01,https://a\n02,https://b\n"
	if got := tab.doc.Buffer.String(); got != want {
		t.Fatalf("text %q, want %q", got, want)
	}
	sels := tab.state.Selections()
	if len(sels) != 1 {
		t.Fatalf("selections %+v", sels)
	}
	if got := tab.doc.Buffer.Slice(sels[0].Start(), sels[0].End()); got != want {
		t.Fatalf("selection is %q, want %q", got, want)
	}
}

func TestLineNumbersPadsToTwoDigits(t *testing.T) {
	text := "a\nb\nc\nd\ne\nf\ng\nh"
	a, tab := toolTab(t, text)
	selectAll(tab, 0, tab.doc.Buffer.Len())
	a.runLineNumbers(tab, ",")

	want := "01,a\n02,b\n03,c\n04,d\n05,e\n06,f\n07,g\n08,h"
	if got := tab.doc.Buffer.String(); got != want {
		t.Fatalf("text %q, want %q", got, want)
	}
}

func TestLineNumbersSeparators(t *testing.T) {
	cases := []struct {
		sep  string
		want string
	}{
		{",", "01,a\n02,b"},
		{". ", "01. a\n02. b"},
		{") ", "01) a\n02) b"},
		{": ", "01: a\n02: b"},
	}
	for _, c := range cases {
		a, tab := toolTab(t, "a\nb")
		selectAll(tab, 0, tab.doc.Buffer.Len())
		a.runLineNumbers(tab, c.sep)

		if got := tab.doc.Buffer.String(); got != c.want {
			t.Errorf("sep %q: text %q, want %q", c.sep, got, c.want)
		}
	}
}

func TestLineNumbersPartialSelection(t *testing.T) {
	a, tab := toolTab(t, "x\nhttps://a\nhttps://b\ny")
	selectAll(tab, 2, tab.doc.Buffer.Len()-2)
	a.runLineNumbers(tab, ",")

	if got, want := tab.doc.Buffer.String(), "x\n01,https://a\n02,https://b\ny"; got != want {
		t.Fatalf("text %q, want %q", got, want)
	}
}

func TestLineNumbersContinuesAcrossSelections(t *testing.T) {
	a, tab := toolTab(t, "a\nb\nc\nd")
	tab.state.SetSelections(
		editor.Selection{Anchor: 0, Head: 1},
		editor.Selection{Anchor: 4, Head: 7},
	)
	a.runLineNumbers(tab, ",")

	if got, want := tab.doc.Buffer.String(), "01,a\nb\n02,c\n03,d"; got != want {
		t.Fatalf("text %q, want %q", got, want)
	}
}

func TestLineNumbersUndoIsOneStep(t *testing.T) {
	a, tab := toolTab(t, "a\nb\nc\nd")
	tab.state.SetSelections(
		editor.Selection{Anchor: 0, Head: 1},
		editor.Selection{Anchor: 4, Head: 7},
	)
	a.runLineNumbers(tab, ",")
	tab.doc.Buffer.Undo()

	if got := tab.doc.Buffer.String(); got != "a\nb\nc\nd" {
		t.Fatalf("after one undo the text is %q", got)
	}
}

func TestLineNumbersWithoutSelection(t *testing.T) {
	a, tab := toolTab(t, "a\nb")
	tab.state.SetSelections(editor.Selection{Anchor: 0, Head: 0})
	a.runLineNumbers(tab, ",")

	if got := tab.doc.Buffer.String(); got != "a\nb" {
		t.Fatalf("text %q", got)
	}
	if len(a.pendingToasts) != 1 || a.pendingToasts[0].Title != locales.TextToolNoSelection() {
		t.Fatalf("toasts %+v", a.pendingToasts)
	}
}

func TestLineNumbersReadOnly(t *testing.T) {
	a, tab := toolTab(t, "a\nb")
	tab.doc.ReadOnly = true
	tab.state.ReadOnly = true
	selectAll(tab, 0, tab.doc.Buffer.Len())
	a.runLineNumbers(tab, ",")

	if got := tab.doc.Buffer.String(); got != "a\nb" {
		t.Fatalf("text %q", got)
	}
}

func TestToolsMenuNumbersSelectedLines(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "a\nb"))
	tab := a.activeTab()
	selectAll(tab, 0, tab.doc.Buffer.Len())

	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()

	if err := tt.Click(locales.ToolsMenu()); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(locales.LineNumberGroup(), locales.LineNumberComma()); err != nil {
		t.Fatal(err)
	}
	if got, want := tab.doc.Buffer.String(), "01,a\n02,b"; got != want {
		t.Fatalf("text %q, want %q", got, want)
	}
}

func TestToolsMenuLineNumberStyles(t *testing.T) {
	cases := map[string]string{
		locales.LineNumberParen(): "01) a\n02) b",
		locales.LineNumberColon(): "01: a\n02: b",
	}
	for item, want := range cases {
		a := New(settings.Default(), t.TempDir())
		a.addDocument(docAt("/tmp/a.txt", "a\nb"))
		tab := a.activeTab()
		selectAll(tab, 0, tab.doc.Buffer.Len())

		tt := ui.NewTester(a.View, 900, 600)
		tt.Frame()
		if err := tt.Click(locales.ToolsMenu()); err != nil {
			t.Fatal(err)
		}
		if err := tt.ChooseMenuItem(locales.LineNumberGroup(), item); err != nil {
			t.Fatal(err)
		}
		if got := tab.doc.Buffer.String(); got != want {
			t.Fatalf("%q: text %q, want %q", item, got, want)
		}
	}
}

func TestToolsMenuDisablesLineNumbersWithoutSelection(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "a\nb"))
	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()

	if err := tt.Click(locales.ToolsMenu()); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(locales.LineNumberGroup(), locales.LineNumberComma()); err == nil {
		t.Fatal("line numbering is enabled without a selection")
	}
	tt.CloseMenu()
}

func TestToolsMenuRunsToolOnSelection(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "hello"))
	tab := a.activeTab()
	selectAll(tab, 0, 5)

	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()

	if err := tt.Click(locales.ToolsMenu()); err != nil {
		t.Fatal(err)
	}
	if !menuHas(tt.Menu(), locales.Base64Group()) {
		t.Fatalf("tools menu %q has no %q group", tt.Menu(), locales.Base64Group())
	}
	if err := tt.ChooseMenuItem(locales.Base64Group(), locales.Base64Encode()); err != nil {
		t.Fatal(err)
	}
	if got := tab.doc.Buffer.String(); got != "aGVsbG8=" {
		t.Fatalf("text %q", got)
	}
}

func TestToolsMenuDisablesToolsWithoutSelection(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "hello"))
	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()

	if err := tt.Click(locales.ToolsMenu()); err != nil {
		t.Fatal(err)
	}
	err := tt.ChooseMenuItem(locales.Base64Group(), locales.Base64Encode())
	if err == nil {
		t.Fatal("Base64 Encode is enabled without a selection")
	}
	tt.CloseMenu()
}

func TestEditorContextMenuHasTools(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", strings.Repeat("hello world ", 40)))
	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()

	// The secondary button keeps a selection only when it lands inside
	// one, so the first click places the caret, the test selects the word
	// there, and the second click opens the menu on it.
	tt.RightClickAt(400, 200)
	tt.CloseMenu()
	tab := a.activeTab()
	end := tab.state.Selections()[0].End()
	start := end - len("hello")
	if start < 0 {
		start = 0
	}
	tab.state.SetSelections(editor.Selection{Anchor: start, Head: end})
	before := tab.doc.Buffer.Slice(start, end)
	tt.Frame()
	tt.RightClickAt(400, 200)

	if !menuHas(tt.Menu(), locales.ToolsMenu()) {
		t.Fatalf("editor menu %q has no %q item", tt.Menu(), locales.ToolsMenu())
	}
	if err := tt.ChooseMenuItem(locales.ToolsMenu(), locales.Base64Group(), locales.Base64Encode()); err != nil {
		t.Fatal(err)
	}
	want := texttools.EncodeBase64(before)
	if got := tab.doc.Buffer.Slice(start, start+len(want)); got != want {
		t.Fatalf("text at the selection is %q, want %q", got, want)
	}
}
