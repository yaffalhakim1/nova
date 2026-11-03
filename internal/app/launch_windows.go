package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// startGame starts a game's executable from its own folder and returns the
// running process. The caller decides whether to wait for it; a launcher
// never does.
func startGame(e entry) (*os.Process, error) {
	cmd := exec.Command(e.inst.Exe)
	// Many games resolve their data files relative to the working directory,
	// so it is the game's own folder, never the launcher's.
	cmd.Dir = filepath.Dir(e.inst.Exe)
	hideWindow(cmd)

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd.Process, nil
}

// launch starts a game and releases it, for when nobody needs the process.
func launch(e entry) error {
	proc, err := startGame(e)
	if err != nil {
		return err
	}
	return proc.Release()
}

// canLaunch reports whether a game has an executable to start.
func canLaunch(e entry) bool {
	return e.inst.Exe != "" && fileExists(e.inst.Exe)
}

// launchProblem explains why a game cannot be started, or returns nothing
// when it can.
func launchProblem(e entry) string {
	switch {
	case e.inst.Dir == "":
		return "no install folder"
	case e.inst.Exe == "":
		return "no executable found in " + e.inst.Dir
	case !fileExists(e.inst.Exe):
		return "missing " + e.inst.Exe
	default:
		return ""
	}
}

// fileExists reports whether path is a file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// shortExe is a game's executable name, for the row that shows it.
func shortExe(e entry) string {
	if e.inst.Exe == "" {
		return ""
	}
	return filepath.Base(e.inst.Exe)
}

// trimPath shortens a folder to its last two parts, so a long install path
// reads as `common\Cyberpunk 2077`.
func trimPath(dir string) string {
	parts := strings.FieldsFunc(dir, func(r rune) bool { return r == '\\' || r == '/' })
	if len(parts) <= 2 {
		return dir
	}
	return filepath.Join(parts[len(parts)-2:]...)
}
