# AGENTS.md: nova

Read this before changing anything. It is the state of the project, not a
wishlist.

## What this is

A native Windows **game launcher**: it lists the games already installed on this
machine and starts them. It does not mod them, patch them, or touch their
memory. That is the whole product; anything that smells like a trainer does not
belong here.

Built on **MyGo** (`github.com/egoist/mygo`, `ui` package), used as a versioned
upstream module. Native UI only: no web view, no HTML, no CSS. There is no local
replacement; read the toolkit source in the Go module cache or at
`https://github.com/egoist/mygo`.

## Hard rules

- **The MyGo skill is always on.** Any UI question starts with `docs/ui/*.md`
  and `examples/gallery` in upstream MyGo at `https://github.com/egoist/mygo`.
  Read the docs for the version pinned in `go.mod`. Never guess an API, never
  hand-roll a widget that `ui` already ships.
- **No web assets.** `design/home-detail.html` is the design reference, not code
  to embed. Translate it into `ui` calls.
- **`go build ./...`, `go vet ./...`, `go test ./...` must all pass** before
  calling anything done.
- Every shell command carries an explicit timeout.
- No dead code, no TODO stubs, no panic on a user path.

## Architecture

```
main.go                      → app.Run()
internal/paths               the app's own data folder
internal/library             find installed games (Steam manifests, folders, manual adds)
internal/gse                 read local GSE emulator achievement saves
internal/backup              portable JSON export and import
internal/steam               Steam store data: description, screenshots, search
internal/app                 the whole interface
```

### Data flow

1. `paths.DataDir()` → `%APPDATA%\nova`, shared by the dev and release builds
   (only MyGo's window state is per-identifier).
2. `library.Detect` walks Steam manifests and custom folders, then the games
   added by hand. **Manual entries come first and win**, because they carry a
   name and executable the user chose. An install is a manifest plus a folder
   plus an executable; a folder with no executable, or a Steam tool, is not a
   game.
3. A game's store page and artwork come from Steam, keyed by app id: the
   manifest's id, or a store search by folder name for a repack.
4. Covers, store pages and screenshots are cached on disk. A failed cover fetch
   is retried once, then left on generated art rather than retried forever.

### Adding a game by hand

`app.addGameOverlay` → `chooseGameFolder` → `library.AddManual`.
- The **folder picker is the system dialog** (`mygo.Dialog.Open` with
  `Directory: true`). It blocks, so it runs on its own goroutine and returns
  through `a.update`. Never call it on the view's goroutine.
- The picked folder **is the game's own folder**, not a folder full of games.
  That is the whole difference from adding a games folder, which still exists
  under Settings.
- **`ManualGame.Exe` stays relative to `Dir`.** `ExeFor` refuses a path
  that climbs out with `..` or is absolute. A relative path survives the folder
  moving; an absolute one does not.
- Genre, description and cover hue are shown only when Steam has nothing, or
  in place of what the store would say. `coverArt` applies the hue.

### Steam play time

`app.importSteam` (`stats.go`) → `steam.DefaultAccount` → `steam.Playtime` →
`app.mergeSteam`.
- Read from **`userdata\<id>\config\localconfig.vdf`**, block
  `Software → Valve → Steam → apps → <appid>`, with `Playtime` in **minutes** and
  `LastPlayed` in unix seconds.
- **The account is the one Steam used last** (`loginusers.vdf` `timestamp`), and
  its folder id is the low 32 bits of SteamID64 (`76561198000000000` →
  `384463082`).
- **Merge is `max`, never sum.** Steam and nova count the same sessions; adding
  them double-counts. Max is also idempotent, so importing on every scan is free.
  Sessions stay nova's own, because Steam does not count them.
- **A Steam *install* root is not a library folder.** `InstallRoot` looks for
  `config\loginusers.vdf` or `userdata\`, which live only under the install
  (`C:\Program Files (x86)\Steam`), not under `Y:\SteamLibrary`.
- **Two traps in the VDF reader** (`internal/steam/local.go`), both found on real
  files and both pinned by tests:
  1. Steam stores JSON as an escaped string,
     `"Key" "{\"a\":1}"`. Braces inside a value are not block delimiters, and
     escaped quotes must not end the string.
  2. Empty values exist, as `"storepreferences" ""`. An empty token is a value,
     **not** end of input; treating it as the end silently drops the rest of the
     file, which is how a whole account's play time once read as zero.

### Launch path

`app.play(e)`:

1. `startGame(e)` asks Explorer to start `e.inst.Exe` with `cmd.Dir` set to the
   **game's own folder**. That folder rule is the one thing that matters:
   launching from the launcher's cwd is the classic "game starts then crashes".
   Explorer must start the game: a process created directly by Go is invisible
   to Discord's game detection, even with the same path and window title.
   Helper processes may use `CREATE_NO_WINDOW`; the game must not.
2. A goroutine finds the game's PID and watches it. The launcher stays usable,
   and closing the game does not close the launcher. The watch ends the play
   session, so the player need not press End session.
3. While a session runs, `checkpointPlay` writes elapsed time every ten seconds.
   A forced exit loses at most one checkpoint. Quitting records only the time
   since the last checkpoint, and counts one session.
4. Nothing is injected, attached or hooked. If a game needs Steam running, it
   starts Steam itself or fails visibly.

## Artwork and colour

- **Tiles are 2:3**, never square. Steam ships no square asset; its portrait
  covers are exactly 2:3, so the tile ratio matches and nothing is cropped.
  Changing the tile ratio re-introduces cropping.
- Cover sources, in order: Steam's own cached `library_capsule.jpg` (300x450,
  no network), the portrait files on the CDN (600x900, 300x450), then the store
  header.
- **The window takes the selected game's colour.** `accentOf` samples the cover
  as it decodes: saturated pixels only, greens weighted down, then lifted to
  read on the dark ground. The accent drives `Theme.Accent`, the focus ring and
  the backdrop glows, and the glows are centred on the selected capsule.
- **The detail page's full-bleed art is the store's wide background**, not the
  cover. `backdropURLs` asks for Steam's `page_bg_raw.jpg`, which is
  **1438x810 (16:9)**, the shape that page needs. The 2:3 cover is only a
  fallback, because filling a wide page with a portrait crops most of the game
  away. Tiles keep the portrait cover; the two never swap.
- Changing game **cross-fades** the backdrops over one second: the outgoing
  game's glows at `1-t`, the incoming at `t`, over a shared ground. Painting a
  second full backdrop painted opaque on top covers the first instead of fading.

## Interface map

| File | Holds |
|---|---|
| `app.go` | `app` struct, `Run`, `load`, `scanLibrary`, `refresh`, filters, `entry` |
| `home.go` | carousel stage, tiles, focus block, Jump back in, library stats |
| `detail.go` | detail page: the hub shell, `open`, `forget`, `describe`, play history |
| `hub.go` | the hub's cards: trophies, activity, sessions, info, screenshots, about |
| `overlays.go` | search, add-folder, shortcuts, hint bar, toasts |
| `addgame.go` | the "Add a game" form: folder picker, draft, commit |
| `playing.go` | the running-session overlay with its live timer |
| `play.go` | `play`, `quitPlay`, `watchGame`, `checkpointPlay`, session timing |
| `backup_ui.go` | export and import through the native file dialogs |
| `settings.go` | the settings panel that replaced the native menu bar |
| `chrome.go` | top bar, filter pills, icons, glass panel, buttons |
| `art.go` | generated artwork, backdrop, `paintGame`, `accentOf` |
| `covers.go` | real cover art: fetch, cache, decode, sample |
| `store.go` | Steam identity resolution, cover URLs, accent |
| `screens.go` | store screenshots |
| `backdrop.go` | the cross-fade |
| `particles.go` | the ambient drift |
| `sound.go` | the four synthesised UI tones |
| `stats.go` | play history: `%APPDATA%\nova\stats.json`, Steam merge |
| `fonts.go` | the embedded Hanken Grotesk variable font, registered at startup |
| `surfaces.go` | hover/press surfaces |
| `theme.go` | palette tokens, `theme()`, `hsl` |

## Known constraints

- **Steam folders without `appmanifest_*.acf` are invisible.** No manifest means
  no app id, and the project deliberately does not guess: only games detected as
  installed appear.
- Games that need the Steam client running must start it themselves. The fix, not
  implemented, is `steam://rungameid/<appid>` as a fallback when `startGame`
  fails.
- The description's "About" card shows six lines; `blurb` still cuts the store
  text at 380 characters first, which is not a measured line count, because
  MyGo's `MaxLines` truncates visually but cannot be asked whether it truncated.
- The hub's Completion and Recent sessions cards state that plainly: this app
  has no completion percentage and no per-session log (`gameStats` keeps only
  `Played`, `Sessions`, `Last`). Do not invent them. Trophies are real: Steam
  unlocks need a Web API key and an account; a repack's local GSE save file is
  read without either from `%APPDATA%\GSE Saves\<appid>\achievements.json`.
- MyGo toasts are bottom-centre; the design puts them top-centre.

## Testing

- `ui.NewTester(view, w, h)` renders headlessly: `Frame()`, `Advance(d)`,
  `Image()` for a PNG, `Find(text)` for a rect, `Click(label)`.
- `Find` returns **content** coordinates, not window ones, so it is not a
  scroll probe. Use `Texts()`/`HasText` for what is painted.
- A wheel **glides**: `Advance` after `Wheel`. `Scroll` applies at once.
- Grids and lists are virtualised, so only visible rows paint.
- `ui.Modal` needs a stable `*bool` (a field on `app`), never a fresh local: a
  per-frame pointer restarts its animation and the content never settles.
- **Tests must not start a real game.** `installsOf` builds folders that do not
  exist, which is enough for every view.
- Anything that writes a file must do it synchronously: a background write
  outlives `t.TempDir` and races its cleanup.

## The landing page

`docs/index.html` is a single static page that GitHub Pages serves from `docs/`.
Its hero plays `docs/media/tour.mp4`: a nine second silent loop of the real
launcher, made by `TestFilmTour` (`internal/app/film_test.go`). That test drives
the app through `ui.NewTester` and pipes the frames to ffmpeg. The page has no
build step and no framework.

The clip's poster is its own first frame, so nothing jumps when playback starts.
The clip loops while it is on screen and pauses when it is not. A visitor who
asks for reduced motion keeps the poster and the clip never plays.

Rebuild the clip when the interface changes:

```
$env:STEAM_API_KEY="..."; $env:STEAM_ID="..."
$env:NOVA_FILM="docs\media\tour.mp4"
go test ./internal/app -run TestFilmTour -v -timeout 900s
ffmpeg -y -i docs\media\tour.mp4 -frames:v 1 -q:v 3 docs\media\tour-poster.jpg
```

Without `NOVA_FILM` the test skips, so `go test ./...` never encodes video. The
achievements are real, so the clip needs the key and the account.

## Commands

```
go build ./... && go vet ./... && go test ./...
go tool mygo dev            # live reload; `r` rebuilds, `q` quits
python scripts/icons.py     # rebuild the app icons from the mark
```
