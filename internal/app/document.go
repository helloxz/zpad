package app

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/helloxz/zpad/internal/atomicfile"
	"github.com/helloxz/zpad/internal/draft"
	"github.com/helloxz/zpad/internal/encoding"
	"github.com/helloxz/zpad/locales"
)

const (
	maxLines         = 500_000
	maxHighlightLine = 128 << 10
	gcDocumentBytes  = 10 << 20
	sniffBytes       = 8192
)

// Document is an open file and the text the editor edits.
type Document struct {
	ID       string
	Path     string
	Buffer   *editor.Buffer
	Encoding encoding.Kind
	// Language is the file's language, shown in the status bar; the buffer
	// gets it only when the file is short enough to highlight.
	Language *editor.Language
	Size     int64
	ModTime  time.Time
	ReadOnly bool
	// UntitledSeq is the number the document was created with, for the name
	// its save dialog starts from; 0 for a document with a path.
	UntitledSeq int

	// DraftID names the recovery copy of the document's unsaved text. It
	// stays the same across renames and saves, so that a copy is dropped or
	// rewritten rather than looked for by path.
	DraftID string

	// The recovery copy's state. draftEpoch is read by the goroutine that
	// writes a copy, so it is atomic; the rest is the main thread's.
	draftEpoch       atomic.Uint64
	draftSeenVersion uint64
	draftEditAt      time.Time
	draftSince       time.Time
	draftBacked      uint64
	draftExists      bool

	// md are the Markdown preview's options, shared by every preview of the
	// document; nil until one shows. mdPath is the path their image base
	// came from, and previewText caches the buffer at previewVersion.
	md             *markdown.Options
	mdPath         string
	mdThemeUI      *ui.Theme
	mdFontSize     float32
	mdFontFamily   string
	previewText    string
	previewVersion uint64
	previewLoaded  bool
}

// setLanguage gives the buffer the file's language, or none when the text
// is too long to highlight, and remembers the language for the status bar.
// It runs after the text is in the buffer: both reloads replace the text
// first. A file's name is the only thing that says its language: an
// untitled document's choice (selectLanguage) goes when it is saved.
func (d *Document) setLanguage(path string) {
	d.applyLanguage(editor.LanguageFor(path), filepath.Base(path))
}

// selectLanguage gives an untitled document the language chosen in the
// status bar, which is what its save starts from; the extension it is saved
// with decides the language from then on.
func (d *Document) selectLanguage(lang *editor.Language) {
	d.applyLanguage(lang, "untitled")
}

// applyLanguage gives the buffer lang, or none when the text is read-only,
// too long or minified onto a huge single line, and remembers it for the
// status bar. name is what the log calls the document.
func (d *Document) applyLanguage(lang *editor.Language, name string) {
	d.Language = lang
	if lang != nil {
		switch {
		case d.Buffer.LineCount() > maxLines:
			log.Printf("no highlight: %s %d lines > %d (%s)", name,
				d.Buffer.LineCount(), maxLines, lang.Name)
			d.Buffer.SetLanguage(nil)
			return
		case d.ReadOnly:
			log.Printf("no highlight: %s %s read-only (%s)", name,
				formatSize(int64(d.Buffer.Len())), lang.Name)
			d.Buffer.SetLanguage(nil)
			return
		case hasLongLine(d.Buffer, maxHighlightLine):
			log.Printf("no highlight: %s line > %d bytes (%s)", name,
				maxHighlightLine, lang.Name)
			d.Buffer.SetLanguage(nil)
			return
		}
	}
	d.Buffer.SetLanguage(lang)
}

// hasLongLine reports whether any line of buf exceeds limit bytes, walking
// chunks without allocating strings.
func hasLongLine(buf *editor.Buffer, limit int) bool {
	if limit <= 0 || buf.Len() <= limit {
		return false
	}
	snap := buf.Snapshot()
	run := 0
	found := false
	snap.Chunks(0, snap.Len(), func(s string) bool {
		if len(s) <= limit {
			first := strings.IndexByte(s, '\n')
			if first < 0 {
				run += len(s)
				if run > limit {
					found = true
					return false
				}
				return true
			}
			if run+first > limit {
				found = true
				return false
			}
			last := strings.LastIndexByte(s, '\n')
			run = len(s) - 1 - last
			return true
		}
		for {
			i := strings.IndexByte(s, '\n')
			if i < 0 {
				run += len(s)
				if run > limit {
					found = true
					return false
				}
				return true
			}
			if run+i > limit {
				found = true
				return false
			}
			run = 0
			s = s[i+1:]
		}
	})
	return found
}

// loadedText holds a file's decoded UTF-8 text and disk metadata before it
// enters a buffer, so reloads update an existing buffer directly rather than
// building and discarding a second one.
type loadedText struct {
	path     string
	text     string
	encoding encoding.Kind
	size     int64
	modTime  time.Time
	maxBytes int64
}

func maxFileBytes(maxMB int) int64 {
	if maxMB <= 0 {
		maxMB = 20
	}
	return int64(maxMB) << 20
}

// sniffOpenable checks a file's metadata and first 8 KB without allocating or
// reading the rest of the file, rejecting directories and binary files up
// front before prompting about large files.
func sniffOpenable(path string) (os.FileInfo, error) {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, errors.New(locales.IsFolder())
	}
	var head [sniffBytes]byte
	n, err := io.ReadFull(f, head[:])
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, err
	}
	if !encoding.IsUTF16(head[:n]) && isBinary(head[:n]) {
		return nil, errors.New(locales.BinaryFile())
	}
	return info, nil
}

// readDocumentBytes opens path once, checks its first 8 KB on the stack to
// reject binary files without reading the rest, and returns the owned file
// bytes when it passes.
func readDocumentBytes(path string, kindHint *encoding.Kind) (string, []byte, os.FileInfo, error) {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	f, err := os.Open(path)
	if err != nil {
		return "", nil, nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", nil, nil, err
	}
	if info.IsDir() {
		return "", nil, nil, errors.New(locales.IsFolder())
	}

	var head [sniffBytes]byte
	n, err := io.ReadFull(f, head[:])
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", nil, nil, err
	}
	utf16 := false
	if kindHint != nil {
		utf16 = *kindHint == encoding.UTF16LE || *kindHint == encoding.UTF16BE
	} else {
		utf16 = encoding.IsUTF16(head[:n])
	}
	if !utf16 && isBinary(head[:n]) {
		return "", nil, nil, errors.New(locales.BinaryFile())
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		data := make([]byte, n)
		copy(data, head[:n])
		return path, data, info, nil
	}

	size := info.Size()
	if size <= int64(n) {
		rest, rerr := io.ReadAll(f)
		if rerr != nil {
			return "", nil, nil, rerr
		}
		data := make([]byte, n+len(rest))
		copy(data, head[:n])
		copy(data[n:], rest)
		return path, data, info, nil
	}

	data := make([]byte, size)
	copy(data, head[:n])
	read := n
	for read < len(data) {
		k, rerr := f.Read(data[read:])
		read += k
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				data = data[:read]
				return path, data, info, nil
			}
			return "", nil, nil, rerr
		}
	}
	// Read any bytes appended since Stat.
	extra, rerr := io.ReadAll(f)
	if rerr != nil {
		return "", nil, nil, rerr
	}
	if len(extra) > 0 {
		data = append(data, extra...)
	}
	return path, data, info, nil
}

// loadDocument reads a file and detects its encoding.
func loadDocument(path string, maxMB int) (*Document, error) {
	maxBytes := maxFileBytes(maxMB)
	absPath, data, info, err := readDocumentBytes(path, nil)
	if err != nil {
		return nil, err
	}
	text, kind, err := encoding.DetectAndDecodeOwned(data)
	if err != nil {
		if errors.Is(err, encoding.ErrUnknownEncoding) {
			return nil, errors.New(locales.UnknownEncoding())
		}
		return nil, errors.New(locales.DecodeFailed())
	}
	size := int64(len(data))
	doc := &Document{
		ID:       absPath,
		Path:     absPath,
		Buffer:   editor.NewBuffer(text),
		Encoding: kind,
		Size:     size,
		ModTime:  info.ModTime(),
		DraftID:  draft.NewID(),
	}
	doc.ReadOnly = size > maxBytes || doc.Buffer.LineCount() > maxLines
	doc.setLanguage(absPath)
	return doc, nil
}

func readDocumentWith(path string, kind encoding.Kind, maxMB int) (loadedText, error) {
	maxBytes := maxFileBytes(maxMB)
	absPath, data, info, err := readDocumentBytes(path, &kind)
	if err != nil {
		return loadedText{}, err
	}
	text, err := encoding.DecodeOwned(data, kind)
	if err != nil {
		return loadedText{}, errors.New(locales.DecodeFailed())
	}
	return loadedText{
		path:     absPath,
		text:     text,
		encoding: kind,
		size:     int64(len(data)),
		modTime:  info.ModTime(),
		maxBytes: maxBytes,
	}, nil
}

// loadDocumentWith reads a file again, interpreting its bytes as kind.
func loadDocumentWith(path string, kind encoding.Kind, maxMB int) (*Document, error) {
	loaded, err := readDocumentWith(path, kind, maxMB)
	if err != nil {
		return nil, err
	}
	doc := &Document{
		ID:       loaded.path,
		Path:     loaded.path,
		Buffer:   editor.NewBuffer(loaded.text),
		Encoding: loaded.encoding,
		Size:     loaded.size,
		ModTime:  loaded.modTime,
		DraftID:  draft.NewID(),
	}
	doc.ReadOnly = loaded.size > loaded.maxBytes || doc.Buffer.LineCount() > maxLines
	doc.setLanguage(loaded.path)
	return doc, nil
}

func newUntitled(seq int, defaultEncoding string) *Document {
	buf := editor.NewBuffer("")
	return &Document{
		ID:          fmt.Sprintf("untitled-%d", seq),
		Buffer:      buf,
		Encoding:    encoding.Parse(defaultEncoding),
		UntitledSeq: seq,
		DraftID:     draft.NewID(),
	}
}

// name is what the tab, the title and the prompts call the document: a saved
// file by its file name, an untitled one by the number it was created with.
// The number stays as the other tabs close, so that a prompt naming one points
// at it rather than at whichever tab moved into its place.
func (d *Document) Name() string {
	if d.Path == "" {
		return d.untitledName()
	}
	return filepath.Base(d.Path)
}

func (d *Document) untitledName() string {
	if d.UntitledSeq <= 0 {
		return locales.Untitled()
	}
	return locales.UntitledN(d.UntitledSeq)
}

// saveName is the file name a save dialog starts with. An untitled
// document gets an English name with its creation number and the text
// extension, so the dialog does not offer a localized name that the user
// would have to replace.
func (d *Document) saveName() string {
	if d.Path != "" {
		return filepath.Base(d.Path)
	}
	seq := d.UntitledSeq
	if seq <= 0 {
		seq = 1
	}
	return fmt.Sprintf("untitled-%d%s", seq, d.languageExtension())
}

// languageExtension is the extension an untitled document is saved with: the
// chosen language's first one, so that the saved file is found by the same
// language, else plain text's.
func (d *Document) languageExtension() string {
	if d.Path == "" && d.Language != nil && len(d.Language.Extensions) > 0 {
		return d.Language.Extensions[0]
	}
	return ".txt"
}

// withExtension gives path ext when its name has none, so that a file saved
// under a name typed without one is still found by its language.
func withExtension(path, ext string) string {
	if ext == "" || filepath.Ext(path) != "" {
		return path
	}
	return path + ext
}

// DisplayPath is the path relative to root when it is inside it.
func (d *Document) DisplayPath(root string) string {
	if d.Path == "" {
		return d.untitledName()
	}
	if root != "" {
		if rel, err := filepath.Rel(root, d.Path); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return rel
		}
	}
	return d.Path
}

// writeDocument streams d to path in its encoding and line endings, keeping
// the file's mode when it exists and returning the bytes written.
func writeDocument(d *Document, path string) (int64, error) {
	snap := d.Buffer.Snapshot()
	var written int64
	err := atomicfile.WriteFileTo(path, 0o644, func(w io.Writer) error {
		n, err := encoding.EncodeStream(w, d.Encoding, func(uw io.Writer) error {
			_, err := snap.WriteTo(uw)
			return err
		})
		written = n
		return err
	})
	if err != nil {
		if errors.Is(err, encoding.ErrInvalidEncoding) {
			return 0, errors.New(locales.InvalidEncoding())
		}
		return 0, err
	}
	return written, nil
}

func isBinary(data []byte) bool {
	n := len(data)
	if n > sniffBytes {
		n = sniffBytes
	}
	for i := 0; i < n; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}
