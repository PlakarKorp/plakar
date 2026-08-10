package utils

import (
	"io"

	"go.yaml.in/yaml/v3"
)

func YAMLEncode(w io.Writer, x any) error {
	enc := yaml.NewEncoder(w)
	if err := enc.Encode(x); err != nil {
		enc.Close()
		return err
	}
	return enc.Close()
}
