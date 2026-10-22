package launcher

import "github.com/xianxu/pair/cmd/internal/zellijquery"

// ResolveZellijSocketRoot preserves the native socket spelling through the shared read seam.
func ResolveZellijSocketRoot(explicitRoot, tempDir, runtimeDir, platform string, uid int) (string, error) {
	return zellijquery.ResolveZellijSocketRoot(explicitRoot, tempDir, runtimeDir, platform, uid)
}
func appendNativeSocketPath(root, suffix string) string {
	return zellijquery.AppendNativeSocketPath(root, suffix)
}
