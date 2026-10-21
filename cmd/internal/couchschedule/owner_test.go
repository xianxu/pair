package couchschedule

import (
	"context"
	"encoding/json"
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestOwnerLifetimeExclusionCancellationAndReacquisition(t *testing.T) {
	store := StateStore{Dir: catalogRoot(t)}
	owner, err := store.Own(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	stage := filepath.Join(store.Dir, abandonedStateName)
	if err := os.WriteFile(stage, []byte("live-owner-stage"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if other, err := store.Own(ctx); !errors.Is(err, context.DeadlineExceeded) || other != nil {
		t.Fatalf("contender acquired: %v %v", other, err)
	}
	if _, err := os.Stat(stage); err != nil {
		t.Fatalf("contender touched live stage: %v", err)
	}
	if err := owner.Update(t.Context(), func(l *Ledger) error {
		l.Unknown = map[string]json.RawMessage{"marker": json.RawMessage(`"held"`)}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := owner.Close(); err != nil {
		t.Fatal(err)
	}
	if err := owner.Update(t.Context(), func(*Ledger) error { return nil }); err == nil {
		t.Fatal("closed owner wrote")
	}
	next, err := store.Own(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	if len(ledgerTemps(t, store.Dir)) != 0 {
		t.Fatal("reacquisition did not recover residue")
	}
	got, err := next.ViewDurable(t.Context())
	if err != nil || string(got.Unknown["marker"]) != `"held"` {
		t.Fatalf("lost state: %+v %v", got, err)
	}
}

func TestOwnerCopiedHandleRefusesWithoutReleasingAuthority(t *testing.T) {
	store := StateStore{Dir: catalogRoot(t)}
	owner, err := store.Own(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	copied := *owner
	if err := copied.Update(t.Context(), func(*Ledger) error { return nil }); err == nil {
		t.Fatal("copied capability wrote")
	}
	if _, err := copied.ViewDurable(t.Context()); err == nil {
		t.Fatal("copied capability read durable state")
	}
	if err := copied.Close(); err == nil {
		t.Fatal("copied capability released fence")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if other, err := store.Own(ctx); other != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("copy released authority: %v %v", other, err)
	}
}

func TestOwnerDurableViewRefusesFailedSyncAndCanRecover(t *testing.T) {
	fail := false
	store := StateStore{Dir: catalogRoot(t), BeforeDirectorySync: func() error {
		if fail {
			return errors.New("sync fault")
		}
		return nil
	}}
	owner, err := store.Own(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	fail = true
	if err := owner.Update(t.Context(), func(l *Ledger) error {
		l.Unknown = map[string]json.RawMessage{"marker": json.RawMessage(`"renamed"`)}
		return nil
	}); err == nil {
		t.Fatal("false persistence acknowledgment")
	}
	if _, err := owner.ViewDurable(t.Context()); err == nil {
		t.Fatal("false durable read after failed sync")
	}
	fail = false
	got, err := owner.ViewDurable(t.Context())
	if err != nil || string(got.Unknown["marker"]) != `"renamed"` {
		t.Fatalf("canonical recovery: %+v %v", got, err)
	}
}

// The direct exec step bypasses os/exec's child-fd cleanup: this checks the
// actual lifetime lock descriptor's close-on-exec flag and kernel behavior.
func TestOwnerExecHelper(t *testing.T) {
	mode := os.Getenv("PAIR_OWNER_EXEC_HELPER")
	if mode == "" {
		return
	}
	if mode == "acquire" {
		owner, err := (StateStore{Dir: os.Getenv("PAIR_OWNER_DIR")}).Own(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		_ = owner // deliberately no Close before exec
		env := []string{"PAIR_OWNER_EXEC_HELPER=after-exec"}
		for _, entry := range os.Environ() {
			if !strings.HasPrefix(entry, "PAIR_OWNER_EXEC_HELPER=") {
				env = append(env, entry)
			}
		}
		if err := syscall.Exec(os.Args[0], []string{os.Args[0], "-test.run=^TestOwnerExecHelper$"}, env); err != nil {
			t.Fatal(err)
		}
	} else {
		lock, err := os.Stat(filepath.Join(os.Getenv("PAIR_OWNER_DIR"), ".state.lock"))
		if err != nil {
			t.Fatal(err)
		}
		for fd := 3; fd < 1024; fd++ {
			var st unix.Stat_t
			if unix.Fstat(fd, &st) == nil {
				expected := lock.Sys().(*syscall.Stat_t)
				if uint64(st.Dev) == uint64(expected.Dev) && uint64(st.Ino) == uint64(expected.Ino) {
					t.Fatalf("owner lock inherited through exec on fd %d", fd)
				}
			}
		}
		// If no descriptor leaked, another owner can immediately acquire.
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		owner, err := (StateStore{Dir: os.Getenv("PAIR_OWNER_DIR")}).Own(ctx)
		if err != nil {
			t.Fatal(err)
		}
		owner.Close()
	}
}
func TestOwnerDescriptorDoesNotSurviveExec(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestOwnerExecHelper$")
	cmd.Env = append(os.Environ(), "PAIR_OWNER_EXEC_HELPER=acquire", "PAIR_OWNER_DIR="+catalogRoot(t))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("exec ownership: %v %s", err, out)
	}
}

func TestOwnerCancelledMethodWaitAndCloseJoin(t *testing.T) {
	store := StateStore{Dir: catalogRoot(t)}
	owner, err := store.Own(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	entered, finish := make(chan struct{}), make(chan struct{})
	updateDone := make(chan error, 1)
	go func() {
		updateDone <- owner.Update(t.Context(), func(*Ledger) error { close(entered); <-finish; return nil })
	}()
	<-entered
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	called := false
	if err := owner.Update(ctx, func(*Ledger) error { called = true; return nil }); !errors.Is(err, context.Canceled) || called {
		t.Fatalf("cancelled method waited/entered: %v %v", called, err)
	}
	closeDone := make(chan error, 1)
	go func() { closeDone <- owner.Close() }()
	close(finish)
	if err := <-updateDone; err != nil {
		t.Fatal(err)
	}
	if err := <-closeDone; err != nil {
		t.Fatal(err)
	}
	if _, err := owner.ViewDurable(t.Context()); err == nil {
		t.Fatal("closed owner acknowledged durable view")
	}
	next, err := store.Own(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	next.Close()
}
