package configr

import (
	"encoding/json"
	"io"
)

// jsonDecoder is the default decoder. It uses encoding/json from the standard library
type jsonDecoder struct{}

func (jsonDecoder) Decode(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields() // fail when there is a typo in the config file
	return dec.Decode(v)
}

// JSON is a ready to use JSON Decoder value. Pass it to WithDecoder if you want to be
// explicit rather than relying on the default.
//
// configr.New[MyConfig]("config.json", configr.WithDecoder[MyConfig](configr.JSON))
var JSON Decoder = jsonDecoder{}
