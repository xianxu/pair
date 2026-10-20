package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

type messagePhaseEndpoint struct {
	serviceEndpointFake
	active atomic.Bool
	phase  string
	err    error
}

func (e *messagePhaseEndpoint) Observe(ctx context.Context) (couchmessage.Observation, error) {
	if e.active.Load() && e.phase == "observe" {
		return couchmessage.Observation{}, e.err
	}
	return e.serviceEndpointFake.Observe(ctx)
}

func (e *messagePhaseEndpoint) Reserve(ctx context.Context, id string, sequence uint64) error {
	if e.active.Load() && e.phase == "reserve" {
		return e.err
	}
	return e.serviceEndpointFake.Reserve(ctx, id, sequence)
}

func TestMessageFailureNamesSideAndPhase(t *testing.T) {
	for _, target := range []string{"hub", "pair:0"} {
		for _, size := range []int{6, couchmessage.MaxBodyBytes} {
			for _, failure := range []string{"none", "sender-missing", "recipient-missing", "sender-current", "observe-unavailable", "reserve-unavailable", "observe-error", "reserve-error"} {
				t.Run(fmt.Sprintf("%s/%d/%s", target, size, failure), func(t *testing.T) {
					r := newServiceRig(t, func(a messageAuthority) messageAuthority { a.hub = "/repo0"; return a })
					sender := r.connect(7)
					hub := r.world.add(0)
					endpoint := &messagePhaseEndpoint{}
					r.world.mu.Lock()
					r.world.endpoints[hub] = endpoint
					r.world.mu.Unlock()
					r.attach(hub, "h0")
					r.wrapper(hub)
					r.waitConnected(hub, true)
					r.waitWorkspace(sender)
					r.waitWorkspace(hub)
					prefix, cause, code := "", "", "unavailable"
					switch failure {
					case "sender-missing":
						r.disconnect(sender)
						prefix, cause = "caller lookup", couchmessage.ErrUnavailable.Error()
					case "recipient-missing":
						r.disconnect(hub)
						prefix = "recipient " + target + " lookup"
						cause = couchmessage.ErrUnavailable.Error()
						if target == "hub" {
							cause = "the hub isn't connected"
						}
					case "sender-current":
						r.world.set(sender, func(s *worldSlot) { s.recorded = false })
						prefix, cause = "caller pair:7 current", "launch record moved on"
					case "observe-unavailable", "reserve-unavailable", "observe-error", "reserve-error":
						endpoint.phase = strings.Split(failure, "-")[0]
						endpoint.err = couchmessage.ErrUnavailable
						if strings.HasSuffix(failure, "-error") {
							endpoint.err = errors.New("fixture endpoint failure")
							code = "refused"
						}
						prefix, cause = "recipient pair:0 "+endpoint.phase, endpoint.err.Error()
						endpoint.active.Store(true)
					}
					req := callerRequest(sender, "send")
					req.ID, req.Target, req.Body = "phase-evidence", target, strings.Repeat("x", size)
					response := r.s.handle(t.Context(), req)
					if failure == "none" {
						if response.Code != "accepted" || response.Receipt == nil {
							t.Fatalf("live send refused: %+v", response)
						}
						message := response.Receipt.Message
						if message.ID != req.ID || message.From != sender || message.To != hub || message.Body != req.Body {
							t.Fatalf("hub alias/exact send changed message identity: %+v", message)
						}
						return
					}
					if response.Code != code || response.Receipt != nil {
						t.Errorf("phase labeling changed refusal policy: %+v, want code %s without receipt", response, code)
					}
					if !strings.HasPrefix(response.Error, prefix) || !strings.Contains(response.Error, cause) {
						t.Errorf("missing side, phase or original cause: %q, want prefix %q and cause %q", response.Error, prefix, cause)
					}
					if strings.Contains(strings.ToLower(response.Error), "restart") {
						t.Errorf("unproven restart advice: %q", response.Error)
					}
				})
			}
		}
	}
}

func TestMessageCLILegacyUnavailableDoesNotAdviseRestart(t *testing.T) {
	for _, op := range []string{"send", "actors", "status"} {
		for _, jsonOutput := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/json=%v", op, jsonOutput), func(t *testing.T) {
				var out, errout bytes.Buffer
				call := func(_ context.Context, _ string, _ any, result any) error {
					*result.(*couchmessage.Response) = couchmessage.Response{Code: "unavailable", Error: couchmessage.ErrUnavailable.Error()}
					return nil
				}
				inv := cliInvocation{kind: cliMessage, messageOp: op, ref: "pair:0", messageBody: "report", jsonOutput: jsonOutput}
				if op == "status" {
					inv.ref = "known-message"
				}
				if exit := runMessageCLIWithCall(inv, messageRuntime(), &out, &errout, call); exit != 1 {
					t.Fatalf("unavailable exit %d, want 1", exit)
				}
				if text := strings.ToLower(out.String() + errout.String()); strings.Contains(text, "restart") {
					t.Errorf("legacy ambiguous refusal acquired restart advice: %s", text)
				}
				if jsonOutput {
					var response couchmessage.Response
					if err := json.Unmarshal(out.Bytes(), &response); err != nil || response.Code != "unavailable" || response.Receipt != nil {
						t.Fatalf("legacy unavailable response changed: %q, %v", out.String(), err)
					}
				} else if !strings.Contains(errout.String(), "unavailable") {
					t.Fatalf("legacy refusal lost unavailable code: %q", errout.String())
				}
			})
		}
	}
}
