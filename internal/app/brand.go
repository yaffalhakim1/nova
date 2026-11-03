package app

import (
	_ "embed"
	"fmt"
)

// The app's mark, embedded so the binary carries its own icon: the .exe's is
// set by the build, and this is what the window and the taskbar button show.
//
//go:embed icon.png
var iconData []byte

// iconPNG returns the app's icon as PNG bytes.
func iconPNG() ([]byte, error) {
	if len(iconData) == 0 {
		return nil, fmt.Errorf("no embedded icon")
	}
	return iconData, nil
}

// setWindowIcon puts the app's mark in the title bar and on the taskbar
// button, which is what Windows shows for a window before the .exe's own icon
// is picked up.
func (a *app) setWindowIcon() {
	raw, err := iconPNG()
	if err != nil {
		// No icon is worth failing the window for.
		return
	}
	_ = a.win.SetIcon(raw)
}
