// Package library finds the games already installed on this machine: the ones
// Steam has a manifest for, and the ones in a folder of the user's own.
package library

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Source is where an install was found.
type Source string

const (
	// Steam is a game installed through Steam.
	Steam Source = "steam"
	// Manual is a game in a folder of the user's own.
	Manual Source = "manual"
)

// Install is one game found on disk.
type Install struct {
	// Name is what the game is called: the manifest's name, or the folder's.
	Name string
	// Dir is the game's folder.
	Dir string
	// Exe is the game's main executable, or empty when none was found.
	Exe string
	// AppID is the Steam app id, for a Steam install.
	AppID string
	Source Source

	// Genre, Description and Hue are what the user typed when adding a game by
	// hand. They are empty for every scanned game, which takes its details from
	// the store instead.
	Genre       string
	Description string
	Hue         int
}

// Label is a short line saying where the game came from, in words a reader
// understands: never the internal source name.
func (in Install) Label() string {
	switch in.Source {
	case Steam:
		return "Steam"
	case Manual:
		return "added by hand"
	default:
		return "folder"
	}
}

// Roots is where to look for games.
type Roots struct {
	// SteamLibraries are Steam library folders, each holding steamapps.
	SteamLibraries []string
	// Custom are folders whose immediate children are games, as a hand-made
	// games folder is.
	Custom []string
	// Manual are the games added one at a time by picking their own folder.
	Manual []ManualGame
}

// Detect finds every game under the roots, in name order.
func Detect(r Roots) []Install {
	seen := map[string]bool{}
	var out []Install

	// The games the user added by hand come first, because they are the ones
	// with a name, a genre and an executable they chose; the scans fill in
	// around them and cannot override them.
	for _, in := range manualInstalls(r.Manual) {
		key := strings.ToLower(in.Dir)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, in)
	}

	for _, lib := range r.SteamLibraries {
		for _, in := range scanSteam(lib) {
			key := strings.ToLower(in.Dir)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, in)
		}
	}

	for _, dir := range r.Custom {
		for _, in := range scanFolder(dir) {
			key := strings.ToLower(in.Dir)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, in)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// scanSteam reads the app manifests of one library folder. A library that
// cannot be read yields nothing rather than an error: a missing drive is a
// normal state on a machine that unplugged one.
func scanSteam(library string) []Install {
	apps := filepath.Join(library, "steamapps")
	manifests, err := filepath.Glob(filepath.Join(apps, "appmanifest_*.acf"))
	if err != nil {
		return nil
	}

	var out []Install
	for _, path := range manifests {
		in, ok := readManifest(path, apps)
		if !ok {
			continue
		}
		out = append(out, in)
	}
	return out
}

// readManifest turns one appmanifest into an install.
func readManifest(path, apps string) (Install, bool) {
	fields, err := readVDF(path)
	if err != nil {
		return Install{}, false
	}

	appID := fields["appid"]
	installdir := fields["installdir"]
	if appID == "" || installdir == "" {
		return Install{}, false
	}

	dir := filepath.Join(apps, "common", installdir)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return Install{}, false
	}

	// A tool or a redistributable is not a game, and there are dozens of them
	// in a Steam library.
	if isJunkInstall(fields["name"]) {
		return Install{}, false
	}

	in := Install{
		Name:   fields["name"],
		Dir:    dir,
		AppID:  appID,
		Source: Steam,
	}
	in.Exe = findExe(dir, in.Name)
	if in.Exe == "" {
		return Install{}, false
	}
	return in, true
}

// isJunkInstall reports whether a Steam entry is a tool or a redistribution
// folder rather than a game.
func isJunkInstall(name string) bool {
	lower := strings.ToLower(name)
	for _, bad := range []string{
		"steamworks common redistributables",
		"steam linux runtime",
		"steamworks shared",
		"proton",
		"steamvr",
	} {
		if strings.Contains(lower, bad) {
			return true
		}
	}
	return false
}

// scanFolder reads one games folder: each immediate child that holds an
// executable, at the top or in a subfolder, is a game.
func scanFolder(root string) []Install {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	var out []Install
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		exe := findExe(dir, e.Name())
		if exe == "" {
			continue
		}
		out = append(out, Install{
			Name:   gameName(dir, exe, e.Name()),
			Dir:    dir,
			Exe:    exe,
			Source: Manual,
		})
	}
	return out
}

// gameName names an install: the folder inside the game's that holds its
// executable when there is one, as a repack nests the game in a folder of its
// own, and the folder's own name otherwise.
func gameName(dir, exe, fallback string) string {
	rel, err := filepath.Rel(dir, filepath.Dir(exe))
	if err != nil || rel == "." {
		return cleanName(fallback)
	}
	if i := strings.IndexRune(rel, filepath.Separator); i > 0 {
		rel = rel[:i]
	}
	if name := cleanName(rel); name != "" {
		return name
	}
	return cleanName(fallback)
}

// findExe looks for the game's main executable: a file named after the game,
// in the folder or a few levels down, and failing that the largest executable
// that is not a redistributable installer. Game builds put theirs as deep as
// Binaries\Win64, so the search goes four levels.
func findExe(dir, name string) string {
	var best string
	var bestSize int64

	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != dir && isNoiseDir(d.Name()) {
				return filepath.SkipDir
			}
			if depthOf(dir, path) > 4 {
				return filepath.SkipDir
			}
			return nil
		}

		lower := strings.ToLower(d.Name())
		if !strings.HasSuffix(lower, ".exe") {
			return nil
		}
		base := strings.TrimSuffix(d.Name(), ".exe")
		if isNoise(base) {
			return nil
		}

		// A game's executable is named after it, with or without the spaces
		// its folder has: "Cyberpunk2077" for "Cyberpunk 2077".
		if withoutSpaces(normalize(base)) == withoutSpaces(normalize(name)) {
			best = path
			return filepath.SkipAll
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		if best == "" || info.Size() > bestSize {
			best, bestSize = path, info.Size()
		}
		return nil
	})

	return best
}

// withoutSpaces drops the spaces of a normalized name, so the executable's
// name and its folder's compare equal.
func withoutSpaces(s string) string {
	return strings.ReplaceAll(s, " ", "")
}

// depthOf is how many folders deep path sits under root.
func depthOf(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}

// isNoiseDir reports whether a folder holds installers or engine helpers
// rather than the game.
func isNoiseDir(name string) bool {
	lower := strings.ToLower(name)
	for _, bad := range []string{
		"_commonredist", "redist", "directx", "dotnet", "__installer",
		"support", "installer", "crashreport", "easyanticheat", "battleye",
	} {
		if strings.Contains(lower, bad) {
			return true
		}
	}
	return false
}

// isNoise reports whether an executable is an installer or a helper rather
// than the game.
func isNoise(base string) bool {
	lower := strings.ToLower(base)
	for _, bad := range []string{
		"unins", "setup", "redist", "vcredist", "dxsetup", "dxwebsetup",
		"dotnet", "oalinst", "launcher_installer", "crashreport",
		"cleanup", "helper", "unitycrashhandler", "eac_", "easyanticheat",
	} {
		if strings.Contains(lower, bad) {
			return true
		}
	}
	return false
}

// cleanName turns a folder name into something a person would read: the
// release tags a repack appends are dropped, and the separator characters a
// folder name uses in place of spaces become spaces.
func cleanName(folder string) string {
	name := strings.TrimSpace(folder)
	if i := strings.Index(name, " - "); i > 0 {
		name = name[:i]
	}
	if i := strings.Index(name, "_"); i > 0 {
		name = name[:i]
	}
	for _, tag := range []string{"-SteamRIP", "-Repack", "-ElAmigos", "-FitGirl", "-DODI", ".com"} {
		if i := strings.Index(strings.ToLower(name), strings.ToLower(tag)); i > 0 {
			name = name[:i]
		}
	}
	name = strings.NewReplacer("-", " ", ".", " ").Replace(name)
	return strings.TrimSpace(name)
}
