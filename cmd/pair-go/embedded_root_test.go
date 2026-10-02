package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedAssetsHonorSelectedGlobalRootBeforeLauncher(t *testing.T) {
	for _, tc := range []struct {
		name, selected string
		scoped         bool
		invalid        bool
	}{
		{name: "selected-global", selected: "selected"},
		{name: "selected-global-over-scoped", selected: "selected", scoped: true},
		{name: "invalid-selected-never-falls-back", selected: "relative-root", scoped: true, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			home, xdg, scoped := filepath.Join(root, "home"), filepath.Join(root, "xdg"), filepath.Join(root, "inherited-scope")
			selected := filepath.Join(root, tc.selected)
			if tc.invalid {
				selected = tc.selected
			} else if err := os.Mkdir(selected, 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("HOME", home)
			t.Setenv("XDG_DATA_HOME", xdg)
			t.Setenv("COUCH_PAIR_DATA_DIR", selected)
			t.Setenv("PAIR_DATA_DIR", "")
			if tc.scoped {
				t.Setenv("PAIR_DATA_DIR", scoped)
			}
			assetRoot, err := (osLegacyRuntime{}).EmbeddedAssetRoot()
			if tc.invalid {
				if err == nil || assetRoot != "" {
					t.Fatalf("invalid selected root extracted assets: root=%q err=%v", assetRoot, err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if !strings.HasPrefix(assetRoot, filepath.Join(selected, "runtime")+string(filepath.Separator)) || !validAssetRoot(osLegacyRuntime{}, assetRoot) {
					t.Fatalf("embedded asset root=%q; want usable assets under selected global root %q", assetRoot, selected)
				}
			}
			for _, ambient := range []string{home, xdg, scoped} {
				if _, err := os.Lstat(ambient); !os.IsNotExist(err) {
					t.Errorf("embedded extraction touched ambient root %q: %v", ambient, err)
				}
			}
		})
	}
}
