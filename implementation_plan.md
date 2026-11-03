# implementation_plan.md — manual game add, Steam playtime, owned games later

## Status

- **Phase 1 — manual game add: DONE.** Folder picker, relative exe, genre,
  description, cover hue. Rendered and verified.
- **Phase 2 — Steam playtime import: DONE.** Real files on this machine read
  correctly (Cyberpunk 2077: 130h 33m, last played 3 days ago), merged by max.
- **Phase 3 — owned-games API: not built.** The seam is in place: `mergeSteam`
  takes a `map[string]steam.Entry` and does not care where it came from.

Decisions taken (from the user):
1. **Folder picker**, not a typed path. A bare `.exe` is *not* accepted: a game
   must have a folder to be its working directory.
2. **Merge policy 2 — max of the two** for Steam playtime.
3. **Most recently used Steam account** by default, no picker for now.
4. **Owned-games Web API — later.**

Nothing here mods, patches or injects into a game. It launches folders.

---

## Verified API facts (do not re-derive)

| Need | Verified |
|---|---|
| Folder picker | `mygo.Dialog.Open(mygo.OpenDialogOptions{Directory: true})` → `[]string`; **blocks**, so it must run on its own goroutine, then hand the result back with `a.update`. `dialog.go:35`, `docs/native.md:7-13`. |
| Future API key storage | `mygo.Secrets.Set/Get` → Windows Credential Manager, 2560-byte limit. `docs/native.md:496-515`. Not used in this phase. |
| Steam playtime | `userdata\<accountID>\config\localconfig.vdf`, block `Software → Valve → Steam → apps → <appid>` with sibling keys `Playtime` (minutes) and `LastPlayed` (unix seconds). Verified real values on this machine, e.g. `"377"`. |
| Steam accounts | `config\loginusers.vdf` → `users → <steamID64>{ AccountName, PersonaName, timestamp }`. Most recent = greatest `timestamp`. Verified: `Leetcode Enjoyer` (1793612794) and `CBS Khazan \| 2` (1778641173). |
| `<accountID>` | Steam's folder id is the **low 32 bits of the SteamID64**: `76561198000000000` → `384463082`. Both were cross-checked against the folders that exist. |
| VDF shape | The existing `readVDF` does not *stop* at a nested block: it skips `{` lines and keeps flattening every `"key" "value"` pair at any depth into one map, last wins. Harmless for `appmanifest` (unique top-level fields), but unusable for `localconfig.vdf`, where every app has its own `Playtime`. A depth-aware reader is required. |
| Steam **install** root ≠ library root | `loginusers.vdf` and `userdata\` live only under the Steam install (`C:\Program Files (x86)\Steam`), **not** under a library folder such as `Y:\SteamLibrary`, which is all `library.SteamRoots()` promises. A dedicated install-root lookup is needed. |

---

## Phase 1 — Manual game add (folder picker)

### Why
A game Steam does not recognise gets generated art and the line "Found in a
games folder". There is no way to name it, describe it, or correct a wrong
`findExe`. This is the bare-repack-tile problem seen on camera.

### UI
Replace the **Add a games folder** overlay with **Add a game**:
- "Choose folder…" → `mygo.Dialog.Open(Directory: true)`.
- Picked folder **is the game's own folder**; `findExe` runs inside it and
  pre-fills the detected exe name, which stays editable as a **relative** path.
- Name (pre-filled from folder), Genre, Description, Cover hue — the design's
  `#formAdd` fields without the ones we can infer.
- Add / Cancel.

Kept: the existing whole-folder scan remains available, so nothing is lost.

### Data
`library.Config` gains a list, so manual games survive a rescan:

```go
type Manual struct {
    Dir         string `json:"dir"`
    Exe         string `json:"exe,omitempty"`   // relative to Dir
    Name        string `json:"name"`
    Genre       string `json:"genre,omitempty"`
    Description string `json:"description,omitempty"`
    Hue         int    `json:"hue,omitempty"`   // 0 means "derive from slug"
}
```

`Detect(Roots)` gains `Custom` games from this list and returns them with
`Source: Manual`, so the rest of the app needs no special case. A manual entry
whose folder or exe has gone is **skipped**, not an error.

### Seams
- `library.Roots` gains `Manual map[string]Manual` (by dir) so `Detect` stays a
  pure function of its input and remains testable.
- `app.accent`/`artOf` need a hue override for manual games. Add
  `entry.hue()` returning `Manual.Hue` when set, else `artOf(slug).hue`.
- `describe()` returns `Manual.Description` when set, before the Steam lookup.
- `genreLine()` returns `Manual.Genre` when set.

### Files
| File | Change |
|---|---|
| `internal/library/config.go` | `Manual` struct + field, `Roots` carries it |
| `internal/library/detect.go` | merge manual entries, skip vanished, keep `Source: Manual` |
| `internal/library/manual.go` *(new)* | validate one picked folder: dir exists, exe resolves |
| `internal/app/overlays.go` | replace the folder overlay body with the form |
| `internal/app/addgame.go` *(new)* | picker goroutine, results, field state |
| `internal/app/art.go`, `store.go` | honour a hue override |
| `internal/app/detail.go` | `describe`/`genreLine` read manual metadata |

### Tests (before the code)
- A picked folder with an exe at `Binaries\Win64\X.exe` becomes one install,
  exe set, `Source: Manual`.
- A picked folder that is a **single game's folder**, not a library of games —
  the case the old scan would have missed.
- A manual entry whose folder was deleted disappears from the library and does
  not panic.
- Reload: config saved, reloaded, the manual game is still there.
- A repo-pack nested folder gets the right name (existing `gameName` rules).

---

## Phase 2 — Steam playtime import, max merge

### Why
nova records only what it launched. Steam already knows the real numbers, so a
fresh install shows an empty history for games played for years.

### Reader
New `internal/steam/local.go`, a real VDF reader:

- **Depth-tracking** reader: returns nested blocks, unlike `library.readVDF`.
  Keep it in `internal/steam` so `library`'s flat reader stays untouched.
- `Accounts() []Account` — from `loginusers.vdf`, newest `timestamp` first.
- `Playtime(accountID) map[string]Entry` — `apps` block → appid →
  `{Played time.Duration, Last time.Time}`.
- A malformed or absent file yields an empty result, never an error.

### Merge, per the decision
For each app id present in both:

```
Played  = max(stats.Played, steam.Played)
Last    = max(stats.Last,   steam.Last)
Sessions= stats.Sessions          // Steam has no session count
```

Max is idempotent, so re-importing every launch never inflates. nova's own
sessions after the import still add on top through `record()`.

Applied once at `load()`, for the most recently used account, and recorded in
`stats` with nothing extra saved. A **Steam-sourced** value can be marked so the
detail page can say where it came from later; not required in this phase.

### Files
| File | Change |
|---|---|
| `internal/steam/local.go` *(new)* | VDF blocks, accounts, playtime, install-root lookup |
| `internal/app/stats.go` | `mergeSteam(steam map[string]Entry)` applying max |
| `internal/app/app.go` | call it in `load()` after `loadStats` |

### Tests (before the code)
- Fixture `localconfig.vdf` with two apps → both mapped by app id.
- `Playtime 377` becomes `6h17m`, not 377 seconds or 377 hours.
- Max merge: local 41h + Steam 100h → 100h; then merge again → still 100h.
- Local 100h + Steam 41h → 100h (local wins when larger).
- Two accounts → the one with the greater `timestamp` is chosen.
- Missing `userdata` → empty map, app still loads.
- Account id derivation: `76561198000000000` → `384463082`.

---

## Phase 3 — Owned games (later, seam only)

Not built now. The seam that makes it cheap later: `mergeSteam` takes a
`map[string]Entry` and does not care whether it came from a file or an HTTP
call, so the Web API slots in behind it. The API key will go through
`mygo.Secrets` (Windows Credential Manager), never a JSON file in `%APPDATA%`.

Known constraints to record now, so they are not a surprise:
- `GetOwnedGames` needs a key **and** a public "game details" profile.
- Rate limits; own caching layer, same as covers.
- It reports lifetime minutes, not per-day; achievements are a second call.
- It would also list **uninstalled** games, which this launcher deliberately
  never shows today: that is a product decision, not a technical one.

---

## Risks

1. **`localconfig.vdf` layout drift.** Valve changes these files. The reader
   must key off block names, not indentation, and treat anything unexpected as
   empty. Mitigation: fixtures for the two shapes seen here.
2. **Double counting** is the real danger. Max merges are idempotent; do not
   "add" Steam time into local time.
3. **Manual exe as an absolute path** would break if the folder moves. Store it
   relative to `Dir` and re-resolve on load.
4. **Dialog blocks the UI thread** if called from the view. Must be a goroutine
   plus `a.update`, which also means the form must handle "user cancelled".
5. **A library folder is not an install.** `Y:\SteamLibrary` has `steamapps`
   but no `userdata`/`loginusers.vdf`; phase 2 must look for the install root
   separately or it silently finds no accounts on a machine whose games are on
   another drive.

## Out of scope
- Achievements, friends, wishlist, cloud saves.
- Any write to Steam's files. Read-only, always.
- A bare `.exe` with no folder (decided against).
- An account picker (decided against for now).

## Order of work
Phase 1 → Phase 2 → Phase 3 seam. Each phase: tests first, `go build`,
`go vet`, `go test`, then a PNG render at 1280×840 before it is called done.
