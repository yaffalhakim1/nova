package steam

import (
	"os"
	"path/filepath"
	"strings"
)

// capsuleName is the portrait library art Steam's own client shows: 300x450,
// exactly 2:3, which is what a tile is.
const capsuleName = "library_capsule.jpg"

// Capsule looks for a game's library capsule in Steam's own cache. The client
// keeps one for every installed game, so the launcher shows the same art the
// Steam library does, with no network and no credentials.
func Capsule(appID string) (string, bool) {
	if appID == "" {
		return "", false
	}

	for _, root := range steamRoots() {
		dir := filepath.Join(root, "appcache", "librarycache", appID)
		if path, ok := findCapsule(dir); ok {
			return path, true
		}
	}
	return "", false
}

// findCapsule looks for the capsule under an app's cache folder, which keeps
// its images in a subfolder named after a hash.
func findCapsule(dir string) (string, bool) {
	if path := filepath.Join(dir, capsuleName); isFile(path) {
		return path, true
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if path := filepath.Join(dir, e.Name(), capsuleName); isFile(path) {
			return path, true
		}
	}
	return "", false
}

// isFile reports whether path names a non-empty file.
func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}

// steamRoots are the Steam installs this machine has.
func steamRoots() []string {
	var roots []string
	for _, guess := range []string{
		`C:\Program Files (x86)\Steam`,
		`C:\Program Files\Steam`,
	} {
		if info, err := os.Stat(filepath.Join(guess, "appcache")); err == nil && info.IsDir() {
			roots = append(roots, guess)
		}
	}
	return dedupeRoots(roots)
}

// dedupeRoots drops repeats, keeping the order.
func dedupeRoots(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		k := strings.ToLower(filepath.Clean(v))
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, v)
	}
	return out
}
