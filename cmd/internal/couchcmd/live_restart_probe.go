package couchcmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// liveRestartProbe gathers pair#421's admission facts. The message service is
// attached after construction, because requests can arrive as soon as its
// socket is up; until then no session is known and admission refuses
// busy-unknown, which is safe.
type liveRestartProbe struct {
	service atomic.Pointer[messageService]
	git     couchcore.GitRunner
	proc    couchcore.ProcOps
	// reload-context's confirmation wait (pair#421 PQ-4): bounded, polled.
	confirmWithin, pollEvery time.Duration
	lookPath                 func(string) (string, error)
	getenv                   func(string) string
	build                    func(string) (couchmessage.BuildIdentity, error)
}

func newLiveRestartProbe(git couchcore.GitRunner, proc couchcore.ProcOps) *liveRestartProbe {
	return &liveRestartProbe{git: git, proc: proc, lookPath: exec.LookPath, getenv: os.Getenv, build: couchmessage.BuildIdentityOfFile,
		confirmWithin: 20 * time.Second, pollEvery: 100 * time.Millisecond}
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

// RestartConversation is reload-context's effect (pair#421). It signals the
// wrapper the BROKER holds for the thread, never a pid file (PQ-1), after
// re-checking that the PID still names the process that said hello (Start
// is its kernel start token). SIGUSR2 is `pair agent restart`'s mechanism: the
// wrapper ends its agent and re-execs into a fresh conversation. The re-exec
// keeps the binding byte-identical, so a NEW session token is the evidence.
func (p *liveRestartProbe) RestartConversation(ctx context.Context, address couchcore.ThreadAddress) error {
	svc := p.service.Load()
	scope, tag := address.RepoScope, string(address.Tag)
	before, ok := svc.LivenessForThread(scope, tag)
	if !ok {
		return &couchcore.SlotOperationError{Code: couchcore.LiveRestartBusyUnknown, Detail: "no wrapper session is connected for " + tag + "; nothing was signalled"}
	}
	if p.proc == nil {
		return &couchcore.SlotOperationError{Code: couchcore.LiveRestartUnavailable, Detail: "this Couch cannot signal processes"}
	}
	pid := before.Binding.PID
	if identity, err := p.proc.Identity(pid); err != nil || identity != before.Binding.Start {
		return &couchcore.SlotOperationError{Code: couchcore.LiveRestartBusyUnknown,
			Detail: fmt.Sprintf("wrapper pid %d no longer matches the process that connected; nothing was signalled", pid)}
	}
	if err := p.proc.Signal(pid, syscall.SIGUSR2); err != nil {
		return fmt.Errorf("signal wrapper %d: %w", pid, err)
	}
	deadline := time.NewTimer(p.confirmWithin)
	defer deadline.Stop()
	tick := time.NewTicker(p.pollEvery)
	defer tick.Stop()
	for {
		if after, ok := svc.LivenessForThread(scope, tag); ok && after.Session != before.Session {
			return nil
		}
		select {
		case <-ctx.Done():
			return &couchcore.ReloadUnconfirmed{Detail: "the signal was delivered but the wait was cancelled; peek the slot before retrying"}
		case <-deadline.C:
			return &couchcore.ReloadUnconfirmed{Detail: fmt.Sprintf("the signal was delivered but no new wrapper session appeared within %s; peek the slot before retrying", p.confirmWithin)}
		case <-tick.C:
		}
	}
}

// withAdmissionNote adapts a queue job's outcome hook so the admission note
// (freshness, an override used) reaches the receipt on success AND failure
// (pair#421 M2 review). Extracted so both paths are tested.
func withAdmissionNote(note *string, finished func(any, error)) func(any, error) {
	return func(value any, err error) {
		if *note != "" {
			value = notedResult{value: value, note: *note}
		}
		finished(value, err)
	}
}
