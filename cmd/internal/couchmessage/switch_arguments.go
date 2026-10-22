package couchmessage

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

// SwitchArguments preserves optional argv as a comparable value. Only its zero
// value is absent; a constructed empty value is an explicit empty JSON array.
type SwitchArguments struct {
	canonical string
}

// NewSwitchArguments owns an exact copy of argv. A nil input explicitly selects
// no startup parameters; use the zero value to leave arguments unspecified.
func NewSwitchArguments(argv []string) (SwitchArguments, error) {
	if err := validateSwitchArgumentText(argv); err != nil {
		return SwitchArguments{}, err
	}
	raw, err := json.Marshal(append([]string{}, argv...))
	if err != nil {
		return SwitchArguments{}, err
	}
	return SwitchArguments{canonical: string(raw)}, nil
}

func validateSwitchArgumentText(argv []string) error {
	for _, arg := range argv {
		if !utf8.ValidString(arg) || strings.ContainsRune(arg, 0) {
			return errors.New("switch-agent startup parameters contain invalid UTF-8 or NUL")
		}
	}
	if len(launcher.FormatLaunchParameters(argv)) > couchcore.MaxRepoArgumentBytes {
		return fmt.Errorf("switch-agent startup parameters exceed %d formatted bytes", couchcore.MaxRepoArgumentBytes)
	}
	return nil
}

// IsZero reports that no explicit argv was supplied and lets JSON omit absence.
func (a SwitchArguments) IsZero() bool { return a.canonical == "" }

// Argv returns an independent exact argv and whether it was explicitly supplied.
func (a SwitchArguments) Argv() ([]string, bool) {
	if a.IsZero() {
		return nil, false
	}
	var argv []string
	// The private representation is populated only by the validated constructor.
	_ = json.Unmarshal([]byte(a.canonical), &argv)
	return argv, true
}

// MarshalJSON emits a literal array. Request's omitzero field omits absence.
func (a SwitchArguments) MarshalJSON() ([]byte, error) {
	if a.IsZero() {
		return []byte("null"), nil
	}
	return []byte(a.canonical), nil
}

// UnmarshalJSON rejects null rather than confusing it with an absent field.
func (a *SwitchArguments) UnmarshalJSON(raw []byte) error {
	if len(raw) > 32768 || !utf8.Valid(raw) {
		return errors.New("switch-agent startup parameters exceed transport limit or contain invalid UTF-8")
	}
	argv, err := couchcore.DecodeArgumentArray(raw)
	if err != nil {
		return err
	}
	next, err := NewSwitchArguments(argv)
	if err != nil {
		return err
	}
	*a = next
	return nil
}
