package app

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
)

const (
	// previewImageMaxBytes caps what one remote image may weigh as it comes,
	// before it is decoded: a Markdown file is other people's text, and a
	// link to a 200 MB picture should not fill the heap. 10 MB is well over
	// what a photo for reading needs, and small enough that a page of
	// pictures cannot pile up.
	previewImageMaxBytes = 10 << 20
	// previewImageMaxPixels caps the pixels decoded before the image is
	// scaled down. Decoding averages the whole picture, so a 48 MP photo
	// would peak at about 190 MB for a moment.
	previewImageMaxPixels = 24 << 20
	// previewImageMaxSide is how large a remote image is kept, in pixels:
	// the preview is 820 DIP wide, and this keeps an image sharp on a 2×
	// display without holding more than the screen shows.
	previewImageMaxSide = 1600
	// previewImageSlots bounds the images downloading and decoding at once,
	// so a document of 30 images neither opens 30 connections at once nor
	// decodes them all at the same moment.
	previewImageSlots = 4
	// previewImageTimeout covers a whole request, body included, so a
	// server that stops mid-picture cannot hold a slot forever.
	previewImageTimeout = 15 * time.Second
	// previewImageCacheBytes bounds the decoded images kept after the
	// preview closes, which makes opening it again quick.
	previewImageCacheBytes = 32 << 20
)

// previewImageClient downloads remote images. Its default transport keeps
// the environment's proxy, and the timeout is the whole request.
var previewImageClient = &http.Client{Timeout: previewImageTimeout}

// previewImagesRunning is the semaphore of previewImageSlots: send to take a
// slot, receive to give it back.
var previewImagesRunning = make(chan struct{}, previewImageSlots)

// cachedPreviewImage is one image of previewImageCache, with the bytes its
// pixels take.
type cachedPreviewImage struct {
	src  ui.ImageSource
	size int
}

// previewImageCache keeps decoded remote images by address, so toggling the
// preview off and on again shows them at once. A document's options hold
// their images while a preview shows; this bounds what stays once it closes.
type previewImageCache struct {
	mu      sync.Mutex
	bytes   int
	urls    []string // oldest first
	entries map[string]cachedPreviewImage
}

var previewImages previewImageCache

// get returns the cached image at url, marking it the newest.
func (c *previewImageCache) get(url string) (ui.ImageSource, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[url]
	if !ok {
		return nil, false
	}
	for i, u := range c.urls {
		if u == url {
			c.urls = append(c.urls[:i], c.urls[i+1:]...)
			c.urls = append(c.urls, url)
			break
		}
	}
	return e.src, true
}

// put caches src at url, dropping the oldest images to stay within the
// budget. It takes images of the app's own goroutines, so it locks.
func (c *previewImageCache) put(url string, src ui.ImageSource) {
	w, h := ui.ImageSize(src)
	size := int(w*h) * 4
	if size <= 0 {
		size = 4
	}
	if size > previewImageCacheBytes {
		return // it alone fills the cache: keep it nowhere
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = map[string]cachedPreviewImage{}
	}
	if _, ok := c.entries[url]; ok {
		return
	}
	for c.bytes+size > previewImageCacheBytes && len(c.urls) > 0 {
		oldest := c.urls[0]
		c.urls = slices.Delete(c.urls, 0, 1)
		if e, ok := c.entries[oldest]; ok {
			c.bytes -= e.size
			delete(c.entries, oldest)
		}
	}
	c.urls = append(c.urls, url)
	c.entries[url] = cachedPreviewImage{src: src, size: size}
	c.bytes += size
}

// clear drops every cached remote image, as when the last Markdown document
// closes.
func (c *previewImageCache) clear() {
	c.mu.Lock()
	c.bytes = 0
	c.urls = nil
	c.entries = nil
	c.mu.Unlock()
}

// loadPreviewImage loads one image of a Markdown preview: an address on the
// web, an inline data: URI, or a file beside the document in dir.
func loadPreviewImage(dir, raw string) (ui.ImageSource, error) {
	switch {
	case strings.HasPrefix(raw, "data:"):
		return loadInlineImage(raw)
	case strings.HasPrefix(raw, "//"):
		// A protocol-relative address, as some Markdown writes.
		return loadRemoteImage("https:" + raw)
	case strings.HasPrefix(raw, "http://"), strings.HasPrefix(raw, "https://"):
		return loadRemoteImage(raw)
	}
	return loadLocalPreviewImage(dir, raw)
}

// loadLocalPreviewImage reads an image file relative to dir, enforcing the
// same byte, pixel and display-side bounds as remote images so a local
// high-resolution photo does not hold tens of megabytes in memory.
func loadLocalPreviewImage(dir, raw string) (ui.ImageSource, error) {
	if strings.Contains(raw, "://") || strings.HasPrefix(raw, "data:") {
		return nil, os.ErrNotExist
	}
	path := strings.TrimPrefix(raw, "file:")
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, filepath.FromSlash(path))
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, os.ErrInvalid
	}
	if info.Size() > previewImageMaxBytes {
		return nil, fmt.Errorf("image %q: over the limit of %d bytes", raw, previewImageMaxBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(filepath.Ext(path), ".svg") {
		return ui.ParseSVG(data)
	}
	return decodePreviewImage(data)
}

// loadRemoteImage downloads and decodes the image at raw. Markdown's view
// calls it on a goroutine of its own, so it may wait for the network; what
// it returns is what the view shows on its next build.
func loadRemoteImage(raw string) (ui.ImageSource, error) {
	if src, ok := previewImages.get(raw); ok {
		return src, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("image %q: unsupported address", raw)
	}
	previewImagesRunning <- struct{}{}
	defer func() { <-previewImagesRunning }()
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ZPad/"+Version)
	req.Header.Set("Accept", "image/*")
	resp, err := previewImageClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image %q: %s", raw, resp.Status)
	}
	if resp.ContentLength > previewImageMaxBytes {
		return nil, fmt.Errorf("image %q: over the limit of %d bytes", raw, previewImageMaxBytes)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, previewImageMaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > previewImageMaxBytes {
		return nil, fmt.Errorf("image %q: over the limit of %d bytes", raw, previewImageMaxBytes)
	}
	src, err := decodePreviewImage(data)
	if err != nil {
		return nil, err
	}
	previewImages.put(raw, src)
	return src, nil
}

// loadInlineImage decodes a data: URI, which Markdown embeds small images
// with. Only the base64 form is taken; the plain one carries text.
func loadInlineImage(raw string) (ui.ImageSource, error) {
	comma := strings.IndexByte(raw, ',')
	if comma < 0 {
		return nil, errors.New("data URI without a payload")
	}
	header, payload := raw[len("data:"):comma], raw[comma+1:]
	if !strings.HasSuffix(header, ";base64") {
		return nil, errors.New("data URI is not base64")
	}
	// Base64 is 4/3 of the bytes it carries: refuse before decoding what is
	// too large.
	if int64(len(payload)) > int64(previewImageMaxBytes)*4/3+4 {
		return nil, fmt.Errorf("data URI over the limit of %d bytes", previewImageMaxBytes)
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, err
	}
	return decodePreviewImage(data)
}

// decodePreviewImage decodes image data as SVG from its markup, or as a
// bitmap scaled down to previewImageMaxSide. A picture with more pixels than
// previewImageMaxPixels is refused rather than decoded and then scaled: the
// scaling comes after the decoding, and that is when the memory goes.
func decodePreviewImage(data []byte) (ui.ImageSource, error) {
	if isSVGImage(data) {
		return ui.ParseSVG(data)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if int64(cfg.Width)*int64(cfg.Height) > previewImageMaxPixels {
		return nil, fmt.Errorf("image of %d×%d pixels is over the limit", cfg.Width, cfg.Height)
	}
	return ui.DecodeBitmapScaled(data, previewImageMaxSide)
}

// isSVGImage reports whether data is SVG markup, which image.DecodeConfig
// does not know. The root element comes at the start, after an XML
// declaration or comments at most.
func isSVGImage(data []byte) bool {
	head := data
	if len(head) > 1024 {
		head = head[:1024]
	}
	return bytes.Contains(head, []byte("<svg"))
}
