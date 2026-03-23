// Package yaml provides a YAML Decoder for use with github.com/Adam-445/configr.
// This sub package depends on gopkg.in/yaml.v3. Add it with:
//
// Import this package alongside configr:
//
// import (
//
//	"github.com/Adam-445/configr"
//	configryaml "github.com/Adam-445/configr/yaml"
//
// )
//
// loader, err := configr.New[MyConfig](
//
//	"config.yaml",
//	configr.WithDecoder[MyConfig](configryaml.YAML)
//
// )
//
// This sub-package depends on gopkg.in/yaml.v3. Add it with:
//
// go get gopkg.in/yaml.v3
package yaml

import (
	"io"

	"gopkg.in/yaml.v3"
)

type yamlDecoder struct{}

// Decode implements configr.Decoder using gopkg.in/yaml.v3
func (yamlDecoder) Decode(r io.Reader, v any) error {
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true) // fail on unknown fields (like the json decoder)
	return dec.Decode(v)
}

// YAML is a ready to use Yaml Decoder value.
var YAML = yamlDecoder{}
