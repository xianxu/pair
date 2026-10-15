package couchmessage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
)

type EndpointRequest struct {
	Drive    *DriveStart `json:",omitempty"`
	Op       string
	Binding  Binding
	ID       string
	Sequence uint64
	Message  *Message
	// Lines is a tail request's line count (pair#425); tail only.
	Lines int `json:",omitempty"`
}

type EndpointResponse struct {
	DriveObservation *DriveObservation  `json:",omitempty"`
	DriveResult      *agentdrive.Result `json:",omitempty"`
	Error            string
	Observation      Observation
	Receipt          *Receipt
	Tail             *Tail `json:",omitempty"`
}

// ReceiptTimeout permits read-only outcome collection after input stops.
// It never extends Message.Horizon or authorizes another commit/paste.
const ReceiptTimeout = 2 * time.Second

func ValidateEndpointRequest(r EndpointRequest) error {
	if err := r.Binding.Validate(); err != nil {
		return err
	}
	if r.Drive != nil && r.Op != "drive-start" {
		return errors.New("drive payload applies only to drive-start")
	}
	if r.Lines != 0 && r.Op != "tail" {
		return errors.New("a line count applies only to tail")
	}
	switch r.Op {
	case "drive-observe":
		if r.ID != "" || r.Sequence != 0 || r.Message != nil || r.Lines != 0 || r.Drive != nil {
			return errors.New("drive-observe requires only a binding")
		}
	case "drive-status", "drive-cancel":
		if !validMessageID(r.ID) || r.Sequence != 0 || r.Message != nil || r.Lines != 0 || r.Drive != nil {
			return errors.New("drive status/cancel requires binding and ID")
		}
	case "drive-start":
		if !validMessageID(r.ID) || r.Sequence != 0 || r.Message != nil || r.Lines != 0 || r.Drive == nil {
			return errors.New("drive-start requires binding, ID and typed drive")
		}
		return r.Drive.Validate(time.Now())

	case "tail":
		if r.ID != "" || r.Sequence != 0 || r.Message != nil {
			return errors.New("tail requires only a binding and a line count")
		}
		return ValidTailLines(r.Lines)
	case "observe":
		if r.ID != "" || r.Sequence != 0 || r.Message != nil {
			return errors.New("observe requires only a binding")
		}
	case "reserve":
		if !validMessageID(r.ID) || r.Message != nil {
			return errors.New("reserve requires message ID and observation sequence")
		}
	case "release", "status":
		if !validMessageID(r.ID) || r.Sequence != 0 || r.Message != nil {
			return errors.New("operation requires only binding and message ID")
		}
	case "commit":
		if r.Message == nil || !validMessageID(r.ID) || r.Sequence != 0 {
			return errors.New("commit requires binding, ID and message")
		}
		m := r.Message
		if m.ID != r.ID || m.To != r.Binding || m.From.Validate() != nil || m.Deadline.IsZero() {
			return errors.New("commit message identity mismatch")
		}
		return ValidateBody(m.Body)
	default:
		return fmt.Errorf("unknown endpoint operation %q", r.Op)
	}
	return nil
}

func EndpointSocket(namespace string, binding Binding) (string, error) {
	if err := binding.Validate(); err != nil {
		return "", err
	}
	// JSON field boundaries avoid ambiguous concatenations of identity values.
	raw, err := json.Marshal(binding)
	if err != nil {
		return "", err
	}
	socket, err := SocketPath(namespace, "wrapper:"+string(raw))
	if err != nil {
		return "", err
	}
	// Retain the exact-incarnation hash while making dead owners discoverable.
	return filepath.Join(filepath.Dir(socket), fmt.Sprintf("wrapper-%d-%s", binding.PID, filepath.Base(socket))), nil
}

// RemoteEndpoint sends commit exactly once. Subsequent RPCs only observe the
// retained outcome; neither transport failure nor timeout retries PTY input.
type RemoteEndpoint struct {
	Namespace string
	Binding   Binding
	socket    string // isolated socket injection for package tests
}

var _ DeliveryEndpoint = RemoteEndpoint{}

func (e RemoteEndpoint) call(ctx context.Context, r EndpointRequest) (EndpointResponse, error) {
	r.Binding = e.Binding
	if err := ValidateEndpointRequest(r); err != nil {
		return EndpointResponse{}, err
	}
	socket := e.socket
	if socket == "" {
		var err error
		socket, err = EndpointSocket(e.Namespace, e.Binding)
		if err != nil {
			return EndpointResponse{}, err
		}
	}
	var response EndpointResponse
	if err := Call(ctx, socket, r, &response); err != nil {
		var dial *net.OpError
		if errors.As(err, &dial) && dial.Op == "dial" && (errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ECONNREFUSED)) {
			return EndpointResponse{}, fmt.Errorf("messaging unavailable until next relaunch: %w", err)
		}
		return EndpointResponse{}, err
	}
	switch {
	case response.Error == ErrAlreadyCommitted.Error() && response.Receipt != nil:
		return EndpointResponse{}, &AlreadyCommittedError{Receipt: *response.Receipt}
	case response.Error == ErrUnknownDelivery.Error():
		return EndpointResponse{}, ErrUnknownDelivery
	case response.Error == ErrRecipientBusy.Error():
		// A wrapper refusing a reservation holds the message in the broker's
		// queue instead of refusing the sender (pair#492).
		return EndpointResponse{}, ErrRecipientBusy
	case response.Error != "":
		return EndpointResponse{}, errors.New(response.Error)
	}
	return response, nil
}

// Retained reads the wrapper's receipt for id without any effect.
func (e RemoteEndpoint) Retained(ctx context.Context, id string) (Receipt, error) {
	r, err := e.call(ctx, EndpointRequest{Op: "status", ID: id})
	if err != nil {
		return Receipt{}, err
	}
	if r.Receipt == nil || r.Receipt.Message.ID != id {
		return Receipt{}, errors.New("wrapper returned a receipt for another message")
	}
	return *r.Receipt, nil
}

func (e RemoteEndpoint) Observe(ctx context.Context) (Observation, error) {
	r, err := e.call(ctx, EndpointRequest{Op: "observe"})
	return r.Observation, err
}

// TailReader reads a wrapper's in-memory terminal tail (pair#425).
type TailReader interface {
	Tail(context.Context, int) (Tail, error)
}

var _ TailReader = RemoteEndpoint{}

func (e RemoteEndpoint) Tail(ctx context.Context, lines int) (Tail, error) {
	r, err := e.call(ctx, EndpointRequest{Op: "tail", Lines: lines})
	if err != nil {
		return Tail{}, err
	}
	if r.Tail == nil {
		return Tail{}, errors.New("wrapper returned no tail")
	}
	return BoundTail(*r.Tail), nil
}

func (e RemoteEndpoint) Reserve(ctx context.Context, id string, sequence uint64) error {
	_, err := e.call(ctx, EndpointRequest{Op: "reserve", ID: id, Sequence: sequence})
	return err
}
func (e RemoteEndpoint) Release(ctx context.Context, id string) error {
	_, err := e.call(ctx, EndpointRequest{Op: "release", ID: id})
	return err
}

func matchingReceipt(r Receipt, m Message) bool {
	return r.Message.ID == m.ID && r.Message.From == m.From && r.Message.To == m.To && r.Message.Body == m.Body && r.Message.Deadline.Equal(m.Deadline)
}

// peerHeldPoll paces the status poll while the recipient holds a delivery,
// which may last MaxHold.
const peerHeldPoll = time.Second

func (e RemoteEndpoint) Deliver(parent context.Context, m Message, progress func(Receipt)) (Receipt, error) {
	ctx, cancel := context.WithDeadline(parent, m.Horizon().Add(ReceiptTimeout))
	defer cancel()
	commitCtx, stopCommit := context.WithDeadline(ctx, m.Deadline)
	r, err := e.call(commitCtx, EndpointRequest{Op: "commit", ID: m.ID, Message: &m})
	stopCommit()
	if err != nil {
		return Receipt{}, err
	}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	var last Receipt
	for {
		if r.Receipt != nil {
			if !matchingReceipt(*r.Receipt, m) {
				return Receipt{}, errors.New("endpoint returned a mismatched receipt")
			}
			if r.Receipt.Status.Terminal() {
				return *r.Receipt, nil
			}
			if r.Receipt.Status != Queued && r.Receipt.Status != Held && r.Receipt.Status != Delivering {
				return Receipt{}, errors.New("endpoint returned an invalid delivery status")
			}
			if r.Receipt.Status != last.Status {
				poll := 50 * time.Millisecond
				if r.Receipt.Status == Held {
					poll = peerHeldPoll
				}
				ticker.Reset(poll)
			}
			if progress != nil && (r.Receipt.Status != last.Status || r.Receipt.Detail != last.Detail) {
				progress(*r.Receipt)
			}
			last = *r.Receipt
		}
		select {
		case <-ctx.Done():
			return Receipt{}, ctx.Err()
		case <-ticker.C:
		}
		r, err = e.call(ctx, EndpointRequest{Op: "status", ID: m.ID})
		if err != nil {
			return Receipt{}, err
		}
		if r.Receipt == nil {
			return Receipt{}, errors.New("endpoint returned no delivery status")
		}
	}
}
