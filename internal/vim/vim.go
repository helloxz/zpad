// Package vim is the modal editing ZPad offers when Vim mode is on: the
// modes, the motions and operators, and the text objects. It knows nothing
// of MyGo's widgets: the app gives a Host the text to read, the caret and
// selection to move, and the edits to make.
package vim

import (
	"strings"
	"unicode/utf8"
)

// Mode is what keys mean now.
type Mode uint8

const (
	// Normal is Vim's normal mode: keys are commands and the caret sits
	// on a character.
	Normal Mode = iota
	// Insert is the editor as it always is: keys type text.
	Insert
	// Visual selects characters, VisualLine whole lines.
	Visual
	VisualLine
)

// Buffer is the text the session reads, as editor.Snapshot is.
type Buffer interface {
	Len() int
	LineCount() int
	Line(i int) string
	LineStart(i int) int
	LineEnd(i int) int
	LineOf(offset int) int
	Slice(start, end int) string
}

// Host is what the session asks of the app. It is implemented by a few
// lines over an editor.State, so that the session itself needs no UI.
type Host interface {
	Text() Buffer
	Cursor() int
	SetCursor(offset int)
	SetSelection(anchor, head int)
	Replace(start, end int, text string)
	Undo() bool
	Redo() bool
	MoveToBracket() bool
	Page(forward bool) bool
	FindNext(previous bool, currentStart, currentEnd int) (start, end int, ok bool)
	OpenCommandLine(kind rune)
	Save()
	Close()
	SaveAndClose()
	GoToLine(line int)
	Indent() string
	ReadOnly() bool
	ReadClipboard() string
	WriteClipboard(text string)
}

// Key is a key that is not text: the keys a terminal names, and the few
// control keys Vim uses.
type Key uint8

const (
	KeyNone Key = iota
	KeyEscape
	KeyEnter
	KeyBackspace
	KeyDelete
	KeyTab
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyCtrlR
)

// Session is one tab's modal state: its mode, the count and operator being
// typed, the visual selection, and the unnamed register.
type Session struct {
	mode                   Mode
	count                  int
	op                     rune
	opCount                int
	prefix                 rune // g, waiting for the second key
	obj                    rune // i or a, waiting for the text object
	reg                    string
	linewise               bool
	vAnchor                int
	vHead                  int
	goal                   int // the rune column j and k keep
	searchStart, searchEnd int
}

// New returns a session in normal mode.
func New() *Session { return &Session{goal: -1} }

// Mode returns what keys mean now.
func (s *Session) Mode() Mode { return s.mode }

// Pending reports whether a count or an operator is being typed, which the
// app may show in the status bar and underline the caret for.
func (s *Session) Pending() bool {
	return s.count > 0 || s.op != 0 || s.prefix != 0 || s.obj != 0
}

// Reset puts the session in normal mode with nothing pending, as turning
// Vim mode off or opening a preview does.
func (s *Session) Reset() {
	s.mode = Normal
	s.count, s.op, s.opCount, s.prefix, s.obj = 0, 0, 0, 0, 0
	s.goal = -1
	s.vAnchor, s.vHead = 0, 0
	s.searchStart, s.searchEnd = 0, 0
}

// Rune feeds a typed character, in normal or visual mode. Insert mode is
// the editor's own, so it is the app's to let through.
func (s *Session) Rune(r rune, h Host) {
	if s.mode == Insert {
		return
	}
	if s.mode == Visual || s.mode == VisualLine {
		s.visualRune(r, h)
		return
	}
	s.normalRune(r, h)
}

// Text feeds the characters of an InputText event. It returns the text
// that is left once the session entered insert mode, which the app inserts
// as typing does: a paste of ihello enters insert at i and types hello.
func (s *Session) Text(text string, h Host) (left string) {
	for i, r := range text {
		if s.mode == Insert {
			return text[i:]
		}
		s.Rune(r, h)
	}
	return ""
}

// Key feeds a key that types no text.
func (s *Session) Key(k Key, h Host) bool {
	if s.mode == Insert {
		if k == KeyEscape {
			s.escape(h)
			return true
		}
		return false
	}
	if s.mode == Visual || s.mode == VisualLine {
		switch k {
		case KeyEscape:
			s.escape(h)
		case KeyLeft:
			s.visualRune('h', h)
		case KeyRight:
			s.visualRune('l', h)
		case KeyUp:
			s.visualRune('k', h)
		case KeyDown:
			s.visualRune('j', h)
		case KeyHome:
			s.visualRune('0', h)
		case KeyEnd:
			s.visualRune('$', h)
		case KeyEnter:
			s.visualRune('j', h)
		case KeyBackspace:
			s.visualRune('h', h)
		case KeyDelete:
			s.visualRune('x', h)
		}
		return true
	}
	switch k {
	case KeyEscape:
		s.escape(h)
	case KeyEnter:
		s.enter(h)
	case KeyBackspace:
		s.motionRune('h', 1, h)
	case KeyDelete:
		s.deleteChar(h, 1)
	case KeyTab:
		// Vim's Tab jumps through the jumplist; ZPad has none.
	case KeyLeft:
		s.motionRune('h', 1, h)
	case KeyRight:
		s.motionRune('l', 1, h)
	case KeyUp:
		s.motionRune('k', 1, h)
	case KeyDown:
		s.motionRune('j', 1, h)
	case KeyHome:
		s.motionRune('0', 1, h)
	case KeyEnd:
		s.motionRune('$', 1, h)
	case KeyPageUp:
		if h.Page(false) {
			s.setNormal(h, h.Cursor())
		}
	case KeyPageDown:
		if h.Page(true) {
			s.setNormal(h, h.Cursor())
		}
	case KeyCtrlR:
		if h.Redo() {
			s.setNormal(h, h.Cursor())
		}
	}
	return true
}

// escape leaves insert or visual mode, or cancels what is pending.
func (s *Session) escape(h Host) {
	if s.mode == Insert {
		s.mode = Normal
		off := h.Cursor()
		t := h.Text()
		if off > t.LineStart(t.LineOf(off)) {
			// Leaving insert steps back onto the character typed last, as
			// Vim's Escape does; at the start of a line it stays there.
			off = prevRune(t, off)
		}
		s.setNormal(h, off)
		return
	}
	if s.mode == Visual || s.mode == VisualLine {
		off := s.vHead
		s.mode = Normal
		s.setNormal(h, off)
		return
	}
	s.count, s.op, s.opCount, s.prefix, s.obj = 0, 0, 0, 0, 0
	s.goal = -1
	s.setNormal(h, h.Cursor())
}

// normalRune handles a character in normal mode.
func (s *Session) normalRune(r rune, h Host) {
	if s.op != 0 {
		s.operatorRune(r, h)
		return
	}
	if s.prefix == 'g' {
		s.prefix = 0
		if r == 'g' {
			s.goDoc(h, s.takeCount())
		} else {
			s.count = 0
		}
		return
	}
	if s.prefix == '>' || s.prefix == '<' {
		if r == s.prefix {
			s.indentLines(h, s.cursor(h), s.takeCount(), r == '>')
		}
		s.prefix = 0
		return
	}
	if s.digit(r) {
		return
	}
	switch r {
	case 'g':
		s.prefix = 'g'
	case 'i':
		s.enterInsert(h, s.cursor(h))
	case 'a':
		off := s.cursor(h)
		if off < lineEnd(h.Text(), off) {
			off = nextRune(h.Text(), off)
		}
		s.enterInsert(h, off)
	case 'I':
		s.enterInsert(h, firstNonBlank(h.Text(), lineStart(h.Text(), s.cursor(h))))
	case 'A':
		s.enterInsert(h, lineEnd(h.Text(), s.cursor(h)))
	case 'o':
		s.openLine(h, false)
	case 'O':
		s.openLine(h, true)
	case 'v':
		s.startVisual(h, false)
	case 'V':
		s.startVisual(h, true)
	case 'd', 'c', 'y':
		s.startOperator(r)
	case '>', '<':
		s.prefix = r
	case ' ':
		// Space moves right in normal mode, as Vim's does.
		s.motionRune('l', s.takeCount(), h)
	case 'x':
		s.deleteChar(h, s.takeCount())
	case 'X':
		s.deleteBefore(h, s.takeCount())
	case 'D':
		s.deleteToEnd(h)
	case 'C':
		s.changeToEnd(h)
	case 's':
		s.deleteChar(h, s.takeCount())
		if !h.ReadOnly() {
			s.mode = Insert
		}
	case 'S':
		s.changeLine(h)
	case 'Y':
		s.yankLine(h, s.takeCount())
	case 'p':
		s.paste(h, false, s.takeCount())
	case 'P':
		s.paste(h, true, s.takeCount())
	case 'u':
		for n := s.takeCount(); n > 0; n-- {
			if !h.Undo() {
				break
			}
		}
		s.setNormal(h, h.Cursor())
	case 'J':
		s.joinLines(h, s.takeCount())
	case '~':
		s.toggleCase(h, s.takeCount())
	case '%':
		s.count = 0
		if h.MoveToBracket() {
			s.setNormal(h, h.Cursor())
		}
	case '/', ':':
		s.count = 0
		s.mode = Normal
		h.OpenCommandLine(r)
	case 'n':
		s.count = 0
		if start, end, ok := h.FindNext(false, s.searchStart, s.searchEnd); ok {
			s.searchStart, s.searchEnd = start, end
			s.setNormal(h, start)
		}
	case 'N':
		s.count = 0
		if start, end, ok := h.FindNext(true, s.searchStart, s.searchEnd); ok {
			s.searchStart, s.searchEnd = start, end
			s.setNormal(h, start)
		}
	case 'G':
		t := h.Text()
		line := t.LineCount()
		if n := s.count; n > 0 {
			line = min(n, line)
		}
		s.count = 0
		s.setNormal(h, firstNonBlank(t, t.LineStart(line-1)))
	default:
		if s.motionRune(r, s.takeCount(), h) {
			return
		}
		s.count = 0
	}
}

// operatorRune handles the key after d, c or y, and the one after their i
// or a.
func (s *Session) operatorRune(r rune, h Host) {
	if s.obj != 0 {
		obj := s.obj
		s.obj = 0
		s.applyObject(obj, r, h)
		return
	}
	if r == 'i' || r == 'a' {
		s.obj = r
		return
	}
	if s.prefix == 'g' {
		s.prefix = 0
		if r == 'g' {
			s.applyLineMotion(s.op, 'g', s.takeOpCount(), h)
			return
		}
		s.cancelOperator()
		return
	}
	if s.digit(r) {
		return
	}
	if r == 'g' {
		s.prefix = 'g'
		return
	}
	if r == s.op {
		s.applyLineMotion(s.op, r, s.takeOpCount(), h)
		return
	}
	switch r {
	case 'j', 'k', 'G':
		s.applyLineMotion(s.op, r, s.takeOpCount(), h)
		return
	}
	if s.applyCharMotion(s.op, r, s.takeOpCount(), h) {
		return
	}
	s.cancelOperator()
}

// startOperator begins d, c or y, keeping the count typed before it.
func (s *Session) startOperator(r rune) {
	s.op = r
	s.opCount = s.takeCount()
	s.count = 0
	s.obj = 0
}

// takeCount returns the count typed, 1 without one.
func (s *Session) takeCount() int {
	n := s.count
	s.count = 0
	if n < 1 {
		return 1
	}
	return n
}

// takeOpCount is the count of an operator and one typed after it,
// multiplied as Vim does: 2d3w deletes six words.
func (s *Session) takeOpCount() int {
	n := s.opCount
	if n < 1 {
		n = 1
	}
	if s.count > 0 {
		n *= s.count
	}
	s.count, s.opCount = 0, 0
	return n
}

func (s *Session) cancelOperator() {
	s.op, s.opCount, s.obj, s.prefix = 0, 0, 0, 0
	s.count = 0
}

// digit accumulates a count, including the 0 of 10 once one has begun.
func (s *Session) digit(r rune) bool {
	if r < '0' || r > '9' {
		return false
	}
	if r == '0' && s.count == 0 {
		return false
	}
	if s.count < 100000 {
		s.count = s.count*10 + int(r-'0')
	}
	return true
}

// cursor returns the caret, in normal mode's place on a character.
func (s *Session) cursor(h Host) int {
	return normalize(h.Text(), h.Cursor())
}

// setNormal puts the caret at off as normal mode has it, and collapses the
// selection.
func (s *Session) setNormal(h Host, off int) {
	s.mode = Normal
	s.goal = -1
	off = normalize(h.Text(), off)
	h.SetCursor(off)
}

// enter moves to the first non-blank of the next line, as Enter does.
func (s *Session) enter(h Host) {
	t := h.Text()
	line := t.LineOf(s.cursor(h))
	if line+1 >= t.LineCount() {
		return
	}
	s.setNormal(h, firstNonBlank(t, t.LineStart(line+1)))
}

// enterInsert puts the session in insert mode at off.
func (s *Session) enterInsert(h Host, off int) {
	if h.ReadOnly() {
		return
	}
	s.mode = Insert
	s.count, s.op, s.opCount, s.prefix, s.obj = 0, 0, 0, 0, 0
	s.goal = -1
	h.SetCursor(off)
}

// openLine makes a line below or above, indented as the current one, and
// starts insert there: Vim's o and O.
func (s *Session) openLine(h Host, above bool) {
	s.count = 0
	if h.ReadOnly() {
		return
	}
	t := h.Text()
	line := t.LineOf(s.cursor(h))
	ls, le := t.LineStart(line), t.LineEnd(line)
	indent := leading(t.Line(line))
	if above {
		h.Replace(ls, ls, indent+"\n")
		s.enterInsert(h, ls+len(indent))
		return
	}
	h.Replace(le, le, "\n"+indent)
	s.enterInsert(h, le+1+len(indent))
}

// joinLines joins the next lines to this one, as J does.
func (s *Session) joinLines(h Host, count int) {
	if h.ReadOnly() {
		return
	}
	t := h.Text()
	off := s.cursor(h)
	line := t.LineOf(off)
	linesJoined := max(count, 2)
	last := min(line+linesJoined-1, t.LineCount()-1)
	if last <= line {
		return
	}
	start := t.LineStart(line)
	end := t.LineEnd(last)
	lines := strings.Split(t.Slice(start, end), "\n")
	joined := lines[0]
	for _, next := range lines[1:] {
		trimmed := strings.TrimLeft(next, " \t")
		if trimmed == "" {
			continue
		}
		if joined != "" {
			joined += " "
		}
		joined += trimmed
	}
	h.Replace(start, end, joined)
	at := min(off, start+len(joined))
	s.setNormal(h, at)
}

// toggleCase flips the case of the characters under the caret and after
// it, count of them.
func (s *Session) toggleCase(h Host, count int) {
	if h.ReadOnly() {
		return
	}
	t := h.Text()
	off := s.cursor(h)
	line := t.LineOf(off)
	end := off
	for i := 0; i < count && end < t.LineEnd(line); i++ {
		end = nextRune(t, end)
	}
	if end <= off {
		return
	}
	var out strings.Builder
	for i := off; i < end; {
		r, n := utf8.DecodeRuneInString(t.Slice(i, min(i+utf8.UTFMax, t.Len())))
		switch {
		case 'a' <= r && r <= 'z':
			out.WriteRune(r - 'a' + 'A')
		case 'A' <= r && r <= 'Z':
			out.WriteRune(r - 'A' + 'a')
		default:
			out.WriteString(t.Slice(i, i+n))
		}
		i += n
	}
	h.Replace(off, end, out.String())
	s.setNormal(h, off+len(out.String()))
}

// deleteChar deletes the characters under the caret, x.
func (s *Session) deleteChar(h Host, count int) {
	if h.ReadOnly() {
		return
	}
	t := h.Text()
	off := s.cursor(h)
	end := off
	for i := 0; i < count; i++ {
		if end >= t.LineEnd(t.LineOf(off)) {
			break
		}
		end = nextRune(t, end)
	}
	if end <= off {
		// x on an empty line deletes its newline, joining the next line.
		if off < t.Len() && t.Slice(off, off+1) == "\n" {
			end = off + 1
		} else {
			return
		}
	}
	text := t.Slice(off, end)
	s.setRegister(h, text, false)
	h.Replace(off, end, "")
	s.setNormal(h, off)
}

// deleteBefore deletes the count of characters before the caret, X.
func (s *Session) deleteBefore(h Host, count int) {
	if h.ReadOnly() {
		return
	}
	t := h.Text()
	off := s.cursor(h)
	start := off
	for i := 0; i < count && start > 0; i++ {
		start = prevRune(t, start)
	}
	if start >= off {
		return
	}
	s.setRegister(h, t.Slice(start, off), false)
	h.Replace(start, off, "")
	s.setNormal(h, start)
}

// deleteToEnd deletes from the caret to the end of the line, D.
func (s *Session) deleteToEnd(h Host) {
	s.count = 0
	off := s.cursor(h)
	end := lineEnd(h.Text(), off)
	if end > off {
		s.setRegister(h, h.Text().Slice(off, end), false)
	}
	if h.ReadOnly() || end <= off {
		return
	}
	h.Replace(off, end, "")
	s.setNormal(h, off)
}

// changeToEnd changes from the caret to the end of the line, C.
func (s *Session) changeToEnd(h Host) {
	s.count = 0
	off := s.cursor(h)
	end := lineEnd(h.Text(), off)
	if end > off {
		s.setRegister(h, h.Text().Slice(off, end), false)
		if h.ReadOnly() {
			return
		}
		h.Replace(off, end, "")
	}
	s.enterInsert(h, off)
}

// changeLine empties this line, keeping its indent, and starts insert, as
// cc and S do.
func (s *Session) changeLine(h Host) {
	s.count = 0
	t := h.Text()
	off := s.cursor(h)
	line := t.LineOf(off)
	start := t.LineStart(line)
	end := t.LineEnd(line)
	indent := leading(t.Line(line))
	if end > start {
		s.setRegister(h, t.Slice(start, end), true)
	}
	if h.ReadOnly() {
		return
	}
	h.Replace(start, end, indent)
	s.enterInsert(h, start+len(indent))
}

// yankLine copies count lines, yy.
func (s *Session) yankLine(h Host, count int) {
	t := h.Text()
	line := t.LineOf(s.cursor(h))
	s.yankLines(h, t, line, count)
	s.setNormal(h, s.cursor(h))
}

// yankLines copies lines first through first+count-1.
func (s *Session) yankLines(h Host, t Buffer, first, count int) {
	last := min(first+count-1, t.LineCount()-1)
	start := t.LineStart(first)
	end := t.Len()
	if last+1 < t.LineCount() {
		end = t.LineStart(last + 1)
	}
	text := t.Slice(start, end)
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	s.setRegister(h, text, true)
}

// paste pastes the unnamed register, or the clipboard without one.
func (s *Session) paste(h Host, before bool, count int) {
	if h.ReadOnly() {
		return
	}
	text, linewise := s.reg, s.linewise
	if text == "" {
		text, linewise = h.ReadClipboard(), false
	}
	if text == "" {
		return
	}
	t := h.Text()
	off := s.cursor(h)
	if linewise {
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text = strings.Repeat(text, count)
		line := t.LineOf(off)
		at := t.LineStart(line)
		if !before {
			if line+1 < t.LineCount() {
				at = t.LineStart(line + 1)
			} else {
				at = t.Len()
				if at > 0 && t.Slice(at-1, at) != "\n" {
					text = "\n" + text
				}
			}
		}
		h.Replace(at, at, text)
		after := h.Text()
		s.setNormal(h, firstNonBlank(after, min(at, after.Len())))
		return
	}
	at := off
	if !before {
		n := nextRune(t, off)
		if t.LineOf(n) != t.LineOf(off) {
			n = t.LineEnd(t.LineOf(off))
		}
		at = n
	}
	text = strings.Repeat(text, count)
	h.Replace(at, at, text)
	after := h.Text()
	end := min(at+len(text), after.Len())
	if end > at {
		s.setNormal(h, prevRune(after, end))
	} else {
		s.setNormal(h, at)
	}
}

// setRegister keeps text as the unnamed register and on the clipboard, as
// a yank or a delete does.
func (s *Session) setRegister(h Host, text string, linewise bool) {
	s.reg, s.linewise = text, linewise
	if text != "" {
		h.WriteClipboard(text)
	}
}

// SetSearchRange remembers the match the app just found, so that n and N
// continue after it rather than finding it again.
func (s *Session) SetSearchRange(start, end int) {
	s.searchStart, s.searchEnd = start, end
}

// GoToLine is the app's part of a :<line> command.
func (s *Session) GoToLine(h Host, line int) {
	if line < 1 {
		return
	}
	h.GoToLine(line)
	s.setNormal(h, h.Cursor())
}

// Normal is for the app after a command line edit.
func (s *Session) Normal(h Host) { s.setNormal(h, h.Cursor()) }
