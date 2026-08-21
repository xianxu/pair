package couchcore

import (
	"strings"
	"testing"
)

func TestFakeGitKeysOnDirNotJustArgv(t *testing.T) {
	// Ignoring dir is what makes "was git run in the right directory" -- the
	// only bug Resolve can have -- untestable, and it is what made an earlier
	// symlink test vacuous.
	g := NewFakeGit(map[GitCall]string{
		{Dir: "/a", Args: "rev-parse --show-toplevel"}: "/a\n",
	})
	if _, err := g.Run("/b", "rev-parse", "--show-toplevel"); err == nil {
		t.Fatal("a reply canned for /a must not answer a call made in /b")
	}
	out, err := g.Run("/a", "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatalf("Run in /a: %v", err)
	}
	if out != "/a" {
		t.Fatalf("out = %q; trailing newline must be trimmed", out)
	}
}

func TestFakeGitRecordsOps(t *testing.T) {
	g := NewFakeGit(map[GitCall]string{{Dir: "/a", Args: "rev-parse --show-toplevel"}: "/a"})
	_, _ = g.Run("/a", "rev-parse", "--show-toplevel")
	if len(g.Ops) != 1 || g.Ops[0] != "/a: rev-parse --show-toplevel" {
		t.Fatalf("Ops = %v", g.Ops)
	}
}

func TestFakePathOpsMapsAndErrors(t *testing.T) {
	p := NewFakePathOps(map[string]string{"/link": "/real"})
	got, err := p.Physical("/link")
	if err != nil || got != "/real" {
		t.Fatalf("Physical(/link) = %q, %v", got, err)
	}
	if _, err := p.Physical("/plain"); err != nil {
		t.Fatalf("an unmapped path must pass through, got error %v", err)
	}
	p.Fail("/gone")
	if _, err := p.Physical("/gone"); err == nil {
		t.Fatal("Physical must return an error, not silently fall back -- an " +
			"unresolvable path would otherwise become its own identity")
	}
}

func TestOSPathOpsResolvesRealSymlink(t *testing.T) {
	dir := t.TempDir()
	real, err := OSPathOps{}.Physical(dir)
	if err != nil {
		t.Fatalf("Physical: %v", err)
	}
	if !strings.HasPrefix(real, "/") {
		t.Fatalf("Physical = %q", real)
	}
	if _, err := (OSPathOps{}).Physical(dir + "/does-not-exist"); err == nil {
		t.Fatal("a nonexistent path must error")
	}
}
