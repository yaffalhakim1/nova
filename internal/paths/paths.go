// Package paths locates the folders this app keeps its own data in.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// DataDir is where the app caches covers, store pages and its own state.
func DataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "nova")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	return dir, nil
}
