package launcher

import (
	"context"
	"os"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func (r OSRuntime) ResolveNativeResume(ctx context.Context, q NativeResumeRequest) (NativeResumeResolution, error) {
	// Indirect Codex selection belongs to native, including direct callers of
	// this boundary. Refuse before observing homes, transcripts or config.
	selector := q.Selector
	selector.Agent = q.Agent
	if CodexIndirectResumeCompatibility(selector) {
		return NativeResumeResolution{Status: sessioninventory.NativeSelectorUnsupported, Reason: "Codex indirect resume selection remains native"}, nil
	}
	native, err := r.nativeInventory(q.Agent, q.CWD)
	if err != nil {
		return NativeResumeResolution{Status: sessioninventory.NativeSelectorIncomplete, Reason: "native transcript scope unavailable"}, nil
	}
	return ResolveNativeResume(ctx, native, q)
}

func (r OSRuntime) ReserveNativeResume(ctx context.Context, q NativeAdmissionRequest) (string, error) {
	return (NativeAdmissionStore{Home: os.Getenv("HOME")}).Reserve(ctx, q, func(ctx context.Context, owners NativeAdmissionOwners) error {
		return CheckRecoveryNativeOwner(ctx, RecoveryNativeOwnerCheck{DataDir: r.globalDataDir(), Agent: q.Agent, NativeID: q.NativeID, Admissions: owners})
	})
}

func (r OSRuntime) CancelNativeResume(ctx context.Context, grant string) error {
	return (NativeAdmissionStore{Home: os.Getenv("HOME")}).CancelBeforeHandoff(ctx, grant)
}
