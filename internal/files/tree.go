// Package files reads the directories a workspace sidebar shows.
package files

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Tree caches the children of the directories opened in the sidebar.
// Children are read on demand and cached until Refresh.
type Tree struct {
	root       string
	showHidden bool
	cache      map[string][]string
	dirs       map[string]bool
}

// New returns a tree rooted at root.
func New(root string, showHidden bool) *Tree {
	return &Tree{
		root:       root,
		showHidden: showHidden,
		cache:      map[string][]string{},
		dirs:       map[string]bool{},
	}
}

// Root is the workspace directory.
func (t *Tree) Root() string { return t.root }

// Name is the name of the workspace directory.
func (t *Tree) Name() string { return filepath.Base(t.root) }

// ShowHidden reports whether dotfiles are listed.
func (t *Tree) ShowHidden() bool { return t.showHidden }

// SetShowHidden changes whether dotfiles are listed and makes the next
// frame read the directories again.
func (t *Tree) SetShowHidden(v bool) {
	if t.showHidden == v {
		return
	}
	t.showHidden = v
	t.Refresh()
}

// Refresh forgets every directory read so far.
func (t *Tree) Refresh() {
	t.cache = map[string][]string{}
	t.dirs = map[string]bool{}
}

// Roots returns the entries at the top of the workspace.
func (t *Tree) Roots() []string {
	if items, ok := t.cache[t.root]; ok {
		return items
	}
	return t.load(t.root)
}

// Children returns the entries of dir, or nil for a file.
func (t *Tree) Children(dir string) []string {
	if !t.IsDir(dir) {
		return nil
	}
	if items, ok := t.cache[dir]; ok {
		return items
	}
	return t.load(dir)
}

// IsDir reports whether path is a directory.
func (t *Tree) IsDir(path string) bool {
	if v, ok := t.dirs[path]; ok {
		return v
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	t.dirs[path] = info.IsDir()
	return info.IsDir()
}

func (t *Tree) load(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.cache[dir] = nil
		return nil
	}
	items := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if shouldSkip(name, t.showHidden) {
			continue
		}
		path := filepath.Join(dir, name)
		t.dirs[path] = e.IsDir()
		items = append(items, path)
	}
	sort.Slice(items, func(i, j int) bool {
		di, dj := t.dirs[items[i]], t.dirs[items[j]]
		if di != dj {
			return di
		}
		return naturalLess(filepath.Base(items[i]), filepath.Base(items[j]))
	})
	t.cache[dir] = items
	return items
}

// naturalLess compares names case-insensitively and orders digit runs by
// their numeric value. Plain string comparison puts "10月" before "2月";
// natural ordering keeps "1月, 2月, ..., 10月, 11月".
func naturalLess(a, b string) bool {
	ar, br := []rune(strings.ToLower(a)), []rune(strings.ToLower(b))
	i, j := 0, 0
	for ; i < len(ar) && j < len(br); {
		if ar[i] >= '0' && ar[i] <= '9' && br[j] >= '0' && br[j] <= '9' {
			ai, bi := i, j
			for ai < len(ar) && ar[ai] == '0' {
				ai++
			}
			for bi < len(br) && br[bi] == '0' {
				bi++
			}
			an, bn := ai, bi
			for an < len(ar) && ar[an] >= '0' && ar[an] <= '9' {
				an++
			}
			for bn < len(br) && br[bn] >= '0' && br[bn] <= '9' {
				bn++
			}
			if an-ai != bn-bi {
				return an-ai < bn-bi
			}
			for k := 0; k < an-ai; k++ {
				if ar[ai+k] != br[bi+k] {
					return ar[ai+k] < br[bi+k]
				}
			}
			i, j = an, bn
			continue
		}
		if ar[i] != br[j] {
			return ar[i] < br[j]
		}
		i++
		j++
	}
	return len(ar)-i < len(br)-j
}

func shouldSkip(name string, showHidden bool) bool {
	switch name {
	case ".DS_Store", ".git", ".mygo", "node_modules":
		return true
	}
	if !showHidden && strings.HasPrefix(name, ".") {
		return true
	}
	return false
}
