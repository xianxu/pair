package testcouch

import (
	"fmt"
	"os"
	"path/filepath"
)

type goCaches struct {
	GOCACHE    string
	GOMODCACHE string
}

// buildEnvironment begins empty. Only selected tools and deliberately shared Go
// caches cross the boundary; user Go configuration and proxies never do.
func buildEnvironment(root, toolPath string, caches goCaches) ([]string, error) {
	for _, path := range []string{root, toolPath, caches.GOCACHE, caches.GOMODCACHE} {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return nil, fmt.Errorf("build environment requires absolute paths: %q", path)
		}
	}
	paths := map[string]string{"HOME": "home", "XDG_CONFIG_HOME": "config", "XDG_DATA_HOME": "data", "XDG_STATE_HOME": "state", "XDG_CACHE_HOME": "cache", "TMPDIR": "tmp", "GOTMPDIR": "tmp"}
	env := []string{"PATH=" + toolPath, "GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOFLAGS=-mod=readonly", "GOCACHE=" + caches.GOCACHE, "GOMODCACHE=" + caches.GOMODCACHE, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0"}
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CACHE_HOME", "TMPDIR", "GOTMPDIR"} {
		path := filepath.Join(root, paths[key])
		if err := os.MkdirAll(path, 0700); err != nil {
			return nil, err
		}
		env = append(env, key+"="+path)
	}
	return env, nil
}
