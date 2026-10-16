package sessioninventory

import (
	"errors"
	"os"
	"path/filepath"
)

// NewNativeTranscriptRuntime keeps discovery and validation in the same native
// transcript namespace. defaultHomeDir is the caller's observed home, including
// private reader fixtures. Historical callers leave startupCWD empty: a session
// working directory does not prove where a relative CODEX_HOME was resolved.
func NewNativeTranscriptRuntime(defaultHomeDir, pairDataDir string, agent Agent, startupCWD string, getenv func(string) string) (OSRuntime, error) {
	runtime := NewOSRuntime(defaultHomeDir, pairDataDir)
	if agent != AgentCodex {
		return runtime, nil
	}
	if getenv == nil {
		getenv = os.Getenv
	}
	home, err := ResolveCodexHome(startupCWD, func(key string) string {
		if key == "HOME" {
			return defaultHomeDir
		}
		return getenv(key)
	})
	if err != nil {
		return OSRuntime{}, err
	}
	return runtime.WithCodexHome(home)
}

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
