package couchcore

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"testing"
)

func TestRecoverParkSlotAdmissionBindsExactTransactionAndSettledRepeat(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	record := validThreadRecord(t)
	record.Address.RepoScope = scope.Key
	record.StartingPath, record.WorkingPath = local.slot.WorktreeRoot, local.slot.WorktreeRoot
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	record.LatestLaunchProfile = &profile
	record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "park-owner", State: IncarnationLive, RepoIdentity: local.slot.RepoIdentity, LaunchProfile: &profile}}
	created, err := local.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	identity := ParkIdentity{Nonce: "original-park", Address: created.Address, PID: 42, ProcessIdentity: "park-owner"}
	begun, err := local.BeginPark(created.Address, created.Revision, identity)
	if err != nil {
		t.Fatal(err)
	}
	target := WorkspaceReference{Repo: local.slot.Repo, Number: local.slot.Number}.String()
	call, _, err := env.Couch.PrepareSlotOperation(context.Background(), "recover-park", target, LiveRestartOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if call.Args["repo-scope"] != created.Address.RepoScope || call.Args["tag"] != string(created.Address.Tag) || call.Args["park-nonce"] != identity.Nonce || call.Args["path"] != "" {
		t.Fatalf("unbound %+v", call)
	}
	rows := ProjectActionableThreads(ThreadProjectionInput{Records: []ThreadRecord{begun}})
	if len(rows) != 1 || !rows[0].HasOpenPark || rows[0].ParkNonce != identity.Nonce {
		t.Fatalf("park projection %+v", rows)
	}
	if _, err := local.FinalizePark(created.Address, begun.Revision, identity, 1, env.Now); err != nil {
		t.Fatal(err)
	}
	call, _, err = env.Couch.PrepareSlotOperation(context.Background(), "recover-park", target, LiveRestartOptions{})
	if err != nil || call.Args["tag"] != string(created.Address.Tag) || call.Args["park-nonce"] != "" {
		t.Fatalf("settled repeat: %+v %v", call, err)
	}
}

func TestRecoverParkExpectedEmptyRefusesNewTransaction(t *testing.T) {
	env, before := interruptedParkEnv(t, false)
	_, err := env.Couch.RecoverParkExpected(context.Background(), before.Address, "")
	if err == nil {
		t.Fatal("known-absent binding recovered a newly opened park")
	}
	after, readErr := env.Couch.Threads.GetThread(before.Address)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if after.Revision != before.Revision || after.Park == nil || after.Park.Identity != before.Park.Identity {
		t.Fatalf("replacement transaction mutated: before=%+v after=%+v", before.Park, after.Park)
	}
}

func TestRecoverParkDispatchRequiresExactAddress(t *testing.T) {
	valid := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0123456789abcdef"}
	for _, args := range []map[string]string{{"repo-scope": "", "tag": string(valid.Tag)}, {"repo-scope": valid.RepoScope, "tag": "../unsafe"}, {"repo-scope": valid.RepoScope, "tag": ""}, {"repo-scope": "../invalid", "tag": string(valid.Tag)}} {
		if _, err := exactParkRecoveryAddress(args); err == nil {
			t.Fatalf("accepted malformed exact address %v", args)
		}
	}
	got, err := exactParkRecoveryAddress(map[string]string{"repo-scope": valid.RepoScope, "tag": string(valid.Tag)})
	if err != nil || got != valid {
		t.Fatalf("exact address %+v %v", got, err)
	}
}
