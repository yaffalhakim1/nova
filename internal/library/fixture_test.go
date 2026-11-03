package library

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile creates a file and its folders, for a fixture.
func writeFile(t *testing.T, path string, content ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := ""
	if len(content) > 0 {
		body = content[0]
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
