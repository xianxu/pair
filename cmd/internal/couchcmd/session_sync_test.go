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
		{[]string{"write", "--section", "--odd heading", "b", "f.md", "--fold", "a, b", "--accept-edits"},
			sessionSyncArgs{verb: "write", section: "--odd heading", bodyFile: "b", file: "f.md", fold: []string{"a", "b"}, acceptEdits: true}},
		{[]string{"write", "--all", "b", "f.md"}, sessionSyncArgs{verb: "write", all: true, bodyFile: "b", file: "f.md"}},
		{[]string{"release", "f.md", "--fold", "x"}, sessionSyncArgs{verb: "release", file: "f.md", fold: []string{"x"}}},
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
		{"write", "b", "f"}, {"write", "--all", "--section", "h", "b", "f"}, {"write", "--all", "f"},
		{"write", "--section"}, {"release", "f", "--fold"}, {"status", "--json", "--json", "f"},
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
	if st["LastWrite"] != nil || st["Holder"] != "free" || st["Owner"] != "ops:0" || st["File"] != file {
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
	code, _, errout := runSS(t, rt, "write", "--section", "Needs you", body, file)
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
	code, out, errout := runSS(t, rt, "write", "--section", "Needs you", body, file, "--fold", "d1")
	if code != 0 || !strings.Contains(out, "wrote ## Needs you") || !strings.Contains(out, "folded d1") {
		t.Fatalf("exit %d out %q err %q", code, out, errout)
	}
	if _, out, _ = runSS(t, rt, "status", file); !strings.Contains(out, "owed: 0") {
		t.Fatalf("status text %q", out)
	}
}
