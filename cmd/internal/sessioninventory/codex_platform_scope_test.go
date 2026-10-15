package sessioninventory

import "testing"

func TestCodexConfigLinuxDriveScopeCompatibility(t *testing.T) {
	// Source: rust-v0.162.1 utils/path-utils/src/lib.rs
	// is_wsl_case_insensitive_path and normalize_for_wsl_with_flag. Linux
	// projections conservatively exclude this scope without detecting WSL.
	cases := []struct {
		goos, cwd string
		want      bool
	}{
		{"linux", "/mnt/c/Users/Work", true},
		{"linux", "/MNT/C/Users/Work", true},
		{"linux", "/mNt/z", true},
		{"linux", "/mnt/A/", true},
		{"linux", "//mnt//c/Work", true},
		{"linux", "/mnt/./c/Work", true},
		{"linux", "/mnt", false},
		{"linux", "/mnt/", false},
		{"linux", "/mnt/cc/Work", false},
		{"linux", "/mnt/case/Work", false},
		{"linux", "/mnt/1/Work", false},
		{"linux", "/mnt/c:/Work", false},
		{"linux", "/mnt/é/Work", false},
		{"linux", "/mntx/c/Work", false},
		{"linux", "/mount/c/Work", false},
		{"linux", "/home/mnt/c/Work", false},
		{"linux", "mnt/c/Work", false},
		{"linux", "/mnt/../c/Work", false},
		{"linux", "", false},
		{"darwin", "/mnt/c/Users/Work", false},
		{"windows", "/mnt/c/Users/Work", false},
		{"windows", `C:\Users\Work`, false},
	}
	for _, tc := range cases {
		t.Run(tc.goos+tc.cwd, func(t *testing.T) {
			if got := codexUnsupportedLinuxDriveCWD(tc.goos, tc.cwd); got != tc.want {
				t.Fatalf("unsupported(%q,%q)=%v want %v", tc.goos, tc.cwd, got, tc.want)
			}
		})
	}
}
