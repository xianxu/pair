package procutil

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestStrictIdentity(t *testing.T) {
	for _, pid := range []string{"", "0", "-1", "self", "2147483646"} {
		if got := StrictIdentity(pid); got != "" {
			t.Errorf("StrictIdentity(%q) = %q", pid, got)
		}
	}
	pid := strconv.Itoa(os.Getpid())
	got := StrictIdentity(pid)
	switch runtime.GOOS {
	case "darwin":
		if got != "darwin:"+Identity(pid) {
			t.Fatalf("missing native timestamp: %q", got)
		}
	case "linux":
		boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
		if err != nil {
			t.Fatal(err)
		}
		if got != "linux:"+strings.TrimSpace(string(boot))+":"+Identity(pid) {
			t.Fatalf("missing boot ID/start ticks: %q", got)
		}
	default:
		if got != "" {
			t.Fatalf("unsupported strict identity = %q", got)
		}
	}
}
