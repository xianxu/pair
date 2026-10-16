package launcher

import (
	"context"

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
