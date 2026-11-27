package app

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helloxz/zpad/internal/settings"
	"github.com/helloxz/zpad/locales"
)

func manyTabs(t *testing.T, n int) (*App, *ui.Tester) {
	t.Helper()
	a := New(settings.Default(), t.TempDir())
	for i := 0; i < n; i++ {
		a.addDocument(docAt(fmt.Sprintf("/tmp/tab-%02d.txt", i), "x"))
	}
	tt := ui.NewTester(a.View, 700, 500)
	// Bounds report the last frame, so the bar needs a frame to learn it
	// overflows, and another to show the button.
	for range 4 {
		tt.Frame()
	}
	return a, tt
}

func TestTabBarOverflowShowsTabList(t *testing.T) {
	a, tt := manyTabs(t, 14)
	if !a.tabOverflow[0] {
		t.Fatal("tab bar did not notice that its tabs overflow")
	}
	if !tt.HasText(locales.AllTabs()) {
		t.Fatalf("no tab list button; texts %q", tt.Texts())
	}
}

// A name longer than the cap must not make one tab as wide as the bar, and
// the tab keeps the name for tooltips and the tab list.
func TestLongTabNameIsCapped(t *testing.T) {
	name := "a-very-long-file-name-that-would-take-the-whole-bar.component.test.tsx"
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/"+name, "x"))
	tt := ui.NewTester(a.View, 700, 500)
	tt.Frame()

	r, ok := tt.Find(name)
	if !ok {
		t.Fatalf("no tab for %q; texts %q", name, tt.Texts())
	}
	// The cap plus the tab's padding, gap and close button, about 240 DIP.
	if maxW := tabTextWidth + 60; r.W > maxW {
		t.Fatalf("tab is %.0f DIP wide, want at most %.0f", r.W, maxW)
	}
	if !tt.HasText(name) {
		t.Fatalf("tab lost the full name; texts %q", tt.Texts())
	}
}

func TestTabListChoosesAHiddenTab(t *testing.T) {
	a, tt := manyTabs(t, 14)
	if err := tt.Click(locales.AllTabs()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	last := a.left.tabs[len(a.left.tabs)-1]
	if err := tt.ChooseMenuItem(last.doc.Name()); err != nil {
		t.Fatal(err)
	}
	if a.left.active != len(a.left.tabs)-1 {
		t.Fatalf("chose tab %d, want %d", a.left.active, len(a.left.tabs)-1)
	}
	if a.revealDoc != "" {
		t.Fatalf("reveal %q still pending after the tab bar was built", a.revealDoc)
	}
}

func TestActivateTabChoosesAndReveals(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "a"))
	a.addDocument(docAt("/tmp/b.txt", "b"))
	a.activateTab(0, 0)
	if a.left.active != 0 {
		t.Fatalf("active tab %d, want 0", a.left.active)
	}
	if a.revealPane != 0 || a.revealDoc != a.left.tabs[0].doc.ID {
		t.Fatalf("reveal %d %q, want 0 %q", a.revealPane, a.revealDoc, a.left.tabs[0].doc.ID)
	}
	a.activateTab(0, 5) // out of range: nothing changes
	if a.left.active != 0 {
		t.Fatalf("out-of-range choice moved the active tab to %d", a.left.active)
	}
}

func TestToolbarButtonShowsHandCursor(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	tt := ui.NewTester(a.View, 1200, 800)
	r, ok := tt.Find(locales.NewFile())
	if !ok {
		t.Fatalf("no New File button; texts %q", tt.Texts())
	}
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	if got := tt.Cursor(); got != ui.CursorPointer {
		t.Fatalf("cursor over the button %v, want the pointing hand", got)
	}
}

// A middle press closes the tab under the pointer at once, as editors do,
// without activating it: MyGo's clicks are the primary button's, so the
// tab takes the press itself.
func TestMiddlePressClosesTab(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "a"))
	a.addDocument(docAt("/tmp/b.txt", "b"))
	tt := ui.NewTester(a.View, 700, 500)
	tt.Frame()
	r, ok := tt.Find("b.txt")
	if !ok {
		t.Fatalf("no tab for b.txt; texts %q", tt.Texts())
	}
	tt.PressButton(2, r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	if len(a.left.tabs) != 1 || a.left.tabs[0].doc.Name() != "a.txt" {
		t.Fatalf("middle press left %d tabs", len(a.left.tabs))
	}
	if a.left.active != 0 {
		t.Fatalf("middle press moved the active tab to %d, want 0", a.left.active)
	}
}

// A middle press beside the tabs, on the bar or its new tab button, closes
// nothing: only a tab takes it.
func TestMiddlePressOnTabBarClosesNothing(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "a"))
	tt := ui.NewTester(a.View, 700, 500)
	tt.Frame()
	r, ok := tt.Find("a.txt")
	if !ok {
		t.Fatalf("no tab for a.txt; texts %q", tt.Texts())
	}
	tt.PressButton(2, r.X+r.W+12, r.Y+r.H/2)
	tt.ReleaseButton(2, r.X+r.W+12, r.Y+r.H/2)
	tt.Frame()
	if len(a.left.tabs) != 1 {
		t.Fatalf("middle press on the bar left %d tabs, want 1", len(a.left.tabs))
	}
}

// Taking the middle press leaves the primary button's click alone: a left
// click still activates the tab, and closes nothing.
func TestLeftClickOnTabStillActivates(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "a"))
	a.addDocument(docAt("/tmp/b.txt", "b"))
	tt := ui.NewTester(a.View, 700, 500)
	tt.Frame()
	r, ok := tt.Find("b.txt")
	if !ok {
		t.Fatalf("no tab for b.txt; texts %q", tt.Texts())
	}
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	if len(a.left.tabs) != 2 {
		t.Fatalf("left click left %d tabs, want 2", len(a.left.tabs))
	}
	if a.left.active != 1 {
		t.Fatalf("left click chose tab %d, want 1", a.left.active)
	}
}

// The tab a middle press closes is the one under the pointer, in whichever
// pane it is: closing the other pane's tab keeps its neighbour there.
func TestMiddlePressClosesTheOtherPanesTab(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "a"))
	a.addDocument(docAt("/tmp/b.txt", "b"))
	a.toggleSplit()
	a.addDocument(docAt("/tmp/c.txt", "c"))
	if !a.split || len(a.left.tabs) != 2 || len(a.right.tabs) != 2 {
		t.Fatalf("split %v, left %d, right %d", a.split, len(a.left.tabs), len(a.right.tabs))
	}
	tt := ui.NewTester(a.View, 1000, 600)
	tt.Frame()
	r, ok := tt.Find("c.txt")
	if !ok {
		t.Fatalf("no tab for c.txt; texts %q", tt.Texts())
	}
	tt.PressButton(2, r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	if !a.split {
		t.Fatal("closing one of the right pane's tabs collapsed the split")
	}
	if len(a.right.tabs) != 1 || a.right.tabs[0].doc.Name() != "b.txt" {
		t.Fatalf("right pane has %d tabs after the middle press", len(a.right.tabs))
	}
	if len(a.left.tabs) != 2 {
		t.Fatalf("left pane has %d tabs, want its own 2", len(a.left.tabs))
	}
}

// doubleClick presses twice at the same point, as a double click does: the
// tester's clock stands still, so the presses count as one.
func doubleClick(tt *ui.Tester, x, y float32) {
	tt.ClickAt(x, y)
	tt.ClickAt(x, y)
}

// plusOnBar returns the new tab button of the tab bar at y, rather than the
// toolbar's button that shares its name.
func plusOnBar(t *testing.T, tt *ui.Tester, y float32) ui.Rect {
	t.Helper()
	for _, info := range tt.AccessAll() {
		if info.Role != ui.RoleButton || info.Name != locales.NewFile() {
			continue
		}
		if b := info.Bounds; b.Y > y-24 && b.Y < y+24 {
			return b
		}
	}
	t.Fatalf("no new tab button on the bar at y %.0f", y)
	return ui.Rect{}
}

// A double click on the bar's empty space adds a file, as browsers and
// editors do, and the new tab takes the focus.
func TestDoubleClickOnTabBarBlankCreatesFile(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "a"))
	tt := ui.NewTester(a.View, 900, 500)
	tt.Frame()
	r, ok := tt.Find("a.txt")
	if !ok {
		t.Fatalf("no tab for a.txt; texts %q", tt.Texts())
	}
	// The bar of one tab is empty well before its right edge.
	doubleClick(tt, 800, r.Y+r.H/2)
	tt.Frame()
	if len(a.left.tabs) != 2 {
		t.Fatalf("double click on the bar left %d tabs, want 2", len(a.left.tabs))
	}
	if a.left.active != 1 || a.left.tabs[1].doc.Path != "" {
		t.Fatalf("double click chose tab %d %q, want the new untitled",
			a.left.active, a.left.tabs[1].doc.Path)
	}
	if !a.left.tabs[1].state.Focused() {
		t.Fatal("the new file did not take the focus")
	}
}

// A single click on the bar's empty space keeps the tabs as they are: only
// a double click adds a file.
func TestSingleClickOnTabBarBlankCreatesNothing(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "a"))
	tt := ui.NewTester(a.View, 900, 500)
	tt.Frame()
	r, ok := tt.Find("a.txt")
	if !ok {
		t.Fatalf("no tab for a.txt; texts %q", tt.Texts())
	}
	tt.ClickAt(800, r.Y+r.H/2)
	tt.Frame()
	if len(a.left.tabs) != 1 {
		t.Fatalf("single click on the bar left %d tabs, want 1", len(a.left.tabs))
	}
}

// A double click on a tab is the tab's own: it chooses the tab and adds no
// file, as it does in the editors that take the gesture.
func TestDoubleClickOnTabCreatesNothing(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "a"))
	a.addDocument(docAt("/tmp/b.txt", "b"))
	tt := ui.NewTester(a.View, 900, 500)
	tt.Frame()
	r, ok := tt.Find("a.txt")
	if !ok {
		t.Fatalf("no tab for a.txt; texts %q", tt.Texts())
	}
	doubleClick(tt, r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	if len(a.left.tabs) != 2 {
		t.Fatalf("double click on a tab left %d tabs, want its 2", len(a.left.tabs))
	}
	if a.left.active != 0 {
		t.Fatalf("double click chose tab %d, want 0", a.left.active)
	}
}

// The new tab button keeps its own clicks: a double click on it adds one
// file, not one for the button and one for the bar behind it.
func TestDoubleClickOnNewTabButtonCreatesOneFile(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "a"))
	tt := ui.NewTester(a.View, 900, 500)
	tt.Frame()
	r, ok := tt.Find("a.txt")
	if !ok {
		t.Fatalf("no tab for a.txt; texts %q", tt.Texts())
	}
	plus := plusOnBar(t, tt, r.Y+r.H/2)
	doubleClick(tt, plus.X+plus.W/2, plus.Y+plus.H/2)
	tt.Frame()
	if len(a.left.tabs) != 2 {
		t.Fatalf("double click on the new tab button left %d tabs, want 2", len(a.left.tabs))
	}
}

// The bar's empty space adds the file to the pane that bar belongs to, and
// makes that pane the active one, even from the other pane.
func TestDoubleClickOnInactivePaneBarCreatesThere(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	a.addDocument(docAt("/tmp/a.txt", "a"))
	a.toggleSplit()
	a.addDocument(docAt("/tmp/b.txt", "b"))
	if !a.split || a.activePane != 1 {
		t.Fatalf("split %v, active pane %d; want the right one active", a.split, a.activePane)
	}
	left := len(a.left.tabs)
	right := len(a.right.tabs)
	tt := ui.NewTester(a.View, 1200, 500)
	tt.Frame()
	r, ok := tt.Find("b.txt")
	if !ok {
		t.Fatalf("no tab for b.txt; texts %q", tt.Texts())
	}
	// x 500 is in the left pane's bar, left of the split at about 600.
	doubleClick(tt, 500, r.Y+r.H/2)
	tt.Frame()
	if len(a.left.tabs) != left+1 {
		t.Fatalf("left pane has %d tabs, want %d", len(a.left.tabs), left+1)
	}
	if len(a.right.tabs) != right {
		t.Fatalf("right pane has %d tabs, want its %d", len(a.right.tabs), right)
	}
	if a.activePane != 0 {
		t.Fatalf("active pane %d after the click, want the left one", a.activePane)
	}
}
