package vim

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// runeAt returns the rune at off, and 0 past the text.
func runeAt(t Buffer, off int) rune {
	if off < 0 || off >= t.Len() {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(t.Slice(off, min(off+utf8.UTFMax, t.Len())))
	return r
}

// nextRune returns the offset after the rune at off.
func nextRune(t Buffer, off int) int {
	if off >= t.Len() {
		return t.Len()
	}
	_, n := utf8.DecodeRuneInString(t.Slice(off, min(off+utf8.UTFMax, t.Len())))
	if n <= 0 {
		return t.Len()
	}
	return min(off+n, t.Len())
}

// prevRune returns the offset of the rune before off.
func prevRune(t Buffer, off int) int {
	if off <= 0 {
		return 0
	}
	if off > t.Len() {
		off = t.Len()
	}
	start := off - 1
	for start > 0 && start > off-utf8.UTFMax && !utf8.RuneStart(t.Slice(start, start+1)[0]) {
		start--
	}
	return start
}

// lineStart and lineEnd return the bytes of the line holding off.
func lineStart(t Buffer, off int) int { return t.LineStart(t.LineOf(off)) }
func lineEnd(t Buffer, off int) int   { return t.LineEnd(t.LineOf(off)) }

// normalize puts off where normal mode's caret goes: on a character, or at
// the start of an empty line, and never past the last character.
func normalize(t Buffer, off int) int {
	n := t.Len()
	if n == 0 {
		return 0
	}
	if off < 0 {
		off = 0
	}
	if off > n {
		off = n
	}
	if off == n {
		if t.Slice(n-1, n) == "\n" {
			// The empty line a trailing newline makes.
			return n
		}
		off = n - 1
	}
	// Keep the offset on a rune boundary.
	for off > 0 && !utf8.RuneStart(t.Slice(off, off+1)[0]) {
		off--
	}
	if runeAt(t, off) == '\n' {
		ls := lineStart(t, off)
		if off > ls {
			off = prevRune(t, off)
		}
	}
	return off
}

// firstNonBlank returns the first character of the line at start that is
// not a space or tab, or start itself.
func firstNonBlank(t Buffer, start int) int {
	line := t.LineOf(start)
	text := t.Line(line)
	i := 0
	for i < len(text) {
		r, n := utf8.DecodeRuneInString(text[i:])
		if r != ' ' && r != '\t' {
			return start + i
		}
		i += n
	}
	return start
}

// leading returns the spaces and tabs line starts with.
func leading(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// runeColumn returns how many runes of its line come before off.
func runeColumn(t Buffer, off int) int {
	start := lineStart(t, off)
	return utf8.RuneCountInString(t.Slice(start, min(off, t.Len())))
}

// offsetAtColumn returns the byte of rune column col of line, clamped to
// its last character (or its start when empty).
func offsetAtColumn(t Buffer, line, col int) int {
	start := t.LineStart(line)
	text := t.Line(line)
	if text == "" {
		return start
	}
	if col < 0 {
		col = 0
	}
	i, n := 0, 0
	for n < col && i < len(text) {
		_, size := utf8.DecodeRuneInString(text[i:])
		i += size
		n++
	}
	if i >= len(text) {
		_, size := utf8.DecodeLastRuneInString(text)
		return start + len(text) - size
	}
	return start + i
}

type charClass uint8

const (
	classSpace charClass = iota
	classPunct
	classWord
)

func classOf(r rune, big bool) charClass {
	if unicode.IsSpace(r) {
		return classSpace
	}
	if big || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) || r == '_' {
		return classWord
	}
	return classPunct
}

func classAt(t Buffer, off int, big bool) charClass {
	return classOf(runeAt(t, off), big)
}

// wordForward is w or W: the start of the next word, count words on.
func wordForward(t Buffer, off, count int, big bool) int {
	for n := 0; n < count; n++ {
		if off >= t.Len() {
			break
		}
		cls := classAt(t, off, big)
		if cls != classSpace {
			for off < t.Len() && classAt(t, off, big) == cls {
				off = nextRune(t, off)
			}
		}
		for off < t.Len() && classAt(t, off, big) == classSpace {
			off = nextRune(t, off)
		}
	}
	return off
}

// wordEnd is e or E: the last character of the next word.
func wordEnd(t Buffer, off, count int, big bool) int {
	for n := 0; n < count; n++ {
		if off >= t.Len() {
			break
		}
		off = nextRune(t, off)
		for off < t.Len() && classAt(t, off, big) == classSpace {
			off = nextRune(t, off)
		}
		if off >= t.Len() {
			off = t.Len() - 1
			break
		}
		cls := classAt(t, off, big)
		for {
			n := nextRune(t, off)
			if n >= t.Len() || classAt(t, n, big) != cls {
				break
			}
			off = n
		}
	}
	return off
}

// wordBack is b or B: the start of the previous word.
func wordBack(t Buffer, off, count int, big bool) int {
	for n := 0; n < count; n++ {
		if off <= 0 {
			break
		}
		off = prevRune(t, off)
		for off > 0 && classAt(t, off, big) == classSpace {
			off = prevRune(t, off)
		}
		cls := classAt(t, off, big)
		for off > 0 && classAt(t, prevRune(t, off), big) == cls {
			off = prevRune(t, off)
		}
	}
	return off
}

// paragraph moves to the blank line before or after the one the caret is
// in, as { and } do.
func paragraph(t Buffer, off int, forward bool) int {
	line := t.LineOf(off)
	count := t.LineCount()
	blank := func(i int) bool { return strings.TrimSpace(t.Line(i)) == "" }
	if forward {
		for line < count && !blank(line) {
			line++
		}
		for line < count && blank(line) {
			line++
		}
		if line >= count {
			line = count - 1
		}
	} else {
		line--
		for line > 0 && blank(line) {
			line--
		}
		for line > 0 && !blank(line) {
			line--
		}
		if line < 0 {
			line = 0
		}
	}
	return t.LineStart(line)
}

// motionRune moves the caret in normal or visual mode and reports whether r
// was a motion.
func (s *Session) motionRune(r rune, count int, h Host) bool {
	t := h.Text()
	off := s.cursor(h)
	goal := -1
	switch r {
	case 'h':
		for i := 0; i < count && off > 0; i++ {
			off = prevRune(t, off)
		}
	case 'l':
		for i := 0; i < count; i++ {
			n := nextRune(t, off)
			if n >= t.Len() || t.LineOf(n) != t.LineOf(off) {
				break
			}
			off = n
		}
	case 'j', 'k', '+', '-':
		delta := count
		if r == 'k' || r == '-' {
			delta = -count
		}
		off, goal = vertical(t, off, delta, s.goal)
	case 'w', 'W':
		off = wordForward(t, off, count, r == 'W')
	case 'e', 'E':
		off = wordEnd(t, off, count, r == 'E')
	case 'b', 'B':
		off = wordBack(t, off, count, r == 'B')
	case '0':
		off = lineStart(t, off)
	case '^':
		off = firstNonBlank(t, lineStart(t, off))
	case '$':
		off = lineEnd(t, off)
	case 'G':
		line := count
		if line <= 1 {
			line = t.LineCount()
		}
		off = firstNonBlank(t, t.LineStart(min(line-1, t.LineCount()-1)))
	case '}':
		off = paragraph(t, off, true)
	case '{':
		off = paragraph(t, off, false)
	default:
		return false
	}
	s.setNormalAt(h, off, goal)
	return true
}

// goDoc handles gg and NG.
func (s *Session) goDoc(h Host, count int) {
	t := h.Text()
	line := count
	if line < 1 {
		line = 1
	}
	line = min(line, t.LineCount())
	s.setNormal(h, firstNonBlank(t, t.LineStart(line-1)))
}

// vertical moves count lines down or up, keeping the rune column goal.
func vertical(t Buffer, off, delta, goal int) (int, int) {
	line := t.LineOf(off)
	if goal < 0 {
		goal = runeColumn(t, off)
	}
	target := max(0, min(line+delta, t.LineCount()-1))
	return offsetAtColumn(t, target, goal), goal
}

// setNormalAt sets the caret and keeps or drops the goalless column.
func (s *Session) setNormalAt(h Host, off, goal int) {
	off = normalize(h.Text(), off)
	h.SetCursor(off)
	s.goal = goal
}

// startVisual begins a visual selection at the caret.
func (s *Session) startVisual(h Host, linewise bool) {
	s.count = 0
	off := s.cursor(h)
	if linewise {
		s.mode = VisualLine
	} else {
		s.mode = Visual
	}
	s.vAnchor, s.vHead = off, off
	s.goal = -1
	s.applyVisual(h)
}

// visualRange returns what the visual selection covers.
func (s *Session) visualRange(t Buffer) Range {
	a, z := s.vAnchor, s.vHead
	if s.mode == VisualLine {
		first := min(t.LineOf(a), t.LineOf(z))
		last := max(t.LineOf(a), t.LineOf(z))
		start := t.LineStart(first)
		end := t.Len()
		if last+1 < t.LineCount() {
			end = t.LineStart(last + 1)
		}
		return Range{Start: start, End: end, Linewise: true}
	}
	start, end := min(a, z), max(a, z)
	if end < t.Len() {
		next := nextRune(t, end)
		if t.LineOf(next) == t.LineOf(end) {
			end = next
		}
	}
	return Range{Start: start, End: end}
}

// applyVisual shows the selection in the editor.
func (s *Session) applyVisual(h Host) {
	r := s.visualRange(h.Text())
	h.SetSelection(r.Start, r.End)
}

// visualRune handles a character in visual mode.
func (s *Session) visualRune(r rune, h Host) {
	if s.prefix == 'g' {
		s.prefix = 0
		if r == 'g' {
			t := h.Text()
			s.vHead = firstNonBlank(t, t.LineStart(min(s.takeCount()-1, t.LineCount()-1)))
			s.applyVisual(h)
		}
		return
	}
	if s.digit(r) {
		return
	}
	switch r {
	case 'g':
		s.prefix = 'g'
	case 'o':
		s.vAnchor, s.vHead = s.vHead, s.vAnchor
		s.applyVisual(h)
	case 'v':
		if s.mode == Visual {
			s.exitVisual(h)
			return
		}
		s.mode = Visual
		s.applyVisual(h)
	case 'V':
		if s.mode == VisualLine {
			s.exitVisual(h)
			return
		}
		s.mode = VisualLine
		s.applyVisual(h)
	case 'd', 'x':
		s.deleteVisual(h)
	case 'y':
		s.yankVisual(h)
	case 'c':
		s.changeVisual(h)
	case 'p':
		s.pasteVisual(h)
	case '>':
		s.indentVisual(h, true)
	case '<':
		s.indentVisual(h, false)
	case ':':
		s.exitVisual(h)
		h.OpenCommandLine(r)
	case '/':
		s.exitVisual(h)
		h.OpenCommandLine(r)
	case 'G':
		t := h.Text()
		line := t.LineCount()
		if n := s.count; n > 0 {
			line = min(n, line)
		}
		s.count = 0
		s.vHead = firstNonBlank(t, t.LineStart(min(line-1, t.LineCount()-1)))
		s.applyVisual(h)
		return
	case 'n', 'N':
		s.exitVisual(h)
		if start, end, ok := h.FindNext(r == 'N', s.searchStart, s.searchEnd); ok {
			s.searchStart, s.searchEnd = start, end
			s.setNormal(h, start)
		}
		return
	default:
		count := s.takeCount()
		t := h.Text()
		off := s.vHead
		goal := -1
		switch r {
		case 'h':
			for i := 0; i < count && off > 0; i++ {
				off = prevRune(t, off)
			}
		case 'l':
			for i := 0; i < count; i++ {
				n := nextRune(t, off)
				if n >= t.Len() || t.LineOf(n) != t.LineOf(off) || n >= lineEnd(t, off) {
					break
				}
				off = n
			}
		case 'j', 'k':
			delta := count
			if r == 'k' {
				delta = -count
			}
			off, goal = vertical(t, off, delta, s.goal)
		case 'w', 'W':
			off = wordForward(t, off, count, r == 'W')
		case 'e', 'E':
			off = wordEnd(t, off, count, r == 'E')
		case 'b', 'B':
			off = wordBack(t, off, count, r == 'B')
		case '0':
			off = lineStart(t, off)
		case '^':
			off = firstNonBlank(t, lineStart(t, off))
		case '$':
			off = lineEnd(t, off)
		default:
			return
		}
		s.vHead = off
		s.goal = goal
		s.applyVisual(h)
	}
}

// deleteVisual deletes the selection.
func (s *Session) deleteVisual(h Host) {
	t := h.Text()
	r := s.visualRange(t)
	text := t.Slice(r.Start, r.End)
	s.setRegister(h, text, r.Linewise)
	start := r.Start
	if r.Linewise && r.End == t.Len() && start > 0 && !strings.HasSuffix(text, "\n") {
		start--
	}
	s.mode = Normal
	if !h.ReadOnly() {
		h.Replace(start, r.End, "")
	}
	s.setNormal(h, start)
}

// yankVisual copies the selection and leaves the caret at its start.
func (s *Session) yankVisual(h Host) {
	t := h.Text()
	r := s.visualRange(t)
	s.setRegister(h, t.Slice(r.Start, r.End), r.Linewise)
	s.mode = Normal
	s.setNormal(h, r.Start)
}

// changeVisual deletes the selection and starts insert at its start.
func (s *Session) changeVisual(h Host) {
	t := h.Text()
	r := s.visualRange(t)
	s.setRegister(h, t.Slice(r.Start, r.End), r.Linewise)
	s.mode = Normal
	if h.ReadOnly() {
		s.setNormal(h, r.Start)
		return
	}
	start := r.Start
	text := t.Slice(r.Start, r.End)
	replacement := ""
	if r.Linewise && strings.Contains(text, "\n") {
		replacement = "\n"
	}
	if r.Linewise && r.End == t.Len() && start > 0 && !strings.HasSuffix(text, "\n") {
		start--
	}
	h.Replace(start, r.End, replacement)
	s.enterInsert(h, start)
}

// pasteVisual replaces the selection with the register.
func (s *Session) pasteVisual(h Host) {
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
	r := s.visualRange(t)
	if linewise && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	h.Replace(r.Start, r.End, text)
	s.mode = Normal
	s.setNormal(h, firstNonBlank(h.Text(), min(r.Start, h.Text().Len())))
}

// indentVisual shifts the lines the selection covers.
func (s *Session) indentVisual(h Host, right bool) {
	t := h.Text()
	r := s.visualRange(t)
	first := t.LineOf(r.Start)
	last := t.LineOf(max(r.Start, r.End-1))
	s.indentRange(h, first, last, right)
	s.mode = Normal
	s.setNormal(h, firstNonBlank(h.Text(), h.Text().LineStart(first)))
}

// exitVisual leaves visual mode at the caret.
func (s *Session) exitVisual(h Host) {
	off := s.vHead
	s.mode = Normal
	s.setNormal(h, off)
}

// indentLines shifts count lines from off, as >> and << do.
func (s *Session) indentLines(h Host, off, count int, right bool) {
	t := h.Text()
	first := t.LineOf(off)
	s.indentRange(h, first, min(first+count-1, t.LineCount()-1), right)
	s.setNormal(h, firstNonBlank(h.Text(), h.Text().LineStart(first)))
}

// indentRange shifts lines first to last with one edit.
func (s *Session) indentRange(h Host, first, last int, right bool) {
	if h.ReadOnly() {
		return
	}
	t := h.Text()
	start := t.LineStart(first)
	end := t.Len()
	if last+1 < t.LineCount() {
		end = t.LineStart(last + 1)
	}
	block := t.Slice(start, end)
	indent := h.Indent()
	if indent == "" {
		indent = "    "
	}
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		if right {
			lines[i] = indent + line
			continue
		}
		lines[i] = outdent(line, indent)
	}
	h.Replace(start, end, strings.Join(lines, "\n"))
}

// outdent removes one level of indentation from line.
func outdent(line, indent string) string {
	if strings.HasPrefix(line, "\t") {
		return line[1:]
	}
	width := len(indent)
	for width > 0 && strings.HasPrefix(line, " ") {
		line = line[1:]
		width--
	}
	return line
}
