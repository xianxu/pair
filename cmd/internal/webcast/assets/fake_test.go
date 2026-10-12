package assets

import (
	"bytes"
	"image/png"
	"testing"
)

func TestMemStoreMeetsTheStoreContract(t *testing.T) {
	RunStoreContract(t, func() Store { return NewMemStore() })
}

func TestTinyPNGDecodes(t *testing.T) {
	if _, err := png.Decode(bytes.NewReader(tinyPNG)); err != nil {
		t.Fatal(err)
	}
}
