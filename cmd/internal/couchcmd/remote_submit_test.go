package couchcmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/remotesession"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

type provenanceServiceEndpoint struct{ serviceEndpointFake }

func (provenanceServiceEndpoint) SupportsRemoteProvenance(context.Context) error { return nil }

func TestLocalRemoteEditedSubmitPinsForgetAndKeepsLocalHeader(t *testing.T) {
	for _, scenario := range []string{"current", "restarted", "renamed"} {
		t.Run(scenario, func(t *testing.T) {
			namespace := t.TempDir()
			file := filepath.Join(t.TempDir(), "session.md")
			if err := os.WriteFile(file, []byte("---\nowner: pair:1\n---\nhello\n"), 0600); err != nil {
				t.Fatal(err)
			}
			registered, err := sessionsync.New(namespace).Register(file, "pair:1")
			if err != nil {
				t.Fatal(err)
			}
			originPath := filepath.Join(t.TempDir(), "origins")
			origins, err := remotesession.OpenOriginStore(originPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := origins.MarkRemote(registered.Registration.ID); err != nil {
				t.Fatal(err)
			}
			if scenario != "current" {
				origins, err = remotesession.OpenOriginStore(originPath)
				if err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "renamed" {
				moved := filepath.Join(filepath.Dir(file), "renamed.md")
				if err := os.Rename(file, moved); err != nil {
					t.Fatal(err)
				}
				changed, err := sessionsync.New(namespace).Rename(file, moved)
				if err != nil {
					t.Fatal(err)
				}
				if changed.Registration.ID != registered.Registration.ID {
					t.Fatal("rename replaced registration identity")
				}
				file = moved
			}
			broker := couchmessage.NewBroker(context.Background(), time.Now, func(context.Context, couchmessage.Binding) (bool, error) { return true, nil })
			t.Cleanup(func() { _ = broker.Close() })
			from := couchmessage.Binding{Slot: "pair:1", Repository: "/repo/.git", Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce", Agent: "codex", Version: "current", PID: 1, Start: "start"}
			target := from
			target.Slot = "pair:2"
			target.Tag = "target"
			target.Session = "target-session"
			for _, b := range []couchmessage.Binding{from, target} {
				if err := broker.Register(b, provenanceServiceEndpoint{}); err != nil {
					t.Fatal(err)
				}
			}
			s := &messageService{broker: broker, origins: origins, originNamespace: namespace}
			request := couchmessage.Request{Op: "send", ID: "marked-local", Target: target.Slot, Body: "submitted: " + file}
			response, handled := s.admitLocalSubmit(t.Context(), from, request)
			if !handled || response.Code != "accepted" || response.Receipt == nil {
				t.Fatalf("response=%+v handled=%v", response, handled)
			}
			m := response.Receipt.Message
			if m.Origin != couchmessage.OriginPeer || !m.RemoteEdited || m.DocumentID != registered.Registration.ID || m.RemoteSource != (couchmessage.RemoteSource{}) {
				t.Fatalf("local provenance=%+v", m)
			}
			seen, err := origins.Submitted(registered.Registration.ID)
			if err != nil || !seen {
				t.Fatal("forget not pinned", seen, err)
			}
		})
	}
}
func TestLocalRemoteSubmitPendingForgetRefusesBeforeBroker(t *testing.T) {
	namespace := t.TempDir()
	file := filepath.Join(t.TempDir(), "session.md")
	if err := os.WriteFile(file, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	reg, err := sessionsync.New(namespace).Register(file, "pair:1")
	if err != nil {
		t.Fatal(err)
	}
	origins, err := remotesession.OpenOriginStore(filepath.Join(t.TempDir(), "origins"))
	if err != nil {
		t.Fatal(err)
	}
	if err := origins.BeginForget(reg.Registration.ID); err != nil {
		t.Fatal(err)
	}
	s := &messageService{origins: origins, originNamespace: namespace}
	response, handled := s.admitLocalSubmit(t.Context(), couchmessage.Binding{Slot: "pair:1"}, couchmessage.Request{Body: "submitted: " + file})
	if !handled || response.Code != "refused" {
		t.Fatalf("response=%+v handled=%v", response, handled)
	}
}

type crossSlotProvenanceEndpoint struct {
	provenanceServiceEndpoint
	messages     chan couchmessage.Message
	capabilities atomic.Int32
}

func (e *crossSlotProvenanceEndpoint) SupportsRemoteProvenance(context.Context) error {
	e.capabilities.Add(1)
	return nil
}
func (e *crossSlotProvenanceEndpoint) Deliver(ctx context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	select {
	case e.messages <- m:
		return couchmessage.Receipt{Status: couchmessage.Submitted}, nil
	case <-ctx.Done():
		return couchmessage.Receipt{}, ctx.Err()
	}
}
func TestLocalSubmitDistinctCallerAndDocumentOwnerProductionService(t *testing.T) {
	for _, marked := range []bool{false, true} {
		t.Run(fmt.Sprint(marked), func(t *testing.T) {
			endpoint := &crossSlotProvenanceEndpoint{messages: make(chan couchmessage.Message, 1)}
			rig := newServiceRig(t, func(a messageAuthority) messageAuthority {
				a.endpoint = func(couchmessage.Binding) couchmessage.DeliveryEndpoint { return endpoint }
				return a
			})
			from, to := rig.connect(1), rig.connect(2)
			path := filepath.Join(t.TempDir(), "recipient-owned.md")
			if err := os.WriteFile(path, []byte("## Work\n\nowned by recipient\n"), 0600); err != nil {
				t.Fatal(err)
			}
			registration, err := sessionsync.New(rig.s.originNamespace).Register(path, to.Slot)
			if err != nil {
				t.Fatal(err)
			}
			if marked {
				if err := rig.s.origins.MarkRemote(registration.Registration.ID); err != nil {
					t.Fatal(err)
				}
			}
			request := callerRequest(from, "send")
			request.ID = "cross-slot"
			request.Target = to.Slot
			request.Body = sessionsync.SubmitPrefix + path
			response := rig.s.handle(t.Context(), request)
			if response.Code != "accepted" || response.Receipt == nil {
				t.Fatalf("valid cross-slot submit refused: %+v", response)
			}
			select {
			case message := <-endpoint.messages:
				if message.From != from || message.To != to || message.RemoteEdited != marked || message.Origin != couchmessage.OriginPeer || message.RemoteSource != (couchmessage.RemoteSource{}) {
					t.Fatalf("cross-slot provenance=%+v", message)
				}
				if marked && message.DocumentID != registration.Registration.ID {
					t.Fatal("registered identity lost")
				}
				if !marked && (message.DocumentID != "" || endpoint.capabilities.Load() != 0) {
					t.Fatal("unmarked peer wire changed")
				}
			case <-testwait.After(t):
				t.Fatal("cross-slot submit not delivered")
			}
		})
	}
}
