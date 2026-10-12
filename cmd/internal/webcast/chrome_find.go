package webcast

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ChromeEnv names the cast Chrome's binary, overriding the search.
const ChromeEnv = "COUCH_WEBCAST_CHROME"

// ErrNoChrome: no Chrome was found and ChromeEnv is unset.
var ErrNoChrome = errors.New("webcast: Chrome not found (set " + ChromeEnv + ")")

// FindChrome locates the Chrome to launch: ChromeEnv when set (it must name an
// existing file; a typo refuses rather than falling back), else the standard
// macOS app bundles, system-wide then the user's, else Chrome or Chromium on
// PATH.
func FindChrome(getenv func(string) string, lookPath func(string) (string, error), exists func(string) bool) (string, error) {
	if v := getenv(ChromeEnv); v != "" {
		if !exists(v) {
			return "", fmt.Errorf("webcast: %s=%q does not exist", ChromeEnv, v)
		}
		return v, nil
	}
	var bundles []string
	for _, dir := range []string{"/Applications", filepath.Join(getenv("HOME"), "Applications")} {
		bundles = append(bundles,
			filepath.Join(dir, "Google Chrome.app/Contents/MacOS/Google Chrome"),
			filepath.Join(dir, "Chromium.app/Contents/MacOS/Chromium"))
	}
	for _, b := range bundles {
		if exists(b) {
			return b, nil
		}
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser"} {
		if p, err := lookPath(name); err == nil {
			return p, nil
		}
	}
	return "", ErrNoChrome
}

func execLookPath(name string) (string, error) { return exec.LookPath(name) }

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
