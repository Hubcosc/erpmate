package master

import (
	"bytes"
	"encoding/json"
)

// Optional distinguishes the three states a JSON field can be in:
//
//	field absent    -> Set=false          (leave column unchanged)
//	"field": null   -> Set=true, Null=true (clear column to NULL)
//	"field": value  -> Set=true, Value=... (update column)
//
// A plain *T cannot tell "absent" from "null", so a frontend could never clear
// a nullable field. Use Optional[T] for nullable columns in update requests.
type Optional[T any] struct {
	Set   bool
	Null  bool
	Value T
}

func (o *Optional[T]) UnmarshalJSON(b []byte) error {
	o.Set = true
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		o.Null = true
		return nil
	}
	return json.Unmarshal(b, &o.Value)
}
