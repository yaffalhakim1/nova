package app

import (
	"time"
)

// playingSession is the game this app started: when it began, and which game.
type playingSession struct {
	slug  string
	name  string
	start time.Time
	// lastCheck is the last time elapsed play time was written to disk.
	lastCheck time.Time
	// recorded is how much of this session is already in stats.json.
	recorded time.Duration
}

// playCheckpointEvery is how often an open session is written to disk. A
// forced exit then loses at most this much play time.
const playCheckpointEvery = 10 * time.Second

// play starts a game and opens the session window.
func (a *app) play(e entry) {
	if a.playing != nil {
		a.overlay = overlayPlaying
		return
	}
	if problem := launchProblem(e); problem != "" {
		a.say("Cannot start "+e.name(), problem)
		return
	}
	if err := startGame(e); err != nil {
		a.say("Cannot start "+e.name(), err.Error())
		return
	}

	now := time.Now()
	a.playing = &playingSession{
		slug:      e.slug(),
		name:      e.name(),
		start:     now,
		lastCheck: now,
	}
	a.playNote = "Started " + shortExe(e)
	a.overlay = overlayPlaying

	// The watcher runs in its own goroutine, so the launcher stays usable and
	// closing the game does not close Nova. It ends the session when the game
	// exits, even if the player never presses End session.
	go a.watchGame(e.inst.Exe)
}

// watchGame ends the session when the game process exits. Explorer starts the
// game, so Nova finds it by name and follows that PID.
func (a *app) watchGame(exe string) {
	pid, ok := waitForGamePID(exe, 30*time.Second)
	if !ok {
		return
	}
	for {
		time.Sleep(2 * time.Second)
		if pidAlive(pid) {
			continue
		}
		a.update(func() {
			if a.playing != nil {
				a.quitPlay()
			}
		})
		return
	}
}

// checkpointPlay writes the time played so far. It runs from the view loop,
// so it shares the view's goroutine and needs no extra lock.
func (a *app) checkpointPlay() {
	if a.playing == nil {
		return
	}
	now := time.Now()
	if now.Sub(a.playing.lastCheck) < playCheckpointEvery {
		return
	}
	ran := now.Sub(a.playing.start) - a.playing.recorded
	if ran <= 0 {
		return
	}
	a.playing.recorded += ran
	a.playing.lastCheck = now
	a.stat.addPlayed(a.playing.slug, ran)
	if err := a.stat.save(a.dataDir); err != nil {
		a.status = "could not save play time: " + err.Error()
	}
}

// quitPlay ends the session the app tracks and records how long it ran.
func (a *app) quitPlay() {
	if a.playing == nil {
		a.overlay = overlayNone
		return
	}

	ran := time.Since(a.playing.start) - a.playing.recorded
	if ran < 0 {
		ran = 0
	}
	a.stat.record(a.playing.slug, ran)

	name := a.playing.name
	a.playing = nil
	a.overlay = overlayNone
	a.say("Session saved", name+" · "+played(ran))

	if err := a.stat.save(a.dataDir); err != nil {
		a.status = "could not save play time: " + err.Error()
	}
}

// sessionElapsed is how long the current session has run.
func (a *app) sessionElapsed() time.Duration {
	if a.playing == nil {
		return 0
	}
	return time.Since(a.playing.start)
}

// clock formats a duration as the session timer: 00:04:12.
func clock(d time.Duration) string {
	total := int(d.Seconds())
	h := total / 3600
	m := total % 3600 / 60
	s := total % 60
	return itoaPad(h) + ":" + itoaPad(m) + ":" + itoaPad(s)
}
