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
