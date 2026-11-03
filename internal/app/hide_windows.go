//go:build windows

package app

import (
	"os/exec"
	"syscall"
)

// createNoWindow is CREATE_NO_WINDOW: the game starts without a console
// window of its own flashing up.
const createNoWindow = 0x08000000

// hideWindow keeps the launched game from opening a console window.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}
