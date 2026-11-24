package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTreeListsDirsFirstAndSkipsNoise(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "b.txt"))
	mustWrite(t, filepath.Join(root, "a.txt"))
	mustWrite(t, filepath.Join(root, ".hidden"))
	mustWrite(t, filepath.Join(root, ".DS_Store"))
	mustWrite(t, filepath.Join(root, ".git", "config"))
	mustWrite(t, filepath.Join(root, "node_modules", "x.js"))
	if err := os.Mkdir(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "src", "main.go"))

	tr := New(root, false)
	roots := tr.Roots()
	want := []string{filepath.Join(root, "src"), filepath.Join(root, "a.txt"), filepath.Join(root, "b.txt")}
	if len(roots) != len(want) {
		t.Fatalf("roots %v, want %v", roots, want)
	}
	for i := range want {
		if roots[i] != want[i] {
			t.Fatalf("roots %v, want %v", roots, want)
		}
	}
	if got := tr.Children(filepath.Join(root, "src")); len(got) != 1 || filepath.Base(got[0]) != "main.go" {
		t.Fatalf("src children %v", got)
	}
	if got := tr.Children(filepath.Join(root, "a.txt")); got != nil {
		t.Fatalf("file children %v, want nil", got)
	}
}

func TestTreeNaturalSort(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"11月", "2月", "1月", "10月", "9月", "a.txt", "a10.txt", "a2.txt"} {
		mustWrite(t, filepath.Join(root, name))
	}
	tr := New(root, false)
	want := []string{"1月", "2月", "9月", "10月", "11月", "a.txt", "a2.txt", "a10.txt"}
	roots := tr.Roots()
	if len(roots) != len(want) {
		t.Fatalf("roots %v, want %v", roots, want)
	}
	for i := range want {
		if filepath.Base(roots[i]) != want[i] {
			t.Fatalf("roots %v, want %v", roots, want)
		}
	}
}

func TestShowHidden(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".env"))
	mustWrite(t, filepath.Join(root, "visible.txt"))

	tr := New(root, false)
	if len(tr.Roots()) != 1 {
		t.Fatalf("hidden file shown: %v", tr.Roots())
	}
	tr.SetShowHidden(true)
	roots := tr.Roots()
	if len(roots) != 2 {
		t.Fatalf("roots %v, want 2 entries", roots)
	}
	// .git and node_modules stay hidden even with dotfiles on.
	mustWrite(t, filepath.Join(root, ".git", "config"))
	tr.Refresh()
	for _, p := range tr.Roots() {
		if filepath.Base(p) == ".git" {
			t.Fatal(".git should never be listed")
		}
	}
}

func mustWrite(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}
