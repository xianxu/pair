package launcher

import (
	"context"
	"strings"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

func TestRecoveryNativeOwnerIgnoresUnrelatedUnreadableProcesses(t *testing.T) {
	w, check := nativeOwnerFixture(t, "codex", "other")
	s := w.snapshots[0]
	for _, name := range []string{"endpointsecurity", "colorsyncd", "git"} {
		// A real native snapshot omits argv when kern.procargs2 returns EINVAL.
		p := procutil.Process{PID: 100 + len(s.Processes), PPID: 1, Identity: "live-root-service", Name: name}
		projected, err := projectRecoveryTerminalSnapshot("", []procutil.Observation{{Process: p, Err: syscall.EINVAL}}, []procutil.Process{p})
		if err != nil {
			t.Fatal(err)
		}
		s.Processes = append(s.Processes, projected.Processes...)
	}
	w.snapshots = []recoveryTerminalSnapshot{s}
	if err := checkRecoveryNativeOwnersWithDiscovery(context.Background(), check, w.Observe, w.Read, noOrphanOwners); err != nil {
		t.Fatalf("unrelated unreadable process blocked resume: %v", err)
	}
}

func TestRecoveryNativeOwnerIgnoresKernelConfirmedZombies(t *testing.T) {
	w, check := nativeOwnerFixture(t, "codex", "selected")
	s := w.snapshots[0]
	for i := range s.Processes {
		s.Processes[i].Zombie = true
		delete(s.Commands, s.Processes[i].PID)
	}
	s.Processes[0].Name = "pair-wrap"
	s.Processes[1].Name = "codex"
	w.snapshots = []recoveryTerminalSnapshot{s}
	if err := checkRecoveryNativeOwnersWithDiscovery(context.Background(), check, w.Observe, w.Read, orphanOwnerDiscovery(check)); err != nil {
		t.Fatalf("terminated wrapper/retained child blocked resume: %v", err)
	}
}

func TestRecoveryNativeOwnerRetainsCandidateUncertainty(t *testing.T) {
	for _, name := range []string{"pair", "pair-wrap", ""} {
		for _, missing := range []string{"command", "identity"} {
			t.Run(name+"/"+missing, func(t *testing.T) {
				w, check := nativeOwnerFixture(t, "codex", "other")
				s := w.snapshots[0]
				s.Processes[0].Name = name
				if missing == "command" {
					delete(s.Commands, 42)
				} else {
					s.Processes[0].Identity = ""
				}
				w.snapshots = []recoveryTerminalSnapshot{s}
				err := checkRecoveryNativeOwnersWithDiscovery(context.Background(), check, w.Observe, w.Read, noOrphanOwners)
				if err == nil || !strings.Contains(err.Error(), "42") || w.observations != 2 {
					t.Fatalf("uncertain candidate: err=%v observations=%d, want PID evidence and one retry", err, w.observations)
				}
			})
		}
	}
}

func TestRecoveryNativeOwnerBoundsChurnRetry(t *testing.T) {
	for _, mode := range []string{"settles", "never-settles", "new-writer", "unrelated-churn"} {
		t.Run(mode, func(t *testing.T) {
			w, check := nativeOwnerFixture(t, "codex", "other")
			stable := w.snapshots[0]
			unstable := stable
			unstable.Processes = append([]procutil.Process(nil), stable.Processes...)
			unstable.Commands = map[int]string{43: stable.Commands[43]}
			switch mode {
			case "settles":
				w.snapshots = []recoveryTerminalSnapshot{unstable, stable}
			case "never-settles":
				w.snapshots = []recoveryTerminalSnapshot{unstable}
			case "new-writer":
				// A candidate appearing during argv reads must be checked on retry.
				w, check = nativeOwnerFixture(t, "codex", "selected")
				w.snapshots = []recoveryTerminalSnapshot{unstable, w.snapshots[0]}
			case "unrelated-churn":
				unstable.Commands = stable.Commands
				unstable.Processes = append(unstable.Processes, procutil.Process{PID: 100, Name: "git", Identity: "new"})
				w.snapshots = []recoveryTerminalSnapshot{stable, unstable}
			}
			err := checkRecoveryNativeOwnersWithDiscovery(context.Background(), check, w.Observe, w.Read, noOrphanOwners)
			wantOK := mode == "settles" || mode == "unrelated-churn"
			wantObservations := map[string]int{"settles": 3, "never-settles": 2, "new-writer": 2, "unrelated-churn": 2}[mode]
			if (err == nil) != wantOK || w.observations != wantObservations {
				t.Fatalf("err=%v observations=%d, want allowed=%t observations=%d", err, w.observations, wantOK, wantObservations)
			}
		})
	}
}

func TestRecoveryNativeOwnerReobservesChangedWrapperSet(t *testing.T) {
	for _, settles := range []bool{false, true} {
		w, check := nativeOwnerFixture(t, "codex", "other")
		initial := w.snapshots[0]
		changed := initial
		changed.Processes = append([]procutil.Process(nil), initial.Processes...)
		changed.Processes[0].PPID = 99
		w.snapshots = []recoveryTerminalSnapshot{initial, changed, changed, initial}
		if settles {
			w.snapshots = w.snapshots[:3]
		}
		err := checkRecoveryNativeOwnersWithDiscovery(context.Background(), check, w.Observe, w.Read, noOrphanOwners)
		if (err == nil) != settles || w.observations != 4 {
			t.Fatalf("settles=%t err=%v observations=%d, want four observations across two attempts", settles, err, w.observations)
		}
	}
}

func TestRecoveryNativeLegacyReadinessNamesMigrationOwner(t *testing.T) {
	for _, orphan := range []bool{false, true} {
		w, check := nativeOwnerFixture(t, "codex", "other")
		w.ready.AgentIdentity = ""
		var err error
		if orphan {
			w.snapshots[0].Processes = w.snapshots[0].Processes[1:]
			w.snapshots[0].Processes[0].PPID = 1
			err = checkRecoveryNativeOwnersWithDiscovery(context.Background(), check, w.Observe, w.Read, orphanOwnerDiscovery(check))
		} else {
			err = checkRecoveryNativeOwnersWithDiscovery(context.Background(), check, w.Observe, w.Read, noOrphanOwners)
		}
		if err == nil {
			t.Fatal("legacy readiness silently admitted")
		}
		for _, evidence := range []string{check.RepoScope + ":other", "codex", "43", "agent_identity", "relaunch", "to migrate"} {
			if !strings.Contains(err.Error(), evidence) {
				t.Fatalf("legacy refusal lacks %q: %v", evidence, err)
			}
		}
	}
}
