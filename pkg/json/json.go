package json //nolint:revive

import (
	"os"

	libjson "encoding/json"
)

type RawMessage = libjson.RawMessage

func Marshal[T any](v T) ([]byte, error) {
	return libjson.Marshal(v)
}

func Unmarshal[T any](data []byte) (T, error) {
	var v T

	err := libjson.Unmarshal(data, &v)

	return v, err
}

func MarshalToFile[T any](v T, filename string) error {
	data, err := Marshal(v)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0600)
}

func UnmarshalFromFile[T any](filename string) (T, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		var v T

		return v, err
	}

	return Unmarshal[T](data)
}
