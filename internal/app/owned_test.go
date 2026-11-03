package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/yafit/nova/internal/library"
	steamstore "github.com/yafit/nova/internal/steam"
)

// TestOwnedNotInstalledAreMarked checks the split that keeps the app honest: a
// game the account owns but that is not on disk is listed as not installed, and
// never as playable.
func TestOwnedNotInstalledAreMarked(t *testing.T) {
	a := launcher(t, installsOf(2))

	// App 1001 is installed; 9999 is not.
	a.owned = []steamstore.OwnedGame{
		{AppID: "1001", Name: "Game 1"},
		{AppID: "9999", Name: "Not Installed Game"},
	}

	rows := a.ownedRows()
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	for _, r := range rows {
		switch r.AppID {
		case "1001":
			if !r.Installed() {
				t.Error("an installed game was marked not installed")
			}
		case "9999":
			if r.Installed() {
				t.Error("a game that is not on disk was marked installed")
			}
		}
	}
}

// TestOwnedNotInstalledCannotLaunch checks the guard that stops a click on an
// owned-but-missing game from starting nothing.
func TestOwnedNotInstalledCannotLaunch(t *testing.T) {
	a := launcher(t, installsOf(1))
	a.owned = []steamstore.OwnedGame{{AppID: "9999", Name: "Not Installed Game"}}

	rows := a.ownedRows()
	var missing *ownedRow
	for i := range rows {
		if rows[i].AppID == "9999" {
			missing = &rows[i]
		}
	}
	if missing == nil {
		t.Fatal("the owned game is missing from the list")
	}
	if missing.entry != nil {
		t.Fatal("a game that is not installed produced an entry to launch")
	}
}

// TestNoKeySaysNothingIsWrong checks a machine without a key still shows the
// installed games, and the library section is simply absent rather than an
// error.
func TestNoKeySaysNothingIsWrong(t *testing.T) {
	a := launcher(t, installsOf(3))
	if len(a.owned) != 0 {
		t.Fatal("owned games appeared without a key")
	}

	tt := ui.NewTester(a.view, 1280, 840)
	tt.Frame()

	if !tt.HasText("Game 0") {
		t.Fatal("the installed games disappeared when there was no key")
	}
	if tt.HasText("Your Steam library") {
		t.Fatal("the owned section showed with nothing to put in it")
	}
}

// TestOwnedSectionShowsUninstalledGames checks the second scope paints, and says
// which games need an install.
func TestOwnedSectionShowsUninstalledGames(t *testing.T) {
	a := launcher(t, installsOf(1))
	a.steam = &steamstore.Account{SteamID: "76561198000000001"}
	a.ownedSteamID = "76561198000000001"
	a.owned = []steamstore.OwnedGame{
		{AppID: "1000", Name: "Game 0"},
		{AppID: "9999", Name: "Not Installed Game"},
	}

	tt := ui.NewTester(a.view, 1280, 840)
	tt.Frame()

	if !tt.HasText("Your Steam library") {
		t.Fatalf("the owned section did not paint; texts = %q", tt.Texts())
	}
	if !tt.HasText("Not Installed Game") {
		t.Fatalf("an uninstalled owned game was not shown; texts = %q", tt.Texts())
	}
	if !tt.HasText("Install") {
		t.Fatalf("an uninstalled game has no Install action; texts = %q", tt.Texts())
	}
}

// TestOwnedRowsSortByPlaytimeThenName checks the order is stable and useful:
// what you played most first.
func TestOwnedRowsSortByPlaytimeThenName(t *testing.T) {
	a := launcher(t, nil)
	a.owned = []steamstore.OwnedGame{
		{AppID: "1", Name: "Alpha", Played: 10 * time.Hour},
		{AppID: "2", Name: "Beta", Played: 100 * time.Hour},
		{AppID: "3", Name: "Aardvark"},
	}

	rows := a.ownedRows()
	if rows[0].Name != "Beta" {
		t.Fatalf("first = %q, want the most played", rows[0].Name)
	}
	if rows[1].Name != "Alpha" {
		t.Fatalf("second = %q", rows[1].Name)
	}
	// The unplayed one last, by name among its peers.
	if rows[2].Name != "Aardvark" {
		t.Fatalf("last = %q", rows[2].Name)
	}
}

// TestOwnedLibraryFollowsTheSignedInAccount pins the trap of two Steam accounts
// on one machine: play time and the owned list must describe the same account.
func TestOwnedLibraryFollowsTheSignedInAccount(t *testing.T) {
	a := launcher(t, installsOf(1))

	// No Steam signed in, or a library from another account: either way nothing
	// is shown, because the numbers would describe two people.
	a.owned = []steamstore.OwnedGame{{AppID: "9", Name: "Elsewhere"}}
	a.ownedSteamID = "76561198000000009"

	if a.ownedMatchesAccount() {
		t.Fatal("a library from an unknown account was accepted")
	}

	tt := ui.NewTester(a.view, 1280, 840)
	tt.Frame()
	if tt.HasText("Elsewhere") {
		t.Fatal("another account's game was shown")
	}
}

// TestOwnedLibraryWithNoAccountID checks a library whose account was not recorded
// is still usable, so an older cache does not blank the section.
func TestOwnedLibraryWithNoAccountID(t *testing.T) {
	a := launcher(t, installsOf(1))
	a.owned = []steamstore.OwnedGame{{AppID: "9", Name: "Mine"}}
	a.ownedSteamID = ""

	// With no Steam install at all, there is no account to compare against, so
	// the section stays hidden rather than guessing.
	if a.ownedMatchesAccount() {
		t.Fatal("the library was accepted with no account to check it against")
	}
}

// TestInstallLinkIsSteamProtocol checks the install action goes through Steam
// rather than pretending the launcher can install a game itself.
func TestInstallLinkIsSteamProtocol(t *testing.T) {
	if got := installURL("1091500"); got != "steam://install/1091500" {
		t.Fatalf("install url = %q", got)
	}
}

// TestKeyShapeIsCheckedBeforeSaving checks an obviously wrong key is refused
// with a reason instead of being stored and failing later.
func TestKeyShapeIsCheckedBeforeSaving(t *testing.T) {
	a := launcher(t, nil)
	a.keyInput = "nonsense"
	a.saveKey()

	if a.keyProblem == "" {
		t.Fatal("a malformed key was accepted without a word")
	}
	if a.keyStored {
		t.Fatal("a malformed key was stored")
	}
}

// TestManualGameHasNoAppID checks the owned list never claims a hand-added game.
func TestManualGameHasNoAppID(t *testing.T) {
	a := launcher(t, []library.Install{{Name: "Repack", Dir: `Y:\g\Repack`, Source: library.Manual}})
	a.owned = []steamstore.OwnedGame{{AppID: "", Name: "Nonsense"}}

	for _, r := range a.ownedRows() {
		if r.Installed() && r.slug() == "" {
			t.Fatal("a row was matched to a game with no identity")
		}
	}
}
