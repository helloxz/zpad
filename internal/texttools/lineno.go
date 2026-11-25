package texttools

import (
	"strconv"
	"strings"
)

// CountLines returns the number of lines NumberLines numbers in s: one more
// than its breaks, without the empty line a trailing break would start, which
// is not numbered. It is zero for an empty text.
func CountLines(s string) int {
	if s == "" {
		return 0
	}
	n := 1 + strings.Count(s, "\n")
	if strings.HasSuffix(s, "\n") {
		n--
	}
	return n
}

// NumberLines prefixes each line of s with its number, from first on, written
// with at least width digits and followed by sep. A text that ends with a
// break gets no numbered empty line after it, and a blank line inside the
// text is numbered like any other.
//
// The buffer keeps only \n breaks, so a selection read from it never contains
// the document's line endings; the line endings are put back when it saves.
func NumberLines(s, sep string, first, width int) string {
	lines := CountLines(s)
	if lines == 0 {
		return s
	}
	if width < 1 {
		width = 1
	}

	var b strings.Builder
	b.Grow(len(s) + lines*(width+len(sep)))

	n := first
	atLineStart := true
	for i := 0; i < len(s); i++ {
		if atLineStart {
			writeNumber(&b, n, width)
			b.WriteString(sep)
			n++
			atLineStart = false
		}
		c := s[i]
		b.WriteByte(c)
		if c == '\n' {
			atLineStart = true
		}
	}
	return b.String()
}

// writeNumber writes n zero-padded to width; a number with more digits is
// written whole rather than cut.
func writeNumber(b *strings.Builder, n, width int) {
	if n < 0 {
		n = 0
	}
	digits := strconv.Itoa(n)
	for i := len(digits); i < width; i++ {
		b.WriteByte('0')
	}
	b.WriteString(digits)
}
