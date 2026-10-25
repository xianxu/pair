package testcouch

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

func TestCallerEvidencePrioritizesFrozenCallerWithinFourSources(t *testing.T) {
	p, err := allocateRoots(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	f := &Fleet{Roots: p, Options: Options{AgentKind: "claude"}}
	caller := couchmessage.Binding{Slot: "fixture:0", Repository: "repo", Scope: "0123456789abcdef", Tag: "caller", Session: "caller-session", Nonce: "caller-nonce", Agent: "claude", Version: "1", PID: os.Getpid(), Start: "frozen-generation"}
	var log strings.Builder
	for i := 0; i < 6; i++ {
		tag, session, nonce := fmt.Sprint(i), fmt.Sprintf("worker-%d", i), fmt.Sprintf("nonce-%d", i)
		if i == 0 {
			tag, session, nonce = caller.Tag, caller.Session, caller.Nonce
		}
		raw, _ := json.Marshal(startupIdentity{PID: os.Getpid(), Scope: caller.Scope, Tag: tag, Session: session, Nonce: nonce})
		fmt.Fprintf(&log, "[00:00:00] STARTUP-identity: %s\n", strconv.Quote(string(raw)))
		paths, e := artifactpath.Resolve(artifactpath.Address{DataDir: filepath.Join(p.Mutable["data"], "pair"), RepoScope: caller.Scope, Tag: tag})
		if e != nil {
			t.Fatal(e)
		}
		if e = os.MkdirAll(paths.ScopeDir(), 0700); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(paths.Pane("claude"), []byte(`{"pane_id":"3"}`), 0600); e != nil {
			t.Fatal(e)
		}
	}
	if err = os.WriteFile(startupLogPath(p), []byte(log.String()), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	snapshot := f.captureCallerEvidence(ctx, caller, func(q context.Context, session, pane string) (string, error) {
		calls++
		if calls == 1 && session != caller.Session {
			t.Fatalf("first pane %q", session)
		}
		parent, _ := ctx.Deadline()
		deadline, bounded := q.Deadline()
		if !bounded || !deadline.Before(parent) {
			t.Fatal("no shared cleanup reserve")
		}
		return strings.Repeat("x", startupSourceBytes+1), nil
	}, func(context.Context) (string, error) { return "later listing", nil })
	for _, want := range []string{`tag="caller"`, "startup_sources=4 truncated=true", "later_native_listing", "sha256=", "truncated=true", "frozen-generation"} {
		if !strings.Contains(snapshot, want) {
			t.Errorf("missing %q: %s", want, snapshot)
		}
	}
	if calls != 4 {
		t.Fatalf("pane calls=%d", calls)
	}
	short, stopShort := context.WithTimeout(t.Context(), 200*time.Millisecond)
	heldCalls := 0
	held := f.captureCallerEvidence(short, caller, func(q context.Context, _, _ string) (string, error) {
		heldCalls++
		<-q.Done()
		return "", q.Err()
	}, func(context.Context) (string, error) {
		t.Fatal("listing got a fresh budget after panes")
		return "", nil
	})
	if heldCalls != 1 || short.Err() != nil || !strings.Contains(held, "shared diagnostic budget") {
		t.Fatalf("calls=%d parent=%v evidence=%s", heldCalls, short.Err(), held)
	}
	stopShort()
	caller.Nonce = "different-attempt"
	mismatch := f.captureCallerEvidence(ctx, caller, func(_ context.Context, session, _ string) (string, error) {
		if session == caller.Session {
			t.Fatal("mismatched caller queried")
		}
		return "", nil
	}, nil)
	if !strings.Contains(mismatch, "caller binding mismatch") {
		t.Fatal(mismatch)
	}
}

func TestCallerEvidenceRefusesUnboundedOrInvalidBinding(t *testing.T) {
	f := &Fleet{}
	for _, ctx := range []context.Context{t.Context(), context.Background()} {
		got := f.CaptureCallerEvidence(ctx, couchmessage.Binding{})
		if !strings.Contains(got, "unavailable") {
			t.Fatal(got)
		}
	}
	valid := couchmessage.Binding{Slot: "fixture:0", Repository: "repo", Scope: "0123456789abcdef", Tag: "caller", Session: "caller-session", Nonce: "caller-nonce", Agent: "claude", Version: "1", PID: os.Getpid(), Start: "generation"}
	if got := f.CaptureCallerEvidence(context.Background(), valid); !strings.Contains(got, "bounded context required") {
		t.Fatal(got)
	}
}

func TestCallerEvidenceReportsSourcesOmittedByCallerPriority(t *testing.T) {
	var history []couchcore.LaunchFailure
	for i := range 4 {
		history = append(history, couchcore.LaunchFailure{Address: couchcore.ThreadAddress{RepoScope: "scope", Tag: couchcore.ThreadTag(fmt.Sprint(i))}})
	}
	caller := couchmessage.Binding{Scope: "scope", Tag: "caller", Agent: "claude"}
	sources, cut, _ := selectStartupSources(history, nil, "claude", caller)
	if len(sources) != 4 || sources[0].tag != caller.Tag || !cut {
		t.Fatalf("omitted source hidden: sources=%+v truncated=%t", sources, cut)
	}
}

func TestCallerEvidenceMissingIdentityAndSharedCancellation(t *testing.T) {
	p, err := allocateRoots(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	f := &Fleet{Roots: p, Options: Options{AgentKind: "claude"}}
	caller := couchmessage.Binding{Slot: "fixture:0", Repository: "repo", Scope: "0123456789abcdef", Tag: "caller", Session: "caller-session", Nonce: "caller-nonce", Agent: "claude", Version: "1", PID: os.Getpid(), Start: "generation"}
	for _, raw := range []string{"", `[00:00:00] STARTUP-identity: "malformed"` + "\n"} {
		if err = os.WriteFile(startupLogPath(p), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		calls := 0
		got := f.captureCallerEvidence(ctx, caller, func(context.Context, string, string) (string, error) {
			t.Fatal("missing identity queried")
			return "", nil
		}, func(q context.Context) (string, error) {
			calls++
			return "listing refused", fmt.Errorf("native refusal")
		})
		cancel()
		if calls != 1 || !strings.Contains(got, "caller binding mismatch") || !strings.Contains(got, "native refusal") {
			t.Fatal(got)
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	cancel()
	got := f.captureCallerEvidence(ctx, caller, nil, func(context.Context) (string, error) { t.Fatal("canceled listing queried"); return "", nil })
	if !strings.Contains(got, "context canceled") {
		t.Fatal(got)
	}
}

func TestCallerEvidenceListingUsesPrivateSocketBoundary(t *testing.T) {
	p, err := allocateRoots(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	invocations := filepath.Join(p.Root, "invocations")
	program := filepath.Join(p.Root, "query")
	script := `#!/bin/sh
printf '%s\n' "$@" > "$INVOCATIONS"
printf 'socket=%s\nsession=%s\nsecret=%s\n' "$ZELLIJ_SOCKET_DIR" "$ZELLIJ_SESSION_NAME" "$AMBIENT_SECRET" >> "$INVOCATIONS"
` + zellijtest.ShellCase(t, "caller-evidence-private-listing") + "\n"
	if err = os.WriteFile(program, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	f := &Fleet{Roots: p, Binaries: Binaries{Zellij: program}, Environment: []string{"INVOCATIONS=" + invocations}}
	t.Setenv("AMBIENT_SECRET", "FOREIGN_SECRET")
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	got, err := f.runStartupQuery(ctx, "", "list-sessions", "--no-formatting")
	if err != nil || got != zellijtest.Case(t, "caller-evidence-private-listing").Stdout {
		t.Fatalf("%q %v", got, err)
	}
	raw, err := os.ReadFile(invocations)
	if err != nil {
		t.Fatal(err)
	}
	want := "list-sessions\n--no-formatting\nsocket=" + p.Mutable["sockets"] + "\nsession=\nsecret=\n"
	if string(raw) != want {
		t.Fatalf("query=%q want=%q", raw, want)
	}
	f.Roots.Mutable["sockets"] = filepath.Join(p.Root, "alias")
	if err = os.Symlink(t.TempDir(), f.Roots.Mutable["sockets"]); err != nil {
		t.Fatal(err)
	}
	if _, err = f.runStartupQuery(ctx, "", "list-sessions", "--no-formatting"); err == nil {
		t.Fatal("aliased socket admitted")
	}
	after, _ := os.ReadFile(invocations)
	if string(after) != string(raw) {
		t.Fatal("unsafe query executed")
	}
}
