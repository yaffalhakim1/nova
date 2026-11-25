package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/yafit/nova/internal/backup"
	"github.com/yafit/nova/internal/gse"
)

// TestSteamTrophiesAskForAKey checks the honest empty state for a Steam game
// whose Web API key has not been stored yet.
func TestSteamTrophiesAskForAKey(t *testing.T) {
	a := launcher(t, installsOf(1))
	tt := openHub(t, a, 1280, 840)
	tt.Frame()

	if !tt.HasText("Add a Steam API key in Settings.") {
		t.Fatalf("the card does not explain the missing key; texts = %q", tt.Texts())
	}
}

// TestBackupRestoresPlaytimeAndUnlocks checks the whole file path a player
// uses after a reinstall: export, lose the data, import, and see the hours and
// local unlocks again.
func TestBackupRestoresPlaytimeAndUnlocks(t *testing.T) {
	a := launcher(t, installsOf(1))
	e := a.entries[0]
	a.stat.record(e.slug(), 30*time.Minute)
	a.favorites[e.slug()] = true
	a.gseRoot = t.TempDir()
	unlocked := time.Unix(1781932638, 0)
	if err := gse.Save(a.gseRoot, "4115450", []gse.Record{
		{APIName: "A_FIRST", Achieved: true, Unlocked: unlocked},
	}); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "nova-backup.json")
	if err := backup.Write(path, a.collectBackup()); err != nil {
		t.Fatal(err)
	}
	loaded, err := backup.Read(path)
	if err != nil {
		t.Fatal(err)
	}

	// A fresh install has no history at all.
	b := launcher(t, installsOf(1))
	b.gseRoot = t.TempDir()
	b.applyBackup(backup.Merge(b.collectBackup(), loaded))

	if got := b.stat.get(e.slug()).Played; got < 30*time.Minute {
		t.Fatalf("restored play time = %v, want 30m", got)
	}
	if got := b.stat.get(e.slug()).Sessions; got != 1 {
		t.Fatalf("restored sessions = %d, want 1", got)
	}
	if !b.favorites[e.slug()] || !b.libCfg.Favorites[e.slug()] {
		t.Fatal("favorite was not restored into the saved config")
	}
	records, ok := gse.Read(b.gseRoot, "4115450")
	if !ok || len(records) != 1 || !records[0].Achieved || !records[0].Unlocked.Equal(unlocked) {
		t.Fatalf("restored achievements = %+v, %v", records, ok)
	}
}
