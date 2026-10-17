package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func requireRelaunchRequiredSource(t *testing.T, err error) {
	t.Helper()
	var classified *couchmessage.RelaunchRequiredError
	if !errors.As(err, &classified) {
		t.Fatalf("source error %v has no relaunch-required classification", err)
	}
	// Diagnostics can gain context without losing their source classification.
	if !errors.As(fmt.Errorf("reworded diagnostic: %w", err), &classified) {
		t.Fatal("diagnostic wrapping lost source classification")
	}
}

func TestMessageRefusalClassCurrentWrapperOwnership(t *testing.T) {
	world := newMessageWorld()
	b := world.add(1)
	world.set(b, func(s *worldSlot) { s.pidFile = b.PID + 1 })
	_, err := world.authority().current(context.Background(), b)
	requireRelaunchRequiredSource(t, err)
}

func TestMessageRefusalClassRecordedRegistration(t *testing.T) {
	for _, scenario := range []string{"nonce changed", "ready session changed", "unreadable ready record", "conversation changed"} {
		t.Run(scenario, func(t *testing.T) {
			fixture := newMessageEvidenceFixture(t, 1, 0)
			b := fixture.add(t, newMessageWorld(), 0)
			paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: fixture.DataDir, RepoScope: b.Scope, Tag: b.Tag})
			if err != nil {
				t.Fatal(err)
			}
			sessionName := func(context.Context, couchcore.ThreadAddress) (string, error) { return b.Session, nil }
			switch scenario {
			case "nonce changed":
				b.Nonce = "replacement-attempt"
			case "ready session changed":
				b.Session = "replacement-session"
			case "unreadable ready record":
				if err := os.WriteFile(paths.AgentReady(b.Agent), []byte("invalid ready record"), 0600); err != nil {
					t.Fatal(err)
				}
			case "conversation changed":
				sessionName = func(context.Context, couchcore.ThreadAddress) (string, error) { return "replacement-session", nil }
			}
			err = recordedMessageAuthority(fixture.reader, sessionName)(context.Background(), b)
			requireRelaunchRequiredSource(t, err)
		})
	}
}

func TestMessageRefusalClassUnlistedTerminal(t *testing.T) {
	rt := newRT(t, "/repo")
	record := seedDetachedThread(t, rt, "/repo")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	binding := couchmessage.Binding{Scope: record.Address.RepoScope, Tag: string(record.Address.Tag), Session: "unlisted-session"}
	_, err = messageThreadResolver(c, nil)(context.Background(), binding)
	requireRelaunchRequiredSource(t, err)
}
