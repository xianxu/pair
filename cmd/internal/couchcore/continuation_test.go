package couchcore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestContinuationStartsTheNextConversationWithItsLaunchPrompt(t *testing.T) {
	t.Parallel()

	env, source := switchEnvWithLiveThread(t)
	proof := ContinuationSource{Agent: "claude", Session: "pair-exact", LaunchOrdinal: 2}
	env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) { return proof, nil }
	path := filepath.Join(t.TempDir(), "handoff.md")
	body := "---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nVerify the durable token.\n"
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	status, err := env.Couch.RequestContinuation(context.Background(), source.Address, proof, path)
	if err != nil {
		t.Fatal(err)
	}
	again, err := env.Couch.RequestContinuation(context.Background(), source.Address, proof, path)
	if err != nil || again.RequestID != status.RequestID {
		t.Fatalf("duplicate = %+v %v", again, err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	registered := false
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return registered, nil }
	env.Runner.AfterAcknowledge = func(id string) error {
		registered = true
		env.Artifacts.SetPairSession(source.Address, continuationChildSession(t, env.Runner, id), true)
		return nil
	}
	var prompt string
	env.Runner.AfterBlockedStart = func(id string) {
		var profile launcher.TrustedLaunchProfile
		for _, entry := range env.Runner.Child(id).Env {
			if raw, ok := strings.CutPrefix(entry, launcher.CouchLaunchProfileEnv+"="); ok {
				if err := json.Unmarshal([]byte(raw), &profile); err != nil {
					t.Error(err)
				}
			}
		}
		prompt = profile.Prompt
	}
	result, err := env.Couch.Continue(context.Background(), source.Address, status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	started, ok := result.Started()
	if !ok || started.Record.Thread != source.Address {
		t.Fatalf("start = %+v", result)
	}
	// Registration completes it: the prompt was part of the launch (#522).
	if result.Status.Phase != checkpoint.Complete || result.Missed != nil {
		t.Fatalf("not complete at registration: %+v", result)
	}
	seed := mustContinuationPath(t, env.Couch, source.Address)
	materialized, err := os.ReadFile(seed)
	if err != nil || string(materialized) != body {
		t.Fatalf("snapshot %q %v", materialized, err)
	}
	if want := ContinuationPrompt(seed, source.Address.Tag); prompt != want {
		t.Fatalf("launch prompt %q, want %q", prompt, want)
	}
	// The coordinates: the conversation it replaced and the continuation.
	record, err := env.Couch.Threads.GetThread(source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if p := record.PreviousSession; p == nil || p.Continuation != seed || p.Source.PID != 42 {
		t.Fatalf("previous session %+v", p)
	}
	// Said once: no saved profile, start argument or record carries it.
	raw, err := os.ReadFile(env.Couch.Threads.recordPath(source.Address))
	if err != nil || strings.Contains(string(raw), "saved continuation at") {
		t.Fatalf("the thread record carries the prompt: %v", err)
	}
	if slices.Contains(started.Record.Args.ExtraArgs, prompt) || slices.Contains(record.LatestLaunchProfile.Argv, prompt) {
		t.Fatal("the prompt reached a saved argv")
	}
}

func TestContinuationRejectsChangedSourceBeforePark(t *testing.T) {
	t.Parallel()

	env, source := switchEnvWithLiveThread(t)
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return false, nil }
	request := testContinuationRequest(t, source)
	request.Source.Agent = "claude"
	request.Source.Session = "pair-exact"
	request.Source.Helper = checkpoint.Process{PID: 42, Identity: "pair-helper"}
	if _, err := env.Couch.Threads.PublishContinuation(source.Address, source.Revision, request); err != nil {
		t.Fatal(err)
	}
	env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) {
		return ContinuationSource{Agent: "codex", Session: "pair-exact", LaunchOrdinal: 3}, nil
	}
	if _, err := env.Couch.Continue(context.Background(), source.Address, request.ID); err == nil {
		t.Fatal("changed generation accepted")
	}
	// Before park: the old conversation keeps running, and the line says the
	// compaction didn't finish without blocking anything.
	current, _ := env.Couch.Threads.GetThread(source.Address)
	if current.Continuation != nil || current.Park != nil || len(current.Incarnations) != 1 || current.ContinuationMissed == nil || current.ContinuationMissed.TargetMayExist {
		t.Fatalf("wrong failure state %+v", current)
	}
}

func TestContinuationRejectsChangedCheckpointDigestBeforePublication(t *testing.T) {
	t.Parallel()

	env, source := switchEnvWithLiveThread(t)
	proof := ContinuationSource{Agent: "claude", Session: "pair-exact", LaunchOrdinal: 2, ExpectedDigest: "outdated-digest"}
	env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) {
		return ContinuationSource{Agent: proof.Agent, Session: proof.Session, LaunchOrdinal: proof.LaunchOrdinal}, nil
	}
	path := filepath.Join(t.TempDir(), "checkpoint.md")
	if err := os.WriteFile(path, []byte("---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nModified handoff.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.RequestContinuation(context.Background(), source.Address, proof, path); err == nil {
		t.Fatal("changed snapshot accepted")
	}
	record, _ := env.Couch.Threads.GetThread(source.Address)
	if record.Continuation != nil || record.Park != nil {
		t.Fatal("digest failure mutated the source")
	}
}

func TestContinuationConcurrentPublicationDeduplicates(t *testing.T) {
	t.Parallel()

	env, source := switchEnvWithLiveThread(t)
	proof := ContinuationSource{Agent: "claude", Session: "pair-exact", LaunchOrdinal: 2}
	env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) { return proof, nil }
	path := filepath.Join(t.TempDir(), "checkpoint.md")
	if err := os.WriteFile(path, []byte("---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nOne accepted request.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	results := make(chan ContinuationStatus, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			s, err := env.Couch.RequestContinuation(context.Background(), source.Address, proof, path)
			results <- s
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	id := ""
	for s := range results {
		if id != "" && id != s.RequestID {
			t.Fatal("publication created multiple requests")
		}
		id = s.RequestID
	}
	record, err := env.Couch.Threads.GetThread(source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if record.Continuation.ID != id || record.Park != nil || record.Continuation.Phase != checkpoint.Pending {
		t.Fatalf("concurrent publication %+v", record)
	}
}

func mustContinuationPath(t *testing.T, c *Couch, address ThreadAddress) string {
	t.Helper()
	path, err := c.continuationPath(address)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

// continuationPromptPath reads the continuation path back out of a launch
// prompt, as a stand-in agent would.
func continuationPromptPath(prompt string) (string, error) {
	rest, ok := strings.CutPrefix(prompt, "Continue from the saved continuation at ")
	if !ok {
		return "", fmt.Errorf("not a continuation prompt: %q", prompt)
	}
	quoted, err := strconv.QuotedPrefix(rest)
	if err != nil {
		return "", err
	}
	return strconv.Unquote(quoted)
}
