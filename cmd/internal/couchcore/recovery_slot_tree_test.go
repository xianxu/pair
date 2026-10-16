package couchcore

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

// slotTreeFixture is a legacy slot: terminal attribution fails, but Couch's
// registry knows its live wrapper (500) with an agent (501) and a session
// watcher (502) below it, and its record names its own zellij session.
func slotTreeFixture(t *testing.T) *recoveryExecutionFixture {
	t.Helper()
	f := newRecoveryExecutionFixture(t)
	updated, err := f.Couch.Threads.updateExistingThread(f.record.Address, f.record.Revision, func(next *ThreadRecord) error {
		next.SessionBinding = &couchidentity.SessionBinding{C: 1, M: 7, Name: "📁1-7", ScopeKey: next.Address.RepoScope, Tag: string(next.Address.Tag), StartNonce: "start-0102030405060708"}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	f.record = updated
	f.Artifacts.SetPairSession(f.record.Address, "📁1-7", true)
	f.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
		return RecoveryRetirement{}, errors.New("legacy readiness has no child birth")
	}
	for pid, identity := range map[int]string{500: "wrapper", 501: "codex", 502: "watch", 600: "unrelated"} {
		f.Proc.Set(pid, identity)
	}
	f.Proc.DiesOn = map[int]os.Signal{500: TermSignal, 501: TermSignal, 502: TermSignal}
	f.Couch.SlotWrapper = func(address ThreadAddress) (ProcessIdentity, bool) {
		return ProcessIdentity{PID: 500, Identity: "wrapper"}, address == f.record.Address
	}
	f.Couch.ProcessTable = func() ([]procutil.Process, error) {
		return []procutil.Process{{PID: 500, PPID: 1, Identity: "wrapper", Name: "pair"}, {PID: 501, PPID: 500, Identity: "codex", Name: "codex"}, {PID: 502, PPID: 500, Identity: "watch", Name: "pair"}, {PID: 600, PPID: 1, Identity: "unrelated"}}, nil
	}
	f.Couch.sleep = func(time.Duration) {}
	return f
}

func (f *recoveryExecutionFixture) rebootFresh(t *testing.T) (RecoveryResumeResult, []RecoveryStopProcess, error) {
	t.Helper()
	selection := RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryFresh}
	result, err := f.Couch.RecoverSelection(t.Context(), selection)
	var required *RecoveryConfirmationRequired
	if !errors.As(err, &required) {
		return result, nil, err
	}
	result, err = f.Couch.ConfirmRecovery(t.Context(), required.Offer.Token)
	return result, required.Offer.Stops, err
}

// pair#607: a reboot of a legacy slot stops everything the slot owns.
func TestRebootLegacySlotStopsRegisteredWrapperTreeAndSession(t *testing.T) {
	f := slotTreeFixture(t)
	result, stops, err := f.rebootFresh(t)
	if err != nil || result.Start.Handle == nil {
		t.Fatalf("reboot: result=%+v err=%v", result, err)
	}
	roles := []string{}
	for _, stop := range stops {
		roles = append(roles, stop.Role)
	}
	for _, want := range []string{"wrapper", "wrapper child codex", "wrapper child pair", "zellij session 📁1-7"} {
		if !strings.Contains(strings.Join(roles, ","), want) {
			t.Fatalf("confirmation stop list %v lacks %q", roles, want)
		}
	}
	for _, pid := range []int{500, 501, 502} {
		if f.Proc.Exists(pid) != Dead {
			t.Fatalf("pid %d survived the reboot", pid)
		}
	}
	if f.Proc.Exists(600) != Live || len(f.Proc.Signals[600]) != 0 {
		t.Fatal("reboot touched an unrelated process")
	}
	quiesced := false
	for _, address := range f.Artifacts.Quiesces() {
		quiesced = quiesced || address == f.record.Address
	}
	if !quiesced {
		t.Fatal("the slot's own zellij session was not killed")
	}
}

func TestRebootWithoutRegisteredWrapperIsUnchanged(t *testing.T) {
	f := slotTreeFixture(t)
	f.Couch.SlotWrapper = func(ThreadAddress) (ProcessIdentity, bool) { return ProcessIdentity{}, false }
	f.Artifacts.SetPairSession(f.record.Address, "📁1-7", false)
	result, stops, err := f.rebootFresh(t)
	if err != nil || result.Start.Handle == nil || len(stops) != 0 {
		t.Fatalf("result=%+v stops=%v err=%v", result, stops, err)
	}
	for _, pid := range []int{500, 501, 502} {
		if len(f.Proc.Signals[pid]) != 0 {
			t.Fatalf("pid %d signalled without a registered wrapper", pid)
		}
	}
}

func TestRebootRefusesWhenSlotTreeExitIsNotConfirmed(t *testing.T) {
	f := slotTreeFixture(t)
	delete(f.Proc.DiesOn, 501)
	_, _, err := f.rebootFresh(t)
	if err == nil || !strings.Contains(err.Error(), "kill not confirmed") || len(f.Runner.order) != 0 {
		t.Fatalf("unconfirmed kill admitted: err=%v children=%v", err, f.Runner.order)
	}
}

// The session is killed only when this slot alone holds it.
func TestSlotSessionSharedWithAnotherLiveSlotIsLeft(t *testing.T) {
	store, ns := newTestThreadStore(t)
	binding := func(r *ThreadRecord) {
		r.SessionBinding = &couchidentity.SessionBinding{C: 1, M: 9, Name: "📁1-9", ScopeKey: r.Address.RepoScope, Tag: string(r.Address.Tag), StartNonce: "start-0102030405060708"}
	}
	var records []ThreadRecord
	for i, tag := range []string{"couch-1111111111111111", "couch-2222222222222222"} {
		r := validThreadRecord(t)
		r.Address.Tag = ThreadTag(tag)
		r.StartingPath, r.WorkingPath = ns.Dir(), ns.Dir()
		binding(&r)
		r.Incarnations = []ThreadIncarnation{{PID: 70 + i, Identity: "helper", State: IncarnationLive}}
		created, err := store.CreateThread(r)
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, created)
	}
	proc := NewFakeProcOps()
	proc.Set(71, "helper")
	couch := &Couch{Threads: store, Proc: proc, Git: NewFakeGit(map[GitCall]string{{Dir: ns.Dir(), Args: "rev-parse --show-toplevel"}: "/work/pair"})}
	if holder, err := couch.otherLiveSessionHolder(context.Background(), records[0], "📁1-9"); err != nil || holder != "pair:0" {
		t.Fatalf("live holder: %q %v", holder, err)
	}
	proc.Kill(71)
	if holder, err := couch.otherLiveSessionHolder(context.Background(), records[0], "📁1-9"); err != nil || holder != "" {
		t.Fatalf("dead holder kept the session: %q %v", holder, err)
	}
}
