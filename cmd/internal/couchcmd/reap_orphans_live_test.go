package couchcmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

// Live conformance for the untagged-orphan sweep and its confirmed reap (#583's
// Done-when #2): a real bare `nvim --embed` reparented to PID 1 on this host,
// classified by the real sampler over the real process table, and ended by the
// supervisor's fresh ownership and revocation gates through real signals.
//
// Run it with:
//
//	PAIR_LIVE_ORPHAN_REAP=1 go test ./cmd/internal/couchcmd/ -run TestLiveOrphanNvim -v
//
// Gated on an env var + t.Skip, this repo's live-test idiom
// (launcher/nvim_reap_live_test.go, procutil/resource_darwin_test.go), because it
// spawns processes and SIGTERMs them.
//
// What only a live run can prove is the composition. Each native read the sweep
// depends on is separately conformed -- the cwd by procutil's
// TestResourceCwdOfAChild, the age by TestStartTimeFromIdentityNamesALiveChildsBirth,
// the argv by launcher's TestLiveNvimReap* -- but a fake states the argv, the cwd
// and the parentage it was handed, so no fake can show that a real orphan's real
// reads satisfy the real classifier. This does, with no synthetic field anywhere:
// the PPID comes from the kernel's own table after a real reparenting.
//
// Blast radius: one nvim process per test, each in its own temp directory, killed
// before the next test starts. The sweep classifies the whole host table, so a real
// run may name other slots' orphans -- the reap is filtered to the pid this test
// spawned, and the rest are logged rather than touched.
//
// A reparenting IS manufacturable on this host, which corrects the note M1 left in
// launcher/nvim_reap_live_test.go (corrected there too): start the embed from a shell
// that exits at once, and hold its stdin open from here. See spawnEmbed for the two
// measured constraints on that construction -- an unheld stdin and an inherited one
// both end the embed before any sweep can see it.
const liveOrphanWait = 30 * time.Second

// The age gate the live runs use, and what a spawn waits for before it sweeps.
// ParseResourcePolicy refuses a duration below 1s, so the gate is already at its
// floor; the wait is derived from it so the two cannot drift apart.
const (
	liveOrphanAgeGate = time.Second
	liveOrphanMinAge  = liveOrphanAgeGate + 100*time.Millisecond
)

func liveOrphanNvim(t *testing.T) string {
	t.Helper()
	if os.Getenv("PAIR_LIVE_ORPHAN_REAP") != "1" {
		t.Skip("set PAIR_LIVE_ORPHAN_REAP=1 to sweep a real orphaned embedded nvim")
	}
	nvim, err := exec.LookPath("nvim")
	if err != nil {
		t.Skipf("nvim is not on PATH: %v", err)
	}
	return nvim
}

// liveOrphanDir is the managed checkout the sweep will match the orphan's cwd
// against. EvalSymlinks because macOS hands out /var/folders/... and the kernel
// reports /private/var/..., and a containment decision on the unresolved path
// would prove nothing.
func liveOrphanDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// spawnEmbed starts a real `nvim --embed --headless -u NONE` serving dir, with its
// stdin held open by this test. orphaned decides the parentage and nothing else: true
// runs the embed under a shell that exits at once, so the kernel reparents it to PID
// 1; false keeps that same shell alive as its parent. Both cases run the same script,
// so PID 1 really is the only difference between the sweep's two verdicts.
//
// The stdin construction has two measured constraints, and dropping either one leaves
// no orphan to sweep -- only a race against a dying process:
//
//   - It has to be held open. M1 measured that an `--embed` nvim ends ~31 ms after its
//     stdin closes, and the real 13-day orphan's stdin was held by a process that
//     outlived its slot.
//   - It has to be redirected explicitly. Bash sends an asynchronous command's stdin
//     from /dev/null in a non-interactive shell, so an inherited descriptor -- an
//     os.Pipe handed to cmd.Stdin -- reaches the embed as an immediate EOF. Measured
//     2026-10-16: at PPID 1 with a live writer holding the other end, the embed was
//     gone from the kernel table within 150 ms. The same embed behind `< fifo` with
//     this process holding the write end stayed alive and stable for the whole poll.
//
// The write end is opened in a goroutine because opening a FIFO for write blocks until
// a reader appears, and the embed's own open for read blocks until a writer appears:
// the two rendezvous, so neither side can race the other.
func spawnEmbed(t *testing.T, nvim, dir string, orphaned bool) (pid int, identity string) {
	t.Helper()
	fifo := filepath.Join(dir, "embed-stdin")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatalf("create the fifo that holds the embed's stdin open: %v", err)
	}
	held := make(chan *os.File, 1)
	go func() {
		f, err := os.OpenFile(fifo, os.O_WRONLY, 0)
		if err != nil {
			t.Errorf("hold the fifo's write end open: %v", err)
			return
		}
		held <- f
	}()

	// The pid reaches this test through a sidecar rather than the shell's stdout: in
	// the live-parent case the shell never exits, so there is no Wait to close a
	// stdout copy against, and reading a buffer a copy goroutine still owns is a race.
	record := filepath.Join(dir, "embed.pid")
	script := `"$1" --embed --headless -u NONE <"$3" >/dev/null 2>&1 & echo $! > "$4"`
	if !orphaned {
		script += `; wait`
	}
	ctx, cancel := context.WithTimeout(context.Background(), liveOrphanWait)
	defer cancel()
	// Plain exec.Command, not CommandContext: cancelling this function's ctx on return
	// would SIGKILL the shell, and in the live-parent case that reparents the embed to
	// PID 1 before the sweep runs -- turning the "never touched" case into the orphan
	// case and passing it for the wrong reason. The cleanup below owns that lifetime.
	cmd := exec.Command("sh", "-c", script, "sh", nvim, "-", fifo, record)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	started := time.Now()
	if err := cmd.Start(); err != nil {
		t.Fatalf("start the embedded nvim: %v", err)
	}
	if orphaned {
		// Waiting for the shell to exit is what makes the embed an orphan: PID 1
		// adopts it, and the kernel's own table is the only authority for that.
		if err := cmd.Wait(); err != nil {
			t.Fatalf("the shell that orphans the embed: %v", err)
		}
	} else {
		t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
	}

	select {
	case f := <-held:
		// Registered before the embed's own kill below, so it runs after it: ending
		// the embed before dropping its stdin keeps this cleanup's EOF from being the
		// thing that looks like a successful reap.
		t.Cleanup(func() { _ = f.Close() })
	case <-time.After(liveOrphanWait):
		t.Fatalf("nothing opened %s for read, so the embed never started", fifo)
	}
	pid = awaitPidRecord(t, record)

	// Wait for the kernel to report the parentage this case needs. Polling the real
	// table rather than sleeping a guess is the point: the PPID the classifier reads
	// is the one the kernel settled on, not the one the test assumed.
	table := launcher.OSProcessTable{}
	for deadline := time.Now().Add(liveOrphanWait); ; time.Sleep(20 * time.Millisecond) {
		rows, err := table.Snapshot(ctx)
		if err != nil {
			t.Fatalf("read the process table: %v", err)
		}
		for _, row := range rows {
			if row.PID == pid && row.Identity != "" && (row.PPID == 1) == orphaned {
				// The age gate's floor is 1s (`ParseResourcePolicy` refuses a shorter
				// duration), so a just-spawned embed is legitimately too young to name.
				// Waiting for the gate rather than lowering it keeps the gate under test.
				if wait := time.Until(started.Add(liveOrphanMinAge)); wait > 0 {
					time.Sleep(wait)
				}
				// Registered only once the generation is known, and fenced on it: a
				// cleanup that killed by pid would SIGKILL whatever recycled it, which
				// is the exact habit #583 removes from the reap path.
				t.Cleanup(func() {
					if table.Identity(pid) == row.Identity {
						_ = syscall.Kill(pid, syscall.SIGKILL)
					}
				})
				return pid, row.Identity
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("the embed %d never reached the parentage this case needs (orphaned=%v)", pid, orphaned)
		}
	}
}

func awaitPidRecord(t *testing.T, record string) int {
	t.Helper()
	for deadline := time.Now().Add(liveOrphanWait); ; time.Sleep(20 * time.Millisecond) {
		if raw, err := os.ReadFile(record); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil && pid > 0 {
				return pid
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("the shell never published %s", record)
		}
	}
}

// liveOrphanSampler is the real sampler over the real host, with the one managed
// root this test owns and the age gate dropped to its floor so a just-spawned
// orphan is old enough to name. The RSS gates are raised out of the way instead, so
// the age gate -- and the cwd containment behind it -- is the only thing admitting a
// candidate.
func liveOrphanSampler(dir string) *resourceSampler {
	policy, err := couchcore.ParseResourcePolicy(nil)
	if err != nil {
		panic(err)
	}
	policy.OrphanNvimAge = liveOrphanAgeGate
	policy.OrphanRSSBytes, policy.ProcessRSSBytes = 1<<62, 1<<62
	return &resourceSampler{policy: policy, slots: func() couchcore.ResourceSlotInventory {
		return couchcore.ResourceSlotInventory{Complete: true, Slots: []couchcore.ResourceSlot{{
			Slot:           "live:0",
			OwnershipKnown: true,
			Workspace:      couchcore.WorkspaceIdentity{WorktreeRoot: dir},
		}}}
	}}
}

func liveSweepOrphans(t *testing.T, dir string) []couchcore.OrphanNvim {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), liveOrphanWait)
	defer cancel()
	sampler := liveOrphanSampler(dir)
	if !sampler.pass(ctx) {
		t.Fatal("the live pass was cut short before it classified anything")
	}
	view := sampler.view()
	t.Logf("live sweep: cwd_calls=%d command_calls=%d orphan_nvim=%d unattributed=%d errors=%v", view.Cost.CwdCalls, view.Cost.CommandCalls, len(view.OrphanNvim), len(view.Unattributed), view.Errors)
	for _, row := range view.OrphanNvim {
		t.Logf("named pid=%d age=%s cwd=%s root=%s reason=%s", row.Key.PID, row.Age.Round(time.Second), row.Cwd, row.Root, row.Reason)
	}
	return view.OrphanNvim
}

func liveRowFor(rows []couchcore.OrphanNvim, pid int) (couchcore.OrphanNvim, bool) {
	for _, row := range rows {
		if row.Key.PID == pid {
			return row, true
		}
	}
	return couchcore.OrphanNvim{}, false
}

// This disposable test world contains only the exact owned fixture subtree.
// Every identity and parent edge still comes from a fresh native census; host
// processes outside the fixture cannot contaminate or become signal targets.
// Full-host sampler qualification remains in the public-command native tests.
type liveOrphanFixtureReader struct {
	procutil.NativeResourceReader
	root couchcore.ResourceProcessKey
}

func (r liveOrphanFixtureReader) Table(ctx context.Context) ([]procutil.Process, int, error) {
	rows, calls, err := r.NativeResourceReader.Table(ctx)
	if err != nil {
		return nil, calls, err
	}
	children := map[int][]procutil.Process{}
	var root procutil.Process
	found := false
	for _, row := range rows {
		children[row.PPID] = append(children[row.PPID], row)
		if row.PID == r.root.PID && row.Identity == r.root.Identity {
			root, found = row, true
		}
	}
	if !found {
		return nil, calls, nil
	}
	owned := []procutil.Process{root}
	seen := map[int]bool{root.PID: true}
	for i := 0; i < len(owned); i++ {
		for _, child := range children[owned[i].PID] {
			if seen[child.PID] || child.Identity == "" {
				return nil, calls, fmt.Errorf("fixture subtree has ambiguous identity or ancestry")
			}
			seen[child.PID] = true
			owned = append(owned, child)
		}
	}
	return owned, calls, nil
}

// The sweep names a real orphan whose cwd is inside a managed checkout, on the
// kernel's own account of its parentage, its argv, its cwd and its age.
func TestLiveOrphanNvimSweepNamesARealReparentedEmbed(t *testing.T) {
	nvim := liveOrphanNvim(t)
	dir := liveOrphanDir(t)
	pid, identity := spawnEmbed(t, nvim, dir, true)
	row, ok := liveRowFor(liveSweepOrphans(t, dir), pid)
	if !ok {
		t.Fatalf("the sweep did not name the real orphan %d", pid)
	}
	if row.Key.Identity != identity {
		t.Fatalf("the sweep named generation %q; the kernel reported %q", row.Key.Identity, identity)
	}
	if row.Cwd != dir || row.Root != dir {
		t.Fatalf("cwd %q root %q; want the managed checkout %q", row.Cwd, row.Root, dir)
	}
	if row.Age < liveOrphanAgeGate {
		t.Fatalf("age %s is below the gate that admitted it", row.Age)
	}
}

// An embed with a live parent is never a candidate, in the same managed checkout,
// with the same argv and the same age gate: PID 1 is the pivot, and nothing else
// about the process distinguishes it from the leak (#583).
func TestLiveOrphanNvimSweepLeavesALiveParentsEmbedAlone(t *testing.T) {
	nvim := liveOrphanNvim(t)
	dir := liveOrphanDir(t)
	pid, identity := spawnEmbed(t, nvim, dir, false)
	if _, ok := liveRowFor(liveSweepOrphans(t, dir), pid); ok {
		t.Fatalf("the sweep named %d, whose parent was still alive", pid)
	}
	// The premise, checked after the verdict rather than assumed before it: a sweep
	// that named nothing proves the live-parent rule only if the parent was still
	// live and the embed still there. An embed that died, or one something reparented
	// to PID 1 mid-run, passes the assertion above for the wrong reason.
	ctx, cancel := context.WithTimeout(context.Background(), liveOrphanWait)
	defer cancel()
	rows, err := launcher.OSProcessTable{}.Snapshot(ctx)
	if err != nil {
		t.Fatalf("read the process table: %v", err)
	}
	for _, row := range rows {
		if row.PID == pid && row.Identity == identity {
			if row.PPID == 1 {
				t.Fatalf("the embed was at PID 1 by the time the sweep ran, so this proved the orphan rule and not the live-parent one")
			}
			return
		}
	}
	t.Fatalf("the embed %d was gone before the verdict could mean anything", pid)
}

// The confirmed reap ends the real orphan with a real signal, and the kernel
// agrees it is gone. Only the pid this test spawned is reaped: the sweep classified
// the whole host, and another slot's orphan is not this test's to end.
func TestLiveOrphanNvimReapEndsARealOrphan(t *testing.T) {
	nvim := liveOrphanNvim(t)
	dir := liveOrphanDir(t)
	pid, identity := spawnEmbed(t, nvim, dir, true)
	row, ok := liveRowFor(liveSweepOrphans(t, dir), pid)
	if !ok {
		t.Fatalf("the sweep did not name the real orphan %d, so there is nothing to confirm", pid)
	}
	ctx, cancel := context.WithTimeout(context.Background(), liveOrphanWait)
	defer cancel()
	// The real worker owns every fresh classification and the sticky loss fence.
	// Unlike the old local tree reaper, no admitted row can bypass a subsequent
	// ownership change or incomplete native collection at the effect boundary.
	sampler := liveOrphanSampler(dir)
	sampler.reader = liveOrphanFixtureReader{root: row.Key}
	sampler.sweeps = make(chan resourceRequest)
	worker, stopWorker := context.WithCancel(ctx)
	joined := make(chan struct{})
	go func() { defer close(joined); sampler.run(worker) }()
	t.Cleanup(func() { stopWorker(); <-joined })
	service := &messageService{provenOrphanProcesses: launcher.OSProcessTable{}}
	service.resources.Store(sampler)
	call := func(ctx context.Context, _ string, raw, reply any) error {
		request := raw.(couchmessage.Request)
		if request.LegacyTarget == nil || request.LegacyTarget.Root != row.Key {
			t.Fatalf("attempt to signal outside the fixture's exact root: %+v", request)
		}
		bounded, cancel := context.WithTimeout(ctx, couchmessage.TransportTimeout)
		defer cancel()
		*reply.(*couchmessage.Response) = service.handleLegacyOrphanSignal(bounded, request)
		return nil
	}
	reaper := launcher.TreeReaper(launcher.OSProcessTable{})
	plan := freezeLegacyReap(ctx, reaper, []couchcore.OrphanNvim{row}, nil)
	outcomes, left := reapFrozenLegacy(ctx, "", couchmessage.Request{}, call, reaper, plan)
	if len(left) != 0 {
		t.Fatalf("unexpected legacy registry outcomes: %+v", left)
	}
	if len(outcomes) != 1 || outcomes[0].Status != orphanReaped {
		t.Fatalf("outcomes %+v", outcomes)
	}
	table := launcher.OSProcessTable{}
	for deadline := time.Now().Add(liveOrphanWait); ; time.Sleep(20 * time.Millisecond) {
		if table.Identity(pid) != identity {
			t.Logf("the kernel no longer reports generation %q for pid %d", identity, pid)
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("pid %d still held generation %q %v after the reap", pid, identity, liveOrphanWait)
		}
	}
}
