package couchcmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

func TestParseSessionSyncFrontmatter(t *testing.T) {
	for _, verb := range []string{"read", "write", "diff", "status", "register"} {
		for mask := 0; mask < 8; mask++ {
			args := []string{"--session-sync", verb}
			scopes := 0
			if mask&1 != 0 {
				args = append(args, "--frontmatter")
				scopes++
			}
			if mask&2 != 0 {
				args = append(args, "--all")
				scopes++
			}
			if mask&4 != 0 {
				args = append(args, "--section", "--literal heading")
				scopes++
			}
			if verb == "write" {
				args = append(args, "body", "file", "--if-hash", "h")
			} else {
				args = append(args, "file")
			}
			_, err := ParseCLI(args, couchcore.Operations())
			valid := (verb == "read" || verb == "write") && scopes == 1
			if verb == "diff" || verb == "status" {
				valid = scopes == 0
			}
			if (err == nil) != valid {
				t.Fatalf("args %v: valid=%v err=%v", args, valid, err)
			}
		}
	}
	if _, err := ParseCLI([]string{"--session-sync", "read", "--frontmatter", "--frontmatter", "file"}, couchcore.Operations()); err == nil {
		t.Fatal("duplicate scope accepted")
	}
}

func TestSessionSyncFrontmatterCLI(t *testing.T) {
	rt, file := sessionSyncRT(t)
	call := func(args ...string) string {
		t.Helper()
		code, out, errout := runSS(t, rt, args...)
		if code != 0 {
			t.Fatalf("%v exit%d %s %s", args, code, out, errout)
		}
		return out
	}
	original, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(original), "old", "operator unread", 1)
	if err := os.WriteFile(file, []byte(edited), 0600); err != nil {
		t.Fatal(err)
	}
	var response struct{ Read sessionsync.ReadResult }
	if err := json.Unmarshal([]byte(call("read", "--frontmatter", file, "--json")), &response); err != nil {
		t.Fatal(err)
	}
	if response.Read.Hash == "" || strings.Contains(response.Read.Body, "##") {
		t.Fatalf("read %+v", response)
	}
	body := filepath.Join(t.TempDir(), "frontmatter.md")
	next := strings.Replace(response.Read.Body, "owner: ops:0", "owner: ops:0\ndiscussions: [new]", 1)
	if err := os.WriteFile(body, []byte(next+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	call("write", "--frontmatter", body, file, "--if-hash", response.Read.Hash)
	current, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(current) != next+edited[len(response.Read.Body):] {
		t.Fatalf("body changed %q", current)
	}
	diff := call("diff", file)
	if !strings.Contains(diff, "+operator unread") || strings.Contains(diff, "+discussions:") {
		t.Fatalf("diff %q", diff)
	}
	if code, _, _ := runSS(t, rt, "write", "--frontmatter", body, file, "--if-hash", response.Read.Hash); code != sessionsync.ExitStale {
		t.Fatalf("stale exit %d", code)
	}
	if err := os.WriteFile(file+".lock", []byte("holder: operator\n"), 0600); err != nil {
		t.Fatal(err)
	}
	call("read", "--frontmatter", file)
	if code, _, _ := runSS(t, rt, "write", "--frontmatter", body, file, "--if-hash", sessionsync.Hash(next)); code != sessionsync.ExitOperatorLock {
		t.Fatalf("lock exit %d", code)
	}
}

func TestSessionSyncFrontmatterHelp(t *testing.T) {
	var out strings.Builder
	usage(&out)
	text := out.String()
	for _, form := range []string{"read --frontmatter FILE", "write --frontmatter BODYFILE FILE --if-hash H"} {
		if !strings.Contains(text, form) {
			t.Errorf("help missing %q", form)
		}
	}
}
