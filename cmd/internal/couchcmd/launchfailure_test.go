package couchcmd

import (
	"bytes"
	"errors"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"strings"
	"testing"
	"time"
)

func TestRenderLaunchFailuresSanitizesAndSeparatesEvidence(t *testing.T) {
	before, after := 23, 143
	f := couchcore.LaunchFailure{Address: couchcore.ThreadAddress{RepoScope: "scope", Tag: "tag"}, Nonce: "attempt", FailedAt: time.Unix(1, 0), Agent: "codex", Stage: "registration", Stream: "combined stdout/stderr", BeforeExit: &before, AfterExit: &after, LaunchError: "launch failed\x1b[2J", FinalError: "source still parked", CleanupError: "cleanup denied", Output: []byte("first\x1b[31m red\nsecond\r overwrite\x1b]0;title\a")}
	var out bytes.Buffer
	renderLaunchFailures(&out, []couchcore.LaunchFailure{f}, "read error\x1b[2J")
	got := out.String()
	for _, want := range []string{"launch failed", "source still parked", "cleanup denied", "before cleanup: 23", "after cleanup: 143", "combined stdout/stderr", "first", "second", "read error"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q: %s", want, got)
		}
	}
	if strings.ContainsAny(got, "\x1b\r\a") {
		t.Fatalf("unsafe terminal controls: %q", got)
	}
}

func TestRenderLaunchFailuresBoundsTailAndDoesNotInventExit(t *testing.T) {
	var out bytes.Buffer
	renderLaunchFailures(&out, []couchcore.LaunchFailure{{Output: []byte(strings.Repeat("x", couchcore.LaunchFailureOutputLimit) + "end")}}, "")
	got := out.String()
	if len(got) > couchcore.LaunchFailureOutputLimit+1024 || !strings.Contains(got, "end") || !strings.Contains(got, "before cleanup: unobserved") || !strings.Contains(got, "after cleanup: unobserved") {
		t.Fatalf("bad bounded evidence len=%d", len(got))
	}
}

func TestRenderShowIncludesLaunchFailures(t *testing.T) {
	var out bytes.Buffer
	render(&out, couchcore.Operation{Name: "show"}, couchcore.ShowResult{LaunchFailures: []couchcore.LaunchFailure{{FinalError: "source still parked"}}})
	if !strings.Contains(out.String(), "source still parked") {
		t.Fatalf("show dropped evidence: %s", out.String())
	}
}

func TestLaunchFailurePresentationKeepsFullErrorForShow(t *testing.T) {
	rt := newRT(t, "/repo")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("fork failed; complete diagnostic\nsecond diagnostic line")
	rt.runner.FailNextStart(cause)
	_, _, err = c.Spawn(couchcore.StartArgs{Cwd: "/repo", Stack: "codex"})
	if !errors.Is(err, cause) || !strings.Contains(err.Error(), cause.Error()) {
		t.Fatalf("lost original: %v", err)
	}
	var out bytes.Buffer
	renderError(&out, err)
	line := out.String()
	receipt := slotOperationOutcome(nil, err)
	for _, got := range []string{strings.TrimSuffix(line, "\n"), receipt.Detail} {
		if strings.ContainsAny(got, "\n\r\x1b") || strings.Count(got, "inspect with") != 1 || strings.Contains(got, "second diagnostic") {
			t.Fatalf("not concise: %q", got)
		}
	}
	failures, readErr := couchcore.NewLaunchFailureStore(c.Namespace).Read()
	if readErr != nil {
		t.Fatal(readErr)
	}
	out.Reset()
	renderLaunchFailures(&out, failures, "")
	if !strings.Contains(out.String(), "second diagnostic line") {
		t.Fatalf("show lost full text: %s", out.String())
	}
}

func TestLifecycleFailureShowGenericOperation(t *testing.T) {
	var out bytes.Buffer
	renderLaunchFailures(&out, []couchcore.LaunchFailure{{Slot: "pair:8", Operation: "recover-park", Outcome: "refused", Nonce: "operation-nonce", FinalError: "still finishing"}}, "")
	got := out.String()
	for _, want := range []string{"pair:8", "recover-park", "refused", "operation-nonce", "still finishing"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q: %s", want, got)
		}
	}
}

func TestLifecycleFailureShowIncludesBoundedHolderIdentity(t *testing.T) {
	var out bytes.Buffer
	renderLaunchFailures(&out, []couchcore.LaunchFailure{{Operation: "park", Phase: "prove", Holders: []couchcore.ProcessIdentityLine{{Process: couchcore.ProcessIdentity{PID: 42, Identity: "exact-owner"}, Kind: "pair-wrap"}}}}, "")
	if got := out.String(); !strings.Contains(got, "pair-wrap") || !strings.Contains(got, "42") || !strings.Contains(got, "exact-owner") {
		t.Fatalf("holder missing: %s", got)
	}
}
