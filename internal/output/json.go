package output

import (
	"encoding/json"
	"io"
)

// WriteJSON writes v as indented JSON to w.
func WriteJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
