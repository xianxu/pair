package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/rowtext"
)

// orphanReapOp is the message operation that asks the supervisor for the reap's
// plan. Only the supervisor can produce one: whether a process is unattributed
// is a fact about the whole slot inventory, and a caller that guessed it from a
// projected report would be guessing about a kill (#583).
const orphanReapOp = "reap-orphans"

// orphanReapBudget bounds the whole verb: one fresh classification, then #399's
// escalation for each row it named. A row is self-bounded by that cadence (five
// seconds), so this is a backstop against a classification that named thousands
// of rows rather than a limit an operator will meet.
const orphanReapBudget = 2 * time.Minute

// The outcomes of one named row. Every row the plan named gets exactly one,
// because the operator confirmed each of them and a row that silently vanished
// from the output is a row they cannot account for.
const (
	// orphanRefused is the unconfirmed verb: it looked and did not act.
	orphanRefused = "refused"
	orphanReaped  = "reaped"
	// orphanSkipped is the world having moved -- the pid is gone, or recycled
	// into another generation. Nothing was signalled, and that is success from
	// the operator's side: the leak they confirmed against is no longer there.
	orphanSkipped = "skipped"
	// orphanFailed is a row that survived SIGKILL, or a host that refused the
	// snapshot. This is the only outcome that fails the verb.
	orphanFailed = "failed"
)

// orphanReapOutcome is one row's end: the evidence the sweep named it on, and
// what the signal did about it.
type orphanReapOutcome struct {
	Row    couchcore.OrphanNvim `json:"row"`
	Status string               `json:"status"`
	Detail string               `json:"detail,omitempty"`
}

// orphanReapReport is the verb's whole answer, confirmed or not: which pass
// classified the rows, and one outcome each.
type orphanReapReport struct {
	Confirmed bool                `json:"confirmed"`
	SampledAt time.Time           `json:"sampled_at"`
	Outcomes  []orphanReapOutcome `json:"outcomes"`
	// LeftRunning is the same verb over the orphan registry (pair#695): the
	// processes teardown named and left, each ended by its exact identity.
	LeftRunning []leftRunningReapOutcome `json:"left_running"`
}

type leftRunningReapOutcome struct {
	Entry  couchcore.LeftRunningOrphan `json:"entry"`
	Status string                      `json:"status"`
	Detail string                      `json:"detail,omitempty"`
}

func runReapOrphansCLI(inv cliInvocation, rt Runtime, stdout, stderr io.Writer, call messageCall, reaper launcher.Reaper) int {
	socket, err := couchmessage.SocketPath(rt.StoreDir(), "broker")
	if err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), orphanReapBudget)
	defer cancel()
	if !inv.confirmed {
		return refuseOrphanReap(inv, ctx, socket, stdout, stderr, call)
	}
	var result couchmessage.Response
	if err := call(ctx, socket, couchmessage.Request{Op: orphanReapOp, Confirmed: true}, &result); err != nil {
		fmt.Fprintf(stderr, "couch: orphan reap unavailable (%v); inspect the running Couch\n", err)
		return 1
	}
	if result.Code != "ok" || result.Resources == nil {
		fmt.Fprintf(stderr, "couch: %s: %s\n", result.Code, rowtext.Sanitize(result.Error))
		return 1
	}
	// What the classification could not see belongs on stderr, ahead of the
	// outcome: a row omitted at the wire's detail limit is a row this verb did
	// not consider, and saying so after the fact reads as an excuse.
	for _, e := range result.Resources.Errors {
		fmt.Fprintln(stderr, "Coverage:", rowtext.Sanitize(e))
	}
	report := orphanReapReport{Confirmed: true, SampledAt: result.Resources.SampledAt, Outcomes: reapOrphanNvim(ctx, reaper, result.Resources.OrphanNvim), LeftRunning: reapLeftRunning(ctx, reaper, result.Resources.LeftRunning)}
	return finishOrphanReap(inv, stdout, stderr, report)
}

// refuseOrphanReap is the verb without --confirm: it names what a reap would end
// and signals nothing. An operator who has not decided yet must be able to look,
// and the rows they look at come from the monitor's report -- the confirmed verb
// is the one that asks for a classification of its own.
func refuseOrphanReap(inv cliInvocation, ctx context.Context, socket string, stdout, stderr io.Writer, call messageCall) int {
	var result couchmessage.Response
	if err := call(ctx, socket, couchmessage.Request{Op: "resources"}, &result); err != nil || result.Code != "ok" || result.Resources == nil {
		if err == nil {
			err = errors.New(rowtext.Sanitize(result.Error))
		}
		fmt.Fprintf(stderr, "couch: --reap-orphans needs --confirm, and the report that names its rows is unavailable (%v)\n", err)
		return 1
	}
	fmt.Fprintln(stderr, "couch: --reap-orphans ends processes and needs --confirm; nothing was signalled")
	outcomes := make([]orphanReapOutcome, 0, len(result.Resources.OrphanNvim))
	for _, row := range result.Resources.OrphanNvim {
		outcomes = append(outcomes, orphanReapOutcome{Row: row, Status: orphanRefused, Detail: "--confirm not given"})
	}
	left := make([]leftRunningReapOutcome, 0, len(result.Resources.LeftRunning))
	for _, entry := range result.Resources.LeftRunning {
		left = append(left, leftRunningReapOutcome{Entry: entry, Status: orphanRefused, Detail: "--confirm not given"})
	}
	return finishOrphanReap(inv, stdout, stderr, orphanReapReport{SampledAt: result.Resources.SampledAt, Outcomes: outcomes, LeftRunning: left})
}

// finishOrphanReap prints one report and decides the exit code: a row that
// survived SIGKILL fails the verb, and so does a refusal. A skip does not -- the
// leak the operator confirmed against is gone either way.
func finishOrphanReap(inv cliInvocation, stdout, stderr io.Writer, report orphanReapReport) int {
	if inv.jsonOutput {
		if err := json.NewEncoder(stdout).Encode(report); err != nil {
			fmt.Fprintln(stderr, "couch:", err)
			return 1
		}
	} else {
		renderOrphanReap(stdout, report)
	}
	for _, o := range report.Outcomes {
		if o.Status == orphanFailed || o.Status == orphanRefused {
			return 1
		}
	}
	for _, o := range report.LeftRunning {
		if o.Status == orphanFailed || o.Status == orphanRefused {
			return 1
		}
	}
	return 0
}

func renderOrphanReap(out io.Writer, report orphanReapReport) {
	if !report.Confirmed {
		rows := make([]couchcore.OrphanNvim, 0, len(report.Outcomes))
		for _, o := range report.Outcomes {
			rows = append(rows, o.Row)
		}
		renderOrphanNvimRows(out, rows)
		entries := make([]couchcore.LeftRunningOrphan, 0, len(report.LeftRunning))
		for _, o := range report.LeftRunning {
			entries = append(entries, o.Entry)
		}
		renderLeftRunningRows(out, entries)
		return
	}
	for _, o := range report.LeftRunning {
		detail := ""
		if o.Detail != "" {
			detail = ": " + rowtext.Sanitize(o.Detail)
		}
		fmt.Fprintf(out, "%s left-running %s%s\n", o.Status, leftRunningRow(o.Entry), detail)
	}
	if len(report.Outcomes) == 0 {
		fmt.Fprintln(out, "No unattributed embedded nvim: nothing to reap.")
		return
	}
	var reaped, skipped, failed int
	for _, o := range report.Outcomes {
		switch o.Status {
		case orphanReaped:
			reaped++
		case orphanSkipped:
			skipped++
		default:
			failed++
		}
		detail := ""
		if o.Detail != "" {
			detail = ": " + rowtext.Sanitize(o.Detail)
		}
		fmt.Fprintf(out, "%s pid=%d age=%s root=%s%s\n", o.Status, o.Row.Key.PID, o.Row.Age.Round(time.Minute), rowtext.Sanitize(o.Row.Root), detail)
	}
	fmt.Fprintf(out, "Reaped %d, skipped %d, failed %d\n", reaped, skipped, failed)
}

// reapOrphanNvim ends each named row's whole tree, on the cadence #399 measured.
// The identity fence is the safety story: the plan named a generation, and a pid
// that no longer holds it belongs to someone else now, so it is skipped rather
// than signalled. The tree matters too -- killing only the editor reparents its
// language servers to launchd and manufactures the next leak.
func reapOrphanNvim(ctx context.Context, reaper launcher.Reaper, rows []couchcore.OrphanNvim) []orphanReapOutcome {
	outcomes := make([]orphanReapOutcome, 0, len(rows))
	for _, row := range rows {
		status, detail := reapExact(ctx, reaper, row.Key.PID, row.Key.Identity, fmt.Sprintf("orphan-nvim-%d", row.Key.PID))
		outcomes = append(outcomes, orphanReapOutcome{Row: row, Status: status, Detail: detail})
	}
	return outcomes
}

// reapLeftRunning ends each registered left-running process's tree by its
// exact identity, on the same cadence and fence as the nvim sweep. A reaped
// or vanished entry drops from the registry at the next sample.
func reapLeftRunning(ctx context.Context, reaper launcher.Reaper, entries []couchcore.LeftRunningOrphan) []leftRunningReapOutcome {
	outcomes := make([]leftRunningReapOutcome, 0, len(entries))
	for _, entry := range entries {
		status, detail := reapExact(ctx, reaper, entry.Process.PID, entry.Process.Identity, fmt.Sprintf("left-running-%d", entry.Process.PID))
		outcomes = append(outcomes, leftRunningReapOutcome{Entry: entry, Status: status, Detail: detail})
	}
	return outcomes
}

// reapExact is one row's reap and its outcome: reaped, skipped when the pid no
// longer holds the named identity, or failed.
func reapExact(ctx context.Context, reaper launcher.Reaper, pid int, identity, label string) (string, string) {
	err := reaper.Reap(ctx, launcher.SessionServerIdentity{PID: pid, Identity: identity, Session: label})
	switch {
	case err == nil:
		return orphanReaped, ""
	case errors.Is(err, launcher.ErrServerChanged):
		return orphanSkipped, "pid no longer holds the identity the sweep named (gone, or recycled)"
	default:
		return orphanFailed, err.Error()
	}
}
