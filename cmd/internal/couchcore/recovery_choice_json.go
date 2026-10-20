package couchcore

import (
	"encoding/json"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// MarshalJSON keeps capture scores visible, including meaningful zero scores,
// while native-only evidence has no screen comparison fields.
func (c RecoveryChoice) MarshalJSON() ([]byte, error) {
	type wire RecoveryChoice
	if c.SourceKind != sessioninventory.RecoverySourceNative {
		return json.Marshal(wire(c))
	}
	return json.Marshal(struct {
		wire
		TailMatch *float64 `json:"tail_match,omitempty"`
		Matched   *int     `json:"matched,omitempty"`
		Total     *int     `json:"total,omitempty"`
	}{wire: wire(c)})
}
