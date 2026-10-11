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
)

// prepareRefreshTimeout bounds weave refresh, as the provisioner bounds
// weave compile (WorkspaceProvisioner.SetupTimeout).
const prepareRefreshTimeout = 20 * time.Minute

// runPrepareCLI is `couch --prepare repo:N` (pair#434): the running Couch
// admits the slot, this process runs weave refresh in it (never on the
// console's serial queue, which a refresh would hold for minutes), and the
// running Couch reloads the agent's context. Weave's output streams to
// stderr; the report goes to stdout. Only a fully successful run exits 0.
func runPrepareCLI(inv cliInvocation, rt Runtime, stdout, stderr io.Writer, call messageCall, clock slotPollClock, commands couchcore.ProvisionIO) int {
	remote := func(op string) (couchmessage.OperationReceipt, error) {
		receipt, err := exchangeSlotOperation(inv, op, rt, call, clock)
		if err != nil {
			return receipt, err
		}
		return receipt, receiptError(receipt)
	}
	// Setup errors (no live Couch, an older Couch) stop before any step.
	admitted, admitErr := remote(couchcore.OpPrepare)
	var refusal *couchcore.SlotOperationError
	var uncertain *slotOperationUncertain
	if admitErr != nil && !errors.As(admitErr, &refusal) && !errors.As(admitErr, &uncertain) && !errors.Is(admitErr, errReceiptFailed) {
		fmt.Fprintln(stderr, "couch:", admitErr)
		return 1
	}
	if uncertain != nil {
		// Unlike the restarts, the admission has no effect, so the remedy is
		// simply to prepare again, not to verify before resending.
		admitErr = fmt.Errorf("prepare %s: admission outcome unknown (%s); it changes nothing, so run couch --prepare again", inv.ref, uncertain.detail)
	}
	report := couchcore.RunPrepare(context.Background(), couchcore.PrepareRun{
		Target: inv.ref, IO: commands, Progress: stderr, Timeout: prepareRefreshTimeout,
		Admit: func(context.Context) (couchcore.PrepareAdmission, error) {
			if admitErr != nil {
				return couchcore.PrepareAdmission{}, admitErr
			}
			if admitted.Path == "" {
				return couchcore.PrepareAdmission{}, errors.New("the running Couch admitted the slot but named no checkout; restart Couch on the current binary")
			}
			return couchcore.PrepareAdmission{Path: admitted.Path}, nil
		},
		Reload: func(context.Context) error {
			_, err := remote(couchcore.OpReloadContext)
			return err
		},
	})
	if inv.jsonOutput {
		if err := json.NewEncoder(stdout).Encode(report); err != nil {
			fmt.Fprintln(stderr, "couch:", err)
			return 1
		}
	} else {
		renderPrepare(stdout, report)
	}
	if !report.OK() {
		return 1
	}
	return 0
}

// errReceiptFailed marks a receipt that finished failed (not refused).
var errReceiptFailed = errors.New("failed")

// receiptError is a terminal receipt as an error: nil on success, the typed
// refusal when refused, errReceiptFailed with the detail otherwise.
func receiptError(r couchmessage.OperationReceipt) error {
	switch r.Status {
	case couchmessage.ReceiptSucceeded:
		return nil
	case couchmessage.ReceiptRefused:
		return &couchcore.SlotOperationError{Code: r.Code, Detail: r.Detail}
	}
	detail := r.Detail
	if r.Code != "" {
		detail = r.Code + ": " + detail
	}
	return fmt.Errorf("%w: %s", errReceiptFailed, detail)
}

func renderPrepare(w io.Writer, r couchcore.PrepareReport) {
	for _, s := range r.Steps {
		line := fmt.Sprintf("%-8s %-8s", s.Name, s.Status)
		if s.Code != "" {
			line += " (" + s.Code + ")"
		}
		if s.Detail != "" {
			line += " " + s.Detail
		}
		fmt.Fprintln(w, line)
	}
	if r.Head != "" {
		fmt.Fprintf(w, "%s %s at %s (origin/main %s)\n", r.Target, r.Path, shortCommit(r.Head), shortCommit(r.OriginMain))
	}
	if len(r.Substrates) > 0 {
		fmt.Fprintln(w, "substrates (the slot's sdlc and weave build from these):")
		for _, s := range r.Substrates {
			fmt.Fprintf(w, "  %s %s %s\n", s.Path, shortCommit(s.Commit), s.Subject)
		}
	}
}

func shortCommit(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	if sha == "" {
		return "unknown"
	}
	return sha
}
