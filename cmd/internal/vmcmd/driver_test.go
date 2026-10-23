package vmcmd

import (
	"runtime"
	"testing"
)

func TestNativeDriverRefusesUnsupportedPlatform(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("native constructor covered without real Tart operations by executable seam")
	}
	driver, err := NewNativeDriver()
	if err == nil || driver != nil {
		t.Fatalf("unsupported platform accepted: %v %v", driver, err)
	}
}
