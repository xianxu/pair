package couchcore

import (
	"context"
	"fmt"
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
	previous := previousSessionEvidence(record, record, orientation.OrientationContext{
		SourceAgent: "claude", SourceSession: "old-native", NativeTranscripts: []string{"/fixture/old.jsonl"},
	})
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
	value, err := dispatchTestOperation(env.Couch, "peek", map[string]string{"repo-scope": record.Address.RepoScope, "ref": string(record.Address.Tag), "previous": "true", "all": "true", "transcripts": "true"})
	if err != nil {
		t.Fatal(err)
	}
	r := value.(PeekResult)
	if !r.Complete || !r.Previous || r.Source != "previous-recording" || len(r.Lines) != len(want) || r.Lines[0] != want[0] || r.NativeSession != "old-native" || len(r.Transcripts) != 1 || r.Raw != "/fixture/old.raw" {
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
