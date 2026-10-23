package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessionsync"
	"github.com/xianxu/pair/cmd/internal/testcouch"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

// TestNativeEditorial qualifies the mechanical contract through disposable
// Claude/Codex personalities. Semantic review and Parley rendering remain
// operator acceptance; these personalities do not make model judgments.
func TestNativeEditorial(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_EDITORIAL") != "1" {
		t.Skip("set PAIR_NATIVE_EDITORIAL=1 to qualify native editorial CLI")
	}
	binaries := testcouch.Build(t)
	raw, err := os.ReadFile(filepath.Join(binaries.Source, "cmd/internal/sessionsync/testdata/editorial-rounds/presentation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name, Before, Result string
		Round                sessionsync.Round
		Expected             []string `json:"expected_effective"`
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 || cases[0].Name != "mixed" {
		t.Fatal("mixed presentation fixture missing")
	}
	mixed := cases[0]
	guidance, err := os.ReadFile(filepath.Join(binaries.Source, "cmd/internal/couchcmd/skills/couch/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"claude", "codex"} {
		if !t.Run(kind, func(t *testing.T) {
			started := time.Now()
			fleet := testcouch.New(t, binaries, testcouch.Options{Workers: 1, AgentKind: kind})
			setup, cancelSetup := context.WithTimeout(t.Context(), 180*time.Second)
			err := fleet.Prepare(setup)
			cancelSetup()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
			defer cancel()
			if err := fleet.Start(ctx); err != nil {
				t.Fatal(err)
			}
			actor, err := fleet.Agent(0)
			if err != nil || actor.Agent != kind {
				t.Fatalf("fixture personality: %+v %v", actor, err)
			}
			call := func(wantExit int, args ...string) agentfixture.Result {
				t.Helper()
				result, err := fleet.ActorCLI(ctx, 0, args...)
				// ActorCLI reports a successfully joined nonzero child as an
				// ordinary error too. Accept precisely that expected refusal;
				// joined cancellation, retirement or ownership errors still fail.
				if wantExit > 0 && err != nil && !result.CleanupUncertain && result.Error == fmt.Sprintf("exit status %d", wantExit) && err.Error() == "fixture CLI: "+result.Error {
					err = nil
				}
				if err != nil || result.ExitCode != wantExit {
					t.Fatalf("ActorCLI %v: got %+v error=%v want exit%d", args, result, err, wantExit)
				}
				return result
			}
			// ActorCLI intentionally bounds every reply to 32 KiB. The complete
			// static skill exceeds that bound; observe the same candidate directly
			// for this read-only command and prove the actor returned its exact
			// prefix, with the expected truncation flag. Mutation stays ActorCLI.
			skillReply := call(0, "--skill")
			skillCtx, cancelSkill := context.WithTimeout(ctx, 10*time.Second)
			command := exec.CommandContext(skillCtx, binaries.Couch, "--skill")
			command.Dir, command.Env = fleet.Workspaces.Primary, fleet.Environment
			output := &nativeEditorialOutput{}
			command.Stdout, command.Stderr = output, output
			err = command.Run()
			cancelSkill()
			if err != nil || output.exceeded {
				t.Fatalf("complete candidate skill: error=%v exceeded=%t", err, output.exceeded)
			}
			skill := output.String()
			if skill != string(guidance) {
				t.Fatal("candidate --skill differs from its exact shipped source")
			}
			prefix := skill
			wantTruncated := len(skill) > agentfixture.MaxOutputBytes
			if wantTruncated {
				prefix = skill[:agentfixture.MaxOutputBytes]
			}
			if skillReply.Stdout != prefix || skillReply.Stderr != "" || skillReply.Truncated != wantTruncated {
				t.Fatalf("actor skill mismatch: candidateBytes=%d actorBytes=%d truncated=%t want=%t stderr=%q", len(skill), len(skillReply.Stdout), skillReply.Truncated, wantTruncated, skillReply.Stderr)
			}
			t.Logf("native guidance: complete candidate bytes=%d actor prefix bytes=%d truncated=%t exact-prefix=PASS", len(skill), len(skillReply.Stdout), skillReply.Truncated)
			for _, required := range []string{"collaboration-mode", "processing-mode", "presentation-mode", "operator-owned", "Never write the markers yourself", "migrate-editorial", "side-doc"} {
				if !strings.Contains(skill, required) {
					t.Fatalf("emitted guidance missing %q", required)
				}
			}
			file := filepath.Join(fleet.Workspaces.Primary, "editorial-mixed.md")
			put := func(path, body string) {
				t.Helper()
				if err := os.WriteFile(path, []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			get := func(path string) string {
				t.Helper()
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				return string(raw)
			}
			put(file, mixed.Before)
			var registration struct{ Registration sessionsync.Registration }
			result := call(0, "--session-sync", "register", file, "--owner", "fixture:0", "--json")
			if err := json.Unmarshal([]byte(result.Stdout), &registration); err != nil {
				t.Fatal(err)
			}
			if registration.Registration.Owner != "fixture:0" || registration.Registration.ID == "" {
				t.Fatalf("registration %s", result.Stdout)
			}
			read := func() sessionsync.ReadResult {
				t.Helper()
				var response struct{ Read sessionsync.ReadResult }
				result := call(0, "--session-sync", "read", "--all", file, "--json")
				if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
					t.Fatal(err)
				}
				return response.Read
			}
			before := read()
			if before.Body != mixed.Before || before.Editorial == nil || len(before.Editorial.Regions) != len(mixed.Expected)+1 {
				t.Fatalf("unresolved mixed read: %+v", before)
			}
			for i, want := range mixed.Expected {
				effective := before.Editorial.Regions[i+1].Effective
				if effective.Presentation != want || effective.Collaboration != "discussion" || effective.Processing != "free-form" {
					t.Fatalf("region%d axes: %+v", i, effective)
				}
			}
			// Keep the clean input independent of the pinned output oracle. Couch
			// alone chooses marker placement and narrowed round spans.
			edits := []map[string]any{
				{"old": "old highlight", "new": "new highlight", "occurrence": 1, "explain": "native highlight qualification"},
				{"old": "old proposal", "new": "new proposal", "occurrence": 1, "explain": "native propose qualification"},
				{"old": "old direct", "new": "new direct", "occurrence": 1, "explain": "native direct qualification"},
				{"old": "old inherited", "new": "new inherited", "occurrence": 1, "explain": "native inherited qualification"},
			}
			editFile := filepath.Join(fleet.Workspaces.Primary, "editorial-edits.json")
			encoded, err := json.Marshal(edits)
			if err != nil {
				t.Fatal(err)
			}
			put(editFile, string(encoded))
			call(0, "--session-sync", "apply", editFile, file, "--if-hash", before.Hash, "--json")
			if got := get(file); got != mixed.Result {
				t.Fatalf("native result differs from pinned fixture\nwant: %q\ngot: %q", mixed.Result, got)
			}
			var roundResponse struct{ Round *sessionsync.Round }
			result = call(0, "--session-sync", "round", file, "--json")
			if err := json.Unmarshal([]byte(result.Stdout), &roundResponse); err != nil {
				t.Fatal(err)
			}
			if roundResponse.Round == nil || roundResponse.Round.BaseHash != mixed.Round.BaseHash || roundResponse.Round.ResultHash != mixed.Round.ResultHash || len(roundResponse.Round.Spans) != len(mixed.Expected) {
				t.Fatalf("round %s", result.Stdout)
			}
			for i, span := range roundResponse.Round.Spans {
				if span.Presentation != mixed.Expected[i] {
					t.Fatalf("span%d presentation %q want %q", i, span.Presentation, mixed.Expected[i])
				}
				want := mixed.Round.Spans[i]
				if span.Start != want.Start || span.End != want.End || span.Old != want.Old || span.New != want.New {
					t.Fatalf("span%d differs from pinned result coordinates/text: got %+v want %+v", i, span, want)
				}
			}
			// Reading acknowledges the successful content before measuring a refused
			// write, so refusal must leave both acknowledgement and round bytes alone.
			current := read()
			metadata := filepath.Join(sessionsync.LedgerDir(registration.Registration.Namespace), "documents", "state", registration.Registration.ID)
			snapshot := map[string]string{}
			for _, path := range []string{file, filepath.Join(metadata, "shadow"), filepath.Join(metadata, "state.json"), filepath.Join(metadata, "round.json")} {
				snapshot[path] = get(path)
			}
			put(editFile, `[{"old":"presentation-mode: direct","new":"presentation-mode: highlight"}]`)
			result = call(sessionsync.ExitRefused, "--session-sync", "apply", editFile, file, "--if-hash", current.Hash, "--json")
			var refused struct{ Code string }
			if err := json.Unmarshal([]byte(result.Stdout), &refused); err != nil {
				t.Fatal(err)
			}
			if refused.Code != "operator-owned" {
				t.Fatalf("protected write %s", result.Stdout)
			}
			for path, want := range snapshot {
				if get(path) != want {
					t.Fatalf("refusal changed %s", path)
				}
			}
			legacy := filepath.Join(fleet.Workspaces.Primary, "editorial-legacy.md")
			legacySource := "---\nopaque: {keep: yes}\ncollaboration-mode: {default: discussion, chat: [Notes]}\n---\n# Notes\nhello\n"
			put(legacy, legacySource)
			call(0, "--session-sync", "register", legacy, "--owner", "fixture:0", "--json")
			preview := func() sessionsync.EditorialMigrationResult {
				t.Helper()
				result := call(0, "--session-sync", "migrate-editorial", legacy, "--json")
				var response struct {
					Migration sessionsync.EditorialMigrationResult
				}
				if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
					t.Fatal(err)
				}
				if len(response.Migration.BeforeHash) != 64 || response.Migration.Applied || !strings.Contains(response.Migration.Candidate, "<!-- collaboration-mode: chat -->") || !strings.Contains(response.Migration.Candidate, "opaque: {keep: yes}") {
					t.Fatalf("migration preview %s", result.Stdout)
				}
				return response.Migration
			}
			planned := preview()
			if get(legacy) != legacySource {
				t.Fatal("migration preview changed source")
			}
			changed := legacySource + "operator edit\n"
			put(legacy, changed)
			result = call(sessionsync.ExitStale, "--session-sync", "migrate-editorial", legacy, "--confirm", "--if-hash", planned.BeforeHash, "--json")
			if !strings.Contains(result.Stdout, `"Code":"stale"`) || get(legacy) != changed {
				t.Fatalf("stale migration %s", result.Stdout)
			}
			planned = preview()
			result = call(0, "--session-sync", "migrate-editorial", legacy, "--confirm", "--if-hash", planned.BeforeHash, "--json")
			var applied struct {
				Migration sessionsync.EditorialMigrationResult
			}
			if err := json.Unmarshal([]byte(result.Stdout), &applied); err != nil {
				t.Fatal(err)
			}
			if !applied.Migration.Applied || get(legacy) != planned.Candidate {
				t.Fatalf("confirmed migration differs from preview: %s", result.Stdout)
			}
			t.Logf("native editorial: agent=%s actor=fixture:0 fixture=editorial-rounds/presentation.json modes=highlight,propose,direct exact-result=PASS protected-refusal=operator-owned unchanged-file-shadow-state-round=PASS migration-preview-confirm-stale=PASS elapsed=%s", kind, time.Since(started))
			cleanup, cancelCleanup := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancelCleanup()
			if err := fleet.Close(cleanup); err != nil {
				t.Fatal(err)
			}
			if err := fleet.AssertCleaned(); err != nil {
				t.Fatal(err)
			}
		}) {
			break
		}
	}
}

// Bound diagnostic memory without changing the shared fixture's output budget.
type nativeEditorialOutput struct {
	buffer   bytes.Buffer
	exceeded bool
}

func (w *nativeEditorialOutput) String() string { return w.buffer.String() }

func (w *nativeEditorialOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := 128<<10 - w.buffer.Len()
	if len(p) > remaining {
		p = p[:remaining]
		w.exceeded = true
	}
	_, _ = w.buffer.Write(p)
	return n, nil
}
