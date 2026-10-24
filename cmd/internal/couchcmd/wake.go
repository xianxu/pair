package couchcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// handleWake answers couch --wake (pair#830). After a full quit the operator
// resumes the main hub; its TL then wakes the other hubs the quit put to
// sleep. Each is one ordinary resume on the slot-operation queue, so recovery,
// admission and receipts are resume's own. Workers are never woken here: each
// woken hub's fleet resume (ariadne#564) wakes its own from the recover plan.
// A second wake finds a woken hub no longer asleep and one still resuming busy
// in the queue, so repeating it starts nothing twice.
func (s *messageService) handleWake(ctx context.Context, r couchmessage.Request) couchmessage.Response {
	ctx, cancel := context.WithTimeout(ctx, slotAdmissionBudget)
	defer cancel()
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	if s.slotOps == nil || s.authority.asleep == nil {
		return couchmessage.Response{Code: "unsupported", Error: "this Couch runs no slot operations"}
	}
	caller, err := s.switchCaller(ctx, r)
	if err != nil {
		return admissionRefusal("caller", err)
	}
	origin, err := s.connectedWorkspace(ctx, caller)
	if err != nil {
		return admissionRefusal("caller workspace", err)
	}
	if !s.authority.isHub(origin) || origin.WorktreeRoot != s.authority.hub {
		return couchmessage.Response{Code: "refused", Error: "only the main hub (" + s.authority.hub + ") may wake the asleep hubs"}
	}
	report := couchmessage.WakeReport{}
	if s.authority.hubs == nil {
		// One hub, and it is the caller: there is no other hub to wake.
		return couchmessage.Response{Code: "ok", Wake: &report}
	}
	rows, err := s.authority.asleep(ctx)
	if err != nil {
		return admissionRefusal("asleep set", err)
	}
	ownership := s.authority.hubs.snapshot()
	sort.Slice(rows, func(i, j int) bool { return rows[i].Label() < rows[j].Label() })
	for _, row := range rows {
		root := row.Target.Slot.WorktreeRoot
		if ownership.Wakes(origin.WorktreeRoot, root) != nil {
			continue
		}
		ref := row.Label()
		resume := r
		resume.Op, resume.Target = "resume", ref
		if resume.ID, err = newRequestID(); err != nil {
			report.Refused = append(report.Refused, couchmessage.WakeRefusal{Target: ref, Code: "unavailable", Error: err.Error()})
			continue
		}
		// The effect gate proves the caller is still the live main hub, the
		// target still a hub (as checkHubSlotOperation does for a resume), and
		// the target still asleep: a hub parked or resumed since admission is
		// not wake's to resume.
		guard := func(ctx context.Context) error {
			if _, err := s.liveCaller(ctx, r); err != nil {
				return fmt.Errorf("caller: %w", err)
			}
			current, err := s.authority.thread(ctx, caller)
			if err != nil {
				return fmt.Errorf("caller workspace: %w", err)
			}
			identity, err := s.authority.workspace(ctx, current)
			if err != nil {
				return fmt.Errorf("caller workspace: %w", err)
			}
			if err := s.authority.hubs.snapshot().Wakes(identity.WorktreeRoot, root); err != nil {
				return err
			}
			sleeping, err := s.authority.asleep(ctx)
			if err != nil {
				return fmt.Errorf("asleep set: %w", err)
			}
			for _, row := range sleeping {
				if row.Target.Slot.WorktreeRoot == root {
					return nil
				}
			}
			return fmt.Errorf("%s is no longer asleep", ref)
		}
		response := s.slotOps.handle(ctx, caller, resume, guard)
		if response.Code == "accepted" && response.Operation != nil {
			report.Admitted = append(report.Admitted, *response.Operation)
			continue
		}
		report.Refused = append(report.Refused, couchmessage.WakeRefusal{Target: ref, Code: response.Code, Error: response.Error})
	}
	return couchmessage.Response{Code: "ok", Wake: &report}
}

// wakeOutcome is one hub's line in couch --wake's summary.
type wakeOutcome struct {
	Target string `json:"target"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// runWakeCLI admits the wake, polls every admitted resume to its end, and
// prints one summary. Only a wake where every asleep hub came back exits 0.
func runWakeCLI(inv cliInvocation, rt Runtime, stdout, stderr io.Writer, call messageCall, clock slotPollClock) int {
	namespace := rt.Getenv("COUCH_STORE_DIR")
	if namespace == "" {
		fmt.Fprintln(stderr, "couch: --wake requires a live Couch slot: resume the main hub from the switcher and run it there")
		return 1
	}
	socket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	request := callerIdentity(rt, "wake")
	if request.ID, err = newRequestID(); err != nil {
		fmt.Fprintln(stderr, "couch: create request ID:", err)
		return 1
	}
	exchange := func(r couchmessage.Request, timeout time.Duration) (couchmessage.Response, error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		var result couchmessage.Response
		return result, call(ctx, socket, r, &result)
	}
	result, err := exchange(request, couchmessage.AdmissionTimeout)
	switch {
	case err != nil:
		fmt.Fprintf(stderr, "couch: wake (request %s): %v\noutcome uncertain; run couch --wake again (it wakes only what is still asleep)\n", request.ID, err)
		return 1
	case result.Code == "invalid-request" && strings.Contains(result.Error, "unknown message operation"):
		fmt.Fprintln(stderr, "couch: the running Couch predates `couch --wake`; restart Couch (switcher Alt+d, then `couch`) on the new binary to use it")
		return 1
	case result.Code != "ok" || result.Wake == nil:
		fmt.Fprintf(stderr, "couch: %s: %s\n", result.Code, result.Error)
		return 1
	}
	var outcomes []wakeOutcome
	for _, admitted := range result.Wake.Admitted {
		status := callerIdentity(rt, "operation-status")
		status.ID = admitted.ID
		receipt, err := pollSlotOperation(admitted, admitted.Target+" wake", status, exchange, clock, stderr)
		switch {
		case err != nil:
			outcomes = append(outcomes, wakeOutcome{Target: admitted.Target, Status: "uncertain", Detail: err.Error()})
		case receipt.Status == couchmessage.ReceiptSucceeded:
			outcomes = append(outcomes, wakeOutcome{Target: receipt.Target, Status: "woken"})
		default:
			outcomes = append(outcomes, wakeOutcome{Target: receipt.Target, Status: string(receipt.Status), Detail: receipt.Detail})
		}
	}
	for _, refused := range result.Wake.Refused {
		outcomes = append(outcomes, wakeOutcome{Target: refused.Target, Status: "not admitted", Detail: refused.Code + ": " + refused.Error})
	}
	code := 0
	for _, o := range outcomes {
		if o.Status != "woken" {
			code = 1
		}
	}
	if inv.jsonOutput {
		if outcomes == nil {
			outcomes = []wakeOutcome{}
		}
		if err := json.NewEncoder(stdout).Encode(outcomes); err != nil {
			fmt.Fprintln(stderr, "couch:", err)
			return 1
		}
		return code
	}
	fmt.Fprintln(stdout, wakeSummary(outcomes))
	return code
}

// wakeSummary is the one line couch --wake prints, then each hub that did not
// come back with why.
func wakeSummary(outcomes []wakeOutcome) string {
	if len(outcomes) == 0 {
		return "no asleep hubs to wake"
	}
	var woken, rest []string
	for _, o := range outcomes {
		if o.Status == "woken" {
			woken = append(woken, o.Target)
			continue
		}
		rest = append(rest, fmt.Sprintf("  %s %s: %s", o.Target, o.Status, o.Detail))
	}
	line := fmt.Sprintf("woke %d of %d asleep hubs", len(woken), len(outcomes))
	if len(woken) > 0 {
		line += ": " + strings.Join(woken, ", ")
	}
	if len(rest) > 0 {
		line += "\n" + strings.Join(rest, "\n") + "\nrun couch --wake again once the cause is fixed; it wakes only what is still asleep"
	}
	return line
}
