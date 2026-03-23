package configr

import "time"

// options holds all optional configuration for a loader
// Usersr never touch this struct directly, instead the use With* functions below
type options[T any] struct {
	decoder       Decoder
	pollInterval  time.Duration
	onChange      func(T)
	validate      func(T) error
	applyDefaults func(*T)
}

// Option is a function that configures a Loader.
// Use the With* constructors to create one.
type Option[T any] func(*options[T])

// WithDecoder sets the format decoder (JSON, YAML, etc.)
// Defaults to JSON if not specified
func WithDecoder[T any](d Decoder) Option[T] {
	return func(o *options[T]) {
		o.decoder = d
	}
}

// WithPollInterval controls how often the watcher checks the config file for changes.
// Defaults to 2 seconds. Lower values react faster and higher values reduce IO
func WithPollInterval[T any](d time.Duration) Option[T] {
	return func(o *options[T]) {
		o.pollInterval = d
	}
}

// WithOnChange registers a callback that is invoked everytime the config file changes
// and the new config passes validation. the callback recieves the fully parsed and
// validated config. It is called in a seperate goroutine, do not call Get() from
// inside it (you already have the new value)
func WithOnChange[T any](fn func(T)) Option[T] {
	return func(o *options[T]) {
		o.onChange = fn
	}
}

// WithValidate registers a validation function that is called after each parse.
// If it returns a non-nil error the new config is discarded and the previous one stays
// in effect.
func WithValidate[T any](fn func(T) error) Option[T] {
	return func(o *options[T]) {
		o.validate = fn
	}
}

// WithDefaults registers a function that fills in zero-value fields before validation.
// Recieve a pointer so you can mutate inplace.
func WithDefaults[T any](fn func(*T)) Option[T] {
	return func(o *options[T]) {
		o.applyDefaults = fn
	}
}
