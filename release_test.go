package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/helloxz/zpad/internal/app"
)

func TestFileAssociations(t *testing.T) {
	data, err := os.ReadFile("mygo.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		FileAssociations []struct {
			Ext      []string `json:"ext"`
			Name     string   `json:"name"`
			MIMEType string   `json:"mimeType"`
		} `json:"fileAssociations"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.FileAssociations) == 0 {
		t.Fatal("no file associations")
	}
	found := map[string]bool{}
	for _, a := range cfg.FileAssociations {
		if a.Name == "" || a.MIMEType == "" {
			t.Fatalf("file association %v has no name or MIME type", a.Ext)
		}
		for _, ext := range a.Ext {
			if found[ext] {
				// Two entries would write one ProgID twice, the later
				// name winning.
				t.Fatalf("file association %q listed twice", ext)
			}
			found[ext] = true
		}
	}
	for _, ext := range []string{
		// Documents.
		"txt", "md", "markdown", "html", "htm", "css", "json", "xml", "yaml", "toml", "log", "csv",
		// Source code.
		"js", "ts", "tsx", "go", "py", "rs", "c", "cpp", "h", "java", "kt", "swift", "cs",
		"rb", "php", "lua", "sh", "sql", "vue", "svelte", "scss", "less", "diff",
	} {
		if !found[ext] {
			t.Fatalf("file association %q missing", ext)
		}
	}
}

// The open dialog offers the text files the app is associated with: a file
// opened from the file manager must be one the dialog can offer too.
func TestTextExtensionsMatchAssociations(t *testing.T) {
	data, err := os.ReadFile("mygo.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		FileAssociations []struct {
			Ext []string `json:"ext"`
		} `json:"fileAssociations"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	associated := map[string]bool{}
	for _, a := range cfg.FileAssociations {
		for _, ext := range a.Ext {
			associated[ext] = true
		}
	}
	offered := app.TextExtensions()
	if len(offered) != len(associated) {
		t.Fatalf("the open dialog offers %d extensions, mygo.json associates %d", len(offered), len(associated))
	}
	for _, ext := range offered {
		if !associated[ext] {
			t.Fatalf("the open dialog offers %q, which mygo.json does not associate", ext)
		}
	}
}

// mygo.json is the app's only version: mygo build links it into the binary,
// and names the artifacts and the tag with it.
func TestVersionIsAReleaseVersion(t *testing.T) {
	data, err := os.ReadFile("mygo.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if !releaseVersion.MatchString(cfg.Version) {
		t.Fatalf("mygo.json says version %q, want MAJOR.MINOR.PATCH", cfg.Version)
	}
}

// releaseVersion is what the tag of a release names, as v1.2.3.
var releaseVersion = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// Installed apps accept only updates signed with the private half of the
// key here, and find their manifests in the latest release of updates.github:
// both are compiled into the app by mygo build.
func TestUpdatesAreConfigured(t *testing.T) {
	data, err := os.ReadFile("mygo.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Updates *struct {
			PublicKey string `json:"publicKey"`
			GitHub    string `json:"github"`
		} `json:"updates"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Updates == nil {
		t.Fatal("mygo.json has no updates: the app cannot update itself")
	}
	key, err := base64.StdEncoding.DecodeString(cfg.Updates.PublicKey)
	if err != nil {
		t.Fatalf("updates.publicKey is not base64: %v", err)
	}
	if len(key) != ed25519.PublicKeySize {
		t.Fatalf("updates.publicKey is %d bytes, want an Ed25519 key of %d", len(key), ed25519.PublicKeySize)
	}
	if cfg.Updates.GitHub != "helloxz/zpad" {
		t.Fatalf("updates.github is %q, want helloxz/zpad", cfg.Updates.GitHub)
	}
}

// The update window shows the version's section of CHANGELOG.md, which mygo
// build writes into the manifests: a version without one fails the build.
func TestChangelogHasTheReleaseNotes(t *testing.T) {
	data, err := os.ReadFile("mygo.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	notes, err := os.ReadFile("CHANGELOG.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(notes), "## "+cfg.Version) {
		t.Fatalf("CHANGELOG.md has no section for %s", cfg.Version)
	}
}

// The site's version comes from the latest GitHub release, which
// layouts/partials/release.html fetches while the site builds; the committed
// snapshot in website/data/release.json is the fallback and the local preview.
// A version typed into the copy goes stale on the next release: the hero badge
// still said v0.3.2 after 0.4.0 shipped. Templates build the badge from the
// release data, and nothing under website/ names a version by hand.
func TestWebsiteHasNoHardcodedVersion(t *testing.T) {
	// The icon SVGs are full of numbers like 1.02.08, so a template only counts
	// as naming a version when it carries the v prefix the site uses.
	templates := regexp.MustCompile(`\bv\d+\.\d+\.\d+`)
	copyText := regexp.MustCompile(`\d+\.\d+\.\d+`)

	var files []string
	for _, glob := range []string{"website/content/*.md", "website/i18n/*.toml"} {
		matched, err := filepath.Glob(glob)
		if err != nil {
			t.Fatal(err)
		}
		if len(matched) == 0 {
			t.Fatalf("%s matches nothing: is the site still there?", glob)
		}
		files = append(files, matched...)
	}
	err := filepath.WalkDir("website/layouts", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".html") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		pattern := copyText
		if strings.HasSuffix(path, ".html") {
			pattern = templates
		}
		for i, line := range strings.Split(string(data), "\n") {
			if m := pattern.FindString(line); m != "" {
				t.Errorf("%s:%d names the version %q; derive it from the release data", path, i+1, m)
			}
		}
	}
}

// The snapshot is what the build falls back to when GitHub cannot be reached,
// and what `hugo server` renders. Keep it a usable release object.
func TestWebsiteReleaseSnapshot(t *testing.T) {
	data, err := os.ReadFile("website/data/release.json")
	if err != nil {
		t.Fatal(err)
	}
	var release struct {
		TagName     string `json:"tag_name"`
		PublishedAt string `json:"published_at"`
		Body        string `json:"body"`
		Assets      []struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(data, &release); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(release.TagName, "v") {
		t.Errorf("tag_name %q has no v prefix", release.TagName)
	}
	if _, err := time.Parse(time.RFC3339, release.PublishedAt); err != nil {
		t.Errorf("published_at %q is not RFC 3339: %v", release.PublishedAt, err)
	}
	if strings.TrimSpace(release.Body) == "" {
		t.Error("the snapshot has no release notes")
	}
	if len(release.Assets) == 0 {
		t.Fatal("the snapshot has no assets; refresh it from the release API")
	}
	const prefix = "https://github.com/helloxz/zpad/releases/download/"
	for _, asset := range release.Assets {
		if asset.Name == "" || asset.Size <= 0 || !strings.HasPrefix(asset.URL, prefix) {
			t.Errorf("asset %+v is not a release download", asset)
		}
	}
}

// The changelog page renders the ten most recent published releases from
// GitHub, and falls back to this snapshot for local previews and a failed
// fetch. It must stay a usable list, in the same order the page shows.
func TestWebsiteReleasesSnapshot(t *testing.T) {
	data, err := os.ReadFile("website/data/releases.json")
	if err != nil {
		t.Fatal(err)
	}
	var releases []struct {
		TagName     string `json:"tag_name"`
		PublishedAt string `json:"published_at"`
		HTMLURL     string `json:"html_url"`
		Prerelease  bool   `json:"prerelease"`
	}
	if err := json.Unmarshal(data, &releases); err != nil {
		t.Fatal(err)
	}
	if len(releases) == 0 {
		t.Fatal("the snapshot has no releases; refresh it from the releases API")
	}
	if len(releases) > 10 {
		t.Errorf("the snapshot has %d releases, want at most 10", len(releases))
	}

	const prefix = "https://github.com/helloxz/zpad/releases/tag/"
	seen := map[string]bool{}
	var previous time.Time
	for i, release := range releases {
		if !strings.HasPrefix(release.TagName, "v") {
			t.Errorf("tag_name %q has no v prefix", release.TagName)
		}
		if seen[release.TagName] {
			t.Errorf("release %q is listed twice", release.TagName)
		}
		seen[release.TagName] = true
		published, err := time.Parse(time.RFC3339, release.PublishedAt)
		if err != nil {
			t.Errorf("published_at %q is not RFC 3339: %v", release.PublishedAt, err)
		} else if i > 0 && previous.Before(published) {
			t.Errorf("release %q is newer than the one before it", release.TagName)
		}
		previous = published
		if release.Prerelease {
			t.Errorf("release %q is a pre-release; the page skips those", release.TagName)
		}
		if !strings.HasPrefix(release.HTMLURL, prefix) {
			t.Errorf("release %q has no GitHub release URL", release.TagName)
		}
	}

	latestData, err := os.ReadFile("website/data/release.json")
	if err != nil {
		t.Fatal(err)
	}
	var latest struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(latestData, &latest); err != nil {
		t.Fatal(err)
	}
	if releases[0].TagName != latest.TagName {
		t.Errorf("the changelog starts at %q but release.json names %q", releases[0].TagName, latest.TagName)
	}
}

func TestAppIconSize(t *testing.T) {
	f, err := os.Open("resources/icon.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() != 1024 || b.Dy() != 1024 {
		t.Fatalf("icon is %dx%d, want 1024x1024", b.Dx(), b.Dy())
	}
}
