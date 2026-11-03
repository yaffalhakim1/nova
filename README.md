# Nova

A native Windows **game launcher**. It finds the games already installed on your
machine — the ones Steam has manifests for, the ones in a folder of your own, and
the ones you add by hand — shows each with its real cover art and store
description, and starts it.

![The home screen](docs/screenshots/home.png)

It is a launcher, not a store: nothing is bought, nothing is downloaded, and no
account is required. It never mods, patches or injects into a game — it lists
what you have and runs it.

Built with [MyGo](https://github.com/egoist/mygo): native UI, no web view, no
bundled runtime.

## What it does

- **Finds your games.** Steam libraries are read from their app manifests, and
  any folder you add is scanned for game executables. Steam's own redistributables
  and tools are skipped.
- **Adds the one game a scan misses.** Pick a repack's own folder and give it a
  name, a genre, a description and a cover colour. Its executable is stored
  relative to that folder, so moving the folder does not break it.
- **Shows the real thing.** Each game gets the cover Steam ships, its store
  description, its screenshots and its wide store artwork, fetched once and
  cached on disk.
- **Takes the game's colour.** The accent, the focus ring and the backdrop glows
  are sampled from the selected game's cover, so the window changes with what you
  pick. Changing game cross-fades the backdrop over one second.
- **Remembers what you played.** Time played and session counts are kept per game.
  Play time is also imported from the Steam account Steam used last, so a fresh
  install shows real hours instead of an empty history — merged as `max`, never
  added, so nothing is ever double-counted.
- **Shows your achievements.** With a Steam Web API key, the full achievement list
  for a game, with each one's real rarity and unlock date.
- **Starts the game.** One click, from the game's own folder, with no console
  window.

## Screenshots

| Library | Game page |
|---|---|
| ![Library](docs/screenshots/library.png) | ![Game page](docs/screenshots/detail.png) |

Every game is a card you can open; the game page is a strip of cards — trophies,
activity, game info, screenshots and the store description — over the game's own
full-bleed artwork.

![Achievements](docs/screenshots/trophies.png)

## Running it

```
go build ./...
go run .
```

Develop with live reload:

```
go tool mygo dev
```

## Requirements

- **Windows 10 or newer.**
- **Go 1.27 or newer.**
- **The MyGo checkout.** `go.mod` points at a sibling folder:

  ```
  replace github.com/egoist/mygo => ../mygo
  ```

  Clone [`egoist/mygo`](https://github.com/egoist/mygo) next to this repository
  and the build works. Point the `replace` at wherever you keep it.
- **Optionally, Steam.** Without it the launcher reads only the folders you add.
  The achievement list additionally needs a free
  [Steam Web API key](https://steamcommunity.com/dev/apikey).

## Where your data lives

Everything this app writes goes to `%APPDATA%\nova\`:

| File | Holds |
|---|---|
| `library.json` | the extra folders to scan, the games added by hand, the games you removed |
| `stats.json` | time played and sessions, per game |
| `covers\` | the cover art, cached |
| `store\` | store descriptions, artwork and achievements, cached |

Steam's own files are only ever **read**, never written: `config\loginusers.vdf`
for the accounts, and `userdata\<id>\config\localconfig.vdf` for play time.

The Steam API key is not stored in that folder. It goes to the system credential
store — Windows Credential Manager, the macOS Keychain, or Secret Service on Linux
— through `mygo.Secrets`, and is only ever sent to Steam.

## Keyboard

| Key | Does |
|---|---|
| ← → | choose a game |
| Enter | play it |
| ↓ | open its page |
| Esc | go back |
| / | search |
| F | favorite |
| N | add a game (pick its folder) |
| S | settings |
| 1 2 3 | Home / Favorites / Library |

## Data model

`library.json` holds the folders to scan and the games added by hand:

```json
{
  "folders": ["Y:\\games"],
  "manual": [
    {
      "dir": "Y:\\games\\Hollow Grove",
      "exe": "Binaries\\Win64\\HollowGrove.exe",
      "name": "Hollow Grove",
      "genre": "Exploration",
      "description": "A quiet forest that rearranges itself when you look away.",
      "hue": 150
    }
  ]
}
```

`exe` is always relative to `dir`. An entry whose folder or executable has gone is
skipped rather than shown as a dead tile.

## How it works

- **`internal/library`** finds installs: Steam manifests plus a folder scan plus
  the games added by hand. An install is a manifest (or a folder) plus an
  executable; a Steam tool or a redistributable is not a game.
- **`internal/steam`** reads Steam. The store page, the account's play time from
  its own config files, the owned-games list and the achievements all live here,
  each cached on disk with an expiry.
- **`internal/app`** is the whole interface. `home.go` is the carousel, `hub.go`
  the game page, `library.go` the grid, `art.go` the generated artwork, and
  `theme.go` the palette.

## Known limits

- **Steam folders without `appmanifest_*.acf` are invisible.** No manifest means no
  app id, and the launcher deliberately does not guess. Add the folder by hand
  instead.
- **Games that need the Steam client running must start it themselves.** The fix
  (not implemented) is `steam://rungameid/<appid>` as a fallback when a launch
  fails.
- **Tiles are 2:3, never square.** Steam ships no square asset; its portrait
  covers are exactly 2:3, so the tile matches the art and nothing is cropped.
- **Achievement tiers come from rarity, not from Steam.** Steam has no
  bronze/silver/gold/platinum — that is a PlayStation concept. A tier here means
  "how few players hold this", using Steam's published percentages.
- **Some store pages open with marketing** before they describe the game. Nova
  strips the common openers, but a page whose award line runs into its
  description with no full stop keeps that line.

## License

Personal project. The artwork, names and descriptions it displays belong to the
games they describe, and are read from Steam's public store API and your own
installation.
