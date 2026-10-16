// Package nativelock owns cooperating Pair wrappers' conversation locks.
// Locks protect wrapper lifetime; a wrapper killed with SIGKILL cannot retain
// ownership on behalf of an orphan native process.
package nativelock

import (
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

// Key is an exact validated agent/conversation pair. Its zero value is invalid.
// pair:m5-concept pure
type Key struct{ agent, id string }

func NewKey(agent, id string) (Key, error) {
	if _, err := artifactpath.NativeLockFilename(agent, id); err != nil {
		return Key{}, err
	}
	return Key{agent: agent, id: id}, nil
}
func (k Key) Agent() string { return k.agent }
func (k Key) ID() string    { return k.id }

const maxHolderBytes = 1024

// ParseHolder reads display-only metadata. Malformed or absent metadata cannot
// weaken the kernel lock and never becomes terminal control text.
// pair:m5-concept pure
func ParseHolder(data []byte) string {
	if !validHolder(string(data)) {
		return "another Pair session"
	}
	return string(data)
}
func validHolder(holder string) bool {
	if len(holder) == 0 || len(holder) > maxHolderBytes || !utf8.ValidString(holder) {
		return false
	}
	for _, r := range holder {
		if unicode.IsControl(r) || unicode.IsSpace(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
	}
	return true
}
