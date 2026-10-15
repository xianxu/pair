package checkpoint

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/internal/reporoot"
)

// The phase vocabulary is written out exactly once, in AllPhases: any other
// []Phase literal -- production or test -- is a restatement that a new phase
// would silently skip.
func TestPhaseListIsWrittenOnlyInAllPhases(t *testing.T) {
	repo, err := reporoot.Current()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(repo, "cmd")
	var found []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(body), "\n") {
			if (strings.Contains(line, "[]Phase{") || strings.Contains(line, "[]checkpoint.Phase{")) && !strings.Contains(line, "func AllPhases()") {
				found = append(found, fmt.Sprintf("%s:%d", path, i+1))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("phase list restated outside AllPhases: %v", found)
	}
}
