package couchcore

import (
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// Recovery ownership uses exact wrapper/readiness generations, independently of
// the ordinary tag-level native binding resolver.
func (f *FakeThreadArtifactCollisionChecker) CheckRecoveryNativeOwner(ctx context.Context, check launcher.RecoveryNativeOwnerCheck) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	excluded := 0
	for _, w := range f.recoveryWrappers {
		if w.Agent != check.Agent {
			continue
		}
		launch, ok := w.Launches[w.ReadyOrdinal]
		if w.PID <= 0 || w.Identity == "" || w.ReadyPID <= 0 || w.ReadyIdentity == "" || w.ReadyParentPID != w.PID || w.ReadyOrdinal == 0 || w.ReadyNonce == "" || !ok || launch.Nonce != w.ReadyNonce || launch.NativeID == "" {
			return errors.New("live wrapper native ownership is unresolved")
		}
		sameSource := w.Address.RepoScope == check.RepoScope && string(w.Address.Tag) == check.Tag && check.SourcePID > 0 && check.SourceIdentity != "" && w.SourcePID == check.SourcePID && w.SourceIdentity == check.SourceIdentity
		bindingMatches := (check.SourceSession == "" || check.SourceSession == w.ReadySession) && (check.SourceNonce == "" || check.SourceNonce == w.ReadyNonce)
		if sameSource && bindingMatches && (w.ParentPID == check.SourcePID || (check.SourceSession != "" && check.SourceNonce != "" && w.ReadySession == check.SourceSession && w.ReadyNonce == check.SourceNonce)) {
			excluded++
			if excluded > 1 {
				return errors.New("multiple source wrappers make native ownership uncertain")
			}
			continue
		}
		if launch.NativeID == check.NativeID {
			return fmt.Errorf("native conversation %s already has a live wrapper", check.NativeID)
		}
	}
	return nil
}

// FakeRecoveryNativeWrapper separates live process/readiness generations from
// tag-level binding observations. Appending a launch never replaces a wrapper.
type FakeRecoveryNativeWrapper struct {
	Address        ThreadAddress
	Agent          string
	PID            int
	Identity       string
	ReadyPID       int
	ReadyIdentity  string
	ReadyOrdinal   uint64
	ReadyNonce     string
	ReadySession   string
	ReadyParentPID int
	ParentPID      int
	SourcePID      int
	SourceIdentity string
	Launches       map[uint64]FakeRecoveryNativeLaunch
}
type FakeRecoveryNativeLaunch struct{ Nonce, NativeID string }

func (f *FakeThreadArtifactCollisionChecker) SetRecoveryNativeWrappers(wrappers ...FakeRecoveryNativeWrapper) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.recoveryWrappers = append([]FakeRecoveryNativeWrapper(nil), wrappers...)
	for i := range f.recoveryWrappers {
		launches := map[uint64]FakeRecoveryNativeLaunch{}
		for ordinal, launch := range wrappers[i].Launches {
			launches[ordinal] = launch
		}
		f.recoveryWrappers[i].Launches = launches
	}
}
