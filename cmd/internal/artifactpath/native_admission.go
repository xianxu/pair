package artifactpath

import (
	"errors"
	"path/filepath"
)

// NativeAdmissions is the bounded cross-repository native-writer authority.
func (p PairCachePaths) NativeAdmissions() (string, error) {
	if p.dir == "" {
		return "", errors.New("unresolved Pair cache")
	}
	return filepath.Join(p.dir, "native-admission-v1.json"), nil
}

// NativeAdmissionLease is a permanent lock inode, never an expiring lease file.
func (p PairCachePaths) NativeAdmissionLease() (string, error) {
	if p.dir == "" {
		return "", errors.New("unresolved Pair cache")
	}
	return filepath.Join(p.dir, "native-admission-v1.lock"), nil
}
