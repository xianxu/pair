package launcher

import (
	"context"
	"os"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func (r OSRuntime) ResolveNativeResume(ctx context.Context, q NativeResumeRequest) (NativeResumeResolution, error) {
	// Configuration scope must be positively resolved by the provider adapter.
	// Unknown effective configuration is not permission to guess the latest ID.
	return ResolveNativeResume(ctx, sessioninventory.NewOSRuntime(os.Getenv("HOME"), r.DataDir), q)
}

func (r OSRuntime) ReserveNativeResume(ctx context.Context, q NativeAdmissionRequest) (string, error) {
	return (NativeAdmissionStore{Home: os.Getenv("HOME")}).Reserve(ctx, q, func(ctx context.Context, owners NativeAdmissionOwners) error {
		return CheckRecoveryNativeOwner(ctx, RecoveryNativeOwnerCheck{DataDir: r.globalDataDir(), Agent: q.Agent, NativeID: q.NativeID, Admissions: owners})
	})
}

func (r OSRuntime) CancelNativeResume(ctx context.Context, grant string) error {
	return (NativeAdmissionStore{Home: os.Getenv("HOME")}).CancelBeforeHandoff(ctx, grant)
}
