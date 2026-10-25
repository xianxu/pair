package launcher

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/cmd/internal/zellijquery"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

type unmarkedRecoveryWorld struct {
	statefulSessionQuiescenceOps
	endpoint net.Listener
	state    string
}

func TestSessionRetirementCollectsExactReadyMarker(t *testing.T) {
	for _, kind := range []string{"session", "owned"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("HOME", root)
			t.Setenv("ZELLIJ_SOCKET_DIR", filepath.Join(root, "native"))
			namespace, err := zellijquery.ReadinessNamespace(nil)
			if err != nil {
				t.Fatal(err)
			}
			identity := zellijquery.ReadyIdentity{Namespace: namespace, Scope: "scope", Tag: "work", Agent: "claude", Session: "pair-work", Nonce: "launch-a"}
			creator, err := zellijquery.AcquireCreator(t.Context(), nil, identity)
			if err != nil {
				t.Fatal(err)
			}
			defer creator.Close()
			if err := creator.Prepare(t.Context()); err != nil {
				t.Fatal(err)
			}
			if err := creator.PublishReady(t.Context(), identity); err != nil {
				t.Fatal(err)
			}
			cache, _ := artifactpath.ResolvePairCache(root)
			marker, _ := cache.ZellijReadyMarker(namespace, "pair-work")
			world := &statefulSessionQuiescenceOps{present: true, serverPID: 4242, serverAlive: true}
			rt := OSRuntime{sessionQuiescence: world, sessionQuiesceWait: time.Second, sessionQuiescePoll: time.Millisecond}
			if kind == "session" {
				err = rt.DeleteSessionContext(t.Context(), "pair-work")
			} else {
				err = rt.DeleteOwnedSessionContext(t.Context(), SessionOwnerObservation{State: SessionOwnerOwned, Name: "pair-work", Server: SessionServerIdentity{PID: 4242, Identity: "start-a", Session: "pair-work", Socket: "/tmp/pair-work"}})
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(marker); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("retired generation left readiness marker: %v", err)
			}
		})
	}
}

func (w *unmarkedRecoveryWorld) SessionPresent(ctx context.Context, name string) (bool, error) {
	return (osSessionQuiescenceOps{}).SessionPresent(ctx, name)
}

func (w *unmarkedRecoveryWorld) KillServer(server SessionServerIdentity) error {
	if err := w.statefulSessionQuiescenceOps.KillServer(server); err != nil {
		return err
	}
	if err := w.endpoint.Close(); err != nil {
		return err
	}
	return os.Remove(w.state)
}

func TestExplicitRecoveryOfUnmarkedEndpointKeepsExactAuthority(t *testing.T) {
	liveReply := zellijtest.ShellCase(t, "recovery-readiness.live-session")
	retiredReply := zellijtest.ShellCase(t, "recovery-readiness.retired-session")
	root := testpair.ShortRoot(t)
	t.Setenv("HOME", root)
	t.Setenv("TMPDIR", root)
	t.Setenv("PATH", root)
	t.Setenv("ZELLIJ_SOCKET_DIR", root)
	state := filepath.Join(root, "state")
	t.Setenv("PAIR_TEST_RECOVERY_STATE", state)
	if err := os.Mkdir(filepath.Join(root, "contract_version_1"), 0700); err != nil {
		t.Fatal(err)
	}
	endpoint, err := net.Listen("unix", filepath.Join(root, "contract_version_1", "pair-work"))
	if err != nil {
		t.Fatal(err)
	}
	defer endpoint.Close()
	if err := os.WriteFile(state, []byte("live"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "zellij"), []byte("#!/bin/sh\nif [ -e \"$PAIR_TEST_RECOVERY_STATE\" ]; then "+liveReply+"; else "+retiredReply+"; fi\n"), 0700); err != nil {
		t.Fatal(err)
	}
	// Legacy listing and explicit recovery both retain their existing behavior.
	_, err = (zellijquery.Source{}).ReadContext(t.Context(), "list-sessions", "--no-formatting")
	if err != nil {
		t.Fatalf("legacy automatic observation refused: %v", err)
	}
	w := &unmarkedRecoveryWorld{statefulSessionQuiescenceOps: statefulSessionQuiescenceOps{present: true, serverPID: 4242, serverAlive: true}, endpoint: endpoint, state: state}
	if err := quiesceZellijSession(t.Context(), "pair-work", w, time.Second, time.Millisecond); err != nil {
		t.Fatal("existing explicit recovery blocked by marker policy:", err)
	}
	if len(w.killed) != 1 || w.killed[0] != 4242 {
		t.Fatalf("wrong kill authority: %v", w.killed)
	}
}
