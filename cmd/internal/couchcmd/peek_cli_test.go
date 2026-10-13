package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestRenderPeek(t *testing.T) {
	var out bytes.Buffer
	renderPeek(&out, couchcore.PeekResult{Ref: "pair:1", Tag: "couch-1", Agent: "claude",
		Lines: []string{"[Couch peer from pair:0; delivery abc]", "❯ "}, SentPrompts: "/d/log.md",
		Transcripts: []string{"/h/s.jsonl"}, Unavailable: []string{"terminal recording: x"}})
	want := "peek pair:1  agent claude  tag couch-1  source none\n--- recent terminal ---\n[Couch peer from pair:0; delivery abc]\n❯ \n---\n" +
		"sent prompts: /d/log.md\ntranscript: /h/s.jsonl\nunavailable: terminal recording: x\n"
	if out.String() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out.String(), want)
	}
	out.Reset()
	renderPeek(&out, couchcore.PeekResult{Ref: "pair:1", Tag: "couch-1"})
	if !strings.HasPrefix(out.String(), "peek pair:1  agent unknown") {
		t.Fatal(out.String())
	}
}

// TestPeekJSONIsTheResult: --json prints PeekResult itself, so agents read
// stable keys.
func TestPeekJSONIsTheResult(t *testing.T) {
	raw, err := json.Marshal(couchcore.PeekResult{Ref: "pair:1", Tag: "t", Lines: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]any
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"ref", "tag", "lines"} {
		if _, ok := keys[key]; !ok {
			t.Fatalf("missing %q in %s", key, raw)
		}
	}
}

func TestPeekPreviousAndFullCLI(t *testing.T) {
	inv, err := ParseCLI([]string{"--peek", "pair:2", "--previous", "--all", "--transcripts", "--json"}, couchcore.Operations())
	if err != nil {
		t.Fatal(err)
	}
	if inv.kind != cliPeek || inv.ref != "pair:2" || len(inv.args) != 4 {
		t.Fatalf("%+v", inv)
	}
	if fastPeek(inv, "/unused", &bytes.Buffer{}, func(context.Context, string, any, any) error {
		t.Fatal("recording request reached live tail")
		return nil
	}) {
		t.Fatal("full/previous request used live tail")
	}
	for _, args := range [][]string{
		{"--peek", "pair:2", "--previous", "--previous"},
		{"--peek", "pair:2", "--all", "--all"},
	} {
		if _, err := ParseCLI(args, couchcore.Operations()); err == nil {
			t.Fatalf("accepted duplicate flags: %q", args)
		}
	}
}

func TestPeekFullHelpNamesCosts(t *testing.T) {
	var out bytes.Buffer
	usage(&out)
	for _, text := range []string{"--previous", "--all", "recording [heavy]", "previous recording [seconds]"} {
		if !strings.Contains(out.String(), text) {
			t.Errorf("help lacks %q", text)
		}
	}
}
