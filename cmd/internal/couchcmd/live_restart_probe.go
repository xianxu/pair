package couchcmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// liveRestartProbe gathers pair#421's admission facts. The message service is
// attached after construction, because requests can arrive as soon as its
// socket is up; until then no session is known and admission refuses
// busy-unknown, which is safe.
type liveRestartProbe struct {
	service  atomic.Pointer[messageService]
	git      couchcore.GitRunner
	lookPath func(string) (string, error)
	getenv   func(string) string
	build    func(string) (couchmessage.BuildIdentity, error)
}

func newLiveRestartProbe(git couchcore.GitRunner) *liveRestartProbe {
	return &liveRestartProbe{git: git, lookPath: exec.LookPath, getenv: os.Getenv, build: couchmessage.BuildIdentityOfFile}
}

func (p *liveRestartProbe) LiveRestartFacts(ctx context.Context, op string, row couchcore.ActionableThreadSummary, path string) (couchcore.LiveRestartFacts, error) {
	f := couchcore.LiveRestartFacts{Live: row.Live()}
	live, ok := p.service.Load().LivenessForThread(row.Address.RepoScope, string(row.Address.Tag))
	if ok {
		f.Session = true
		if live.Settled != nil {
			f.SettledKnown, f.Settled = true, *live.Settled
		}
	}
	if status, err := couchcore.ProbeSlotGit(ctx, p.git, path); err == nil {
		f.GitKnown, f.Dirty = true, status.Dirty
	}
	if op == couchcore.OpRelaunch {
		f.Binary = p.binaryFacts(ctx, live)
	}
	return f, nil
}

// binaryFacts compares the slot's running executable with `pair` as Couch
// resolves it — the binary `pair resume` will exec, since slots inherit
// Couch's environment (couchcore.mergeChildEnvironment).
func (p *liveRestartProbe) binaryFacts(ctx context.Context, live couchmessage.SlotLiveness) couchcore.BinaryFacts {
	b := couchcore.BinaryFacts{DevRebuild: p.getenv("PAIR_DEV") != ""}
	if live.Build != nil {
		b.RunningSHA = live.Build.SHA256
	}
	path, err := p.lookPath("pair")
	if err != nil {
		return b
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	b.OnDiskPath = path
	if id, err := p.build(path); err == nil {
		b.OnDiskSHA, b.OnDiskRevision, b.OnDiskModified = id.SHA256, id.Revision, id.Modified
	}
	// The checkout that builds this binary: bin/pair's parent, when that is a
	// git tree (pair:0 in practice, built from cmd/pair-go by make build).
	if bin := filepath.Dir(path); filepath.Base(bin) == "bin" {
		checkout := filepath.Dir(bin)
		if head, err := p.git.RunContext(ctx, checkout, "rev-parse", "HEAD"); err == nil {
			b.Checkout, b.CheckoutHEAD = checkout, strings.TrimSpace(head)
		}
	}
	return b
}
