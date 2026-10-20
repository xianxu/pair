package couchmessage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
	core "github.com/xianxu/pair/cmd/internal/submission"
)

// Request has closed per-operation shapes. Wrapper lifecycle operations use a
// full binding and must pass the supervisor adapter's live identity check.
type Request struct {
	Drain                                            *DrainRequest       `json:",omitempty"`
	Drive                                            *agentdrive.Request `json:",omitempty"`
	ToHub                                            string              `json:",omitempty"`
	Persist                                          bool                `json:",omitempty"`
	Op, Scope, Tag, Session, Nonce, ID, Target, Body string
	Title                                            string `json:",omitempty"` // notify only
	// Agent narrows a send or selects the switch destination harness.
	Agent                string `json:",omitempty"`
	RecoveryConfirmation string `json:",omitempty"`
	CandidateKey         string `json:",omitempty"`
	// Confirmed declares the operator's confirmation for an operation whose
	// declaration requires one (reboot); resume and reboot only.
	Confirmed bool `json:",omitempty"`
	// SameBinary and ForceUnknown are relaunch/reload-context overrides
	// (pair#421); each is recorded in the receipt when it changed the outcome.
	SameBinary   bool `json:",omitempty"`
	ForceUnknown bool `json:",omitempty"`
	// TailScope and TailTag name the thread whose in-memory terminal tail
	// reads, Lines how much of it (pair#425); tail only. A tail may instead
	// name its slot in Target (pair#429), resolved as a send's exact slot is.
	TailScope string `json:",omitempty"`
	TailTag   string `json:",omitempty"`
	Lines     int    `json:",omitempty"`
	Binding   *Binding
}

// CodeWarming is an admission refusal guaranteeing that nothing was queued.
// Only this response may authorize an automatic retry of the same intent.
const CodeWarming = "warming"

type Response struct {
	DrainDeliveries []DrainDelivery         `json:",omitempty"`
	Drain           *couchcore.DrainBoard   `json:",omitempty"`
	Resources       *couchcore.ResourceView `json:",omitempty"`
	// RetryAfterMillis is a transient admission hint, only for CodeWarming.
	RetryAfterMillis int `json:",omitempty"`
	// Refusals answers refusals: each canonical slot's newest unsuperseded
	// reason and recovery class. No caller credentials: this is unauthenticated.
	Refusals        map[string]RefusalInfo `json:",omitempty"`
	Code, Error     string
	RecoveryPreview *couchcore.RecoveryPreview `json:",omitempty"`
	Submission      *SubmissionStatus          `json:",omitempty"`
	Receipt         *Receipt
	Actors          []Candidate
	// Operation is a resume/reboot request's receipt (pair#367 M2).
	Operation *OperationReceipt `json:",omitempty"`
	// Broadcast answers broadcast-status (pair#413); nil means no broadcast.
	Broadcast *BroadcastStatus `json:",omitempty"`
	// Shares are every link out, by kind (pair#467): the screen broadcast,
	// the web cast's view-only and pointer links (one share, pair#506) and
	// its drive link.
	Shares []ShareStatus `json:",omitempty"`
	// Tail answers tail (pair#425); TailThread names whose it is (pair#429).
	Tail       *Tail       `json:",omitempty"`
	TailThread *TailThread `json:",omitempty"`
	// Replay answers replay-share and replay-status (pair#477).
	Replay *ReplayShare `json:",omitempty"`
	// Replays answers broadcast-status too: the recordings shared for
	// replay through the tunnel (pair#477), by id, never their links.
	Replays []string `json:",omitempty"`
}

// SubmissionStatus is a durable evidence submission's current outcome. Polling
// it never repeats capture or delivery; the client retains ID through retries.
type SubmissionStatus struct {
	Progress  *core.SubmissionProgress `json:"progress,omitempty"`
	Slot      string                   `json:"slot,omitempty"`
	HowToRead string                   `json:"how_to_read,omitempty"`
	ID        string                   `json:"id"`
	State     string                   `json:"state"`
	Path      string                   `json:"path,omitempty"`
	Error     string                   `json:"error,omitempty"`
	Target    core.Target              `json:"target"`
}

// IsDriveOp names the ops that carry agentdrive intent into a hub-owned slot:
// keys, set-model and interrupt (pair#717).
func IsDriveOp(op string) bool { return op == "keys" || op == "set-model" || op == "interrupt" }

func IsSubmissionOp(op string) bool {
	switch op {
	case "submission-prepare", "submission-status", "submission-begin", "submission-ack":
		return true
	}
	return false
}

// ReplayShare is a recording's replay share in the running Couch (pair#477):
// State is opening, ready, failed, lost or closed; Base, once ready, is the
// public <base>/<token> its replay link is built on, and Local says the
// tunnel is off (the link works on that machine only). Unlike
// BroadcastStatus it carries the token: the link is what was asked for.
type ReplayShare struct {
	State string `json:"state"`
	Base  string `json:"base,omitempty"`
	Local bool   `json:"local,omitempty"`
	Error string `json:"error,omitempty"`
}

// IsReplayOp says whether op is a replay share's request (pair#477).
func IsReplayOp(op string) bool { return op == "replay-share" || op == "replay-status" }

// BroadcastStatus is the running console's broadcast, as `couch
// --broadcast-list` prints it (pair#413). It has no field for the link or its
// token -- the token is the broadcast's only credential, so a listing can
// never print it (operator decision).
type BroadcastStatus struct {
	State     string    `json:"state"`
	StartedAt time.Time `json:"started_at,omitzero"`
	Mode      string    `json:"mode,omitempty"`
	Viewers   int       `json:"viewers"`
}

// ShareStatus is one link out, as `couch --broadcast-list` lists it
// (pair#467): which kind it is, its state and who is on it. Like
// BroadcastStatus it has no field for the link or its token.
type ShareStatus struct {
	// Kind is ShareScreen, ShareWebView or ShareWebDrive.
	Kind    string `json:"kind"`
	State   string `json:"state"`
	Viewers int    `json:"viewers"`
	// Driver and Asking: the drive link's control (a viewer's own name).
	Driver string `json:"driver,omitempty"`
	Asking int    `json:"asking,omitempty"`
}

// Share kinds.
const (
	ShareScreen   = "screen"
	ShareWebView  = "web view"
	ShareWebDrive = "web drive"
)

func validMessageID(id string) bool {
	return id != "" && len(id) <= 128 && utf8.ValidString(id) && strings.IndexFunc(id, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) < 0
}

func ValidateRequest(r Request) error {
	if r.Op == "drain" {
		return ValidateDrainRequest(r)
	}
	if r.Drain != nil {
		return errors.New("drain payload applies only to drain")
	}
	if r.Drive != nil && !IsDriveOp(r.Op) {
		return errors.New("drive intent applies only to keys, set-model and interrupt")
	}
	if r.RecoveryConfirmation != "" && (r.Op != "recovery-confirmation" || !validMessageID(r.RecoveryConfirmation) || len(r.RecoveryConfirmation) > 128) {
		return errors.New("recovery confirmation requires a bounded opaque token")
	}
	if r.CandidateKey != "" && (r.Op != "resume" || !utf8.ValidString(r.CandidateKey) || len(r.CandidateKey) > 512 || strings.IndexFunc(r.CandidateKey, unicode.IsSpace) >= 0 || strings.IndexFunc(r.CandidateKey, unicode.IsControl) >= 0) {
		return errors.New("candidate key applies only to resume and must be a bounded opaque token")
	}
	if r.Op != "move-slot" && (r.ToHub != "" || r.Persist) {
		return errors.New("hub destination and persistence apply only to move-slot requests")
	}
	if r.Op == "move-slot" {
		if r.Binding != nil || r.Scope == "" || r.Tag == "" || r.Session == "" || r.Nonce == "" || !validMessageID(r.ID) || r.ToHub == "" || len(r.ToHub) > MaxBindingBytes || r.Body != "" || r.Title != "" || r.Agent != "" || r.Confirmed || r.SameBinary || r.ForceUnknown || r.TailScope != "" || r.TailTag != "" || r.Lines != 0 {
			return errors.New("move-slot requires caller identity, request ID, exact slots and a hub")
		}
		for _, f := range []string{r.Scope, r.Tag, r.Session, r.Nonce} {
			if len(f) > MaxBindingBytes {
				return errors.New("request identity exceeds limit")
			}
		}
		slots := strings.Split(r.Target, ",")
		if len(slots) > couchcore.MaxHubSlots || len(r.Target) > couchcore.MaxHubSlots*MaxBindingBytes {
			return errors.New("too many move slots")
		}
		for _, slot := range slots {
			if len(slot) > MaxBindingBytes {
				return ErrInvalidTarget
			}
			if _, _, err := parseSlot(slot); err != nil {
				return err
			}
		}
		return nil
	}

	if r.Op != "notify" && r.Title != "" {
		return errors.New("a notification title applies only to notify")
	}
	// Caller identity and target fields cannot exceed a complete binding's budget.
	for _, field := range []string{r.Op, r.Scope, r.Tag, r.Session, r.Nonce, r.Target, r.Agent, r.TailScope, r.TailTag} {
		if len(field) > MaxBindingBytes {
			return errors.New("request identity exceeds limit")
		}
	}
	if r.Op == "automatic-input-stall" {
		if r.Binding != nil || r.Scope == "" || r.Tag == "" || r.Session == "" || r.Nonce == "" || !validMessageID(r.ID) || r.Target != "" || r.Agent != "" || r.Confirmed || r.SameBinary || r.ForceUnknown || r.TailScope != "" || r.TailTag != "" || r.Lines != 0 {
			return errors.New("automatic-input-stall requires only caller identity, occurrence ID and typed report")
		}
		_, err := ParseAutomaticInputStall(r.Body)
		return err
	}
	if r.Op == "native-exit" {
		if r.Binding != nil || r.Scope == "" || r.Tag == "" || r.Session == "" || r.Nonce == "" || !validMessageID(r.ID) || r.Target != "" || r.Agent != "" || r.Confirmed || r.SameBinary || r.ForceUnknown || r.TailScope != "" || r.TailTag != "" || r.Lines != 0 {
			return errors.New("native-exit requires only caller identity, occurrence ID and typed report")
		}
		_, err := ParseNativeExit(r.Body)
		return err
	}
	if r.Op == "session-write" {
		if r.Binding != nil || r.Scope == "" || r.Tag == "" || r.Session == "" || r.Nonce == "" || r.ID != "" || r.Body != "" || r.Agent != "" || r.Confirmed || r.SameBinary || r.ForceUnknown || r.TailScope != "" || r.TailTag != "" || r.Lines != 0 {
			return errors.New("session-write requires only the calling conversation identity and exact owner slot")
		}
		_, _, err := parseSlot(r.Target)
		return err
	}
	if r.Op == "notify" {
		if r.Binding != nil || r.Scope == "" || r.Tag == "" || r.Session == "" || r.Nonce == "" || r.ID != "" || r.Target != "" || r.Agent != "" || r.Confirmed || r.SameBinary || r.ForceUnknown || r.TailScope != "" || r.TailTag != "" || r.Lines != 0 {
			return errors.New("notify requires only the calling conversation identity, title and message")
		}
		return notifyosc.ValidateDesktop(r.Title, r.Body)
	}
	if IsSubmissionOp(r.Op) {
		if r.Binding != nil || r.Scope == "" || r.Tag == "" || r.Session == "" || r.Nonce == "" || !core.ValidID(r.ID) || r.Target != "" || r.Body != "" || r.Agent != "" || r.Confirmed || r.SameBinary || r.ForceUnknown || r.TailScope != "" || r.TailTag != "" || r.Lines != 0 {
			return errors.New("submission operation requires only the calling conversation identity and a submission ID")
		}
		return nil
	}

	if r.Op == "register" || r.Op == "operator-submit" {
		if r.Binding == nil || r.Scope != "" || r.Tag != "" || r.Session != "" || r.Nonce != "" || r.ID != "" || r.Target != "" || r.Body != "" || r.Agent != "" {
			return errors.New("wrapper operation requires only an exact binding")
		}
		return r.Binding.Validate()
	}
	if r.Op == "tail" || r.TailScope != "" || r.TailTag != "" || r.Lines != 0 {
		// The operator's peek from any shell (pair#425): like
		// broadcast-status, no conversation identity, and a read-only answer.
		if r.Op != "tail" {
			return errors.New("a tail target applies only to tail")
		}
		if r.Binding != nil || r.Scope != "" || r.Tag != "" || r.Session != "" || r.Nonce != "" || r.ID != "" || r.Body != "" || r.Agent != "" || r.Confirmed || r.SameBinary || r.ForceUnknown {
			return errors.New("tail takes only a thread and a line count")
		}
		byThread := r.TailScope != "" && r.TailTag != ""
		if r.Target != "" {
			if r.TailScope != "" || r.TailTag != "" {
				return errors.New("tail names its thread by scope and tag or by slot, not both")
			}
			if _, _, err := parseSlot(r.Target); err != nil {
				return errors.New("tail requires an exact slot (repo:N)")
			}
		} else if !byThread {
			return errors.New("tail requires a thread scope and tag")
		}
		return ValidTailLines(r.Lines)
	}
	if IsReplayOp(r.Op) {
		// The operator's, or an agent's, request from any shell (pair#477):
		// like broadcast-status, no conversation identity. Target is the
		// recording's id; the recordings check its shape.
		if r.Binding != nil || r.Scope != "" || r.Tag != "" || r.Session != "" || r.Nonce != "" || r.ID != "" || r.Body != "" || r.Agent != "" || r.Confirmed || r.SameBinary || r.ForceUnknown {
			return errors.New(r.Op + " takes only a recording")
		}
		if r.Target == "" {
			return errors.New(r.Op + " requires a recording")
		}
		return nil
	}
	if r.Op == "resources" || r.Op == "refusals" {
		if r != (Request{Op: r.Op}) {
			return errors.New(r.Op + " takes no fields")
		}
		return nil
	}
	if r.Op == "reap-orphans" {
		// The resource query's destructive half (#583): the same owner-only
		// socket, no conversation identity, and exactly one field more. The
		// confirmation is required HERE rather than in the handler that acts on
		// it, because a caller's `--confirm` is presentation -- this is the
		// authority, and a hand-written request without it must not reach a
		// classification that ends processes.
		if !r.Confirmed {
			return errors.New("reap-orphans ends processes and requires --confirm")
		}
		if r != (Request{Op: "reap-orphans", Confirmed: true}) {
			return errors.New("reap-orphans takes only --confirm")
		}
		return nil
	}
	if r.Op == "broadcast-status" {
		// The operator's query from any shell (pair#413): no conversation
		// identity, no message fields. The socket's owner-only store is the
		// access control, and the answer is read-only.
		if r.Binding != nil || r.Scope != "" || r.Tag != "" || r.Session != "" || r.Nonce != "" || r.ID != "" || r.Target != "" || r.Body != "" || r.Agent != "" || r.Confirmed {
			return errors.New(r.Op + " takes no fields")
		}
		return nil
	}
	if r.Binding != nil || r.Scope == "" || r.Tag == "" || r.Session == "" || r.Nonce == "" {
		return errors.New("operation requires the calling conversation identity")
	}
	if r.Agent != "" && r.Op != "switch-agent" && (r.Op != "send" || !validFamily(r.Agent)) {
		return errors.New("an agent filter applies only to send and must be a plain agent name")
	}
	if r.Confirmed && r.Op != "switch-agent" && r.Op != "set-model" && r.Op != "interrupt" && !couchcore.IsSlotOperation(r.Op) {
		return errors.New("a confirmation applies only to slot operations")
	}
	if r.SameBinary && !couchcore.SlotOperationTakesSameBinary(r.Op) {
		return errors.New("--same-binary applies only to relaunch")
	}
	if r.ForceUnknown && !couchcore.SlotOperationTakesForceUnknown(r.Op) {
		return errors.New("--force-unknown applies only to relaunch, reload-context and prepare")
	}
	switch {
	case IsDriveOp(r.Op):
		if !validMessageID(r.ID) || r.Drive == nil || r.Body != "" || r.Agent != "" || r.Title != "" || r.ToHub != "" || r.Persist || r.Lines != 0 || r.TailScope != "" || r.TailTag != "" {
			return errors.New("drive requires only exact target, request ID and typed intent")
		}
		if _, _, err := parseSlot(r.Target); err != nil {
			return err
		}
		if err := r.Drive.Validate(); err != nil {
			return err
		}
		if (r.Op == "set-model") != (r.Drive.Model != nil) || (r.Op == "interrupt") != r.Drive.Interrupt || (r.Op == "keys") == r.Confirmed {
			return errors.New("set-model requires model intent and --confirm, interrupt requires interrupt intent and --confirm; keys takes no confirmation")
		}
	case r.Op == "recovery-confirmation":
		if !validMessageID(r.ID) || r.RecoveryConfirmation == "" || r.Target != "" || r.Body != "" || r.Agent != "" || r.Confirmed || r.CandidateKey != "" || r.Title != "" || r.TailScope != "" || r.TailTag != "" || r.Lines != 0 {
			return errors.New("recovery-confirmation requires only a request ID and token")
		}
	case r.Op == "resume-candidates":
		if r.ID != "" || r.Body != "" || r.Agent != "" || r.Confirmed || r.SameBinary || r.ForceUnknown {
			return errors.New("resume-candidates accepts only an exact slot")
		}
		_, _, err := parseSlot(r.Target)
		return err
	case r.Op == "switch-agent":
		if !validMessageID(r.ID) || !r.Confirmed || !launcher.IsSupportedAgent(r.Agent) {
			return errors.New("switch-agent requires a request ID, supported agent and confirmation")
		}
		if _, _, err := parseSlot(r.Target); err != nil {
			return err
		}
		if r.Body != "" {
			return errors.New("switch-agent does not accept content")
		}

	case couchcore.IsSlotOperation(r.Op):
		// One exact slot: a primitive acts on one slot, never a family.
		if !validMessageID(r.ID) || r.Body != "" {
			return errors.New(r.Op + " requires only a request ID and an exact slot")
		}
		// parseSlot refuses a family ("pair") as well as a malformed slot.
		if _, _, err := parseSlot(r.Target); err != nil {
			return err
		}
	case r.Op == "operation-status":
		if !validMessageID(r.ID) || r.Target != "" || r.Body != "" {
			return errors.New("operation-status requires only a request ID")
		}
	case r.Op == "actors":
		if r.ID != "" || r.Target != "" || r.Body != "" {
			return errors.New("actors takes no message fields")
		}
	case r.Op == "status":
		if !validMessageID(r.ID) || r.Target != "" || r.Body != "" {
			return errors.New("status requires only a message ID")
		}
	case r.Op == "send":
		if !validMessageID(r.ID) {
			return errors.New("invalid message ID")
		}
		if strings.Contains(r.Target, ":") {
			if _, _, err := parseSlot(r.Target); err != nil {
				return err
			}
		} else if !validFamily(r.Target) {
			return ErrInvalidTarget
		}
		return ValidateBody(r.Body)
	default:
		return fmt.Errorf("unknown message operation %q", r.Op)
	}
	return nil
}

// Handle is the broker protocol adapter for sender requests. Wrappers
// register through their lifecycle session (#365); the old per-second
// register and operator-submit requests are answered unsupported without any
// verification work, so a wrapper from an older binary costs nothing.
func Handle(ctx context.Context, b *Broker, r Request) Response {
	if err := ValidateRequest(r); err != nil {
		return Response{Code: "invalid-request", Error: err.Error()}
	}
	if err := ctx.Err(); err != nil {
		return protocolError(err, false)
	}
	if r.Op == "register" || r.Op == "operator-submit" {
		return Response{Code: "unsupported", Error: "this wrapper predates lifecycle sessions; relaunch the slot to receive messages"}
	}
	caller, err := b.Caller(r.Scope, r.Tag, r.Session, r.Nonce)
	if err != nil {
		return protocolError(fmt.Errorf("caller lookup: %w", err), false)
	}
	switch r.Op {
	case "actors":
		actors, err := b.Actors(ctx, caller)
		if err != nil {
			return protocolError(err, false)
		}
		return Response{Code: "ok", Actors: actors}
	case "status":
		receipt, err := b.StatusContext(ctx, caller, r.ID)
		if err != nil {
			return protocolError(err, false)
		}
		return Response{Code: "ok", Receipt: &receipt}
	case "send":
		receipt, err := b.Send(ctx, caller, r.ID, Route{Target: r.Target, Agent: r.Agent}, r.Body)
		if err != nil {
			return protocolError(err, !strings.Contains(r.Target, ":"))
		}
		return Response{Code: "accepted", Receipt: &receipt}
	}
	return Response{Code: "invalid-request", Error: "unknown message operation"}
}

func protocolError(err error, family bool) Response {
	code := "refused"
	switch {
	case errors.Is(err, ErrNoRecipient) || (family && errors.Is(err, ErrUnavailable)):
		code = "not-dispatched"
	case errors.Is(err, ErrUnavailable):
		code = "unavailable"
	case errors.Is(err, ErrRecipientBusy):
		code = "busy"
	case errors.Is(err, ErrBudgetExhausted):
		code = "breaker-exhausted"
	case errors.Is(err, ErrUnsupported):
		code = "unsupported"
	case errors.Is(err, ErrAmbiguous):
		code = "ambiguous"
	case errors.Is(err, ErrInvalidTarget):
		code = "invalid-target"
	case errors.Is(err, ErrUncertain) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
		code = "uncertain"
	}
	return Response{Code: code, Error: err.Error()}
}
