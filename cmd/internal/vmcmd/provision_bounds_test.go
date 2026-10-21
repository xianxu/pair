package vmcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishProvisionManifestSizeBoundary(t *testing.T) {
	for _, size := range []int{65536, 65537} {
		name := "exact-limit"
		if size > 65536 {
			name = "one-byte-over"
		}
		t.Run(name, func(t *testing.T) {
			h, _, _ := hostFixture(t)
			old, err := DecodeImageManifest([]byte(validManifestJSON()))
			if err != nil {
				t.Fatal(err)
			}
			old.Unknown = map[string]json.RawMessage{"future_metadata": json.RawMessage(`"` + strings.Repeat("m", 32000) + `"`)}
			prior, err := EncodeImageManifest(old)
			if err != nil {
				t.Fatal(err)
			}
			if len(prior) > 65536 {
				t.Fatalf("old manifest fixture exceeds reader bound: %d", len(prior))
			}
			if err := os.WriteFile(filepath.Join(h.root, "image.json"), prior, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := readMachineImage(h.root); err != nil {
				t.Fatal("old manifest is unreadable:", err)
			}

			candidate, err := DecodeImageManifest([]byte(validManifestJSON()))
			if err != nil {
				t.Fatal(err)
			}
			candidate.Image = "pair-conformance-new"
			combined, err := DecodeImageManifest(prior)
			if err != nil {
				t.Fatal(err)
			}
			combined.Image = candidate.Image
			previous, err := json.Marshal(old.Image)
			if err != nil {
				t.Fatal(err)
			}
			combined.Unknown["previous_image"] = previous
			tool := combined.Tools["go"]
			tool.Unknown = map[string]json.RawMessage{"future_provenance": json.RawMessage(`""`)}
			combined.Tools["go"] = tool
			base, err := EncodeImageManifest(combined)
			if err != nil {
				t.Fatal(err)
			}
			tool.Unknown["future_provenance"] = json.RawMessage(`"` + strings.Repeat("p", size-len(base)) + `"`)
			want, err := EncodeImageManifest(combined)
			if err != nil {
				t.Fatal(err)
			}
			if len(want) != size {
				t.Fatalf("combined fixture size = %d, want %d", len(want), size)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(want, &fields); err != nil {
				t.Fatal(err)
			}
			reported := fields["tools"]
			if len(reported) > 65536 {
				t.Fatalf("tools report fixture exceeds input bound: %d", len(reported))
			}

			err = publishProvision(context.Background(), h.root, candidate, reported)
			if size > 65536 {
				if err == nil {
					t.Error("published manifest one byte larger than reader bound")
				}
				active, readErr := os.ReadFile(filepath.Join(h.root, "image.json"))
				if readErr != nil {
					t.Fatal(readErr)
				}
				if !bytes.Equal(active, prior) {
					t.Error("refused publication changed prior active manifest bytes")
				}
				if _, readErr := readMachineImage(h.root); readErr != nil {
					t.Error("prior active manifest became unreadable:", readErr)
				}
				return
			}
			if err != nil {
				t.Fatal("exact-limit publication failed:", err)
			}
			active, err := readMachineImage(h.root)
			if err != nil {
				t.Fatal("published manifest is unreadable:", err)
			}
			if !bytes.Equal(active, want) {
				t.Fatal("exact-limit publication lost manifest metadata or tool provenance")
			}
		})
	}
}
