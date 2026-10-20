//go:build darwin

package procutil

import "testing"

func TestDarwinTTYMapsNoDeviceToNone(t *testing.T) {
	if got := darwinTTYName(-1); got != "" {
		t.Fatalf("NODEV = %q, want none", got)
	}
	if got := darwinTTYName(0); got != "" {
		t.Fatalf("0 = %q, want none", got)
	}
	if got := darwinTTYName(268435639); got != "268435639" {
		t.Fatalf("device = %q", got)
	}
}
