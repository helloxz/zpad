package vim

import "strings"

// Range is a piece of the text: bytes Start to End, End exclusive, and
// whether the whole lines are meant, as dd and V select.
type Range struct {
	Start, End int
	Linewise   bool
}

// applyCharMotion is an operator with a characterwise motion: dw, d$, ci(.
func (s *Session) applyCharMotion(op rune, r rune, count int, h Host) bool {
	t := h.Text()
	off := s.cursor(h)
	line := t.LineOf(off)
	start, end := off, off
	switch r {
	case 'h':
		for i := 0; i < count && start > 0; i++ {
			start = prevRune(t, start)
		}
	case 'l':
		for i := 0; i < count; i++ {
			n := nextRune(t, end)
			if n >= t.Len() || t.LineOf(n) != line {
				break
			}
			end = n
		}
	case 'w', 'W':
		if op == 'c' && classAt(t, off, r == 'W') != classSpace {
			// cw changes the word under the caret, not the space after it,
			// as Vim's cw is ce there.
			end = wordEnd(t, off, count, r == 'W')
			if t.LineOf(end) != line {
				end = t.LineEnd(line)
			} else {
				end = nextRune(t, end)
			}
		} else {
			end = wordForward(t, off, count, r == 'W')
			if t.LineOf(end) != line {
				end = t.LineEnd(line)
			}
		}
	case 'e', 'E':
		end = wordEnd(t, off, count, r == 'E')
		if t.LineOf(end) != line {
			end = t.LineEnd(line)
		} else {
			end = nextRune(t, end)
		}
	case 'b', 'B':
		start = wordBack(t, off, count, r == 'B')
	case '0':
		start = t.LineStart(line)
	case '^':
		start = firstNonBlank(t, t.LineStart(line))
	case '$':
		end = t.LineEnd(line)
	default:
		return false
	}
	if start > end {
		start, end = end, start
	}
	if start == end {
		s.cancelOperator()
		return true
	}
	s.applyRange(op, Range{Start: start, End: end}, h)
	s.cancelOperator()
	return true
}

// applyLineMotion is an operator with a linewise motion, or with itself:
// dd, yy, cj, dG and dgg.
func (s *Session) applyLineMotion(op rune, motion rune, count int, h Host) {
	t := h.Text()
	off := s.cursor(h)
	line := t.LineOf(off)
	first, last := line, line
	switch motion {
	case 'j':
		last = min(line+count, t.LineCount()-1)
	case 'k':
		first = max(0, line-count)
	case 'G':
		last = t.LineCount() - 1
	case 'g':
		first = 0
	default:
		// The operator twice: its own line, count of them.
		last = min(line+count-1, t.LineCount()-1)
	}
	start := t.LineStart(first)
	end := t.Len()
	if last+1 < t.LineCount() {
		end = t.LineStart(last + 1)
	}
	if op == 'c' {
		if motion == op {
			s.changeLine(h)
			s.cancelOperator()
			return
		}
		s.changeLineRange(h, Range{Start: start, End: end, Linewise: true})
		s.cancelOperator()
		return
	}
	s.applyRange(op, Range{Start: start, End: end, Linewise: true}, h)
	s.cancelOperator()
}

// applyRange yanks, deletes or changes a range.
func (s *Session) applyRange(op rune, r Range, h Host) {
	t := h.Text()
	if r.Start < 0 {
		r.Start = 0
	}
	if r.End > t.Len() {
		r.End = t.Len()
	}
	if r.Start > r.End {
		r.Start, r.End = r.End, r.Start
	}
	if r.Start == r.End {
		return
	}
	text := t.Slice(r.Start, r.End)
	if op == 'y' {
		s.setRegister(h, text, r.Linewise)
		s.setNormal(h, r.Start)
		return
	}
	s.setRegister(h, text, r.Linewise)
	if h.ReadOnly() {
		s.setNormal(h, r.Start)
		return
	}
	start := r.Start
	if r.Linewise && r.End == t.Len() && start > 0 && !strings.HasSuffix(text, "\n") {
		// Deleting the last line takes the newline before it, so that no
		// empty line is left behind.
		start--
	}
	if op == 'c' {
		h.Replace(start, r.End, "")
		s.enterInsert(h, start)
		return
	}
	h.Replace(start, r.End, "")
	if r.Linewise {
		s.setNormal(h, firstNonBlank(h.Text(), min(start, h.Text().Len())))
		return
	}
	s.setNormal(h, start)
}

// changeLineRange changes whole lines, leaving one empty line and entering
// insert on it, as cj and cG do.
func (s *Session) changeLineRange(h Host, r Range) {
	t := h.Text()
	text := t.Slice(r.Start, r.End)
	s.setRegister(h, text, true)
	if h.ReadOnly() {
		s.setNormal(h, r.Start)
		return
	}
	start := r.Start
	if r.End == t.Len() && start > 0 && !strings.HasSuffix(text, "\n") {
		start--
	}
	replacement := ""
	if strings.Contains(text, "\n") {
		// Vim keeps one empty line where whole lines were changed.
		replacement = "\n"
	}
	h.Replace(start, r.End, replacement)
	s.enterInsert(h, start)
}

// applyObject is an operator with a text object: diw, ci", ya(.
func (s *Session) applyObject(a rune, obj rune, h Host) {
	r, ok := textObjectRange(h.Text(), s.cursor(h), a, obj)
	if !ok {
		s.cancelOperator()
		return
	}
	s.applyRange(s.op, r, h)
	s.cancelOperator()
}
