package launcher

import (
	"os"
	"testing"
)

func TestPrependBinToPath(t *testing.T) {
	sep := string(os.PathListSeparator)
	bin := "/root/bin"
	cases := []struct {
		name   string
		home   string
		exeDir string
		path   string
		want   string
	}{
		// exeDir == binDir (dev/source layout): a single front dir.
		{"empty path", "/root", bin, "", bin},
		{"prepends when absent", "/root", bin, "/usr/bin" + sep + "/bin", bin + sep + "/usr/bin" + sep + "/bin"},
		{"idempotent when already first", "/root", bin, bin + sep + "/usr/bin", bin + sep + "/usr/bin"},
		{"idempotent when it is the whole path", "/root", bin, bin, bin},
		{"dedups a later occurrence to the front", "/root", bin, "/usr/bin" + sep + bin, bin + sep + "/usr/bin"},
		// exeDir differs from binDir (copied/Homebrew layout): pair lives in exeDir,
		// so both are fronted (exeDir first) — this is what makes `pair` resolve.
		{"fronts both exeDir and binDir", "/root", "/opt/hb/bin", "/usr/bin", "/opt/hb/bin" + sep + bin + sep + "/usr/bin"},
		{"empty exeDir fronts binDir only", "/root", "", "/usr/bin", bin + sep + "/usr/bin"},
		// Fully idempotent on re-launch: re-running on a prior output is a no-op
		// (no PATH growth across in-session restarts).
		{"idempotent on prior output", "/root", "/opt/hb/bin", "/opt/hb/bin" + sep + bin + sep + "/usr/bin", "/opt/hb/bin" + sep + bin + sep + "/usr/bin"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := prependBinToPath(tc.home, tc.exeDir, tc.path); got != tc.want {
				t.Fatalf("prependBinToPath(%q,%q,%q) = %q, want %q", tc.home, tc.exeDir, tc.path, got, tc.want)
			}
		})
	}
}
