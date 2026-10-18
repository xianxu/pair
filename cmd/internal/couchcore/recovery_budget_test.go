package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

// A stateful tool double answers identity queries but never answers worktree
// listing before its caller's unchanged product budget expires.
type recoveryDelayedCatalogIO struct {
	identity WorkspaceIdentity
	budget   time.Duration
}

func (p *recoveryDelayedCatalogIO) Run(ctx context.Context, command ProvisionCommand) ([]byte, error) {
	if command.Program == "sdlc" {
		return json.Marshal(p.identity)
	}
	if command.Program == "git" && strings.Join(command.Args, " ") == "worktree list --porcelain -z" {
		p.budget = command.Timeout
		// Simulate the tool deadline firing: elapsed host time is irrelevant
		// to whether callers preserve state after this response.
		return nil, context.DeadlineExceeded
	}
	return nil, fmt.Errorf("unexpected catalog command: %+v", command)
}

type recoveryUnavailableCatalog struct {
	*SlotCatalogFake
	err error
}

func (c recoveryUnavailableCatalog) Discover(context.Context, string) (SlotRepository, error) {
	return SlotRepository{}, c.err
}

func TestRecoveryFreshCatalogDeadlinePreservesRetainedState(t *testing.T) {
	t.Cleanup(func() {
		if t.Failed() {
			t.Log(testwait.HostState())
		}
	})
	env, local := slotRecoverySemanticFixture(t)
	catalog := env.Couch.Slots.(*SlotCatalogFake)
	io := &recoveryDelayedCatalogIO{identity: catalog.Repositories[local.slot.PrimaryRoot].Identity}
	_, err := NewOSSlotCatalog(io).Discover(t.Context(), local.slot.PrimaryRoot)
	var deadline *ProvisionCommandError
	if !errors.As(err, &deadline) || !deadline.TimedOut || io.budget != 5*time.Second {
		t.Fatalf("catalog lost unchanged probe deadline: error=%v budget=%s", err, io.budget)
	}
	env.Couch.Slots = recoveryUnavailableCatalog{SlotCatalogFake: catalog, err: err}
	path := filepath.Join(local.slot.WorktreeRoot, "foreign-retained-state")
	want := []byte("foreign state must stay unchanged")
	if err := os.WriteFile(path, want, 0600); err != nil {
		t.Fatal(err)
	}
	preparations := 0
	env.Couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
		preparations++
		return ProvisionResult{}, errors.New("unexpected repair")
	})
	_, err = env.Couch.RecoverSelection(t.Context(), RecoverySelection{Target: ResumeTarget{Path: local.slot.WorktreeRoot}, Kind: RecoveryFresh})
	got, readErr := os.ReadFile(path)
	if !errors.As(err, &deadline) || preparations != 0 || len(env.Runner.children) != 0 || readErr != nil || !bytes.Equal(got, want) {
		t.Fatalf("deadline recovery changed state: err=%v prepares=%d launches=%d bytes=%q read=%v", err, preparations, len(env.Runner.children), got, readErr)
	}
}
