package couchmessage

import (
	"errors"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"time"
)

// DrainRequest is a closed intent; worker acknowledgements carry no invented
// idle, checkpoint, queue or operator-authority fields.
type DrainRequest struct {
	Action        string `json:"action"`
	ID            string `json:"id,omitempty"`
	Reason        string `json:"reason,omitempty"`
	Fast          bool   `json:"fast,omitempty"`
	TimeoutMillis int64  `json:"timeout_millis,omitempty"`
}

func ValidateDrainRequest(r Request) error {
	d := r.Drain
	if d == nil {
		return errors.New("drain requires a typed intent")
	}
	allowed := Request{Op: "drain", Scope: r.Scope, Tag: r.Tag, Session: r.Session, Nonce: r.Nonce, ID: r.ID, Drain: d}
	if r != allowed {
		return errors.New("drain carries unrelated fields")
	}
	for _, s := range []string{r.Scope, r.Tag, r.Session, r.Nonce} {
		if len(s) > MaxBindingBytes {
			return errors.New("drain identity exceeds limit")
		}
	}
	if d.Action != "status" && (r.Scope == "" || r.Tag == "" || r.Session == "" || r.Nonce == "") {
		return errors.New("drain mutation requires live caller identity")
	}
	switch d.Action {
	case "start":
		if d.ID != "" || d.TimeoutMillis < 1000 || d.TimeoutMillis > int64((24*time.Hour)/time.Millisecond) {
			return errors.New("drain timeout must be 1s through 24h")
		}
		_, err := couchcore.DrainNotice(r.ID, d.Reason, d.Fast)
		return err
	case "ack":
		if r.ID != "" || d.Reason != "" || d.Fast || d.TimeoutMillis != 0 {
			return errors.New("drain ack carries only ID and caller")
		}
		_, err := couchcore.DrainNotice(d.ID, "", false)
		return err
	case "status", "cancel":
		if r.ID != "" || d.ID != "" || d.Reason != "" || d.Fast || d.TimeoutMillis != 0 {
			return errors.New("drain status/cancel carries no start options")
		}
		return nil
	default:
		return errors.New("unknown drain action")
	}
}
