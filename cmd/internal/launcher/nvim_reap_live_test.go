package launcher

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

// Live conformance for the tagged embed reaper (#583's Done-when #1): real
// `nvim --embed` processes on this host, real pid sidecars, and the reaper's real
// syscalls. A fake process table states both sides of the argv comparison itself,
// so only a live run can prove that the argv a spawned embed actually carries is
// the argv the reaper matches.
//
// Run it with:
//
//	PAIR_LIVE_NVIM_REAP=1 go test ./cmd/internal/launcher/ -run TestLiveNvimReap -v
//
// Gated on an env var + t.Skip, this package's live-test idiom
// (session_owner_live_test.go, session_quiescence_live_test.go), because it spawns
// processes and SIGKILLs them.
//
// The blast radius is one temp scope directory. Every kill these tests authorize is
// decided from evidence under it: a sidecar globbed from it, or an argv field naming
// a draft inside it. Real sessions' embeds carry another scope's paths, so no sweep
// here can reach them — the live-tag case asserts exactly that, on a host that had
// ~30 pair embeds running when this was written.
//
// Owner death is supplied the way the sweep reads it — the tag's absence from
// liveTags — and not by killing a parent process, because on this host a dead parent
// takes the embed with it: closing an `--embed` nvim's stdin pipe ends it in ~31 ms
// (measured 2026-10-16, exit status 1), and SIGKILLing the `nvim FILE` TUI parent
// removes its `nvim --embed` child within 3 s. So there is no orphan to reap unless
// something holds the embed's stdin open, which is what these tests do: the embed is
// alive, its tag is not live, and that combination is the whole authorization. The
// real 13-day orphan's stdin was held by a process that outlived its slot.
//
// A live tag's embed *is* a sweep candidate: the pane that spawned it can close while
// the session lives on, and the embed it leaves at ppid 1 is the shape #583 exists for.
// What decides it is the classifier's reapReasonReparented branch, pinned by the ppid-1
// rows in TestSweepOrphanNvimDecidesEachPidOnItsOwnEvidence. The live-tag case below is
// the other half of the same rule — a live tag's embed with its parent alive is decided
// and kept. The live run does not attempt a real reparenting: that needs a disposable
// parent which leaves the embed alive at ppid 1, and this host does not offer one (see
// the measurements above).

// liveEmbedPublish is the VimEnter pid publish the four pair nvim configs do,
// reduced to the one line the reaper depends on: the record names the embed, which
// is the process that has to die, not the TUI that forked it. The real configs are
// covered by tests/nvim-pidfile-lifecycle-test.sh; this test runs `-u NONE` because
// pair's configs refuse to boot without a `pair` binary on PATH (retention.lua), and
// the reaper's contract is the kernel process, its argv and its record.
const liveEmbedPublish = `lua vim.api.nvim_create_autocmd('VimEnter', {once = true, callback = function() ` +
	`pcall(vim.fn.writefile, {tostring(vim.fn.getpid())}, vim.env.PAIR_NVIM_PID_FILE) end})`

// liveSignalWait bounds the wait for the reaper's SIGKILL. A delivered SIGKILL lands
// in microseconds; this only has to outlast the sweep's own syscalls.
const liveSignalWait = 10 * time.Second

func liveNvim(t *testing.T) string {
	t.Helper()
	if os.Getenv("PAIR_LIVE_NVIM_REAP") != "1" {
		t.Skip("set PAIR_LIVE_NVIM_REAP=1 to reap a real embedded nvim")
	}
	nvim, err := exec.LookPath("nvim")
	if err != nil {
		t.Skipf("nvim is not on PATH: %v", err)
	}
	return nvim
}

// liveEmbed is a real embedded nvim this test spawned, parents, and may kill.
type liveEmbed struct {
	cmd *exec.Cmd
	// stdin stays open for the test's lifetime: closing it ends the embed (see the
	// header), and a dead embed is not one the reaper has anything to decide about.
	stdin    io.WriteCloser
	table    ProcessTable
	pid      int
	identity string
	record   string
}

// spawnLiveEmbed starts `nvim --embed --headless -u NONE <scope>/draft-<tag>.md` —
// the argv shape a real draft embed carries on this host (`ps -eo pid,ppid,command`
// shows `nvim --embed -u <pairHome>/nvim/init.lua <scopeDir>/draft-<tag>.md`) — and
// waits for its sidecar. It asserts the host's own argv read carries the two fields
// the reaper matches, so a later "not reaped" cannot be a silent argv mismatch.
func spawnLiveEmbed(t *testing.T, table ProcessTable, nvim, scopeDir, tag string) liveEmbed {
	t.Helper()
	paths, err := artifactpath.ResolveScoped(scopeDir, tag)
	if err != nil {
		t.Fatalf("resolve scoped %q under %q: %v", tag, scopeDir, err)
	}
	if err := os.WriteFile(paths.Draft(), []byte("draft text\n"), 0o600); err != nil {
		t.Fatalf("write the draft the embed serves: %v", err)
	}
	record := paths.NvimPID(nvimKindDraft)
	cmd := exec.Command(nvim, "--embed", "--headless", "-u", "NONE", "--cmd", liveEmbedPublish, paths.Draft())
	cmd.Env = append(os.Environ(), "PAIR_NVIM_PID_FILE="+record)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start the embedded nvim: %v", err)
	}
	t.Cleanup(func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	embed := liveEmbed{cmd: cmd, stdin: stdin, table: table, record: record}
	embed.pid = embed.awaitSidecar(t)
	if embed.pid != cmd.Process.Pid {
		t.Fatalf("the sidecar names %d but the child this test spawned is %d: cleanup kills by pid, so it must not have to guess which process the record meant", embed.pid, cmd.Process.Pid)
	}
	embed.identity = table.Identity(embed.pid)
	if embed.identity == "" {
		t.Fatalf("no kernel start stamp for the spawned embed %d", embed.pid)
	}
	args, ok := table.Command(embed.pid)
	if !ok {
		t.Fatalf("the host could not report argv for the spawned embed %d", embed.pid)
	}
	for _, want := range []string{"--embed", paths.Draft()} {
		if !argvCarriesField(args, want) {
			t.Fatalf("the spawned embed's argv %q carries no %q field, so the reaper could never match it", args, want)
		}
	}
	return embed
}

func (e liveEmbed) awaitSidecar(t *testing.T) int {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if raw, err := os.ReadFile(e.record); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil && pid > 0 {
				return pid
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("the embedded nvim never published %s", e.record)
	return 0
}

// alive reports whether the pid still holds the identity read at spawn, which is the
// reaper's own liveness question: a recycled pid is not this process.
func (e liveEmbed) alive() bool { return e.table.Identity(e.pid) == e.identity }

// waitSignal reaps the child and reports the signal that ended it. The test process
// is the parent, so this is the kernel's own account of the reaper's SIGKILL rather
// than an inference from a poll. It waits under a deadline: a sweep that declined to
// kill leaves Wait blocked for the whole go-test timeout, and a regression should
// fail on its own terms in seconds, not as a panic trace five minutes later.
func (e liveEmbed) waitSignal(t *testing.T) syscall.Signal {
	t.Helper()
	waited := make(chan error, 1)
	go func() { waited <- e.cmd.Wait() }()
	var err error
	select {
	case err = <-waited:
	case <-time.After(liveSignalWait):
		t.Fatalf("the embed was still running %v after the sweep, so no signal reached it", liveSignalWait)
	}
	if err == nil {
		t.Fatal("the embed exited cleanly, so the reaper's signal never reached it")
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("waiting on the embed: %v", err)
	}
	status, ok := exit.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		t.Fatalf("the embed ended by %v, not by a signal", err)
	}
	return status.Signal()
}

func liveSweep(t *testing.T, scopeDir string, liveTags []string) []nvimReapOutcome {
	t.Helper()
	rt := OSRuntime{DataDir: scopeDir}
	outcomes, err := sweepOrphanNvim(context.Background(), rt.processTable(), rt, scopeDir, liveTags)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	return outcomes
}

func outcomesFor(outcomes []nvimReapOutcome, pid int) []nvimReapOutcome {
	var found []nvimReapOutcome
	for _, outcome := range outcomes {
		if outcome.PID == pid {
			found = append(found, outcome)
		}
	}
	return found
}

func TestLiveNvimReapRemovesADeadTagsEmbed(t *testing.T) {
	nvim := liveNvim(t)
	scopeDir := t.TempDir()
	rt := OSRuntime{DataDir: scopeDir}
	table := rt.processTable()
	tag := "live-dead"
	embed := spawnLiveEmbed(t, table, nvim, scopeDir, tag)

	// The owner is gone, so the live-tag set does not carry the tag: that absence is
	// the sweep's whole authorization.
	found := outcomesFor(liveSweep(t, scopeDir, nil), embed.pid)
	// The pid is named twice — by its sidecar and by its own argv — and must still be
	// decided once, or the report claims a reap it did not perform.
	if len(found) != 1 {
		t.Fatalf("the sweep decided pid %d %d times (%v), want exactly one decision", embed.pid, len(found), found)
	}
	if !found[0].Killed {
		t.Errorf("the sweep decided %q without a SIGKILL the kernel accepted", found[0].Reason)
	}
	if !found[0].Removed {
		t.Error("the sweep killed the embed but did not remove its record")
	}
	if found[0].Reason != reapReasonOwnerDead {
		t.Errorf("reason = %q, want %q", found[0].Reason, reapReasonOwnerDead)
	}
	if sig := embed.waitSignal(t); sig != syscall.SIGKILL {
		t.Errorf("the embed ended by %v, want the reaper's SIGKILL", sig)
	}
	if _, err := os.Stat(embed.record); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the sidecar survived the reap: %v", err)
	}
}

// A live tag's embed whose parent is alive. The sweep decides it — a live tag is a
// candidate, since its embed can outlive the pane that spawned it — and keeps it: no
// signal, and the record stays for the next pass.
func TestLiveNvimReapLeavesALiveTagsEmbedAlone(t *testing.T) {
	nvim := liveNvim(t)
	scopeDir := t.TempDir()
	rt := OSRuntime{DataDir: scopeDir}
	tag := "live-owned"
	embed := spawnLiveEmbed(t, rt.processTable(), nvim, scopeDir, tag)

	found := outcomesFor(liveSweep(t, scopeDir, []string{tag}), embed.pid)
	if len(found) != 1 {
		t.Fatalf("outcomes for the embed = %v, want exactly one decision", found)
	}
	if found[0].Killed {
		t.Errorf("a live tag's embed was killed: %+v", found[0])
	}
	if found[0].Reason != reapReasonOwnerLive {
		t.Errorf("reason = %q, want %q", found[0].Reason, reapReasonOwnerLive)
	}
	if found[0].Removed {
		t.Errorf("an undecided outcome removed the record: %+v", found[0])
	}
	if !embed.alive() {
		t.Error("the embed died although its owner is live")
	}
	embed.assertRecordIntact(t)
}

// A stale record naming a process that is not an embedded nvim: the record goes (a
// decided outcome never leaves its evidence behind), the process does not.
func TestLiveNvimReapLeavesANonNvimProcessAlone(t *testing.T) {
	liveNvim(t)
	scopeDir := t.TempDir()
	paths, err := artifactpath.ResolveScoped(scopeDir, "live-stale")
	if err != nil {
		t.Fatal(err)
	}
	record := paths.NvimPID(nvimKindDraft)

	sleep := exec.Command("sleep", "60")
	if err := sleep.Start(); err != nil {
		t.Fatalf("start the innocent process: %v", err)
	}
	t.Cleanup(func() { _ = sleep.Process.Kill(); _, _ = sleep.Process.Wait() })
	if err := os.WriteFile(record, []byte(strconv.Itoa(sleep.Process.Pid)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	found := outcomesFor(liveSweep(t, scopeDir, nil), sleep.Process.Pid)
	if len(found) != 1 {
		t.Fatalf("the sweep decided the stale record %d times (%v), want once", len(found), found)
	}
	if found[0].Killed {
		t.Error("the sweep signalled a process that is not an embedded nvim")
	}
	if found[0].Reason != reapReasonNotNvim {
		t.Errorf("reason = %q, want %q", found[0].Reason, reapReasonNotNvim)
	}
	if err := syscall.Kill(sleep.Process.Pid, 0); err != nil {
		t.Errorf("the innocent process died: %v", err)
	}
	if _, err := os.Stat(record); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the stale sidecar survived a decided outcome: %v", err)
	}
}

func (e liveEmbed) assertRecordIntact(t *testing.T) {
	t.Helper()
	held, err := os.ReadFile(e.record)
	if err != nil || strings.TrimSpace(string(held)) != strconv.Itoa(e.pid) {
		t.Errorf("the sidecar no longer names %d: %q, %v", e.pid, held, err)
	}
}
