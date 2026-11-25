package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helloxz/zpad/internal/settings"
	"github.com/helloxz/zpad/locales"
)

// diffApp opens documents with the given texts in one window, in order,
// leaving the last active.
func diffApp(t *testing.T, texts ...string) (*App, []*Document) {
	t.Helper()
	dir := t.TempDir()
	a := New(settings.Default(), dir)
	docs := make([]*Document, 0, len(texts))
	for i, text := range texts {
		doc := docAt(filepath.Join(dir, fmt.Sprintf("file-%d.txt", i+1)), text)
		a.addDocument(doc)
		docs = append(docs, doc)
	}
	return a, docs
}

// chooseCompare opens the panel for a pair, as the pickers would.
func chooseCompare(t *testing.T, a *App, x, y *Document) {
	t.Helper()
	a.openCompareText(nil)
	a.diffA, a.diffB = x, y
	a.update()
}

func TestComparePrefillsTwoTabs(t *testing.T) {
	a, docs := diffApp(t, "one", "two")
	a.openCompareText(nil)
	if !a.diffOpen {
		t.Fatal("the comparison panel did not open")
	}
	if a.diffA != docs[1] || a.diffB != docs[0] {
		t.Fatalf("prefilled A=%v B=%v, want the active tab and the next one",
			a.diffA, a.diffB)
	}
}

func TestCompareReadsUnsavedBuffers(t *testing.T) {
	a, docs := diffApp(t, "a\nb\n", "a\nc\n")
	// The first tab has an unsaved edit: the comparison reads the buffer,
	// not the file or a recovery copy.
	docs[0].Buffer.Set("a\nz\n")
	chooseCompare(t, a, docs[0], docs[1])
	a.runCompare()

	if a.diff == nil {
		t.Fatal("no diff after Compare")
	}
	added, removed := a.diff.Stats()
	if added != 1 || removed != 1 {
		t.Fatalf("stats +%d -%d, want +1 -1", added, removed)
	}
	if hunks := len(a.diff.Hunks()); hunks != 1 {
		t.Fatalf("%d hunks, want 1", hunks)
	}
	if got := docs[0].Buffer.String(); got != "a\nz\n" {
		t.Fatalf("the document changed to %q", got)
	}
}

func TestCompareIdenticalSkipsTheViewer(t *testing.T) {
	a, docs := diffApp(t, "same\n", "same\n")
	chooseCompare(t, a, docs[0], docs[1])
	a.runCompare()

	if !a.diffIdentical || a.diff != nil {
		t.Fatalf("identical %v, diff %v; want no viewer", a.diffIdentical, a.diff)
	}
	tt := ui.NewTester(a.View, 1100, 800)
	tt.Frame()
	if !tt.HasText(locales.CompareIdentical(1)) {
		t.Fatalf("no identical message; texts %q", tt.Texts())
	}
}

func TestCompareRefusesEmptySide(t *testing.T) {
	a, docs := diffApp(t, "x", "")
	chooseCompare(t, a, docs[0], docs[1])
	want := locales.CompareEmptySide(docs[1].Name())
	if got := a.diffProblem(); got != want {
		t.Fatalf("problem %q, want %q", got, want)
	}
	a.runCompare()
	if a.diff != nil || a.diffIdentical {
		t.Fatal("an empty side was compared")
	}
	tt := ui.NewTester(a.View, 1100, 800)
	tt.Frame()
	if !tt.HasText(want) {
		t.Fatalf("no empty-side hint; texts %q", tt.Texts())
	}
}

func TestCompareRefusesTooLargeSide(t *testing.T) {
	a, docs := diffApp(t, strings.Repeat("x", diffMaxBytes+1), "y")
	chooseCompare(t, a, docs[0], docs[1])
	want := locales.CompareTooLarge(docs[0].Name(), locales.CompareLimit())
	if got := a.diffProblem(); got != want {
		t.Fatalf("problem %q, want %q", got, want)
	}
	a.runCompare()
	if a.diff != nil {
		t.Fatal("an oversized side was compared")
	}
}

func TestCompareRefusesTheSameTab(t *testing.T) {
	a, docs := diffApp(t, "one", "two")
	chooseCompare(t, a, docs[0], docs[1])
	a.diffB = a.diffA
	if got := a.diffProblem(); got != locales.CompareSameTab() {
		t.Fatalf("problem %q, want the same-tab hint", got)
	}
}

func TestCompareForgetsAClosedSide(t *testing.T) {
	a, docs := diffApp(t, "a\nb\n", "a\nc\n")
	chooseCompare(t, a, docs[0], docs[1])
	a.runCompare()
	if a.diff == nil {
		t.Fatal("no diff before closing a side")
	}

	a.removeTab(0, 0) // file-1.txt
	if a.diffA != nil || a.diff != nil || a.diffIdentical {
		t.Fatalf("closed side kept: A=%v diff=%v identical=%v",
			a.diffA, a.diff, a.diffIdentical)
	}
	if !a.diffOpen {
		t.Fatal("closing a side closed the panel too")
	}
}

func TestCompareMenuOpensThePanel(t *testing.T) {
	a, _ := diffApp(t, "one", "two")
	tt := ui.NewTester(a.View, 1100, 800)
	tt.Frame()

	if err := tt.Click(locales.ToolsMenu()); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(locales.CompareText()); err != nil {
		t.Fatal(err)
	}
	if !a.diffOpen {
		t.Fatal("the tools menu item did not open the comparison")
	}
	tt.Frame()
	if !tt.HasText(locales.CompareText()) {
		t.Fatalf("the panel is not on screen; texts %q", tt.Texts())
	}
	if err := tt.Click(locales.CompareRun()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.diff == nil {
		t.Fatal("Compare did not build a diff")
	}
	if !tt.HasText(locales.CompareStats(1, 1)) {
		t.Fatalf("no stats after Compare; texts %q", tt.Texts())
	}
}

func TestCompareIsOnlyInTheToolbarToolsMenu(t *testing.T) {
	a, _ := diffApp(t, "one", "two")
	tt := ui.NewTester(a.View, 1100, 800)
	tt.Frame()

	// The toolbar's menu offers the comparison.
	if err := tt.Click(locales.ToolsMenu()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !menuHas(tt.Menu(), locales.CompareText()) {
		t.Fatalf("toolbar tools menu %q does not offer %q", tt.Menu(), locales.CompareText())
	}
	if !menuHas(tt.Menu(), locales.Base64Group()) {
		t.Fatalf("toolbar tools menu %q lost its text tools", tt.Menu())
	}
	tt.CloseMenu()
	tt.Frame()

	// The editor's 工具 submenu keeps the selection tools only.
	tt.RightClickAt(450, 300)
	tt.Frame()
	if !menuHas(tt.Menu(), locales.ToolsMenu()) {
		t.Fatalf("editor menu %q has no tools submenu", tt.Menu())
	}
	// Choosing a path that runs into a submenu does not act; the absent
	// comparison is reported as no item, with no dialog opened.
	err := tt.ChooseMenuItem(locales.ToolsMenu(), locales.CompareText())
	if err == nil {
		t.Fatal("editor tools menu chose the text comparison")
	}
	if !strings.Contains(err.Error(), "has no item") {
		t.Fatalf("editor tools menu: %v", err)
	}
	if a.diffOpen {
		t.Fatal("the text comparison opened from the editor's tools menu")
	}
	tt.CloseMenu()
}

func TestCompareFooterFollowsTheDiff(t *testing.T) {
	a, docs := diffApp(t, "a\nb\n", "a\nc\n")
	chooseCompare(t, a, docs[0], docs[1])
	a.runCompare()

	tt := ui.NewTester(a.View, 1100, 800)
	tt.Frame()
	if !tt.HasText(locales.CompareStats(1, 1)) {
		t.Fatalf("no stats in the footer; texts %q", tt.Texts())
	}
	if !tt.HasText(locales.CompareHunks(1)) {
		t.Fatalf("no hunk count in the footer; texts %q", tt.Texts())
	}
}

func TestCompareSwapExchangesSides(t *testing.T) {
	a, docs := diffApp(t, "a\nb\n", "a\nc\n")
	chooseCompare(t, a, docs[0], docs[1])
	a.runCompare()
	if a.diff == nil {
		t.Fatal("no diff before swapping")
	}
	a.swapCompare()
	if a.diffA != docs[1] || a.diffB != docs[0] {
		t.Fatalf("A=%v B=%v after swapping", a.diffA, a.diffB)
	}
	if a.diff == nil {
		t.Fatal("swapping dropped the diff instead of comparing again")
	}
}

func TestCompareEscapeCloses(t *testing.T) {
	a, docs := diffApp(t, "a\nb\n", "a\nc\n")
	chooseCompare(t, a, docs[0], docs[1])
	a.runCompare()

	tt := ui.NewTester(a.View, 1100, 800)
	tt.Frame()
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.diffOpen {
		t.Fatal("Escape did not close the comparison")
	}
	if a.diff != nil {
		t.Fatal("the diff outlived the panel")
	}
}
