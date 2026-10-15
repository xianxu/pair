package couchmessage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

func endpointMessage() Message {
	return Message{ID: "message", From: protocolBinding("pair:0"), To: protocolBinding("pair:1"), Body: "do work", Deadline: time.Now().Add(time.Second)}
}

func TestEndpointValidation(t *testing.T) {
	m := endpointMessage()
	for _, op := range []string{"observe", "reserve", "release", "commit", "status"} {
		r := EndpointRequest{Op: op, Binding: m.To}
		if op != "observe" {
			r.ID = m.ID
		}
		if op == "commit" {
			r.Message = &m
		}
		if err := ValidateEndpointRequest(r); err != nil {
			t.Fatalf("%s: %v", op, err)
		}
		r.Binding = Binding{}
		if err := ValidateEndpointRequest(r); err == nil {
			t.Fatalf("%s accepts empty binding", op)
		}
	}
	r := EndpointRequest{Op: "commit", Binding: m.To, ID: "other", Message: &m}
	if ValidateEndpointRequest(r) == nil {
		t.Fatal("mismatching ID accepted")
	}
	r = EndpointRequest{Op: "observe", Binding: m.To, ID: m.ID}
	if ValidateEndpointRequest(r) == nil {
		t.Fatal("extra field accepted")
	}
}

func TestRemoteEndpointStatefulExchange(t *testing.T) {
	m := endpointMessage()
	var mu sync.Mutex
	var reserved string
	var stored *Message
	var commits, polls int
	socket := transportSocket(t)
	s, err := StartServer(context.Background(), socket, func(_ context.Context, raw []byte) ([]byte, error) {
		var r EndpointRequest
		if err := strictjson.Decode(raw, &r); err != nil {
			return nil, err
		}
		if err := ValidateEndpointRequest(r); err != nil {
			return nil, err
		}
		mu.Lock()
		defer mu.Unlock()
		out := EndpointResponse{}
		switch r.Op {
		case "observe":
			out.Observation = Observation{Sequence: 7, LastActivity: time.Now().Add(-time.Minute)}
		case "reserve":
			if r.Sequence != 7 || reserved != "" {
				out.Error = "reservation refused"
			} else {
				reserved = r.ID
			}
		case "release":
			if reserved == r.ID {
				reserved = ""
			}
		case "commit":
			if reserved != r.ID {
				out.Error = "not reserved"
			} else {
				stored = r.Message
				commits++
				out.Receipt = &Receipt{Message: *stored, Status: Queued}
			}
		case "status":
			polls++
			// A held delivery reports its hold before the outcome (pair#435).
			out.Receipt = &Receipt{Message: *stored, Status: Held, Detail: "waiting for empty composer"}
			if polls > 1 {
				out.Receipt = &Receipt{Message: *stored, Status: Submitted}
			}
		}
		return json.Marshal(out)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	e := RemoteEndpoint{Binding: m.To, socket: socket}
	obs, err := e.Observe(context.Background())
	if err != nil || obs.Sequence != 7 {
		t.Fatalf("observe: %#v %v", obs, err)
	}
	if err := e.Reserve(context.Background(), m.ID, obs.Sequence); err != nil {
		t.Fatal(err)
	}
	var interim []Receipt
	r, err := e.Deliver(context.Background(), m, func(r Receipt) { interim = append(interim, r) })
	if err != nil || r.Status != Submitted {
		t.Fatalf("deliver: %#v %v", r, err)
	}
	if len(interim) != 2 || interim[0].Status != Queued || interim[1].Status != Held || interim[1].Detail != "waiting for empty composer" {
		t.Fatalf("interim receipts: %+v", interim)
	}
	if err := e.Release(context.Background(), m.ID); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if commits != 1 || polls != 2 || reserved != "" {
		t.Fatalf("commits %d polls %d reserved %q", commits, polls, reserved)
	}
}

func TestRemoteEndpointRejectsWrongReceiptAndNeverRetriesCommit(t *testing.T) {
	for _, mode := range []string{"wrong-id", "wrong-body", "wrong-binding", "unknown-status", "disconnect"} {
		t.Run(mode, func(t *testing.T) {
			m := endpointMessage()
			socket := transportSocket(t)
			commits := 0
			s, err := StartServer(context.Background(), socket, func(_ context.Context, raw []byte) ([]byte, error) {
				var req EndpointRequest
				if err := strictjson.Decode(raw, &req); err != nil {
					return nil, err
				}
				commits++
				if mode == "disconnect" {
					return nil, errors.New("lost outcome")
				}
				bad := m
				status := Submitted
				switch mode {
				case "wrong-id":
					bad.ID = "other"
				case "wrong-body":
					bad.Body = "changed"
				case "wrong-binding":
					bad.To.Nonce = "replacement"
				case "unknown-status":
					status = "nonsense"
				}
				return json.Marshal(EndpointResponse{Receipt: &Receipt{Message: bad, Status: status}})
			})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			e := RemoteEndpoint{Binding: m.To, socket: socket}
			if _, err := e.Deliver(context.Background(), m, nil); err == nil {
				t.Fatal("bad delivery accepted")
			}
			// Close joins the handler before reading its counter.
			s.Close()
			if commits != 1 {
				t.Fatalf("commit attempts %d", commits)
			}
		})
	}
}

func TestEndpointSocketIncludesExactIncarnation(t *testing.T) {
	b := protocolBinding("pair:0")
	a, err := EndpointSocket("/namespace", b)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Binding){func(b *Binding) { b.Start = "new" }, func(b *Binding) { b.Nonce = "new" }, func(b *Binding) { b.Session = "new" }, func(b *Binding) { b.Scope = "new" }, func(b *Binding) { b.Tag = "new" }} {
		other := b
		change(&other)
		got, err := EndpointSocket("/namespace", other)
		if err != nil || got == a {
			t.Fatalf("identity collision %q %v", got, err)
		}
	}
}

func TestRemoteEndpointPollingStopsAtContextDeadline(t *testing.T) {
	m := endpointMessage()
	socket := transportSocket(t)
	var mu sync.Mutex
	commits := 0
	s, err := StartServer(context.Background(), socket, func(_ context.Context, raw []byte) ([]byte, error) {
		var r EndpointRequest
		if err := strictjson.Decode(raw, &r); err != nil {
			return nil, err
		}
		mu.Lock()
		defer mu.Unlock()
		if r.Op == "commit" {
			commits++
		}
		return json.Marshal(EndpointResponse{Receipt: &Receipt{Message: m, Status: Queued}})
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 125*time.Millisecond)
	defer cancel()
	e := RemoteEndpoint{Binding: m.To, socket: socket}
	if _, err := e.Deliver(ctx, m, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if commits != 1 {
		t.Fatalf("commit attempts: %d", commits)
	}
}

func TestRemoteEndpointCollectsOutcomeAfterInputDeadline(t *testing.T) {
	for _, outcome := range []Status{Expired, Submitted} {
		t.Run(string(outcome), func(t *testing.T) {
			m := endpointMessage()
			m.Deadline = time.Now().Add(100 * time.Millisecond)
			socket := transportSocket(t)
			commits := 0
			s, err := StartServer(context.Background(), socket, func(_ context.Context, raw []byte) ([]byte, error) {
				var req EndpointRequest
				if err := strictjson.Decode(raw, &req); err != nil {
					return nil, err
				}
				status := Queued
				if req.Op == "commit" {
					commits++
				} else if req.Op == "status" && !time.Now().Before(m.Deadline) {
					// The wrapper learns expiry at the input deadline, or reports
					// a submission completed before it. Only read the outcome.
					status = outcome
				}
				return json.Marshal(EndpointResponse{Receipt: &Receipt{Message: m, Status: status}})
			})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			e := RemoteEndpoint{Binding: m.To, socket: socket}
			r, err := e.Deliver(context.Background(), m, nil)
			s.Close()
			if err != nil || r.Status != outcome {
				t.Fatalf("outcome=%s err=%v", r.Status, err)
			}
			if commits != 1 {
				t.Fatalf("replayed input: %d commits", commits)
			}
		})
	}
}

// A wrapper's busy refusal crosses the socket as ErrRecipientBusy, so the
// broker queues the message rather than refusing its sender (pair#492).
func TestRemoteEndpointBusyReservationIsRecipientBusy(t *testing.T) {
	m := endpointMessage()
	socket := transportSocket(t)
	s, err := StartServer(context.Background(), socket, func(context.Context, []byte) ([]byte, error) {
		return json.Marshal(EndpointResponse{Error: ErrRecipientBusy.Error()})
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	e := RemoteEndpoint{Binding: m.To, socket: socket}
	if err := e.Reserve(context.Background(), m.ID, 1); !errors.Is(err, ErrRecipientBusy) {
		t.Fatalf("reserve: %v", err)
	}
}

func TestEndpointMissingOrRefusedExplainsRelaunch(t *testing.T) {
	for _, refused := range []bool{false, true} {
		t.Run(fmt.Sprint(refused), func(t *testing.T) {
			socket := transportSocket(t)
			cause := error(os.ErrNotExist)
			if refused {
				listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socket, Net: "unix"})
				if err != nil {
					t.Fatal(err)
				}
				listener.SetUnlinkOnClose(false)
				if err := listener.Close(); err != nil {
					t.Fatal(err)
				}
				cause = syscall.ECONNREFUSED
			}
			endpoint := RemoteEndpoint{Binding: protocolBinding("pair:1"), socket: socket}
			_, err := endpoint.Observe(context.Background())
			if err == nil || !strings.Contains(err.Error(), "messaging unavailable until next relaunch") || !errors.Is(err, cause) {
				t.Fatalf("endpoint diagnostic/cause = %v; want relaunch advice and %v", err, cause)
			}
		})
	}
}

func TestEndpointCanceledObservationDoesNotClaimBrokenChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	endpoint := RemoteEndpoint{Binding: protocolBinding("pair:1"), socket: transportSocket(t)}
	_, err := endpoint.Observe(ctx)
	if !errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "next relaunch") {
		t.Fatalf("cancellation misdiagnosed: %v", err)
	}
}
