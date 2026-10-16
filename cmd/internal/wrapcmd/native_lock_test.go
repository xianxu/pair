package wrapcmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/nativelock"
	"github.com/xianxu/pair/cmd/internal/sessionwatch"
)

func TestWrapperKnownLocksSurviveStopUntilCleanup(t *testing.T) {
	home := t.TempDir()
	owner := newWrapperNativeLocks(home, "codex", func() string { return "pair:2" })
	key, _ := nativelock.NewKey("codex", "id")
	if err := owner.Discover("id"); err != nil {
		t.Fatal(err)
	}
	if err := owner.Discover("id"); err != nil {
		t.Fatal(err)
	}
	owner.Stop()
	if _, err := nativelock.Acquire(home, key, "pair:3"); err == nil || !strings.Contains(err.Error(), "pair:2") {
		t.Fatalf("released before group cleanup: %v", err)
	}
	if err := owner.Discover("new"); err == nil {
		t.Fatal("accepted late discovery")
	}
	owner.Close()
	h, err := nativelock.Acquire(home, key, "pair:3")
	if err != nil {
		t.Fatal(err)
	}
	h.Close()
}

func TestBindingPipeBoundsCurrentLaunchAndJoins(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stream := readBindingNotices(reader, 7)
	if err := sessionwatch.NotifyBinding(writer, 7, "id"); err != nil {
		t.Fatal(err)
	}
	event := <-stream.events
	if event.id != "id" || event.err != nil {
		t.Fatalf("notice %+v", event)
	}
	writer.Close()
	event = <-stream.events
	if event.err == nil {
		t.Fatal("EOF must be explicit")
	}
	stream.Close()
	stream.Close()
}

func TestNonNativeWrapIgnoresAmbientConversation(t *testing.T) {
	t.Setenv("PAIR_SESSION_ID", "parent-id")
	owner := newWrapperNativeLocks(t.TempDir(), "sh", func() string { return "standalone" })
	defer owner.Close()
	spawned := false
	_, err := startNativePTY(owner, exec.Command("/bin/sh"), io.Discard, func(*exec.Cmd) (*os.File, error) { spawned = true; return nil, nil })
	if err != nil || !spawned {
		t.Fatalf("non-native rejected: %v spawn=%v", err, spawned)
	}
}

func TestKnownNewAndForkDestinationsLockBeforeSpawn(t *testing.T) {
	for _, args := range [][]string{{"--session-id", "new"}, {"--resume", "old", "--fork-session", "--session-id=new"}} {
		t.Setenv("PAIR_SESSION_ID", "")
		home := t.TempDir()
		key, _ := nativelock.NewKey("claude", "new")
		h, err := nativelock.Acquire(home, key, "pair:3")
		if err != nil {
			t.Fatal(err)
		}
		owner := newWrapperNativeLocks(home, "claude", func() string { return "pair:2" })
		spawned := false
		_, err = startNativePTY(owner, exec.Command("claude", args...), io.Discard, func(*exec.Cmd) (*os.File, error) { spawned = true; return nil, nil })
		owner.Close()
		h.Close()
		if err == nil || spawned || !strings.Contains(err.Error(), "pair:3") {
			t.Fatalf("known destination spawned: %v spawn=%v", err, spawned)
		}
	}
}

func TestIndirectCodexNeverLocksASelectorOrInheritedID(t *testing.T) {
	for _, args := range [][]string{{"resume", "--last"}, {"resume", "my-thread"}} {
		home := t.TempDir()
		t.Setenv("PAIR_SESSION_ID", "inherited")
		owner := newWrapperNativeLocks(home, "codex", func() string { return "pair:2" })
		var notice bytes.Buffer
		_, err := startNativePTY(owner, exec.Command("codex", args...), &notice, func(*exec.Cmd) (*os.File, error) { return nil, nil })
		if err != nil || !owner.Unbound() || !strings.Contains(notice.String(), "not guarded") {
			t.Fatalf("indirect selector treated as identity: %v unbound=%v notice=%s", err, owner.Unbound(), &notice)
		}
		owner.Close()
	}
}

// startOwnedSleeper is a real owned agent group for wrapper lifecycle tests.
func startOwnedSleeper(t *testing.T) *ownedGroup {
	t.Helper()
	cmd := exec.Command("/bin/sh", "-c", "sleep 60 & wait")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) })
	return newOwnedGroup(cmd, time.Second)
}

// pair#595 close review BR-2/BR-5: a binding notice that arrives once an agent
// restart has begun is dropped, so the wrapper still re-execs instead of
// failing; the lock taken before the restart is held until cleanup.
func TestRestartThenLateBindingNoticeStillRestarts(t *testing.T) {
	home := t.TempDir()
	locks := newWrapperNativeLocks(home, "codex", func() string { return "pair:2" })
	if err := locks.Discover("known"); err != nil {
		t.Fatal(err)
	}
	group := startOwnedSleeper(t)
	p := &proxy{nativeLocks: locks, ownedGroup: group}
	p.stopForSignal(syscall.SIGUSR2, locks, group)
	p.handleBindingNotice(nativeBindingEvent{id: "late"})
	_ = group.Wait()
	if p.nativeLockErr != nil || !p.restartFresh.Load() {
		t.Fatalf("restart turned into a failure: err=%v restart=%v", p.nativeLockErr, p.restartFresh.Load())
	}
	key, _ := nativelock.NewKey("codex", "known")
	if _, err := nativelock.Acquire(home, key, "pair:3"); err == nil {
		t.Fatal("lock released before cleanup")
	}
	locks.Close()
}

// A notice whose ID cannot name a lock is logged and dropped; the agent keeps
// running.
func TestUnusableBindingNoticeLeavesAgentRunning(t *testing.T) {
	locks := newWrapperNativeLocks(t.TempDir(), "codex", func() string { return "pair:2" })
	group := startOwnedSleeper(t)
	p := &proxy{nativeLocks: locks, ownedGroup: group}
	p.handleBindingNotice(nativeBindingEvent{id: "bad/../id"})
	if p.nativeLockErr != nil || !groupHasLiveMember(group.cmd.Process.Pid) {
		t.Fatalf("unusable notice stopped the agent: %v", p.nativeLockErr)
	}
	locks.Close()
}

// A notice for a conversation another live wrapper holds is a late collision:
// the owned group stops and the wrapper reports the holder.
func TestLateCollisionStopsOwnedGroupAndNamesHolder(t *testing.T) {
	home := t.TempDir()
	other := newWrapperNativeLocks(home, "codex", func() string { return "pair:3" })
	if err := other.Discover("shared"); err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	locks := newWrapperNativeLocks(home, "codex", func() string { return "pair:2" })
	group := startOwnedSleeper(t)
	p := &proxy{nativeLocks: locks, ownedGroup: group}
	p.handleBindingNotice(nativeBindingEvent{id: "shared"})
	done := make(chan struct{})
	go func() { _ = group.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("late collision did not stop the owned group")
	}
	if p.nativeLockErr == nil || !strings.Contains(p.nativeLockErr.Error(), "conversation shared is in use by pair:3") {
		t.Fatalf("collision not reported with its holder: %v", p.nativeLockErr)
	}
	locks.Close()
}

// The holder label is resolved once, and only when a conversation is locked.
func TestHolderLabelResolvesOnlyOnFirstLock(t *testing.T) {
	calls := 0
	locks := newWrapperNativeLocks(t.TempDir(), "codex", func() string { calls++; return "pair:2" })
	locks.Stop()
	locks.Close()
	if calls != 0 {
		t.Fatalf("label resolved for a wrap that never locked: %d", calls)
	}
	locks = newWrapperNativeLocks(t.TempDir(), "codex", func() string { calls++; return "pair:2" })
	for _, id := range []string{"a", "b"} {
		if err := locks.Discover(id); err != nil {
			t.Fatal(err)
		}
	}
	locks.Close()
	if calls != 1 {
		t.Fatalf("label resolved %d times, want once", calls)
	}
}
