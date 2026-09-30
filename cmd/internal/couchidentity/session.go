package couchidentity

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SessionBinding describes a terminal incarnation, independently of a native agent transcript.
type SessionBinding struct {
	C          uint64 `json:"c,omitempty"`
	M          uint64 `json:"m,omitempty"`
	Name       string `json:"name"`
	ScopeKey   string `json:"scope_key"`
	Tag        string `json:"tag"`
	StartNonce string `json:"start_nonce"`
	Legacy     bool   `json:"legacy,omitempty"`
}

func (b SessionBinding) Validate() error {
	if b.Name == "" || b.ScopeKey == "" || b.Tag == "" || b.StartNonce == "" {
		return errors.New("incomplete terminal session binding")
	}
	if err := ValidateStartNonce(b.StartNonce); err != nil {
		return err
	}
	if err := ValidateSessionName(b.Name); err != nil {
		return err
	}
	if b.Legacy {
		if b.C != 0 || b.M != 0 {
			return errors.New("legacy session binding cannot invent counters")
		}
		return nil
	}
	name, err := FormatSessionName(b.C, b.M)
	if err != nil {
		return err
	}
	if name != b.Name {
		return errors.New("terminal session binding name disagrees with counters")
	}
	return nil
}

// ValidateSessionName accepts retained legacy names and allocated terminal names.
func ValidateSessionName(name string) error {
	if !strings.HasPrefix(name, "📁") && !strings.HasPrefix(name, "pair-") {
		return errors.New("terminal session name is outside Pair namespace")
	}
	if name == "📁" || name == "pair-" || unsafeSessionArgument(name) {
		return errors.New("invalid terminal session name")
	}
	return nil
}
func ValidateStartNonce(nonce string) error {
	if nonce == "" || unsafeSessionArgument(nonce) {
		return errors.New("invalid terminal start nonce")
	}
	return nil
}
func unsafeSessionArgument(value string) bool {
	return len(value) > 256 || !utf8.ValidString(value) || strings.ContainsAny(value, "/\\") || strings.IndexFunc(value, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0
}
