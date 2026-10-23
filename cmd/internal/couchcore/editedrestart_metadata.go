package couchcore

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// observeEditedRestartNativeMetadata locates only a previously trusted native
// ID. Metadata completes its file location; it never discovers a replacement ID
// or grants authority from native contents. The existing observer/selector may
// sort metadata; the fixed probe count is not a claim of linear CPU cost.
func observeEditedRestartNativeMetadata(ctx context.Context, rt sessioninventory.Runtime, thread ThreadRecord, nativeID string) (editedRestartNativeObservation, error) {
	unknown := func(reason string) (editedRestartNativeObservation, error) {
		return editedRestartNativeObservation{}, refuseResume(ResumeUnknown, reason)
	}
	if ctx == nil {
		return unknown("edited restart native metadata requires context")
	}
	if err := ctx.Err(); err != nil {
		return editedRestartNativeObservation{}, err
	}
	agent := sessioninventory.Agent(threadAgent(thread))
	if rt == nil || nativeID == "" || !slices.Contains(sessioninventory.SupportedAgents(), agent) {
		return unknown("edited restart native metadata requires a supported agent and exact accepted ID")
	}
	if len(rt.NativeRoots(agent)) == 0 {
		return unknown("edited restart native storage roots are unavailable")
	}
	if err := ctx.Err(); err != nil {
		return editedRestartNativeObservation{}, err
	}
	observations, diagnostics := sessioninventory.ObserveAgentMetadata(rt, agent)
	if err := ctx.Err(); err != nil {
		return editedRestartNativeObservation{}, err
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Code != sessioninventory.DiagnosticStorageAbsent {
			return unknown(fmt.Sprintf("edited restart native metadata is incomplete (%s): %s", diagnostic.Code, diagnostic.Detail))
		}
	}
	target := sessioninventory.SelectTargetWork(sessioninventory.TargetRequest{Mode: sessioninventory.TargetExplicitResume, Agent: agent, NativeID: nativeID}, observations)
	if err := ctx.Err(); err != nil {
		return editedRestartNativeObservation{}, err
	}
	if len(target.Eligible) == 0 {
		return editedRestartNativeObservation{Kind: EditedRestartNativeMissing, NativeID: nativeID}, nil
	}
	if len(target.Eligible) != 1 {
		return unknown("edited restart native ID has multiple matching artifacts")
	}
	artifact := target.Eligible[0].Entry.Artifact
	located := &threadrecord.TranscriptBinding{Agent: string(agent), NativeID: nativeID, StorageRoot: artifact.StorageRoot, RelativePath: artifact.RelativePath, Source: transcriptSource(thread.Transcript)}
	if err := located.ValidateLocation(); err != nil {
		return editedRestartNativeObservation{}, fmt.Errorf("edited restart native artifact location: %w", err)
	}
	_, err := rt.StatFile(artifact)
	if cancelled := ctx.Err(); cancelled != nil {
		return editedRestartNativeObservation{}, cancelled
	}
	out := editedRestartNativeObservation{Kind: EditedRestartNativeFound, NativeID: nativeID, Transcript: threadrecord.CloneTranscriptBinding(located)}
	if errors.Is(err, fs.ErrNotExist) {
		out.Kind = EditedRestartNativeMissing
		return out, nil
	}
	if err != nil {
		return editedRestartNativeObservation{}, fmt.Errorf("stat edited restart native artifact: %w", err)
	}
	return out, nil
}
