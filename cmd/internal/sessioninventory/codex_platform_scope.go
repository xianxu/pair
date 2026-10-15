package sessioninventory

import "strings"

// codexUnsupportedLinuxDriveCWD bounds the WSL-specific native comparison rule
// without probing environment or platform files. Native lowercases ASCII paths
// under /mnt/<ASCII drive> on WSL; plain physical equality cannot reproduce it.
// Source: rust-v0.162.1 utils/path-utils/src/lib.rs
// is_wsl_case_insensitive_path / normalize_for_wsl_with_flag.
func codexUnsupportedLinuxDriveCWD(goos, cwd string) bool {
	if goos != "linux" || !strings.HasPrefix(cwd, "/") {
		return false
	}
	components := make([]string, 0, 2)
	for _, part := range strings.Split(cwd, "/") {
		if part == "" || part == "." {
			continue
		}
		components = append(components, part)
		if len(components) == 2 {
			break
		}
	}
	if len(components) != 2 {
		return false
	}
	mnt, drive := components[0], components[1]
	if len(mnt) != 3 || (mnt[0] != 'm' && mnt[0] != 'M') || (mnt[1] != 'n' && mnt[1] != 'N') || (mnt[2] != 't' && mnt[2] != 'T') || len(drive) != 1 {
		return false
	}
	return (drive[0] >= 'a' && drive[0] <= 'z') || (drive[0] >= 'A' && drive[0] <= 'Z')
}
