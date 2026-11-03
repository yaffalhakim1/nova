package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The interface font is embedded and registered before the window draws, so
// the look does not depend on the machine having Hanken Grotesk installed.
func TestRegisterFonts(t *testing.T) {
	registerFonts()

	for _, f := range ui.FontFamilies() {
		if f.Name != interfaceFont {
			continue
		}
		if !f.Registered {
			t.Fatalf("%s is listed but not the app's registered font", interfaceFont)
		}
		return
	}
	t.Fatalf("%s is not in FontFamilies after registerFonts", interfaceFont)
}

// The theme asks for the interface font by name, with the design's own
// fallbacks behind it.
func TestThemeFont(t *testing.T) {
	want := interfaceFont + ", Segoe UI, system-ui"
	if got := theme().Font; got != want {
		t.Fatalf("theme font = %q, want %q", got, want)
	}
}
