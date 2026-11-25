package app

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"weak"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helloxz/zpad/internal/encoding"
	"github.com/helloxz/zpad/internal/files"
	"github.com/helloxz/zpad/internal/settings"
	"github.com/helloxz/zpad/internal/theme"
	"github.com/helloxz/zpad/locales"
)

// TestClosedDocumentsAreCollected verifies P0 lifecycle fixes: closing tabs
// after opening/closing the diff dialog, searching across files, or toggling
// Markdown preview must leave no strong reference to the closed Document.
func TestClosedDocumentsAreCollected(t *testing.T) {
	a := New(settings.Default(), t.TempDir())

	makeAndClose := func() (weak.Pointer[Document], weak.Pointer[Document]) {
		d1 := docAt("/tmp/leak-a.md", "# Title\nhello match world\n"+strings.Repeat("x", 64<<10))
		d2 := docAt("/tmp/leak-b.txt", "other match text\n"+strings.Repeat("y", 64<<10))
		w1, w2 := weak.Make(d1), weak.Make(d2)

		a.addDocument(d1)
		a.ToggleMarkdownPreview()
		_ = d1.PreviewText()
		a.ToggleMarkdownPreview()
		if d1.previewText != "" || d1.previewLoaded {
			t.Fatalf("leaving preview kept %d bytes of previewText", len(d1.previewText))
		}

		a.addDocument(d2)

		// Run cross-file search so both documents appear in a.searchRows.
		a.openSearchPanel()
		a.searchQuery = "match"
		a.startSearch()
		if len(a.searchRows) == 0 {
			t.Fatal("expected search hits")
		}
		a.closeSearchPanel()

		// Open and dismiss the text comparison dialog via Escape.
		a.openCompareText(nil)
		tt := ui.NewTester(a.View, 1100, 800)
		tt.Frame()
		tt.Key(0, ui.KeyEscape)
		tt.Frame()
		if a.diffA != nil || a.diffB != nil {
			t.Fatalf("closing diff kept diffA=%v diffB=%v", a.diffA, a.diffB)
		}

		// Close both tabs.
		a.removeTab(0, 1)
		a.removeTab(0, 0)
		tt.Frame()
		return w1, w2
	}

	w1, w2 := makeAndClose()
	for i := 0; i < 4; i++ {
		runtime.GC()
	}
	if w1.Value() != nil || w2.Value() != nil {
		t.Fatalf("closed documents leaked: d1=%v d2=%v", w1.Value(), w2.Value())
	}
}

func TestPreviewImageScalingAndLastMarkdownCacheClear(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "huge-local.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 3200, 2000))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(imgPath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	src, err := loadPreviewImage(dir, "huge-local.png")
	if err != nil {
		t.Fatalf("load local image: %v", err)
	}
	bm, ok := src.(*ui.Bitmap)
	if !ok {
		t.Fatalf("got %T, want *ui.Bitmap", src)
	}
	if w, h := bm.Size(); w != 1600 || h != 1000 {
		t.Fatalf("scaled local bitmap = %d×%d, want 1600×1000", w, h)
	}

	// Closing the last Markdown tab clears the remote preview image cache.
	previewImages.put("https://example.com/test.png", bm)
	if _, ok := previewImages.get("https://example.com/test.png"); !ok {
		t.Fatal("expected cached preview image")
	}
	a := New(settings.Default(), dir)
	a.addDocument(docAt(filepath.Join(dir, "README.md"), "# Hi"))
	a.removeTab(0, 0)
	if _, ok := previewImages.get("https://example.com/test.png"); ok {
		t.Fatal("closing the last Markdown tab left previewImages populated")
	}
}

func TestAboutCloseReleasesDecodedIcon(t *testing.T) {
	data, err := os.ReadFile("../../resources/icon.png")
	if err != nil {
		t.Fatal(err)
	}
	a := New(settings.Default(), t.TempDir())
	a.SetAppIcon(data)
	a.openAbout()
	if a.aboutIcon == nil {
		t.Fatal("expected decoded about icon")
	}
	tt := ui.NewTester(a.View, 900, 700)
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.aboutOpen || a.aboutIcon != nil {
		t.Fatalf("closing About left aboutOpen=%v aboutIcon=%v", a.aboutOpen, a.aboutIcon)
	}
}

func TestStatusBarSelectionAndCursorZeroAllocSteadyState(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	doc := docAt("/tmp/utf8.txt", "第一行 hello\n第二行 world\n")
	a.addDocument(doc)
	tab := a.activeTab()
	tab.state.Select(0, doc.Buffer.Len())

	if got, want := selectionLabel(tab), locales.SelectedCount(20); got != want {
		t.Fatalf("selectionLabel = %q, want %q", got, want)
	}
	if got, want := cursorLabel(tab), locales.LineColumn(3, 1); got != want {
		t.Fatalf("cursorLabel = %q, want %q", got, want)
	}
	allocs := testing.AllocsPerRun(200, func() {
		_ = cursorLabel(tab)
	})
	if allocs > 0 {
		t.Fatalf("steady cursorLabel allocates %v per call, want 0", allocs)
	}
}

func TestSameDraftTextStreamingLineEndings(t *testing.T) {
	doc := &Document{Buffer: editor.NewBuffer("alpha\nbeta\ngamma\n")}
	for _, draftText := range []string{
		"alpha\nbeta\ngamma\n",
		"alpha\r\nbeta\r\ngamma\r\n",
		"alpha\rbeta\rgamma\r",
		"alpha\r\nbeta\ngamma\r",
	} {
		if !sameDraftText(doc, draftText) {
			t.Fatalf("sameDraftText(%q) = false, want true", draftText)
		}
	}
	for _, diff := range []string{
		"alpha\nbeta\ngamma",
		"alpha\nbeta\ngamma\n!",
		"alpha\nBeta\ngamma\n",
		"",
	} {
		if sameDraftText(doc, diff) {
			t.Fatalf("sameDraftText(%q) = true, want false", diff)
		}
	}
	allocs := testing.AllocsPerRun(100, func() {
		_ = sameDraftText(doc, "alpha\r\nbeta\r\ngamma\r\n")
	})
	if allocs > 0 {
		t.Fatalf("sameDraftText allocates %v per call, want 0", allocs)
	}
}

func TestMinifiedLongLineAndReadOnlySkipHighlight(t *testing.T) {
	dir := t.TempDir()

	// 1. Single-line minified JS (> 128 KB) skips syntax highlighting.
	minJS := filepath.Join(dir, "bundle.min.js")
	if err := os.WriteFile(minJS, bytes.Repeat([]byte("var a=1;"), 20_000), 0o644); err != nil {
		t.Fatal(err)
	}
	docMin, err := loadDocument(minJS, 20)
	if err != nil {
		t.Fatal(err)
	}
	if docMin.Language == nil || docMin.Language.Name != "JavaScript" {
		t.Fatalf("language = %v, want JavaScript", docMin.Language)
	}
	if docMin.Buffer.Language() != nil {
		t.Fatal("minified single-line JS still enabled syntax highlighting")
	}

	// 2. Oversized read-only Go file (< 500k lines, > 1 MB limit) skips highlighting.
	bigGo := filepath.Join(dir, "generated.go")
	if err := os.WriteFile(bigGo, bytes.Repeat([]byte("var x = 1234567890\n"), 70_000), 0o644); err != nil {
		t.Fatal(err)
	}
	docRO, err := loadDocument(bigGo, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !docRO.ReadOnly || docRO.Buffer.Language() != nil {
		t.Fatalf("readOnly=%v bufferLang=%v, want read-only without highlight",
			docRO.ReadOnly, docRO.Buffer.Language())
	}
	if got, want := languageLabel(docRO), locales.LanguageNoHighlight("Go"); got != want {
		t.Fatalf("languageLabel = %q, want %q", got, want)
	}
}

func TestDraftLoopIdleAndWakeTransitions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	writeFile(t, path, "initial\n")
	a := keepApp(t, dir)
	doc, err := loadDocument(path, 20)
	if err != nil {
		t.Fatal(err)
	}
	a.addDocument(doc)

	// Clean document needs no background ticker.
	if a.hasPendingDraftWork() {
		t.Fatal("clean document reported pending draft work")
	}

	// Editing makes draft work pending until backed up.
	doc.Buffer.Set("unsaved edit")
	if !a.hasPendingDraftWork() {
		t.Fatal("edited document did not report pending draft work")
	}

	// Simulate a running loop reaching idle once the copy is backed up.
	a.draftStop = make(chan struct{})
	a.flushDrafts()
	a.draftTick()
	if a.hasPendingDraftWork() || a.draftStop != nil {
		t.Fatalf("draft loop stayed active after backup: pending=%v stop=%v",
			a.hasPendingDraftWork(), a.draftStop)
	}

	// Undoing back to saved text while a draft exists on disk needs one cleanup pass, then sleeps.
	doc.Buffer.Undo()
	if !a.hasPendingDraftWork() {
		t.Fatal("clean document with leftover draft on disk did not ask for cleanup")
	}
	a.draftStop = make(chan struct{})
	a.draftTick()
	if a.hasPendingDraftWork() || a.draftStop != nil || len(a.drafts.Entries()) != 0 {
		t.Fatalf("cleanup tick left pending=%v stop=%v entries=%d",
			a.hasPendingDraftWork(), a.draftStop, len(a.drafts.Entries()))
	}
}

func TestThemePreviewAndTreeDepthZeroAlloc(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	root := filepath.Join(string(filepath.Separator), "workspace", "proj")
	a.tree = files.New(root, false)
	if got := a.treeDepth(filepath.Join(root, "main.go")); got != 0 {
		t.Fatalf("root file depth = %d, want 0", got)
	}
	if got := a.treeDepth(filepath.Join(root, "a", "b", "c.go")); got != 2 {
		t.Fatalf("nested file depth = %d, want 2", got)
	}
	if got := a.treeDepth(root + "2" + string(filepath.Separator) + "x.go"); got != 0 {
		t.Fatalf("sibling prefix depth = %d, want 0", got)
	}

	nested := filepath.Join(root, "internal", "app", "view.go")
	doc := docAt(filepath.Join(root, "README.md"), "# Hello")
	th := a.themeDefinition().UI
	_ = a.markdownOptions(doc, th) // warm up once

	allocs := testing.AllocsPerRun(200, func() {
		_ = theme.FileTree(th)
		_ = theme.ResultList(th)
		_ = a.treeDepth(nested)
		_ = a.markdownOptions(doc, th)
	})
	if allocs > 0 {
		t.Fatalf("frame helpers allocated %v per frame, want 0", allocs)
	}
}

func TestIndexFoldASCIIEdgeCases(t *testing.T) {
	cases := []struct {
		text, query string
		want        []string
	}{
		{"bAca", "a", []string{"A", "a"}},
		{"x_y_Z", "_", []string{"_", "_"}},
		{"_Needle and _NEEDLE", "_needle", []string{"_Needle", "_NEEDLE"}},
		{"n_N_nEeDlE_N_needle", "needle", []string{"nEeDlE", "needle"}},
	}
	for _, tc := range cases {
		if got := matchStrings(t, tc.text, tc.query, searchOptions{}, 10); len(got) != len(tc.want) {
			t.Fatalf("scanText(%q, %q) = %q, want %q", tc.text, tc.query, got, tc.want)
		}
	}
}

func BenchmarkMemStatusBarSelectAll10MB(b *testing.B) {
	line := bytes.Repeat([]byte("abcdefghij0123456789中文测试\n"), 4) // ~116 bytes
	text := string(bytes.Repeat(line, 90_000))                    // ~10.4 MB
	a := New(settings.Default(), b.TempDir())
	doc := &Document{
		ID:       "/tmp/big.txt",
		Path:     "/tmp/big.txt",
		Buffer:   editor.NewBuffer(text),
		Encoding: encoding.UTF8,
		Size:     int64(len(text)),
	}
	a.addDocument(doc)
	tab := a.activeTab()
	tab.state.Select(0, doc.Buffer.Len())

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cursorLabel(tab)
		_ = selectionLabel(tab)
	}
}

func BenchmarkMemSave10MBUTF8(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "save.txt")
	line := bytes.Repeat([]byte("hello world 1234567890 abcdefgh\n"), 2) // 64 B
	raw := bytes.Repeat(line, 163_840)                                   // 10 MB
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		b.Fatal(err)
	}
	doc, err := loadDocument(path, 20)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := writeDocument(doc, path); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemSave10MBUTF8BOM(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "save-bom.txt")
	line := bytes.Repeat([]byte("hello world 1234567890 abcdefgh\n"), 2)
	raw := append([]byte{0xEF, 0xBB, 0xBF}, bytes.Repeat(line, 163_840)...)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		b.Fatal(err)
	}
	doc, err := loadDocument(path, 20)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := writeDocument(doc, path); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemLoad10MBUTF8(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "load.txt")
	line := bytes.Repeat([]byte("hello world 1234567890 abcdefgh\n"), 2)
	raw := bytes.Repeat(line, 163_840) // 10 MB
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		doc, err := loadDocument(path, 20)
		if err != nil {
			b.Fatal(err)
		}
		_ = doc
	}
}

func BenchmarkMemLoad2MBGB18030(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "gb.txt")
	text := string(bytes.Repeat([]byte("中文编码测试第一行第二行第三行\n"), 50_000))
	raw, err := encoding.Encode(text, encoding.GB18030)
	if err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		doc, err := loadDocument(path, 20)
		if err != nil {
			b.Fatal(err)
		}
		_ = doc
	}
}

func BenchmarkMemReject10MBBinary(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "blob.bin")
	raw := make([]byte, 10<<20)
	copy(raw, []byte{0x89, 'P', 'N', 'G', 0x00, 0x01, 0x02})
	for i := 8; i < len(raw); i++ {
		raw[i] = 'a'
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := loadDocument(path, 20); err == nil {
			b.Fatal("expected binary error")
		}
	}
}

func BenchmarkMemReloadExternal10MB(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "reload.go")
	line := []byte("package main // 0123456789abcdef0123456789abcdef\n")
	raw := bytes.Repeat(line, 210_000) // ~10.3 MB
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		b.Fatal(err)
	}
	doc, err := loadDocument(path, 20)
	if err != nil {
		b.Fatal(err)
	}
	a := New(settings.Default(), dir)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		a.reloadExternal(doc)
	}
}

func BenchmarkMemSameDraftText10MB(b *testing.B) {
	line := bytes.Repeat([]byte("hello world 1234567890 abcdefgh\r\n"), 2)
	raw := string(bytes.Repeat(line, 160_000)) // ~10.5 MB CRLF
	doc := &Document{Buffer: editor.NewBuffer(raw)}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !sameDraftText(doc, raw) {
			b.Fatal("expected equal")
		}
	}
}

func BenchmarkMemLocalPreviewImage3200x2400(b *testing.B) {
	dir := b.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 3200, 2400))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "big.png"), buf.Bytes(), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		src, err := loadPreviewImage(dir, "big.png")
		if err != nil {
			b.Fatal(err)
		}
		_ = src
	}
}

func BenchmarkMemSearch10MBASCIIFold(b *testing.B) {
	line := "2026/11/25 19:00:00 [INFO] request processed successfully status=200 latency=3ms\n"
	text := strings.Repeat(line, 120_000) + "TargetMarkerFoundHere\n" // ~9.7 MB
	doc := docAt("/tmp/log.txt", text)
	inputs := []searchInput{{doc: doc, snap: doc.Buffer.Snapshot()}}
	cancel := make(chan struct{})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rows, _, err, _ := scanDocuments(inputs, "targetmarkerfoundhere", searchOptions{}, cancel)
		if err != nil || len(rows) != 2 {
			b.Fatalf("rows=%d err=%v", len(rows), err)
		}
	}
}

func BenchmarkMemSearch20DocsUnicode(b *testing.B) {
	text := strings.Repeat("普通文本内容第一行第二行第三行\n", 500) + "关键目标词\n"
	inputs := make([]searchInput, 20)
	for i := range inputs {
		d := docAt("/tmp/doc.txt", text)
		inputs[i] = searchInput{doc: d, snap: d.Buffer.Snapshot()}
	}
	cancel := make(chan struct{})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rows, _, err, _ := scanDocuments(inputs, "关键目标词", searchOptions{}, cancel)
		if err != nil || len(rows) != 40 {
			b.Fatalf("rows=%d err=%v", len(rows), err)
		}
	}
}

func BenchmarkMemThemeAndTreeDepth(b *testing.B) {
	a := New(settings.Default(), b.TempDir())
	a.tree = files.New("/Users/xiaoz/MyProject/zpad", false)
	t := a.themeDefinition().UI
	paths := []string{
		"/Users/xiaoz/MyProject/zpad/main.go",
		"/Users/xiaoz/MyProject/zpad/internal/app/view.go",
		"/Users/xiaoz/MyProject/zpad/internal/encoding/encoding.go",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = theme.FileTree(t)
		_ = theme.ResultList(t)
		for _, p := range paths {
			_ = a.treeDepth(p)
		}
	}
}
