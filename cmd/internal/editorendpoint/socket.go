package editorendpoint

import (
	"errors"
	"os"
	"path/filepath"
)

// ValidateEndpointSocket observes private endpoint metadata; it never removes
// a socket or creates an editor-owned directory. Native RPC and process birth
// verification establish the live attachment separately.
func ValidateEndpointSocket(path string) error {
	if !canonicalPath(path) || len(path) > 103 {
		return errors.New("endpoint socket must be a canonical path of at most 103 bytes")
	}
	parent := filepath.Dir(path)
	physical, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return err
	}
	dir, err := os.Lstat(parent)
	if err != nil {
		return err
	}
	if physical != parent || !dir.IsDir() || !owned(dir) || dir.Mode().Perm() != 0700 {
		return errors.New("endpoint socket parent must be a physical owned 0700 directory")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 || !owned(info) {
		return errors.New("endpoint must be an owned Unix socket")
	}
	return nil
}
