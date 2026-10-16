package sessioninventory

import (
	"errors"
	"os"
	"path/filepath"
)

// ResolveCodexHome resolves the transcript namespace, without loading native
// configuration. startupCWD is the native process directory before any -C flag.
// Codex rust-v0.162.1 utils/home-dir/src/lib.rs canonicalizes explicit homes;
// the default HOME/.codex need not exist. An unobservable home fails closed.
func ResolveCodexHome(startupCWD string, getenv func(string) string) (string, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	unavailable := errors.New("Codex native home unavailable")
	home := getenv("CODEX_HOME")
	if home == "" {
		fallback := getenv("HOME")
		if !filepath.IsAbs(fallback) {
			return "", unavailable
		}
		return filepath.Join(fallback, ".codex"), nil
	}
	if !filepath.IsAbs(home) {
		if !filepath.IsAbs(startupCWD) {
			return "", unavailable
		}
		home = filepath.Join(startupCWD, home)
	}
	physical, err := filepath.EvalSymlinks(home)
	if err != nil {
		return "", unavailable
	}
	info, err := os.Stat(physical)
	if err != nil || !info.IsDir() {
		return "", unavailable
	}
	return physical, nil
}
