package testcouch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/cmd/internal/wrapcmd"
)

func TestStartupEvidenceRetainsPrivateSourcesBeforeTeardown(t *testing.T) {
	p, err := allocateRoots(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	f := &Fleet{Roots: p, Options: Options{AgentKind: "claude"}}
	const scope, tag = "0123456789abcdef", "actual-thread"
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: filepath.Join(p.Mutable["data"], "pair"), RepoScope: scope, Tag: tag})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.ScopeDir(), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Pane("claude"), []byte(`{"pane_id":"3","cwd":"private"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.AgentReady("codex"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	// Consume the actual wrapper producer's emitted profile, not a fabricated
	// schema copy. Missing argv exercises an ordinary fatal startup path.
	testpair.ApplyEnvironment(t, p.Mutable["home"], "PAIR_WRAP_LOG="+startupLogPath(p), "PAIR_SCOPE_KEY="+scope, "PAIR_TAG="+tag, "PAIR_SESSION_NAME=actual-session", "PAIR_LAUNCH_NONCE=actual-nonce", "SECRET_TOKEN=SECRET_DO_NOT_COPY")
	if code := wrapcmd.Run(nil, strings.NewReader(""), io.Discard, io.Discard); code != 1 {
		t.Fatalf("startup error exit=%d", code)
	}
	log, err := os.OpenFile(startupLogPath(p), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(log, "[01:02:03] AGENT-READY-write-fail: \"writer refused\"\n[01:02:03] STDIN: \"SECRET_DO_NOT_COPY\"\n"); err != nil {
		t.Fatal(err)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	ns, err := couchcore.ResolveCouchNamespace(p.Mutable["store"], "")
	if err != nil {
		t.Fatal(err)
	}
	if err := couchcore.NewLaunchFailureStore(ns).Save(t.Context(), couchcore.LaunchFailure{Nonce: "operation-real", Address: couchcore.ThreadAddress{RepoScope: scope, Tag: tag}, Agent: "claude", FailedAt: time.Now(), FinalError: "failed"}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	snapshot := f.captureStartupEvidence(t.Context(), func(ctx context.Context, session, pane string) (string, error) {
		calls++
		if session != "actual-session" || pane != "3" {
			t.Fatalf("foreign query %q %q", session, pane)
		}
		return "distinct pane stderr", nil
	})
	for _, want := range []string{"writer refused", "actual-session", "actual-nonce", "distinct pane stderr", filepath.Base(paths.AgentReady("codex"))} {
		if !strings.Contains(snapshot, want) {
			t.Errorf("missing %q: %s", want, snapshot)
		}
	}
	if calls != 1 || strings.Contains(snapshot, "SECRET_DO_NOT_COPY") {
		t.Fatalf("unsafe capture calls=%d snapshot=%s", calls, snapshot)
	}
	readyRaw, err := readiness.Encode(readiness.ReadyRecord{Tag: tag, Agent: "claude", Session: "other-session", Nonce: "other-nonce", PID: os.Getpid()})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.AgentReady("claude"), []byte(readyRaw), 0600); err != nil {
		t.Fatal(err)
	}
	mismatch := f.captureStartupEvidence(t.Context(), func(context.Context, string, string) (string, error) {
		return "", errors.New("owned pane removed by rollback")
	})
	for _, want := range []string{"matches_emitted=false", "other-session", "pane dump unavailable: native query failed"} {
		if !strings.Contains(mismatch, want) {
			t.Errorf("missing mismatch/query refusal %q: %s", want, mismatch)
		}
	}
	cleanup, stopCleanup := context.WithTimeout(t.Context(), time.Second)
	held := f.captureStartupEvidence(cleanup, func(query context.Context, _ string, _ string) (string, error) {
		parentDeadline, _ := cleanup.Deadline()
		queryDeadline, bounded := query.Deadline()
		if !bounded || !queryDeadline.Before(parentDeadline) {
			t.Fatal("diagnostics did not reserve cleanup context")
		}
		<-query.Done()
		return "", query.Err()
	})
	if cleanup.Err() != nil || !strings.Contains(held, "context deadline exceeded") {
		t.Fatalf("held diagnostics consumed cleanup budget: parent=%v evidence=%s", cleanup.Err(), held)
	}
	stopCleanup()
	if err := couchcore.NewLaunchFailureStore(ns).Save(t.Context(), couchcore.LaunchFailure{Nonce: "operation-new", Address: couchcore.ThreadAddress{RepoScope: scope, Tag: tag}, Agent: "claude", LaunchNonce: "different-recorded-attempt", FailedAt: time.Now().Add(time.Second), FinalError: "failed"}); err != nil {
		t.Fatal(err)
	}
	wrongAttempt := f.captureStartupEvidence(t.Context(), func(context.Context, string, string) (string, error) {
		t.Fatal("different attempt authorized attributed pane query")
		return "", nil
	})
	if !strings.Contains(wrongAttempt, "emitted launch nonce mismatch") {
		t.Fatal(wrongAttempt)
	}
	if err := os.Remove(couchcore.NewLaunchFailureStore(ns).Path()); err != nil {
		t.Fatal(err)
	}
	if err := couchcore.NewLaunchFailureStore(ns).Save(t.Context(), couchcore.LaunchFailure{Nonce: "operation-incomplete", Target: "fixture:3", FailedAt: time.Now().Add(time.Second), FinalError: "missing address"}); err != nil {
		t.Fatal(err)
	}
	var fallbackCalls int
	fallback := f.captureStartupEvidence(t.Context(), func(context.Context, string, string) (string, error) {
		fallbackCalls++
		return "fallback actual pane", nil
	})
	if fallbackCalls != 1 || !strings.Contains(fallback, "fallback actual pane") || !strings.Contains(fallback, "incomplete failure address") {
		t.Fatalf("incomplete history hid emitted identity: calls=%d snapshot=%s", fallbackCalls, fallback)
	}
	// A later teardown must not change the already-owned evidence string.
	if err := os.Remove(paths.Pane("claude")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(snapshot, "distinct pane stderr") {
		t.Fatal("snapshot borrowed live source")
	}
	missing := f.captureStartupEvidence(t.Context(), func(context.Context, string, string) (string, error) {
		t.Fatal("missing pane authorized a query")
		return "", nil
	})
	if !strings.Contains(missing, "unavailable") {
		t.Fatal(missing)
	}
	// A sibling escaping this root is listed by name but never read or queried.
	if err := os.Symlink(filepath.Join(t.TempDir(), "outside"), paths.Pane("claude")); err != nil {
		t.Fatal(err)
	}
	unsafe := f.captureStartupEvidence(t.Context(), func(context.Context, string, string) (string, error) {
		t.Fatal("escaped pane authorized query")
		return "", nil
	})
	if !strings.Contains(unsafe, "unavailable") {
		t.Fatal(unsafe)
	}
}

func TestStartupDiagnosticBudgetReservesCleanupTime(t *testing.T) {
	now := time.Unix(100, 0)
	for _, remaining := range []time.Duration{time.Second, 16 * time.Second, time.Minute} {
		got, err := startupDiagnosticDeadline(now, now.Add(remaining))
		if err != nil || !got.Equal(now.Add(remaining/4)) {
			t.Fatalf("budget remaining=%v deadline=%v error=%v", remaining, got, err)
		}
	}
	if _, err := startupDiagnosticDeadline(now, now); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expired budget=%v", err)
	}
}

func TestStartupEvidenceBoundsAndCountsSelection(t *testing.T) {
	for _, n := range []int{8, 16} {
		var b strings.Builder
		for i := 0; i < n; i++ {
			b.WriteString("[00:00:00] AGENT-READY-write-fail: \"bounded startup refusal\"\n")
		}
		_, _, visited := selectStartupRecords([]byte(b.String()))
		if visited != n {
			t.Fatalf("rows=%d visits=%d", n, visited)
		}
		dir := t.TempDir()
		for i := 0; i < n; i++ {
			if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("agent-ready-thread-%d.json", i)), nil, 0600); err != nil {
				t.Fatal(err)
			}
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		names, cut, count := selectReadyNames(entries)
		if cut || count != n || len(names) != n {
			t.Fatalf("siblings n=%d visits=%d retained=%d truncated=%t", n, count, len(names), cut)
		}
		identities := map[string]startupIdentity{}
		for i := 0; i < n; i++ {
			id := startupIdentity{Scope: "0123456789abcdef", Tag: fmt.Sprint(i), order: i}
			identities[id.Tag] = id
		}
		sources, cut, visits := selectStartupSources(nil, identities, "claude")
		if len(sources) != 4 || !cut || visits != n || sources[0].tag != fmt.Sprint(n-1) {
			t.Fatalf("source selection n=%d visits=%d sources=%+v cut=%t", n, visits, sources, cut)
		}
	}
	p, err := allocateRoots(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(startupLogPath(p), []byte(strings.Repeat("[01:02:03] AGENT-READY-write-fail: \"recognized bounded error\"\n", 2000)), 0600); err != nil {
		t.Fatal(err)
	}
	f := &Fleet{Roots: p}
	snapshot := f.captureStartupEvidence(t.Context(), func(context.Context, string, string) (string, error) { return "", errors.New("no pane") })
	if !strings.Contains(snapshot, "truncated") || len(snapshot) > startupReadBytes*8 {
		t.Fatalf("unbounded/unmarked snapshot length=%d %s", len(snapshot), snapshot)
	}
	// Both inode and directory aliases must fail without reading foreign bytes.
	foreign := t.TempDir()
	outside := filepath.Join(foreign, "secret")
	if err := os.WriteFile(outside, []byte("FOREIGN_SECRET"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(startupLogPath(p)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, startupLogPath(p)); err != nil {
		t.Fatal(err)
	}
	unsafe := f.captureStartupLog()
	if strings.Contains(unsafe, "FOREIGN_SECRET") || !strings.Contains(unsafe, "read_error=") {
		t.Fatal(unsafe)
	}
	r, err := openEvidenceRoot(p.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer r.root.Close()
	alias := filepath.Join(p.Root, "outside-dir")
	if err := os.Symlink(foreign, alias); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.read(filepath.Join(alias, "secret"), 4096, false); err == nil {
		t.Fatal("escaped intermediate directory admitted")
	}
	entries := make([]os.DirEntry, 0, 140)
	dir := t.TempDir()
	for i := 0; i < 140; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("agent-ready-%03d.json", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	entries, err = os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names, cut, visited := selectReadyNames(entries)
	if !cut || len(names) != startupSiblingNames || visited != startupDirectoryEntries {
		t.Fatalf("census bound names=%d visited=%d truncated=%t", len(names), visited, cut)
	}
}

func TestStartupIdentityProjectionDropsUnknownFields(t *testing.T) {
	row := `{"actualPID":42,"PAIR_SCOPE_KEY":"scope","PAIR_TAG":"tag","PAIR_SESSION_NAME":"session","PAIR_LAUNCH_NONCE":"nonce","SECRET_TOKEN":"FOREIGN_SECRET"}`
	selected, identities, visited := selectStartupRecords([]byte("[00:00:00] STARTUP-identity: " + strconv.Quote(row) + "\n[00:00:00] STDIN: \"FOREIGN_SECRET\"\n"))
	if visited != 2 || len(identities) != 1 || strings.Contains(selected, "SECRET") || !strings.Contains(selected, "session") {
		t.Fatalf("unsafe identity projection: %s %+v", selected, identities)
	}
}

func TestStartupPaneQueryExecutableBoundary(t *testing.T) {
	p, err := allocateRoots(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(p.Root, "screen-state")
	invocations := filepath.Join(p.Root, "query-invocations")
	failure := filepath.Join(p.Root, "query-fail")
	if err := os.WriteFile(state, []byte("distinct native pane stderr\n"), 0600); err != nil {
		t.Fatal(err)
	}
	program := filepath.Join(p.Root, "zellij-query")
	// A stateful executable boundary, not a callback: it records argv and the
	// environment the real adapter supplied, then answers from its owned state.
	script := `#!/bin/sh
printf '%s\n' "$@" >> "$INVOCATIONS"
printf 'sockets=%s\nsession=%s\nsecret=%s\n' "$ZELLIJ_SOCKET_DIR" "$ZELLIJ_SESSION_NAME" "$AMBIENT_SECRET" >> "$INVOCATIONS"
if [ -f "$FAILURE" ]; then printf 'owned query refused' >&2; exit 7; fi
while IFS= read -r line; do printf '%s\n' "$line"; done < "$STATE"
`
	if err := os.WriteFile(program, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZELLIJ_SOCKET_DIR", "/ambient/foreign-sockets")
	t.Setenv("ZELLIJ_SESSION_NAME", "ambient-session")
	t.Setenv("AMBIENT_SECRET", "must-not-inherit")
	f := &Fleet{Roots: p, Binaries: Binaries{Zellij: program}, Environment: []string{"STATE=" + state, "INVOCATIONS=" + invocations, "FAILURE=" + failure}}
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	text, err := f.dumpStartupPane(ctx, "actual-session", "3")
	if err != nil || text != "distinct native pane stderr\n" {
		t.Fatalf("query output=%q error=%v", text, err)
	}
	raw, err := os.ReadFile(invocations)
	if err != nil {
		t.Fatal(err)
	}
	want := "-s\nactual-session\naction\ndump-screen\n--pane-id\nterminal_3\nsockets=" + p.Mutable["sockets"] + "\nsession=actual-session\nsecret=\n"
	if string(raw) != want {
		t.Fatalf("query boundary=%q want=%q", raw, want)
	}
	if err := os.WriteFile(failure, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := f.dumpStartupPane(ctx, "actual-session", "3"); err == nil || !strings.Contains(err.Error(), "owned query refused") {
		t.Fatalf("lost native refusal: %v", err)
	}
	if err := os.Remove(failure); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state, []byte(strings.Repeat("x", startupSourceBytes+100)+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	large, largeErr := f.dumpStartupPane(ctx, "actual-session", "3")
	if len(large) != startupSourceBytes || largeErr == nil || !strings.Contains(largeErr.Error(), "stdout_truncated=true") {
		t.Fatalf("native output bound bytes=%d error=%v", len(large), largeErr)
	}
	// An escaped socket directory refuses before another executable invocation.
	before, _ := os.ReadFile(invocations)
	foreign := t.TempDir()
	f.Roots.Mutable["sockets"] = filepath.Join(p.Root, "aliased-sockets")
	if err := os.Symlink(foreign, f.Roots.Mutable["sockets"]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.dumpStartupPane(ctx, "actual-session", "3"); err == nil {
		t.Fatal("escaped socket root admitted")
	}
	after, _ := os.ReadFile(invocations)
	if string(before) != string(after) {
		t.Fatal("unsafe sockets reached command")
	}
}
