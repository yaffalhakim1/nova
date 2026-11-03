# task.md — queue

Everything the user asks goes here first and is done in order, so work in
flight is never disturbed.

## In flight (finishing now)

- [x] **Carousel crops the 5th tile.** Fixed: the row was clipping at its own
      content edge. It is now given the window's full width so the clip happens
      at the window edge.
- [x] **"Ready to play" → playtime.** The meta line always shows
      `genre · time played · last played`, with "Not played yet" for no history.
- [x] **Carousel animation.** Proven by `TestCarouselTrackSlides`: the frame
      changes 120ms into the 550ms slide and settles after it.
- [x] **Owned games in the capsule**, ordered by playtime, marked
      "Not installed" (which replaces the New badge rather than stacking on it).

## Queue

1. [x] **D1/D2. Detail page game hub** — done, then reviewed and corrected by me.
       Adds `hub.go`; the full-bleed art now uses Steam's 16:9 `page_bg_raw`.
2. [~] **D3. Trophies card — real achievements.** Subagent `ses_e60e04442ffeitdbX03L4RmtbZ`
       is writing `steam/achievements.go` + `hub_achievements.go` right now.
       The build is red *only* because its files are half-written; my own
       changes compile clean. Verify when it reports.
3. [~] **L1. Library grid page — written by me, awaiting a build+render.**
       New `internal/app/library.go`: heading with count and sort select, genre
       chips, `ui.GridView` auto-fill grid of 2:3 cover cards, add card last.
       State added: `sortBy`, `sortChoice`, `genreFilter`, `lastGenre`.
       `filtered` now honours the genre, and `view()` routes the Library tab to
       `libraryPage` instead of rendering the home page.
       **Blocks on the subagent finishing**, because both touch the same build.
4. [ ] **Settings overlay** (`S`, gear) — toggles for sound, particles, hints.
5. [ ] **Sessions still 0 for Steam-imported games** — Steam doesn't count
       sessions. Decide: hide the stat when zero, or derive it.

## Achievements — verified API facts

Checked live against the account on this machine, 2026-11-03:

```
GET https://api.steampowered.com/ISteamUserStats/GetPlayerAchievements/v1/
    ?key=<key>&steamid=<id>&appid=292030&l=english
→ {"playerstats":{"gameName":"...","achievements":[
     {"apiname":"LILAC","achieved":1,"unlocktime":1771980824,
      "name":"Lilac and Gooseberries","description":"..."}]}}
```
- 78 achievements for The Witcher 3, **36 achieved**.
- `description` is empty for some (hidden achievements), never missing keys.
- **Steam has no bronze/silver/gold/platinum.** The design's tiers are not in
  the API. Rarity is, though:

```
GET https://api.steampowered.com/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v2/?gameid=292030
→ {"achievementpercentages":{"achievements":[{"name":"LILAC","percent":60.3}]}}
```
- **No key needed** for the rarity call. 78 entries, matching by `name`.
- So tiers can be derived from real rarity: <5% platinum-like, <15% gold,
  <50% silver, else bronze. That is honest — the tier means "how rare", not a
  PlayStation trophy type. Say so in the code.

## Detail page — data gaps to solve honestly

The v2 detail page is a card strip, and three of its six cards have no data
behind them in this app:

| Card | Data available | Honest fallback |
|---|---|---|
| Trophies | none | "Not tracked for this game." |
| Activity | `stat.Played`, `stat.Sessions` | real numbers; completion bar has no meaning |
| Recent sessions | **no per-session log exists** | "No sessions recorded." |
| Game info | Steam `developers`, `releaseDate`, `Dir` | Version/Install size unknown |
| Screenshots | `storeFor(e).Screenshots` | generated stills |
| About | store description / manual | real |

`gameStats` holds only Played/Sessions/Last — there is no list of sessions, so
"Recent sessions" cannot be filled without a new field. Do not invent one.

## Done this session

- [x] H1 header: Home / Favorites / Library + gear + controller status.
- [x] H3 focus block: blurb, Play + Details.
- [x] H4 meta wording; H5 Favorites in stats; H6 hint bar; H8 Enter plays.
- [x] Detail page two-column About/Screenshots (760px breakpoint).
- [x] Last played: one function, one wording.
- [x] Home → detail black flash fixed.
- [x] Manual add by folder picker; Steam playtime import; owned-games API.

## Found and fixed

- ⚠ `stats.get` swallowed writes on a nil map; `record` had a value receiver.
  A finished session could vanish. Fixed with two regression tests.
