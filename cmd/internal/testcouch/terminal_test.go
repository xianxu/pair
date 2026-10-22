package testcouch

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

func TestTerminalUsesEmptyExplicitEnvironmentAndJoins(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.Command("/bin/sh", "-c", "printf 'private:%s\\n' \"$HOME\"; read line; printf 'got:%s\\n' \"$line\"")
	cmd.Env = []string{"HOME=/owned/home", "TERM=xterm-256color"}
	p, err := startTerminal(cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close(ctx)
	if err := waitUntil(ctx, func() (bool, error) { return strings.Contains(p.Screen(), "private:/owned/home"), nil }); err != nil {
		t.Fatal(err)
	}
	if err := p.Input([]byte("hello\r")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-p.done:
	case <-ctx.Done():
		t.Fatal("PTY child or reader not joined")
	}
	if !strings.Contains(p.Output(), "got:hello") {
		t.Fatalf("lost child output: %q", p.Output())
	}
}

func TestTerminalFailureStopsInputBeforeAnEffect(t *testing.T) {
	want := errors.New("terminal feed failed")
	p := &terminalProcess{readErr: want}
	if err := p.Input([]byte("ordinary key")); !errors.Is(err, want) {
		t.Fatalf("input hid terminal failure: %v", err)
	}
}

func TestTerminalCapturesOwnedFullNoticeClipboard(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	cmd := exec.Command("/bin/sh", "-c", "printf '\\033]52;c;bmF0aXZlLW93bmVyLWRpYWdub3N0aWM=\\007'; read line")
	cmd.Env = []string{"TERM=xterm-256color"}
	p, err := startTerminal(cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close(ctx)
	if err := waitUntil(ctx, func() (bool, error) { return p.copiedText() == "native-owner-diagnostic", p.failure() }); err != nil {
		t.Fatalf("owned OSC52 notice not observed: %q %v", p.copiedText(), err)
	}
}

func TestPhysicalDetachUsesNegotiatedKeyboardPolicy(t *testing.T) {
	for _, flags := range []uint32{1, 3, 8, 10} {
		key := physicalKey(flags, "\x1bd", 100, 3)
		var decoder terminal.Decoder
		events, err := decoder.Feed(key)
		if err != nil || len(events) != 1 {
			t.Fatalf("flags=%d key=%q decode: %+v %v", flags, key, events, err)
		}
		press, ok := events[0].Event.(uv.KeyPressEvent)
		if !ok || uv.Key(press).Code != 'd' || uv.Key(press).Mod != uv.ModAlt {
			t.Fatalf("flags=%d key=%q is not physical Alt+d: %+v", flags, key, events)
		}
		if flags&2 == 0 {
			chord, rest, ok := workbenchshortcut.DecodeChordPrefix(key)
			if !ok || chord != workbenchshortcut.ChordAltD || len(rest) != 0 {
				t.Fatalf("flags=%d key=%q does not invoke public detach", flags, key)
			}
		}
	}
	legacy := physicalKey(0, "\x1bd", 100, 3)
	if _, _, ok := workbenchshortcut.DecodeChordPrefix(legacy); ok {
		t.Fatalf("legacy Alt+d must remain ordinary input: %q", legacy)
	}
	if got := string(physicalKey(3, "\x00", 32, 5)); got != "\x1b[32;5:1u" {
		t.Fatalf("switcher key ignored negotiated event mode: %q", got)
	}
}
