package app

import "testing"

// TestParseGamePID checks the tasklist row the session watcher reads. The CSV
// has a quoted image name and a quoted PID, and other processes must be ignored.
func TestParseGamePID(t *testing.T) {
	out := "\"explorer.exe\",\"1234\",\"Console\",\"1\",\"10 K\"\r\n" +
		"\"PBZ.exe\",\"2792\",\"Console\",\"1\",\"128,844 K\"\r\n"

	pid, ok := parseGamePID(out, "PBZ.exe")
	if !ok || pid != 2792 {
		t.Fatalf("parseGamePID = %d, %v; want 2792, true", pid, ok)
	}
	if _, ok := parseGamePID(out, "missing.exe"); ok {
		t.Fatal("a missing process was reported as running")
	}
	if _, ok := parseGamePID("not csv", "PBZ.exe"); ok {
		t.Fatal("junk output was parsed as a PID")
	}
}
