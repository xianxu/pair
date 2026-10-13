package couchcore

import (
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// The artifact fake derives ownership from its existing session/binding world.
func (f *FakeThreadArtifactCollisionChecker) CheckRecoveryNativeOwner(ctx context.Context, check launcher.RecoveryNativeOwnerCheck) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for owner, binding := range f.nativeBindings {
		if owner.Agent != check.Agent {
			continue
		}
		if check.SourcePID > 0 && owner.Address.RepoScope == check.RepoScope && string(owner.Address.Tag) == check.Tag {
			continue
		}
		session, known := f.pairSessions[owner.Address]
		if known && !session.Present {
			continue
		}
		if !known || binding.ObservationIncomplete || binding.Status == sessioninventory.BindingAmbiguous {
			return errors.New("native owner observation incomplete")
		}
		if binding.NativeID == check.NativeID || binding.RequestedNativeID == check.NativeID {
			return fmt.Errorf("native conversation %s already has a live wrapper", check.NativeID)
		}
	}
	return nil
}
