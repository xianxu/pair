package couchcore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

type relaunchMigrationKey struct{}
type relaunchMigration struct{ Agent, NativeID string }

func isRelaunchMigration(ctx context.Context, choice RecoveryChoice) bool {
	source, ok := ctx.Value(relaunchMigrationKey{}).(relaunchMigration)
	return ok && source.Agent == choice.Agent && source.NativeID != "" && source.NativeID == choice.NativeID
}

type recoveryNativeLockChecker interface {
	CheckNativeResumeCapability(context.Context, launcher.NativeLockExemption) error
}

func (c ScopedThreadArtifactCollisionChecker) CheckNativeResumeCapability(ctx context.Context, exemption launcher.NativeLockExemption) error {
	return launcher.CheckNativeLockCapability(ctx, c.GlobalDataDir, exemption)
}
func (c *Couch) checkRecoveryNativeLock(ctx context.Context, choice RecoveryChoice) error {
	if choice.NativeID == "" || isRelaunchMigration(ctx, choice) {
		return ctx.Err()
	}
	checker, ok := c.Artifacts.(recoveryNativeLockChecker)
	if !ok {
		return errors.New("native lock capability observer unavailable")
	}
	return checker.CheckNativeResumeCapability(ctx, launcher.NativeLockExemption{})
}
func (f *FakeThreadArtifactCollisionChecker) CheckNativeResumeCapability(ctx context.Context, _ launcher.NativeLockExemption) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.legacyNativeLockSlots) > 0 {
		return fmt.Errorf("relaunch older slots first: %s", strings.Join(f.legacyNativeLockSlots, ", "))
	}
	return nil
}
