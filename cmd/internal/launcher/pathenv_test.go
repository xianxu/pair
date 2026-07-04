package launcher

import (
	"os"
	"testing"
)

func TestPrependBinToPath(t *testing.T) {
	sep := string(os.PathListSeparator)
	bin := "/root/bin"
	cases := []struct {
		name string
		home string
		path string
		want string
	}{
		{"empty path", "/root", "", bin},
		{"prepends when absent", "/root", "/usr/bin" + sep + "/bin", bin + sep + "/usr/bin" + sep + "/bin"},
		{"idempotent when already first", "/root", bin + sep + "/usr/bin", bin + sep + "/usr/bin"},
		{"idempotent when it is the whole path", "/root", bin, bin},
		{"prepends even if present later (dup is harmless)", "/root", "/usr/bin" + sep + bin, bin + sep + "/usr/bin" + sep + bin},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := prependBinToPath(tc.home, tc.path); got != tc.want {
				t.Fatalf("prependBinToPath(%q,%q) = %q, want %q", tc.home, tc.path, got, tc.want)
			}
		})
	}
}
