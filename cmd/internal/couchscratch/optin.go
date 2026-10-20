package couchscratch

import (
	"path/filepath"
	"strings"
)

// ChooseSlotScratch requires the operator's persistent setup declaration.
func ChooseSlotScratch(probe Probe, optedIn bool) bool { return optedIn && ChooseScratch(probe) }

// ScratchOptInPath locates the operator-owned declaration, never on the RAM disk.
func ScratchOptInPath(home, configHome string) string {
	if configHome == "" {
		if !filepath.IsAbs(home) || strings.ContainsRune(home, '\x00') {
			return ""
		}
		configHome = filepath.Join(home, ".config")
	}
	if !filepath.IsAbs(configHome) || strings.ContainsRune(configHome, '\x00') {
		return ""
	}
	return filepath.Join(configHome, "pair", "sandbox-allowed")
}
