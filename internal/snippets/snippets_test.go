package snippets

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	g, err := s.AddGroup("JavaScript")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddSnippet(g.ID, "console.log", "console.log($0);\r\n"); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	again, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	groups := again.Groups()
	if len(groups) != 1 || groups[0].Name != "JavaScript" {
		t.Fatalf("groups %#v", groups)
	}
	snips := groups[0].Snippets
	if len(snips) != 1 || snips[0].Name != "console.log" {
		t.Fatalf("snippets %#v", snips)
	}
	if snips[0].Body != "console.log($0);\n" {
		t.Fatalf("body %q: the CRLF was not folded into LF", snips[0].Body)
	}
	if snips[0].ID == "" || groups[0].ID == "" {
		t.Fatal("ids were not written")
	}
}

func TestMissingFileIsAnEmptyStore(t *testing.T) {
	s, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Groups()) != 0 || s.Count() != 0 {
		t.Fatalf("groups %#v", s.Groups())
	}
}

func TestCorruptFileIsKeptAside(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Groups()) != 0 {
		t.Fatal("a corrupt file did not come back empty")
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("the corrupt file was not kept: %v", err)
	}
}

func TestLoadRepairsHandEditedFile(t *testing.T) {
	dir := t.TempDir()
	body, err := json.Marshal(fileFormat{Groups: []Group{
		{Name: "  SQL  ", Snippets: []Snippet{
			{Name: "select", Body: "select *\r\nfrom t;"},
			{Name: "  ", Body: "nameless"},
		}},
		{Name: "", Snippets: []Snippet{{Name: "x", Body: "y"}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), body, 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	groups := s.Groups()
	if len(groups) != 1 {
		t.Fatalf("groups %#v: the nameless group was kept", groups)
	}
	g := groups[0]
	if g.Name != "SQL" || g.ID == "" {
		t.Fatalf("group %#v", g)
	}
	if len(g.Snippets) != 1 {
		t.Fatalf("snippets %#v: the nameless snippet was kept", g.Snippets)
	}
	if g.Snippets[0].Body != "select *\nfrom t;" || g.Snippets[0].ID == "" {
		t.Fatalf("snippet %#v", g.Snippets[0])
	}
}

func TestAddRefusesEmptyAndDuplicateNames(t *testing.T) {
	s, _ := Load(t.TempDir())
	if _, err := s.AddGroup("  "); !errors.Is(err, ErrNameEmpty) {
		t.Fatalf("empty group name: %v", err)
	}
	g, err := s.AddGroup("Go")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddGroup("go"); !errors.Is(err, ErrNameTaken) {
		t.Fatalf("duplicate group name: %v", err)
	}
	if err := s.RenameGroup(g.ID, "GO"); err != nil {
		t.Fatalf("renaming a group to its own name: %v", err)
	}
	if _, err := s.AddSnippet(g.ID, "", "x"); !errors.Is(err, ErrNameEmpty) {
		t.Fatalf("empty snippet name: %v", err)
	}
	if _, err := s.AddSnippet(g.ID, "main", "  \n "); !errors.Is(err, ErrBodyEmpty) {
		t.Fatalf("empty snippet text: %v", err)
	}
	if _, err := s.AddSnippet(g.ID, "main", "func main() {}"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddSnippet(g.ID, "MAIN", "x"); !errors.Is(err, ErrNameTaken) {
		t.Fatalf("duplicate snippet name: %v", err)
	}
}

func TestAddRefusesBeyondTheLimits(t *testing.T) {
	s, _ := Load(t.TempDir())
	g, err := s.AddGroup("G")
	if err != nil {
		t.Fatal(err)
	}
	big := strings.Repeat("a", MaxBodyBytes+1)
	if _, err := s.AddSnippet(g.ID, "big", big); !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("oversized body: %v", err)
	}
	if _, err := s.AddSnippet(g.ID, "ok", strings.Repeat("a", MaxBodyBytes)); err != nil {
		t.Fatalf("a body of exactly the limit was refused: %v", err)
	}
	for i := 1; i < MaxSnippets; i++ {
		if _, err := s.AddSnippet(g.ID, "n"+strings.Repeat("x", i), "x"); err != nil {
			t.Fatalf("snippet %d: %v", i, err)
		}
	}
	if s.Count() != MaxSnippets {
		t.Fatalf("count %d", s.Count())
	}
	if _, err := s.AddSnippet(g.ID, "one more", "x"); !errors.Is(err, ErrTooManySnippets) {
		t.Fatalf("beyond the limit: %v", err)
	}
}

func TestUpdateAndDelete(t *testing.T) {
	s, _ := Load(t.TempDir())
	g, _ := s.AddGroup("G")
	sn, err := s.AddSnippet(g.ID, "first", "one")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateSnippet(g.ID, sn.ID, "second", "two\r\nlines"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Group(g.ID)
	if len(got.Snippets) != 1 || got.Snippets[0].Name != "second" ||
		got.Snippets[0].Body != "two\nlines" {
		t.Fatalf("snippet %#v", got.Snippets)
	}
	if err := s.UpdateSnippet(g.ID, "missing", "x", "y"); !errors.Is(err, ErrSnippetNotFound) {
		t.Fatalf("unknown snippet: %v", err)
	}
	if err := s.UpdateSnippet("missing", sn.ID, "x", "y"); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("unknown group: %v", err)
	}
	if err := s.DeleteSnippet(g.ID, sn.ID); err != nil {
		t.Fatal(err)
	}
	if s.Count() != 0 {
		t.Fatalf("count %d", s.Count())
	}
	if err := s.DeleteGroup(g.ID); err != nil {
		t.Fatal(err)
	}
	if len(s.Groups()) != 0 {
		t.Fatalf("groups %#v", s.Groups())
	}
	if err := s.DeleteGroup(g.ID); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("deleting a gone group: %v", err)
	}
}

func TestSaveWithoutDirKeepsNothing(t *testing.T) {
	s, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if s.Path() != "" {
		t.Fatalf("path %q", s.Path())
	}
	if _, err := s.AddGroup("G"); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
}
