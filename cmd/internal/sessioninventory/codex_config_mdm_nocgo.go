//go:build darwin && !cgo

package sessioninventory

func codexForcedPreferences() (present, known bool) { return false, false }
