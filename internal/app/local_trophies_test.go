package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yafit/nova/internal/library"
)

// TestLocalTrophiesComeFromGSE checks the whole path a repack uses: the app id
// in the game folder, the emulator's save file, and the card that reads it.
// No Steam key and no network are involved.
func TestLocalTrophiesComeFromGSE(t *testing.T) {
	const appID = "4115450"

	gameDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(gameDir, "steam_appid.txt"), []byte(appID), 0o644); err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	saveDir := filepath.Join(root, appID)
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{
		"A_First": {"earned": true, "earned_time": 1781932638},
		"B_Second": {"earned": false, "earned_time": 0}
	}`
	if err := os.WriteFile(filepath.Join(saveDir, "achievements.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	installs := installsOf(1)
	installs[0].AppID = ""
	installs[0].Dir = gameDir
	installs[0].Source = library.Manual

	a := launcher(t, installs)
	a.gseRoot = root

	tt := openHub(t, a, 1280, 840)
	for i := 0; i < 20; i++ {
		tt.Frame()
		tt.Advance(50 * time.Millisecond)
	}

	for _, want := range []string{"1 of 2", "A_First"} {
		if !tt.HasText(want) {
			t.Fatalf("trophies card does not show %q; texts = %q", want, tt.Texts())
		}
	}
	if tt.HasText("Not tracked for this game.") {
		t.Fatal("trophies card still shows the empty state")
	}
}
