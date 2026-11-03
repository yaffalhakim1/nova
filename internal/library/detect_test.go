package library

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"Batman: Arkham Knight", "batman arkham knight"},
		{"Batman.Arkham.Knight", "batman arkham knight"},
		{"CONTROL-Resonant-SteamRIP.com", "control resonant steamrip com"},
		{"  Red   Dead 2  ", "red dead 2"},
	} {
		if got := normalize(tc.in); got != tc.want {
			t.Errorf("normalize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCleanName(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"CONTROL-Resonant-SteamRIP.com", "CONTROL Resonant"},
		{"Assassin's Creed Shadows", "Assassin's Creed Shadows"},
		{"DARK SOULS III", "DARK SOULS III"},
		{"Hollow Knight - Voidheart", "Hollow Knight"},
	} {
		if got := cleanName(tc.in); got != tc.want {
			t.Errorf("cleanName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsNoise(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"dxwebsetup", true},
		{"vcredist_x64", true},
		{"UnityCrashHandler64", true},
		{"Cyberpunk2077", false},
		{"CONTROLResonant", false},
	} {
		if got := isNoise(tc.in); got != tc.want {
			t.Errorf("isNoise(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestGameNamePrefersTheInnerFolder checks a repack's nested game folder names
// the install, not the release folder around it.
func TestGameNamePrefersTheInnerFolder(t *testing.T) {
	dir := `Y:\games\CONTROL-Resonant-SteamRIP.com`
	exe := dir + `\CONTROL Resonant\CONTROLResonant.exe`
	if got := gameName(dir, exe, "CONTROL-Resonant-SteamRIP.com"); got != "CONTROL Resonant" {
		t.Fatalf("gameName = %q, want the inner folder", got)
	}

	flat := `Y:\SteamLibrary\steamapps\common\Batman Arkham Knight`
	if got := gameName(flat, flat+`\BatmanAK.exe`, "Batman Arkham Knight"); got != "Batman Arkham Knight" {
		t.Fatalf("gameName flat = %q", got)
	}
}

func TestVDFPair(t *testing.T) {
	key, value, ok := vdfPair(`	"installdir"		"Cyberpunk 2077"`)
	if !ok || key != "installdir" || value != "Cyberpunk 2077" {
		t.Fatalf("vdfPair = %q %q %v", key, value, ok)
	}
	if _, _, ok := vdfPair(`	{`); ok {
		t.Fatal("a block opener read as a pair")
	}
}

// TestDetectCustomFolder checks a games folder is read, and the executables a
// repack ships are not mistaken for games.
func TestDetectCustomFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"\\CONTROL-Resonant-SteamRIP.com\\CONTROL Resonant\\CONTROLResonant.exe")
	writeFile(t, root+"\\CONTROL-Resonant-SteamRIP.com\\_CommonRedist\\dxwebsetup.exe")
	writeFile(t, root+"\\Assassin's Creed Shadows\\ACShadows.exe")

	got := Detect(Roots{Custom: []string{root}})
	if len(got) != 2 {
		t.Fatalf("found %d games, want 2: %+v", len(got), got)
	}
	for _, in := range got {
		if in.Exe == "" {
			t.Errorf("%q has no executable", in.Name)
		}
		if in.Name == "" {
			t.Errorf("%q has no name", in.Dir)
		}
	}
}

// TestDetectSteamManifest checks a manifest becomes an install with its app id.
func TestDetectSteamManifest(t *testing.T) {
	lib := t.TempDir()
	writeFile(t, lib+"\\steamapps\\appmanifest_1091500.acf",
		"\"AppState\"\n{\n\t\"appid\"\t\t\"1091500\"\n\t\"name\"\t\t\"Cyberpunk 2077\"\n\t\"installdir\"\t\t\"Cyberpunk 2077\"\n}\n")
	writeFile(t, lib+"\\steamapps\\common\\Cyberpunk 2077\\bin\\x64\\Cyberpunk2077.exe")

	got := Detect(Roots{SteamLibraries: []string{lib}})
	if len(got) != 1 {
		t.Fatalf("found %d games, want 1: %+v", len(got), got)
	}
	in := got[0]
	if in.AppID != "1091500" || in.Source != Steam {
		t.Fatalf("install = %+v", in)
	}
	if in.Exe == "" {
		t.Fatal("no executable found")
	}
}

// TestSteamToolsAreNotGames checks the redistributable folders Steam installs
// are left out.
func TestSteamToolsAreNotGames(t *testing.T) {
	lib := t.TempDir()
	writeFile(t, lib+"\\steamapps\\appmanifest_228980.acf",
		"\"AppState\"\n{\n\t\"appid\"\t\t\"228980\"\n\t\"name\"\t\t\"Steamworks Common Redistributables\"\n\t\"installdir\"\t\t\"Steamworks Shared\"\n}\n")
	writeFile(t, lib+"\\steamapps\\common\\Steamworks Shared\\some.exe")

	if got := Detect(Roots{SteamLibraries: []string{lib}}); len(got) != 0 {
		t.Fatalf("found %d games, want none: %+v", len(got), got)
	}
}

// TestMissingRootIsQuiet checks a folder that does not exist finds nothing
// instead of failing.
func TestMissingRootIsQuiet(t *testing.T) {
	if got := Detect(Roots{Custom: []string{t.TempDir() + "\\nope"}}); len(got) != 0 {
		t.Fatalf("found %d games in a missing folder", len(got))
	}
}
