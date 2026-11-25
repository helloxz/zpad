package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadMissingUsesDefaults(t *testing.T) {
	got, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if want := Default(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestRestoreSessionDefaultsOff(t *testing.T) {
	if Default().RestoreSession {
		t.Fatal("restore session is on by default")
	}
}

func TestKeepUnsavedDefaultsOff(t *testing.T) {
	if Default().KeepUnsaved {
		t.Fatal("keeping unsaved content is on by default")
	}
}

// Vim mode is experimental: a first launch and an upgrade leave it off.
func TestVimModeDefaultsOff(t *testing.T) {
	if Default().VimMode {
		t.Fatal("Vim mode is on by default")
	}
}

// Whitespace shows within selections by default, as VS Code and Sublime do.
func TestWhitespaceDefaultsToSelection(t *testing.T) {
	if got := Default().Whitespace; got != "selection" {
		t.Fatalf("whitespace %q by default, want selection", got)
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := Default()
	want.Theme = "dark"
	want.Language = "zh-Hans"
	want.FontSize = 16
	want.LineHeight = 1.7
	want.SoftWrap = true
	want.LineNumbers = "relative"
	want.TabSize = 2
	want.UseTabs = true
	want.ShowGuides = true
	want.Whitespace = "all"
	want.AutoClose = false
	want.DefaultEncoding = "gb18030"
	want.DefaultEOL = "\r\n"
	want.LargeFileMB = 50
	want.SidebarOpen = true
	want.SidebarWidth = 300
	want.ShowHidden = true
	want.RestoreSession = false
	want.ConfirmClose = false
	want.KeepUnsaved = true

	if err := Save(dir, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestEnsureWritesOnlyWhenMissing(t *testing.T) {
	dir := t.TempDir()
	s := Default()
	s.Theme = "light"
	if err := Ensure(dir, s); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Theme != "light" {
		t.Fatalf("theme %q, want light", got.Theme)
	}

	// A second call must not overwrite the file.
	if err := Ensure(dir, Default()); err != nil {
		t.Fatal(err)
	}
	got, err = Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Theme != "light" {
		t.Fatalf("theme %q after second Ensure, want light", got.Theme)
	}
}

func TestCorruptFileIsBackedUp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fileName)
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := Default(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want defaults %+v", got, want)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("backup not written: %v", err)
	}
}

func TestNormalizeClampsOutOfRangeValues(t *testing.T) {
	dir := t.TempDir()
	bad := Default()
	bad.FontSize = 99
	bad.LineHeight = -1
	bad.TabSize = 3
	bad.Whitespace = "everything"
	bad.DefaultEncoding = "latin-1"
	bad.DefaultEOL = "\r\r"
	bad.LargeFileMB = 999
	bad.SidebarWidth = 10

	if err := Save(dir, bad); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// Theme ids are checked by the app against the theme registry, so the
// settings file must keep them as written.
func TestThemeIDSurvivesLoad(t *testing.T) {
	dir := t.TempDir()
	s := Default()
	s.Theme = "dracula"
	if err := Save(dir, s); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Theme != "dracula" {
		t.Fatalf("theme %q, want dracula", got.Theme)
	}
}

func TestSavePermissions(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, Default()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, fileName))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("permissions %o, want 0600", perm)
	}
}
