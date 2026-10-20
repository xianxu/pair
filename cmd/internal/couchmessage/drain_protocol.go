package couchmessage

import (
	"errors"
	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// DrainRequest changes transient requested posture, never checkpoint authority.
type DrainRequest struct {
	Action  string `json:"action"`
	Fast    bool   `json:"fast,omitempty"`
	Message string `json:"message,omitempty"`
}

type DrainDelivery struct {
	Workspace string
	Receipt   *Receipt `json:",omitempty"`
	Error     string   `json:",omitempty"`
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
	if d.Action != "status" && (r.Scope == "" || r.Tag == "" || r.Session == "" || r.Nonce == "" || r.ID == "") {
		return errors.New("drain mutation requires live caller identity and request ID")
	}
	switch d.Action {
	case "start":
		if d.Message != "" {
			return errors.New("drain start carries no continuation message")
		}
	case "undrain":
		if d.Fast {
			return errors.New("undrain carries no fast option")
		}
		_, err := couchcore.UndrainNotice(d.Message)
		return err
	case "status":
		if r.ID != "" || d.Fast || d.Message != "" {
			return errors.New("drain status carries no mutation options")
		}
	default:
		return errors.New("unsupported drain action; see couch --help")
	}
	return nil
}
