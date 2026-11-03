package app

import (

	"github.com/egoist/mygo/ui"

	steamstore "github.com/yafit/nova/internal/steam"
)

// legacyCDN serves a game's artwork by Steam app id.
const legacyCDN = "https://cdn.cloudflare.steamstatic.com/steam/apps/"

// portraitFiles are the covers a tile can use, best first. They are 600x900
// and 300x450: both 2:3, which is what a tile is, so nothing is cropped.
var portraitFiles = []string{"library_600x900_2x.jpg", "library_600x900.jpg"}

// resolve looks up a game's Steam identity and store page, once, in the
// background. Everything that needs artwork or a description waits for it.
func (a *app) resolve(e entry) {
	slug := e.slug()

	a.mu.Lock()
	if a.resolved == nil {
		a.resolved = map[string]bool{}
	}
	if a.resolved[slug] {
		a.mu.Unlock()
		return
	}
	a.resolved[slug] = true
	a.mu.Unlock()

	go func() {
		id := e.inst.AppID
		if id == "" {
			for _, name := range []string{e.inst.Name, e.name()} {
				if found, ok := a.store.Search(name); ok {
					id = found
					break
				}
			}
		}

		var details *steamstore.Details
		if id != "" {
			details, _ = a.store.Get(id)
		}

		a.update(func() {
			a.mu.Lock()
			if a.appIDs == nil {
				a.appIDs = map[string]string{}
			}
			if a.storeData == nil {
				a.storeData = map[string]*steamstore.Details{}
			}
			a.appIDs[slug] = id
			if details != nil {
				a.storeData[slug] = details
			}
			a.mu.Unlock()
		})
	}()
}

// storeFor is a game's store page, or nil until it has resolved.
func (a *app) storeFor(e entry) *steamstore.Details {
	slug := e.slug()

	a.mu.Lock()
	d, ok := a.storeData[slug]
	_, looked := a.resolved[slug]
	a.mu.Unlock()

	if ok {
		return d
	}
	if !looked {
		a.resolve(e)
	}
	return nil
}

// coverURLs are the addresses a game's cover can be fetched from, best first.
// Every one of them is 2:3, matching the tile, so the art fills it whole:
//
//  1. Steam's own library capsule, which its client caches per installed
//     game: 300x450, exactly the tile's ratio, and no network.
//  2. The portrait files on Steam's CDN, 600x900 and 300x450.
//  3. The store's header, for the apps whose portrait art Steam never
//     published.
func (a *app) coverURLs(e entry) []string {
	var urls []string

	// The install's own app id is authoritative and known immediately; the
	// resolved one only fills in for a game whose id Steam had to be asked for
	// (a repack found by name). Reading only the resolved map meant a Steam
	// game showed generated art until the background lookup answered.
	id := e.inst.AppID

	a.mu.Lock()
	if id == "" {
		id = a.appIDs[e.slug()]
	}
	a.mu.Unlock()

	if id != "" {
		if path, ok := steamstore.Capsule(id); ok {
			urls = append(urls, path)
		}
		for _, name := range portraitFiles {
			urls = append(urls, legacyCDN+id+"/"+name)
		}
	}

	if d := a.storeFor(e); d != nil && d.HeaderImage != "" {
		urls = append(urls, d.HeaderImage)
	}
	return urls
}

// backdropURLs are the sources for the detail page's full-bleed artwork, best
// first. Steam's `page_bg_raw.jpg` is 1438x810 (16:9), which is the shape the
// page wants: it fills the window and survives the design's 1.12 scale without
// being stretched.
//
// The portrait cover is a last resort, and the generated art after it. It is
// 2:3 and the page is wide, so using it fills but crops hard — which is why it
// comes only after the store's own wide art.
func (a *app) backdropURLs(e entry) []string {
	var urls []string

	if d := a.storeFor(e); d != nil && d.Background != "" {
		urls = append(urls, d.Background)
	}

	// The install's own app id, then the resolved one, as `coverURLs` does.
	id := e.inst.AppID
	a.mu.Lock()
	if id == "" {
		id = a.appIDs[e.slug()]
	}
	a.mu.Unlock()
	if id != "" {
		// The store's own page background, by app id, for a store page that has
		// not arrived but whose id is known.
		urls = append(urls, legacyCDN+id+"/page_bg_raw.jpg")
	}

	// A game with neither yet still gets painted: the cover loads from Steam's
	// local cache by app id, and failing that the page draws its generated art.
	return append(urls, a.coverURLs(e)...)
}

// backdropOf returns the loaded full-bleed artwork, or nil while it loads.
func (a *app) backdropOf(e entry) *ui.Bitmap {
	if a.covers == nil {
		return nil
	}
	return a.covers.get(a.backdropURLs(e))
}

// coverOf returns a game's loaded cover, for the places that need the bitmap
// itself rather than a painted box.
func (a *app) coverOf(e entry) *ui.Bitmap {
	if a.covers == nil {
		return nil
	}
	return a.covers.get(a.coverURLs(e))
}

// accentFor is the window's accent for a game: sampled from its cover when
// that has loaded, and the theme's own white otherwise.
func (a *app) accentFor(e entry) ui.Color {
	if a.covers == nil {
		return white
	}
	if c, ok := a.covers.accent(a.coverURLs(e)); ok {
		return c
	}
	return white
}

// appIDString is a game's resolved Steam app id, or "" while it is pending.
func (a *app) appIDString(e entry) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.appIDs[e.slug()]
}

