package couchcore

import (
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

type recoveryNativeOwnerChecker interface {
	CheckRecoveryNativeOwner(context.Context, launcher.RecoveryNativeOwnerCheck) error
}

func (c ScopedThreadArtifactCollisionChecker) CheckRecoveryNativeOwner(ctx context.Context, check launcher.RecoveryNativeOwnerCheck) error {
	check.DataDir = c.GlobalDataDir
	return launcher.CheckRecoveryNativeOwner(ctx, check)
}

func (c *Couch) checkRecoveryNativeOwner(ctx context.Context, record ThreadRecord, choice RecoveryChoice, allowSource bool) error {
	if choice.NativeID == "" {
		return ctx.Err()
	}
	checker, ok := c.Artifacts.(recoveryNativeOwnerChecker)
	if !ok {
		return errors.New("native conversation ownership observer unavailable")
	}
	check := launcher.RecoveryNativeOwnerCheck{RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag), Agent: choice.Agent, NativeID: choice.NativeID}
	if allowSource {
		if binding := recordSessionBinding(record); binding != nil {
			check.SourceSession, check.SourceNonce = binding.Name, binding.StartNonce
		}
		for _, inc := range record.Incarnations {
			if observeExactProcess(c.Proc, ProcessIdentity{PID: inc.PID, Identity: inc.Identity}) == Live {
				if check.SourcePID != 0 {
					return errors.New("multiple source incarnations make native ownership uncertain")
				}
				check.SourcePID, check.SourceIdentity = inc.PID, inc.Identity
			}
		}
	}
	err := checker.CheckRecoveryNativeOwner(ctx, check)
	var legacy *launcher.LegacyNativeOwnerError
	if !errors.As(err, &legacy) {
		return err
	}
	snapshot, previewErr := c.Threads.PreviewSnapshotContext(ctx)
	if previewErr != nil {
		return err
	}
	var friendly string
	for _, slot := range snapshot.Slots {
		if slot.Err != nil || slot.Address.RepoScope != legacy.Owner.RepoScope || string(slot.Address.Tag) != legacy.Owner.Tag {
			continue
		}
		if friendly != "" || slot.Identity.validateLocation() != nil {
			return err
		}
		friendly = fmt.Sprintf("%s:%d", slot.Identity.Repo, slot.Identity.Number)
	}
	if friendly == "" {
		return err
	}
	return fmt.Errorf("%w; relaunch %s to migrate: couch --relaunch %s --confirm", err, friendly, friendly)
}
