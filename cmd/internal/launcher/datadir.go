package launcher

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveGlobalDataDir preserves standalone HOME/XDG behavior while consuming
// Couch's selected global root independently of PAIR_DATA_DIR's scoped meaning.
// Selection was already published by Couch; absence or changed physical identity
// is a recovery error, never a reason to initialize ambient storage.
func ResolveGlobalDataDir(home, xdgDataHome, selectedGlobal string) (string, error) {
	if selectedGlobal == "" {
		return ResolveDataDir(home, xdgDataHome), nil
	}
	if !filepath.IsAbs(selectedGlobal) || filepath.Clean(selectedGlobal) != selectedGlobal {
		return "", fmt.Errorf("COUCH_PAIR_DATA_DIR requires a canonical absolute selected root: %q", selectedGlobal)
	}
	physical, err := filepath.EvalSymlinks(selectedGlobal)
	if err != nil {
		return "", fmt.Errorf("COUCH_PAIR_DATA_DIR selected root unavailable: %w", err)
	}
	if physical != selectedGlobal {
		return "", fmt.Errorf("COUCH_PAIR_DATA_DIR selected root changed physical identity: %q", selectedGlobal)
	}
	info, err := os.Stat(selectedGlobal)
	if err != nil {
		return "", fmt.Errorf("COUCH_PAIR_DATA_DIR selected root unavailable: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("COUCH_PAIR_DATA_DIR selected root is not a directory: %q", selectedGlobal)
	}
	return selectedGlobal, nil
}

// ResolveDataDir returns Pair's data directory from explicit environment values.
func ResolveDataDir(home, xdgDataHome string) string {
	if xdgDataHome != "" {
		return filepath.Join(xdgDataHome, "pair")
	}
	return filepath.Join(home, ".local", "share", "pair")
}

func ScopedLaunchDataDir(globalDataDir, cwd string) string {
	scope, err := ResolveRepoScope(cwd)
	if err != nil {
		return globalDataDir
	}
	return NewScopedPaths(globalDataDir, scope, "").ScopeDir()
}

func scopeKeyFromDataDir(globalDataDir, dataDir string) string {
	rel, err := filepath.Rel(globalDataDir, dataDir)
	if err != nil {
		return ""
	}
	dir, key := filepath.Split(rel)
	if filepath.Clean(dir) != "repos" || key == "" {
		return ""
	}
	return key
}
