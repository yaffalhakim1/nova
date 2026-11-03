package app

import (
	_ "embed"

	"github.com/egoist/mygo/ui"
)

// The interface font, embedded so the app looks the same on a machine that has
// never installed it. The design asks for Hanken Grotesk; a Windows machine
// without it falls back to Segoe UI, and the thin weights the whole look depends
// on are not there.
//
// This is the variable font: one file carries every weight from Thin to Black,
// which is what the design uses (200 for titles, 300 for body, 400 to 500 for
// controls). Hanken Grotesk is licensed under the SIL Open Font License 1.1;
// see fonts/OFL.txt.
//
//go:embed fonts/HankenGrotesk.ttf
var hankenGrotesk []byte

// interfaceFont is the family name the theme asks for.
const interfaceFont = "Hanken Grotesk"

// registerFonts makes the embedded font available to the UI toolkit. It is
// called once, before the window is built.
//
// A font that cannot be registered is not fatal: the interface falls back to the
// system's font, which is what it looked like before this was added.
func registerFonts() {
	if len(hankenGrotesk) == 0 {
		return
	}
	_ = ui.RegisterFont(hankenGrotesk, interfaceFont)
}
