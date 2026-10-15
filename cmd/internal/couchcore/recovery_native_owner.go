package couchcore

import (
	"context"
	"errors"

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
	return checker.CheckRecoveryNativeOwner(ctx, check)
}
