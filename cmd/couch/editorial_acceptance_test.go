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

var nativeEditorialFixtures = []struct{ name, file, notice, protected string }{
	{"chat-tag", "session.md", "", "@@chat@@"},
	{"legacy-chat-map", "status.md", "collaboration-mode map is a legacy key; its chat sections are read as @@chat@@", "collaboration-mode"},
	{"ask-command", "design.md", "", ""},
	{"ask-question", "design.md", "", ""},
	{"legacy-keys-ignored", "note.md", "retired keys ignored: presentation, presentation-mode, process-instruction, processing-instruction, processing-mode", "presentation: direct"},
}

// Historical direct/propose saved-round fixtures remain in sessionsync tests.
// This preflight checks that authored upstream decisions are accepted as clean
// highlight edits before spending time on the disposable native fleet.
func TestEditorialAuthoredFixtureInputs(t *testing.T) {
	for _, fixture := range nativeEditorialFixtures {
		t.Run(fixture.name, func(t *testing.T) {
			root := filepath.Join("../internal/editorial/testdata/contract838/examples", fixture.name)
			before, err := os.ReadFile(filepath.Join(root, "before", fixture.file))
			if err != nil {
				t.Fatal(err)
			}
			after, err := os.ReadFile(filepath.Join(root, "after", fixture.file))
			if err != nil {
				t.Fatal(err)
			}
			current := string(before)
			for _, record := range nativeEditorialFixtureSteps(t, fixture.name, current, string(after)) {
				edit, err := sessionsync.BuildEdit(current, []sessionsync.EditRecord{record}, time.Time{}, false)
				if err != nil {
					t.Fatal(err)
				}
				if len(edit.Round.Spans) == 0 {
					t.Fatal("authored step made no round")
				}
				for _, span := range edit.Round.Spans {
					if span.Presentation != "highlight" {
						t.Fatal("new span is not highlight")
					}
				}
				current = edit.Next
			}
			if current != string(after) {
				t.Fatal("authored sequence differs from exact fixture")
			}

		})
	}
}

// The existing line-diff owner coalesces adjacent chat answer appends into
// an indivisible hunk. It may refuse a coarse whole-file request even though a
// sequence of localized edits reaches the authored result. Preserve that guard.
func TestEditorialAuthoredCoarseChatRefusal(t *testing.T) {
	root := "../internal/editorial/testdata/contract838/examples/chat-tag"
	before, err := os.ReadFile(filepath.Join(root, "before/session.md"))
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(root, "after/session.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = sessionsync.BuildEdit(string(before), []sessionsync.EditRecord{{Old: string(before), New: string(after), Occurrence: 1}}, time.Time{}, false)
	if err == nil || !strings.Contains(err.Error(), "editorial-chat") {
		t.Fatalf("coarse chat write no longer has the expected membership refusal: %v", err)
	}
}

// This exact-line fixture adapter matches the non-native CLI replay. It
// executes the already-authored decisions without classifying natural language.
func nativeEditorialFixtureSteps(t *testing.T, name, before, after string) []sessionsync.EditRecord {
	t.Helper()
	if name != "chat-tag" {
		return []sessionsync.EditRecord{{Old: before, New: after, Occurrence: 1}}
	}
	oldLines := strings.SplitAfter(before, "\n")
	at := strings.Index(after, oldLines[0])
	if at <= 0 {
		t.Fatal("chat fixture no longer has a leading index addition")
	}
	prefix, body := after[:at], after[at:]
	newLines := strings.SplitAfter(body, "\n")
	if len(oldLines) != len(newLines) {
		t.Fatal("chat fixture requires an updated exact-line adapter")
	}
	steps := []sessionsync.EditRecord{{Old: oldLines[0], New: prefix + oldLines[0], Occurrence: 1}}
	current := strings.Replace(before, oldLines[0], prefix+oldLines[0], 1)
	for i, old := range oldLines {
		if old == newLines[i] {
			continue
		}
		if strings.Count(current, old) != 1 {
			t.Fatalf("authored line is not unique: %q", old)
		}
		steps = append(steps, sessionsync.EditRecord{Old: old, New: newLines[i], Occurrence: 1})
		current = strings.Replace(current, old, newLines[i], 1)
	}
	if current != after {
		t.Fatal("localized steps do not reconstruct authored after")
	}
	return steps
}

// TestNativeEditorial qualifies the mechanical contract through disposable
// Claude/Codex personalities. Semantic review and Parley rendering remain
// operator acceptance; these personalities do not make model judgments.
func TestNativeEditorial(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_EDITORIAL") != "1" {
		t.Skip("set PAIR_NATIVE_EDITORIAL=1 to qualify native editorial CLI")
	}
	binaries := testcouch.Build(t)
	selected := os.Getenv("PAIR_NATIVE_EDITORIAL_AGENT")
	if selected != "" && selected != "claude" && selected != "codex" {
		t.Fatal("PAIR_NATIVE_EDITORIAL_AGENT must be claude or codex")
	}
	guidance, err := os.ReadFile(filepath.Join(binaries.Source, "cmd/internal/couchcmd/skills/couch/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"claude", "codex"} {
		if selected != "" && selected != kind {
			continue
		}
		if !t.Run(kind, func(t *testing.T) {
			started := time.Now()
			fleet := testcouch.New(t, binaries, testcouch.Options{Workers: 1, AgentKind: kind})
			setup, cancelSetup := context.WithTimeout(t.Context(), 180*time.Second)
			err := fleet.Prepare(setup)
			cancelSetup()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 300*time.Second)
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
			for _, required := range []string{"@@chat@@", "numbered shallowest-indent", "ordinary marker routing", "Couch does not classify natural language", "new edits always use highlight", "Legacy chat scalars, maps and heading comments", "Report each shared notice once, verbatim", "operator-owned", "migrate-editorial", "Never renumber asks or fill"} {
				if !strings.Contains(skill, required) {
					t.Fatalf("emitted guidance missing %q", required)
				}
			}
			put := func(path, body string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
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
			register := func(file string) sessionsync.Registration {
				t.Helper()
				var response struct{ Registration sessionsync.Registration }
				result := call(0, "--session-sync", "register", file, "--owner", "fixture:0", "--json")
				if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
					t.Fatal(err)
				}
				if response.Registration.Owner != "fixture:0" || response.Registration.ID == "" {
					t.Fatalf("registration %s", result.Stdout)
				}
				return response.Registration
			}
			read := func(file string) sessionsync.ReadResult {
				t.Helper()
				call(0, "--session-sync", "diff", file)
				var response struct{ Read sessionsync.ReadResult }
				result := call(0, "--session-sync", "read", "--all", file, "--json")
				if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
					t.Fatal(err)
				}
				return response.Read
			}
			editFile := filepath.Join(fleet.Workspaces.Primary, "editorial-edits.json")
			apply := func(file, hash, old, next string, exit int) agentfixture.Result {
				t.Helper()
				raw, err := json.Marshal([]sessionsync.EditRecord{{Old: old, New: next, Occurrence: 1, Explain: "authored upstream fixture outcome; mechanical qualification"}})
				if err != nil {
					t.Fatal(err)
				}
				put(editFile, string(raw))
				return call(exit, "--session-sync", "apply", editFile, file, "--if-hash", hash, "--json")
			}
			checkRound := func(file string) {
				t.Helper()
				var response struct{ Round *sessionsync.Round }
				result := call(0, "--session-sync", "round", file, "--json")
				if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
					t.Fatal(err)
				}
				if response.Round == nil || len(response.Round.Spans) == 0 || response.Round.Presentation != "highlight" {
					t.Fatalf("missing highlight round: %s", result.Stdout)
				}
				for _, span := range response.Round.Spans {
					if span.Presentation != "highlight" {
						t.Fatalf("non-highlight span: %+v", span)
					}
				}
			}
			fixtureRoot := filepath.Join(binaries.Source, "cmd/internal/editorial/testdata/contract838/examples")
			for _, fixture := range nativeEditorialFixtures {
				input := filepath.Join(fixtureRoot, fixture.name, "before")
				output := filepath.Join(fixtureRoot, fixture.name, "after")
				dir := filepath.Join(fleet.Workspaces.Primary, "authored-"+fixture.name)
				// Before files are operator input. After files are authored fixture
				// decisions, applied through ActorCLI; no model classifies the asks.
				if err := filepath.WalkDir(input, func(source string, entry os.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if entry.IsDir() {
						return nil
					}
					rel, err := filepath.Rel(input, source)
					if err != nil {
						return err
					}
					put(filepath.Join(dir, rel), get(source))
					return nil
				}); err != nil {
					t.Fatal(err)
				}
				file := filepath.Join(dir, fixture.file)
				registration := register(file)
				before := read(file)
				if before.Body != get(filepath.Join(input, fixture.file)) || before.Editorial == nil {
					t.Fatalf("fixture read missing original source: %+v", before)
				}
				for _, region := range before.Editorial.Regions {
					if region.Effective.Presentation != "highlight" {
						t.Fatalf("retired key selected presentation: %+v", region)
					}
				}
				if fixture.notice != "" {
					found := 0
					for _, notice := range before.Editorial.Diagnostics {
						if notice.Message == fixture.notice {
							found++
						}
					}
					if found != 1 {
						t.Fatalf("fixture %s missing compatibility notice %q: %+v", fixture.name, fixture.notice, before.Editorial.Diagnostics)
					}
				}
				// Materialize each authored side-doc outcome through the same CLI.
				if err := filepath.WalkDir(output, func(source string, entry os.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if entry.IsDir() {
						return nil
					}
					rel, err := filepath.Rel(output, source)
					if err != nil {
						return err
					}
					if rel == fixture.file {
						return nil
					}
					target := filepath.Join(dir, rel)
					if _, err := os.Stat(target); os.IsNotExist(err) {
						put(target, "Fixture staging placeholder.\n")
					} else if err != nil {
						return err
					}
					register(target)
					snapshot := read(target)
					apply(target, snapshot.Hash, snapshot.Body, get(source), 0)
					if get(target) != get(source) {
						t.Fatalf("authored side-doc differs: %s", rel)
					}
					checkRound(target)
					return nil
				}); err != nil {
					t.Fatal(err)
				}
				want := get(filepath.Join(output, fixture.file))
				current := before
				for _, record := range nativeEditorialFixtureSteps(t, fixture.name, before.Body, want) {
					apply(file, current.Hash, record.Old, record.New, 0)
					checkRound(file)
					current = read(file)
				}
				if get(file) != want {
					t.Fatalf("fixture %s authored result mismatch", fixture.name)
				}
				checkRound(file)
				if fixture.protected != "" {
					current := read(file)
					metadata := filepath.Join(sessionsync.LedgerDir(registration.Namespace), "documents", "state", registration.ID)
					snapshot := map[string]string{}
					for _, path := range []string{file, filepath.Join(metadata, "shadow"), filepath.Join(metadata, "state.json"), filepath.Join(metadata, "round.json")} {
						snapshot[path] = get(path)
					}
					result := apply(file, current.Hash, fixture.protected, "unauthorized replacement", sessionsync.ExitRefused)
					var refusal struct{ Code string }
					if err := json.Unmarshal([]byte(result.Stdout), &refusal); err != nil {
						t.Fatal(err)
					}
					if refusal.Code != "operator-owned" {
						t.Fatalf("owned refusal: %s", result.Stdout)
					}
					for path, want := range snapshot {
						if get(path) != want {
							t.Fatalf("owned refusal changed %s", path)
						}
					}
				}
				t.Logf("native authored fixture: agent=%s case=%s exact-parent-and-side-docs=PASS highlight-rounds=PASS notices=%q owned-refusal=%t", kind, fixture.name, fixture.notice, fixture.protected != "")
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
				if len(response.Migration.BeforeHash) != 64 || response.Migration.Applied || !strings.Contains(response.Migration.Candidate, "@@chat@@\n# Notes") || strings.Contains(response.Migration.Candidate, "collaboration-mode:") || !strings.Contains(response.Migration.Candidate, "opaque: {keep: yes}") {
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
			result := call(sessionsync.ExitStale, "--session-sync", "migrate-editorial", legacy, "--confirm", "--if-hash", planned.BeforeHash, "--json")
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
			lossy := strings.Replace(legacySource, "hello", "- legacy bullet ask", 1)
			put(legacy, lossy)
			result = call(sessionsync.ExitRefused, "--session-sync", "migrate-editorial", legacy, "--json")
			if get(legacy) != lossy || !strings.Contains(result.Stdout, "Notes") {
				t.Fatalf("lossy migration not refused with section: %s", result.Stdout)
			}
			t.Logf("native editorial: agent=%s actor=fixture:0 upstream=354e5fac5c96e0b2c26350ee712eb835746a6d67 authored-decisions=mechanical highlight-only=PASS unchanged-file-shadow-state-round-on-owned-refusal=PASS migration-preview-confirm-stale-lossy=PASS elapsed=%s", kind, time.Since(started))
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
