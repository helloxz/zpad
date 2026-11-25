package vim

import (
	"strings"
	"unicode/utf8"
)

// textObjectRange returns the text object around off, as diw, ci" and ya(
// take it. a is i or a: inside the object, or around it with its
// delimiters.
func textObjectRange(t Buffer, off int, a rune, obj rune) (Range, bool) {
	around := a == 'a'
	switch obj {
	case 'w', 'W':
		return wordObject(t, off, around, obj == 'W')
	case '"', '\'', '`':
		return quoteObject(t, off, around, obj)
	case '(', ')', '[', ']', '{', '}', '<', '>':
		return bracketObject(t, off, around, obj)
	case 'p':
		return paragraphObject(t, off, around)
	}
	return Range{}, false
}

// wordObject selects the word under the caret, or around it with the spaces
// after it.
func wordObject(t Buffer, off int, around, big bool) (Range, bool) {
	line := t.LineOf(off)
	start, end := t.LineStart(line), t.LineEnd(line)
	if start == end {
		return Range{}, false
	}
	if off >= end {
		off = prevRune(t, end)
	}
	cls := classAt(t, off, big)
	a := off
	for a > start && classAt(t, prevRune(t, a), big) == cls {
		a = prevRune(t, a)
	}
	z := off
	for z < end && classAt(t, z, big) == cls {
		z = nextRune(t, z)
	}
	if around {
		if cls == classSpace {
			// Vim's aw on spaces takes the spaces and the word after.
			for z < end && classAt(t, z, big) == classSpace {
				z = nextRune(t, z)
			}
			if z < end {
				c := classAt(t, z, big)
				for z < end && classAt(t, z, big) == c {
					z = nextRune(t, z)
				}
			}
		} else {
			// Trailing spaces join the word; without any, the leading ones
			// do, as Vim's aw.
			spaces := z
			for spaces < end && classAt(t, spaces, big) == classSpace {
				spaces = nextRune(t, spaces)
			}
			if spaces > z {
				z = spaces
			} else {
				for a > start && classAt(t, prevRune(t, a), big) == classSpace {
					a = prevRune(t, a)
				}
			}
		}
	}
	if a == z {
		return Range{}, false
	}
	return Range{Start: a, End: z}, true
}

// quoteObject selects the text between the quotes around the caret, or the
// quotes too.
func quoteObject(t Buffer, off int, around bool, quote rune) (Range, bool) {
	line := t.LineOf(off)
	start, end := t.LineStart(line), t.LineEnd(line)
	text := t.Slice(start, end)
	col := off - start
	var opens []int
	for i := 0; i < len(text); {
		r, n := utf8.DecodeRuneInString(text[i:])
		if r == quote && !escaped(text, i) {
			opens = append(opens, i)
		}
		i += n
	}
	for i := 0; i+1 < len(opens); i += 2 {
		open, close := opens[i], opens[i+1]
		if col >= open && col <= close {
			if around {
				return Range{Start: start + open, End: start + close + 1}, true
			}
			return Range{Start: start + open + 1, End: start + close}, true
		}
	}
	// From outside a quoted string, the next one on the line is the object,
	// as di" finds it.
	for i := 0; i+1 < len(opens); i += 2 {
		open, close := opens[i], opens[i+1]
		if open >= col {
			if around {
				return Range{Start: start + open, End: start + close + 1}, true
			}
			return Range{Start: start + open + 1, End: start + close}, true
		}
	}
	return Range{}, false
}

// escaped reports whether the byte at i in text is preceded by an odd
// number of backslashes.
func escaped(text string, i int) bool {
	n := 0
	for i > 0 && text[i-1] == '\\' {
		n++
		i--
	}
	return n%2 == 1
}

// bracketObject selects the text inside the nearest brackets around the
// caret, or the brackets too.
func bracketObject(t Buffer, off int, around bool, bracket rune) (Range, bool) {
	var open, close rune
	switch bracket {
	case '(':
		open, close = '(', ')'
	case ')':
		open, close = '(', ')'
	case '[':
		open, close = '[', ']'
	case ']':
		open, close = '[', ']'
	case '{':
		open, close = '{', '}'
	case '}':
		open, close = '{', '}'
	case '<':
		open, close = '<', '>'
	case '>':
		open, close = '<', '>'
	}
	line := t.LineOf(off)
	start, end := t.LineStart(line), t.LineEnd(line)
	text := t.Slice(start, end)
	col := off - start
	type pair struct{ open, close int }
	var stack []int
	var pairs []pair
	for i := 0; i < len(text); {
		r, n := utf8.DecodeRuneInString(text[i:])
		switch r {
		case open:
			stack = append(stack, i)
		case close:
			if len(stack) > 0 {
				o := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				pairs = append(pairs, pair{o, i})
			}
		}
		i += n
	}
	best := -1
	for i, p := range pairs {
		if col >= p.open && col <= p.close {
			if best < 0 || p.close-p.open < pairs[best].close-pairs[best].open {
				best = i
			}
		}
	}
	if best < 0 {
		return Range{}, false
	}
	p := pairs[best]
	if around {
		return Range{Start: start + p.open, End: start + p.close + 1}, true
	}
	return Range{Start: start + p.open + 1, End: start + p.close}, true
}

// paragraphObject selects the paragraph around the caret, or that and the
// blank lines after it.
func paragraphObject(t Buffer, off int, around bool) (Range, bool) {
	line := t.LineOf(off)
	count := t.LineCount()
	blank := func(i int) bool { return strings.TrimSpace(t.Line(i)) == "" }
	first, last := line, line
	for first > 0 && !blank(first-1) {
		first--
	}
	for last+1 < count && !blank(last+1) {
		last++
	}
	if around {
		for last+1 < count && blank(last+1) {
			last++
		}
	}
	start := t.LineStart(first)
	end := t.Len()
	if last+1 < count {
		end = t.LineStart(last + 1)
	}
	if start == end {
		return Range{}, false
	}
	return Range{Start: start, End: end, Linewise: true}, true
}
