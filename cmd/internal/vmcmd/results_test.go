package vmcmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeResultManifestRefusesInvalidCatalogAndBudgets(t *testing.T) {
	for _, raw := range []string{
		`null`, `{"version":null,"files":[]}`, `{"version":1,"files":null}`,
		`{"version":1,"files":[{"path":"../escape","size":1}]}`,
		`{"version":1,"files":[{"path":"status","size":129}]}`,
		`{"version":1,"files":[{"path":"status","size":null}]}`,
		`{"version":1,"files":[{"path":"status","size":0},{"path":"status","size":0}]}`,
		`{"version":1,"files":[{"path":"diagnostics/zellij-8.log","size":0}]}`,
		`{"version":1,"files":[{"path":"diagnostics/zellij-0.log","size":65537}]}`,
		`{"version":1,"files":[{"path":"status","path":"status","size":0}]}`,
	} {
		if _, err := decodeResultManifest([]byte(raw), 1<<20); err == nil {
			t.Fatal("invalid manifest admitted", raw)
		}
	}
	raw := []byte(`{"version":1,"future":true,"files":[{"path":"status","size":2}]}`)
	if _, err := decodeResultManifest(raw, int64(len(raw)+2)); err != nil {
		t.Fatal(err)
	}
	if _, err := decodeResultManifest(raw, int64(len(raw)+1)); err == nil {
		t.Fatal("combined manifest/member overflow admitted")
	}
}
func TestExportGuestResultsBoundsEveryHostWrite(t *testing.T) {
	for _, overflow := range []bool{false, true} {
		t.Run(map[bool]string{false: "exact", true: "overflow"}[overflow], func(t *testing.T) {
			root := t.TempDir()
			contents := map[string][]byte{"status": []byte("ok"), "diagnostics/zellij-0.log": []byte(strings.Repeat("x", 65536))}
			manifest := resultManifest{Version: 1, Files: []resultFile{{Path: "status", Size: 2}, {Path: "diagnostics/zellij-0.log", Size: 65536}}}
			raw, _ := json.Marshal(manifest)
			contents["manifest.json"] = raw
			limit := int64(len(raw) + 2 + 65536)
			if overflow {
				contents["diagnostics/zellij-0.log"] = append(contents["diagnostics/zellij-0.log"], 'x')
			}
			calls := 0
			err := exportGuestResults(context.Background(), "/private/tmp/guest-results", root, limit, func(_ context.Context, path string, budget int64) ([]byte, bool, error) {
				calls++
				value, present := contents[filepath.Base(path)]
				if strings.HasSuffix(path, "diagnostics/zellij-0.log") {
					value, present = contents["diagnostics/zellij-0.log"]
				}
				return value, present, nil
			})
			if (err != nil) != overflow {
				t.Fatal(err)
			}
			if calls != 3 {
				t.Fatal("unexpected export population", calls)
			}
			size, err := boundedArtifactSize(context.Background(), root, limit)
			if err != nil || size > limit {
				t.Fatal("export exceeded admitted disk bytes", size, err)
			}
		})
	}
}
func TestExportGuestResultsRefusesExistingSymlink(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(outside, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "status")); err != nil {
		t.Fatal(err)
	}
	called := false
	err := exportGuestResults(context.Background(), "/private/tmp/results", root, 1<<20, func(context.Context, string, int64) ([]byte, bool, error) { called = true; return nil, false, nil })
	if err == nil || called {
		t.Fatal("existing result alias admitted")
	}
	raw, _ := os.ReadFile(outside)
	if string(raw) != "unchanged" {
		t.Fatal("result export changed outside file")
	}
}
