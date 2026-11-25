// Package encoding converts text files between UTF-8 and the encodings
// ZPad edits. The editor itself works on UTF-8.
package encoding

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// Kind is a text encoding.
type Kind string

const (
	UTF8    Kind = "utf-8"
	UTF8BOM Kind = "utf-8-bom"
	UTF16LE Kind = "utf-16le"
	UTF16BE Kind = "utf-16be"
	GB18030 Kind = "gb18030"
)

var (
	// ErrUnknownEncoding reports that no supported encoding could decode a
	// file's bytes.
	ErrUnknownEncoding = errors.New("unknown encoding")
	// ErrInvalidEncoding reports that text could not be encoded in the
	// target encoding.
	ErrInvalidEncoding = errors.New("invalid encoding")
)

// All lists the encodings the status bar offers.
var All = []Kind{UTF8, UTF8BOM, UTF16LE, UTF16BE, GB18030}

// Label is the name shown in the UI.
func Label(k Kind) string {
	switch k {
	case UTF8:
		return "UTF-8"
	case UTF8BOM:
		return "UTF-8 BOM"
	case UTF16LE:
		return "UTF-16 LE"
	case UTF16BE:
		return "UTF-16 BE"
	case GB18030:
		return "GB18030"
	}
	return string(k)
}

// Parse turns a saved setting into a Kind, defaulting to UTF-8.
func Parse(s string) Kind {
	for _, k := range All {
		if string(k) == s {
			return k
		}
	}
	return UTF8
}

var (
	bomUTF8    = []byte{0xEF, 0xBB, 0xBF}
	bomUTF16LE = []byte{0xFF, 0xFE}
	bomUTF16BE = []byte{0xFE, 0xFF}
)

// IsUTF16 reports whether head starts with a UTF-16 BOM or matches the UTF-16
// NUL-byte pattern. Only the first 4096 bytes are inspected, so callers can
// distinguish UTF-16 from binary files using a small header read.
func IsUTF16(head []byte) bool {
	if bytes.HasPrefix(head, bomUTF16LE) || bytes.HasPrefix(head, bomUTF16BE) {
		return true
	}
	_, ok := detectUTF16(head)
	return ok
}

// Detect reads the encoding of data from its BOM, its UTF-16 byte pattern
// or its UTF-8 validity, falling back to GB18030.
func Detect(data []byte) (Kind, error) {
	switch {
	case bytes.HasPrefix(data, bomUTF8):
		return UTF8BOM, nil
	case bytes.HasPrefix(data, bomUTF16LE):
		return UTF16LE, nil
	case bytes.HasPrefix(data, bomUTF16BE):
		return UTF16BE, nil
	}
	if k, ok := detectUTF16(data); ok {
		return k, nil
	}
	if utf8.Valid(data) {
		return UTF8, nil
	}
	// GB18030 covers all Unicode code points and is the fallback for
	// Chinese legacy files. If it cannot decode the bytes, let the user
	// pick another encoding from the status bar.
	if _, err := decodeGB18030Owned(data); err == nil {
		return GB18030, nil
	}
	return GB18030, ErrUnknownEncoding
}

// DetectAndDecodeOwned detects the encoding of data and decodes it in one
// pass. Callers must own data and never mutate it afterward, allowing UTF-8
// text to be handed to the rope builder without a second validation pass or
// a full-file string copy, and GB18030 to be decoded once instead of twice.
func DetectAndDecodeOwned(data []byte) (string, Kind, error) {
	switch {
	case bytes.HasPrefix(data, bomUTF8):
		body := data[len(bomUTF8):]
		if !utf8.Valid(body) {
			return "", UTF8BOM, errors.New("invalid UTF-8")
		}
		return bytesToString(body), UTF8BOM, nil
	case bytes.HasPrefix(data, bomUTF16LE):
		s, err := decodeUTF16Owned(data, unicode.LittleEndian)
		return s, UTF16LE, err
	case bytes.HasPrefix(data, bomUTF16BE):
		s, err := decodeUTF16Owned(data, unicode.BigEndian)
		return s, UTF16BE, err
	}
	if k, ok := detectUTF16(data); ok {
		if k == UTF16LE {
			s, err := decodeUTF16Owned(data, unicode.LittleEndian)
			return s, UTF16LE, err
		}
		s, err := decodeUTF16Owned(data, unicode.BigEndian)
		return s, UTF16BE, err
	}
	if utf8.Valid(data) {
		return bytesToString(data), UTF8, nil
	}
	s, err := decodeGB18030Owned(data)
	if err != nil {
		return "", GB18030, ErrUnknownEncoding
	}
	return s, GB18030, nil
}

func detectUTF16(data []byte) (Kind, bool) {
	if len(data) < 4 {
		return "", false
	}
	n := len(data)
	if n > 4096 {
		n = 4096
	}
	var evenNul, oddNul int
	for i := 0; i < n; i++ {
		if data[i] != 0 {
			continue
		}
		if i%2 == 0 {
			evenNul++
		} else {
			oddNul++
		}
	}
	pairs := n / 2
	if pairs == 0 {
		return "", false
	}
	// UTF-16LE ASCII has NULs at odd offsets; UTF-16BE at even ones.
	if oddNul > pairs/3 && evenNul < pairs/10 {
		return UTF16LE, true
	}
	if evenNul > pairs/3 && oddNul < pairs/10 {
		return UTF16BE, true
	}
	return "", false
}

// Decode returns data as UTF-8 text.
func Decode(data []byte, kind Kind) (string, error) {
	switch kind {
	case UTF8, UTF8BOM:
		if bytes.HasPrefix(data, bomUTF8) {
			data = data[len(bomUTF8):]
		}
		if !utf8.Valid(data) {
			return "", errors.New("invalid UTF-8")
		}
		return string(data), nil
	case UTF16LE:
		return decodeUTF16Owned(data, unicode.LittleEndian)
	case UTF16BE:
		return decodeUTF16Owned(data, unicode.BigEndian)
	case GB18030:
		return decodeGB18030Owned(data)
	default:
		return "", fmt.Errorf("unsupported encoding %q", kind)
	}
}

// DecodeOwned decodes data in kind, reusing data without copying when it is
// already UTF-8. Callers must own data and never mutate it afterward.
func DecodeOwned(data []byte, kind Kind) (string, error) {
	if kind == UTF8 || kind == UTF8BOM {
		if bytes.HasPrefix(data, bomUTF8) {
			data = data[len(bomUTF8):]
		}
		if !utf8.Valid(data) {
			return "", errors.New("invalid UTF-8")
		}
		return bytesToString(data), nil
	}
	return Decode(data, kind)
}

func decodeUTF16Owned(data []byte, order unicode.Endianness) (string, error) {
	dec := unicode.UTF16(order, unicode.UseBOM).NewDecoder()
	out, _, err := transform.Bytes(dec, data)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(out) {
		return "", errors.New("invalid UTF-16")
	}
	return bytesToString(out), nil
}

func decodeGB18030Owned(data []byte) (string, error) {
	dec := simplifiedchinese.GB18030.NewDecoder()
	out, _, err := transform.Bytes(dec, data)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(out) {
		return "", errors.New("invalid GB18030")
	}
	return bytesToString(out), nil
}

func bytesToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// Encode returns text in kind.
func Encode(text string, kind Kind) ([]byte, error) {
	switch kind {
	case UTF8:
		return []byte(text), nil
	case UTF8BOM:
		out := make([]byte, len(bomUTF8)+len(text))
		copy(out, bomUTF8)
		copy(out[len(bomUTF8):], text)
		return out, nil
	case UTF16LE:
		return encodeUTF16(text, unicode.LittleEndian)
	case UTF16BE:
		return encodeUTF16(text, unicode.BigEndian)
	case GB18030:
		enc := simplifiedchinese.GB18030.NewEncoder()
		out, _, err := transform.Bytes(enc, []byte(text))
		if err != nil {
			return nil, err
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported encoding %q", kind)
	}
}

func encodeUTF16(text string, order unicode.Endianness) ([]byte, error) {
	// UseBOM writes a byte order mark, as UTF-16 files usually have.
	enc := unicode.UTF16(order, unicode.UseBOM).NewEncoder()
	out, _, err := transform.Bytes(enc, []byte(text))
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EncodeStream encodes the UTF-8 text written by writeUTF8 into dst in kind,
// returning the number of encoded bytes written to dst. Any invalid UTF-8 or
// unencodable sequence returns ErrInvalidEncoding.
func EncodeStream(dst io.Writer, kind Kind, writeUTF8 func(io.Writer) error) (int64, error) {
	cw := &countWriter{w: dst}
	switch kind {
	case UTF8:
		vw := &utf8CheckWriter{w: cw}
		if err := writeUTF8(vw); err != nil {
			return cw.n, err
		}
		if err := vw.Close(); err != nil {
			return cw.n, err
		}
		return cw.n, nil
	case UTF8BOM:
		if _, err := cw.Write(bomUTF8); err != nil {
			return cw.n, err
		}
		vw := &utf8CheckWriter{w: cw}
		if err := writeUTF8(vw); err != nil {
			return cw.n, err
		}
		if err := vw.Close(); err != nil {
			return cw.n, err
		}
		return cw.n, nil
	case UTF16LE:
		return encodeTransformStream(cw, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder(), writeUTF8)
	case UTF16BE:
		return encodeTransformStream(cw, unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewEncoder(), writeUTF8)
	case GB18030:
		return encodeTransformStream(cw, simplifiedchinese.GB18030.NewEncoder(), writeUTF8)
	default:
		return 0, fmt.Errorf("%w: %q", ErrInvalidEncoding, kind)
	}
}

func encodeTransformStream(cw *countWriter, t transform.Transformer, writeUTF8 func(io.Writer) error) (int64, error) {
	tw := transform.NewWriter(cw, t)
	vw := &utf8CheckWriter{w: tw}
	if err := writeUTF8(vw); err != nil {
		if errors.Is(err, ErrInvalidEncoding) {
			return cw.n, err
		}
		return cw.n, fmt.Errorf("%w: %v", ErrInvalidEncoding, err)
	}
	if err := vw.Close(); err != nil {
		if errors.Is(err, ErrInvalidEncoding) {
			return cw.n, err
		}
		return cw.n, fmt.Errorf("%w: %v", ErrInvalidEncoding, err)
	}
	if err := tw.Close(); err != nil {
		return cw.n, fmt.Errorf("%w: %v", ErrInvalidEncoding, err)
	}
	return cw.n, nil
}

type countWriter struct {
	w io.Writer
	n int64
}

func (c *countWriter) Write(p []byte) (int, error) {
	k, err := c.w.Write(p)
	c.n += int64(k)
	return k, err
}

func (c *countWriter) WriteString(s string) (int, error) {
	k, err := io.WriteString(c.w, s)
	c.n += int64(k)
	return k, err
}

type utf8CheckWriter struct {
	w        io.Writer
	carry    [utf8.UTFMax]byte
	n        int
	sawBytes bool
}

func (u *utf8CheckWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	u.sawBytes = true
	total := len(p)
	if u.n > 0 {
		for len(p) > 0 && !utf8.FullRune(u.carry[:u.n]) {
			u.carry[u.n] = p[0]
			u.n++
			p = p[1:]
		}
		if !utf8.FullRune(u.carry[:u.n]) {
			return total, nil
		}
		r, width := utf8.DecodeRune(u.carry[:u.n])
		if r == utf8.RuneError && width == 1 {
			return 0, ErrInvalidEncoding
		}
		if _, err := u.w.Write(u.carry[:u.n]); err != nil {
			return 0, err
		}
		u.n = 0
	}
	end := len(p)
	for i := 1; i < utf8.UTFMax && i <= len(p); i++ {
		b := p[len(p)-i]
		if (b & 0xC0) == 0x80 {
			continue
		}
		if !utf8.FullRune(p[len(p)-i:]) {
			end = len(p) - i
		}
		break
	}
	if end > 0 {
		if !utf8.Valid(p[:end]) {
			return 0, ErrInvalidEncoding
		}
		if _, err := u.w.Write(p[:end]); err != nil {
			return 0, err
		}
	}
	if end < len(p) {
		u.n = copy(u.carry[:], p[end:])
	}
	return total, nil
}

func (u *utf8CheckWriter) WriteString(s string) (int, error) {
	if len(s) == 0 {
		return 0, nil
	}
	if u.n == 0 && s[len(s)-1] < utf8.RuneSelf {
		if !utf8.ValidString(s) {
			return 0, ErrInvalidEncoding
		}
		u.sawBytes = true
		return io.WriteString(u.w, s)
	}
	return u.Write(unsafe.Slice(unsafe.StringData(s), len(s)))
}

func (u *utf8CheckWriter) Close() error {
	if u.n > 0 {
		if !utf8.Valid(u.carry[:u.n]) {
			return ErrInvalidEncoding
		}
		_, err := u.w.Write(u.carry[:u.n])
		u.n = 0
		return err
	}
	return nil
}
