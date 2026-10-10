package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

type fakeProbeGit struct{ head string }

func (g fakeProbeGit) Run(dir string, args ...string) (string, error) {
	return g.RunContext(context.Background(), dir, args...)
}
func (g fakeProbeGit) RunContext(_ context.Context, dir string, args ...string) (string, error) {
	if len(args) >= 2 && args[0] == "rev-parse" && args[1] == "HEAD" {
		if g.head == "" {
			return "", errors.New("not a repository")
		}
		return g.head + "\n", nil
	}
	return "", errors.New("unexpected git " + strings.Join(args, " "))
}

// The probe's binary half: `pair` as Couch resolves it, its checkout (bin/'s
// parent), PAIR_DEV, and the running wrapper's hash from its session.
func TestLiveRestartProbeBinaryFacts(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "pair", "bin")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	pair := filepath.Join(bin, "pair")
	if err := os.WriteFile(pair, []byte("binary"), 0700); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{}
	p := &liveRestartProbe{
		git:      fakeProbeGit{head: "headrev"},
		lookPath: func(string) (string, error) { return pair, nil },
		getenv:   func(k string) string { return env[k] },
		build: func(string) (couchmessage.BuildIdentity, error) {
			return couchmessage.BuildIdentity{SHA256: "disk", Revision: "builtrev", Modified: true}, nil
		},
	}
	running := couchmessage.SlotLiveness{Build: &couchmessage.BuildIdentity{SHA256: "running"}}
	resolved, _ := filepath.EvalSymlinks(pair)
	got := p.binaryFacts(context.Background(), running)
	want := couchcore.BinaryFacts{RunningSHA: "running", OnDiskPath: resolved, OnDiskSHA: "disk", OnDiskRevision: "builtrev", OnDiskModified: true,
		Checkout: filepath.Dir(filepath.Dir(resolved)), CheckoutHEAD: "headrev"}
	if got != want {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
	env["PAIR_DEV"] = "1"
	if !p.binaryFacts(context.Background(), running).DevRebuild {
		t.Fatal("PAIR_DEV not seen")
	}
	// A legacy wrapper (no build) and a binary outside a git checkout.
	p.git = fakeProbeGit{}
	got = p.binaryFacts(context.Background(), couchmessage.SlotLiveness{})
	if got.RunningSHA != "" || got.Checkout != "" || got.CheckoutHEAD != "" {
		t.Fatalf("unknowns invented: %+v", got)
	}
}

func TestNotedResultForwardsAndJoins(t *testing.T) {
	n := notedResult{value: couchcore.RelaunchResult{Outcome: couchcore.Relaunched}, note: "built from x"}
	if _, ok := n.Started(); !ok {
		t.Fatal("Started not forwarded")
	}
	out := slotOperationOutcome(n, nil)
	if out.Status != couchmessage.ReceiptSucceeded || out.Warning != "built from x" {
		t.Fatalf("outcome %+v", out)
	}
}
