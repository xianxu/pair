package artifactpath

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

func validateTerminalSocket(socket string) error {
	if socket == "" || len(socket) > 4096 || !utf8.ValidString(socket) || strings.IndexFunc(socket, unicode.IsControl) >= 0 || !filepath.IsAbs(socket) || filepath.Clean(socket) != socket || socket == string(filepath.Separator) {
		return errors.New("session socket must be a canonical absolute path")
	}
	return nil
}
func (p PairCachePaths) TerminalLease(canonicalSocket string) (string, error) {
	// Layout v2 uses 256 permanent lock inodes. Hash collisions serialize unrelated
	// sockets harmlessly. Changing
	// this layout requires coordinated adoption by every cooperating process.
	if p.dir == "" {
		return "", errors.New("unresolved Pair cache")
	}
	if err := validateTerminalSocket(canonicalSocket); err != nil {
		return "", err
	}
	key := sha256.Sum256([]byte(canonicalSocket))
	return filepath.Join(p.dir, "session-locks-v2", hex.EncodeToString(key[:1])+".lock"), nil
}
