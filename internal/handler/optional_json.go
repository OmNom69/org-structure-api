package handler

import (
	"bytes"
	"encoding/json"
)

type optionalJSONField[T any] struct {
	Set   bool
	Value *T
}

func (f *optionalJSONField[T]) UnmarshalJSON(data []byte) error {
	f.Set = true

	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		f.Value = nil
		return nil
	}

	var value T

	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	f.Value = &value
	return nil
}
