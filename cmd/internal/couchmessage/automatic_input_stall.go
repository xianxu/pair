package couchmessage

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// AutomaticInputRecoveryHint names Pair's submit chord, not native Return.
const AutomaticInputRecoveryHint = "press Alt+Enter or clear it"

const (
	AutomaticInputStallUnsubmittedComposer        = "unsubmitted-composer"
	AutomaticInputStallNoRender                   = "no-render"
	AutomaticInputStallUnrecognizedComposer       = "unrecognized-composer"
	MaxAutomaticInputStallBytes                   = 2048
	MaxAutomaticInputStallWaitMillis        int64 = 604800000
)

// AutomaticInputStall is a bounded observation, never input authority or worker-supplied notification text.
type AutomaticInputStall struct {
	Sequence     uint64
	Cause        string
	WaitMillis   int64
	Fence, Epoch uint64
	Released     bool
}

func ParseAutomaticInputStall(body string) (AutomaticInputStall, error) {
	var report AutomaticInputStall
	if len(body) > MaxAutomaticInputStallBytes {
		return report, errors.New("automatic input stall exceeds limit")
	}
	// Pointers distinguish absent/null fields from legitimate zero evidence.
	var wire struct {
		Sequence     *uint64
		Cause        *string
		WaitMillis   *int64
		Fence, Epoch *uint64
		Released     *bool
	}
	if err := strictjson.Decode([]byte(body), &wire); err != nil {
		return report, err
	}
	if wire.Sequence == nil || wire.Cause == nil || wire.WaitMillis == nil || wire.Fence == nil || wire.Epoch == nil || wire.Released == nil {
		return report, errors.New("automatic input stall requires all evidence fields")
	}
	report = AutomaticInputStall{Sequence: *wire.Sequence, Cause: *wire.Cause, WaitMillis: *wire.WaitMillis, Fence: *wire.Fence, Epoch: *wire.Epoch, Released: *wire.Released}
	if report.Sequence == 0 || report.WaitMillis < 0 || report.WaitMillis > MaxAutomaticInputStallWaitMillis {
		return AutomaticInputStall{}, errors.New("invalid automatic input stall sequence or wait")
	}
	if report.Cause != AutomaticInputStallNoRender && report.Cause != AutomaticInputStallUnrecognizedComposer && report.Cause != AutomaticInputStallUnsubmittedComposer {
		return AutomaticInputStall{}, errors.New("invalid automatic input stall cause")
	}
	if report.Released && report.Cause != AutomaticInputStallNoRender {
		return AutomaticInputStall{}, errors.New("occupied or unrecognized composer cannot release automatic input")
	}
	return report, nil
}

// AutomaticInputStallID scopes the sequence to the exact wrapper incarnation,
// including PID reuse and launch replacement, as EndpointSocket does.
func AutomaticInputStallID(binding Binding, sequence uint64) string {
	raw, _ := json.Marshal(binding) // Binding contains only JSON-safe scalar fields.
	digest := sha256.Sum256(raw)
	return fmt.Sprintf("automatic-input-%x-%d", digest[:16], sequence)
}
