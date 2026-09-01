package couchtty

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/notifycmd"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestNotificationPTYHelper(t *testing.T) {
	if os.Getenv("PAIR_NOTIFICATION_PTY_HELPER") != "1" {
		return
	}
	separator := 0
	for i, arg := range os.Args {
		if arg == "--" {
			separator = i + 1
		}
	}
	if separator == 0 || notifycmd.Run(os.Args[separator:], notifycmd.OSRuntime{}, os.Stderr) != 0 {
		os.Exit(2)
	}
}

func TestNotificationPTYConformance(t *testing.T) {
	f := newFixture(t, 24, 80)
	dir := t.TempDir()
	sidecar := dir + "/outer-tty"
	gate := dir + "/focused"
	message := "review ready"
	command := fmt.Sprintf(
		"tty > %q; %q -test.run '^TestNotificationPTYHelper$' -- %q; while [ ! -f %q ]; do sleep 0.01; done; %q -test.run '^TestNotificationPTYHelper$' -- %q; sleep 1",
		sidecar, os.Args[0], message, gate, os.Args[0], message,
	)
	child, err := ptychild.Start(ptychild.Options{
		Argv: []string{"sh", "-c", command},
		Env: []string{
			"PAIR_NOTIFICATION_PTY_HELPER=1",
			"PAIR_TAG=conformance",
			"PAIR_OUTER_TTY_PATH=" + sidecar,
		},
		Size: f.con.ChildSize(),
		Sink: func(batch ptychild.OutputBatch) { f.con.Deliver("notify", batch) },
	})
	if err != nil {
		t.Fatalf("start notification actor: %v", err)
	}
	t.Cleanup(func() { _ = child.Close() })
	f.con.Attach("notify", "notify", child)

	envelope := notifyosc.Encode(message)
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
	if strings.Count(f.host.Written(), message) < 2 {
		t.Fatalf("outer terminal did not receive both exact envelopes: %q", f.host.Written())
	}

	// Let the helper's bounded sleep finish before cleanup so the watcher owns
	// one ordinary reap path rather than racing Close against process exit.
	deadline := time.Now().Add(2 * time.Second)
	for !child.Done() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
}
