// Package snippets stores the user's code snippets: named groups of text
// that the editor inserts at the cursor.
//
// The snippets are one JSON file in the user data directory, beside
// settings.json. It is written whole, through atomicfile, on each change;
// the store holds at most MaxSnippets snippets of at most MaxBodyBytes
// each, which bounds both the file and the memory it takes. The limits
// apply to what the app writes: a file edited by hand is read as it is.
package snippets

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/helloxz/zpad/internal/atomicfile"
)

const (
	// FileName is the file in the user data directory.
	FileName = "snippets.json"
	// Version is the format the store writes; a later one would migrate.
	Version = 1
	// MaxSnippets is the most snippets the store keeps, in all groups
	// together: a collection to scan by eye, not a snippet library.
	MaxSnippets = 100
	// MaxBodyBytes is the most bytes one snippet's text may take. A
	// snippet is a phrase or a few lines of code, not a document, and the
	// cap keeps the whole file small enough to hold in memory.
	MaxBodyBytes = 8 << 10
	// MaxNameRunes is the most characters a group or snippet name may take.
	// The column it is listed in shows about twenty, and a name is a label:
	// longer ones are cut in the list, and refused when one is typed.
	MaxNameRunes = 40
)

// The reasons a change is refused. A caller shows them in the dialog that
// made the change, so each one says what to fix.
var (
	ErrGroupNotFound   = errors.New("no such group")
	ErrSnippetNotFound = errors.New("no such snippet")
	ErrNameEmpty       = errors.New("the name is empty")
	ErrNameTaken       = errors.New("another one has that name")
	ErrNameTooLong     = errors.New("the name is too long")
	ErrBodyEmpty       = errors.New("the text is empty")
	ErrBodyTooLarge    = errors.New("the text is too long")
	ErrTooManySnippets = errors.New("the snippet limit is reached")
)

// Snippet is one piece of text and the name the user finds it by.
type Snippet struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Body string `json:"body"`
}

// Group is a named set of snippets, as the listing shows them.
type Group struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Snippets []Snippet `json:"snippets,omitempty"`
}

// fileFormat is what snippets.json holds.
type fileFormat struct {
	Version int     `json:"version"`
	Groups  []Group `json:"groups"`
}

// Store is the snippets and the directory they are saved in. Its methods
// are for the UI thread, as the settings' are: a change is written at
// once, and never while the user types.
type Store struct {
	dir    string
	groups []Group
}

// Load reads dir/snippets.json. A missing file is an empty store; a
// corrupt one is kept as snippets.json.bak, and the store comes back
// empty rather than refusing to start.
func Load(dir string) (*Store, error) {
	s := &Store{dir: dir}
	if dir == "" {
		return s, nil
	}
	path := filepath.Join(dir, FileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	var f fileFormat
	if err := json.Unmarshal(data, &f); err != nil {
		_ = os.WriteFile(path+".bak", data, 0o600)
		return s, nil
	}
	s.groups = f.Groups
	s.repair()
	return s, nil
}

// repair fills in what a file written by hand, or by an older version,
// leaves out: the ids names are matched by, and the line endings the
// editor's buffer takes. Only an item without a name goes, since it has
// nothing to show: a name that is now too long is kept, and editing it
// asks for a shorter one, because dropping it would lose what it holds.
func (s *Store) repair() {
	groups := s.groups[:0]
	for _, g := range s.groups {
		g.Name = normalizeName(g.Name)
		if g.Name == "" {
			continue
		}
		if g.ID == "" {
			g.ID = newID("g")
		}
		snips := g.Snippets[:0]
		for _, sn := range g.Snippets {
			sn.Name = normalizeName(sn.Name)
			if sn.Name == "" {
				continue
			}
			if sn.ID == "" {
				sn.ID = newID("s")
			}
			sn.Body = NormalizeBody(sn.Body)
			snips = append(snips, sn)
		}
		g.Snippets = snips
		groups = append(groups, g)
	}
	s.groups = groups
}

// Save writes the snippets atomically to dir/snippets.json.
func (s *Store) Save() error {
	if s.dir == "" {
		return nil
	}
	groups := s.groups
	if groups == nil {
		groups = []Group{}
	}
	data, err := json.MarshalIndent(fileFormat{Version: Version, Groups: groups}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicfile.Write(s.dir, FileName, data)
}

// Path is the file the store saves to, "" when it keeps nothing on disk.
func (s *Store) Path() string {
	if s.dir == "" {
		return ""
	}
	return filepath.Join(s.dir, FileName)
}

// Groups returns the groups in the order the user has them. The caller
// reads them; every change goes through the methods below, so that a name,
// a limit and a save stay in one place.
func (s *Store) Groups() []Group { return s.groups }

// Count returns how many snippets the store holds, in all groups.
func (s *Store) Count() int {
	n := 0
	for _, g := range s.groups {
		n += len(g.Snippets)
	}
	return n
}

// Group returns the group with id.
func (s *Store) Group(id string) (Group, bool) {
	if i := s.groupIndex(id); i >= 0 {
		return s.groups[i], true
	}
	return Group{}, false
}

// AddGroup adds a group at the end, and returns it.
func (s *Store) AddGroup(name string) (Group, error) {
	name, err := cleanName(name)
	if err != nil {
		return Group{}, err
	}
	if s.groupNamed(name, "") >= 0 {
		return Group{}, ErrNameTaken
	}
	g := Group{ID: newID("g"), Name: name}
	s.groups = append(s.groups, g)
	return g, nil
}

// RenameGroup gives the group with id a new name.
func (s *Store) RenameGroup(id, name string) error {
	i := s.groupIndex(id)
	if i < 0 {
		return ErrGroupNotFound
	}
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	if s.groupNamed(name, id) >= 0 {
		return ErrNameTaken
	}
	s.groups[i].Name = name
	return nil
}

// DeleteGroup removes a group and the snippets in it.
func (s *Store) DeleteGroup(id string) error {
	i := s.groupIndex(id)
	if i < 0 {
		return ErrGroupNotFound
	}
	s.groups = append(s.groups[:i], s.groups[i+1:]...)
	return nil
}

// AddSnippet adds a snippet at the end of a group, and returns it.
func (s *Store) AddSnippet(groupID, name, body string) (Snippet, error) {
	i := s.groupIndex(groupID)
	if i < 0 {
		return Snippet{}, ErrGroupNotFound
	}
	name, body, err := cleanSnippet(name, body)
	if err != nil {
		return Snippet{}, err
	}
	if s.Count() >= MaxSnippets {
		return Snippet{}, ErrTooManySnippets
	}
	if s.snippetNamed(i, name, "") >= 0 {
		return Snippet{}, ErrNameTaken
	}
	sn := Snippet{ID: newID("s"), Name: name, Body: body}
	s.groups[i].Snippets = append(s.groups[i].Snippets, sn)
	return sn, nil
}

// UpdateSnippet rewrites the snippet with id in a group. body goes through
// NormalizeBody, so a text pasted with CRLF inserts as LF.
func (s *Store) UpdateSnippet(groupID, id, name, body string) error {
	i := s.groupIndex(groupID)
	if i < 0 {
		return ErrGroupNotFound
	}
	j := s.snippetIndex(i, id)
	if j < 0 {
		return ErrSnippetNotFound
	}
	name, body, err := cleanSnippet(name, body)
	if err != nil {
		return err
	}
	if s.snippetNamed(i, name, id) >= 0 {
		return ErrNameTaken
	}
	s.groups[i].Snippets[j].Name = name
	s.groups[i].Snippets[j].Body = body
	return nil
}

// DeleteSnippet removes one snippet from a group.
func (s *Store) DeleteSnippet(groupID, id string) error {
	i := s.groupIndex(groupID)
	if i < 0 {
		return ErrGroupNotFound
	}
	j := s.snippetIndex(i, id)
	if j < 0 {
		return ErrSnippetNotFound
	}
	snips := s.groups[i].Snippets
	s.groups[i].Snippets = append(snips[:j], snips[j+1:]...)
	return nil
}

// NormalizeBody turns the line endings of a text into the LF the editor's
// buffer holds, and leaves the rest as it is.
func NormalizeBody(body string) string {
	if !strings.ContainsRune(body, '\r') {
		return body
	}
	body = strings.ReplaceAll(body, "\r\n", "\n")
	return strings.ReplaceAll(body, "\r", "\n")
}

// cleanSnippet checks a name and a body together: a snippet needs both.
func cleanSnippet(name, body string) (string, string, error) {
	name, err := cleanName(name)
	if err != nil {
		return "", "", err
	}
	body = NormalizeBody(body)
	if strings.TrimSpace(body) == "" {
		return "", "", ErrBodyEmpty
	}
	if len(body) > MaxBodyBytes {
		return "", "", ErrBodyTooLarge
	}
	return name, body, nil
}

// cleanName trims a name and folds its spaces into one each, so that a
// name typed with a stray newline or a double space is still a name.
// cleanName folds a name's whitespace and refuses an empty or long one.
func cleanName(name string) (string, error) {
	name = normalizeName(name)
	if name == "" {
		return "", ErrNameEmpty
	}
	if utf8.RuneCountInString(name) > MaxNameRunes {
		return "", ErrNameTooLong
	}
	return name, nil
}

// normalizeName folds the spaces of a name into one each and drops the
// control characters a paste may bring, which the list could only draw as
// boxes. A name is a label, not a path: a slash in it is allowed.
func normalizeName(name string) string {
	name = strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' {
			// Fields folds these into a space.
			return ' '
		}
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return -1
		}
		return r
	}, name)
	return strings.Join(strings.Fields(name), " ")
}

func (s *Store) groupIndex(id string) int {
	for i, g := range s.groups {
		if g.ID == id {
			return i
		}
	}
	return -1
}

// groupNamed finds a group by name, skipping the one with exceptID, so a
// rename to its own name is allowed.
func (s *Store) groupNamed(name, exceptID string) int {
	for i, g := range s.groups {
		if g.ID != exceptID && strings.EqualFold(g.Name, name) {
			return i
		}
	}
	return -1
}

func (s *Store) snippetIndex(group int, id string) int {
	for i, sn := range s.groups[group].Snippets {
		if sn.ID == id {
			return i
		}
	}
	return -1
}

// snippetNamed finds a snippet of a group by name, skipping the one with
// exceptID, so an edit that keeps the name is allowed.
func (s *Store) snippetNamed(group int, name, exceptID string) int {
	for i, sn := range s.groups[group].Snippets {
		if sn.ID != exceptID && strings.EqualFold(sn.Name, name) {
			return i
		}
	}
	return -1
}

// newID names a group or a snippet. The ids are the handle a rename or an
// edit keeps, so a name that changes does not lose what points at it.
func newID(prefix string) string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s%x", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}
