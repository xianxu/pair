package couchcmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

func TestParseSessionSyncForms(t *testing.T) {
	for _, c := range []struct {
		args []string
		want sessionSyncArgs
	}{
		{[]string{"diff", "f.md"}, sessionSyncArgs{verb: "diff", file: "f.md"}},
		{[]string{"status", "--json", "f.md"}, sessionSyncArgs{verb: "status", file: "f.md", json: true}},
		{[]string{"write", "--section", "--odd heading", "b", "f.md", "--fold", "a, b", "--if-hash", "h1", "--accept-gap"},
			sessionSyncArgs{verb: "write", section: "--odd heading", bodyFile: "b", file: "f.md", ifHash: "h1", fold: []string{"a", "b"}, accepts: sessionsync.Accepts{Gap: true}}},
		{[]string{"write", "--all", "b", "f.md", "--if-hash", "h"}, sessionSyncArgs{verb: "write", all: true, bodyFile: "b", file: "f.md", ifHash: "h"}},
		{[]string{"read", "--section", "Needs you", "f.md", "--json"}, sessionSyncArgs{verb: "read", section: "Needs you", file: "f.md", json: true}},
		{[]string{"read", "--all", "f.md"}, sessionSyncArgs{verb: "read", all: true, file: "f.md"}},
		// release is a retired no-op that still takes its old flags.
		{[]string{"release", "f.md", "--fold", "x"}, sessionSyncArgs{verb: "release", file: "f.md", fold: []string{"x"}}},
		{[]string{"release", "--accept-edits", "--accept-gap", "f.md"}, sessionSyncArgs{verb: "release", file: "f.md", accepts: sessionsync.Accepts{Gap: true}}},
	} {
		inv, err := ParseCLI(append([]string{"--session-sync"}, c.args...), couchcore.Operations())
		if err != nil {
			t.Fatalf("%v: %v", c.args, err)
		}
		if inv.kind != cliSessionSync || !reflect.DeepEqual(*inv.sessionSync, c.want) {
			t.Fatalf("%v: got %+v", c.args, inv.sessionSync)
		}
	}
	for _, bad := range [][]string{
		{}, {"nope", "f"}, {"diff"}, {"diff", "a", "b"}, {"status", "--fold", "x", "f"},
		{"write", "b", "f", "--if-hash", "h"}, {"write", "--all", "--section", "h", "b", "f", "--if-hash", "h"},
		{"write", "--all", "f", "--if-hash", "h"}, {"write", "--section"}, {"release", "f", "--fold"},
		{"status", "--json", "--json", "f"},
		// An unconditional write, and the retired --accept-edits.
		{"write", "--all", "b", "f"}, {"write", "--all", "b", "f", "--if-hash", "h", "--accept-edits"},
		{"read", "f"}, {"read", "--all", "--section", "h", "f"}, {"read", "--all", "f", "--if-hash", "h"},
	} {
		if _, err := ParseCLI(append([]string{"--session-sync"}, bad...), couchcore.Operations()); err == nil {
			t.Fatalf("%v must be a usage error", bad)
		}
	}
}

// sessionSyncRT is a runtime whose namespace is a recording Couch store,
// plus a session-sync file owned by ops:0.
func sessionSyncRT(t *testing.T) (testRT, string) {
	rt := newRT(t)
	store := t.TempDir()
	ns, err := couchcore.ResolveCouchNamespace(store, "/")
	if err != nil {
		t.Fatal(err)
	}
	rt.namespace = ns
	ledger := &sessionsync.Ledger{Dir: sessionsync.LedgerDir(ns.Dir())}
	if err := ledger.MarkRecording(sessionsync.Recording{PID: 1, Started: time.Now()}); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "tl.md")
	doc := "---\ntype: session-sync\nowner: ops:0\n---\n\n## Needs you\n\nold\n"
	os.WriteFile(file, []byte(doc), 0o644)
	os.WriteFile(sessionsync.PathsOf(file).Shadow, []byte(doc), 0o644)
	// After the shadow: its mtime is the owed horizon of a file never
	// written through this verb.
	if err := ledger.Append("ops:0", sessionsync.Entry{ID: "d1", From: "pair:5", At: time.Now().Add(time.Second), Head: "done"}); err != nil {
		t.Fatal(err)
	}
	return rt, file
}

func runSS(t *testing.T, rt testRT, args ...string) (int, string, string) {
	var out, errout bytes.Buffer
	code := RunWithRuntime(append([]string{"--session-sync"}, args...), strings.NewReader(""), &out, &errout, rt)
	return code, out.String(), errout.String()
}

func TestSessionSyncStatusJSONIsTheAgreedEnvelope(t *testing.T) {
	rt, file := sessionSyncRT(t)
	code, out, errout := runSS(t, rt, "status", "--json", file)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errout)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if got["Code"] != "ok" || got["Error"] != "" {
		t.Fatalf("got %v", got)
	}
	st := got["Status"].(map[string]any)
	keys := []string{}
	for k := range st {
		keys = append(keys, k)
	}
	for _, k := range []string{"File", "Owner", "Holder", "LastWrite", "Owed"} {
		if _, ok := st[k]; !ok {
			t.Fatalf("Status lacks %s: %v", k, keys)
		}
	}
	if st["LastWrite"] != nil || st["Holder"] != "" || st["Owner"] != "ops:0" || st["File"] != file {
		t.Fatalf("status %v", st)
	}
	owed := st["Owed"].([]any)
	row := owed[0].(map[string]any)
	if len(owed) != 1 || row["ID"] != "d1" || row["Kind"] != "delivery" || row["From"] != "pair:5" {
		t.Fatalf("owed %v", owed)
	}
	if _, err := time.Parse(time.RFC3339, row["At"].(string)); err != nil {
		t.Fatalf("At is RFC3339: %v", err)
	}
}

func TestSessionSyncRefusalsExitNonZeroWithACode(t *testing.T) {
	rt, file := sessionSyncRT(t)
	os.WriteFile(file+".lock", []byte("holder: operator\n"), 0o644)
	body := filepath.Join(t.TempDir(), "body")
	os.WriteFile(body, []byte("new"), 0o644)
	code, _, errout := runSS(t, rt, "write", "--section", "Needs you", body, file, "--if-hash", sessionsync.Hash("\nold\n"))
	if code != sessionsync.ExitOperatorLock || !strings.Contains(errout, "operator holds the lock") {
		t.Fatalf("exit %d: %s", code, errout)
	}
	os.Remove(file + ".lock")
	os.WriteFile(file, []byte("plain\n"), 0o644)
	code, out, _ := runSS(t, rt, "status", "--json", file)
	var got struct {
		Code, Error string
		Status      any
	}
	json.Unmarshal([]byte(out), &got)
	if code == 0 || got.Code != "refused" || got.Status != nil {
		t.Fatalf("exit %d out %s", code, out)
	}
}

func TestSessionSyncWriteFoldsAndStatusTextShowsIt(t *testing.T) {
	rt, file := sessionSyncRT(t)
	_, out, _ := runSS(t, rt, "status", file)
	if !strings.Contains(out, "owed: 1") || !strings.Contains(out, "d1 from pair:5") {
		t.Fatalf("status text %q", out)
	}
	body := filepath.Join(t.TempDir(), "body")
	os.WriteFile(body, []byte("handled d1"), 0o644)
	_, out, _ = runSS(t, rt, "read", "--section", "Needs you", file)
	hash, rest, _ := strings.Cut(strings.TrimPrefix(out, "hash: "), "\n")
	if rest != "\nold\n" || hash != sessionsync.Hash(rest) {
		t.Fatalf("read text %q", out)
	}
	code, out, errout := runSS(t, rt, "write", "--section", "Needs you", body, file, "--fold", "d1", "--if-hash", hash)
	if code != 0 || !strings.Contains(out, "wrote ## Needs you") || !strings.Contains(out, "folded d1") ||
		!strings.Contains(out, "hash: "+sessionsync.Hash("\nhandled d1\n")) {
		t.Fatalf("exit %d out %q err %q", code, out, errout)
	}
	if _, out, _ = runSS(t, rt, "status", file); !strings.Contains(out, "owed: 0") {
		t.Fatalf("status text %q", out)
	}
}

func TestSessionSyncStaleAndReadJSON(t *testing.T) {
	rt, file := sessionSyncRT(t)
	code, out, _ := runSS(t, rt, "read", "--section", "Needs you", "--json", file)
	var got struct {
		Code string
		Read sessionsync.ReadResult
		Hash string
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil || code != 0 || got.Code != "ok" ||
		got.Read.Body != "\nold\n" || got.Read.Hash != sessionsync.Hash("\nold\n") || got.Read.Section != "Needs you" {
		t.Fatalf("exit %d out %s", code, out)
	}
	body := filepath.Join(t.TempDir(), "body")
	os.WriteFile(body, []byte("new"), 0o644)
	code, out, errout := runSS(t, rt, "write", "--section", "Needs you", body, file, "--if-hash", "0000000000000000")
	if code != sessionsync.ExitStale || out != "hash: "+got.Read.Hash+"\n" || !strings.Contains(errout, "changed since you read it") {
		t.Fatalf("exit %d out %q err %q", code, out, errout)
	}
}

func TestSessionSyncReleaseIsARetiredNoOp(t *testing.T) {
	rt, file := sessionSyncRT(t)
	before, _ := os.ReadFile(file)
	code, out, errout := runSS(t, rt, "release", file, "--accept-edits", "--fold", "d1")
	after, _ := os.ReadFile(file)
	if code != 0 || out != "" || !strings.Contains(errout, "retired") || string(before) != string(after) {
		t.Fatalf("exit %d out %q err %q", code, out, errout)
	}
	if _, out, _ = runSS(t, rt, "status", file); !strings.Contains(out, "owed: 1") || !strings.Contains(out, "lock: none") {
		t.Fatalf("release must fold nothing: %q", out)
	}
}
