package vmcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPublishProvisionPreservesNestedMetadataAcrossRefresh(t *testing.T) {
	for _, version := range []string{"go1.26.3", "next-version"} {
		t.Run(version, func(t *testing.T) {
			h, _, _ := hostFixture(t)
			old, err := DecodeImageManifest([]byte(validManifestJSON()))
			if err != nil {
				t.Fatal(err)
			}
			old.Unknown = map[string]json.RawMessage{"future_top": json.RawMessage(`{ "old" : [1, true] }`)}
			tool := old.Tools["go"]
			tool.Unknown = map[string]json.RawMessage{"future_metadata": json.RawMessage(`{ "nested" : [1, { "x" : true }] }`)}
			old.Tools["go"] = tool
			prior, err := EncodeImageManifest(old)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(h.root, "image.json"), prior, 0600); err != nil {
				t.Fatal(err)
			}
			old, err = DecodeImageManifest(prior)
			if err != nil {
				t.Fatal(err)
			}
			candidate, err := DecodeImageManifest([]byte(validManifestJSON()))
			if err != nil {
				t.Fatal(err)
			}
			candidate.Image = "pair-conformance-refreshed"
			candidate.Unknown = map[string]json.RawMessage{"new_top": json.RawMessage(`{"current":true}`)}
			tool = candidate.Tools["go"]
			tool.Version = version
			tool.Unknown = map[string]json.RawMessage{"observed_only": json.RawMessage(`{"new":true}`)}
			candidate.Tools["go"] = tool
			raw, err := EncodeImageManifest(candidate)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(raw, &fields); err != nil {
				t.Fatal(err)
			}
			if err := publishProvision(context.Background(), h.root, candidate, fields["tools"]); err != nil {
				t.Fatal(err)
			}
			active, err := readMachineImage(h.root)
			if err != nil {
				t.Fatal(err)
			}
			got, err := DecodeImageManifest(active)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got.Tools["go"].Unknown["future_metadata"], old.Tools["go"].Unknown["future_metadata"]) {
				t.Error("refresh lost existing nested raw metadata")
			}
			if !bytes.Equal(got.Unknown["future_top"], old.Unknown["future_top"]) {
				t.Error("refresh lost existing top metadata")
			}
			if len(got.Tools["go"].Unknown["observed_only"]) == 0 || len(got.Unknown["new_top"]) == 0 {
				t.Error("refresh lost newly supplied metadata")
			}
			if got.Tools["go"].Version != version {
				t.Error("old metadata replaced current observed version")
			}
		})
	}
}
