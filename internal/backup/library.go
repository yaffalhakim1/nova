package backup

import "github.com/yafit/nova/internal/library"

// mergeLibrary joins the two lists without dropping a game the user added on
// either machine. A manual entry is identified by its folder.
func mergeLibrary(fresh, old library.Config) library.Config {
	out := fresh
	out.Steam = unique(append(append([]string{}, fresh.Steam...), old.Steam...))
	out.Folders = unique(append(append([]string{}, fresh.Folders...), old.Folders...))
	out.Hidden = unique(append(append([]string{}, fresh.Hidden...), old.Hidden...))

	byDir := map[string]library.ManualGame{}
	for _, m := range old.Manual {
		byDir[m.Dir] = m
	}
	for _, m := range fresh.Manual {
		byDir[m.Dir] = m
	}
	out.Manual = make([]library.ManualGame, 0, len(byDir))
	for _, m := range byDir {
		out.Manual = append(out.Manual, m)
	}
	return out
}

// unique keeps the first copy of each value and the original order.
func unique(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
