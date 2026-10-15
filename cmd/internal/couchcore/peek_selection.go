package couchcore

import (
	"context"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

type PeekSelection struct {
	Previous, All, Transcripts bool
	Lines                      int
}

type PeekSource struct {
	Agent, Token string
	Binding      *couchidentity.SessionBinding
	Capture      *threadrecord.RecoveryCapture
}

// SelectPeekSource never infers history from filenames or the current native
// owner. A missing previous descriptor is an unavailable source, not a fallback.
func SelectPeekSource(record ThreadRecord, previous bool) (PeekSource, error) {
	if !previous {
		return PeekSource{Agent: RecordAgent(record), Binding: cloneSessionBinding(record.SessionBinding)}, nil
	}
	if intent := retainedRecoveryIntent(record); intent != nil && intent.Kind() != threadrecord.RecoveryInferred {
		if err := threadrecord.ValidateRecoveryIntent(*intent, toPersistedThreadAddress(record.Address)); err != nil {
			return PeekSource{}, err
		}
		if intent.Orientation == nil || intent.Orientation.Capture == nil {
			return PeekSource{}, fmt.Errorf("previous recovery screen is unavailable")
		}
		capture := *intent.Orientation.Capture
		if capture.Token == "" {
			return PeekSource{}, fmt.Errorf("previous recovery screen has no retained token")
		}
		return PeekSource{Agent: intent.Agent, Token: capture.Token, Capture: &capture}, nil
	}
	// Recovery carries an exact immutable screen independently of park proof.
	// The original source survives later output and choices on this same tag.
	if context := record.RecoveryContext; context != nil {
		if err := threadrecord.ValidateRecoveryContext(*context, toPersistedThreadAddress(record.Address)); err != nil {
			return PeekSource{}, err
		}
		if context.Capture.Token == "" {
			return PeekSource{}, fmt.Errorf("recovery screen has no retained capture token")
		}
		return PeekSource{Agent: RecordAgent(record), Token: context.Capture.Token}, nil
	}
	p := threadrecord.ClonePreviousSession(record.PreviousSession)
	if p == nil {
		return PeekSource{}, fmt.Errorf("previous session descriptor is unavailable")
	}
	if err := p.Validate(toPersistedThreadAddress(record.Address)); err != nil {
		return PeekSource{}, err
	}
	source := PeekSource{Agent: p.Agent}
	if p.Capture != nil {
		source.Token = p.Capture.Token
	}
	return source, nil
}

// TerminalRecording carries the exact source paths even when replay fails.
type TerminalRecording struct {
	Lines       []string
	Raw, Events string
}
type SlotCaptureReader func(ThreadAddress, string, string, int) (TerminalRecording, error)

// SlotRetainedCaptureReader renders display-only retained orientation evidence.
type SlotRetainedCaptureReader func(context.Context, ThreadAddress, string, threadrecord.RecoveryCapture, int) (TerminalRecording, error)

// retainedRecoveryIntent gives the in-flight exact source precedence over a
// completed attempt, matching the lifetime of terminal orientation evidence.
func retainedRecoveryIntent(record ThreadRecord) *threadrecord.RecoveryIntent {
	for _, inc := range record.Incarnations {
		if inc.Start != nil && inc.Start.RecoveryIntent != nil {
			return inc.Start.RecoveryIntent
		}
	}
	return record.RecoveryIntent
}
