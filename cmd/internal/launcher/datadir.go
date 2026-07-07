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
