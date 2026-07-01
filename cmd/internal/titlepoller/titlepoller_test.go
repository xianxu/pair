package titlepoller

import (
	"testing"
	"time"
)

func TestPrefixForAge(t *testing.T) {
	cases := []struct {
		age  time.Duration
		want string
	}{
		{1 * time.Hour, prefixHot + " "},
		{oneDay - time.Second, prefixHot + " "},
		{oneDay, prefixWarm + " "},
		{threeDays - time.Second, prefixWarm + " "},
		{threeDays, prefixLukewarm + " "},
		{tenDays - time.Second, prefixLukewarm + " "},
		{tenDays, prefixCool + " "},
		{twentyOneDays - time.Second, prefixCool + " "},
		{twentyOneDays, ""},
		{100 * 24 * time.Hour, ""},
	}
	for _, c := range cases {
		if got := prefixForAge(c.age); got != c.want {
			t.Errorf("prefixForAge(%v) = %q, want %q", c.age, got, c.want)
		}
	}
}

func TestAbbrevCwd(t *testing.T) {
	home := "/Users/x"
	cases := []struct{ path, want string }{
		{"/Users/x", "~"},
		{"/Users/x/repo", "~/repo"},
		{"/Users/x/a/b", "~/a/b"},
		{"/Users/xyz", "/Users/xyz"}, // not under $HOME/ — no abbreviation
		{"/etc", "/etc"},
	}
	for _, c := range cases {
		if got := abbrevCwd(c.path, home); got != c.want {
			t.Errorf("abbrevCwd(%q) = %q, want %q", c.path, got, c.want)
		}
	}
	if got := abbrevCwd("/Users/x", ""); got != "/Users/x" {
		t.Errorf("abbrevCwd with empty home should return path unchanged, got %q", got)
	}
}

func TestFrameTitle(t *testing.T) {
	if got := frameTitle("claude", "970k", "~/repo"); got != "claude (970k) [~/repo]" {
		t.Errorf("with count: %q", got)
	}
	if got := frameTitle("claude", "", "~/repo"); got != "claude [~/repo]" {
		t.Errorf("no count: %q", got)
	}
}

func TestCmuxWorkspaceTitle(t *testing.T) {
	if got := cmuxWorkspaceTitle("🔴 ", "pair-brain"); got != "🔴 ♋-🧠" {
		t.Errorf("substitutions: %q", got)
	}
	if got := cmuxWorkspaceTitle("", "pair-21"); got != "♋-21" {
		t.Errorf("cold prefix: %q", got)
	}
}

// Mirrors the shell harness Part A (poller_alive identity guard).
func TestPollerArgvMatches(t *testing.T) {
	const bin = "/Users/x/.local/share/pair/runtime/abc/pair-home/bin/pair-title"
	cases := []struct {
		name string
		argv string
		tag  string
		want bool
	}{
		{"live poller for the tag", bin + " 211 claude", "211", true},
		{"recycled pid (unrelated process)", "/usr/sbin/cupsd", "211", false},
		{"tag prefix collision 21 vs 211", bin + " 211 claude", "21", false},
		{"empty argv", "", "211", false},
		{"empty tag", bin + " 211 claude", "", false},
		{"the .sh shim itself is not the running poller", "/bin/bash /x/bin/pair-title.sh 211 claude", "211", false},
	}
	for _, c := range cases {
		if got := pollerArgvMatches(c.argv, c.tag); got != c.want {
			t.Errorf("%s: pollerArgvMatches(%q, %q) = %v, want %v", c.name, c.argv, c.tag, got, c.want)
		}
	}
}

func TestShouldClaimWorkspace(t *testing.T) {
	cases := []struct {
		owner, tag string
		alive      bool
		want       bool
	}{
		{"", "21", false, true},   // unowned → claim
		{"21", "21", true, true},  // ours → claim
		{"99", "21", true, false}, // another live owner → defer
		{"99", "21", false, true}, // stale owner (session dead) → reclaim
	}
	for _, c := range cases {
		if got := shouldClaimWorkspace(c.owner, c.tag, c.alive); got != c.want {
			t.Errorf("shouldClaimWorkspace(%q,%q,alive=%v) = %v, want %v", c.owner, c.tag, c.alive, got, c.want)
		}
	}
}

func TestFrameCacheUnchangedSkip(t *testing.T) {
	c := frameCache{}
	if !c.changed("7", "claude (970k) [~/repo]") {
		t.Fatal("first write must be a change")
	}
	if c.changed("7", "claude (970k) [~/repo]") {
		t.Fatal("identical title must be skipped")
	}
	if !c.changed("7", "claude (980k) [~/repo]") {
		t.Fatal("new title must be a change")
	}
	if !c.changed("8", "claude (970k) [~/repo]") {
		t.Fatal("different pane must be a change")
	}
}
