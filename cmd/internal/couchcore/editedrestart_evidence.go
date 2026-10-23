package couchcore

import (
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

type editedRestartNativeObservation struct {
	Kind       EditedRestartNativeKind
	NativeID   string
	Transcript *threadrecord.TranscriptBinding
}

// resolveEditedRestartEvidence observes native availability against the accepted
// anchor. A nil expected ID discovers the preview's anchor; a pointer to empty
// explicitly pins initial absence, so a later conversation cannot replace it.
func (c *Couch) resolveEditedRestartEvidence(ctx context.Context, thread ThreadRecord, expected *string) (editedRestartNativeObservation, error) {
	if err := ctx.Err(); err != nil {
		return editedRestartNativeObservation{}, err
	}
	bound, err := c.bindTranscript(ctx, thread)
	if cancelled := ctx.Err(); cancelled != nil {
		return editedRestartNativeObservation{}, cancelled
	}
	observed, refusal := classifyEditedRestartNative(thread, expected, bound, err)
	// Filename-confirmed and requested-resume ledgers may name only an ID.
	// Complete that usable identity through its artifact owner; probation or
	// an ID alone never proves presence or absence for an edited restart.
	metadata, ok := c.Artifacts.(editedRestartMetadataObserver)
	resolution := bound.Resolution
	located := bound.Observed != nil || bound.Store != nil && bound.Store.Established()
	if !ok || err != nil || located || resolution.ObservationIncomplete || resolution.FreshRequired || resolution.NativeID == "" || !coldResumeAuthorized(resolution) || observed.Kind == EditedRestartNativeChanged {
		return observed, refusal
	}
	if resolution.RequestedNativeID != "" && resolution.RequestedNativeID != resolution.NativeID {
		return editedRestartNativeObservation{}, ErrSwitchResolutionChanged
	}
	observed, err = metadata.ObserveEditedRestartNativeMetadata(ctx, thread, resolution.NativeID)
	if cancelled := ctx.Err(); cancelled != nil {
		return editedRestartNativeObservation{}, cancelled
	}
	if err != nil {
		return editedRestartNativeObservation{}, err
	}
	if observed.NativeID != resolution.NativeID {
		return editedRestartNativeObservation{Kind: EditedRestartNativeChanged, NativeID: observed.NativeID}, ErrSwitchResolutionChanged
	}
	if observed.Kind != EditedRestartNativeFound && observed.Kind != EditedRestartNativeMissing {
		return editedRestartNativeObservation{}, refuseResume(ResumeUnknown, "edited restart native presence is unconfirmed")
	}
	observed.Transcript = threadrecord.CloneTranscriptBinding(observed.Transcript)
	return observed, nil
}

func classifyEditedRestartNative(thread ThreadRecord, expected *string, bound TranscriptBind, probeErr error) (editedRestartNativeObservation, error) {
	unknown := func(err error) (editedRestartNativeObservation, error) {
		if err == nil {
			err = refuseResume(ResumeUnknown, "edited restart native identity is unconfirmed")
		}
		return editedRestartNativeObservation{}, err
	}
	if errors.Is(probeErr, context.Canceled) || errors.Is(probeErr, context.DeadlineExceeded) {
		return unknown(probeErr)
	}
	if bound.Resolution.ObservationIncomplete {
		return unknown(errors.Join(probeErr, refuseResume(ResumeBindingProvisional, "edited restart native observation is incomplete")))
	}
	selected := ""
	if observed := bound.Observed; observed != nil {
		if observed.Agent != threadAgent(thread) || !observed.Established() {
			return unknown(refuseResume(ResumeTranscriptInvalid, "edited restart selected artifact is not an exact agent transcript"))
		}
		if err := observed.ValidateLocation(); err != nil {
			return unknown(fmt.Errorf("edited restart selected artifact: %w", err))
		}
		selected = observed.NativeID
	}
	if bound.Resolution.NativeID != "" {
		if selected != "" && selected != bound.Resolution.NativeID {
			return unknown(refuseResume(ResumeBindingAmbiguous, "edited restart selected artifact and native resolution disagree"))
		}
		selected = bound.Resolution.NativeID
	}
	if expected != nil && selected != "" && selected != *expected {
		return editedRestartNativeObservation{Kind: EditedRestartNativeChanged, NativeID: selected}, errors.Join(ErrSwitchResolutionChanged, probeErr)
	}
	if probeErr != nil {
		if ResumeDiagnosticOf(probeErr) == ResumeTranscriptMissing && bound.Observed != nil && bound.Resolution == (NativeBindingResolution{}) {
			return editedRestartNativeObservation{Kind: EditedRestartNativeMissing, NativeID: selected, Transcript: threadrecord.CloneTranscriptBinding(bound.Observed)}, nil
		}
		// The existing resolver reports positive unbound evidence with its
		// typed refusal. A default/provisional resolution never qualifies.
		if ResumeDiagnosticOf(probeErr) != ResumeBindingUnbound || bound.Resolution.Status != sessioninventory.BindingUnbound {
			return unknown(probeErr)
		}
	}
	switch bound.Resolution.Status {
	case sessioninventory.BindingEstablished:
		if selected == "" {
			return unknown(nil)
		}
		transcript := bound.Store
		if transcript == nil {
			transcript = bound.Observed
		}
		return editedRestartNativeObservation{Kind: EditedRestartNativeFound, NativeID: selected, Transcript: threadrecord.CloneTranscriptBinding(transcript)}, nil
	case sessioninventory.BindingUnbound:
		if bound.Resolution.FreshRequired || bound.Resolution.RequestedNativeID != "" {
			return unknown(probeErr)
		}
		if selected != "" {
			return unknown(nil)
		}
		if expected != nil {
			selected = *expected
		}
		return editedRestartNativeObservation{Kind: EditedRestartNativeMissing, NativeID: selected}, nil
	default:
		return unknown(nil)
	}
}
