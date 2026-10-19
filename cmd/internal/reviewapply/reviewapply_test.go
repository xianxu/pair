package reviewapply

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

type cli struct {
	t                *testing.T
	doc, edits, data string
	round            string
}

func newCLI(t *testing.T, content string) *cli {
	repo := t.TempDir()
	c := &cli{t: t, doc: filepath.Join(repo, "doc.md"), edits: filepath.Join(t.TempDir(), "edits.json"), data: t.TempDir()}
	c.round = filepath.Join(c.data, "round.json")
	os.WriteFile(c.doc, []byte(content), 0o644)
	return c
}

func (c *cli) run(edits string, args ...string) (int, map[string]json.RawMessage, string) {
	c.t.Helper()
	os.WriteFile(c.edits, []byte(edits), 0o644)
	env := map[string]string{"PAIR_DATA_DIR": c.data, "PAIR_TAG": "t", "PAIR_REVIEW_ROUND_PATH": c.round}
	var out, errout bytes.Buffer
	code := RunCLI(append([]string{c.edits, c.doc}, args...), func(k string) string { return env[k] }, &out, &errout)
	var reply map[string]json.RawMessage
	if strings.Contains(strings.Join(args, " "), "--json") {
		if err := json.Unmarshal(out.Bytes(), &reply); err != nil {
			c.t.Fatalf("reply %q: %v", out.String(), err)
		}
	}
	return code, reply, errout.String()
}

func (c *cli) hash() string {
	raw, _ := os.ReadFile(c.doc)
	return sessionsync.Hash(string(raw))
}

func code(reply map[string]json.RawMessage) string {
	var s string
	json.Unmarshal(reply["Code"], &s)
	return s
}

// The contract ariadne#437 builds against: exit codes, Code values, Invalid
// on a bad batch, the current Hash on stale, and Body plus Round on success.
// A repository gets nothing but the document's own change.
func TestPairReviewApplyContract(t *testing.T) {
	c := newCLI(t, "The quick fox.\n")
	exit, reply, _ := c.run(`[{"old":"quick","new":"slow","explain":"pace"}]`, "--if-hash", c.hash(), "--json")
	if exit != 0 || code(reply) != "ok" || reply["Body"] == nil || reply["Round"] == nil || reply["Hash"] == nil {
		t.Fatalf("ok reply exit %d %v", exit, reply)
	}
	if entries, _ := os.ReadDir(filepath.Dir(c.doc)); len(entries) != 1 {
		t.Fatalf("apply left files beside the document: %v", entries)
	}
	if _, err := os.Stat(c.round); err != nil {
		t.Fatal("no round artifact")
	}
	before, _ := os.ReadFile(c.doc)
	exit, reply, _ = c.run(`[{"old":"slow","new":"fast"},{"old":"absent","new":"x"}]`, "--if-hash", c.hash(), "--json")
	var invalid []sessionsync.BadEdit
	json.Unmarshal(reply["Invalid"], &invalid)
	if exit != 1 || code(reply) != "edits-invalid" || len(invalid) != 1 || invalid[0] != (sessionsync.BadEdit{Index: 2, Reason: "not found"}) {
		t.Fatalf("bad batch exit %d %v", exit, reply)
	}
	exit, reply, _ = c.run(`not json`, "--if-hash", c.hash(), "--json")
	if exit != 1 || code(reply) != "edits-invalid" || reply["Invalid"] != nil {
		t.Fatalf("malformed edits exit %d %v", exit, reply)
	}
	exit, reply, _ = c.run(`[{"old":"slow","new":"fast"}]`, "--if-hash", "0000000000000000", "--json")
	var hash string
	json.Unmarshal(reply["Hash"], &hash)
	if exit != 6 || code(reply) != "stale" || hash != c.hash() {
		t.Fatalf("stale exit %d %v", exit, reply)
	}
	os.WriteFile(c.doc+".lock", []byte("holder: operator\nat: 2026-10-18T08:55\n"), 0o644)
	exit, reply, _ = c.run(`[{"old":"slow","new":"fast"}]`, "--if-hash", c.hash(), "--json")
	if exit != 3 || code(reply) != "operator-lock" {
		t.Fatalf("locked exit %d %v", exit, reply)
	}
	if after, _ := os.ReadFile(c.doc); string(after) != string(before) {
		t.Fatal("a refused apply wrote")
	}
}

func TestPairReviewApplyUsage(t *testing.T) {
	c := newCLI(t, "x\n")
	for _, args := range [][]string{{"--json"}, {"--if-hash"}, {"x", "y", "--if-hash", "h"}} {
		var out, errout bytes.Buffer
		if got := RunCLI(append([]string{c.edits}, args...), func(string) string { return "" }, &out, &errout); got != 2 {
			t.Errorf("%v: exit %d, want usage 2", args, got)
		}
	}
}

// pair review round recovers a lost apply reply: the stored Body and Round of
// this file, Current while the file is still the round's result.
func TestPairReviewRoundRecoversALostReply(t *testing.T) {
	c := newCLI(t, "The quick fox.\n")
	env := map[string]string{"PAIR_DATA_DIR": c.data, "PAIR_TAG": "t", "PAIR_REVIEW_ROUND_PATH": c.round}
	round := func(file string) map[string]json.RawMessage {
		var out, errout bytes.Buffer
		if code := RunRoundCLI([]string{file, "--json"}, func(k string) string { return env[k] }, &out, &errout); code != 0 {
			t.Fatalf("round exit %d: %s %s", code, out.String(), errout.String())
		}
		var reply map[string]json.RawMessage
		json.Unmarshal(out.Bytes(), &reply)
		return reply
	}
	if r := round(c.doc); string(r["Round"]) != "null" {
		t.Fatalf("no round yet: %v", r)
	}
	_, applied, _ := c.run(`[{"old":"quick","new":"slow"}]`, "--if-hash", c.hash(), "--json")
	r := round(c.doc)
	if string(r["Body"]) != string(applied["Body"]) || string(r["Current"]) != "true" {
		t.Fatalf("recovered %v, applied %v", r, applied)
	}
	os.WriteFile(c.doc, []byte("operator moved on\n"), 0o644)
	if r := round(c.doc); string(r["Current"]) != "false" {
		t.Fatalf("a changed file still reads current: %v", r)
	}
	other := filepath.Join(filepath.Dir(c.doc), "other.md")
	os.WriteFile(other, []byte("x\n"), 0o644)
	if r := round(other); string(r["Round"]) != "null" {
		t.Fatalf("another file's round: %v", r)
	}
	var out, errout bytes.Buffer
	if code := RunRoundCLI([]string{c.doc}, func(string) string { return "" }, &out, &errout); code != 2 {
		t.Fatalf("round without --json: exit %d", code)
	}
}

// A reply with no edits (pair#690, the shape ariadne#453's xx-fix sends):
// both wire forms exit 0 with an empty Round and no Body, leave the document
// alone, and record a round the pane can see.
func TestPairReviewApplyWithNoRecordsIsAnEmptyRound(t *testing.T) {
	for _, edits := range []string{`[]`, `{"records":[]}`} {
		c := newCLI(t, "The quick fox.\n")
		h := c.hash()
		exit, reply, _ := c.run(edits, "--if-hash", h, "--json")
		var round sessionsync.Round
		json.Unmarshal(reply["Round"], &round)
		if exit != 0 || code(reply) != "ok" || reply["Body"] != nil || round.ResultHash != h || len(round.Spans) != 0 {
			t.Fatalf("%s: exit %d reply %v", edits, exit, reply)
		}
		if c.hash() != h {
			t.Fatalf("%s: an empty round changed the document", edits)
		}
		if a, err := sessionsync.ReadRoundArtifact(c.round); err != nil || a == nil || a.Body != "" || a.Round.ResultHash != h {
			t.Fatalf("%s: artifact %+v err %v", edits, a, err)
		}
	}
}
