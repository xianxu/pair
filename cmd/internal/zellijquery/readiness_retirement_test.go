package zellijquery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func putRetirementMarker(t *testing.T, env []string, identity ReadyIdentity, raw []byte) string {
	t.Helper()
	store, err := newReadinessStore(env)
	if err != nil {
		t.Fatal(err)
	}
	path, err := store.paths.ZellijReadyMarker(identity.Namespace, identity.Session)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if raw == nil {
		raw, err = EncodeReadyMarker(ReadyMarker{Identity: identity})
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRetireReadyAfterRemovesOnlyConfirmedExactGeneration(t *testing.T) {
	env, identity := privateReadinessEnv(t)
	path := putRetirementMarker(t, env, identity, nil)
	calls := 0
	if err := RetireReadyAfter(t.Context(), env, identity.Session, func(context.Context) error { calls++; return nil }); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("retirement calls=%d", calls)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("marker remains: %v", err)
	}
	// Repeating established retirement with an absent marker remains harmless.
	if err := RetireReadyAfter(t.Context(), env, identity.Session, func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestRetireReadyAfterPreservesUncertainAndSuccessorMarkers(t *testing.T) {
	for _, kind := range []string{"malformed", "wrong-namespace", "callback-failure", "successor", "malformed-successor", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			env, identity := privateReadinessEnv(t)
			markerIdentity := identity
			var raw []byte
			if kind == "malformed" {
				raw = []byte(`{"identity":`)
			}
			if kind == "wrong-namespace" {
				markerIdentity.Namespace += "-foreign"
				raw, _ = EncodeReadyMarker(ReadyMarker{Identity: markerIdentity})
			}
			path := putRetirementMarker(t, env, identity, raw)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			sentinel := errors.New("retirement not confirmed")
			calls := 0
			err = RetireReadyAfter(ctx, env, identity.Session, func(context.Context) error {
				calls++
				switch kind {
				case "callback-failure":
					return sentinel
				case "successor":
					next := identity
					next.Nonce += "-next"
					putRetirementMarker(t, env, next, nil)
				case "malformed-successor":
					putRetirementMarker(t, env, identity, []byte(`{"identity":`))
				case "cancelled":
					cancel()
				}
				return nil
			})
			if calls != 1 {
				t.Fatalf("retirement calls=%d", calls)
			}
			if kind == "callback-failure" && !errors.Is(err, sentinel) {
				t.Fatalf("failure lost: %v", err)
			}
			if kind == "cancelled" && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation lost: %v", err)
			}
			if kind != "callback-failure" && kind != "cancelled" && err != nil {
				t.Fatal(err)
			}
			after, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("marker removed: %v", readErr)
			}
			if kind == "successor" {
				marker, err := DecodeReadyMarker(after)
				if err != nil || marker.Identity.Nonce != identity.Nonce+"-next" {
					t.Fatalf("successor changed: %s %v", after, err)
				}
			} else if kind == "malformed-successor" {
				if string(after) != `{"identity":` {
					t.Fatalf("uncertain successor changed: %q", after)
				}
			} else if string(after) != string(before) {
				t.Fatalf("uncertain marker changed: %q -> %q", before, after)
			}
		})
	}
}

func TestRetireReadyAfterCleanupWaitsForNamespaceExclusion(t *testing.T) {
	env, identity := privateReadinessEnv(t)
	path := putRetirementMarker(t, env, identity, nil)
	creator, err := AcquireCreator(t.Context(), env, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer creator.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer cancel()
	calls := 0
	err = RetireReadyAfter(ctx, env, identity.Session, func(context.Context) error { calls++; return nil })
	if !errors.Is(err, context.DeadlineExceeded) || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cleanup skipped exclusion: %v", err)
	}
}

func TestRetireReadyAfterFreezesEnvironmentBeforeCallback(t *testing.T) {
	env, identity := privateReadinessEnv(t)
	for _, entry := range env {
		key, value, _ := strings.Cut(entry, "=")
		t.Setenv(key, value)
	}
	path := putRetirementMarker(t, env, identity, nil)
	creator, err := AcquireCreator(t.Context(), env, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer creator.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer cancel()
	err = RetireReadyAfter(ctx, nil, identity.Session, func(context.Context) error {
		t.Setenv("ZELLIJ_SOCKET_DIR", identity.Namespace+"-foreign")
		return nil
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cleanup acquired a different namespace: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("foreign lock allowed marker removal: %v", err)
	}
}
