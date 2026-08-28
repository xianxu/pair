package sessioninventory

import (
	"encoding/json"
	"os"
	"testing"
)

type normalizationGolden struct {
	SchemaVersion int `json:"schema_version"`
	Cases         []struct {
		Name  string `json:"name"`
		Input string `json:"input"`
		Want  string `json:"want"`
	} `json:"cases"`
}

func TestNormalizePairTextGolden(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("testdata/normalization/v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden normalizationGolden
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatal(err)
	}
	if golden.SchemaVersion != 1 {
		t.Fatalf("schema_version = %d, want 1", golden.SchemaVersion)
	}
	for _, test := range golden.Cases {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizePairText(test.Input); got != test.Want {
				t.Fatalf("NormalizePairText(%q) = %q, want %q", test.Input, got, test.Want)
			}
		})
	}
}
