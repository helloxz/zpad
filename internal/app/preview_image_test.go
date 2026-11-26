package app

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"weak"

	"github.com/egoist/mygo/ui"
	"github.com/helloxz/zpad/internal/settings"
)

// testPNG returns a width × height PNG.
func testPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// testPNGHeader returns a PNG that says it is width × height, with no
// pixels: enough for image.DecodeConfig, and small enough to keep a test of
// the pixel limit from decoding one.
func testPNGHeader(width, height uint32) []byte {
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], width)
	binary.BigEndian.PutUint32(ihdr[4:], height)
	ihdr[8], ihdr[9] = 8, 2 // bit depth 8, truecolor

	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	binary.Write(&b, binary.BigEndian, uint32(len(ihdr)))
	chunk := append([]byte("IHDR"), ihdr...)
	b.Write(chunk)
	binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return b.Bytes()
}

func TestLoadPreviewImageFromTheWeb(t *testing.T) {
	var hits int32
	data := testPNG(t, 5, 3)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if agent := r.Header.Get("User-Agent"); !strings.HasPrefix(agent, "ZPad/") {
			t.Errorf("user agent %q, want ZPad's", agent)
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(data)
	}))
	defer srv.Close()

	dir := t.TempDir()
	src, err := loadPreviewImage(dir, srv.URL+"/shot.png")
	if err != nil {
		t.Fatalf("load image: %v", err)
	}
	bm, ok := src.(*ui.Bitmap)
	if !ok {
		t.Fatalf("loaded %T, want a bitmap", src)
	}
	if w, h := bm.Size(); w != 5 || h != 3 {
		t.Fatalf("bitmap %d×%d, want 5×3", w, h)
	}

	// A second look comes from the cache, without another request.
	if _, err := loadPreviewImage(dir, srv.URL+"/shot.png"); err != nil {
		t.Fatalf("load cached image: %v", err)
	}
	if n := atomic.LoadInt32(&hits); n != 1 {
		t.Fatalf("%d requests, want one", n)
	}
}

func TestLoadPreviewImageSVG(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="7" height="4"></svg>`))
	}))
	defer srv.Close()

	src, err := loadPreviewImage(t.TempDir(), srv.URL+"/logo.svg")
	if err != nil {
		t.Fatalf("load SVG: %v", err)
	}
	if _, ok := src.(*ui.SVG); !ok {
		t.Fatalf("loaded %T, want an SVG", src)
	}
	if w, h := ui.ImageSize(src); w != 7 || h != 4 {
		t.Fatalf("SVG %v×%v, want 7×4", w, h)
	}
}

func TestLoadPreviewImageInline(t *testing.T) {
	data := testPNG(t, 2, 2)
	raw := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
	src, err := loadPreviewImage(t.TempDir(), raw)
	if err != nil {
		t.Fatalf("load inline image: %v", err)
	}
	bm, ok := src.(*ui.Bitmap)
	if !ok {
		t.Fatalf("loaded %T, want a bitmap", src)
	}
	if w, h := bm.Size(); w != 2 || h != 2 {
		t.Fatalf("bitmap %d×%d, want 2×2", w, h)
	}

	for _, bad := range []string{
		"data:image/png;base64",
		"data:image/png,notbase64",
		"data:image/png;base64,!!!!",
	} {
		if _, err := loadPreviewImage(t.TempDir(), bad); err == nil {
			t.Errorf("data URI %q loaded, want an error", bad)
		}
	}
}

func TestLoadPreviewImageFailures(t *testing.T) {
	var bigBody bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/missing.png":
			http.NotFound(w, r)
		case "/page.html":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<html>not a picture</html>"))
		case "/declared.png":
			w.Header().Set("Content-Length", "30000000")
			w.WriteHeader(http.StatusOK)
		case "/streamed.png":
			bigBody = true
			chunk := make([]byte, 1<<20)
			for i := 0; i < 11; i++ {
				if _, err := w.Write(chunk); err != nil {
					return
				}
			}
		default:
			w.Write(testPNG(t, 1, 1))
		}
	}))
	defer srv.Close()

	for _, path := range []string{"/missing.png", "/page.html", "/declared.png", "/streamed.png"} {
		if _, err := loadPreviewImage(t.TempDir(), srv.URL+path); err == nil {
			t.Errorf("%s loaded, want an error", path)
		}
	}
	if !bigBody {
		t.Fatal("the streamed image never reached the server")
	}
}

func TestLoadPreviewImageRefusesTooManyPixels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(testPNGHeader(7000, 5000))
	}))
	defer srv.Close()

	if _, err := loadPreviewImage(t.TempDir(), srv.URL+"/huge.png"); err == nil {
		t.Fatal("an image over the pixel limit loaded")
	}
}

func TestDropPreviewOptions(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	doc := docAt("/tmp/README.md", "![x](a.png)")
	a.addDocument(doc)
	a.ToggleMarkdownPreview()
	a.markdownOptions(doc, ui.LightTheme()) // what the view builds while it shows
	if doc.md == nil {
		t.Fatal("the preview made no options")
	}

	// A split shows the same document in both panes: closing the preview in
	// one keeps what the other still shows.
	a.toggleSplit()
	for _, tab := range a.left.tabs {
		tab.preview = false
	}
	for _, tab := range a.right.tabs {
		tab.preview = false
	}
	a.right.tabs[0].preview = true
	a.dropPreviewOptions(doc)
	if doc.md == nil {
		t.Fatal("options dropped while the other pane previews")
	}

	a.right.tabs[0].preview = false
	a.dropPreviewOptions(doc)
	if doc.md != nil {
		t.Fatal("options kept with no preview showing")
	}
}

func TestToggleMarkdownPreviewDropsOptions(t *testing.T) {
	a := New(settings.Default(), t.TempDir())
	doc := docAt("/tmp/README.md", "![x](a.png)")
	a.addDocument(doc)
	a.ToggleMarkdownPreview()
	a.markdownOptions(doc, ui.LightTheme()) // what the view builds while it shows
	if doc.md == nil {
		t.Fatal("the preview made no options")
	}
	a.ToggleMarkdownPreview()
	if doc.md != nil {
		t.Fatal("returning to the source kept the preview's images")
	}
}

// TestMarkdownPreviewAsksForRemoteImages checks the whole way from the view
// to the loader: a rendered preview requests the images its Markdown names.
func TestMarkdownPreviewAsksForRemoteImages(t *testing.T) {
	var hits int32
	data := testPNG(t, 4, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "image/png")
		w.Write(data)
	}))
	defer srv.Close()

	dir := t.TempDir()
	doc := docAt(filepath.Join(dir, "README.md"), "![shot]("+srv.URL+"/shot.png)")
	a := New(settings.Default(), dir)
	a.addDocument(doc)
	tt := ui.NewTester(a.View, 900, 700)
	tt.Frame()
	a.ToggleMarkdownPreview()
	tt.Frame()

	deadline := time.Now().Add(3 * time.Second)
	for atomic.LoadInt32(&hits) == 0 && time.Now().Before(deadline) {
		tt.Frame()
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt32(&hits) == 0 {
		t.Fatal("the preview never asked for the image")
	}
}

func TestPreviewImageDecodeConcurrencyBounded(t *testing.T) {
	data := testPNG(t, 8, 8)
	prev := scalePreviewBitmap
	defer func() { scalePreviewBitmap = prev }()

	var active, peak atomic.Int32
	entered := make(chan struct{}, 8)
	release := make(chan struct{})
	scalePreviewBitmap = func(raw []byte, maxSide int) (*ui.Bitmap, error) {
		cur := active.Add(1)
		for {
			old := peak.Load()
			if cur <= old || peak.CompareAndSwap(old, cur) {
				break
			}
		}
		entered <- struct{}{}
		<-release
		active.Add(-1)
		return prev(raw, maxSide)
	}

	const total = 6
	var wg sync.WaitGroup
	errs := make(chan error, total)
	for range total {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := decodePreviewImage(data)
			errs <- err
		}()
	}

	for range previewImageDecodeSlots {
		<-entered
	}
	// Give remaining goroutines a chance to contend on previewImagesDecoding.
	time.Sleep(25 * time.Millisecond)
	if got := active.Load(); got != previewImageDecodeSlots {
		close(release)
		wg.Wait()
		t.Fatalf("active decoders = %d, want %d", got, previewImageDecodeSlots)
	}

	close(release)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("decodePreviewImage failed: %v", err)
		}
	}
	if got := peak.Load(); got != previewImageDecodeSlots {
		t.Fatalf("peak concurrent decoders = %d, want %d", got, previewImageDecodeSlots)
	}
}

func waitLargeDocGCIdle(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && (largeDocGCRunning.Load() || largeDocGCRequested.Load()) {
		time.Sleep(15 * time.Millisecond)
	}
	if largeDocGCRunning.Load() || largeDocGCRequested.Load() {
		t.Fatal("timed out waiting for background GC to settle")
	}
}

func TestPreviewImageDropSchedulesOSMemoryReclamation(t *testing.T) {
	previewImages.clear()
	waitLargeDocGCIdle(t)

	dir := t.TempDir()
	shotPath := filepath.Join(dir, "shot.png")
	if err := os.WriteFile(shotPath, testPNG(t, 16, 12), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. Plain-text Markdown preview does not schedule FreeOSMemory on exit.
	a := New(settings.Default(), dir)
	plain := docAt(filepath.Join(dir, "plain.md"), "# Heading\n\nJust text.")
	a.addDocument(plain)
	tt := ui.NewTester(a.View, 900, 700)
	tt.Frame()
	a.ToggleMarkdownPreview()
	tt.Frame()
	a.ToggleMarkdownPreview()
	tt.Frame()
	if largeDocGCRunning.Load() || largeDocGCRequested.Load() {
		t.Fatal("leaving plain-text Markdown preview scheduled FreeOSMemory")
	}

	// 2. Leaving a preview that loaded an image schedules FreeOSMemory.
	withImg := docAt(filepath.Join(dir, "shots.md"), "![shot](shot.png)")
	a.addDocument(withImg)
	a.ToggleMarkdownPreview()
	tt.Frame()
	deadline := time.Now().Add(2 * time.Second)
	for !withImg.mdLoadedImage.Load() && time.Now().Before(deadline) {
		tt.Frame()
		time.Sleep(10 * time.Millisecond)
	}
	if !withImg.mdLoadedImage.Load() {
		t.Fatal("preview did not load local image")
	}
	a.ToggleMarkdownPreview()
	if !largeDocGCRunning.Load() && !largeDocGCRequested.Load() {
		t.Fatal("leaving image-bearing preview did not schedule FreeOSMemory")
	}
	waitLargeDocGCIdle(t)
}

func TestPreviewImageTempDiskCacheReleasesBitmapsAndCleansUp(t *testing.T) {
	previewImages.clear()
	defer previewImages.clear()
	waitLargeDocGCIdle(t)

	pngBytes := testPNG(t, 2400, 1600)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()

	var lastBM atomic.Pointer[weak.Pointer[ui.Bitmap]]
	prevScale := scalePreviewBitmap
	defer func() { scalePreviewBitmap = prevScale }()
	scalePreviewBitmap = func(data []byte, maxSide int) (*ui.Bitmap, error) {
		bm, err := prevScale(data, maxSide)
		if err == nil && bm != nil {
			w := weak.Make(bm)
			lastBM.Store(&w)
		}
		return bm, err
	}

	dir := t.TempDir()
	imgURL := srv.URL + "/shot.png"
	doc := docAt(filepath.Join(dir, "post.md"), "![shot]("+imgURL+")")
	a := New(settings.Default(), dir)
	a.addDocument(doc)
	tt := ui.NewTester(a.View, 1000, 700)
	tt.Frame()

	// 1. First preview downloads, scales to 1512×1008, and caches raw bytes on disk.
	a.ToggleMarkdownPreview()
	tt.Frame()
	deadline := time.Now().Add(3 * time.Second)
	for (!doc.mdLoadedImage.Load() || previewImages.dirPath() == "") && time.Now().Before(deadline) {
		tt.Frame()
		time.Sleep(10 * time.Millisecond)
	}
	if !doc.mdLoadedImage.Load() || hits.Load() != 1 {
		t.Fatalf("loaded=%v hits=%d, want true and 1", doc.mdLoadedImage.Load(), hits.Load())
	}
	cacheDir := previewImages.dirPath()
	if _, err := os.Stat(cacheDir); err != nil {
		t.Fatalf("temp cache dir %q missing: %v", cacheDir, err)
	}
	wbm := lastBM.Load()
	if wbm == nil || wbm.Value() == nil {
		t.Fatal("expected decoded bitmap in preview")
	}
	if w, h := wbm.Value().Size(); w != 1512 || h != 1008 {
		t.Fatalf("scaled remote bitmap = %d×%d, want 1512×1008", w, h)
	}

	// 2. Returning to edit mode drops the decoded *ui.Bitmap from memory even
	// while the Markdown tab stays open, keeping only the temp file on disk.
	a.ToggleMarkdownPreview()
	tt.Frame()
	tt.Frame()
	waitLargeDocGCIdle(t)
	for range 4 {
		runtime.GC()
	}
	if wbm.Value() != nil {
		t.Fatal("decoded *ui.Bitmap stayed alive in memory after leaving preview")
	}
	if _, err := os.Stat(cacheDir); err != nil {
		t.Fatalf("temp cache dir %q removed while Markdown tab remained open: %v", cacheDir, err)
	}

	// 3. Re-entering preview decodes from the temp file without another HTTP request.
	a.ToggleMarkdownPreview()
	tt.Frame()
	deadline = time.Now().Add(3 * time.Second)
	for !doc.mdLoadedImage.Load() && time.Now().Before(deadline) {
		tt.Frame()
		time.Sleep(10 * time.Millisecond)
	}
	if !doc.mdLoadedImage.Load() || hits.Load() != 1 {
		t.Fatalf("re-preview loaded=%v hits=%d, want true and 1", doc.mdLoadedImage.Load(), hits.Load())
	}
	a.ToggleMarkdownPreview()
	tt.Frame()
	waitLargeDocGCIdle(t)

	// 4. If the temp file vanishes externally, loadPreviewImage falls back cleanly.
	_ = os.RemoveAll(cacheDir)
	if _, err := loadPreviewImage(dir, imgURL); err != nil {
		t.Fatalf("fallback reload after external temp removal failed: %v", err)
	}
	if hits.Load() != 2 {
		t.Fatalf("hits after external temp removal = %d, want 2", hits.Load())
	}
	cacheDir2 := previewImages.dirPath()

	// 5. Closing the last Markdown tab removes the temp directory on disk.
	a.removeTab(0, 0)
	if _, err := os.Stat(cacheDir2); !os.IsNotExist(err) {
		t.Fatalf("temp cache dir %q still exists after closing last Markdown tab: err=%v", cacheDir2, err)
	}
}
