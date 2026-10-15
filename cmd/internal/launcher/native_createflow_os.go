package launcher

import (
	"context"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func (r OSRuntime) ResolveNativeResume(ctx context.Context, q NativeResumeRequest) (NativeResumeResolution, error) {
	native, err := r.nativeInventory(q.Agent, q.CWD)
	if err != nil {
		return NativeResumeResolution{Status: sessioninventory.NativeSelectorIncomplete, Reason: "native transcript scope unavailable"}, nil
	}
	if q.Agent == "codex" {
		if q.Selector.Kind == NativeResumeLatest {
			invocation := q.Selector.CodexConfig
			invocation.StartupCWD = q.CWD
			invocation.CWD = q.CWD
			if q.Selector.CWD != "" {
				invocation.CWD = q.Selector.CWD
				if !filepath.IsAbs(invocation.CWD) {
					invocation.CWD = filepath.Join(q.CWD, invocation.CWD)
				}
			}
			reader := sessioninventory.CodexConfigReader{}
			if r.codexConfigReader != nil {
				reader = *r.codexConfigReader
			}
			projection, err := reader.Read(ctx, invocation)
			if err != nil {
				return NativeResumeResolution{}, err
			}
			if projection.Status != sessioninventory.CodexConfigComplete {
				return NativeResumeResolution{Status: sessioninventory.NativeSelectorUnsupported, Reason: string(projection.Reason)}, nil
			}
			native, err = native.WithCodexScope(projection.Scope)
			if err != nil {
				return NativeResumeResolution{Status: sessioninventory.NativeSelectorIncomplete, Reason: "Codex physical scope unavailable"}, nil
			}
			q.Config.CodexScope = &projection.Scope
			q.Config.Provider = projection.Scope.Provider
			q.Config.ProviderResolved = true
			q.Config.SQLiteHome = projection.Scope.SQLiteHome
			q.Config.SQLiteHomeResolved = true

		}
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
