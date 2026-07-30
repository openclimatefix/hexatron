package utils

import (
	"encoding/json"
	"io"
)

// DecodeJSON decodes the JSON body from r into v.
func DecodeJSON(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}
