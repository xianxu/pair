package launcher

import "path/filepath"

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
