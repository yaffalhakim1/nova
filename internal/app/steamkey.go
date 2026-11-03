package app

import (
	"strings"

	"github.com/egoist/mygo/ui"

	steamstore "github.com/yafit/nova/internal/steam"
)

// steamKeyOverlay asks for the Steam Web API key that unlocks the owned-games
// list.
//
// The key is the one thing a Steam account cannot give this app from its own
// files, so it is optional: without it the library still lists what is
// installed, and only "Your Steam library" is missing.
func (a *app) steamKeyOverlay(c *ui.Context) {
	ui.Modal(c, &a.steamKeyOpen, func() {
		if !a.steamKeyOpen {
			a.overlay = overlayNone
			return
		}

		ui.Text(c, "Your Steam library").FontSize(32).FontWeight(200).TextColor(ink)
		ui.Text(c, "A Steam Web API key lets the launcher list every game you own, "+
			"including the ones not installed. Reading it needs nothing else, and it is optional.").
			FontSize(14).TextColor(inkDim)

		a.keyShownRow(c)
		a.keyField(c)
		a.keyPrivacyNote(c)

		if a.keyProblem != "" {
			ui.Text(c, a.keyProblem).FontSize(13).TextColor(c.Theme().Danger)
		}
		if a.ownedProblem != "" {
			ui.Text(c, a.ownedProblem).FontSize(13).TextColor(c.Theme().Warning)
		}

		a.keyActions(c)
	})
}

// keyShownRow says whether a key is already stored, and never shows it.
func (a *app) keyShownRow(c *ui.Context) {
	if !a.keyStored {
		return
	}
	ui.Text(c, "A key is stored: "+redactedKey(a.steamKey)).
		FontSize(13).
		TextColor(c.Theme().Success)
}

// keyField is the input, with the link that explains where a key comes from.
func (a *app) keyField(c *ui.Context) {
	ui.TextInput(c, &a.keyInput).
		Placeholder("32 characters, from steamcommunity.com/dev/apikey").
		AutoFocus().
		Label("Steam API key")

	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "Get a key").
			FontSize(13).
			TextColor(inkDim).
			OnClick(func() { a.openKeyPage() })
		ui.Text(c, "·").FontSize(13).TextColor(inkFaint)
		ui.Text(c, "Your profile's game details must be public").
			FontSize(13).
			TextColor(inkFaint)
	})
}

// keyPrivacyNote is the promise the user is trusting: where the key goes.
func (a *app) keyPrivacyNote(c *ui.Context) {
	ui.Text(c, "The key is stored only on this PC, in Windows Credential Manager. "+
		"It is never written to a file, never sent anywhere but Steam, and never shown on screen.").
		FontSize(13).
		TextColor(inkFaint)
}

// keyActions are the form's buttons.
func (a *app) keyActions(c *ui.Context) {
	ui.Row(c).Gap(12).Justify(ui.End).Children(func() {
		primaryButton(c, "Save key", func() { a.saveKey() })
		if a.keyStored {
			ui.Button(c, "Forget key").Radius(999).Padding(13, 28).OnClick(func() {
				a.forgetKey()
			})
		}
		ui.Button(c, "Close").Radius(999).Padding(13, 28).OnClick(func() {
			a.closeSteamKey()
		})
	})
}

// openKeyPage opens Steam's own page for issuing a key, so the user does not
// have to find it.
func (a *app) openKeyPage() {
	_ = openURL("https://steamcommunity.com/dev/apikey")
}

// redactedKey is how a stored key is described. The window never holds the rest.
func redactedKey(key string) string { return steamstore.RedactedKey(key) }

// saveKey checks the key's shape, then stores it in the credential store.
func (a *app) saveKey() {
	key := strings.TrimSpace(a.keyInput)
	if !steamstore.ValidKeyShape(key) {
		a.keyProblem = "That is not a Steam Web API key: it should be 32 letters and digits."
		return
	}

	if err := steamstore.SaveKey(key); err != nil {
		a.keyProblem = "Could not store the key: " + err.Error()
		return
	}

	a.steamKey = key
	a.keyStored = true
	a.keyInput = ""
	a.keyProblem = ""
	a.say("Steam library", "key stored on this PC")

	a.loadOwnedGames()
}

// forgetKey removes the key and the library it unlocked.
func (a *app) forgetKey() {
	if err := steamstore.ForgetKey(); err != nil {
		a.keyProblem = "Could not remove the key: " + err.Error()
		return
	}

	a.steamKey = ""
	a.keyStored = false
	a.keyInput = ""
	a.keyProblem = ""
	a.owned = nil
	a.ownedProblem = ""
	a.lastKey = -1
	a.refresh()
	a.say("Steam library", "key forgotten")
}

// closeSteamKey shuts the overlay.
func (a *app) closeSteamKey() {
	a.steamKeyOpen = false
	a.overlay = overlayNone
	a.keyInput = ""
	a.keyProblem = ""
}

// openSteamKey shows the overlay on a clean field.
func (a *app) openSteamKey() {
	a.keyInput = ""
	a.keyProblem = ""
	a.steamKeyOpen = true
	a.overlay = overlaySteamKey
}
