package steam

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The files Steam keeps for a signed-in account, relative to the install.
const (
	loginUsersFile = "loginusers.vdf"
	userdataFolder = "userdata"
	configFolder   = "config"
	localConfigVDF = "localconfig.vdf"
)

// Account is one Steam account that has signed in on this machine.
type Account struct {
	// ID is the folder under userdata: the low 32 bits of SteamID64, decimal.
	ID string
	// SteamID is the 64-bit account id, as Steam writes it.
	SteamID string
	// Persona is the name Steam shows for the account.
	Persona string
	// Last is when the account last signed in, which is how the most recently
	// used one is chosen.
	Last time.Time
}

// Entry is what Steam knows about one app's play time.
type Entry struct {
	// Played is the total time Steam recorded.
	Played time.Duration
	// Last is the last time Steam recorded the game running, or zero.
	Last time.Time
}

// node is one block of a Valve data file: the key/value pairs that sit in it and
// the blocks nested under it.
type node struct {
	pairs  map[string]string
	blocks map[string]*node
}

// get walks a path of lowercased block names and returns the block it names.
func (n *node) get(path ...string) *node {
	cur := n
	for _, name := range path {
		if cur == nil {
			return nil
		}
		cur = cur.blocks[name]
	}
	return cur
}

// Accounts returns the accounts that have signed in on this machine, the most
// recently used first. A machine without Steam yields nothing, never an error:
// the launcher still starts.
func Accounts(root string) []Account {
	tree := readVDFTree(filepath.Join(root, "config", loginUsersFile))
	users := tree.get("users")
	if users == nil {
		return nil
	}

	var out []Account
	for _, id := range sortedKeys(users.blocks) {
		fields := users.blocks[id]
		persona := fields.pairs["personaname"]
		if persona == "" {
			continue
		}
		out = append(out, Account{
			ID:      accountID(id),
			SteamID: id,
			Persona: persona,
			Last:    unixTime(fields.pairs["timestamp"]),
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Last.After(out[j].Last)
	})
	return out
}

// DefaultAccount is the account Steam itself used last, or nil when it has none.
func DefaultAccount(root string) *Account {
	accounts := Accounts(root)
	if len(accounts) == 0 {
		return nil
	}
	return &accounts[0]
}

// Playtime reads one account's play time, by Steam app id. An app Steam tracked
// only for cloud saves, or never ran, is left out rather than reported as zero.
func Playtime(root, account string) map[string]Entry {
	if account == "" {
		return nil
	}

	path := filepath.Join(root, userdataFolder, account, configFolder, localConfigVDF)
	apps := readVDFTree(path).get("userlocalconfigstore", "software", "valve", "steam", "apps")
	if apps == nil {
		return nil
	}

	out := map[string]Entry{}
	for id, entry := range apps.blocks {
		mins, err := strconv.Atoi(strings.TrimSpace(entry.pairs["playtime"]))
		if err != nil || mins <= 0 {
			continue
		}
		out[id] = Entry{
			Played: time.Duration(mins) * time.Minute,
			Last:   unixTime(entry.pairs["lastplayed"]),
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// readVDFTree parses a Valve data file. A missing or damaged file is an empty
// tree, so every caller has one shape to handle.
func readVDFTree(path string) *node {
	empty := &node{pairs: map[string]string{}, blocks: map[string]*node{}}

	raw, err := os.ReadFile(path)
	if err != nil {
		return empty
	}

	toks := tokenize(string(raw))
	p := &parser{toks: toks}
	return p.document()
}

// document reads the file's outer blocks.
func (p *parser) document() *node {
	root := &node{pairs: map[string]string{}, blocks: map[string]*node{}}

	for !p.end() {
		name, ok := p.key()
		if !ok {
			// A stray brace at the top level: skip it rather than stop, so one
			// bad character cannot hide the rest of the file.
			p.pos++
			continue
		}
		if p.peek() == "{" {
			p.pos++
			root.blocks[name] = p.block()
			continue
		}
		// A pair at the top level, which a document's own settings may use.
		if value, ok := p.value(); ok {
			root.pairs[name] = value
		}
	}
	return root
}

// block reads one block's pairs and nested blocks, up to its closing brace.
func (p *parser) block() *node {
	n := &node{pairs: map[string]string{}, blocks: map[string]*node{}}

	for !p.end() {
		tok := p.peek()
		switch tok {
		case "}":
			p.pos++
			return n
		case "{":
			// A stray opener: skip it.
			p.pos++
			continue
		}

		name, ok := p.key()
		if !ok {
			p.pos++
			continue
		}

		if p.peek() == "{" {
			p.pos++
			n.blocks[name] = p.block()
			continue
		}

		value, ok := p.value()
		if !ok {
			// A key with no value before the block closes.
			return n
		}
		n.pairs[name] = value
	}
	return n
}

// parser reads a flat token stream.
type parser struct {
	toks []string
	pos  int
}

// end reports whether the stream is finished. It does not test for an empty
// token: a Valve file holds empty values, as `"storepreferences" ""`, and
// treating those as the end would drop everything after them.
func (p *parser) end() bool { return p.pos >= len(p.toks) }

// peek returns the next token.
func (p *parser) peek() string {
	if p.end() {
		return ""
	}
	return p.toks[p.pos]
}

// key reads a block or pair name, lowercased like Valve's own files compare.
func (p *parser) key() (string, bool) {
	if p.end() {
		return "", false
	}
	tok := p.toks[p.pos]
	if tok == "{" || tok == "}" {
		return "", false
	}
	p.pos++
	return strings.ToLower(tok), true
}

// value reads a plain value, which may be empty.
func (p *parser) value() (string, bool) {
	if p.end() {
		return "", false
	}
	tok := p.toks[p.pos]
	if tok == "{" || tok == "}" {
		return "", false
	}
	p.pos++
	return tok, true
}

// tokenize returns the quoted strings and the braces of a Valve data file, in
// order.
//
// Escaped quotes inside a value are honoured, and a brace inside one is not a
// block delimiter: Steam stores JSON as an escaped string, as in
// `"ChatStore" "{\"a\":1}"`, and those braces would otherwise unbalance the
// whole file.
func tokenize(raw string) []string {
	var out []string
	var b strings.Builder
	in := false
	escaped := false

	for _, r := range raw {
		if escaped {
			escaped = false
			if in {
				b.WriteRune(r)
			}
			continue
		}

		switch {
		case r == '\\' && in:
			escaped = true
		case r == '"':
			if in {
				out = append(out, b.String())
				b.Reset()
			}
			in = !in
		case in:
			b.WriteRune(r)
		case r == '{' || r == '}':
			out = append(out, string(r))
		}
	}
	return out
}

// accountID turns a SteamID64 into the folder name under userdata: the low 32
// bits of the id, in decimal.
func accountID(steamID string) string {
	const base = 76561197960265728
	id, err := strconv.ParseUint(strings.TrimSpace(steamID), 10, 64)
	if err != nil || id < base {
		return ""
	}
	return strconv.FormatUint(id-base, 10)
}

// unixTime reads a unix-seconds field, or the zero time when it is missing.
func unixTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return time.Time{}
	}
	return time.Unix(n, 0)
}

// installGuesses are the places a Steam *install* sits, which is where its
// accounts and their play time live. This is deliberately not the same list as
// a Steam library: a library folder such as Y:\SteamLibrary holds steamapps and
// games but no userdata, so looking for accounts there finds none.
func installGuesses() []string {
	var out []string
	for _, guess := range []string{
		`C:\Program Files (x86)\Steam`,
		`C:\Program Files\Steam`,
		`D:\Steam`,
		`E:\Steam`,
	} {
		if isInstall(guess) {
			out = append(out, guess)
		}
	}

	// Steam can be installed elsewhere, and a registry-free guess is the drive
	// letter next to it.
	for c := 'C'; c <= 'Z'; c++ {
		guess := string(c) + `:\Steam`
		if isInstall(guess) {
			out = append(out, guess)
		}
	}
	return out
}

// isInstall reports whether dir looks like a Steam install that has signed an
// account in, rather than merely a library folder.
func isInstall(dir string) bool {
	if info, err := os.Stat(filepath.Join(dir, "config", loginUsersFile)); err == nil && !info.IsDir() {
		return true
	}
	entries, err := os.ReadDir(filepath.Join(dir, userdataFolder))
	return err == nil && len(entries) > 0
}

// InstallRoot returns the Steam install to read accounts and play time from, or
// "" when this machine has none.
func InstallRoot() string {
	roots := installGuesses()
	if len(roots) == 0 {
		return ""
	}
	return roots[0]
}
func sortedKeys(m map[string]*node) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
