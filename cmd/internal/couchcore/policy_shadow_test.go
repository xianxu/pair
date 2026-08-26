package couchcore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizedFleetPolicyHasNoProductionShadow(t *testing.T) {
	prohibited := []string{
		"type PolicyTable",
		"type Mode string",
		"policy.json",
		"func (c *Couch) Policy",
		".SameTree",
		`"same-tree"`,
		"--same-tree",
	}
	check := func(root string) error {
		return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, shadow := range prohibited {
				if strings.Contains(string(raw), shadow) {
					t.Errorf("%s retains normalized-policy shadow %q", path, shadow)
				}
			}
			return nil
		})
	}
	for _, root := range []string{".", "../couchcmd"} {
		if err := check(root); err != nil {
			t.Fatal(err)
		}
	}
}
