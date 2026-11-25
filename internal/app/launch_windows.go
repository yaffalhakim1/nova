package app

import (
	"encoding/csv"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// noWindow hides a helper's console. It is never applied to a game: SW_HIDE
// on a game window breaks Discord's game detection.
const noWindow = 0x08000000

// startGame asks Explorer to start the game. A process created directly by
// Nova is invisible to Discord's game detection; the same exe launched by
// Explorer is detected. Explorer inherits the game's folder as its cwd.
func startGame(e entry) error {
	cmd := exec.Command("explorer.exe", e.inst.Exe)
	cmd.Dir = filepath.Dir(e.inst.Exe)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: noWindow}
	return cmd.Start()
}

// waitForGamePID finds the process Explorer started. The shell returns before
// the game appears, so the watcher polls for a bounded time.
func waitForGamePID(exe string, limit time.Duration) (uint32, bool) {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if pid, ok := gamePID(exe); ok {
			return pid, true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return 0, false
}

// gamePID returns the running process with the game's executable name.
func gamePID(exe string) (uint32, bool) {
	name := filepath.Base(exe)
	out := tasklist("/FI", "IMAGENAME eq "+name, "/FO", "CSV", "/NH")
	return parseGamePID(out, name)
}

// parseGamePID reads tasklist's CSV output. It is separate so the parser can
// be tested without starting a process.
func parseGamePID(out, name string) (uint32, bool) {
	rows := csv.NewReader(strings.NewReader(out))
	records, err := rows.ReadAll()
	if err != nil {
		return 0, false
	}
	for _, row := range records {
		if len(row) < 2 || !strings.EqualFold(row[0], name) {
			continue
		}
		pid, err := strconv.ParseUint(strings.TrimSpace(row[1]), 10, 32)
		if err == nil {
			return uint32(pid), true
		}
	}
	return 0, false
}

// pidAlive reports whether the game process is still listed.
func pidAlive(pid uint32) bool {
	id := strconv.FormatUint(uint64(pid), 10)
	out := tasklist("/FI", "PID eq "+id, "/FO", "CSV", "/NH")
	return strings.Contains(out, id)
}

// tasklist runs Windows' process lister without showing a console.
func tasklist(args ...string) string {
	cmd := exec.Command("tasklist", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: noWindow}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
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
