package couchcore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/xianxu/ariadne/pkg/layergraph"
)

// PrepareAdmission is the running Couch's answer to `couch --prepare`'s
// admission (pair#434): the thread that stands for the slot, and the checkout
// the CLI's refresh runs in.
type PrepareAdmission struct {
	Address ThreadAddress
	Path    string
}

// ReceiptTag names the admitted thread.
func (a PrepareAdmission) ReceiptTag() string { return string(a.Address.Tag) }

// ReceiptPath names the admitted checkout.
func (a PrepareAdmission) ReceiptPath() string { return a.Path }

// Prepare step names and statuses, in the order RunPrepare takes them.
const (
	PrepareAdmit   = "admit"
	PrepareRefresh = "refresh"
	PrepareRecheck = "recheck"
	PrepareReload  = "reload"

	PrepareOK      = "ok"
	PrepareRefused = "refused"
	PrepareFailed  = "failed"
	PrepareSkipped = "skipped"

	// PrepareRefreshDirty: weave refresh left the checkout modified, so the
	// agent was not reloaded onto it.
	PrepareRefreshDirty = "refresh-dirty"
)

// PrepareStep is one step's outcome; Code is a refusal's code.
type PrepareStep struct {
	Name, Status string
	Code, Detail string `json:",omitempty"`
}

// SubstrateCommit is the commit one declared substrate is at after the
// refresh: what the slot's layer tools (sdlc, weave) now build from.
type SubstrateCommit struct {
	Path, Commit, Subject string
}

// PrepareReport is what `couch --prepare` did, one row per step.
type PrepareReport struct {
	Target, Path string
	Steps        []PrepareStep
	// Head and OriginMain are the checkout's commits after the refresh.
	Head, OriginMain string            `json:",omitempty"`
	Substrates       []SubstrateCommit `json:",omitempty"`
}

// OK reports whether every step succeeded.
func (r PrepareReport) OK() bool {
	for _, s := range r.Steps {
		if s.Status != PrepareOK {
			return false
		}
	}
	return len(r.Steps) > 0
}

// PrepareRun is RunPrepare's world: the two remote halves (the running
// Couch's admission and reload-context), and the command seam the refresh
// and git probes run through.
type PrepareRun struct {
	Target   string
	Admit    func(context.Context) (PrepareAdmission, error)
	Reload   func(context.Context) error
	IO       ProvisionIO
	Progress io.Writer
	Timeout  time.Duration
}

// RunPrepare brings an admitted slot up for new work (pair#434): admit, weave
// refresh, re-check the checkout, reload the agent's context. A step that
// does not succeed stops the run; the steps after it are skipped. Nothing is
// journaled: running it again converges, since refresh is idempotent.
func RunPrepare(ctx context.Context, run PrepareRun) PrepareReport {
	report := PrepareReport{Target: run.Target}
	steps := []string{PrepareAdmit, PrepareRefresh, PrepareRecheck, PrepareReload}
	stop := func(step PrepareStep) PrepareReport {
		report.Steps = append(report.Steps, step)
		for _, name := range steps[len(report.Steps):] {
			report.Steps = append(report.Steps, PrepareStep{Name: name, Status: PrepareSkipped})
		}
		return report
	}
	admission, err := run.Admit(ctx)
	if err != nil {
		return stop(prepareFailure(PrepareAdmit, err))
	}
	report.Path = admission.Path
	report.Steps = append(report.Steps, PrepareStep{Name: PrepareAdmit, Status: PrepareOK, Detail: "idle, clean, on its resting branch"})

	git := func(args ...string) (string, error) {
		out, err := run.IO.Run(ctx, ProvisionCommand{Dir: report.Path, Program: "git", Args: args, Timeout: 10 * time.Second})
		return strings.TrimSpace(string(out)), err
	}
	_, err = run.IO.Run(ctx, ProvisionCommand{Dir: report.Path, Program: "weave", Args: []string{"refresh"},
		Progress: run.Progress, StreamOutput: true, Timeout: run.Timeout})
	report.Head, _ = git("rev-parse", "HEAD")
	report.OriginMain, _ = git("rev-parse", "--verify", "--quiet", "refs/remotes/origin/main")
	report.Substrates = substrateCommits(report.Path, git)
	if err != nil {
		return stop(PrepareStep{Name: PrepareRefresh, Status: PrepareFailed, Detail: err.Error()})
	}
	report.Steps = append(report.Steps, PrepareStep{Name: PrepareRefresh, Status: PrepareOK})

	// -z: entries are NUL-terminated and never quoted; each is "XY path", and
	// a rename or copy is followed by its source path as a field of its own.
	status, err := run.IO.Run(ctx, ProvisionCommand{Dir: report.Path, Program: "git", Args: []string{"status", "--porcelain=v1", "-z", "--untracked-files=no"}, Timeout: 10 * time.Second})
	var paths []string
	fields := strings.Split(string(status), "\x00")
	for i := 0; i < len(fields); i++ {
		if entry := fields[i]; len(entry) > 3 {
			paths = append(paths, entry[3:])
			if entry[0] == 'R' || entry[0] == 'C' {
				i++
			}
		}
	}
	switch {
	case err != nil:
		return stop(PrepareStep{Name: PrepareRecheck, Status: PrepareFailed, Detail: "read the checkout's status: " + err.Error()})
	case len(paths) > 0:
		return stop(PrepareStep{Name: PrepareRecheck, Status: PrepareRefused, Code: PrepareRefreshDirty,
			Detail: "weave refresh left " + strings.Join(paths, ", ") + " modified; commit that compile output on main, then prepare again (the agent was not reloaded)"})
	}
	report.Steps = append(report.Steps, PrepareStep{Name: PrepareRecheck, Status: PrepareOK, Detail: "clean"})

	if err := run.Reload(ctx); err != nil {
		return stop(prepareFailure(PrepareReload, err))
	}
	report.Steps = append(report.Steps, PrepareStep{Name: PrepareReload, Status: PrepareOK, Detail: "fresh agent conversation"})
	return report
}

// prepareFailure: a typed slot refusal keeps its code; anything else failed.
func prepareFailure(name string, err error) PrepareStep {
	var refusal *SlotOperationError
	if errors.As(err, &refusal) {
		return PrepareStep{Name: name, Status: PrepareRefused, Code: refusal.Code, Detail: refusal.Detail}
	}
	return PrepareStep{Name: name, Status: PrepareFailed, Detail: err.Error()}
}

// substrateCommits names the commit each present declared substrate is at.
// A substrate that cannot be read is named with its error, never dropped.
func substrateCommits(root string, git func(...string) (string, error)) []SubstrateCommit {
	declared, err := layergraph.DeclaredSubstrates(layergraph.OSFS{}, root)
	if err != nil {
		return []SubstrateCommit{{Path: root, Subject: "substrates unknown: " + err.Error()}}
	}
	var out []SubstrateCommit
	for _, d := range declared {
		if !d.Present {
			continue
		}
		line, err := git("-C", d.Path, "log", "-1", "--format=%H %s")
		if err != nil {
			out = append(out, SubstrateCommit{Path: d.Path, Subject: fmt.Sprintf("unreadable: %v", err)})
			continue
		}
		commit, subject, _ := strings.Cut(line, " ")
		out = append(out, SubstrateCommit{Path: d.Path, Commit: commit, Subject: subject})
	}
	return out
}
