package app

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
)

// shotsName is the folder inside the cache that holds screenshots.
const shotsName = "screens"

// shotLimit is how many screenshots the detail page shows.
const shotLimit = 3

// shots loads and keeps a game's store screenshots, cached on disk.
type shots struct {
	dir   string
	http  *http.Client
	mu    sync.Mutex
	byURL map[string]*ui.Bitmap
	busy  map[string]bool
	fails map[string]bool
	// ready is called when a screenshot arrives, to ask for a frame.
	ready func()
}

// newShots returns a screenshot loader caching under dir.
func newShots(dir string) *shots {
	s := &shots{
		dir:   filepath.Join(dir, shotsName),
		http:  &http.Client{Timeout: 30 * time.Second},
		byURL: map[string]*ui.Bitmap{},
		busy:  map[string]bool{},
		fails: map[string]bool{},
	}
	_ = os.MkdirAll(s.dir, 0o755)
	return s
}

// get returns a screenshot if it is loaded, starting its load on a miss.
func (s *shots) get(url string) *ui.Bitmap {
	if s == nil || url == "" {
		return nil
	}

	s.mu.Lock()
	if b, ok := s.byURL[url]; ok {
		s.mu.Unlock()
		return b
	}
	start := !s.busy[url] && !s.fails[url]
	if start {
		s.busy[url] = true
	}
	s.mu.Unlock()

	if start {
		go s.load(url)
	}
	return nil
}

// load reads a screenshot from disk or downloads it.
func (s *shots) load(url string) {
	b, err := s.fromDisk(url)
	if err != nil {
		b, err = s.download(url)
	}

	s.mu.Lock()
	s.busy[url] = false
	if err != nil {
		s.fails[url] = true
	} else {
		s.byURL[url] = b
	}
	ready := s.ready
	s.mu.Unlock()

	if err == nil && ready != nil {
		ready()
	}
}

// path is where one screenshot is cached: its URL's last two parts, which are
// unique per shot.
func (s *shots) path(url string) string {
	clean := strings.Map(func(r rune) rune {
		if r == '?' || r == '&' || r == '=' || r == ':' || r == '/' {
			return '_'
		}
		return r
	}, url)
	if len(clean) > 120 {
		clean = clean[len(clean)-120:]
	}
	return filepath.Join(s.dir, clean+".jpg")
}

// fromDisk reads a cached screenshot.
func (s *shots) fromDisk(url string) (*ui.Bitmap, error) {
	raw, err := os.ReadFile(s.path(url))
	if err != nil {
		return nil, err
	}
	return ui.DecodeBitmap(raw)
}

// download fetches a screenshot and caches it.
func (s *shots) download(url string) (*ui.Bitmap, error) {
	res, err := s.http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, err
	}

	raw, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	b, err := ui.DecodeBitmap(raw)
	if err != nil {
		return nil, err
	}
	_ = os.WriteFile(s.path(url), raw, 0o644)
	return b, nil
}
