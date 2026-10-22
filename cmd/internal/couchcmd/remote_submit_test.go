package couchcmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/remotesession"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
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
