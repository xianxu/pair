package couchcmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/rowtext"
)

// scheduleRunService is the single scheduler owner's transport-facing surface.
// Submit acknowledges only durable reservation; Observe is strictly read-only.
type scheduleRunService interface {
	Submit(context.Context, couchschedule.ManualIntent) (couchschedule.ManualStatus, error)
	Observe(context.Context, string, string, string, int) (couchschedule.ManualStatus, error)
	Cancel(context.Context, string, string, string) (couchschedule.ManualStatus, error)
}

func handleScheduleRun(ctx context.Context, r couchmessage.Request, hubs []string, owner scheduleRunService) couchmessage.Response {
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	if !couchmessage.IsScheduleRunOp(r.Op) || r.Manual == nil {
		return couchmessage.Response{Code: "invalid-request", Error: "not a manual scheduler operation"}
	}
	m := r.Manual.Intent
	configured := false
	for _, hub := range hubs {
		if hub == m.Hub {
			configured = true
			break
		}
	}
	if !configured {
		return couchmessage.Response{Code: "refused", Error: "manual hub is not a configured canonical scheduler hub"}
	}
	if owner == nil {
		return couchmessage.Response{Code: "unavailable", Error: "scheduler owner is unavailable"}
	}
	if err := ctx.Err(); err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	var status couchschedule.ManualStatus
	var err error
	switch r.Op {
	case "schedule-run":
		status, err = owner.Submit(ctx, m)
	case "schedule-run-status":
		status, err = owner.Observe(ctx, m.Hub, m.Task, m.ID, r.Manual.Offset)
	case "schedule-run-cancel":
		status, err = owner.Cancel(ctx, m.Hub, m.Task, m.ID)
	}
	if err != nil {
		code := "refused"
		// An owner write can fail after rename. Without an explicit no-effect
		// proof, a submit error cannot authorize another occurrence.
		if r.Op == "schedule-run" || r.Op == "schedule-run-cancel" {
			code = "uncertain"
		}
		return couchmessage.Response{Code: code, Error: fmt.Sprintf("run ID %s: %v", m.ID, err)}
	}
	if err = validateManualResponse(status, m, r.Manual.Offset, r.Op == "schedule-run-status"); err != nil {
		return couchmessage.Response{Code: "invalid-response", Error: err.Error()}
	}
	return couchmessage.Response{Code: "ok", Manual: &status}
}
func validateManualResponse(s couchschedule.ManualStatus, m couchschedule.ManualIntent, offset int, checkOffset bool) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if s.ID != m.ID || s.Hub != m.Hub || s.Task != m.Task || checkOffset && s.Offset != offset {
		return errors.New("manual response does not match requested occurrence or offset")
	}
	return nil
}

func runScheduleRemote(ctx context.Context, a *scheduleArgs, hub, socket string, stdout, stderr io.Writer, call messageCall) int {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return scheduleCLIError(stderr, err)
	}
	intent := couchschedule.ManualIntent{ID: hex.EncodeToString(nonce[:]), Hub: hub, Task: a.task, Args: append([]string(nil), a.args...)}
	if err := intent.Validate(); err != nil {
		return scheduleCLIError(stderr, err)
	}
	exchange := func(ctx context.Context, op string, offset int) (couchschedule.ManualStatus, error) {
		m := intent
		if op != "schedule-run" {
			m.Args = nil
		}
		request := couchmessage.Request{Op: op, Manual: &couchschedule.ManualRequest{Intent: m, Offset: offset}}
		rpcCtx, cancel := context.WithTimeout(ctx, couchmessage.TransportTimeout)
		defer cancel()
		var response couchmessage.Response
		if err := call(rpcCtx, socket, request, &response); err != nil {
			return couchschedule.ManualStatus{}, err
		}
		if response.Code != "ok" || response.Manual == nil {
			return couchschedule.ManualStatus{}, fmt.Errorf("manual request refused (%s): %s", response.Code, response.Error)
		}
		if err := validateManualResponse(*response.Manual, intent, offset, op == "schedule-run-status"); err != nil {
			return couchschedule.ManualStatus{}, err
		}
		return *response.Manual, nil
	}
	cancelRun := func() int {
		cleanup, done := context.WithTimeout(context.Background(), couchmessage.TransportTimeout)
		defer done()
		status, err := exchange(cleanup, "schedule-run-cancel", 0)
		if err != nil || !status.Terminal() {
			fmt.Fprintf(stderr, "couch: schedule: cancellation uncertain; run ID %s; inspect history %s --run %s\n", intent.ID, intent.Task, intent.ID)
		}
		return 130
	}
	if err := ctx.Err(); err != nil {
		return scheduleCLIError(stderr, err)
	}
	status, err := exchange(ctx, "schedule-run", 0)
	if err != nil {
		fmt.Fprintf(stderr, "couch: schedule: unknown admission; run ID %s; inspect history %s --run %s: %v\n", intent.ID, intent.Task, intent.ID, rowtext.Sanitize(err.Error()))
		if ctx.Err() != nil {
			return cancelRun()
		}
		return 1
	}
	for !status.Terminal() {
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return cancelRun()
		case <-timer.C:
		}
		status, err = exchange(ctx, "schedule-run-status", 0)
		if err != nil {
			if ctx.Err() != nil {
				return cancelRun()
			}
			return scheduleCLIError(stderr, fmt.Errorf("run ID %s: status failed; inspect history: %w", intent.ID, err))
		}
	}
	// Completion output is immutable; every page remains tied to the same run.
	total := status.Total
	for {
		if status.Total != total || !status.Terminal() {
			return scheduleCLIError(stderr, errors.New("manual completion changed during output pagination"))
		}
		if _, err = stdout.Write(status.Output); err != nil {
			return scheduleCLIError(stderr, err)
		}
		next := status.Offset + len(status.Output)
		if next == total {
			break
		}
		if len(status.Output) == 0 {
			return scheduleCLIError(stderr, errors.New("manual output pagination made no progress"))
		}
		status, err = exchange(ctx, "schedule-run-status", next)
		if err != nil {
			return scheduleCLIError(stderr, err)
		}
	}
	if status.Error != "" {
		fmt.Fprintln(stderr, "couch: schedule:", rowtext.Sanitize(status.Error))
	}
	if status.Exit == nil || *status.Exit < 0 || *status.Exit > 255 {
		return 1
	}
	return *status.Exit
}
