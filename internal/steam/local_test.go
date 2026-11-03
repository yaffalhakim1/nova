package steam

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeLocalFile writes a fixture inside a temporary Steam install.
func writeLocalFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// localConfig is a fixture shaped like the real file: the playtime block sits
// under Software → Valve → Steam → apps, one block per app id.
const localConfig = `"UserLocalConfigStore"
{
	"Software"
	{
		"Valve"
		{
			"Steam"
			{
				"apps"
				{
					"1091500"
					{
						"LastPlayed"		"1774516242"
						"Playtime"		"377"
					}
					"228980"
					{
						"Playtime"		"0"
					}
					"431960"
					{
						"cloud"
						{
							"last_sync_state"		"synchronized"
						}
					}
				}
			}
		}
	}
}
`

// TestPlaytimeReadsMinutesAndLastPlayed checks the one conversion that matters:
// Steam counts minutes, so 377 is 6h17m, not 377 seconds or 377 hours.
func TestPlaytimeReadsMinutesAndLastPlayed(t *testing.T) {
	root := t.TempDir()
	writeLocalFile(t, filepath.Join(root, "userdata", "384463082", "config", "localconfig.vdf"), localConfig)

	got := Playtime(root, "384463082")

	e, ok := got["1091500"]
	if !ok {
		t.Fatalf("app 1091500 missing: %+v", got)
	}
	if want := 6*time.Hour + 17*time.Minute; e.Played != want {
		t.Fatalf("played = %v, want %v", e.Played, want)
	}
	if e.Last.Unix() != 1774516242 {
		t.Fatalf("last = %v", e.Last)
	}
}

// TestPlaytimeSkipsAppsWithNoTime checks an app that Steam tracked only for
// cloud saves is not reported as a played game.
func TestPlaytimeSkipsAppsWithNoTime(t *testing.T) {
	root := t.TempDir()
	writeLocalFile(t, filepath.Join(root, "userdata", "384463082", "config", "localconfig.vdf"), localConfig)

	got := Playtime(root, "384463082")
	if _, ok := got["431960"]; ok {
		t.Fatal("an app with no playtime was reported")
	}
	if _, ok := got["228980"]; ok {
		t.Fatal("an app played for zero minutes was reported")
	}
}

// TestAccountsPicksTheMostRecent checks the default account is the one Steam
// itself used last, which is what "most recently used" means here. The ids are a
// real SteamID64 and its folder, so the arithmetic between them is proven rather
// than assumed.
func TestAccountsPicksTheMostRecent(t *testing.T) {
	root := t.TempDir()
	writeLocalFile(t, filepath.Join(root, "config", "loginusers.vdf"), `"users"
{
	"76561198344728810"
	{
		"AccountName"		"frostbite12325"
		"PersonaName"		"Leetcode Enjoyer"
		"timestamp"		"1793612794"
	}
	"76561198769879948"
	{
		"AccountName"		"other"
		"PersonaName"		"CBS Khazan | 2"
		"timestamp"		"1778641173"
	}
}
`)

	accounts := Accounts(root)
	if len(accounts) != 2 {
		t.Fatalf("accounts = %d, want 2", len(accounts))
	}
	if accounts[0].Persona != "Leetcode Enjoyer" {
		t.Fatalf("most recent = %q", accounts[0].Persona)
	}
	if accounts[0].ID != "384463082" {
		t.Fatalf("folder id = %q, want the low 32 bits of the steam id", accounts[0].ID)
	}
	if accounts[1].Persona != "CBS Khazan | 2" {
		t.Fatalf("second = %q", accounts[1].Persona)
	}
}

// TestEmptyValueDoesNotEndTheFile pins the bug that made this reader silently
// drop a whole account: Steam writes empty values, as `"storepreferences" ""`,
// and treating an empty token as the end of input loses everything after it.
func TestEmptyValueDoesNotEndTheFile(t *testing.T) {
	root := t.TempDir()
	writeLocalFile(t, filepath.Join(root, "userdata", "1", "config", "localconfig.vdf"), `"UserLocalConfigStore"
{
	"friends"
	{
		"storepreferences"		""
		"after"		"kept"
	}
	"Software"
	{
		"Valve"
		{
			"Steam"
			{
				"apps"
				{
					"480"
					{
						"Playtime"		"377"
					}
				}
			}
		}
	}
}
`)

	got := Playtime(root, "1")
	e, ok := got["480"]
	if !ok {
		t.Fatalf("the apps block after an empty value was lost: %+v", got)
	}
	if want := 6*time.Hour + 17*time.Minute; e.Played != want {
		t.Fatalf("played = %v, want %v", e.Played, want)
	}
}

// TestBracesInsideValuesAreNotBlocks pins the other real-file trap: Steam stores
// JSON as an escaped string, and counting its braces as block delimiters
// unbalances the whole file.
func TestBracesInsideValuesAreNotBlocks(t *testing.T) {
	root := t.TempDir()
	writeLocalFile(t, filepath.Join(root, "userdata", "1", "config", "localconfig.vdf"), `"UserLocalConfigStore"
{
	"UIStoreLocalSteamUIState"		"{\"nVersion\":1,\"collectionFilter\":{\"a\":[1,2]}}"
	"keyboard_haptic_custom"		"{\"eHapticType\":2}"
	"Software"
	{
		"Valve"
		{
			"Steam"
			{
				"apps"
				{
					"480"
					{
						"Playtime"		"60"
					}
				}
			}
		}
	}
}
`)

	got := Playtime(root, "1")
	if _, ok := got["480"]; !ok {
		t.Fatalf("braces inside a value broke the parse: %+v", got)
	}
	if got["480"].Played != time.Hour {
		t.Fatalf("played = %v, want 1h", got["480"].Played)
	}
}

// TestMissingSteamIsQuiet checks a machine without Steam yields nothing rather
// than an error the launcher would have to handle.
func TestMissingSteamIsQuiet(t *testing.T) {
	root := filepath.Join(t.TempDir(), "nope")
	if got := Accounts(root); len(got) != 0 {
		t.Fatalf("accounts = %+v", got)
	}
	if got := Playtime(root, "384463082"); len(got) != 0 {
		t.Fatalf("playtime = %+v", got)
	}
}

// TestBrokenLocalConfigIsQuiet checks a damaged file is an empty history, not a
// crash on startup.
func TestBrokenLocalConfigIsQuiet(t *testing.T) {
	root := t.TempDir()
	writeLocalFile(t, filepath.Join(root, "userdata", "1", "config", "localconfig.vdf"), "not a vdf {{{\"")

	if got := Playtime(root, "1"); len(got) != 0 {
		t.Fatalf("playtime = %+v", got)
	}
}

// TestInstallRootIsNotALibraryFolder pins the trap: a Steam *library* folder has
// steamapps but no userdata, so looking for accounts there finds none.
func TestInstallRootIsNotALibraryFolder(t *testing.T) {
	lib := t.TempDir()
	writeLocalFile(t, filepath.Join(lib, "steamapps", "libraryfolders.vdf"), "")

	if got := Accounts(lib); len(got) != 0 {
		t.Fatalf("a library folder was treated as an install: %+v", got)
	}

	install := t.TempDir()
	writeLocalFile(t, filepath.Join(install, "config", "loginusers.vdf"), `"users"
{
	"76561198000000001"
	{
		"PersonaName"		"Solo"
		"timestamp"		"100"
	}
}
`)
	if got := Accounts(install); len(got) != 1 {
		t.Fatalf("install root not recognised: %+v", got)
	}
}
