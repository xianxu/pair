package couchcmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/ariadne/pkg/sidedoc"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

// These are authored expected edits, not a question/command classifier. The
// shared manifest chooses the primary documents; filesystem-move examples are
// historical move contracts and do not describe an editorial-policy write.
func TestCollaborativeFixturesPublicCLI(t *testing.T) {
	const root = "../editorial/testdata/contract838"
	raw, err := os.ReadFile(filepath.Join(root, "export.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Members []struct {
			Role, Name, Path, SHA256 string
			Bytes                    int
		}
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	type fixture struct{ name, before, after string }
	members := map[string][]byte{}
	primaries := map[string]string{}
	moves := map[string]bool{}
	for _, member := range manifest.Members {
		if member.Role != "fixture" {
			continue
		}
		if !fs.ValidPath(member.Path) {
			t.Fatalf("invalid fixture path %q", member.Path)
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(member.Path)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if len(data) != member.Bytes || hex.EncodeToString(sum[:]) != member.SHA256 {
			t.Fatalf("fixture differs from manifest: %s", member.Path)
		}
		members[member.Path] = data
		parts := strings.Split(member.Path, "/")
		if len(parts) != 4 || parts[2] != "before" {
			continue
		}
		switch member.Name {
		case "move-rename", "move-orphan":
			moves[member.Name] = true
		default:
			primaries[member.Name] = member.Path
		}
	}
	if len(primaries) != 13 || len(moves) != 2 {
		t.Fatalf("coverage: %d editorial examples, %d historical moves", len(primaries), len(moves))
	}
	var cases []fixture
	for name, before := range primaries {
		after := strings.Replace(before, "/before/", "/after/", 1)
		if _, ok := members[after]; !ok {
			t.Fatalf("missing after primary: %s", after)
		}
		cases = append(cases, fixture{name, before, after})
	}
	sort.Slice(cases, func(i, j int) bool { return cases[i].name < cases[j].name })
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rt, _ := sessionSyncRT(t)
			file := filepath.Join(t.TempDir(), filepath.Base(tc.before))
			before, after := string(members[tc.before]), string(members[tc.after])
			if err := os.WriteFile(file, []byte(before), 0600); err != nil {
				t.Fatal(err)
			}
			collaborativeFixtureMustRun(t, rt, "register", file, "--owner", "ops:0", "--json")
			snapshot := collaborativeFixtureRead(t, rt, file, before)
			if tc.name == "chat-tag" {
				// Adjacent chat-item answer appends plus a new discussions block
				// coalesce in the whole-file diff. Apply each authored line delta
				// separately so each transaction preserves item source membership.
				collaborativeFixtureChatTagEdits(t, rt, file, before, after, snapshot.Hash)
			} else {
				body := filepath.Join(t.TempDir(), "after.md")
				if err := os.WriteFile(body, []byte(after), 0600); err != nil {
					t.Fatal(err)
				}
				collaborativeFixtureMustRun(t, rt, "write", "--all", body, file, "--if-hash", snapshot.Hash)
			}
			actual, err := os.ReadFile(file)
			if err != nil || string(actual) != after {
				t.Fatalf("write result %q, want authored %q: %v", actual, after, err)
			}
			final := collaborativeFixtureRead(t, rt, file, after)
			if len(snapshot.Editorial.Owned) != len(final.Editorial.Owned) {
				t.Fatal("protected source count changed")
			}
			for i, owned := range snapshot.Editorial.Owned {
				got := final.Editorial.Owned[i]
				if owned.Key != got.Key || !bytes.Equal(owned.Raw, got.Raw) {
					t.Fatalf("protected source changed: %q -> %q", owned.Raw, got.Raw)
				}
			}
			out := collaborativeFixtureMustRun(t, rt, "round", file, "--json")
			var response struct{ Round *sessionsync.Round }
			if err := json.Unmarshal([]byte(out), &response); err != nil {
				t.Fatal(err)
			}
			if before == after {
				if response.Round != nil {
					t.Fatal("no-op authored fixture created a round")
				}
			} else {
				if response.Round == nil || response.Round.Presentation != "highlight" || len(response.Round.Spans) == 0 {
					t.Fatalf("round: %s", out)
				}
				for _, span := range response.Round.Spans {
					if span.Presentation != "highlight" {
						t.Fatalf("non-highlight span: %+v", span)
					}
				}
				if response.Round.ResultHash != sessionsync.Hash(after) {
					t.Fatal("round does not describe authored clean output")
				}
			}
		})
	}
	// Never rewrite pinned historical inputs while exercising their copies.
	for path, before := range members {
		after, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil || !bytes.Equal(before, after) {
			t.Fatalf("pinned fixture changed: %s: %v", path, err)
		}
	}
}

func TestCollaborativeFixturesRetiredValuesCLINotices(t *testing.T) {
	for name, keys := range map[string]string{
		"empty":     "presentation-mode:\nprocessing-mode:\n",
		"unknown":   "presentation-mode: unknown\nprocessing-instruction: ignored\n",
		"malformed": "processing-mode: [unterminated\n",
	} {
		t.Run(name, func(t *testing.T) {
			rt, _ := sessionSyncRT(t)
			file := filepath.Join(t.TempDir(), "note.md")
			before := "---\n" + keys + "---\n# Notes\nold text\n"
			if err := os.WriteFile(file, []byte(before), 0600); err != nil {
				t.Fatal(err)
			}
			collaborativeFixtureMustRun(t, rt, "register", file, "--owner", "ops:0", "--json")
			snapshot := collaborativeFixtureRead(t, rt, file, before)
			if len(snapshot.Editorial.Diagnostics) == 0 {
				t.Fatal("retired values emitted no compatibility notice")
			}
			edits := filepath.Join(t.TempDir(), "edits.json")
			if err := os.WriteFile(edits, []byte(`[{"old":"old text","new":"clean text","occurrence":1}]`), 0600); err != nil {
				t.Fatal(err)
			}
			collaborativeFixtureMustRun(t, rt, "apply", edits, file, "--if-hash", snapshot.Hash, "--json")
			after := strings.Replace(before, "old text", "clean text", 1)
			actual, err := os.ReadFile(file)
			if err != nil || string(actual) != after {
				t.Fatalf("retired setting/source changed: %q %v", actual, err)
			}
			collaborativeFixtureRead(t, rt, file, after)
		})
	}
}

func collaborativeFixtureMustRun(t *testing.T, rt testRT, args ...string) string {
	t.Helper()
	code, out, stderr := runSS(t, rt, args...)
	if code != 0 || stderr != "" {
		t.Fatalf("%v: code=%d stdout=%s stderr=%s", args, code, out, stderr)
	}
	return out
}

func collaborativeFixtureRead(t *testing.T, rt testRT, file, source string) sessionsync.ReadResult {
	t.Helper()
	scan := sidedoc.ChatSections(source)
	want := append([]string(nil), scan.Notices...)
	for _, line := range scan.Misplaced {
		want = append(want, "@@chat@@ at line "+strconv.Itoa(line)+" is not directly above a heading")
	}
	out := collaborativeFixtureMustRun(t, rt, "read", "--all", file, "--json")
	var response struct {
		Code string
		Read *sessionsync.ReadResult
	}
	if err := json.Unmarshal([]byte(out), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "ok" || response.Read == nil || response.Read.Editorial == nil || response.Read.Body != source {
		t.Fatalf("read: %s", out)
	}
	var got []string
	for _, d := range response.Read.Editorial.Diagnostics {
		got = append(got, d.Message)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("notice sequence %q, want shared %q", got, want)
	}
	code, text, stderr := runSS(t, rt, "read", "--all", file)
	if code != 0 || text != "hash: "+response.Read.Hash+"\n"+source {
		t.Fatalf("human read: %d %q %q", code, text, stderr)
	}
	var expected strings.Builder
	for _, notice := range want {
		expected.WriteString("couch: editorial: " + notice + "\n")
	}
	if stderr != expected.String() {
		t.Fatalf("human notices duplicated/reworded: %q, want %q", stderr, expected.String())
	}
	return *response.Read
}

// The authored chat-tag fixture adds a leading index and changes three lines.
// This adapter of the fixture's exact delta makes no judgment about its asks.
func collaborativeFixtureChatTagEdits(t *testing.T, rt testRT, file, before, after, hash string) {
	t.Helper()
	oldLines := strings.SplitAfter(before, "\n")
	at := strings.Index(after, oldLines[0])
	if at <= 0 {
		t.Fatal("chat fixture no longer has a leading index addition")
	}
	prefix, body := after[:at], after[at:]
	newLines := strings.SplitAfter(body, "\n")
	if len(oldLines) != len(newLines) {
		t.Fatal("chat fixture needs an updated exact-line adapter")
	}
	current := before
	apply := func(old, replacement string) {
		t.Helper()
		if strings.Count(current, old) != 1 {
			t.Fatalf("authored line is not unique: %q", old)
		}
		records, err := json.Marshal([]sessionsync.EditRecord{{Old: old, New: replacement, Occurrence: 1}})
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "edits.json")
		if err := os.WriteFile(path, records, 0600); err != nil {
			t.Fatal(err)
		}
		collaborativeFixtureMustRun(t, rt, "apply", path, file, "--if-hash", hash, "--json")
		current = strings.Replace(current, old, replacement, 1)
		snapshot := collaborativeFixtureRead(t, rt, file, current)
		hash = snapshot.Hash
		out := collaborativeFixtureMustRun(t, rt, "round", file, "--json")
		var result struct{ Round *sessionsync.Round }
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatal(err)
		}
		if result.Round == nil || result.Round.Presentation != "highlight" || result.Round.ResultHash != sessionsync.Hash(current) {
			t.Fatalf("clean round missing: %s", out)
		}
		for _, span := range result.Round.Spans {
			if span.Presentation != "highlight" {
				t.Fatalf("span=%+v", span)
			}
		}
	}
	apply(oldLines[0], prefix+oldLines[0])
	for i, old := range oldLines {
		if old != newLines[i] {
			apply(old, newLines[i])
		}
	}
	if current != after {
		t.Fatal("line adapter did not reconstruct the authored after document")
	}
}
