package notes

import (
	"encoding/json"
	"os"
	"testing"
)

func TestAnchorConformance(t *testing.T) {
	var cases []struct {
		Name       string
		Anchor     AnchorCandidate
		Candidates []AnchorCandidate
		Want       string
		Truncated  bool
		Route      string
	}
	data, err := os.ReadFile("testdata/anchor-candidates.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			got, ok := ChooseAnchor(c.Anchor, c.Candidates, c.Truncated, c.Route)
			if ok != (c.Want != "") || got.Key != c.Want {
				t.Fatalf("got %q,%v want %q", got.Key, ok, c.Want)
			}
		})
	}
}
