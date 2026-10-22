package couchcmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

const remoteForgetFollowup = "pair#795"

func requestSessionForget(namespace, file string, call messageCall) (bool, error) {
	socket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		return false, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), couchmessage.TransportTimeout)
	defer cancel()
	var response couchmessage.Response
	if err := call(ctx, socket, couchmessage.Request{Op: "session-forget", Target: file}, &response); err != nil {
		return false, fmt.Errorf("forget outcome unconfirmed; inspect the registration, then start Couch and retry: %w", err)
	}
	if predatesOp(response) || response.Code == "invalid-request" {
		return false, errors.New("running Couch cannot coordinate forget; start Couch and retry")
	}
	if response.Code != "ok" || response.Forget == nil {
		return false, fmt.Errorf("forget refused: %s", response.Error)
	}
	return response.Forget.Changed, nil
}

// handleSessionForget runs only on the owner-private local broker socket. It
// accepts no remote-web route and creates no second origin writer.
func (s *messageService) handleSessionForget(r couchmessage.Request) couchmessage.Response {
	if r.Op != "session-forget" {
		return couchmessage.Response{Code: "invalid-request", Error: "unsupported forget action"}
	}
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	s.originAdmission.Lock()
	defer s.originAdmission.Unlock()
	if s.origins == nil || s.originErr != nil || s.originNamespace == "" {
		return couchmessage.Response{Code: "unavailable", Error: "origin evidence unavailable; start Couch and retry"}
	}
	syncer := sessionsync.New(s.originNamespace)
	// Metadata lookup resolves aliases and path spelling; eventual removal is
	// guarded by the exact returned ID inside the registry transaction.
	registration, err := syncer.RegistrationForPath(r.Target)
	if err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	if registration == nil {
		if err := s.reconcilePendingForgetsLocked(); err != nil {
			return couchmessage.Response{Code: "refused", Error: err.Error()}
		}
		return couchmessage.Response{Code: "ok", Forget: &couchmessage.ForgetResult{}}
	}
	id := registration.ID
	if err := s.forgetSettled(id); err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	if err := s.origins.BeginForget(id); err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	result, err := syncer.ForgetExact(r.Target, id)
	if err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	if err := s.reconcilePendingForgetsLocked(); err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	return couchmessage.Response{Code: "ok", Forget: &couchmessage.ForgetResult{Changed: result.Changed}}
}

func (s *messageService) forgetSettled(id string) error {
	seen, err := s.origins.Submitted(id)
	if err != nil {
		return err
	}
	if seen {
		return fmt.Errorf("this document had remote submits; forget is not available yet, see %s", remoteForgetFollowup)
	}

	if control := s.remote.Load(); control != nil && control.supervisor != nil {
		if err := control.supervisor.FenceDocument(id); err != nil {
			return err
		}
	}
	// An operation can mark and settle between the first read and the fence.
	// Re-read after fencing; new remote work is now excluded and local submits
	// remain excluded by originAdmission until the registry removal completes.
	seen, err = s.origins.Submitted(id)
	if err != nil {
		return err
	}
	if seen {
		return fmt.Errorf("this document had remote submits; forget is not available yet, see %s", remoteForgetFollowup)
	}
	return nil
}

// Called only by an explicit local forget, never by boot reconciliation. Mere
// absence is not forget authority: only exact persisted intent IDs can retire,
// only for never-submitted IDs after active work settles and a complete registry
// read proves absence. Submitted IDs stay blocked regardless of retention age.
func (s *messageService) reconcilePendingForgetsLocked() error {
	if s.origins == nil || s.originErr != nil {
		return errors.New("origin evidence unavailable")
	}
	ids, err := s.origins.PendingForgets()
	if err != nil {
		return err
	}
	registrations, err := sessionsync.New(s.originNamespace).List()
	if err != nil {
		return err
	}
	present := make(map[string]bool, len(registrations))
	for _, registration := range registrations {
		present[registration.ID] = true
	}
	for _, id := range ids {
		if present[id] {
			continue
		}
		if err := s.forgetSettled(id); err != nil {
			return err
		}
		if err := s.origins.RetireForgotten(id); err != nil {
			return err
		}
		if control := s.remote.Load(); control != nil && control.supervisor != nil {
			if err := control.supervisor.CompleteForget(id); err != nil {
				return err
			}
		}
	}
	return nil
}
