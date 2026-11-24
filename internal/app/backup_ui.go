package app

import (
	"path/filepath"

	"github.com/egoist/mygo"

	"github.com/yafit/nova/internal/backup"
	"github.com/yafit/nova/internal/gse"
	"github.com/yafit/nova/internal/library"
)

// exportData offers one portable JSON file with play time, library entries,
// favorites and local achievement unlocks. The dialog blocks, so it runs on
// its own goroutine and returns through update like the folder picker.
func (a *app) exportData() {
	if a.backupBusy {
		return
	}
	a.backupBusy = true
	win := a.win
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent:      win,
			Title:       "Export Nova data",
			DefaultPath: "nova-backup.json",
			ButtonLabel: "Export",
			Filters:     []mygo.FileFilter{{Name: "JSON", Extensions: []string{"json"}}},
		})
		a.update(func() {
			a.backupBusy = false
			if err != nil || path == "" {
				return
			}
			if err := backup.Write(path, a.collectBackup()); err != nil {
				a.say("Export failed", err.Error())
				return
			}
			a.say("Data exported", filepath.Base(path))
		})
	}()
}

// importData reads a backup and merges it into what is already here. An older
// file cannot lower play time or turn an unlock off.
func (a *app) importData() {
	if a.backupBusy {
		return
	}
	a.backupBusy = true
	win := a.win
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:      win,
			Title:       "Import Nova data",
			ButtonLabel: "Import",
			Filters:     []mygo.FileFilter{{Name: "JSON", Extensions: []string{"json"}}},
		})
		a.update(func() {
			a.backupBusy = false
			if err != nil || len(paths) == 0 {
				return
			}
			old, err := backup.Read(paths[0])
			if err != nil {
				a.say("Import failed", err.Error())
				return
			}
			a.applyBackup(backup.Merge(a.collectBackup(), old))
			a.say("Data imported", filepath.Base(paths[0]))
		})
	}()
}

// collectBackup turns the app's live state into the portable shape.
func (a *app) collectBackup() backup.Bundle {
	games := map[string]backup.Game{}
	for key, g := range a.stat.Games {
		games[key] = backup.Game{Played: g.Played, Sessions: g.Sessions, Last: g.Last}
	}
	return backup.Bundle{
		Library:      a.libCfg,
		Favorites:    a.favorites,
		Games:        games,
		Achievements: gse.All(a.gseRoot),
	}
}

// applyBackup writes the merged data synchronously, so a later exit cannot
// race the file write, and clears the achievement board so it reloads.
func (a *app) applyBackup(b backup.Bundle) {
	a.stat.Games = map[string]*gameStats{}
	for key, g := range b.Games {
		a.stat.Games[key] = &gameStats{Played: g.Played, Sessions: g.Sessions, Last: g.Last}
	}
	if err := a.stat.save(a.dataDir); err != nil {
		a.status = "could not save play time: " + err.Error()
	}

	cfg := b.Library
	cfg.Favorites = b.Favorites
	a.libCfg = cfg
	a.favorites = cfg.Favorites
	if a.favorites == nil {
		a.favorites = map[string]bool{}
	}
	if err := library.SaveConfig(a.dataDir, cfg); err != nil {
		a.status = "could not save library: " + err.Error()
	}
	if err := gse.Restore(a.gseRoot, b.Achievements); err != nil {
		a.status = "could not restore achievements: " + err.Error()
	}

	a.achievements = trophyBoard{}
	a.refresh()
}
