package ui

import (
	"os"
	"time"

	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
)

// programSender is the subset of *tea.Program the watcher needs (seam for tests).
type programSender interface {
	Send(tea.Msg)
}

// watchConfigFile polls the kafui config file's mtime every interval and, on a
// change, reloads it and pushes a ConfigReloadedMsg (or a UIError on failure)
// into the program (AC-16). It uses simple mtime polling — no fsnotify, no new
// dependency — and debounces by only acting on an mtime that differs from the
// last observed one. It runs until done is closed; on process exit the leaked
// goroutine dies with the process.
func watchConfigFile(p programSender, path string, interval time.Duration, done <-chan struct{}) {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	last := configModTime(path)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			cur := configModTime(path)
			if cur.Equal(last) {
				continue
			}
			last = cur
			cfg, err := appconfig.Load(path)
			if err != nil {
				shared.Log.Warn("config auto-reload failed", "path", path, "err", err)
				p.Send(shared.NewUIError("config-reload", "Config reload failed", err))
				continue
			}
			c := cfg
			p.Send(core.ConfigReloadedMsg{Config: &c})
		}
	}
}

// configModTime returns the file's modification time, or the zero time when the
// file is missing/unreadable (so a later appearance registers as a change).
func configModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
