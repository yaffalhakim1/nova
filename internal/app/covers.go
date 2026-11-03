package app

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
)

// covers loads and keeps the games' real artwork, cached on disk. The
// generated art stands in until a cover arrives, and for games with no Steam
// presence at all.
type covers struct {
	dir   string
	http  *http.Client
	mu    sync.Mutex
	byURL map[string]*ui.Bitmap
	// accents is the colour each cover lends the window, sampled as it decodes.
	accents map[string]ui.Color
	busy    map[string]bool
	fails   map[string]bool
	// ready is called when a cover arrives, to ask for a frame that shows it.
	ready func()
}

// newCovers returns a cover loader caching under dir.
func newCovers(dir string) *covers {
	c := &covers{
		dir:     filepath.Join(dir, "covers"),
		http:    &http.Client{Timeout: 30 * time.Second},
		byURL:   map[string]*ui.Bitmap{},
		accents: map[string]ui.Color{},
		busy:    map[string]bool{},
		fails:   map[string]bool{},
	}
	_ = os.MkdirAll(c.dir, 0o755)
	return c
}

// get returns a cover if it is loaded, starting its load on a miss. urls are
// tried in order, so a game whose portrait is missing still gets its header.
func (c *covers) get(urls []string) *ui.Bitmap {
	if c == nil || len(urls) == 0 {
		return nil
	}
	key := urls[0]

	c.mu.Lock()
	if b, ok := c.byURL[key]; ok {
		c.mu.Unlock()
		return b
	}
	start := !c.busy[key] && !c.fails[key]
	if start {
		c.busy[key] = true
	}
	c.mu.Unlock()

	if start {
		go c.load(key, urls)
	}
	return nil
}

// load reads a cover from disk, or downloads the first URL that answers.
// load reads a cover from disk, or downloads the first URL that answers. A
// failure is retried once, because a burst of concurrent glyph loads can make
// one of them give up on a transient error and would otherwise leave that game
// with generated art for the life of the process.
func (c *covers) load(key string, urls []string) {
	b, img, raw, err := c.fromDisk(key)
	if err != nil {
		b, img, raw, err = c.download(urls)
	}
	if err != nil {
		// One retry, after the burst has passed.
		time.Sleep(150 * time.Millisecond)
		b, img, raw, err = c.download(urls)
	}

	c.mu.Lock()
	c.busy[key] = false
	if err != nil {
		c.fails[key] = true
	} else {
		c.byURL[key] = b
		if img != nil {
			c.accents[key] = accentOf(img)
		}
	}
	ready := c.ready
	c.mu.Unlock()

	if err == nil && raw != nil {
		_ = os.WriteFile(c.path(key), raw, 0o644)
	}
	if err == nil && ready != nil {
		ready()
	}
}

// path is where one cover is cached, named after its key.
func (c *covers) path(key string) string {
	return filepath.Join(c.dir, sanitizeName(hashKey(key))+".img")
}

// fromDisk reads a cached cover.
func (c *covers) fromDisk(key string) (*ui.Bitmap, image.Image, []byte, error) {
	raw, err := os.ReadFile(c.path(key))
	if err != nil {
		return nil, nil, nil, err
	}
	b, img, err := decodeCover(raw)
	return b, img, raw, err
}

// download fetches the first URL that answers and caches it.
func (c *covers) download(urls []string) (*ui.Bitmap, image.Image, []byte, error) {
	var lastErr error
	for _, url := range urls {
		raw, err := c.fetch(url)
		if err != nil {
			lastErr = err
			continue
		}
		b, img, err := decodeCover(raw)
		if err != nil {
			lastErr = err
			continue
		}
		return b, img, raw, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no cover")
	}
	return nil, nil, nil, lastErr
}

// decodeCover decodes the bytes twice: once for the GPU, and once as a plain
// image to sample the game's colour from.
func decodeCover(raw []byte) (*ui.Bitmap, image.Image, error) {
	b, err := ui.DecodeBitmap(raw)
	if err != nil {
		return nil, nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return b, nil, nil
	}
	return b, img, nil
}

// accent returns the colour a cover lends the window, by the same key get took.
func (c *covers) accent(urls []string) (ui.Color, bool) {
	if c == nil || len(urls) == 0 {
		return white, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	col, ok := c.accents[urls[0]]
	return col, ok
}

// fetch reads one source's bytes: a local file Steam already cached, or a URL.
func (c *covers) fetch(src string) ([]byte, error) {
	// Steam's own cache is a folder of files, so a source that names an
	// existing file needs no network at all.
	if info, err := os.Stat(src); err == nil && !info.IsDir() {
		return os.ReadFile(src)
	}

	res, err := c.http.Get(src)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cover %s: HTTP %d", src, res.StatusCode)
	}
	return io.ReadAll(io.LimitReader(res.Body, 8<<20))
}

// sanitizeName keeps a string safe as a file name.
func sanitizeName(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_':
			b = append(b, c)
		default:
			b = append(b, '_')
		}
	}
	if len(b) == 0 {
		return "cover"
	}
	return string(b)
}

// hashKey makes a readable, stable file name out of a URL.
func hashKey(s string) string {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return fmt.Sprintf("%08x", h)
}
