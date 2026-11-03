package steam

import (
	"regexp"
	"strings"

	"github.com/egoist/mygo"
)

// keySecretName is where the Steam Web API key is kept: the system's credential
// store (Windows Credential Manager), never a file in the data folder.
const keySecretName = "steam-api-key"

// keyShape is what a Steam Web API key looks like: 32 hex characters. Checking
// it catches an obvious typo before it becomes a 401 the user has to interpret.
var keyShape = regexp.MustCompile(`^[0-9A-Fa-f]{32}$`)

// ValidKeyShape reports whether s could be a Steam Web API key.
func ValidKeyShape(s string) bool { return keyShape.MatchString(strings.TrimSpace(s)) }

// RedactedKey is how a key is shown: its first four characters and its length, so
// the user can tell which key is stored without the window holding the rest.
func RedactedKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "••••"
	}
	return key[:4] + "••••••••"
}

// LoadKey reads the stored Steam Web API key, or "" when there is none.
//
// A read failure is treated as "no key" rather than an error: the launcher works
// without one, and an unavailable credential store must not stop it.
func LoadKey() string {
	raw, err := mygo.Secrets.Get(keySecretName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

// SaveKey stores a key in the system's credential store. It does not check the
// key's shape: that is the window's job, so the reason can be explained there.
func SaveKey(key string) error {
	return mygo.Secrets.Set(keySecretName, []byte(strings.TrimSpace(key)))
}

// ForgetKey removes the stored key.
func ForgetKey() error {
	return mygo.Secrets.Delete(keySecretName)
}
