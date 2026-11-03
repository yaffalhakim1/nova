package app

import (
	"time"
)

// playingSession is the game this app started: when it began, and which game.
type playingSession struct {
	slug  string
	name  string
	pid   uint32
	start time.Time
}

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

	proc, err := startGame(e)
	if err != nil {
		a.say("Cannot start "+e.name(), err.Error())
		return
	}

	a.playing = &playingSession{
		slug:  e.slug(),
		name:  e.name(),
		pid:   uint32(proc.Pid),
		start: time.Now(),
	}
	a.playNote = "Started " + shortExe(e)
	a.overlay = overlayPlaying

	// The process is released once it is running: the launcher does not wait
	// on the game, and closing the game does not close the launcher.
	_ = proc.Release()
}

// quitPlay ends the session the app tracks and records how long it ran.
func (a *app) quitPlay() {
	if a.playing == nil {
		a.overlay = overlayNone
		return
	}

	ran := time.Since(a.playing.start)
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
