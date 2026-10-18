package couchmessage

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// MaxNativeExitBytes bounds a native-exit report body.
const MaxNativeExitBytes = 1024

// NativeExit is the wrapper's evidence that the native agent child it owns
// exited on its own (pair#657): reaped by the wrapper after Wait, not stopped
// by it. LaunchOrdinal and AgentIdentity name that exact child, so Couch can
// refuse a report about any child but the current one.
type NativeExit struct {
	LaunchOrdinal uint64
	AgentIdentity string
	ExitCode      int
}

func ParseNativeExit(body string) (NativeExit, error) {
	if len(body) > MaxNativeExitBytes {
		return NativeExit{}, errors.New("native exit report exceeds limit")
	}
	var wire struct {
		LaunchOrdinal *uint64
		AgentIdentity *string
		ExitCode      *int
	}
	if err := strictjson.Decode([]byte(body), &wire); err != nil {
		return NativeExit{}, err
	}
	if wire.LaunchOrdinal == nil || wire.AgentIdentity == nil || wire.ExitCode == nil {
		return NativeExit{}, errors.New("native exit report requires launch ordinal, agent identity and exit code")
	}
	report := NativeExit{LaunchOrdinal: *wire.LaunchOrdinal, AgentIdentity: *wire.AgentIdentity, ExitCode: *wire.ExitCode}
	if report.LaunchOrdinal == 0 || report.AgentIdentity == "" || len(report.AgentIdentity) > MaxBindingBytes || report.ExitCode < 0 || report.ExitCode > 255 {
		return NativeExit{}, errors.New("invalid native exit report")
	}
	return report, nil
}

// NativeExitID scopes one report to the exact wrapper incarnation and child
// launch, as AutomaticInputStallID scopes a stall.
func NativeExitID(binding Binding, ordinal uint64) string {
	raw, _ := json.Marshal(binding) // Binding contains only JSON-safe scalar fields.
	digest := sha256.Sum256(raw)
	return fmt.Sprintf("native-exit-%x-%d", digest[:16], ordinal)
}
