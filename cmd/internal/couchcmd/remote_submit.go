package couchcmd

import (
	"context"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

// admitLocalSubmit runs after the ordinary caller/current/route checks. It is
// active even when remote mode is off; receipt text cannot choose provenance.
func (s *messageService) admitLocalSubmit(ctx context.Context, from couchmessage.Binding, request couchmessage.Request) (couchmessage.Response, bool) {
	path, submit := sessionsync.SubmitPath(request.Body)
	if !submit {
		return couchmessage.Response{}, false
	}
	s.originAdmission.Lock()
	defer s.originAdmission.Unlock()
	if s.originNamespace == "" {
		return couchmessage.Response{}, false
	}
	refuse := func(reason string) (couchmessage.Response, bool) {
		return couchmessage.Response{Code: "refused", Error: reason}, true
	}
	syncer := sessionsync.New(s.originNamespace)
	registration, err := syncer.RegistrationForPath(path)
	if err != nil {
		return refuse("registered submit evidence unavailable; inspect registration")
	}
	if registration == nil {
		return couchmessage.Response{}, false
	}
	if s.origins == nil || s.originErr != nil {
		return refuse("remote origin evidence unavailable; inspect storage and restart Couch")
	}
	info, err := syncer.InspectRegistered(path)
	if err != nil || info.ID != registration.ID || info.Owner != from.Slot {
		return refuse("registered submit owner changed or unavailable; inspect registration")
	}
	marked, pending, err := s.origins.Inspect(info.ID)
	if err != nil {
		return refuse("remote origin evidence unavailable; inspect storage and restart Couch")
	}
	if pending {
		return refuse("document forget pending; inspect registration and retry")
	}
	if !marked {
		return couchmessage.Response{}, false
	} // Preserve legacy unmarked peer wire.
	// The conservative bit survives receipt and audit retention. M3 never clears
	// it: pair#795 owns proof-based settlement and eventual forget recovery.
	if err := s.origins.MarkSubmitted(info.ID); err != nil {
		return refuse("remote submit history unavailable; inspect storage")
	}
	receipt, err := s.broker.SendLocal(ctx, from, request.ID, couchmessage.Route{Target: request.Target, Agent: request.Agent}, request.Body, true, info.ID)
	if err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error(), Receipt: &receipt}, true
	}
	return couchmessage.Response{Code: "accepted", Receipt: &receipt}, true
}
