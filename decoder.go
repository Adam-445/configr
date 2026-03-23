package configr

import "io"

// Decoder knows how to unmarshal raw bytes (from a config file) into a go value.
// Implementing this interface is all you need to add a new format (TOML, INI, etc.)
//
// The built-in implementations live in the sub-packages :
// - github.com/Adam-445/configr/json (default, no extra dependencies)
// - github.com/Adam-445/configr/yaml (requires gopkg.in/yaml.v3)
type Decoder interface {
	Decode(r io.Reader, v any) error
}
