package library

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// readVDF reads the flat key/value pairs of a Valve data file, such as an
// appmanifest. It stops at the first nested block, which is all a manifest
// needs: every field this package reads sits at the top level.
func readVDF(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := map[string]string{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64<<10), 1<<20)

	for scanner.Scan() {
		key, value, ok := vdfPair(scanner.Text())
		if ok {
			out[strings.ToLower(key)] = value
			continue
		}
		// A nested block opens with a bare quoted name; its own pairs are
		// indented further and are not ours to read.
		if strings.HasPrefix(strings.TrimSpace(scanner.Text()), "{") {
			continue
		}
	}
	return out, scanner.Err()
}

// vdfPair reads one `"key" "value"` line.
func vdfPair(line string) (key, value string, ok bool) {
	parts := quotedFields(line)
	if len(parts) < 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// quotedFields returns the quoted strings of a line, in order.
func quotedFields(line string) []string {
	var out []string
	in := false
	var b strings.Builder

	for _, r := range line {
		switch {
		case r == '"':
			if in {
				out = append(out, b.String())
				b.Reset()
			}
			in = !in
		case in:
			b.WriteRune(r)
		}
	}
	return out
}

// LibraryFolders reads Steam's libraryfolders.vdf, which lists every drive a
// Steam library sits on, and returns the library roots themselves.
func LibraryFolders(steamRoot string) []string {
	path := filepath.Join(steamRoot, "steamapps", "libraryfolders.vdf")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var out []string
	for _, line := range strings.Split(string(raw), "\n") {
		fields := quotedFields(line)
		if len(fields) != 2 || fields[0] != "path" {
			continue
		}
		lib := filepath.FromSlash(fields[1])
		if info, err := os.Stat(filepath.Join(lib, "steamapps")); err == nil && info.IsDir() {
			out = append(out, lib)
		}
	}
	return out
}

// SteamRoots returns the Steam library folders this machine has: the ones its
// install knows about, plus the conventional places a library is put.
func SteamRoots() []string {
	var roots []string

	for _, guess := range []string{
		`C:\Program Files (x86)\Steam`,
		`C:\Program Files\Steam`,
		`D:\Steam`,
		`E:\Steam`,
	} {
		if info, err := os.Stat(filepath.Join(guess, "steamapps")); err == nil && info.IsDir() {
			roots = append(roots, guess)
			roots = append(roots, LibraryFolders(guess)...)
		}
	}

	// Library folders can sit on a drive without Steam itself there, so read
	// every drive's usual location too.
	for _, drive := range driveLetters() {
		for _, name := range []string{"SteamLibrary", "Steam", "Games\\Steam"} {
			lib := filepath.Join(drive, name)
			if info, err := os.Stat(filepath.Join(lib, "steamapps")); err == nil && info.IsDir() {
				roots = append(roots, lib)
			}
		}
	}

	return dedupe(roots)
}

// driveLetters lists the drives that exist.
func driveLetters() []string {
	var out []string
	for c := 'A'; c <= 'Z'; c++ {
		dir := string(c) + `:\`
		if _, err := os.Stat(dir); err == nil {
			out = append(out, dir)
		}
	}
	return out
}

// dedupe drops repeated paths, keeping the order.
func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		key := strings.ToLower(filepath.Clean(v))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, v)
	}
	return out
}
