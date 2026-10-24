package launcher

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func confirmedRecoveryFixture(t *testing.T, n int) (*recoveryTerminalWorld, recoveryTerminalDriver, *readiness.ReadyRecord) {
	t.Helper()
	w, d := recoveryTerminalFixture(t, n)
	w.cascadeServer = false
	ready := &readiness.ReadyRecord{Tag: "work", Agent: "codex", Session: "session", Nonce: "nonce", PID: 13, AgentIdentity: "native-start", LaunchOrdinal: 1}
	w.snapshot.Args = map[int][]string{}
	for _, p := range w.snapshot.Processes {
		if p.PPID != 1 {
			w.snapshot.Args[p.PID] = strings.Fields(w.snapshot.Commands[p.PID])
		} else {
			w.snapshot.Args[p.PID] = []string{"zellij", "--server", "/tmp/sockets/session"}
		}
	}
	if n > 0 {
		w.snapshot.Processes = append(w.snapshot.Processes, procutil.Process{PID: 12, PPID: 10, Identity: "wrapper-start"}, procutil.Process{PID: 13, PPID: 12, Identity: "native-start"})
		w.snapshot.Args[12] = []string{"pair-wrap", "--scrollback-log", "/data/repos/0123456789abcdef/scrollback-work-codex.raw", "codex"}
		w.snapshot.Args[13] = []string{"codex", "--resume", "native-id"}
		for _, pid := range []int{12, 13} {
			w.snapshot.Commands[pid] = strings.Join(w.snapshot.Args[pid], " ")
		}
	}
	d.read = func(context.Context, string, int64) ([]byte, error) {
		raw, err := readiness.Encode(*ready)
		return []byte(raw), err
	}
	return w, d, ready
}

func TestConfirmedRecoveryTerminalCapturesOwnedWrapperAndImmutableSummary(t *testing.T) {
	w, d, _ := confirmedRecoveryFixture(t, 1)
	if _, err := d.prepare(context.Background(), "/data", "0123456789abcdef", "work", "session"); err == nil {
		t.Fatal("strict path accepted live wrapper")
	}
	plan, err := d.prepareConfirmed(context.Background(), "/data", "0123456789abcdef", "work", "session")
	if err != nil {
		t.Fatal(err)
	}
	summary := plan.Processes()
	if len(summary) != 4 || summary[2].Role != "wrapper" || summary[3].Role != "native-agent" || summary[3].ParentPID != 12 {
		t.Fatalf("summary=%+v", summary)
	}
	digest := plan.Digest()
	summary[0].PID = 999
	if digest == "" || plan.Digest() != digest || plan.Processes()[0].PID == 999 {
		t.Fatal("mutable public kill authority")
	}
	if err := d.recheck(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(w.events, ","), "kill") {
		t.Fatal("preview/recheck signalled")
	}
	if err := d.retire(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
}

func TestConfirmedRecoveryTerminalRejectsUnprovenLiveMembers(t *testing.T) {
	for _, kind := range []string{"foreign", "unknown", "missing-argv", "missing-server-argv", "missing-ready", "nonce", "session", "identity", "ancestry"} {
		t.Run(kind, func(t *testing.T) {
			w, d, r := confirmedRecoveryFixture(t, 1)
			switch kind {
			case "foreign":
				w.snapshot.Args[12][2] = "/data/repos/0123456789abcdef/scrollback-other-codex.raw"
			case "unknown":
				w.snapshot.Processes = append(w.snapshot.Processes, procutil.Process{PID: 14, PPID: 10, Identity: "unknown"})
				w.snapshot.Args[14] = []string{"sh"}
				w.snapshot.Commands[14] = "sh"
			case "missing-argv":
				delete(w.snapshot.Args, 12)
			case "missing-server-argv":
				delete(w.snapshot.Args, 10)
			case "missing-ready":
				d.read = func(context.Context, string, int64) ([]byte, error) { return nil, errors.New("missing") }
			case "nonce":
				r.Nonce = ""
			case "session":
				r.Session = "other"
			case "identity":
				r.AgentIdentity = "reused"
			case "ancestry":
				w.snapshot.Processes[3].PPID = 10
			}
			if _, err := d.prepareConfirmed(context.Background(), "/data", "0123456789abcdef", "work", "session"); err == nil {
				t.Fatal("unproven member accepted")
			}
			if strings.Contains(strings.Join(w.events, ","), "kill") {
				t.Fatal("refusal signalled")
			}
		})
	}
}

func TestConfirmedRecoveryTerminalRechecksAcceptedSetBeforeEverySignal(t *testing.T) {
	for _, kind := range []string{"new-child", "reused", "reparented", "readiness", "vanished"} {
		t.Run(kind, func(t *testing.T) {
			w, d, r := confirmedRecoveryFixture(t, 2)
			plan, err := d.prepareConfirmed(context.Background(), "/data", "0123456789abcdef", "work", "session")
			if err != nil {
				t.Fatal(err)
			}
			w.events = nil
			w.onProcessKill = func(int) {
				w.onProcessKill = nil
				switch kind {
				case "new-child":
					w.snapshot.Processes = append(w.snapshot.Processes, procutil.Process{PID: 90, PPID: 20, Identity: "new"})
					w.snapshot.Commands[90] = "sh"
					w.snapshot.Args[90] = []string{"sh"}
				case "reused":
					w.snapshot.Processes[3].Identity = "replacement"
				case "reparented":
					w.snapshot.Processes[3].PPID = 1
				case "readiness":
					r.Nonce = "replaced"
				case "vanished":
					w.snapshot.Processes = append(w.snapshot.Processes[:1], w.snapshot.Processes[2:]...)
				}
			}
			err = d.retire(context.Background(), plan)
			kills := len(w.signals)
			if kind == "vanished" {
				if err != nil || kills != len(plan.Processes())-1 {
					t.Fatalf("vanishing accepted child: kills=%d err=%v", kills, err)
				}
			} else if err == nil || kills != 1 {
				t.Fatalf("continued changed plan: kills=%d err=%v", kills, err)
			}
		})
	}
}

func TestConfirmedRecoveryTerminalDigestBindsProcessSetAndIgnoresTableOrder(t *testing.T) {
	w, d, _ := confirmedRecoveryFixture(t, 1)
	p, err := d.prepareConfirmed(context.Background(), "/data", "0123456789abcdef", "work", "session")
	if err != nil {
		t.Fatal(err)
	}
	for i, j := 0, len(w.snapshot.Processes)-1; i < j; i, j = i+1, j-1 {
		w.snapshot.Processes[i], w.snapshot.Processes[j] = w.snapshot.Processes[j], w.snapshot.Processes[i]
	}
	q, err := d.prepareConfirmed(context.Background(), "/data", "0123456789abcdef", "work", "session")
	if err != nil {
		t.Fatal(err)
	}
	if p.Digest() != q.Digest() || !reflect.DeepEqual(p.Processes(), q.Processes()) {
		t.Fatal("table order changed digest")
	}
	w.snapshot.Args[13] = append(w.snapshot.Args[13], "new-arg")
	q, err = d.prepareConfirmed(context.Background(), "/data", "0123456789abcdef", "work", "session")
	if err != nil {
		t.Fatal(err)
	}
	if p.Digest() == q.Digest() {
		t.Fatal("changed argv did not change digest")
	}
}

func TestConfirmedRecoveryTerminalReadOnlyRecheckRefusesStaleEvidence(t *testing.T) {
	for _, kind := range []string{"ready-nonce", "argv", "server-parent", "server-argv", "child-parent", "child-gone", "new-child"} {
		t.Run(kind, func(t *testing.T) {
			w, d, r := confirmedRecoveryFixture(t, 1)
			p, err := d.prepareConfirmed(context.Background(), "/data", "0123456789abcdef", "work", "session")
			if err != nil {
				t.Fatal(err)
			}
			w.events = nil
			switch kind {
			case "ready-nonce":
				r.Nonce = "changed"
			case "argv":
				w.snapshot.Args[12][2] = "/data/repos/0123456789abcdef/scrollback-other-codex.raw"
			case "server-parent":
				w.snapshot.Processes[0].PPID = 99
			case "server-argv":
				w.snapshot.Args[10] = []string{"zellij", "--server", "/tmp/sockets/session", "extra"}
			case "child-parent":
				w.snapshot.Processes[3].PPID = 11
			case "child-gone":
				w.snapshot.Processes = w.snapshot.Processes[:3]
			case "new-child":
				w.snapshot.Processes = append(w.snapshot.Processes, procutil.Process{PID: 99, PPID: 12, Identity: "new"})
			}
			if err := d.recheck(context.Background(), p); err == nil {
				t.Fatal("stale recheck accepted")
			}
			if strings.Contains(strings.Join(w.events, ","), "acquire") || strings.Contains(strings.Join(w.events, ","), "kill") {
				t.Fatal("recheck performed effects")
			}
			if kind == "child-gone" {
				// A normal park between preview and retirement may stop any
				// accepted member. Missing members never grant new targets.
				if err := d.retire(context.Background(), p); err != nil {
					t.Fatal(err)
				}
				return
			}
			if err := d.retire(context.Background(), p); err == nil {
				t.Fatal("stale retirement accepted")
			}
			if strings.Contains(strings.Join(w.events, ","), "kill") {
				t.Fatal("stale retirement signalled")
			}
		})
	}
}

// A transient ownership refusal must end the actual retirement call, even if
// a later read would return the original evidence again.
func TestConfirmedRecoveryTerminalOwnershipRefusalIsSticky(t *testing.T) {
	w, d, r := confirmedRecoveryFixture(t, 1)
	p, err := d.prepareConfirmed(context.Background(), "/data", "0123456789abcdef", "work", "session")
	if err != nil {
		t.Fatal(err)
	}
	w.onProcessKill = func(int) { w.onProcessKill = nil; r.Nonce = "replaced" }
	read := d.read
	d.read = func(ctx context.Context, path string, limit int64) ([]byte, error) {
		raw, err := read(ctx, path, limit)
		r.Nonce = "nonce"
		return raw, err
	}
	if err = d.retire(context.Background(), p); err == nil {
		t.Fatal("ownership refusal forgotten")
	}
	if got := strings.Join(w.signals, ","); got != "13" {
		t.Fatalf("resumed signals after refusal: %s", got)
	}
}

// A server's disappearance is independent of native/wrapper process lifetime.
func TestConfirmedRetirementRequiresEveryAcceptedOwnerAbsent(t *testing.T) {
	w, d, _ := confirmedRecoveryFixture(t, 1)
	p, err := d.prepareConfirmed(context.Background(), "/data", "0123456789abcdef", "work", "session")
	if err != nil {
		t.Fatal(err)
	}
	w.events = nil
	if err = d.retire(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if len(w.snapshot.Processes) != 0 {
		t.Fatalf("retirement succeeded with accepted owners alive: %+v", w.snapshot.Processes)
	}
	if got := strings.Join(w.signals, ","); got != "13,11,12,10" {
		t.Fatalf("child-before-parent exact stop order=%s", got)
	}
}

func TestConfirmedRetirementRefusesChangedSurvivingOwners(t *testing.T) {
	for _, kind := range []string{"reparented", "replaced", "new-child"} {
		t.Run(kind, func(t *testing.T) {
			w, d, _ := confirmedRecoveryFixture(t, 1)
			p, err := d.prepareConfirmed(context.Background(), "/data", "0123456789abcdef", "work", "session")
			if err != nil {
				t.Fatal(err)
			}
			w.events = nil
			// The first accepted child dies, but another accepted owner changes before
			// the next signal. Neither that changed process nor a foreign child is a target.
			w.onProcessKill = func(pid int) {
				w.onProcessKill = nil
				switch kind {
				case "reparented":
					w.snapshot.Processes[2].PPID = 1
				case "replaced":
					w.snapshot.Processes[2].Identity = "replacement"
				case "new-child":
					w.snapshot.Processes = append(w.snapshot.Processes, procutil.Process{PID: 90, PPID: 12, Identity: "foreign"})
					w.snapshot.Args[90] = []string{"sh"}
				}
			}
			if err = d.retire(context.Background(), p); err == nil {
				t.Fatal("changed surviving owner accepted")
			}
			if got := strings.Join(w.signals, ","); got != "13" {
				t.Fatalf("signals continued after mutation: %s", got)
			}
		})
	}
}

func TestConfirmedRetirementWaitsForIndependentAbsence(t *testing.T) {
	w, d, _ := confirmedRecoveryFixture(t, 1)
	p, err := d.prepareConfirmed(context.Background(), "/data", "0123456789abcdef", "work", "session")
	if err != nil {
		t.Fatal(err)
	}
	w.keepProcesses = true
	ctx, expire, cancel := testwait.DeadlineOnCue(context.Background(), d.timeout)
	defer cancel()
	observedSurvivor := false
	w.onObserve = func(snapshot recoveryTerminalSnapshot) {
		if len(w.signals) != 1 || w.signals[0] != "13" {
			return
		}
		for _, process := range snapshot.Processes {
			if process.PID == 13 {
				observedSurvivor = true
				expire()
			}
		}
	}
	if err = d.retire(ctx, p); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("retirement with surviving signaled owner: %v", err)
	}
	if !observedSurvivor {
		t.Fatal("deadline preceded independent surviving-owner observation")
	}
	if got := strings.Join(w.signals, ","); got != "13" {
		t.Fatalf("signalled parent while child survived: %s", got)
	}
}

func (w *recoveryTerminalWorld) KillProcess(ctx context.Context, v recoveryProcessWitness) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	w.signals = append(w.signals, fmt.Sprint(v.PID))
	w.events = append(w.events, "process-signal")
	if w.onProcessKill != nil {
		w.onProcessKill(v.PID)
	}
	if !w.keepProcesses {
		w.removeProcess(v.PID)
	}
	return nil
}
func (w *recoveryTerminalWorld) removeProcess(pid int) {
	var remaining []procutil.Process
	for _, p := range w.snapshot.Processes {
		if p.PID != pid {
			remaining = append(remaining, p)
		}
	}
	w.snapshot.Processes = remaining
}

func TestConfirmedRetirementRechecksAfterTerminalListing(t *testing.T) {
	w, d, _ := confirmedRecoveryFixture(t, 1)
	p, err := d.prepareConfirmed(context.Background(), "/data", "0123456789abcdef", "work", "session")
	if err != nil {
		t.Fatal(err)
	}
	w.onPresent = func() {
		w.snapshot.Servers = append(w.snapshot.Servers, SessionServerIdentity{PID: 90, Identity: "foreign", Session: "session", Socket: "/tmp/new/session"})
	}
	if err = d.retire(context.Background(), p); err == nil {
		t.Fatal("terminal created during final listing check accepted as absent")
	}
	if got := strings.Join(w.signals, ","); got != "13,11,12,10" {
		t.Fatalf("new server was signalled: %s", got)
	}
}

func TestConfirmedRetirementAcceptsOwnersAlreadyStoppedByNormalPark(t *testing.T) {
	for _, kind := range []string{"all", "native-child"} {
		t.Run(kind, func(t *testing.T) {
			w, d, _ := confirmedRecoveryFixture(t, 1)
			p, err := d.prepareConfirmed(context.Background(), "/data", "0123456789abcdef", "work", "session")
			if err != nil {
				t.Fatal(err)
			}
			w.events = nil
			if kind == "all" {
				w.snapshot.Processes = nil
				w.snapshot.Servers = nil
				d.read = func(context.Context, string, int64) ([]byte, error) {
					t.Fatal("read readiness for gone wrapper")
					return nil, errors.New("missing")
				}
			} else {
				w.removeProcess(13)
			}
			if err = d.retire(context.Background(), p); err != nil {
				t.Fatal(err)
			}
			expected := ""
			if kind == "native-child" {
				expected = "11,12,10"
			}
			if got := strings.Join(w.signals, ","); got != expected {
				t.Fatalf("signals=%s want=%s", got, expected)
			}
		})
	}
}
