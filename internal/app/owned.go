package app

import (
	"context"
	"sort"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	steamstore "github.com/yafit/nova/internal/steam"
)

// ownedRow is one row of "Your Steam library": a game the account owns, with the
// installed one it corresponds to, when it is on this machine.
type ownedRow struct {
	steamstore.OwnedGame
	// entry is the installed game, or nil when the game is not on disk. A row
	// with no entry has no way to start, which is why it is never playable.
	entry *entry
}

// Installed reports whether the row's game is on this machine.
func (r ownedRow) Installed() bool { return r.entry != nil }

// slug identifies the row for a stable list.
func (r ownedRow) slug() string {
	if r.entry != nil {
		return r.entry.slug()
	}
	return "steam-" + r.AppID
}

// installURL opens a game's page in Steam so it can be installed. The launcher
// never installs anything itself.
func installURL(appID string) string { return "steam://install/" + appID }

// ownedRows joins the owned games with the installed ones: most played first,
// then by name, with the games that are not on disk placed after their peers.
func (a *app) ownedRows() []ownedRow {
	if len(a.owned) == 0 {
		return nil
	}

	byAppID := map[string]*entry{}
	for i := range a.entries {
		if id := a.entries[i].inst.AppID; id != "" {
			byAppID[id] = &a.entries[i]
		}
	}

	out := make([]ownedRow, 0, len(a.owned))
	for _, g := range a.owned {
		if g.AppID == "" {
			continue
		}
		out = append(out, ownedRow{OwnedGame: g, entry: byAppID[g.AppID]})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Played != out[j].Played {
			return out[i].Played > out[j].Played
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// ownedLimit is how many of the owned games the home screen lists. The rest are
// reachable through search, and listing hundreds of tiles is not useful.
const ownedLimit = 12

// steamLibrary is the second scope the home screen shows: the games the account
// owns that are not installed, so the launcher knows the whole library without
// ever pretending a game can start.
func (a *app) steamLibrary(c *ui.Context) {
	if !a.ownedMatchesAccount() {
		return
	}

	var missing []ownedRow
	for _, r := range a.ownedRows() {
		if !r.Installed() {
			missing = append(missing, r)
		}
	}
	if len(missing) == 0 {
		return
	}

	total := len(missing)
	shown := missing
	if len(shown) > ownedLimit {
		shown = shown[:ownedLimit]
	}

	ui.Column(c).Gap(14).Padding(0, 0, 44, 0).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(12).Children(func() {
			ui.Text(c, "Your Steam library").FontSize(15).FontWeight(500).TextColor(inkDim)
			// The count is the real one, even when only some are listed, so the
			// number never overstates what is on screen.
			ui.Text(c, itoa(total)+" not installed").FontSize(13).TextColor(inkFaint)
		})

		ui.Row(c).Gap(12).Wrap().Children(func() {
			for _, r := range shown {
				a.ownedCard(c, r)
			}
		})

		if total > len(shown) {
			ui.Text(c, "and "+itoa(total-len(shown))+" more, through search").
				FontSize(12).TextColor(inkFaint)
		}
	})
}

// ownedCard is one game that is owned but not installed: no cover, no Play
// button, and one honest action.
func (a *app) ownedCard(c *ui.Context, r ownedRow) {
	// MaxWidth keeps a short last row from stretching its cards across the whole
	// window, which is what Grow alone would do.
	card := ui.Column(c).
		Basis(210).
		Grow(1).
		MaxWidth(300).
		Gap(8).
		Padding(12).
		Radius(18)
	card = raisedSurface(card)

	card.Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).FillWidth().Children(func() {
			// No cover: the game is not installed, so there is no installed art
			// to show and a fake tile would read as "you have this".
			ui.Box(c).Size(44, 44).Radius(10).
				Background(ui.RGBA(255, 255, 255, 0.06)).
				Center().
				Children(func() {
					ui.Text(c, initialsOf(r.Name)).FontSize(15).TextColor(inkDim)
				})

			ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
				// Two lines, so a real game name reads whole rather than being
				// cut in the middle of a word.
				ui.Text(c, r.Name).FontSize(15).TextColor(ink).MaxLines(2)
				ui.Text(c, ownedDetail(r)).FontSize(12).TextColor(inkFaint).SingleLine()
			})
		})

		// The one action is a pill, so it reads as something to press rather
		// than a word that happens to be there.
		ui.Row(c).Padding(2, 0, 0, 0).Children(func() {
			b := ui.Button(c, "Install").Radius(999).Padding(6, 16).AlignSelf(ui.Start)
			b.OnClick(func() { a.installOwned(r) })
		})
	})
}

// installOwned opens the game's page in Steam, which handles installing it. The
// launcher never downloads or installs anything itself.
func (a *app) installOwned(r ownedRow) {
	a.playSound("tick")
	_ = mygo.Shell.OpenExternal(installURL(r.AppID))
	a.say(r.Name, "opening in Steam")
}

// ownedDetail is the line under an uninstalled game's name. The section heading
// already says these are not installed, so repeating it on every card would only
// crowd out the one fact that differs: how long it was played.
func ownedDetail(r ownedRow) string {
	if r.Played <= 0 {
		return "Never played"
	}
	return played(r.Played) + " played"
}

// initialsOf is one or two letters for a game with no artwork.
func initialsOf(name string) string {
	words := strings.Fields(name)
	if len(words) == 0 {
		return "?"
	}
	out := string([]rune(words[0])[0:1])
	if len(words) > 1 {
		out += string([]rune(words[1])[0:1])
	}
	return strings.ToUpper(out)
}

// steamAccount is the signed-in Steam account this app reads everything for:
// the play time from Steam's own files and the library from the Web API both
// use this one, so they can never describe two different accounts.
//
// Resolved once, in load, and kept: reading the file per frame would be a disk
// read for every repaint.
func (a *app) steamAccount() *steamstore.Account { return a.steam }

// resolveSteamAccount finds the account Steam used last, once at startup.
func (a *app) resolveSteamAccount() {
	a.steam = steamstore.DefaultAccount(steamstore.InstallRoot())
}

// steamAccountID is the account's SteamID64, or "" when Steam has none.
func (a *app) steamAccountID() string {
	if a.steam == nil {
		return ""
	}
	return a.steam.SteamID
}

// loadOwnedGames fetches the account's library in the background. Without a key,
// or without a Steam account, it does nothing: the app works either way.
func (a *app) loadOwnedGames() {
	key := a.steamKey
	if key == "" {
		return
	}

	steamID := a.steamAccountID()
	if steamID == "" {
		return
	}

	go func() {
		games, ok := a.store.OwnedGames(context.Background(), key, steamID)
		if !ok {
			a.update(func() {
				a.ownedProblem = "Steam did not return a library. Check that the key is right " +
					"and that your profile's game details are public."
			})
			return
		}
		a.update(func() {
			a.owned = games
			a.ownedSteamID = steamID
			a.ownedProblem = ""
			a.lastKey = -1
			a.refresh()
		})
	}()
}

// ownedMatchesAccount reports whether the loaded library belongs to the account
// Steam is currently signed in as. When it does not, the play time and the
// library would describe two different people, so the library is not shown.
func (a *app) ownedMatchesAccount() bool {
	if len(a.owned) == 0 || a.ownedSteamID == "" {
		return false
	}
	return a.ownedSteamID == a.steamAccountID()
}
