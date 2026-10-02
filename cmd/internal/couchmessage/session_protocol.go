package couchmessage

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// Session frames ride one long-lived wrapper→broker connection (#365). The
// connection itself is the lifecycle fact: it closes when the wrapper dies or
// execs (Go sockets are close-on-exec), so no frame announces departure.
const (
	FrameHello    = "hello"    // wrapper → broker, first frame, carries the binding
	FrameAck      = "ack"      // broker → wrapper, once, answers hello
	FrameActivity = "activity" // wrapper → broker, coalesced observation change
	FrameSubmit   = "submit"   // wrapper → broker, a genuine operator submission
)

// MaxSessionFrameBytes bounds every session frame: the largest is a hello,
// one encoded binding plus envelope.
const MaxSessionFrameBytes = MaxBindingBytes + 1024

type SessionFrame struct {
	Op          string
	Binding     *Binding     `json:",omitempty"`
	Observation *Observation `json:",omitempty"`
	Code        string       `json:",omitempty"` // ack: "ok" or a protocol code
	Error       string       `json:",omitempty"`
}

func (f SessionFrame) Validate() error {
	switch f.Op {
	case FrameHello:
		if f.Binding == nil || f.Observation != nil {
			return errors.New("hello carries exactly a binding")
		}
		return f.Binding.Validate()
	case FrameAck:
		if f.Code == "" || f.Binding != nil || f.Observation != nil {
			return errors.New("ack carries only a code")
		}
	case FrameActivity, FrameSubmit:
		if f.Observation == nil || f.Binding != nil {
			return errors.New("observation frames carry exactly an observation")
		}
	default:
		return errors.New("unknown session frame")
	}
	return nil
}

func EncodeSessionFrame(f SessionFrame) ([]byte, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(f)
	if err == nil && len(raw) > MaxSessionFrameBytes {
		err = errors.New("session frame exceeds limit")
	}
	return raw, err
}

func DecodeSessionFrame(raw []byte) (SessionFrame, error) {
	var f SessionFrame
	if len(raw) > MaxSessionFrameBytes {
		return f, errors.New("session frame exceeds limit")
	}
	if err := strictjson.Decode(raw, &f); err != nil {
		return SessionFrame{}, err
	}
	return f, f.Validate()
}

// ReconnectBackoff paces a wrapper's redials while the broker is absent: the
// cost is one failed connect() per interval, never a scan.
type ReconnectBackoff struct {
	Base, Cap time.Duration
	// StableAfter is how long a session must have lasted for the next failure
	// to start again from Base.
	StableAfter time.Duration
}

var DefaultReconnectBackoff = ReconnectBackoff{Base: 250 * time.Millisecond, Cap: 5 * time.Second, StableAfter: 30 * time.Second}

// Next is the delay before redial attempt n (0-based).
func (b ReconnectBackoff) Next(n int) time.Duration {
	d := b.Base
	for i := 0; i < n && d < b.Cap; i++ {
		d *= 2
	}
	if d > b.Cap {
		d = b.Cap
	}
	return d
}
