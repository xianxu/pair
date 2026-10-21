package testcouch

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
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
