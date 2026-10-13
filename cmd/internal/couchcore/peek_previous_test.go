package couchcore

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
)

func TestPeekPreviousNeverReadsCurrentWhenUnavailable(t *testing.T) {
	t.Parallel()
	env, record := peekEnv(t)
	currentReads := 0
	env.Couch.SlotTerminal = func(ThreadAddress, string, int) ([]string, error) {
		currentReads++
		return []string{"wrong incarnation"}, nil
	}
	value, err := dispatchTestOperation(env.Couch, "peek", map[string]string{
		"repo-scope": record.Address.RepoScope, "ref": string(record.Address.Tag), "previous": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	r, ok := value.(PeekResult)
	if !ok || len(r.Unavailable) == 0 || len(r.Lines) != 0 || currentReads != 0 {
		t.Fatalf("previous selection fell through: %#v, current reads %d", value, currentReads)
	}
}

func TestPeekPreviousFullUsesExactSavedSource(t *testing.T) {
	t.Parallel()
	env, record := peekEnv(t)
	var err error
	record, err = env.Couch.Threads.updateExistingThread(record.Address, record.Revision, func(r *ThreadRecord) error {
		r.VerifiedPark.Scrollback = &pairlifecycle.PreservedScrollback{Agent: "claude", Token: "exact-old-token", Events: true}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	previous := previousSessionEvidence(record, record)
	if _, err := env.Couch.Threads.RecordPreviousSession(record.Address, record.Revision, *previous); err != nil {
		t.Fatal(err)
	}
	env.Couch.SwitchContext = switchContextFunc(func(context.Context, ThreadRecord) (orientation.OrientationContext, error) {
		t.Fatal("previous selection queried current native owner")
		return orientation.OrientationContext{}, nil
	})
	env.Couch.SlotTail = func(context.Context, ThreadAddress, int) (TerminalTail, error) {
		t.Fatal("full previous read used current live tail")
		return TerminalTail{}, nil
	}
	want := make([]string, 3001)
	for i := range want {
		want[i] = fmt.Sprintf("line %d", i)
	}
	env.Couch.SlotCapture = func(address ThreadAddress, agent, token string, limit int) (TerminalRecording, error) {
		if address != record.Address || agent != "claude" || token != "exact-old-token" || limit != 0 {
			t.Fatalf("wrong source: %v %s %s %d", address, agent, token, limit)
		}
		return TerminalRecording{Lines: want, Raw: "/fixture/old.raw", Events: "/fixture/old.events.jsonl"}, nil
	}
	value, err := dispatchTestOperation(env.Couch, "peek", map[string]string{"repo-scope": record.Address.RepoScope, "ref": string(record.Address.Tag), "previous": "true", "all": "true"})
	if err != nil {
		t.Fatal(err)
	}
	r := value.(PeekResult)
	if !r.Complete || !r.Previous || r.Source != "previous-recording" || len(r.Lines) != len(want) || r.Lines[0] != want[0] || r.NativeSession != "" || len(r.Transcripts) != 0 || r.Raw != "/fixture/old.raw" {
		t.Fatalf("wrong full result: %+v", r)
	}
}

func TestPeekFullRefusesConflictingSelection(t *testing.T) {
	t.Parallel()
	env, record := peekEnv(t)
	for _, args := range []map[string]string{
		{"repo-scope": record.Address.RepoScope, "ref": string(record.Address.Tag), "all": "true", "lines": "40"},
		{"repo-scope": record.Address.RepoScope, "ref": "pair:1:2", "all": "true"},
	} {
		if _, err := dispatchTestOperation(env.Couch, "peek", args); err == nil {
			t.Fatal("ambiguous full read admitted")
		}
	}
}

func TestPeekPreviousRejectsNativeTranscripts(t *testing.T) {
	t.Parallel()
	env, record := peekEnv(t)
	_, err := env.Couch.PeekThreadSelection(context.Background(), "pair:2", record.Address, PeekSelection{Previous: true, Transcripts: true})
	if err == nil || !strings.Contains(err.Error(), "capture native coordinates before switching") {
		t.Fatalf("expected native coordinate guidance, got %v", err)
	}
}

func TestPeekCurrentFullUsesCurrentCapture(t *testing.T) {
	t.Parallel()
	env, record := peekEnv(t)
	env.Couch.SlotTail = func(context.Context, ThreadAddress, int) (TerminalTail, error) {
		t.Fatal("full read used live tail")
		return TerminalTail{}, nil
	}
	env.Couch.SlotCapture = func(address ThreadAddress, agent, token string, limit int) (TerminalRecording, error) {
		if address != record.Address || agent != RecordAgent(record) || token != "" || limit != 0 {
			t.Fatalf("wrong current capture: %v %s %s %d", address, agent, token, limit)
		}
		return TerminalRecording{Lines: []string{"first", "last"}, Raw: "/fixture/current.raw", Events: "/fixture/current.events.jsonl"}, nil
	}
	result, err := env.Couch.PeekThreadSelection(context.Background(), "pair:2", record.Address, PeekSelection{All: true})
	if err != nil || !result.Complete || result.Previous || result.Source != "recording" || len(result.Lines) != 2 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestPeekPreviousMissingCaptureDoesNotReadCurrent(t *testing.T) {
	t.Parallel()
	env, record := peekEnv(t)
	previous := previousSessionEvidence(record, record)
	if previous == nil {
		t.Fatal("fixture missing verified park")
	}
	if _, err := env.Couch.Threads.RecordPreviousSession(record.Address, record.Revision, *previous); err != nil {
		t.Fatal(err)
	}
	env.Couch.SlotCapture = func(ThreadAddress, string, string, int) (TerminalRecording, error) {
		t.Fatal("missing old capture read current recording")
		return TerminalRecording{}, nil
	}
	result, err := env.Couch.PeekThreadSelection(context.Background(), "pair:2", record.Address, PeekSelection{Previous: true, All: true})
	if err != nil || result.Complete || len(result.Lines) != 0 || len(result.Unavailable) == 0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
