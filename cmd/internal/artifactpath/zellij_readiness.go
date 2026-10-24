package artifactpath

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
)

// ZellijNamespaceLease is a permanent, bounded family separate from session
// handoff locks. Colliding namespaces serialize; online cleanup never unlinks it.
func (p PairCachePaths) ZellijNamespaceLease(namespace string) (string, error) {
	if p.dir == "" {
		return "", errors.New("unresolved Pair cache")
	}
	if err := validateTerminalSocket(namespace); err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(namespace))
	return filepath.Join(p.dir, "zellij-scan-locks-v1", hex.EncodeToString(hash[:1])+".lock"), nil
}

// ZellijReadyDirectory contains only the current positive marker per session.
func (p PairCachePaths) ZellijReadyDirectory(namespace string) (string, error) {
	if p.dir == "" {
		return "", errors.New("unresolved Pair cache")
	}
	if err := validateTerminalSocket(namespace); err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(namespace))
	return filepath.Join(p.dir, "zellij-ready-v1", hex.EncodeToString(hash[:])), nil
}

func (p PairCachePaths) ZellijReadyMarker(namespace, session string) (string, error) {
	if err := validateCacheSession(session); err != nil {
		return "", err
	}
	dir, err := p.ZellijReadyDirectory(namespace)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(session))
	return filepath.Join(dir, hex.EncodeToString(hash[:])+".json"), nil
}
