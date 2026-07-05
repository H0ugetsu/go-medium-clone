package httputil

import "encoding/json"

// Field distinguishes three JSON states on a PATCH-like update: the key
// absent (don't touch), the key present with value null (explicit clear),
// and the key present with a value.
type Field[T any] struct {
	Present bool
	Null    bool
	Value   T
}

func (f *Field[T]) UnmarshalJSON(data []byte) error {
	f.Present = true
	if string(data) == "null" {
		f.Null = true
		return nil
	}
	return json.Unmarshal(data, &f.Value)
}
