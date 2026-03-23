package configr

import (
	"os"
	"time"
)

// watcher polls a file for mtime changes and calls notify when a change is detected.
// It uses polling rather than ionotify/kqueue because:
//   - Polling works identically on Linux, macOS, and Windows
//   - Config files changes rarely, a 2 second poll adds negligible overhead
//   - No additional dependencies
//
// if you need sub-second reaction times consider replacing this with fsnotify
type watcher struct {
	path     string
	interval time.Duration
	lastMod  time.Time
	notify   func()
	stop     chan struct{}
}

func newWatcher(path string, interval time.Duration, notify func()) *watcher {
	return &watcher{
		path:     path,
		interval: interval,
		notify:   notify,
		stop:     make(chan struct{}),
	}
}

// start begins polling in a new goroutine. Call stop() to halt it.
func (w *watcher) start() {
	// Capture the current mtime so the first tick doesn't fire a reload
	if info, err := os.Stat(w.path); err == nil {
		w.lastMod = info.ModTime()
	}

	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				w.check()
			case <-w.stop:
				return
			}
		}
	}()
}

func (w *watcher) check() {
	info, err := os.Stat(w.path)
	if err != nil {
		return // file temporarily unavailable (write in progress etc.). Skip tick
	}
	if info.ModTime().After(w.lastMod) {
		w.lastMod = info.ModTime()
		w.notify()
	}
}
