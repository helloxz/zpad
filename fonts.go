package main

import (
	_ "embed"
	"fmt"

	"github.com/egoist/mygo/ui"
)

var (
	//go:embed fonts/JetBrainsMono-Regular.ttf
	fontJetBrainsRegular []byte
	//go:embed fonts/JetBrainsMono-Bold.ttf
	fontJetBrainsBold []byte
	//go:embed fonts/JetBrainsMono-Italic.ttf
	fontJetBrainsItalic []byte
	//go:embed fonts/JetBrainsMono-BoldItalic.ttf
	fontJetBrainsBoldItalic []byte
)

// registerFonts adds the bundled JetBrains Mono faces. Call it before any
// window exists.
func registerFonts() error {
	for _, face := range []struct {
		name string
		data []byte
	}{
		{"Regular", fontJetBrainsRegular},
		{"Bold", fontJetBrainsBold},
		{"Italic", fontJetBrainsItalic},
		{"BoldItalic", fontJetBrainsBoldItalic},
	} {
		if err := ui.RegisterFont(face.data, "JetBrains Mono"); err != nil {
			return fmt.Errorf("register JetBrains Mono %s: %w", face.name, err)
		}
	}
	return nil
}
