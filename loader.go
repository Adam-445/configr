package configr

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

const defaultPollInterval = 2 * time.Second

// Loader holds a live, thread-safe view of your config file.
// T must be a struct type whose fields match he config format you're using
//
// Zero-value is not useful, create via New or Load.
type Loader[T any] struct {
	path    string
	opts    options[T]
	val     atomic.Pointer[T] // lock-free read path
	watcher *watcher
}

// Load is a one liner API. It reads the file at path, decodes it into T,
// applies defaults and validation, and returns the result.
//
// Load does NOT watch for changes. If you need hot-reload use New instead.
//
// cfg, err := configr.Load[MyConfig]("config.json")
func Load[T any](path string, opts ...Option[T]) (*T, error) {
	l, err := New[T](path, opts...)
	if err != nil {
		return nil, err
	}
	cfg := l.Get()
	return &cfg, nil
}

// New creates a Loader that reads a path, applies your options, and if WithOnChange
// was provided, automatically reloads when the file changes.
//
// loader, err := configr.New[MyConfig](
//
//	"config.json",
//	configr.WithDecoder[MyConfig](jsonDecoder),
//	configr.WithOnChange(func(cfg MyConfig) {
//		server.Reload(cfg)
//	}),
//
// )
func New[T any](path string, opts ...Option[T]) (*Loader[T], error) {
	l := &Loader[T]{
		path: path,
		opts: options[T]{
			pollInterval: defaultPollInterval,
		},
	}

	for _, o := range opts {
		o(&l.opts)
	}

	// Default to JSON if no decoder was specified
	if l.opts.decoder == nil {
		l.opts.decoder = jsonDecoder{}
	}

	// Load once eagerly so callers can fail fast on startup
	if err := l.reload(); err != nil {
		return nil, fmt.Errorf("configr: initial load of %q failed: %w", path, err)
	}

	// Start the watcher only if the caller registered an onChange callback or
	// explicitly called Watch() (see Watch() below)
	if l.opts.onChange != nil {
		l.startWatcher()
	}

	return l, nil
}

// Get returns the current config. It is safe to call from any goroutine and doesnt block,
// reads are served from an atomic pointer.
//
// The returned value is a copy, so mutating it has no effect on future Get calls.
func (l *Loader[T]) Get() T {
	return *l.val.Load()
}

// Watch starts background file watching even if no OnChange callback was registered.
// Useful when ou poll Get() yourself rather than using callbacks.
// Calling Watch on an already-watching Loader is a no-op
func (l *Loader[T]) Watch() {
	if l.watcher != nil {
		return
	}
	l.startWatcher()
}

// Stop halts background watching. After Stop, Get still returns the last successfully
// loaded config. Calling Stop on a non-watching Loader is a no-op.
func (l *Loader[T]) Stop() {
	if l.watcher == nil {
		return
	}
	close(l.watcher.stop)
	l.watcher = nil
}

// startWatcher wires up the polling loop
func (l *Loader[T]) startWatcher() {
	l.watcher = newWatcher(l.path, l.opts.pollInterval, func() {
		if err := l.reload(); err != nil {
			// Parse or validation error.
			// Keep the current config in effect.
			// Production systems should log this, however we dont impose a logger here
			_ = err
		}
	})
	l.watcher.start()
}

// reload reads the file, decodes it, applies defaults, validates, and if everything passes
// atomically swaps in the new config.
func (l *Loader[T]) reload() error {
	f, err := os.Open(l.path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close() // the error here is non-impactful so we ignore it
	}()

	var cfg T
	if err := l.opts.decoder.Decode(f, &cfg); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	if l.opts.applyDefaults != nil {
		l.opts.applyDefaults(&cfg)
	}

	if l.opts.validate != nil {
		if err := l.opts.validate(cfg); err != nil {
			return fmt.Errorf("validate: %w", err)
		}
	}

	// Atomically swap in the new config. Readers using Get() see either the
	// old or new value. never a partial write (the same safety guarantee Raft
	// provides for cluster configuration transitions).
	// Taking &cfg here causes it to escape the heap. The GC will keep it alive as long
	// as the atomic.Pointer holds a reference to it.
	l.val.Store(&cfg)

	if l.opts.onChange != nil {
		go l.opts.onChange(cfg)
	}

	return nil
}
