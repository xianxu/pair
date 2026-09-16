package couchtty

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/notifycmd"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
	"github.com/xianxu/pair/cmd/internal/notifytransport"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestNotificationPTYHelper(t *testing.T) {
	if os.Getenv("PAIR_NOTIFICATION_PTY_HELPER") != "1" {
		return
	}
	broker, err := notifytransport.Start(os.Getenv("PAIR_PAIR_WRAP_PID_PATH"), os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()
	for phase := 0; phase < 2; phase++ {
		if phase == 1 {
			deadline := time.Now().Add(5 * time.Second)
			for {
				if _, err := os.Stat(os.Getenv("PAIR_NOTIFICATION_GATE")); err == nil {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("focus gate timeout")
				}
				time.Sleep(time.Millisecond)
			}
		}
		var diagnostic bytes.Buffer
		if code := notifycmd.Run([]string{"review ready"}, notifycmd.OSRuntime{}, &diagnostic); code != 0 || diagnostic.Len() != 0 {
			t.Fatalf("hook=%d %s", code, diagnostic.String())
		}
		select {
		case message := <-broker.Messages():
			if message != "review ready" {
				t.Fatalf("broker message=%q", message)
			}
			if _, err := os.Stdout.Write(notifyosc.Encode(message)); err != nil {
				t.Fatal(err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("hook did not reach broker")
		}
	}
	time.Sleep(time.Second)
}

func assertPrivateNotificationSocket(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	sockets := 0
	for _, entry := range entries {
		if entry.Type()&os.ModeSocket != 0 {
			sockets++
		}
	}
	if sockets != 1 {
		t.Fatalf("private notification namespace %s contains %d sockets, want 1", dir, sockets)
	}
}

func TestNotificationPTYConformance(t *testing.T) {
	// macOS Unix-domain addresses need a short path. Register cleanup before
	// the Console fixture so its accepted broker child is joined first.
	socketDir, err := os.MkdirTemp("/tmp", "pcnotify-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(socketDir); err != nil {
			t.Errorf("remove private notification fixture: %v", err)
		}
	})
	f := newFixture(t, 24, 80)
	dir := t.TempDir()
	binding := dir + "/wrapper-pid"
	gate := dir + "/focused"
	message := "review ready"

	var env []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "PAIR_") {
			env = append(env, key+"=")
		}
	}
	env = append(env, "PAIR_NOTIFY_SOCKET_DIR="+socketDir, "PAIR_NOTIFICATION_PTY_HELPER=1", "PAIR_TAG=conformance", "PAIR_PAIR_WRAP_PID_PATH="+binding, "PAIR_NOTIFICATION_GATE="+gate)
	child, err := ptychild.Start(ptychild.Options{
		Argv: []string{os.Args[0], "-test.run=^TestNotificationPTYHelper$"},
		Env:  env,
		Size: f.con.ChildSize(),
		Sink: func(ctx context.Context, batch ptychild.OutputBatch) error {
			return f.con.Deliver(ctx, "notify", batch)
		},
	})
	if err != nil {
		t.Fatalf("start notification actor: %v", err)
	}
	f.con.Attach("notify", "notify", child)

	envelope := []byte("\x1b]777;notify;pair;" + message + "\x1b\\")
	waitFor(t, "inactive PTY notification", func() bool {
		f.con.mu.Lock()
		pane := f.con.panes["notify"]
		var retained []AttentionMessage
		if pane != nil {
			retained = f.con.attention.Projection(pane.thread)
		}
		f.con.mu.Unlock()
		return bytes.Count([]byte(f.host.Written()), envelope) == 1 && len(retained) == 1 && retained[0].Text == message
	})

	assertPrivateNotificationSocket(t, socketDir)

	f.con.mu.Lock()
	address := f.con.panes["notify"].thread
	f.con.attention.Acknowledge(f.con.attention.Capture(address))
	f.con.syncAttentionLocked()
	f.con.mu.Unlock()
	f.con.Switch("notify")
	waitFor(t, "notification actor focus", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.focus == FocusActor("notify")
	})
	if err := os.WriteFile(gate, []byte("go\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "focused PTY notification", func() bool {
		return bytes.Count([]byte(f.host.Written()), envelope) == 2
	})
	f.con.mu.Lock()
	retained := f.con.attention.Projection(address)
	f.con.mu.Unlock()
	if len(retained) != 0 {
		t.Fatalf("focused notification retained attention: %+v", retained)
	}

	// Let the helper's bounded sleep finish before cleanup so the watcher owns
	// one ordinary reap path rather than racing Close against process exit.
	deadline := time.Now().Add(2 * time.Second)
	for !child.Done() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
}
