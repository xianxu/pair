package launcher

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
)

// ResolveZellijSocketRoot resolves the native Zellij socket namespace from
// a snapshot taken before any slot-specific temporary-directory overlay. An
// explicit root retains its spelling: expanding a short symlink can exceed a
// native Unix socket pathname limit.
func ResolveZellijSocketRoot(explicitRoot, tempDir, runtimeDir, platform string, uid int) (string, error) {
	root := explicitRoot
	// directories-rs 6.0.0 uses an absolute XDG_RUNTIME_DIR plus the lowercase
	// application name on Linux; absent/relative XDG values have no runtime_dir.
	// Darwin has no ProjectDirs runtime_dir. Preserve path spelling as Rust's
	// PathBuf::join does, including a short symlink followed by parent segments.
	// https://docs.rs/directories/6.0.0/src/directories/lin.rs.html#64-94
	if root == "" && platform == "linux" && filepath.IsAbs(runtimeDir) {
		root = appendNativeSocketPath(runtimeDir, "zellij")
	}
	if root == "" {
		if uid < 0 || !filepath.IsAbs(tempDir) || strings.ContainsRune(tempDir, 0) {
			return "", errors.New("invalid Zellij socket fallback directory or uid")
		}
		root = appendNativeSocketPath(tempDir, "zellij-"+strconv.Itoa(uid))
	}
	if !filepath.IsAbs(root) || strings.ContainsRune(root, 0) {
		return "", errors.New("Zellij socket directory must be absolute and contain no NUL")
	}
	return root, nil
}

// appendNativeSocketPath matches native PathBuf append for already-absolute
// roots and relative suffixes. Preserve every root byte so the kernel resolves
// symlink/parent segments and repeated separators in the native spelling.
func appendNativeSocketPath(root, suffix string) string {
	if strings.HasSuffix(root, "/") {
		return root + suffix
	}
	return root + "/" + suffix
}
