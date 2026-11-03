# Nova Design System

A game mods app styled after the PlayStation 5 home screen. Dark, calm, cinematic. The artwork of the selected game sets the mood of the whole screen.

## 1. Principles

1. **The game is the color.** The UI is neutral (white, glass, deep blue-black). All hue comes from the selected game's artwork.
2. **One focus at a time.** Exactly one tile is enlarged and glowing. Everything else dims.
3. **Home is a list, detail is a page.** Home shows only games. Mods, about and screenshots live on the game detail page.
4. **Light type, generous space.** Thin weights, large titles, wide margins.
5. **Motion explains focus.** Animate selection, opening and closing. Nothing moves for decoration except slow ambient particles.

## 2. Screens

### Home
```
[ All games | Recently modded | Favorites ]        [search][+][sound] battery 14:55 (avatar)

   [tile] [tile] [ TILE ] [tile] [tile] [ + Add ]      <- selected tile scaled 1.24, glow
                  Game title (64px, weight 200)
                  Genre · mods ready · last played
                  [ Open > ]  (star)
```
- Selected tile is left-anchored near 6vw; the track slides to keep it there.
- Click the selected tile, press Enter, or press Down to open the detail page.

### Game detail
```
< Back to games
[cover 210px]  Game title
               Genre · mods ready · last played
               [ > Play with 3 mods ] (star) (trash)

[ Mods panel               ] [ About panel           ]
  Infinite health      [on]    description
  Move speed  2x  ----o---     mods active / time / sessions
  ...                          screenshots
```
- Opens as a full-screen glass layer over the blurred game gradient.
- Esc or Back returns to the same selected tile.

### Overlays
Search, Add game, and the play session (launch bar, then running timer with Quit).

## 3. Color

Single dark theme. Neutrals are tinted blue, never pure grey.

| Token | Value | Use |
|---|---|---|
| `--deep` | `#05070f` | Page ground, base of every backdrop |
| `--ink` | `#f4f7ff` | Primary text |
| `--ink-dim` | `rgba(220,230,255,.62)` | Secondary text, inactive tabs |
| `--ink-faint` | `rgba(220,230,255,.34)` | Hints, placeholders, locked states |
| `--glass` | `rgba(255,255,255,.07)` | Chips, round buttons, inputs |
| `--glass-line` | `rgba(255,255,255,.14)` | 1px inset outlines on glass |
| `--accent` | `hsl(game.h2 85% 62%)` | Switches and glow, set per game |
| Success | `#6dffb0` | "Mods ready" dot |
| Favorite | `#ffd36a` | Star |

**Backdrop per game** (two layers cross-fading over 1s):
```
radial-gradient(90% 70% at 25% 20%, hsl(h 70% 38%), transparent 70%),
radial-gradient(80% 80% at 85% 70%, hsl(h2 65% 26%), transparent 70%),
hsl(h 55% 9%)
```
plus a bottom vignette: `radial-gradient(120% 80% at 50% 110%, rgba(5,7,15,.92), transparent 60%)`.

**Cover art** is generated per game: diagonal gradient `hsl(h 75% 42%)` to `hsl(h2 70% 16%)`, one of six shape patterns, and a bottom fade behind the title.

## 4. Typography

Family: **Hanken Grotesk** (fallback `SF Pro Display`, `Segoe UI`, system-ui). Light weights give the console feel.

| Role | Size | Weight |
|---|---|---|
| Game title | `clamp(34px, 5.4vw, 64px)` | 200 |
| Overlay title | 32px | 200 |
| Play timer | 44px, tabular numerals | 200 |
| Stat value | 26px | 200 |
| Body / description | 17px, line-height 1.6, max 60ch | 300 |
| Tabs, buttons, mod names | 16 to 18px | 400 to 500 |
| Meta line | 15px | 300 |
| Captions, hotkeys, hints | 12 to 13px | 300 to 400 |

Sentence case everywhere. No all-caps labels.

## 5. Layout and spacing

- Side gutter: `max(16px, safe-area-inset)` on both sides.
- Detail page content max width: 1180px, centered.
- Tile size: `clamp(120px, 15vw, 196px)`, square, gap 6px, side margin 12% of tile size so the scaled tile never overlaps neighbors.
- Panels: radius 24px, padding 26 x 28px, two columns (1.1fr / 1fr), one column under 760px.
- Radii: tiles 20px, panels 24px, cover 24px, inputs 14px, buttons and chips fully round.

## 6. Components

**Tile.** Square cover. Default: scale 1, opacity .62. Selected: scale 1.24, opacity 1, 3px white ring, white glow (`0 0 46px rgba(255,255,255,.38)`), deep shadow. Favorites show a small star badge top right. Last tile is "Add game" (glass, dashed none, large plus).

**Tabs.** Pills. Active: glass fill, 1px inset line, white text, backdrop blur 14px.

**Primary button (Play / Open).** White pill, dark text 18px, padding 15 x 44px. Hover and focus: scale 1.05 and white glow.

**Round button.** 52px circle, glass, icon stroke 1.5px. Pressed state (favorite) fills the icon `#ffd36a`.

**Mod row.** Name (17px) + hotkey chip `Alt+N`, description (13px dim), switch on the right. Hover and focus-within lighten the row. Slider mods show a range input and a live value (`2x`) beneath. A disabled mod dims its slider to 45%.

**Switch.** 52 x 30px. Off: white 18%. On: game accent fill and glow, knob slides 22px with a slight overshoot.

**Chips and meta.** "N mods ready" has a green glowing dot. "No mods on" is plain dim text.

**Overlays.** Full-screen `rgba(3,5,12,.72)` with 26px blur. Content boxes are max 560px.

**Toast.** Top center pill, glass dark, slides down 30px, visible 2.2s.

## 7. Motion

| Moment | Spec |
|---|---|
| Tile select | transform .45s `cubic-bezier(.22,1.2,.36,1)` (small spring) |
| Track slide | .55s `cubic-bezier(.22,1,.36,1)` |
| Backdrop change | opacity cross-fade 1s |
| Detail open | fade + scale from 1.03, .45s `cubic-bezier(.22,1,.36,1)` |
| Overlay open | fade .3s |
| Switch | .3s |
| Particles | 46 soft dots drifting up, 15 to 75% opacity |

`prefers-reduced-motion`: all transitions near zero, particles off.

## 8. Input

| Action | Keyboard | Gamepad |
|---|---|---|
| Move | Left / Right | D-pad, left stick |
| Open game | Enter, Down | X (cross) |
| Back / close | Esc | O (circle) |
| Favorite | F | Triangle |
| Search | `/` | none |
| Add game | N | none |
| Filter tabs | 1, 2, 3 | none |
| Toggle mod | Alt+1 to Alt+9 | none |

Mouse and touch: click a tile to select, click the selected tile to open, swipe the carousel. Focus is always visible: 2px white outline, 3px offset.

## 9. Sound

WebAudio blips, off by default until first interaction, toggle in the top bar. Navigate: short high tick. Select: two rising tones. Back: short falling tone. Mod on and off use the select and back sounds.

## 10. Data model

```json
{
  "id": "s1", "name": "Starfall Odyssey", "genre": "Space adventure",
  "h": 225, "h2": 285, "pat": 0,
  "desc": "...", "play": 148500, "sess": 38, "last": 1762050000000,
  "fav": true, "prog": 62,
  "mods": [
    { "k": "spd", "name": "Move speed", "ds": "...", "t": "s",
      "min": 1, "max": 5, "step": 0.5, "u": "x", "val": 2, "on": true },
    { "k": "hp", "name": "Infinite health", "ds": "...", "t": "t", "on": false }
  ]
}
```
`t` is `"t"` for toggle and `"s"` for slider. Stored in `localStorage` under `nova-launcher-v1`.

## 11. Accessibility

- Text on glass and the dark backdrop meets 4.5:1 for body and 3:1 for large text.
- State is never color alone: selected tile also scales and gets a ring, enabled mods also move the switch knob.
- Dialogs use `role="dialog"` and `aria-modal`, Esc closes them, focus returns to the opener.
- All inputs have labels or `aria-label`.

## 12. Notes for the real app

- This web build only saves mod state. Applying mods to a running game needs a native or Electron backend.
- When moving to Electron, keep tokens in CSS variables and keep the game data model above as the contract between UI and backend.
- Real cover art should replace the generated SVG, using the same square crop and bottom fade.
