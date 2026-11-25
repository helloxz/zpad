package vim

import (
	"testing"

	"github.com/egoist/mygo/ui/editor"
)

// testHost is a Host over a Buffer: enough of an editor for a session.
type testHost struct {
	buf      *editor.Buffer
	cur      int
	anchor   int
	clip     string
	readonly bool
	cmd      rune
	saved    bool
	closed   bool
	saveQuit bool
	line     int
}

func newTestHost(text string) *testHost {
	return &testHost{buf: editor.NewBuffer(text)}
}

func (h *testHost) Text() Buffer { return h.buf.Snapshot() }

func (h *testHost) Cursor() int { return h.cur }

func (h *testHost) SetCursor(off int) {
	h.cur = min(max(off, 0), h.buf.Len())
	h.anchor = h.cur
}

func (h *testHost) SetSelection(anchor, head int) {
	h.anchor = min(max(anchor, 0), h.buf.Len())
	h.cur = min(max(head, 0), h.buf.Len())
}

func (h *testHost) Replace(start, end int, text string) {
	if h.readonly {
		return
	}
	h.buf.Edit(editor.Edit{Start: start, End: end, Text: text})
}

func (h *testHost) Undo() bool { return h.buf.Undo() }
func (h *testHost) Redo() bool { return h.buf.Redo() }

func (h *testHost) MoveToBracket() bool                      { return false }
func (h *testHost) Page(bool) bool                           { return false }
func (h *testHost) FindNext(bool, int, int) (int, int, bool) { return 0, 0, false }

func (h *testHost) OpenCommandLine(kind rune) { h.cmd = kind }
func (h *testHost) Save()                     { h.saved = true }
func (h *testHost) Close()                    { h.closed = true }
func (h *testHost) SaveAndClose()             { h.saved, h.closed = true, true }
func (h *testHost) GoToLine(line int)         { h.line = line }

func (h *testHost) Indent() string { return "    " }

func (h *testHost) ReadOnly() bool { return h.readonly }

func (h *testHost) ReadClipboard() string { return h.clip }

func (h *testHost) WriteClipboard(text string) { h.clip = text }

// feed feeds keys as typing does, inserting what is left once insert mode
// begins.
func feed(t *testing.T, s *Session, h *testHost, keys string) {
	t.Helper()
	for _, r := range keys {
		if s.Mode() == Insert {
			h.Replace(h.cur, h.cur, string(r))
			h.cur += len(string(r))
			continue
		}
		s.Rune(r, h)
	}
}

func wantText(t *testing.T, h *testHost, want string) {
	t.Helper()
	if got := h.buf.String(); got != want {
		t.Fatalf("text %q, want %q", got, want)
	}
}

func wantCursor(t *testing.T, h *testHost, want int) {
	t.Helper()
	if h.cur != want {
		t.Fatalf("cursor %d, want %d", h.cur, want)
	}
}

func TestNormalMotions(t *testing.T) {
	h := newTestHost("abc\ndef")
	s := New()
	feed(t, s, h, "ll")
	wantCursor(t, h, 2)
	feed(t, s, h, "j")
	wantCursor(t, h, 6)
	feed(t, s, h, "$")
	wantCursor(t, h, 6)
	feed(t, s, h, "0")
	wantCursor(t, h, 4)
	feed(t, s, h, "G")
	wantCursor(t, h, 4)
	feed(t, s, h, "gg")
	wantCursor(t, h, 0)
}

func TestWordMotions(t *testing.T) {
	h := newTestHost("foo bar\nbaz")
	s := New()
	feed(t, s, h, "w")
	wantCursor(t, h, 4)
	feed(t, s, h, "e")
	wantCursor(t, h, 6)
	feed(t, s, h, "b")
	wantCursor(t, h, 4)
	feed(t, s, h, "W")
	wantCursor(t, h, 8)
	feed(t, s, h, "B")
	wantCursor(t, h, 4)
}

func TestChineseWords(t *testing.T) {
	h := newTestHost("中文 测试")
	s := New()
	feed(t, s, h, "w")
	wantCursor(t, h, 7)
	feed(t, s, h, "b")
	wantCursor(t, h, 0)
	// A Chinese run is one word, as Vim's default iskeyword has it.
	feed(t, s, h, "e")
	wantCursor(t, h, 3)
}

func TestInsertAndEscape(t *testing.T) {
	h := newTestHost("ab")
	s := New()
	left := s.Text("ihello", h)
	if left != "hello" {
		t.Fatalf("left %q, want hello", left)
	}
	if s.Mode() != Insert {
		t.Fatal("i did not enter insert mode")
	}
	h.Replace(h.cur, h.cur, left)
	h.cur += len(left)
	feed(t, s, h, "!")
	s.Key(KeyEscape, h)
	if s.Mode() != Normal {
		t.Fatal("Escape did not return to normal mode")
	}
	wantText(t, h, "hello!ab")
	wantCursor(t, h, 5)
}

func TestAppendAndLineEnds(t *testing.T) {
	h := newTestHost("ab\ncd")
	s := New()
	feed(t, s, h, "A")
	feed(t, s, h, "!")
	s.Key(KeyEscape, h)
	wantText(t, h, "ab!\ncd")
	h2 := newTestHost("ab\ncd")
	s = New()
	feed(t, s, h2, "jI")
	feed(t, s, h2, "#")
	s.Key(KeyEscape, h2)
	wantText(t, h2, "ab\n#cd")
}

func TestDeleteAndUndo(t *testing.T) {
	h := newTestHost("one\ntwo\nthree")
	s := New()
	feed(t, s, h, "dd")
	wantText(t, h, "two\nthree")
	if h.clip != "one\n" {
		t.Fatalf("register %q, want one\\n", h.clip)
	}
	feed(t, s, h, "u")
	wantText(t, h, "one\ntwo\nthree")
}

func TestDeleteWordAndChangeWord(t *testing.T) {
	h := newTestHost("foo bar baz")
	s := New()
	feed(t, s, h, "dw")
	wantText(t, h, "bar baz")
	h2 := newTestHost("foo bar baz")
	s = New()
	feed(t, s, h2, "wciw")
	feed(t, s, h2, "X")
	s.Key(KeyEscape, h2)
	wantText(t, h2, "foo X baz")
}

func TestCounts(t *testing.T) {
	h := newTestHost("a\nb\nc\nd")
	s := New()
	feed(t, s, h, "2dd")
	wantText(t, h, "c\nd")
	h2 := newTestHost("a b c d")
	s = New()
	feed(t, s, h2, "2dw")
	wantText(t, h2, "c d")
	h3 := newTestHost("abcdef")
	s = New()
	feed(t, s, h3, "3x")
	wantText(t, h3, "def")
}

func TestLineEdits(t *testing.T) {
	h := newTestHost("foo\nbar")
	s := New()
	feed(t, s, h, "A!")
	s.Key(KeyEscape, h)
	wantText(t, h, "foo!\nbar")
	feed(t, s, h, "jD")
	wantText(t, h, "foo!\nba")
	feed(t, s, h, "u")
	wantText(t, h, "foo!\nbar")
	feed(t, s, h, "oX")
	s.Key(KeyEscape, h)
	wantText(t, h, "foo!\nbar\nX")
	feed(t, s, h, "O#")
	s.Key(KeyEscape, h)
	wantText(t, h, "foo!\nbar\n#\nX")
}

func TestJoinLines(t *testing.T) {
	h := newTestHost("foo\n   bar\nbaz")
	s := New()
	feed(t, s, h, "J")
	wantText(t, h, "foo bar\nbaz")
	feed(t, s, h, "2J")
	wantText(t, h, "foo bar baz")
}

func TestPaste(t *testing.T) {
	h := newTestHost("one\ntwo")
	s := New()
	feed(t, s, h, "yy")
	feed(t, s, h, "p")
	wantText(t, h, "one\none\ntwo")
	h2 := newTestHost("one\ntwo")
	s = New()
	feed(t, s, h2, "yy")
	feed(t, s, h2, "P")
	wantText(t, h2, "one\none\ntwo")
	h3 := newTestHost("abc")
	s = New()
	feed(t, s, h3, "yl")
	feed(t, s, h3, "l")
	feed(t, s, h3, "p")
	wantText(t, h3, "abac")
}

func TestVisualDeleteAndYank(t *testing.T) {
	h := newTestHost("abcdef")
	s := New()
	feed(t, s, h, "vlld")
	wantText(t, h, "def")
	h2 := newTestHost("abcdef")
	s = New()
	feed(t, s, h2, "vlly")
	wantText(t, h2, "abcdef")
	if h2.clip != "abc" {
		t.Fatalf("yank %q, want abc", h2.clip)
	}
}

func TestVisualLine(t *testing.T) {
	h := newTestHost("one\ntwo\nthree")
	s := New()
	feed(t, s, h, "Vjd")
	wantText(t, h, "three")
}

func TestQuoteTextObject(t *testing.T) {
	h := newTestHost(`say "hello" now`)
	s := New()
	feed(t, s, h, `di"`)
	if got := h.buf.String(); got != `say "" now` {
		t.Fatalf("text %q, want say \"\" now", got)
	}
}

func TestBracketTextObject(t *testing.T) {
	h := newTestHost("if (a + b) {}")
	s := New()
	for h.cur < 5 {
		feed(t, s, h, "l")
	}
	feed(t, s, h, "di(")
	if got := h.buf.String(); got != "if () {}" {
		t.Fatalf("text %q, want if () {}", got)
	}
}

func TestParagraphTextObject(t *testing.T) {
	h := newTestHost("a\nb\n\nc\nd")
	s := New()
	feed(t, s, h, "dip")
	wantText(t, h, "\nc\nd")
}

func TestCommandsOpen(t *testing.T) {
	h := newTestHost("abc")
	s := New()
	feed(t, s, h, "/")
	if h.cmd != '/' {
		t.Fatalf("command %q, want /", h.cmd)
	}
	h.cmd = 0
	feed(t, s, h, ":")
	if h.cmd != ':' {
		t.Fatalf("command %q, want :", h.cmd)
	}
}

func TestIndent(t *testing.T) {
	h := newTestHost("a\nb")
	s := New()
	feed(t, s, h, ">>")
	wantText(t, h, "    a\nb")
	feed(t, s, h, "<<")
	wantText(t, h, "a\nb")
}

func TestReadOnly(t *testing.T) {
	h := newTestHost("abc")
	h.readonly = true
	s := New()
	feed(t, s, h, "x")
	wantText(t, h, "abc")
	feed(t, s, h, "i")
	if s.Mode() == Insert {
		t.Fatal("i entered insert in a read-only editor")
	}
}

func TestChangeWordKeepsTheSpace(t *testing.T) {
	h := newTestHost("foo bar")
	s := New()
	feed(t, s, h, "cwX")
	s.Key(KeyEscape, h)
	wantText(t, h, "X bar")
}

func TestDocMotions(t *testing.T) {
	h := newTestHost("a\nb\nc")
	s := New()
	feed(t, s, h, "jdgg")
	wantText(t, h, "c")
	h2 := newTestHost("a\nb\nc")
	s = New()
	feed(t, s, h2, "jdG")
	wantText(t, h2, "a")
}

func TestEscapeCancelsPending(t *testing.T) {
	h := newTestHost("abc")
	s := New()
	feed(t, s, h, "d")
	s.Key(KeyEscape, h)
	feed(t, s, h, "x")
	wantText(t, h, "bc")
}

func TestDeleteToLineEnd(t *testing.T) {
	h := newTestHost("abcdef\nnext")
	s := New()
	feed(t, s, h, "lld$")
	wantText(t, h, "ab\nnext")
}

func TestVisualLineYankPaste(t *testing.T) {
	h := newTestHost("a\nb\nc")
	s := New()
	feed(t, s, h, "Vjy")
	feed(t, s, h, "Gp")
	wantText(t, h, "a\nb\nc\na\nb\n")
}

func TestEmptyLines(t *testing.T) {
	h := newTestHost("a\n\nb")
	s := New()
	feed(t, s, h, "j")
	wantCursor(t, h, 2)
	feed(t, s, h, "x")
	wantText(t, h, "a\nb")
}

func TestUndoRedo(t *testing.T) {
	h := newTestHost("one\ntwo")
	s := New()
	feed(t, s, h, "dd")
	wantText(t, h, "two")
	feed(t, s, h, "u")
	wantText(t, h, "one\ntwo")
	s.Key(KeyCtrlR, h)
	wantText(t, h, "two")
}

func TestEscapeAtLineStartStays(t *testing.T) {
	h := newTestHost("a")
	s := New()
	feed(t, s, h, "o")
	s.Key(KeyEscape, h)
	wantText(t, h, "a\n")
	wantCursor(t, h, 2)
}

func TestChangeLineAndToEnd(t *testing.T) {
	h := newTestHost("foo bar\nnext")
	s := New()
	feed(t, s, h, "SX")
	s.Key(KeyEscape, h)
	wantText(t, h, "X\nnext")
	h2 := newTestHost("abcdef")
	s = New()
	feed(t, s, h2, "llCX")
	s.Key(KeyEscape, h2)
	wantText(t, h2, "abX")
}

func TestDeleteBeforeAndLastLine(t *testing.T) {
	h := newTestHost("abc")
	s := New()
	feed(t, s, h, "llX")
	wantText(t, h, "ac")
	h2 := newTestHost("a\nb")
	s = New()
	feed(t, s, h2, "jdd")
	wantText(t, h2, "a")
}

func TestToggleCase(t *testing.T) {
	h := newTestHost("aBc")
	s := New()
	feed(t, s, h, "ll~")
	wantText(t, h, "aBC")
	feed(t, s, h, "~")
	wantText(t, h, "aBc")
}

func TestChangeLinesKeepsOneLine(t *testing.T) {
	h := newTestHost("a\nb\nc")
	s := New()
	feed(t, s, h, "cjX")
	s.Key(KeyEscape, h)
	wantText(t, h, "X\nc")
}

func TestVisualLineChangeKeepsOneLine(t *testing.T) {
	h := newTestHost("a\nb\nc")
	s := New()
	feed(t, s, h, "VjcX")
	s.Key(KeyEscape, h)
	wantText(t, h, "X\nc")
}

func TestVimKeepsLineEndings(t *testing.T) {
	h := newTestHost("a\r\nb\r\n")
	s := New()
	feed(t, s, h, "dd")
	if got := h.buf.LineEnding(); got != "\r\n" {
		t.Fatalf("line ending %q after dd, want CRLF", got)
	}
	feed(t, s, h, "oX")
	s.Key(KeyEscape, h)
	if got := h.buf.LineEnding(); got != "\r\n" {
		t.Fatalf("line ending %q after o, want CRLF", got)
	}
}
