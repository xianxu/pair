package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// The scheduler persists the binding, ID and body before calling this port.
// An admitted notice is observed through the broker's one canonical receipt.
type scheduleNoticePort interface {
	Binding(root string) (json.RawMessage, bool)
	Admit(ctx context.Context, root string, binding json.RawMessage, id, body string) (notice string, err error)
	Observe(id string) (notice string, known, active bool)
}
type scheduleNotice struct{ service *messageService }

var _ scheduleNoticePort = scheduleNotice{}

func (n scheduleNotice) Binding(root string) (json.RawMessage, bool) {
	if n.service == nil {
		return nil, false
	}
	b, ok := n.service.hubBindingAt(root)
	if !ok || b.Validate() != nil {
		return nil, false
	}
	raw, err := json.Marshal(b)
	return raw, err == nil
}
func (n scheduleNotice) Admit(ctx context.Context, root string, frozen json.RawMessage, id, body string) (string, error) {
	var expected couchmessage.Binding
	if len(frozen) > couchmessage.MaxBindingBytes || strictjson.Decode(frozen, &expected) != nil || expected.Validate() != nil {
		return "dropped: hub changed", nil
	}
	if n.service == nil || n.service.broker == nil {
		return "dropped: hub unavailable", nil
	}
	release, err := holdHubEffect(ctx, n.service.authority.hubs)
	if err != nil {
		return "not-admitted", err
	}
	defer release()
	current, ok := n.service.hubBindingAt(root)
	if !ok {
		return "dropped: hub unavailable", nil
	}
	if current != expected {
		return "dropped: hub changed", nil
	}
	_, err = n.service.broker.SendSystem(ctx, expected, id, couchmessage.Route{Target: expected.Slot}, body)
	if err != nil {
		return scheduleNoticeAdmissionError(err), err
	}
	notice, _, _ := n.Observe(id)
	return notice, nil
}
func (n scheduleNotice) Observe(id string) (notice string, known, active bool) {
	if n.service == nil || n.service.broker == nil {
		return "", false, false
	}
	receipt, known, active := n.service.broker.ReceiptSnapshot(id)
	if !known || active {
		return "", known, active
	}
	return scheduleNoticeReceipt(receipt), true, false
}
func scheduleNoticeAdmissionError(err error) string {
	var refused *couchmessage.RefusedError
	if errors.As(err, &refused) {
		return "not-admitted"
	}
	return "uncertain"
}
func scheduleNoticeReceipt(r couchmessage.Receipt) string {
	switch r.Status {
	case couchmessage.Submitted:
		return "submitted"
	case couchmessage.Expired:
		return "expired"
	case couchmessage.Cancelled:
		return "dropped: cancelled"
	case couchmessage.Indeterminate:
		return "uncertain"
	case couchmessage.Refused:
		reason := strings.TrimSpace(strings.Map(func(r rune) rune {
			if unicode.IsControl(r) {
				return -1
			}
			return r
		}, r.Detail))
		if reason == "" {
			reason = "refused"
		}
		return "dropped: " + truncateBytes(reason, 512)
	}
	return ""
}
