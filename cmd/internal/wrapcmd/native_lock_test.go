package wrapcmd

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/nativelock"
	"github.com/xianxu/pair/cmd/internal/sessionwatch"
)

func TestWrapperKnownLocksSurviveStopUntilCleanup(t *testing.T) {
	home := t.TempDir()
	owner := newWrapperNativeLocks(home, "codex", "pair:2")
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
	owner := newWrapperNativeLocks(t.TempDir(), "sh", "standalone")
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
		owner := newWrapperNativeLocks(home, "claude", "pair:2")
		spawned := false
		_, err = startNativePTY(owner, exec.Command("claude", args...), io.Discard, func(*exec.Cmd) (*os.File, error) { spawned = true; return nil, nil })
		owner.Close()
		h.Close()
		if err == nil || spawned || !strings.Contains(err.Error(), "pair:3") {
			t.Fatalf("known destination spawned: %v spawn=%v", err, spawned)
		}
	}
}
