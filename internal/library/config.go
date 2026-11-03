package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Config is the file that says where this machine keeps its games.
type Config struct {
	// Steam are Steam library folders, each holding steamapps. Empty means
	// discover them.
	Steam []string `json:"steam"`
	// Folders are folders whose immediate children are games, as a hand-made
	// games folder is.
	Folders []string `json:"folders"`
	// Hidden are the games the user removed from the library, by key. The
	// folders stay on disk; the launcher just stops showing them.
	Hidden []string `json:"hidden,omitempty"`
	// Manual are the games added one at a time, each with its own folder.
	Manual []ManualGame `json:"manual,omitempty"`
}

// ManualGame is one game the user added by picking its folder: the case a scan
// of a whole games folder cannot help with.
//
// Exe is relative to Dir on purpose. An absolute path would break the moment
// the folder moved, and a relative one is resolved again on every load.
type ManualGame struct {
	// Dir is the game's own folder, which is also its working directory when
	// it is launched.
	Dir string `json:"dir"`
	// Exe is the main executable, relative to Dir. Empty means "find it".
	Exe string `json:"exe,omitempty"`
	// Name is what the game is called. Empty means "use the folder's name".
	Name string `json:"name"`
	// Genre and Description are what the user typed, shown when Steam has no
	// store page for the game.
	Genre       string `json:"genre,omitempty"`
	Description string `json:"description,omitempty"`
	// Hue is the colour of the generated cover, 0 meaning "derive it from the
	// name".
	Hue int `json:"hue,omitempty"`
}

// ConfigName is the file inside the data directory that holds Config.
const ConfigName = "library.json"

// DefaultConfig is where games are looked for when the user has said nothing:
// Steam's own libraries, and every drive's games folder.
func DefaultConfig() Config {
	return Config{
		Steam:   SteamRoots(),
		Folders: gamesFolders(),
	}
}// LoadConfig reads the config file, falling back to the defaults for anything
// it does not say. A missing or damaged file is the default, not an error: the
// launcher still starts.
func LoadConfig(dataDir string) Config {
	cfg := Config{}

	raw, err := os.ReadFile(filepath.Join(dataDir, ConfigName))
	if err == nil {
		_ = json.Unmarshal(raw, &cfg)
	}

	if cfg.Steam == nil {
		cfg.Steam = SteamRoots()
	}
	if cfg.Folders == nil {
		cfg.Folders = gamesFolders()
	}
	return cfg
}

// SaveConfig writes the config file, so the folders found once are kept.
func SaveConfig(dataDir string, cfg Config) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dataDir, ConfigName), raw, 0o644)
}

// Roots turns a config into the roots Detect reads.
func (c Config) Roots() Roots {
	return Roots{SteamLibraries: c.Steam, Custom: c.Folders, Manual: c.Manual}
}

// AddManual records a game added by picking its folder. A second add of the same
// folder replaces the first rather than listing the game twice.
func (c *Config) AddManual(m ManualGame) {
	for i, cur := range c.Manual {
		if strings.EqualFold(cur.Dir, m.Dir) {
			c.Manual[i] = m
			return
		}
	}
	c.Manual = append(c.Manual, m)
}

// ManualFor returns the manual entry for a game folder, if there is one.
func (c Config) ManualFor(dir string) (ManualGame, bool) {
	for _, m := range c.Manual {
		if strings.EqualFold(m.Dir, dir) {
			return m, true
		}
	}
	return ManualGame{}, false
}

// gamesFolders lists the folders on this machine that hold games outside
// Steam: every drive's "games" and "Games", in that order.
func gamesFolders() []string {
	var out []string
	for _, drive := range driveLetters() {
		for _, name := range []string{"games", "Games", "GAMES"} {
			dir := filepath.Join(drive, name)
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				out = append(out, dir)
				break
			}
		}
	}
	return dedupe(out)
}

// Describe is a short line about where games were looked for, for the window
// to show when it found none.
func (c Config) Describe() string {
	var parts []string
	if len(c.Steam) > 0 {
		parts = append(parts, "Steam")
	}
	for _, f := range c.Folders {
		parts = append(parts, f)
	}
	if len(parts) == 0 {
		return "no Steam library and no games folder"
	}
	return strings.Join(parts, ", ")
}
