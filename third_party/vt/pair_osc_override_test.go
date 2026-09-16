package vt

import "testing"

func TestRegisteredOSCAdapterPrecedesEffectFallback(t *testing.T) {
	e := NewEmulator(8, 2)
	defer e.Close()
	var fallback, adapted int
	e.SetCallbacks(Callbacks{Notification: func(_, _ string) { fallback++ }})
	e.RegisterOscHandler(9, func(data []byte) bool {
		if string(data) != "9;adapter" {
			return false
		}
		adapted++
		return true
	})
	e.Write([]byte("\x1b]9;adapter\a\x1b]9;ordinary\a"))
	if adapted != 1 || fallback != 1 {
		t.Fatalf("adapter=%d fallback=%d", adapted, fallback)
	}
}
