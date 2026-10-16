// Package nativeid is the one rule for a native agent conversation ID, shared
// by every boundary that carries one (pair#614): the conversation lock key and
// the watcher's binding notice. A notice that passes can always be locked.
package nativeid

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxLen bounds an ID in bytes.
const MaxLen = 512

// Validate reports whether id can name a conversation: nonempty, at most
// MaxLen bytes of valid UTF-8, not a path element, and free of control, space
// and format characters.
// pair:m5-concept pure
func Validate(id string) error {
	if len(id) == 0 || len(id) > MaxLen || !utf8.ValidString(id) || id == "." || id == ".." || strings.ContainsAny(id, "/\\") {
		return errors.New("invalid native conversation ID")
	}
	for _, r := range id {
		if unicode.IsControl(r) || unicode.IsSpace(r) || unicode.Is(unicode.Cf, r) {
			return errors.New("invalid native conversation ID")
		}
	}
	return nil
}
