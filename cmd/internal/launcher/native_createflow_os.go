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

func (r OSRuntime) CheckNativeResumeCapability(ctx context.Context, exemption NativeLockExemption) error {
	return CheckNativeLockCapability(ctx, r.globalDataDir(), exemption)
}
