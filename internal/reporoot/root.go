// Package reporoot locates checkout assets without depending on compiled source
// filenames, which are module-relative when Go builds with -trimpath.
package reporoot

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Current finds the Pair checkout containing the working directory.
func Current() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return Find(dir)
}

// Find walks from start to its nearest module root and verifies Pair's module
// identity. It never falls through a different module to an enclosing checkout.
func Find(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("checkout directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("checkout directory %q is not a directory", dir)
	}
	for {
		path := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(path)
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line, _, _ = strings.Cut(line, "//")
				fields := strings.Fields(line)
				if len(fields) == 2 && fields[0] == "module" && (fields[1] == "github.com/xianxu/pair" || fields[1] == `"github.com/xianxu/pair"`) {
					return dir, nil
				}
			}
			return "", fmt.Errorf("%s is not the github.com/xianxu/pair module", path)
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("read checkout module: %w", err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no Pair checkout containing %q", start)
		}
		dir = parent
	}
}
