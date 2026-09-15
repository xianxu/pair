package procutil

import (
	"os"
	"runtime"
	"strings"
)

// StrictIdentity is an incarnation token suitable for persisted ownership.
// Linux includes its boot ID because start ticks repeat across reboots. Other
// platforms must supply native identity before opting into destructive cleanup;
// the second-resolution ps fallback used by Identity is insufficient here.
func StrictIdentity(pid string) string {
	if _, ok := positivePID(pid); !ok {
		return ""
	}
	switch runtime.GOOS {
	case "darwin":
		if birth := Identity(pid); birth != "" {
			return "darwin:" + birth
		}
	case "linux":
		boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
		if err != nil {
			return ""
		}
		bootID := strings.TrimSpace(string(boot))
		if birth := Identity(pid); birth != "" && bootID != "" {
			return "linux:" + bootID + ":" + birth
		}
	}
	return ""
}
