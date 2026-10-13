package couchcore

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"testing"
	"time"
)

func TestAgentUsabilityHelperAloneDoesNotProveLive(t *testing.T) {
	record := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	for _, session := range []SessionObservation{{}, {State: SessionPresent}, {State: SessionAbsent}} {
		state, _ := ClassifyThread(record, ThreadEvidence{Live: []ProcessIdentity{{PID: 42, Identity: "helper"}}, Session: session})
		if state == ThreadLive {
			t.Fatalf("helper without usable agent classified live: %+v", session)
		}
	}
}

func TestAgentUsabilityListedSessionDoesNotProveDetached(t *testing.T) {
	record := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	state, _ := ClassifyThread(record, ThreadEvidence{Session: SessionObservation{State: SessionPresent}})
	if state == ThreadDetached {
		t.Fatal("listed session without usable agent classified detached")
	}
}

func TestAgentUsabilityExplicitEvidence(t *testing.T) {
	record := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	for _, tc := range []struct {
		name    string
		session SessionObservation
		live    bool
		want    ActionableThreadState
		reason  ThreadReason
	}{
		{"healthy detached", SessionObservation{State: SessionPresent, Terminal: &launcher.TerminalSafety{State: launcher.TerminalUsable}}, false, ThreadDetached, ""},
		{"healthy hosted", SessionObservation{State: SessionPresent, Terminal: &launcher.TerminalSafety{State: launcher.TerminalUsable}}, true, ThreadLive, ""},
		{"legacy", SessionObservation{State: SessionPresent, Terminal: &launcher.TerminalSafety{State: launcher.TerminalUnsafe, Reason: "missing policy"}}, false, ThreadUnusable, ReasonNeedsRelaunch},
		{"helper only", SessionObservation{State: SessionAbsent}, true, ThreadUnusable, ReasonNeedsRelaunch},
		{"contested hosted", SessionObservation{State: SessionUnresolved}, true, ThreadUnusable, ReasonUnknown},
		{"orphan hosted", SessionObservation{State: SessionOrphaned}, true, ThreadUnusable, ReasonNeedsRelaunch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			evidence := ThreadEvidence{Session: tc.session}
			if tc.live {
				evidence.Live = []ProcessIdentity{{PID: 42, Identity: "helper"}}
			}
			state, reason := ClassifyThread(record, evidence)
			if state != tc.want || reason != tc.reason {
				t.Fatalf("got %s/%s want %s/%s", state, reason, tc.want, tc.reason)
			}
		})
	}
	if ArchivableState(ThreadUnusable, ReasonNeedsRelaunch) || len(ActorActions(ActorRowFacts{State: ThreadUnusable, Reason: ReasonNeedsRelaunch})) != 0 {
		t.Fatal("unsafe terminal offered automatic lifecycle action")
	}
}

func TestAgentUsabilityRefreshSharesOneProcessTable(t *testing.T) {
	data := t.TempDir()
	indexSession(t, data, addressA, "📁repo-a")
	indexSession(t, data, addressB, "📁repo-b")
	checker, _ := sandboxedChecker(t, data, map[string]string{"📁repo-a": "detached", "📁repo-b": "detached"})
	checker.Servers = fakeServerStates{"📁repo-a": {Verdict: launcher.ServerReachable, Server: launcher.SessionServerIdentity{Session: "📁repo-a", PID: 1, Identity: "one"}}, "📁repo-b": {Verdict: launcher.ServerReachable, Server: launcher.SessionServerIdentity{Session: "📁repo-b", PID: 2, Identity: "two"}}}
	reads, calls := 0, 0
	checker.ProcessTable = func() ([]procutil.Process, error) { reads++; return []procutil.Process{{PID: 99}}, nil }
	checker.TerminalUsability = func(server launcher.SessionServerIdentity, root, scope, tag string, table []procutil.Process) launcher.TerminalSafety {
		calls++
		if len(table) != 1 || table[0].PID != 99 {
			t.Fatal("process table not shared")
		}
		return launcher.TerminalSafety{State: launcher.TerminalUsable}
	}
	got, err := checker.SessionPresence(context.Background(), []ThreadAddress{addressA, addressB})
	if err != nil {
		t.Fatal(err)
	}
	if reads != 1 || calls != 2 {
		t.Fatalf("reads=%d calls=%d", reads, calls)
	}
	for _, a := range []ThreadAddress{addressA, addressB} {
		if got[a].Terminal == nil || !got[a].Terminal.Usable() {
			t.Fatalf("missing usability for %v", a)
		}
	}
}

func TestAgentUsabilityUnreadableProcessTableStaysUnknown(t *testing.T) {
	data := t.TempDir()
	indexSession(t, data, addressA, "📁repo-a")
	checker, _ := sandboxedChecker(t, data, map[string]string{"📁repo-a": "detached"})
	checker.Servers = fakeServerStates{"📁repo-a": {Verdict: launcher.ServerReachable, Server: launcher.SessionServerIdentity{Session: "📁repo-a", PID: 1, Identity: "one"}}}
	checker.ProcessTable = func() ([]procutil.Process, error) { return nil, errors.New("cannot read processes") }
	checker.TerminalUsability = func(launcher.SessionServerIdentity, string, string, string, []procutil.Process) launcher.TerminalSafety {
		t.Fatal("observed without process snapshot")
		return launcher.TerminalSafety{State: launcher.TerminalUsable}
	}
	got, err := checker.SessionPresence(context.Background(), []ThreadAddress{addressA})
	if err != nil {
		t.Fatal(err)
	}
	if got[addressA].Terminal != nil || agentUsabilityReason(got[addressA]) != ReasonUnknown {
		t.Fatalf("unreadable became a positive verdict: %+v", got[addressA])
	}
}

func TestAgentUsabilitySuccessfulFixtureStartHasReadyAgent(t *testing.T) {
	env := newTestEnv(t, "/repo")
	actor, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	got, err := env.Artifacts.SessionPresence(context.Background(), []ThreadAddress{actor.Thread})
	if err != nil {
		t.Fatal(err)
	}
	if reason := agentUsabilityReason(got[actor.Thread]); reason != "" {
		t.Fatalf("successful fixture launch lacks agent-ready evidence: %s", reason)
	}
}

func TestAgentUsabilityRejectsDifferentSavedAgent(t *testing.T) {
	record := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	terminal := launcher.TerminalSafety{State: launcher.TerminalUsable}
	terminal.Policy.Agent = "codex"
	state, reason := ClassifyThread(record, ThreadEvidence{Session: SessionObservation{State: SessionPresent, Terminal: &terminal}})
	if state != ThreadUnusable || reason != ReasonNeedsRelaunch {
		t.Fatalf("different agent classified %s/%s", state, reason)
	}
}

func TestAgentUsabilityContestedServerNeedsRelaunch(t *testing.T) {
	record := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	observed := ProjectSessionPresence([]SessionNameBinding{{Address: record.Address, SessionName: "session"}}, []launcher.Session{{Name: "session", State: launcher.SessionLive}}, map[string]launcher.ServerState{"session": {Verdict: launcher.ServerContested}}, map[string]int{"session": 1})
	state, reason := ClassifyThread(record, ThreadEvidence{Session: observed[record.Address]})
	if state != ThreadUnusable || reason != ReasonNeedsRelaunch {
		t.Fatalf("contested classified %s/%s", state, reason)
	}
}

func TestAgentUsabilityOrphanNeedsManualRelaunch(t *testing.T) {
	record := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	server := launcher.SessionServerIdentity{PID: 42, Identity: "server"}
	e := ThreadEvidence{Session: SessionObservation{State: SessionOrphaned, Orphan: &server}}
	state, reason := ClassifyThread(record, e)
	if state != ThreadUnusable || reason != ReasonNeedsRelaunch {
		t.Fatalf("orphan classified %s/%s", state, reason)
	}
	orphan := orphanOf(state, reason, e)
	if orphan == nil || *orphan != server {
		t.Fatal("lost direct reap identity")
	}
	if actions := ActorActions(ActorRowFacts{State: state, Reason: reason, Orphaned: true}); len(actions) != 0 {
		t.Fatalf("unsafe terminal offers automatic actions %v", actions)
	}
}

func TestAgentUsabilityListedButServerGoneNeedsRelaunch(t *testing.T) {
	data := t.TempDir()
	indexSession(t, data, addressA, "📁repo-a")
	checker, _ := sandboxedChecker(t, data, map[string]string{"📁repo-a": "detached"})
	checker.Servers = fakeServerStates{}
	checker.ProcessTable = func() ([]procutil.Process, error) { return nil, nil }
	got, err := checker.SessionPresence(context.Background(), []ThreadAddress{addressA})
	if err != nil {
		t.Fatal(err)
	}
	if reason := agentUsabilityReason(got[addressA]); reason != ReasonNeedsRelaunch {
		t.Fatalf("listed terminal without server: %s", reason)
	}
}

func TestAgentUsabilityUnknownInspectionStaysUnknown(t *testing.T) {
	observation := SessionObservation{State: SessionPresent, Terminal: &launcher.TerminalSafety{Reason: "policy inspection unavailable"}}
	t.Run("projection", func(t *testing.T) {
		if got := agentUsabilityReason(observation); got != ReasonUnknown {
			t.Fatalf("unknown inspection projected %s", got)
		}
	})
	t.Run("agent mismatch", func(t *testing.T) {
		checker := ScopedThreadArtifactCollisionChecker{TerminalUsability: func(launcher.SessionServerIdentity, string, string, string, []procutil.Process) launcher.TerminalSafety {
			return *observation.Terminal
		}}
		got := checker.terminalUsability(launcher.SessionServerIdentity{}, SessionNameBinding{Agent: "claude"}, nil)
		if got.State != launcher.TerminalUnknown || got.Reason != observation.Terminal.Reason {
			t.Fatalf("missing policy agent overwrote unknown inspection: %+v", got)
		}
	})
}
