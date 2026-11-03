// Package app is the launcher's interface: a console home screen of the games
// installed on this machine, and a detail page for the one in focus.
package app

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/yafit/nova/internal/library"
	"github.com/yafit/nova/internal/paths"
	steamstore "github.com/yafit/nova/internal/steam"
)

// The three sections of the top bar. They are pages, as the design's are: each
// one replaces the body under the header rather than narrowing one list.
const (
	sectionHome = iota
	sectionFavorites
	sectionLibrary
)

type app struct {
	win   *mygo.Window
	theme *ui.Theme

	dataDir string

	libCfg   library.Config
	installs []library.Install
	scanning bool

	// section is the page under the header. query narrows the library, and
	// sortBy orders it; genreFilter narrows it to one genre, "" for all.
	section    int
	sortBy     int
	sortChoice string
	genreFilter string
	query      string
	lastQuery  string
	// lastKey is the section, sort and query the entries were built for, so a
	// frame that changed none of them does not rebuild the list.
	lastKey int
	lastSor int
	lastGenre string
	entries []entry
	selected int

	favorites map[string]bool

	// controller lights the header's pad when one is plugged in.
	controller bool

	current *entry

	overlay     int
	searchOpen  bool
	addOpen     bool
	addGameOpen bool
	playingOpen bool
	searchQuery string
	folderInput string
	muted       bool
	toast       string

	// draft is the "Add a game" form being filled in.
	draft addDraft

	// owned is the games the Steam account owns, installed or not, and the
	// problem that kept it from loading, if any.
	owned        []steamstore.OwnedGame
	ownedSteamID string
	ownedProblem string

	// steamKey is the Web API key, kept in the credential store rather than in
	// a file; keyStored says whether one is there.
	steamKey     string
	keyStored    bool
	keyInput     string
	keyProblem   string
	steamKeyOpen bool

	// steam is the signed-in Steam account, resolved once at startup so the
	// play time and the owned library always describe the same account.
	steam *steamstore.Account

	// stat is the play history, and playing the session the app started.
	stat     stats
	playing  *playingSession
	playNote string

	// covers holds the games' real artwork, loaded in the background.
	covers *covers

	// achievements holds the Steam achievements loaded per game, so the
	// trophies card reads them without touching the network in a frame.
	achievements trophyBoard

	// store is the Steam store data, and shots its screenshots.
	store *steamstore.Client
	shots *shots

	// The game whose blurb is expanded, by key.
	expanded map[string]bool
	// openedAt is when the detail page was opened, so it can rise in.
	openedAt time.Time

	// bg cross-fades the screen between games' colours.
	bg backdrop

	// appIDs is the Steam app id resolved for each game, by key.
	mu     sync.Mutex
	appIDs map[string]string
	// storeData is each game's store page, by key, once resolved.
	storeData map[string]*steamstore.Details
	// resolved marks the games whose Steam identity has been looked up.
	resolved map[string]bool
	// hidden are the games the user removed, by key.
	hidden map[string]bool

	status string

	// epoch is when the app started, the clock the ambient drift runs on.
	epoch time.Time
	// drift is the ambient particles' own state, advanced a frame at a time.
	drift particleField

	// trophiesOpen shows the full achievement list for the open game.
	trophiesOpen bool
}

// entry is one game on the home screen: a folder found on disk.
type entry struct {
	inst library.Install
}

// name is what the game is called.
func (e entry) name() string { return e.inst.Name }

// slug identifies the game across frames and in the favorites. A Steam install
// uses its app id, which is stable when the folder moves; a folder uses its
// path.
func (e entry) slug() string {
	if e.inst.AppID != "" {
		return "steam-" + e.inst.AppID
	}
	return strings.ToLower(e.inst.Dir)
}

// installed reports whether the game is on disk.
func (e entry) installed() bool { return e.inst.Dir != "" }

// Run loads what the app needs and shows the window.
func Run() error {
	a := &app{
		favorites: map[string]bool{},
		expanded:  map[string]bool{},
		lastKey:   -1,
		epoch:     time.Now(),
	}
	if err := a.load(); err != nil {
		return err
	}

	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:           "Nova",
			Width:           1280,
			Height:          840,
			MinWidth:        900,
			MinHeight:       600,
			StateKey:        "main",
			BackgroundColor: "#05070f",
			Content:         ui.View(a.view),
		})
		a.setWindowIcon()
		a.setMenu()
	})
	return mygo.App.Run()
}

// load prepares the library, the covers and the play history. Every step
// degrades instead of aborting, so the window opens on a machine with no games
// and no network.
func (a *app) load() error {
	a.theme = theme()

	dir, err := paths.DataDir()
	if err != nil {
		return fmt.Errorf("locate data folder: %w", err)
	}
	a.dataDir = dir

	a.covers = newCovers(a.dataDir)
	a.covers.ready = func() { a.update(func() {}) }

	a.store = steamstore.NewClient(a.dataDir)
	a.shots = newShots(a.dataDir)
	a.shots.ready = func() { a.update(func() {}) }

	a.libCfg = library.LoadConfig(a.dataDir)
	a.stat = loadStats(a.dataDir)

	// The Steam key is read once, from the credential store, and never written
	// to the data folder.
	a.steamKey = steamstore.LoadKey()
	a.keyStored = a.steamKey != ""
	a.resolveSteamAccount()

	a.scanLibrary()
	a.refresh()
	return nil
}

// scanLibrary finds the games installed on this machine. Walking every drive
// is slow enough to be done in the background: the window opens on the games
// found before, and fills in when the scan answers.
func (a *app) scanLibrary() {
	a.scanning = true
	go func() {
		found := library.Detect(a.libCfg.Roots())
		a.update(func() {
			a.installs = found
			a.scanning = false
			a.lastKey = -1
			a.refresh()

			// Steam already knows the play time of the games it installed, so
			// fold it in once the entries exist: a fresh install then shows real
			// hours instead of an empty history. Merging is a no-op the second
			// time, so a rescan is free.
			a.importSteam()
			if err := a.stat.save(a.dataDir); err != nil {
				a.status = "could not save play time: " + err.Error()
			}

			// The owned list needs a key and the network, so it loads in its own
			// background pass and never holds up the games already found.
			a.loadOwnedGames()

			a.status = fmt.Sprintf("%d games installed", len(found))
		})
	}()
}

// update runs fn and asks for a new frame.
func (a *app) update(fn func()) {
	if a.win == nil {
		fn()
		return
	}
	a.win.Update(fn)
}

// view builds the window: the home screen, or the open game over it.
func (a *app) view(c *ui.Context) {
	a.refresh()
	a.applyTheme(c)

	a.shortcuts(c)
	a.flushToast(c)

	// The modal's open flag lives on the app: a fresh pointer each frame
	// restarts the overlay's animation and its content never settles.
	a.searchOpen = a.overlay == overlaySearch
	a.addOpen = a.overlay == overlayAddFolder
	a.addGameOpen = a.overlay == overlayAddGame
	a.steamKeyOpen = a.overlay == overlaySteamKey
	a.playingOpen = a.overlay == overlayPlaying

	// The backdrop is the window's, not a page's: it is painted once, at full
	// strength, under whichever screen is showing. That is what keeps the
	// detail page's rise from fading the ground to black on its first frames.
	ui.Column(c).Fill().
		Draw(func(p *ui.Painter, r ui.Rect) {
			a.bg.paint(p, r, c)
			a.paintParticlesStateful(p, r)
		}).
		Children(func() {
			if a.current != nil {
				a.detailPage(c)
			} else if a.section == sectionLibrary {
				a.libraryPage(c)
			} else {
				a.homePage(c)
			}
		})
	a.searchOverlay(c)
	a.addFolderOverlay(c)
	a.addGameOverlay(c)
	a.steamKeyOverlay(c)
	a.trophiesModal(c)
	a.playingOverlay(c)
}

// applyTheme gives the window the theme of the game in focus: its accent
// colours the primary buttons and the focus ring, so the whole screen takes on
// that game's colour.
func (a *app) applyTheme(c *ui.Context) {
	t := *a.theme

	var e *entry
	if a.current != nil {
		e = a.current
	} else {
		e = a.focused()
	}

	if e != nil {
		accent := a.accentFor(*e)
		t.Accent = accent
		t.AccentHover = mixColor(accent, white, 0.18)
		t.AccentPressed = mixColor(accent, deep, 0.15)
		t.Focus = accent
		t.AccentText = ui.Hex("#080c1a")
	}

	c.SetTheme(&t)
}

// mixColor blends two colours, t of the way from a to b.
func mixColor(a, b ui.Color, t float32) ui.Color {
	ar, ag, ab := float32(a.R)/255, float32(a.G)/255, float32(a.B)/255
	br, bg, bb := float32(b.R)/255, float32(b.G)/255, float32(b.B)/255
	return ui.RGBA(
		uint8(clamp01(ar+(br-ar)*t)*255),
		uint8(clamp01(ag+(bg-ag)*t)*255),
		uint8(clamp01(ab+(bb-ab)*t)*255),
		255,
	)
}

// refresh rebuilds the entries for the current section, sort, genre and query.
func (a *app) refresh() {
	if a.query == a.lastQuery && a.section == a.lastKey &&
		a.sortBy == a.lastSor && a.genreFilter == a.lastGenre && a.entries != nil {
		return
	}
	a.lastQuery, a.lastKey, a.lastSor, a.lastGenre = a.query, a.section, a.sortBy, a.genreFilter

	all := a.allEntries()
	a.entries = a.filtered(all)

	if a.selected >= len(a.entries) {
		a.selected = max(0, len(a.entries)-1)
	}
}

// allEntries is every game the launcher knows: the ones installed on this
// machine, then the ones the Steam account owns but does not have installed.
//
// The owned games are included so the carousel can browse the whole library
// rather than only what happens to be on disk. They carry no executable, so
// `canLaunch` refuses them and nothing ever pretends they can start.
func (a *app) allEntries() []entry {
	out := make([]entry, 0, len(a.installs))
	for _, in := range a.installs {
		e := entry{inst: in}
		if a.isHidden(e.slug()) {
			continue
		}
		out = append(out, e)
	}

	out = append(out, a.ownedOnlyEntries(out)...)
	return out
}

// ownedOnlyEntries are the owned games that are not installed, as entries.
//
// They are only produced when the loaded library belongs to the signed-in
// account, so two accounts can never be mixed, and they borrow the account's
// play time so the carousel shows real hours for a game that is not on disk.
func (a *app) ownedOnlyEntries(installed []entry) []entry {
	if !a.ownedMatchesAccount() {
		return nil
	}

	// The app id is the identity a Steam game has either way.
	have := make(map[string]bool, len(installed))
	for _, e := range installed {
		if id := e.inst.AppID; id != "" {
			have[id] = true
		}
	}

	var out []entry
	for _, g := range a.owned {
		if g.AppID == "" || have[g.AppID] || a.isHidden("steam-"+g.AppID) {
			continue
		}
		out = append(out, entry{inst: library.Install{
			Name:   g.Name,
			AppID:  g.AppID,
			Source: library.Steam,
		}})
	}
	return out
}

// isHidden reports whether the user removed a game from the library.
func (a *app) isHidden(slug string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.hidden[slug] {
		return true
	}
	for _, h := range a.libCfg.Hidden {
		if h == slug {
			return true
		}
	}
	return false
}

// filtered applies the section, the genre, the search and the sort to the
// entries.
func (a *app) filtered(all []entry) []entry {
	out := make([]entry, 0, len(all))
	for _, e := range all {
		if a.query != "" && !matchesQuery(e, a.query) {
			continue
		}
		if a.section == sectionFavorites && !a.favorites[e.slug()] {
			continue
		}
		// The genre chips are the library page's own filter; "" is every genre.
		if a.section == sectionLibrary && a.genreFilter != "" &&
			a.genreLine(e) != a.genreFilter {
			continue
		}
		out = append(out, e)
	}

	a.sortEntries(out)
	return out
}

// libraryGenres lists the genres the library page offers as chips, with the
// most common first so the useful ones are reachable without scrolling. The
// first entry is always "All".
func (a *app) libraryGenres() []string {
	counts := map[string]int{}
	for _, e := range a.allEntries() {
		if g := a.genreLine(e); g != "" {
			counts[g]++
		}
	}

	genres := make([]string, 0, len(counts))
	for g := range counts {
		genres = append(genres, g)
	}
	sort.SliceStable(genres, func(i, j int) bool {
		if counts[genres[i]] != counts[genres[j]] {
			return counts[genres[i]] > counts[genres[j]]
		}
		return strings.ToLower(genres[i]) < strings.ToLower(genres[j])
	})

	return append([]string{"All"}, genres...)
}

// The ways the library can be sorted, as the design's select offers them.
const (
	sortRecent = iota
	sortAZ
	sortMostPlayed
	sortAdded
)

// ownedFor returns what Steam knows about a game's play time, for a game the
// account owns. It is how a game that is not installed still shows its real
// hours in the carousel.
func (a *app) ownedFor(e entry) (steamstore.OwnedGame, bool) {
	if e.inst.AppID == "" || !a.ownedMatchesAccount() {
		return steamstore.OwnedGame{}, false
	}
	for _, g := range a.owned {
		if g.AppID == e.inst.AppID {
			return g, true
		}
	}
	return steamstore.OwnedGame{}, false
}

// playedOf is how long a game has been played: what the Web API reports for an
// owned game, and this app's own record otherwise. The larger of the two wins,
// the same rule the play history uses, so a game cannot shrink.
func (a *app) playedOf(e entry) time.Duration {
	local := a.stat.get(e.slug()).Played
	if g, ok := a.ownedFor(e); ok && g.Played > local {
		return g.Played
	}
	return local
}

// lastOf is when a game was last played, from either source.
func (a *app) lastOf(e entry) time.Time {
	local := a.stat.get(e.slug()).Last
	if g, ok := a.ownedFor(e); ok && g.Last.After(local) {
		return g.Last
	}
	return local
}

// sortEntries orders the games the way the current sort asks.
//
// Home and Favorites open on **the games played the most**, which is the order
// a console's library reads in and what the design's tiles are a cabinet of.
// The Library page obeys its own select instead.
func (a *app) sortEntries(list []entry) {
	switch {
	case a.section == sectionLibrary && a.sortBy == sortAZ:
		sort.SliceStable(list, func(i, j int) bool {
			return strings.ToLower(list[i].name()) < strings.ToLower(list[j].name())
		})
	case a.section == sectionLibrary && a.sortBy == sortAdded:
		// A game's install order is the order the scan found it in, which is
		// already stable; nothing better is recorded yet.
	case a.section == sectionLibrary && a.sortBy == sortRecent:
		sort.SliceStable(list, func(i, j int) bool {
			return recentFirst(a, list[i], list[j])
		})
	default:
		// Most played first, then the never-played ones by name. Ties fall back
		// to the most recent session, so two games with equal hours still read
		// in a sensible order.
		sort.SliceStable(list, func(i, j int) bool {
			pi, pj := a.playedOf(list[i]), a.playedOf(list[j])
			if pi != pj {
				return pi > pj
			}
			return recentFirst(a, list[i], list[j])
		})
	}
}

// recentFirst orders two games by when they were last played, with a game never
// played after those that were, and by name when neither has a session.
func recentFirst(a *app, x, y entry) bool {
	li, lj := a.lastOf(x), a.lastOf(y)
	if li.IsZero() != lj.IsZero() {
		return lj.IsZero()
	}
	if !li.IsZero() {
		return li.After(lj)
	}
	return strings.ToLower(x.name()) < strings.ToLower(y.name())
}

// matchesQuery reports whether a game's name contains the search.
func matchesQuery(e entry, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	return strings.Contains(strings.ToLower(e.name()), q)
}

// selectGame moves the focus to a game.
func (a *app) selectGame(i int) {
	if i < 0 || i >= len(a.entries) {
		return
	}
	a.selected = i
}

// focused is the game in focus, or nil when the library shows nothing.
func (a *app) focused() *entry {
	if len(a.entries) == 0 {
		return nil
	}
	if a.selected < 0 || a.selected >= len(a.entries) {
		return nil
	}
	return &a.entries[a.selected]
}
