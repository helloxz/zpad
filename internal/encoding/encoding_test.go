package encoding

import (
	"bytes"
	"io"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	texts := []string{
		"hello\nworld\n",
		"中文测试\n第二行\n",
		"mixed 中文 and English\n",
		"",
	}
	for _, kind := range All {
		for _, text := range texts {
			data, err := Encode(text, kind)
			if err != nil {
				t.Fatalf("Encode(%s): %v", kind, err)
			}
			got, err := Decode(data, kind)
			if err != nil {
				t.Fatalf("Decode(%s): %v", kind, err)
			}
			if got != text {
				t.Fatalf("%s round trip: got %q, want %q", kind, got, text)
			}
		}
	}
}

func TestDetectBOM(t *testing.T) {
	cases := []struct {
		kind Kind
		text string
	}{
		{UTF8BOM, "hello"},
		{UTF16LE, "hello"},
		{UTF16BE, "hello"},
	}
	for _, tc := range cases {
		data, err := Encode(tc.text, tc.kind)
		if err != nil {
			t.Fatal(err)
		}
		got, err := Detect(data)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.kind {
			t.Fatalf("Detect(%s) = %s, want %s", tc.kind, got, tc.kind)
		}
	}
}

func TestDetectUTF16WithoutBOM(t *testing.T) {
	for _, kind := range []Kind{UTF16LE, UTF16BE} {
		data, err := Encode("hello\nworld\n", kind)
		if err != nil {
			t.Fatal(err)
		}
		// Strip the BOM so the heuristic has to decide.
		data = data[2:]
		got, err := Detect(data)
		if err != nil {
			t.Fatal(err)
		}
		if got != kind {
			t.Fatalf("Detect(%s without BOM) = %s, want %s", kind, got, kind)
		}
	}
}

func TestDetectUTF8AndGB18030(t *testing.T) {
	if got, err := Detect([]byte("plain text\n")); err != nil || got != UTF8 {
		t.Fatalf("Detect UTF-8 = %s, %v; want UTF-8", got, err)
	}
	data, err := Encode("中文测试\n", GB18030)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := Detect(data); err != nil || got != GB18030 {
		t.Fatalf("Detect GB18030 = %s, %v; want GB18030", got, err)
	}
}

func TestInvalidUTF8(t *testing.T) {
	data := []byte{0xff, 0xfe, 0xfd}
	if _, err := Decode(data, UTF8); err == nil {
		t.Fatal("invalid UTF-8 decoded")
	}
}

func TestParseAndLabel(t *testing.T) {
	for _, kind := range All {
		if got := Parse(string(kind)); got != kind {
			t.Fatalf("Parse(%q) = %q", kind, got)
		}
		if Label(kind) == "" {
			t.Fatalf("Label(%q) is empty", kind)
		}
	}
	if Parse("unknown") != UTF8 {
		t.Fatal("unknown encoding did not fall back to UTF-8")
	}
}

func TestUTF8BOMEncode(t *testing.T) {
	data, err := Encode("hi", UTF8BOM)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, bomUTF8) {
		t.Fatal("UTF-8 BOM missing")
	}
}

func TestGB18030FourByteSequence(t *testing.T) {
	text := "𠀀"
	data, err := Encode(text, GB18030)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 4 {
		t.Fatalf("GB18030 encoded %q in %d bytes, want 4", text, len(data))
	}
	got, err := Decode(data, GB18030)
	if err != nil {
		t.Fatal(err)
	}
	if got != text {
		t.Fatalf("got %q, want %q", got, text)
	}
}

func TestDecodeUTF16WithoutBOMExplicitly(t *testing.T) {
	for _, kind := range []Kind{UTF16LE, UTF16BE} {
		data, err := Encode("hello\nworld", kind)
		if err != nil {
			t.Fatal(err)
		}
		got, err := Decode(data[2:], kind)
		if err != nil {
			t.Fatal(err)
		}
		if got != "hello\nworld" {
			t.Fatalf("%s without BOM: got %q", kind, got)
		}
	}
}

func TestEncodeStreamMatchesEncodeAcrossChunkSplits(t *testing.T) {
	texts := []string{
		"",
		"hello\nworld\n",
		"中文测试\r\n第二行\n𠀀四字节字符\n",
	}
	for _, kind := range All {
		for _, text := range texts {
			want, err := Encode(text, kind)
			if err != nil {
				t.Fatalf("Encode(%s): %v", kind, err)
			}
			// Feed one byte at a time to split every multi-byte UTF-8 rune.
			var out bytes.Buffer
			n, err := EncodeStream(&out, kind, func(w io.Writer) error {
				for i := 0; i < len(text); i++ {
					if _, err := w.Write([]byte{text[i]}); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				t.Fatalf("EncodeStream(%s, %q): %v", kind, text, err)
			}
			if n != int64(len(want)) || !bytes.Equal(out.Bytes(), want) {
				t.Fatalf("EncodeStream(%s, %q) = %d bytes %x, want %d bytes %x",
					kind, text, n, out.Bytes(), len(want), want)
			}
		}
	}

	// Invalid UTF-8 must be rejected in every encoding.
	for _, kind := range All {
		var out bytes.Buffer
		_, err := EncodeStream(&out, kind, func(w io.Writer) error {
			_, err := w.Write([]byte{0xff, 0xfe, 'a'})
			return err
		})
		if err == nil {
			t.Fatalf("EncodeStream(%s) accepted invalid UTF-8", kind)
		}
	}
}

func TestDetectAndDecodeOwned(t *testing.T) {
	for _, kind := range All {
		want := "中文与 ASCII\n𠀀\n"
		raw, err := Encode(want, kind)
		if err != nil {
			t.Fatal(err)
		}
		got, gotKind, err := DetectAndDecodeOwned(bytes.Clone(raw))
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if gotKind != kind || got != want {
			t.Fatalf("%s: got (%q, %s), want (%q, %s)", kind, got, gotKind, want, kind)
		}
	}
}
