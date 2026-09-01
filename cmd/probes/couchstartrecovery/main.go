// couchstartrecovery proves M2's two real-process restart outcomes against the
// installed pair-launch-helper and kernel process identities.
//
//	go run ./cmd/probes/couchstartrecovery [./bin/pair-launch-helper]
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func main() {
	helper := "./bin/pair-launch-helper"
	if len(os.Args) > 1 {
		helper = os.Args[1]
	}
	absHelper, err := filepath.Abs(helper)
	if err != nil {
		fail("resolve helper", err)
	}

	if err := runScenario(absHelper, false); err != nil {
		fail("dead unregistered helper rolls back", err)
	}
	fmt.Println("PASS  dead unregistered helper rolls back exact nonce")
	if err := runScenario(absHelper, true); err != nil {
		fail("established live helper promotes", err)
	}
	fmt.Println("PASS  established live helper promotes after restart")
}

func runScenario(helper string, establish bool) error {
	root, err := os.MkdirTemp("", "pair-couch-m2-probe-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)

	namespace, err := couchcore.ResolveCouchNamespace(filepath.Join(root, "store"), root)
	if err != nil {
		return err
	}
	dataDir := filepath.Join(root, "pair-data")
	address := couchcore.ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0123456789abcdef"}
	artifacts := couchcore.NewScopedThreadArtifactCollisionChecker(dataDir)
	claim, err := artifacts.Claim(address)
	if err != nil {
		return err
	}
	defer claim.Release()

	runner := couchcore.ExecRunner{LaunchHelper: helper}
	handle, err := runner.StartBlocked(context.Background(), root, []string{"sh", "-c", "sleep 30"}, nil, 3*time.Second)
	if err != nil {
		return err
	}
	defer func() {
		if handle.Alive() {
			_ = handle.Signal(os.Kill)
		}
		_ = handle.Wait()
	}()

	proc := couchcore.OSProcOps{}
	owner, err := proc.Current()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	record := couchcore.ThreadRecord{
		SchemaVersion: couchcore.ThreadSchemaVersion,
		Address:       address,
		StartingPath:  root,
		WorkingPath:   root,
		CreatedAt:     now,
		Revision:      1,
		Incarnations:  []couchcore.ThreadIncarnation{{State: couchcore.IncarnationCreating, StartedAt: now}},
	}
	record, err = couchcore.AdvanceStartTransaction(record, couchcore.StartEvent{
		Kind: couchcore.StartClaimed, Nonce: "start-0123456789abcdef",
		Owner: couchcore.SupervisorOwner{PID: owner.PID, Identity: owner.Identity},
	})
	if err != nil {
		return err
	}
	record, err = couchcore.AdvanceStartTransaction(record, couchcore.StartEvent{
		Kind: couchcore.StartHelperRecorded, Nonce: "start-0123456789abcdef",
		Helper: couchcore.ProcessIdentity{PID: handle.PID(), Identity: handle.Identity()},
	})
	if err != nil {
		return err
	}
	store := couchcore.NewThreadStore(namespace)
	created, err := store.CreateThread(record)
	if err != nil {
		return err
	}

	if establish {
		if err := handle.Acknowledge(); err != nil {
			return err
		}
		if err := launcher.EnsureThreadAddressForPair(dataDir, launcher.RepoScope{Key: address.RepoScope}, string(address.Tag), true); err != nil {
			return err
		}
	} else {
		if err := handle.Cancel(); err != nil {
			return err
		}
		if code := handle.Wait(); code == 0 {
			return errors.New("cancelled helper exited successfully")
		}
	}

	_, err = couchcore.New(
		namespace, couchcore.NewFakeRunner(), couchcore.NewFakePathOps(nil), couchcore.NewFakeGit(nil), proc,
		couchcore.NewStore(namespace.Dir()), couchcore.SystemClock{}, couchcore.NewFixedIDGen("probe"),
		couchcore.NewFakePolicyResolver(), bytes.NewReader(make([]byte, 64)), artifacts,
	)
	if err != nil {
		return err
	}
	got, err := store.GetThread(address)
	if !establish {
		if !errors.Is(err, couchcore.ErrThreadNotFound) {
			return fmt.Errorf("rollback lookup = %+v, %w", got, err)
		}
		return nil
	}
	if err != nil {
		return err
	}
	if got.Revision != created.Revision+1 || len(got.Incarnations) != 1 || got.Incarnations[0].State != couchcore.IncarnationLive || got.Incarnations[0].Start != nil {
		return fmt.Errorf("promoted record = %+v", got)
	}
	return nil
}

func fail(name string, err error) {
	fmt.Fprintf(os.Stderr, "FAIL  %s: %v\n", name, err)
	os.Exit(1)
}
