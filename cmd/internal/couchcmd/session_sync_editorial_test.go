package couchcmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/editorial"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

func TestParseEditorialMigrationClosedForms(t *testing.T) {
	hash := strings.Repeat("a", 64)
	for _, args := range [][]string{
		{"migrate-editorial", "file"},
		{"migrate-editorial", "file", "--json"},
		{"migrate-editorial", "file", "--confirm", "--if-hash", hash, "--json"},
	} {
		if _, err := parseSessionSyncCLI(append([]string{"--session-sync"}, args...)); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	for _, args := range [][]string{
		{"migrate-editorial", "file", "payload"},
		{"migrate-editorial", "file", "--body", "payload"},
		{"migrate-editorial", "file", "--all"},
		{"migrate-editorial", "file", "--accept-gap"},
		{"migrate-editorial", "file", "--confirm"},
		{"migrate-editorial", "file", "--if-hash", hash},
		{"migrate-editorial", "file", "--confirm", "--if-hash", "short"},
		{"migrate-editorial", "file", "--confirm", "--confirm", "--if-hash", hash},
	} {
		if _, err := parseSessionSyncCLI(append([]string{"--session-sync"}, args...)); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestEditorialHelpNamesPolicyAndMigration(t *testing.T) {
	out, _, code := runPublicRT(newRT(t), "--help")
	if code != 0 {
		t.Fatal(code)
	}
	for _, want := range []string{"migrate-editorial FILE", "--confirm --if-hash H", "document byte offsets", "operator-owned"} {
		if !strings.Contains(out, want) {
			t.Errorf("help omits %q", want)
		}
	}
}

func TestRunEditorialMigrationCLI(t *testing.T) {
	rt, file := sessionSyncRT(t)
	source := "---\ntype: session-sync\nowner: ops:0\ncollaboration-mode: {default: discussion, chat: [Notes]}\n---\n# Notes\nhello\n"
	if err := os.WriteFile(file, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	if code, out, errout := runSS(t, rt, "register", file, "--owner", "ops:0", "--json"); code != 0 {
		t.Fatalf("register %d %s %s", code, out, errout)
	}
	preview := func() sessionsync.EditorialMigrationResult {
		t.Helper()
		code, out, errout := runSS(t, rt, "migrate-editorial", file, "--json")
		if code != 0 {
			t.Fatalf("preview %d %s %s", code, out, errout)
		}
		var response struct {
			Code      string
			Migration sessionsync.EditorialMigrationResult
		}
		if err := json.Unmarshal([]byte(out), &response); err != nil {
			t.Fatal(err)
		}
		if response.Code != "ok" || len(response.Migration.BeforeHash) != 64 || response.Migration.Applied {
			t.Fatalf("preview %s", out)
		}
		return response.Migration
	}
	p := preview()
	want, err := editorial.PlanMigration([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if p.Candidate != string(want.After) {
		t.Fatalf("candidate is not exact transform: %q", p.Candidate)
	}
	if raw, _ := os.ReadFile(file); string(raw) != source {
		t.Fatal("preview mutated source")
	}
	if code, out, _ := runSS(t, rt, "migrate-editorial", file); code != 0 || !strings.Contains(out, p.BeforeHash) || !strings.Contains(out, "candidate:") {
		t.Fatalf("text preview %d %s", code, out)
	}
	if err := os.WriteFile(file, []byte(source+"new\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := runSS(t, rt, "migrate-editorial", file, "--confirm", "--if-hash", p.BeforeHash, "--json"); code != sessionsync.ExitStale || !strings.Contains(out, `"Code":"stale"`) {
		t.Fatalf("stale %d %s", code, out)
	}
	p = preview()
	if err := os.WriteFile(file+".lock", []byte("operator"), 0600); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := runSS(t, rt, "migrate-editorial", file, "--confirm", "--if-hash", p.BeforeHash, "--json"); code != sessionsync.ExitOperatorLock || !strings.Contains(out, "operator-lock") {
		t.Fatalf("held %d %s", code, out)
	}
	if err := os.Remove(file + ".lock"); err != nil {
		t.Fatal(err)
	}
	code, out, errout := runSS(t, rt, "migrate-editorial", file, "--confirm", "--if-hash", p.BeforeHash, "--json")
	if code != 0 {
		t.Fatalf("confirm %d %s %s", code, out, errout)
	}
	var response struct {
		Migration sessionsync.EditorialMigrationResult
	}
	if err := json.Unmarshal([]byte(out), &response); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(file)
	if !response.Migration.Applied || string(raw) != p.Candidate {
		t.Fatalf("confirm %s file %q", out, raw)
	}
}
