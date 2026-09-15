package couchcmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"golang.org/x/term"
)

// This layer exercises client PTY -> actual Pair config -> raw pane input. It
// does not run Couch or a wrapper; portable composed acceptance covers those.
func TestAgentShortcutInputConformanceLive(t *testing.T) {
	if os.Getenv("PAIR_LIVE_COUCH") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1 for disposable real Zellij input conformance")
	}
	config, err := filepath.Abs("../../../zellij/config.kdl")
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	t.Setenv("TMPDIR", "/tmp") // Keep Darwin Unix-socket paths below their limit.
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME"} {
		t.Setenv(key, t.TempDir())
	}
	t.Setenv("SHELL", "/bin/sh")
	for _, key := range []string{"ZELLIJ", "ZELLIJ_SESSION_NAME", "ZELLIJ_PANE_ID", "PAIR_SESSION", "PAIR_TAG", "PAIR_DATA_DIR", "COUCH_MOUSE_TRACE", "COUCH_TRACE"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PAIR_SHORTCUT_CAPTURE", base)
	// A regression to consuming help/changelog bindings must not invoke real
	// workbench commands. Such execution remains a visible test failure.
	stubDir := filepath.Join(base, "bin")
	if err := os.Mkdir(stubDir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"pair", "pair-help"} {
		if err := os.WriteFile(filepath.Join(stubDir, name), []byte("#!/bin/sh\nprintf called > \"$PAIR_SHORTCUT_CAPTURE/unexpected-command\"\nexit 0\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	layout := filepath.Join(base, "layout.kdl")
	body := fmt.Sprintf("layout {\n pane split_direction=\"vertical\" {\n pane command=%s { args \"-test.run=^TestShortcutRawRecorder$\"; }\n pane command=%s focus=true { args \"-test.run=^TestShortcutRawRecorder$\"; }\n }\n}\n", strconv.Quote(binary), strconv.Quote(binary))
	if err := os.WriteFile(layout, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	var entropy [10]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		t.Fatal(err)
	}
	session := "pair-keys-" + hex.EncodeToString(entropy[:])
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fixture, err := pairlifecycletest.StartControlledZellijWithOptions(ctx, session, pairlifecycletest.ZellijOptions{ConfigFile: config, LayoutFile: layout})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fixture.Close() })
	waitShortcutCapture(t, ctx, func() bool { files, _ := filepath.Glob(filepath.Join(base, "ready-*")); return len(files) == 2 })
	// Identify the pane that actually receives client input rather than assuming
	// Zellij pane IDs or layout focus conventions.
	if _, err := fixture.WriteInput([]byte("READY")); err != nil {
		t.Fatal(err)
	}
	var capture string
	waitShortcutCapture(t, ctx, func() bool {
		files, _ := filepath.Glob(filepath.Join(base, "input-*"))
		for _, file := range files {
			b, _ := os.ReadFile(file)
			if string(b) == "READY" {
				capture = file
				return true
			}
		}
		return false
	})
	snapshot := func() string {
		t.Helper()
		out, err := exec.CommandContext(ctx, "zellij", "--session", session, "action", "dump-layout").CombinedOutput()
		if err != nil {
			t.Fatalf("dump fixture layout: %v", err)
		}
		return string(out)
	}
	initial := snapshot()
	expected := []byte("READY")
	for _, tc := range []struct{ name, input, want string }{
		{"inherited-alt-f", "\x1bf", "\x1bf"},
		{"inherited-alt-bracket", "\x1b[", "\x1b["},
		{"inherited-alt-plus", "\x1b+", "\x1b+"},
		{"alt-up", "\x1b[1;3A", "\x1b[1;3A"},
		{"alt-down", "\x1b[1;3B", "\x1b[1;3B"},
		{"alt-left", "\x1b[1;3D", "\x1b[1;3D"},
		{"alt-right", "\x1b[1;3C", "\x1b[1;3C"},
		{"alt-h", "\x1bh", "\x1b[104;3u"},
		{"alt-l", "\x1bl", "\x1b[108;3u"},
		{"alt-x", "\x1bx", "\x1b[120;3u"},
		{"reserved-shift-alt-t", "\x1b[84;4u", "\x1b[84;4u"},
		{"reserved-shift-alt-left", "\x1b[1;4D", "\x1b[1;4D"},
		{"reserved-shift-alt-right", "\x1b[1;4C", "\x1b[1;4C"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Observe the key at the receiver before sending the barrier. Bare
			// ESC+[ followed by ! is an incomplete CSI if the parser has not
			// resolved Alt+[ yet; sender-side sleep cannot establish that.
			for _, delivery := range []struct{ input, want string }{{tc.input, tc.want}, {"!", "!"}} {
				if _, err := fixture.WriteInput([]byte(delivery.input)); err != nil {
					t.Fatal(err)
				}
				expected = append(expected, []byte(delivery.want)...)
				deadline := time.Now().Add(time.Second)
				var got []byte
				for time.Now().Before(deadline) {
					got, _ = os.ReadFile(capture)
					if bytes.Equal(got, expected) {
						break
					}
					time.Sleep(10 * time.Millisecond)
				}
				if !bytes.Equal(got, expected) {
					t.Fatalf("client input %q delivery %q: capture %q, want %q", tc.input, delivery.input, got, expected)
				}
			}
			if got := snapshot(); got != initial {
				t.Fatal("shortcut changed pane/tab/layout state")
			}
			if _, err := os.Stat(filepath.Join(base, "unexpected-command")); !os.IsNotExist(err) {
				t.Fatal("shortcut executed a workbench command")
			}
		})
		if t.Failed() {
			break
		}
	}
}

func waitShortcutCapture(t *testing.T, ctx context.Context, ready func() bool) {
	t.Helper()
	for !ready() {
		select {
		case <-ctx.Done():
			t.Fatalf("fixture readiness: %v", ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// Test-binary subprocess, with no coding agent or user shell startup files.
func TestShortcutRawRecorder(t *testing.T) {
	base := os.Getenv("PAIR_SHORTCUT_CAPTURE")
	pane := os.Getenv("ZELLIJ_PANE_ID")
	if base == "" || pane == "" {
		t.Skip("only used by disposable Zellij fixture")
	}
	if strings.ContainsAny(pane, "/\\") {
		t.Fatal("invalid fixture pane ID")
	}
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), state)
	capture, err := os.OpenFile(filepath.Join(base, "input-"+pane), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer capture.Close()
	if err := os.WriteFile(filepath.Join(base, "ready-"+pane), nil, 0600); err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(capture, os.Stdin)
}
