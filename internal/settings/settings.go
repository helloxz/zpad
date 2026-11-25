// Package settings stores ZPad's preferences in the user data directory.
package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/helloxz/zpad/internal/atomicfile"
)

const fileName = "settings.json"

// Settings is everything ZPad remembers between launches.
//
// Fields are exported so the view can read them directly and tests can
// build values without a constructor. Missing fields in an older file keep
// the defaults of Default.
type Settings struct {
	Theme    string `json:"theme"`    // system | light | dark | a named theme id
	Language string `json:"language"` // "" follows the system, else zh-Hans | en

	FontFamily string  `json:"fontFamily"` // "" is the system's monospaced font
	FontSize   float32 `json:"fontSize"`
	LineHeight float32 `json:"lineHeight"`

	SoftWrap    bool   `json:"softWrap"`
	LineNumbers string `json:"lineNumbers"` // absolute | relative | off
	TabSize     int    `json:"tabSize"`
	UseTabs     bool   `json:"useTabs"`
	ShowGuides  bool   `json:"showGuides"`
	// Whitespace says where spaces show as dots and tabs as arrows:
	// selection | none | all | trailing.
	Whitespace string `json:"whitespace"`
	AutoClose  bool   `json:"autoClose"`
	// VimMode makes every editor modal: normal, insert and visual modes,
	// with Vim's motions and operators. It is off by default.
	VimMode bool `json:"vimMode"`

	DefaultEncoding string `json:"defaultEncoding"`   // utf-8 | utf-8-bom | utf-16le | utf-16be | gb18030
	DefaultEOL      string `json:"defaultLineEnding"` // "\n" | "\r\n" | "\r"
	LargeFileMB     int    `json:"largeFileMB"`

	SidebarOpen    bool    `json:"sidebarOpen"`
	SidebarWidth   float32 `json:"sidebarWidth"`
	ShowHidden     bool    `json:"showHidden"`
	RestoreSession bool    `json:"restoreSession"`
	ConfirmClose   bool    `json:"confirmClose"`
	// KeepUnsaved keeps unsaved content, including untitled documents, in
	// the user data directory and restores it on the next launch, instead
	// of asking to save it when the window closes or the app quits.
	KeepUnsaved bool `json:"keepUnsaved"`
}

// Default returns the settings of a first launch.
func Default() Settings {
	return Settings{
		Theme:    "system",
		Language: "",

		FontFamily: "",
		FontSize:   13,
		LineHeight: 1.5,

		SoftWrap:    false,
		LineNumbers: "absolute",
		TabSize:     4,
		UseTabs:     false,
		ShowGuides:  true,
		Whitespace:  "selection",
		AutoClose:   true,
		VimMode:     false,

		DefaultEncoding: "utf-8",
		DefaultEOL:      "\n",
		LargeFileMB:     20,

		SidebarOpen:    false,
		SidebarWidth:   240,
		ShowHidden:     false,
		RestoreSession: false,
		ConfirmClose:   true,
		KeepUnsaved:    false,
	}
}

// Load reads dir/settings.json. A missing file returns the defaults. A
// corrupt file is copied to settings.json.bak and the defaults are used.
func Load(dir string) (Settings, error) {
	s := Default()
	path := filepath.Join(dir, fileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		_ = os.WriteFile(path+".bak", data, 0o600)
		return Default(), nil
	}
	s.normalize()
	return s, nil
}

// Save writes s atomically to dir/settings.json.
func Save(dir string, s Settings) error {
	s.normalize()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicfile.Write(dir, fileName, data)
}

// Ensure writes s when dir has no settings file yet, so a first launch
// leaves the defaults on disk for the user to inspect and edit.
func Ensure(dir string, s Settings) error {
	path := filepath.Join(dir, fileName)
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return Save(dir, s)
}

func (s *Settings) normalize() {
	// Theme ids are checked against the theme registry by the app, which
	// knows every built-in palette; an empty value only means the user
	// has not chosen one yet.
	if s.Theme == "" {
		s.Theme = "system"
	}
	switch s.Language {
	case "", "zh-Hans", "en":
	default:
		s.Language = ""
	}
	if s.FontSize < 10 || s.FontSize > 28 {
		s.FontSize = 13
	}
	if s.LineHeight < 1.2 || s.LineHeight > 2.0 {
		s.LineHeight = 1.5
	}
	switch s.LineNumbers {
	case "absolute", "relative", "off":
	default:
		s.LineNumbers = "absolute"
	}
	switch s.TabSize {
	case 2, 4, 8:
	default:
		s.TabSize = 4
	}
	switch s.Whitespace {
	case "selection", "none", "all", "trailing":
	default:
		s.Whitespace = "selection"
	}
	switch s.DefaultEncoding {
	case "utf-8", "utf-8-bom", "utf-16le", "utf-16be", "gb18030":
	default:
		s.DefaultEncoding = "utf-8"
	}
	switch s.DefaultEOL {
	case "\n", "\r\n", "\r":
	default:
		s.DefaultEOL = "\n"
	}
	switch s.LargeFileMB {
	case 10, 20, 50:
	default:
		s.LargeFileMB = 20
	}
	if s.SidebarWidth < 160 || s.SidebarWidth > 480 {
		s.SidebarWidth = 240
	}
}
