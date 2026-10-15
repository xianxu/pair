//go:build !darwin

package sessioninventory

import "runtime"

func codexForcedPreferences() (present, known bool) {
	// Windows uses a separate ProgramData policy namespace. Until that namespace
	// is observed, absence of macOS policy is not proof of complete local config.
	if runtime.GOOS == "windows" {
		return false, false
	}
	return false, true
}
