package library

import (
	"strings"
)

// normalize lowercases a name and keeps only its letters and digits, so
// "Batman: Arkham Knight", "Batman Arkham Knight" and "Batman.Arkham.Knight"
// all read the same.
func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_' || r == '.' || r == ':' || r == '\'':
			b.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
