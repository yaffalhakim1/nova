package library

import (
	"os"
	"path/filepath"
	"strings"
)

// AddManual turns a folder the user picked into a manual game.
//
// The folder picked is the game's own, not a folder full of games: that is the
// difference from adding a games folder, and it is what makes a single repack
// installable by hand.
func AddManual(dir string) (ManualGame, bool) {
	dir = filepath.Clean(strings.TrimSpace(dir))
	if dir == "" || dir == "." {
		return ManualGame{}, false
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return ManualGame{}, false
	}

	name := cleanName(filepath.Base(dir))
	exe := findExe(dir, name)
	if exe == "" {
		return ManualGame{}, false
	}

	rel, err := filepath.Rel(dir, exe)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		// The executable must live inside the game's own folder: that folder is
		// its working directory when it starts.
		return ManualGame{}, false
	}

	return ManualGame{Dir: dir, Exe: rel, Name: name}, true
}

// ExeFor resolves a manual entry's main executable, or reports that it is gone.
//
// A relative Exe is resolved against the game's folder, and refused when it
// climbs out of it: a hand-edited config must not be able to point the launcher
// at an arbitrary file.
func ExeFor(m ManualGame) (string, bool) {
	dir := filepath.Clean(m.Dir)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return "", false
	}

	if m.Exe == "" {
		exe := findExe(dir, m.Name)
		if exe == "" {
			return "", false
		}
		return exe, true
	}

	rel := filepath.Clean(m.Exe)
	if filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}

	exe := filepath.Join(dir, rel)
	info, err := os.Stat(exe)
	if err != nil || info.IsDir() {
		return "", false
	}
	return exe, true
}

// manualInstalls turns the saved manual games into installs, skipping the ones
// whose folder or executable has gone since they were added.
func manualInstalls(list []ManualGame) []Install {
	var out []Install
	for _, m := range list {
		exe, ok := ExeFor(m)
		if !ok {
			continue
		}

		name := strings.TrimSpace(m.Name)
		if name == "" {
			name = cleanName(filepath.Base(filepath.Clean(m.Dir)))
		}
		if name == "" {
			continue
		}

		out = append(out, Install{
			Name:        name,
			Dir:         filepath.Clean(m.Dir),
			Exe:         exe,
			Source:      Manual,
			Genre:       strings.TrimSpace(m.Genre),
			Description: strings.TrimSpace(m.Description),
			Hue:         m.Hue,
		})
	}
	return out
}
