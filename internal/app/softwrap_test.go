package app

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helloxz/zpad/internal/settings"
	"github.com/helloxz/zpad/locales"
)

func TestStatusBarSoftWrapButtonTogglesTheSetting(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()

	if info, ok := tt.Access(locales.SoftWrap()); ok {
		t.Fatalf("the status bar shows the word wrap button with no tab open: %+v", info)
	}

	a.addDocument(docAt("/tmp/wrap.txt", "x"))
	tt.Frame()
	info, ok := tt.Access(locales.SoftWrap())
	if !ok || info.Disabled {
		t.Fatalf("the status bar has no word wrap button: %+v, found %v", info, ok)
	}
	if a.Settings.SoftWrap {
		t.Fatal("word wrap starts on")
	}

	if err := tt.Click(locales.SoftWrap()); err != nil {
		t.Fatal(err)
	}
	if !a.Settings.SoftWrap {
		t.Fatal("the button did not turn word wrap on")
	}
	tt.Frame()
	if st := a.activeTab().state; !st.SoftWrap {
		t.Fatal("the editor state did not follow word wrap on")
	}

	if err := tt.Click(locales.SoftWrap()); err != nil {
		t.Fatal(err)
	}
	if a.Settings.SoftWrap {
		t.Fatal("the button did not turn word wrap off")
	}
	tt.Frame()
	if st := a.activeTab().state; st.SoftWrap {
		t.Fatal("the editor state did not follow word wrap off")
	}
}

// A file past the wrapping threshold or a read-only one keeps its lines
// unwrapped whatever the setting says, so the button waits for another tab
// rather than pretend it works here.
func TestStatusBarSoftWrapButtonWaitsOnReadOnlyFiles(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	doc := docAt("/tmp/locked.txt", "x")
	doc.ReadOnly = true
	a.addDocument(doc)

	tt := ui.NewTester(a.View, 900, 600)
	tt.Frame()
	info, ok := tt.Access(locales.SoftWrap())
	if !ok {
		t.Fatal("the status bar has no word wrap button")
	}
	if !info.Disabled {
		t.Fatalf("the word wrap button is on for a read-only file: %+v", info)
	}
}

func TestViewMenuTogglesSoftWrap(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	item := findMenuItem(a.menuItems(), locales.SoftWrap())
	if item == nil {
		t.Fatalf("the menu bar has no %q item", locales.SoftWrap())
	}
	if item.Type != mygo.MenuItemCheckbox {
		t.Fatalf("%q is type %v, want a checkbox", locales.SoftWrap(), item.Type)
	}
	if item.Accelerator != "Alt+Z" {
		t.Fatalf("%q accelerator %q, want Alt+Z", locales.SoftWrap(), item.Accelerator)
	}
	if item.Checked {
		t.Fatalf("%q starts checked", locales.SoftWrap())
	}

	item.Click(item, nil)
	if !a.Settings.SoftWrap {
		t.Fatal("the item did not turn word wrap on")
	}
	if !item.Checked {
		t.Fatal("the item did not check itself")
	}

	item.Click(item, nil)
	if a.Settings.SoftWrap {
		t.Fatal("the item did not turn word wrap off")
	}
	if item.Checked {
		t.Fatal("the item did not uncheck itself")
	}

	// The switch in settings changes the setting behind the menu item's
	// back; the sync keeps the native check mark in step.
	a.Settings.SoftWrap = true
	a.syncSoftWrapMenu()
	if !item.Checked {
		t.Fatal("the item kept its old check mark")
	}
}
