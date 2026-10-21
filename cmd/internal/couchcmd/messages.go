package couchcmd

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

type messageCall func(context.Context, string, any, any) error

func runMessageCLI(inv cliInvocation, rt Runtime, stdout, stderr io.Writer) int {
	return runMessageCLIWithCall(inv, rt, stdout, stderr, couchmessage.Call)
}

func runMessageCLIWithCall(inv cliInvocation, rt Runtime, stdout, stderr io.Writer, call messageCall) int {
	if inv.messageOp == "automation" {
		return runAutomationCLI(inv, rt, stdout, stderr, call)
	}
	if couchmessage.IsDriveOp(inv.messageOp) {
		return runAgentDriveCLI(inv, rt, stdout, stderr, call)
	}
	if inv.messageOp == "resources" {
		return runResourcesCLI(inv, rt, stdout, stderr, call)
	}
	if inv.messageOp == orphanReapOp {
		// The reap runs here, in the caller's process, and not in the console:
		// its escalation waits seconds per process, which is the same reason
		// peek is a CLI verb. The supervisor owns the classification, because
		// only it holds the whole slot inventory a row is unattributed against.
		return runReapOrphansCLI(inv, rt, stdout, stderr, call, launcher.TreeReaper(launcher.OSProcessTable{}))
	}
	if inv.messageOp == "notify" {
		return runNotifyCLI(inv, rt, stdout, stderr, call)
	}
	// The one verb list (pair#424): a hand-kept list here let relaunch and
	// reload-context fall through to the message path, which sends no ID.
	if inv.messageOp == "resume-candidates" {
		return runResumeCandidatesCLI(inv, rt, stdout, stderr, call)
	}
	if inv.messageOp == "switch-agent" || inv.messageOp == "operation-status" {
		return runSwitchAgentCLI(inv, rt, stdout, stderr, call)
	}
	if inv.messageOp == couchcore.OpPrepare {
		return runPrepareCLI(inv, rt, stdout, stderr, call, slotPollClock{now: time.Now, sleep: time.Sleep}, couchcore.OSProvisionIO{})
	}
	if inv.messageOp == "recovery-confirmation" || couchcore.IsSlotOperation(inv.messageOp) || inv.messageOp == "move-slot" {
		return runSlotOperationCLI(inv, rt, stdout, stderr, call, slotPollClock{now: time.Now, sleep: time.Sleep})
	}
	if inv.messageOp == "broadcast-status" {
		return runBroadcastListCLI(inv, rt, stdout, stderr, call)
	}
	namespace := rt.Getenv("COUCH_STORE_DIR")
	if namespace == "" {
		fmt.Fprintln(stderr, "couch: messaging requires a live Couch slot")
		return 1
	}
	socket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	r := callerIdentity(rt, inv.messageOp)
	switch inv.messageOp {
	case "send":
		if r.ID, err = newRequestID(); err != nil {
			fmt.Fprintln(stderr, "couch: create message ID:", err)
			return 1
		}
		r.Target = inv.ref
		r.Agent = inv.messageAgent
		r.Body = inv.messageBody
		if inv.messageFile != "" {
			if r.Body, err = readMessageFile(inv.messageFile); err != nil {
				fmt.Fprintln(stderr, "couch:", err)
				return 1
			}
		}
	case "status":
		r.ID = inv.ref
	}
	if err := couchmessage.ValidateRequest(r); err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	// The broker spends up to AdmissionTimeout checking and admitting a send, then
	// answers with its verdict (pair#618); waiting exactly as long gave up at
	// the moment the answer was due and hid the reason.
	ctx, cancel := context.WithTimeout(context.Background(), couchmessage.SendTransportTimeout)
	defer cancel()
	var result couchmessage.Response
	if err := call(ctx, socket, r, &result); err != nil {
		if r.Op == "send" {
			printUncertainMessage(stderr, r.ID, err.Error())
		} else {
			fmt.Fprintln(stderr, "couch:", err)
		}
		return 1
	}
	if result.Code == "uncertain" && r.Op == "send" {
		printUncertainMessage(stderr, r.ID, result.Error)
		return 1
	}
	// Older supervisors use the recipient error for missing callers too.
	// Actors/status can only have a missing caller; send may lack either end.
	// A generic unavailable reply does not prove registration loss or justify a restart.
	if result.Code == "unavailable" && result.Error == couchmessage.ErrUnavailable.Error() {
		if r.Op == "send" {
			result.Error = "Couch could not verify caller or recipient availability; inspect couch --actors and report a persistent refusal"
		} else {
			result.Error = "Couch could not verify this caller; inspect couch --actors and report a persistent refusal"
		}
	}
	success := result.Code == "ok" || result.Code == "accepted" || result.Code == "not-dispatched"
	if inv.jsonOutput {
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			fmt.Fprintln(stderr, "couch:", err)
			return 1
		}
	} else if !success {
		fmt.Fprintf(stderr, "couch: %s: %s\n", result.Code, result.Error)
	} else if result.Code == "not-dispatched" {
		fmt.Fprintf(stdout, "not-dispatched: %s\n", result.Error)
	} else if r.Op == "actors" {
		if len(result.Actors) == 0 {
			fmt.Fprintln(stdout, "No live message-capable slots.")
		}
		now := time.Now()
		for _, a := range result.Actors {
			slot := a.Binding.Slot
			if a.Alias != "" {
				slot += " (" + a.Alias + ")"
			}
			fmt.Fprintf(stdout, "%s\t%s\t%s\tallowance=%d/%d%s\n", slot, a.Binding.Agent, messageActorAvailability(a, now), a.Remaining, a.Capacity, niceColumn(a.Nice))
		}
	} else if result.Receipt != nil {
		receipt := result.Receipt
		fmt.Fprintf(stdout, "%s %s -> %s %s\n", receipt.Message.ID, receipt.Message.From.Slot, receipt.Message.To.Slot, receipt.Status)
		if receipt.Detail != "" {
			fmt.Fprintln(stdout, receipt.Detail)
		}
		if r.Op == "status" {
			fmt.Fprintln(stdout, receipt.Message.Body)
		}
	} else {
		fmt.Fprintln(stderr, "couch: broker returned no message receipt")
		return 1
	}
	if !success {
		return 1
	}
	return 0
}

// readMessageFile is --message-file's body (pair#486): the file verbatim,
// less one trailing newline (LF or CRLF), so an agent writes a message with its editor and
// sends it without quoting it in a shell. The read is bounded just past the
// broker's body limit, which validation then refuses with the rest.
func readMessageFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("--message-file: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, couchmessage.MaxBodyBytes+2))
	if err != nil {
		return "", fmt.Errorf("--message-file: %w", err)
	}
	body := string(data)
	if b, ok := strings.CutSuffix(body, "\r\n"); ok {
		body = b
	} else {
		body = strings.TrimSuffix(body, "\n")
	}
	if strings.TrimSpace(body) == "" {
		return "", fmt.Errorf("--message-file %s is empty", path)
	}
	if len(body) > couchmessage.MaxBodyBytes {
		return "", fmt.Errorf("--message-file %s exceeds the %d-byte message limit", path, couchmessage.MaxBodyBytes)
	}
	return body, nil
}

// callerIdentity is a request from this slot's conversation: the identity
// the broker resolves to a live, registered binding.
func callerIdentity(rt Runtime, op string) couchmessage.Request {
	return couchmessage.Request{Op: op, Scope: rt.Getenv("COUCH_THREAD_SCOPE"), Tag: rt.Getenv("COUCH_THREAD_TAG"), Session: rt.Getenv("PAIR_SESSION_NAME"), Nonce: rt.Getenv("PAIR_LAUNCH_NONCE")}
}

// newRequestID is a random (version 4) UUID naming one request.
func newRequestID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:]), nil
}

// Slot operations admit, then poll: a resume can outlast the socket's 2 s
// exchange, so the CLI polls the receipt within this budget (pair#367 M2).
const (
	// Admission may retry only an explicit warming refusal (nothing queued).
	// Bound both elapsed time, including exchanges, and the number of attempts.
	slotAdmissionRetryBudget   = 30 * time.Second
	slotAdmissionMaxAttempts   = 8
	slotAdmissionMaxRetryAfter = 5 * time.Second
	slotOperationPollInterval  = 500 * time.Millisecond
	// Covers the operation plus a restart's readiness wait (pair#427).
	slotOperationPollBudget = 5 * time.Minute
	// slotOperationProgressEvery paces the "still running" line, so a start
	// slowed by load reads as progress, not a hang (pair#489).
	slotOperationProgressEvery = 15 * time.Second
)

// slotPollClock is the poll loop's time: the budget is measured on now and
// waits use sleep (tests advance both). Each exchange still runs on a real
// per-call context.
type slotPollClock struct {
	now   func() time.Time
	sleep func(time.Duration)
}

// runSlotOperationCLI admits resume/reboot <slot> through the running Couch
// and polls its receipt (exchangeSlotOperation), then reports it.
func runSlotOperationCLI(inv cliInvocation, rt Runtime, stdout, stderr io.Writer, call messageCall, clock slotPollClock) int {
	receipt, err := exchangeSlotOperation(inv, inv.messageOp, rt, call, clock, stderr)
	var uncertain *slotOperationUncertain
	switch {
	case errors.As(err, &uncertain):
		printUncertainSlotOperation(stderr, uncertain.request, uncertain.detail)
		return 1
	case err != nil:
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	return reportSlotOperation(stdout, stderr, receipt, inv.jsonOutput)
}

// slotOperationUncertain: the request was (or may have been) admitted, and
// its outcome was lost. The report, not a resend, is the next step.
type slotOperationUncertain struct {
	request couchmessage.Request
	detail  string
}

func (e *slotOperationUncertain) Error() string {
	return fmt.Sprintf("%s %s (request %s): %s; outcome uncertain, verify with couch --recover-plan-from-sdlc before resending", e.request.Op, e.request.Target, e.request.ID, e.detail)
}

// exchangeSlotOperation admits op on inv's slot through the running Couch
// and polls its receipt to a terminal status. Explicit warming refusals may
// retry the same request within the admission retry budget. Each call is
// bounded by AdmissionTimeout and the remaining admission budget. Once
// admitted, receipt polling has its own unchanged poll budget. After
// admission any lost or unreadable outcome (dial failure, timeout, an
// unavailable caller after a Couch restart, a receipt no longer held) is a
// *slotOperationUncertain: the report, not the receipt, is the evidence.
func exchangeSlotOperation(inv cliInvocation, op string, rt Runtime, call messageCall, clock slotPollClock, progress io.Writer) (couchmessage.OperationReceipt, error) {
	var none couchmessage.OperationReceipt
	namespace := rt.Getenv("COUCH_STORE_DIR")
	if namespace == "" {
		return none, fmt.Errorf("--%s requires a live Couch slot", inv.messageOp)
	}
	socket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		return none, err
	}
	admit := callerIdentity(rt, op)
	admit.Target, admit.Confirmed = inv.ref, inv.confirmed
	admit.CandidateKey = inv.candidateKey
	admit.RecoveryConfirmation = inv.recoveryConfirmation
	admit.ToHub, admit.Persist = inv.toHub, inv.persistHub
	admit.SameBinary, admit.ForceUnknown = inv.sameBinary, inv.forceUnknown
	if admit.ID, err = newRequestID(); err != nil {
		return none, fmt.Errorf("create request ID: %w", err)
	}
	if err := couchmessage.ValidateRequest(admit); err != nil {
		return none, err
	}
	exchange := func(r couchmessage.Request, timeout time.Duration) (couchmessage.Response, error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		var result couchmessage.Response
		return result, call(ctx, socket, r, &result)
	}
	uncertain := func(detail string) (couchmessage.OperationReceipt, error) {
		return none, &slotOperationUncertain{request: admit, detail: detail}
	}
	admissionStart := clock.now()
	var result couchmessage.Response
	for attempt := 1; ; attempt++ {
		remaining := slotAdmissionRetryBudget - clock.now().Sub(admissionStart)
		if remaining <= 0 {
			return none, fmt.Errorf("slot catalog still warming after %s; nothing was queued; retry", slotAdmissionRetryBudget)
		}
		result, err = exchange(admit, min(couchmessage.AdmissionTimeout, remaining))
		if err != nil || result.Code != couchmessage.CodeWarming {
			break
		}
		// The typed refusal is the guarantee that retrying cannot duplicate an
		// effect. An attached receipt or malformed hint does not authorize it.
		if result.Operation != nil || result.RetryAfterMillis <= 0 || result.RetryAfterMillis > int(slotAdmissionMaxRetryAfter/time.Millisecond) {
			break
		}
		remaining = slotAdmissionRetryBudget - clock.now().Sub(admissionStart)
		if attempt >= slotAdmissionMaxAttempts || remaining <= 0 {
			return none, fmt.Errorf("slot catalog still warming after %d admission attempts (%s); nothing was queued; retry", attempt, clock.now().Sub(admissionStart).Round(time.Second))
		}
		fmt.Fprintf(progress, "couch: %s %s waiting for slot catalog to warm (attempt %d/%d)\n", inv.ref, op, attempt, slotAdmissionMaxAttempts)
		clock.sleep(min(time.Duration(result.RetryAfterMillis)*time.Millisecond, remaining))
	}
	switch {
	case err != nil:
		return uncertain(err.Error())
	case result.Code == "uncertain":
		return uncertain(result.Error)
	case result.Code == "invalid-request" && (strings.Contains(result.Error, "unknown message operation") || strings.Contains(result.Error, "unknown field")):
		// The running Couch is older than this CLI: it has no such operation.
		return none, fmt.Errorf("the running Couch predates `couch --%s`; restart Couch (switcher Alt+d, then `couch`) on the new binary to use it", inv.messageOp)
	case result.Code != "accepted" || result.Operation == nil:
		return none, fmt.Errorf("%s: %s", result.Code, result.Error)
	}
	receipt := *result.Operation
	status := callerIdentity(rt, "operation-status")
	status.ID = admit.ID
	start := clock.now()
	// Every status line derives staleness from when Couch last answered, so
	// a status repeated from before timed-out polls says so, and stops saying
	// so once an answer arrives.
	lastAnswer := start
	stale := func() string {
		if age := clock.now().Sub(lastAnswer); age > 2*slotOperationPollInterval {
			return fmt.Sprintf(" (last answer %s ago; status polls are timing out)", age.Round(time.Second))
		}
		return ""
	}
	for told := start; !receipt.Status.Terminal(); {
		if clock.now().Sub(start) >= slotOperationPollBudget {
			return uncertain(fmt.Sprintf("still %s after %s%s", receipt.Status, slotOperationPollBudget, stale()))
		}
		if clock.now().Sub(told) >= slotOperationProgressEvery {
			told = clock.now()
			fmt.Fprintf(progress, "couch: %s %s still %s after %s%s\n", inv.ref, op, receipt.Status, told.Sub(start).Round(time.Second), stale())
		}
		clock.sleep(slotOperationPollInterval)
		result, err := exchange(status, couchmessage.AdmissionTimeout)
		switch {
		case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded):
			// A status read is idempotent, and a Couch busy launching under
			// load can answer one late (pair#489: runs ended uncertain at 12 s
			// and 24 s). Ask again; only the poll budget gives up.
			continue
		case err != nil:
			return uncertain(err.Error())
		case result.Code != "ok" || result.Operation == nil:
			return uncertain(result.Code + ": " + result.Error)
		case result.Operation.Status == couchmessage.ReceiptUnknown:
			return uncertain("Couch no longer holds this request (it may have restarted)")
		}
		receipt, lastAnswer = *result.Operation, clock.now()
	}
	return receipt, nil
}

// reportSlotOperation prints a terminal receipt: JSON, or one line naming the
// outcome. Only success exits 0.
func reportSlotOperation(stdout, stderr io.Writer, receipt couchmessage.OperationReceipt, asJSON bool) int {
	code := 1
	if receipt.Status == couchmessage.ReceiptSucceeded {
		code = 0
	}
	if asJSON {
		if err := json.NewEncoder(stdout).Encode(receipt); err != nil {
			fmt.Fprintln(stderr, "couch:", err)
			return 1
		}
		return code
	}
	switch receipt.Status {
	case couchmessage.ReceiptSucceeded:
		line := receipt.Target + " " + receipt.Op + " succeeded"
		if receipt.Op == "move-slot" {
			mode := "until restart"
			if receipt.Persist {
				mode = "persisted"
			}
			line += " -> " + receipt.ToHub + " (" + mode + ")"
		}
		if receipt.Tag != "" {
			line += " (tag " + receipt.Tag + ")"
		}
		if receipt.Archived != "" {
			line += "; archived " + receipt.Archived
		}
		fmt.Fprintln(stdout, line)
		if receipt.Warning != "" {
			fmt.Fprintln(stdout, receipt.Warning)
		}
	case couchmessage.ReceiptRefused:
		fmt.Fprintf(stderr, "couch: %s %s refused (%s): %s\n", receipt.Target, receipt.Op, receipt.Code, receipt.Detail)
	default:
		code := ""
		if receipt.Diagnostic != "" {
			code = " (" + receipt.Diagnostic + ")"
		}
		fmt.Fprintf(stderr, "couch: %s %s failed%s: %s\n", receipt.Target, receipt.Op, code, receipt.Detail)
	}
	return code
}

// printUncertainSlotOperation is the one text for a slot operation whose
// outcome is unknown: verify with the report before any resend.
func printUncertainSlotOperation(w io.Writer, r couchmessage.Request, detail string) {
	if r.Op == "move-slot" {
		fmt.Fprintf(w, "couch: move-slot %s -> %s (request %s): %s\noutcome uncertain; query couch --operation-status %s before another move; after a Couch restart inspect the started hubs’ .hub claims\n", r.Target, r.ToHub, r.ID, detail, r.ID)
		return
	}

	fmt.Fprintf(w, "couch: %s %s (request %s): %s\noutcome uncertain; verify with couch --recover-plan-from-sdlc before resending (a resend is refused harmlessly once the slot is live)\n", r.Op, r.Target, r.ID, detail)
}

func printUncertainMessage(w io.Writer, id, detail string) {
	fmt.Fprintf(w, "couch: message %s outcome uncertain: %s\nQuery couch --message-status %s before any further send; do not automatically retry.\n", id, detail, id)
}

func messageActorAvailability(a couchmessage.Candidate, now time.Time) string {
	switch {
	case !a.Supported:
		return "unsupported"
	case a.Pending:
		return "pending"
	case a.Remaining <= 0:
		return "breaker-exhausted"
	case !a.Known:
		return "unknown"
	case !a.Resting:
		return "occupied"
	case a.LastActivity.IsZero() || now.Before(a.LastActivity.Add(couchmessage.QuietInterval)):
		return "active"
	default:
		return "available"
	}
}

// niceColumn is a listing's renicer facts (#502): the agent's niceness and
// how many of its descendants are low; nothing before the renicer looked.
func niceColumn(f *couchmessage.NiceFacts) string {
	switch {
	case f == nil:
		return ""
	case !f.Confirmed:
		return "\tnice=?"
	}
	s := fmt.Sprintf("\tnice=%d lowered=%d", f.AgentNice, f.Lowered+f.AtLevel)
	if f.Denied > 0 {
		s += fmt.Sprintf(" denied=%d", f.Denied)
	}
	return s
}
