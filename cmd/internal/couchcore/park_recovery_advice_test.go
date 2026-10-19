package couchcore

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type parkAdviceArtifacts struct {
	*FakeThreadArtifactCollisionChecker
	binding    NativeBindingResolution
	bindingErr error
}

func (a parkAdviceArtifacts) ResolveEstablished(context.Context, string, string, string) (NativeBindingResolution, error) {
	return a.binding, a.bindingErr
}

func TestParkRecoveryNextActionAuthority(t *testing.T) {
	for _, tc := range []struct {
		name       string
		listed     bool
		listingErr error
		binding    NativeBindingResolution
		bindingErr error
		want       string
	}{
		{name: "listed terminal needs no native ledger", listed: true, bindingErr: errors.New("ledger unreadable"), want: "continue-live"},
		{name: "retained native conversation", binding: NativeBindingResolution{Status: sessioninventory.BindingEstablished, NativeID: "native-id"}, want: "resume-from-disk"},
		{name: "confirmed missing conversation", binding: NativeBindingResolution{Status: sessioninventory.BindingUnbound}, bindingErr: refuseBinding(ResumeBindingUnbound), want: "start-fresh"},
		{name: "listing unavailable", listingErr: errors.New("list unavailable")},
		{name: "native storage IO", bindingErr: errors.New("native storage unreadable")},
		{name: "incomplete native listing", binding: NativeBindingResolution{Status: sessioninventory.BindingUnbound, ObservationIncomplete: true}, bindingErr: refuseBinding(ResumeBindingUnbound)},
		{name: "provisional conversation", binding: NativeBindingResolution{Status: sessioninventory.BindingProvisional}, bindingErr: refuseBinding(ResumeBindingProvisional)},
		{name: "unknown status", binding: NativeBindingResolution{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, record := envWithLiveThread(t)
			record.Incarnations = nil
			env.Artifacts.SetPairSession(record.Address, "pair-exact", tc.listed)
			if tc.listingErr != nil {
				env.Artifacts.ListedSessionHook = func(ThreadAddress) error { return tc.listingErr }
			}
			env.Couch.Artifacts = parkAdviceArtifacts{env.Artifacts, tc.binding, tc.bindingErr}
			outcome, command, err := env.Couch.ParkRecoveryNextAction(context.Background(), record)
			if tc.want == "" {
				if err == nil || outcome != "" || command != "" {
					t.Fatalf("uncertain observation produced %q %q %v", outcome, command, err)
				}
			} else if err != nil || outcome != tc.want || command == "" {
				t.Fatalf("got %q %q %v; want %q", outcome, command, err, tc.want)
			}
			if len(env.Runner.Ops) != 0 {
				t.Fatalf("advice launched processes: %v", env.Runner.Ops)
			}
		})
	}
}

func TestParkRecoveryAdviceTargets(t *testing.T) {
	env, record := envWithLiveThread(t)
	if got := env.Couch.ParkRecoveryAdvice(context.Background(), record.Address); got != "Tab → Recover park" {
		t.Fatalf("ordinary thread advice = %q", got)
	}
	// Storage origin, rather than a tag or a repository-looking path, proves the numbered slot.
	store := testLocalThreadStore(t)
	local := recordAtCheckout(t, store.slot.WorktreeRoot, store.slot.WorktreeRoot, "couch-0123456789abcdef")
	if _, err := store.CreateThread(local); err != nil {
		t.Fatal(err)
	}
	c := &Couch{Threads: store}
	if got := c.ParkRecoveryAdvice(context.Background(), local.Address); got != "couch --recover-park pair:1 --confirm" {
		t.Fatalf("slot advice = %q", got)
	}
	foreign := local.Address
	foreign.Tag = "couch-ffffffffffffffff"
	if got := c.ParkRecoveryAdvice(context.Background(), foreign); strings.Contains(got, "--recover-park") {
		t.Fatalf("foreign local address invented a slot command: %q", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := c.ParkRecoveryAdvice(ctx, local.Address); strings.Contains(got, "--recover-park") {
		t.Fatalf("failed identity observation invented command: %q", got)
	}
}

func TestParkRecoveryNextActionCommandTargets(t *testing.T) {
	env, record := envWithLiveThread(t)
	record.Incarnations = nil
	env.Artifacts.SetPairSession(record.Address, "pair-exact", false)
	for _, tc := range []struct {
		binding NativeBindingResolution
		want    string
	}{
		{NativeBindingResolution{Status: sessioninventory.BindingEstablished, NativeID: "native-id"}, "Tab → Resume from disk"},
		{NativeBindingResolution{Status: sessioninventory.BindingUnbound}, "Tab → Start fresh"},
	} {
		env.Couch.Artifacts = parkAdviceArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts, binding: tc.binding}
		_, command, err := env.Couch.ParkRecoveryNextAction(context.Background(), record)
		if err != nil || command != tc.want {
			t.Fatalf("ordinary thread command = %q %v; want %q", command, err, tc.want)
		}
	}
}

func TestParkRecoveryNextActionRefusesUnsettledOwnership(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*relaunchEnv, *ThreadRecord)
	}{
		{"live owner without terminal", func(_ *relaunchEnv, _ *ThreadRecord) {}},
		{"unknown owner", func(e *relaunchEnv, r *ThreadRecord) { e.Proc.SetUnknown(r.Incarnations[0].PID) }},
		{"open park", func(_ *relaunchEnv, r *ThreadRecord) { r.Park = &ParkTransaction{Phase: ParkRequested} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, record := envWithLiveThread(t)
			env.Artifacts.SetPairSession(record.Address, "pair-exact", false)
			tc.change(env, &record)
			outcome, command, err := env.Couch.ParkRecoveryNextAction(context.Background(), record)
			if err == nil || outcome != "" || command != "" {
				t.Fatalf("unsettled ownership advertised %q %q %v", outcome, command, err)
			}
		})
	}
}

func TestParkRecoveryNextActionSlotCommands(t *testing.T) {
	env, record := envWithLiveThread(t)
	store := testLocalThreadStore(t)
	env.Artifacts = NewFakeThreadArtifactCollisionChecker()
	profile := record.LatestLaunchProfile
	record = recordAtCheckout(t, store.slot.WorktreeRoot, store.slot.WorktreeRoot, "couch-0123456789abcdef")
	record.LatestLaunchProfile = profile
	record.Incarnations = nil
	if _, err := store.CreateThread(record); err != nil {
		t.Fatal(err)
	}
	env.Couch.Threads = store
	env.Couch.Path = OSPathOps{}
	for _, tc := range []struct {
		listed                   bool
		status                   sessioninventory.BindingStatus
		wantOutcome, wantCommand string
	}{
		{true, sessioninventory.BindingUnbound, "continue-live", "couch --resume pair:1"},
		{false, sessioninventory.BindingEstablished, "resume-from-disk", "couch --resume pair:1"},
		{false, sessioninventory.BindingUnbound, "start-fresh", "couch --reboot pair:1 --confirm"},
	} {
		env.Artifacts.SetPairSession(record.Address, "pair-exact", tc.listed)
		env.Couch.Artifacts = parkAdviceArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts, binding: NativeBindingResolution{Status: tc.status, NativeID: func() string {
			if tc.status == sessioninventory.BindingEstablished {
				return "native-id"
			}
			return ""
		}()}}
		outcome, command, err := env.Couch.ParkRecoveryNextAction(context.Background(), record)
		if err != nil || outcome != tc.wantOutcome || command != tc.wantCommand {
			t.Fatalf("slot result=%q %q %v; want %q %q", outcome, command, err, tc.wantOutcome, tc.wantCommand)
		}
	}
}

func TestParkRecoveryMissingTerminalBindingIsNotAbsence(t *testing.T) {
	env, record := envWithLiveThread(t)
	env.Proc.Kill(42)
	env.Artifacts.SetPairSession(record.Address, "", false)
	outcome, command, err := env.Couch.ParkRecoveryNextAction(context.Background(), record)
	if err == nil || outcome != "" || command != "" {
		t.Fatalf("missing terminal binding guessed an outcome: %q %q %v", outcome, command, err)
	}
}
