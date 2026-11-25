package texttools

import "testing"

func TestCountLines(t *testing.T) {
	cases := map[string]int{
		"":       0,
		"a":      1,
		"a\n":    1,
		"a\nb":   2,
		"a\nb\n": 2,
		"\n":     1,
		"a\n\nb": 3,
		"a\n\n":  2,
	}
	for in, want := range cases {
		if got := CountLines(in); got != want {
			t.Errorf("CountLines(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestNumberLines(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		sep   string
		first int
		width int
		want  string
	}{
		{"empty", "", ",", 1, 2, ""},
		{"one line", "a", ",", 1, 2, "01,a"},
		{"no trailing break", "a\nb\nc", ",", 1, 2, "01,a\n02,b\n03,c"},
		{"trailing break", "a\nb\n", ",", 1, 2, "01,a\n02,b\n"},
		{"blank line", "a\n\nb", ",", 1, 2, "01,a\n02,\n03,b"},
		{"dot and space", "a\nb", ". ", 1, 2, "01. a\n02. b"},
		{"right paren", "a\nb", ") ", 1, 2, "01) a\n02) b"},
		{"colon", "a\nb", ": ", 1, 2, "01: a\n02: b"},
		{"five-digit start", "a\nb\nc", ",", 998, 3, "998,a\n999,b\n1000,c"},
		{"wide enough", "a\nb", ",", 9, 2, "09,a\n10,b"},
		{"width one", "a\nb", ":", 1, 1, "1:a\n2:b"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NumberLines(c.in, c.sep, c.first, c.width); got != c.want {
				t.Fatalf("NumberLines(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
