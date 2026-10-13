//go:build !darwin

package webcast

// activateProcess: elsewhere, activating the cast tab is all Chrome needs.
func activateProcess(int) func() error { return nil }
