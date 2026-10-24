package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

type recoveryTerminalWorld struct {
	snapshot      recoveryTerminalSnapshot
	events        []string
	observeErr    error
	onAcquire     func()
	onKill        func()
	onPresent     func()
	onProcessKill func(int)
	onObserve     func(recoveryTerminalSnapshot)
	signals       []string
	keepProcesses bool
	cascadeServer bool
}
type recoveryTerminalLease struct{ w *recoveryTerminalWorld }

func (l recoveryTerminalLease) Close() error { l.w.events = append(l.w.events, "release"); return nil }
func (w *recoveryTerminalWorld) Acquire(ctx context.Context, session string, servers []SessionServerIdentity) (io.Closer, error) {
	w.events = append(w.events, "acquire")
	if w.onAcquire != nil {
		w.onAcquire()
	}
	return recoveryTerminalLease{w}, ctx.Err()
}
func (w *recoveryTerminalWorld) Observe(ctx context.Context, session string) (recoveryTerminalSnapshot, error) {
	w.events = append(w.events, "observe")
	if w.observeErr != nil {
		return recoveryTerminalSnapshot{}, w.observeErr
	}
	if err := ctx.Err(); err != nil {
		return recoveryTerminalSnapshot{}, err
	}
	if w.onObserve != nil {
		w.onObserve(w.snapshot)
	}
	return w.snapshot, nil
}
func (w *recoveryTerminalWorld) SessionPresent(context.Context, string) (bool, error) {
	present := len(w.snapshot.Servers) > 0
	if w.onPresent != nil {
		w.onPresent()
	}
	return present, nil
}
func (w *recoveryTerminalWorld) KillServer(server SessionServerIdentity) error {
	w.events = append(w.events, "kill")
	if w.onKill != nil {
		w.onKill()
	}
	var remaining []SessionServerIdentity
	for _, s := range w.snapshot.Servers {
		if s != server {
			remaining = append(remaining, s)
		}
	}
	w.snapshot.Servers = remaining
	if w.cascadeServer {
		rows, _ := recoveryRows(w.snapshot)
		var pids []int
		for _, p := range w.snapshot.Processes {
			if recoveryDescendant(server.PID, p.PID, rows) {
				pids = append(pids, p.PID)
			}
		}
		for _, pid := range pids {
			w.removeProcess(pid)
		}
	}
	w.removeProcess(server.PID)
	w.signals = append(w.signals, fmt.Sprint(server.PID))
	return nil
}
func recoveryTerminalFixture(t *testing.T, n int) (*recoveryTerminalWorld, recoveryTerminalDriver) {
	t.Helper()
	owner, err := artifactpath.NewStorageOwner("/data", "0123456789abcdef", "work")
	if err != nil {
		t.Fatal(err)
	}
	paths, _ := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
	w := &recoveryTerminalWorld{cascadeServer: true, snapshot: recoveryTerminalSnapshot{Commands: map[int]string{}}}
	for i := 0; i < n; i++ {
		pid := 10 + i*10
		server := SessionServerIdentity{PID: pid, Identity: "server-start", Session: "session", Socket: "/tmp/sockets/session"}
		w.snapshot.Servers = append(w.snapshot.Servers, server)
		w.snapshot.Processes = append(w.snapshot.Processes, procutil.Process{PID: pid, PPID: 1, Identity: server.Identity}, procutil.Process{PID: pid + 1, PPID: pid, Identity: "editor-start"})
		w.snapshot.Commands[pid+1] = "nvim " + paths.Draft()
	}
	return w, recoveryTerminalDriver{ops: w, timeout: time.Second, poll: time.Nanosecond}
}
func TestRecoveryTerminalRetiresOnlyAttributedSnapshotUnderLease(t *testing.T) {
	w, d := recoveryTerminalFixture(t, 2)
	plan, err := d.prepare(context.Background(), "/data", "0123456789abcdef", "work", "session")
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range w.events {
		if event == "kill" {
			t.Fatal("prepare had effect")
		}
	}
	w.events = nil
	if err := d.retire(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	events := strings.Join(w.events, ",")
	if !strings.HasPrefix(events, "acquire,observe") || !strings.HasSuffix(events, "release") || strings.Count(events, "kill") != 2 {
		t.Fatalf("effects=%v", w.events)
	}
}
func TestRecoveryTerminalRefusesUnattributedForeignAndWrapper(t *testing.T) {
	for _, kind := range []string{"empty", "foreign", "wrapper", "unknown"} {
		t.Run(kind, func(t *testing.T) {
			w, d := recoveryTerminalFixture(t, 1)
			switch kind {
			case "empty":
				w.snapshot.Processes = w.snapshot.Processes[:1]
			case "foreign":
				w.snapshot.Commands[11] = "nvim /data/repos/0123456789abcdef/draft-other.md"
			case "wrapper":
				w.snapshot.Commands[11] = "pair-wrap --scrollback-log /data/repos/0123456789abcdef/scrollback-work-codex.raw codex"
			case "unknown":
				w.observeErr = errors.New("ps unreadable")
			}
			if _, err := d.prepare(context.Background(), "/data", "0123456789abcdef", "work", "session"); err == nil {
				t.Fatal("unproven retirement accepted")
			}
			if strings.Contains(strings.Join(w.events, ","), "kill") {
				t.Fatal("refusal signalled process")
			}
		})
	}
}
func TestRecoveryTerminalRevalidatesWholePlan(t *testing.T) {
	for _, kind := range []string{"reused", "new-server", "new-child", "changed-owner"} {
		t.Run(kind, func(t *testing.T) {
			w, d := recoveryTerminalFixture(t, 1)
			plan, err := d.prepare(context.Background(), "/data", "0123456789abcdef", "work", "session")
			if err != nil {
				t.Fatal(err)
			}
			w.events = nil
			w.onAcquire = func() {
				switch kind {
				case "reused":
					w.snapshot.Servers[0].Identity = "reused"
				case "new-server":
					w.snapshot.Servers = append(w.snapshot.Servers, SessionServerIdentity{PID: 20, Identity: "new", Session: "session", Socket: "/tmp/other/session"})
				case "new-child":
					w.snapshot.Processes = append(w.snapshot.Processes, procutil.Process{PID: 12, PPID: 10, Identity: "new-child"})
					w.snapshot.Commands[12] = "codex"
				case "changed-owner":
					w.snapshot.Commands[11] = "nvim /data/repos/0123456789abcdef/draft-other.md"
				}
			}
			if err := d.retire(context.Background(), plan); err == nil {
				t.Fatal("changed snapshot retired")
			}
			if strings.Contains(strings.Join(w.events, ","), "kill") {
				t.Fatal("changed snapshot signalled")
			}
		})
	}
}
func TestRecoveryTerminalEmptyPlanCannotAuthorizeLaterServer(t *testing.T) {
	w, d := recoveryTerminalFixture(t, 0)
	plan, err := d.prepare(context.Background(), "/data", "0123456789abcdef", "work", "session")
	if err != nil {
		t.Fatal(err)
	}
	if err := d.retire(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	w.snapshot.Servers = []SessionServerIdentity{{PID: 9, Identity: "new", Session: "session", Socket: "/tmp/session"}}
	if err := d.retire(context.Background(), plan); err == nil {
		t.Fatal("empty plan authorized later server")
	}
}
func TestRecoveryTerminalStopsWhenNewServerAppearsDuringRetirement(t *testing.T) {
	w, d := recoveryTerminalFixture(t, 2)
	plan, err := d.prepare(context.Background(), "/data", "0123456789abcdef", "work", "session")
	if err != nil {
		t.Fatal(err)
	}
	w.events = nil
	w.onKill = func() {
		w.snapshot.Servers = append(w.snapshot.Servers, SessionServerIdentity{PID: 90, Identity: "new", Session: "session", Socket: "/tmp/other/session"})
		w.onKill = nil
	}
	if err := d.retire(context.Background(), plan); err == nil {
		t.Fatal("expanded immutable authorization")
	}
	if n := strings.Count(strings.Join(w.events, ","), "kill"); n != 1 {
		t.Fatalf("kills after new server=%d", n)
	}
}

func TestRecoveryTerminalCommandSnapshotHasHardByteBound(t *testing.T) {
	var output recoveryCommandOutput
	_, err := io.Copy(&output, io.LimitReader(strings.NewReader(strings.Repeat("x", 9<<20)), 9<<20))
	if err == nil || output.Len() > 8<<20 {
		t.Fatalf("command bound bypassed len=%d err=%v", output.Len(), err)
	}
}
func TestRecoveryTerminalRefusesUncertainDescendantIdentity(t *testing.T) {
	w, d := recoveryTerminalFixture(t, 1)
	w.snapshot.Processes = append(w.snapshot.Processes, procutil.Process{PID: 12, PPID: 10})
	w.snapshot.Commands[12] = "pair-wrap --scrollback-log /data/repos/0123456789abcdef/scrollback-work-codex.raw codex"
	if _, err := d.prepare(context.Background(), "/data", "0123456789abcdef", "work", "session"); err == nil {
		t.Fatal("uncertain descendant omitted from attribution")
	}
}

func TestRecoveryTerminalAcquireCannotRewriteInitialSnapshot(t *testing.T) {
	w, d := recoveryTerminalFixture(t, 1)
	w.onAcquire = func() {
		w.snapshot.Servers[0].PID = 20
		w.snapshot.Processes[0].PID = 20
		w.snapshot.Processes[1].PPID = 20
	}
	if _, err := d.prepare(context.Background(), "/data", "0123456789abcdef", "work", "session"); err == nil {
		t.Fatal("lease wait silently replaced authorized server")
	}
	if !strings.HasSuffix(strings.Join(w.events, ","), "release") {
		t.Fatal("changed acquisition leaked lease")
	}
}

func TestRecoveryTerminalStrictDoesNotMistakeServerExitForOwnerAbsence(t *testing.T) {
	w, d := recoveryTerminalFixture(t, 1)
	w.cascadeServer = false
	p, err := d.prepare(context.Background(), "/data", "0123456789abcdef", "work", "session")
	if err != nil {
		t.Fatal(err)
	}
	ctx, expire, cancel := testwait.DeadlineOnCue(context.Background(), d.timeout)
	defer cancel()
	observedSurvivor := false
	w.onObserve = func(snapshot recoveryTerminalSnapshot) {
		if len(w.signals) != 1 || w.signals[0] != "10" {
			return
		}
		for _, process := range snapshot.Processes {
			if process.PID == 11 {
				observedSurvivor = true
				expire()
			}
		}
	}
	if err = d.retire(ctx, p); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("strict retirement with surviving accepted editor: %v", err)
	}
	if !observedSurvivor {
		t.Fatal("deadline preceded independent surviving-owner observation")
	}
	if len(w.signals) != 1 || w.signals[0] != "10" {
		t.Fatalf("strict path expanded authority: %v", w.signals)
	}
}
