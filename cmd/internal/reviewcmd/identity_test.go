package reviewcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func identityRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"config", "user.name", "Test"}, {"config", "user.email", "test@example.com"}} {
		identityGit(t, dir, args...)
	}
	os.WriteFile(filepath.Join(dir, "doc.md"), []byte("first\n"), 0600)
	identityGit(t, dir, "add", ".")
	identityGit(t, dir, "commit", "-qm", "initial")
	identityGit(t, dir, "checkout", "-qb", "review/doc")
	return dir
}
func identityGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}
func TestResolveIdentityHistoryAndReceipt(t *testing.T) {
	dir := identityRepo(t)
	rt := NewOSRuntime()
	got := resolveIdentity(rt, dir, "", "")
	if got.Status != "missing" {
		t.Fatalf("first open: %+v", got)
	}
	got = resolveIdentity(rt, dir, "doc.md", got.Head)
	if got.Status != "resolved" || got.File != "doc.md" {
		t.Fatalf("receipt: %+v", got)
	}
	got = resolveIdentity(rt, dir, "doc.md", "stale")
	if got.Status != "invalid" {
		t.Fatalf("stale receipt: %+v", got)
	}
	os.WriteFile(filepath.Join(dir, "doc.md"), []byte("second\n"), 0600)
	identityGit(t, dir, "commit", "-qam", "review(doc): human r1")
	identityGit(t, dir, "commit", "--allow-empty", "-qm", "review(doc): agent r2\n\n[]")
	got = resolveIdentity(rt, dir, "", "")
	if got.Status != "resolved" || got.File != "doc.md" || got.AgentRound != 2 || strings.TrimSpace(got.LatestAgentBody) != "[]" || got.LatestAgentCommit == "" {
		t.Fatalf("history: %+v", got)
	}
	os.WriteFile(filepath.Join(dir, "other.md"), []byte("other"), 0600)
	identityGit(t, dir, "add", ".")
	identityGit(t, dir, "commit", "-qm", "review(unrelated): agent r99")
	got = resolveIdentity(rt, dir, "", "")
	if got.Status != "resolved" || got.AgentRound != 2 {
		t.Fatalf("unrelated: %+v", got)
	}
	os.WriteFile(filepath.Join(dir, "other.md"), []byte("changed"), 0600)
	identityGit(t, dir, "commit", "-qam", "review(doc): human r3 — summary")
	if got = resolveIdentity(rt, dir, "", ""); got.Status != "ambiguous" {
		t.Fatalf("conflict: %+v", got)
	}
}
func TestRunOpenRefusesLiveWithoutEffects(t *testing.T) {
	rt := newFake()
	rt.sizes["/r/doc.md"] = 1
	rt.files["/dd/review-t.open"] = "123\n7\n"
	rt.alive["123"] = true
	var err bytes.Buffer
	code := RunOpen(OpenOptions{File: "/r/doc.md", Tag: "t", DataDir: "/dd", PairHome: "/h"}, rt, &err)
	if code == 0 || len(rt.killed) > 0 || len(rt.removed) > 0 || rt.spawn != nil {
		t.Fatalf("code=%d killed=%v removed=%v spawn=%v", code, rt.killed, rt.removed, rt.spawn)
	}
}

func TestPrepareRealFirstOpenReceipt(t *testing.T) {
	for _, tracked := range []bool{false, true} {
		t.Run(fmt.Sprint(tracked), func(t *testing.T) {
			dir := identityRepo(t)
			identityGit(t, dir, "checkout", "-q", "main")
			file := filepath.Join(dir, "new.md")
			os.WriteFile(file, []byte("new\n"), 0600)
			if tracked {
				identityGit(t, dir, "add", ".")
				identityGit(t, dir, "commit", "-qm", "track")
			}
			_, source, _, _ := runtime.Caller(0)
			home := filepath.Clean(filepath.Join(filepath.Dir(source), "../../.."))
			data := t.TempDir()
			opts := ReadinessOptions{File: file, Prepare: true, PairHome: home, DataDir: data, Tag: "t", SessionID: "sid"}
			var out bytes.Buffer
			if code := RunReadiness(opts, NewOSRuntime(), &out, &out); code != 0 {
				t.Fatalf("prepare: %d %s", code, out.String())
			}
			raw, err := os.ReadFile(filepath.Join(data, "review-target-t.json"))
			if err != nil {
				t.Fatal(err)
			}
			var target targetDoc
			if err = json.Unmarshal(raw, &target); err != nil {
				t.Fatal(err)
			}
			if target.Identity == nil || target.Identity.File != "new.md" || target.Identity.Branch != "review/new" || target.Session != "sid" {
				t.Fatalf("receipt: %s", raw)
			}
			out.Reset()
			if code := RunReadiness(opts, NewOSRuntime(), &out, &out); code != 0 {
				t.Fatalf("explicit zero-history reselection: %d %s", code, out.String())
			}
		})
	}
}

func TestIdentityRejectsUnsafeAndMissing(t *testing.T) {
	for _, kind := range []string{"deleted", "symlink", "detached", "directory"} {
		t.Run(kind, func(t *testing.T) {
			dir := identityRepo(t)
			os.WriteFile(filepath.Join(dir, "doc.md"), []byte("round"), 0600)
			identityGit(t, dir, "commit", "-qam", "review(doc): human r1")
			switch kind {
			case "deleted":
				os.Remove(filepath.Join(dir, "doc.md"))
			case "symlink":
				os.Remove(filepath.Join(dir, "doc.md"))
				os.Symlink("outside.md", filepath.Join(dir, "doc.md"))
				os.WriteFile(filepath.Join(dir, "outside.md"), []byte("outside"), 0600)
			case "directory":
				os.Remove(filepath.Join(dir, "doc.md"))
				os.Mkdir(filepath.Join(dir, "doc.md"), 0700)
			case "detached":
				identityGit(t, dir, "checkout", "--detach", "-q")
			}
			if got := resolveIdentity(NewOSRuntime(), dir, "", ""); got.Status != "invalid" {
				t.Fatalf("unsafe identity: %+v", got)
			}
		})
	}
}
func FuzzClassifyIdentity(f *testing.F) {
	f.Add("review(doc): agent r1", "doc.md")
	f.Add("review(other): human r2", "../bad")
	f.Fuzz(func(t *testing.T, subject, path string) {
		out := classifyIdentity("review/doc", []identityRound{{commit: "hash", subject: subject, paths: []string{path}}})
		if out.Status == "resolved" && (!safeIdentityPath(out.File) || out.File != path) {
			t.Fatalf("unsafe classification: %+v", out)
		}
	})
}
func FuzzParseIdentityHistory(f *testing.F) {
	f.Add("\x00sha\x00review(doc): human r1\x00\x00\ndoc.md\x00")
	f.Add("\x00")
	f.Fuzz(func(t *testing.T, raw string) {
		rounds, err := parseIdentityHistory(raw)
		if err == nil {
			classifyIdentity("review/doc", rounds)
		}
	})
}

type disturbedIdentityRuntime struct {
	Runtime
	mode string
}

func (r *disturbedIdentityRuntime) GitContext(ctx context.Context, limit int, dir string, args ...string) (string, error) {
	if args[0] == "log" {
		switch r.mode {
		case "failure":
			return "", fmt.Errorf("injected Git read failure")
		case "oversize":
			return strings.Repeat("x", identityLimit+1), nil
		case "move":
			r.Runtime.(*fakeRuntime).identityBranch = "review/moved"
		case "timeout":
			<-ctx.Done()
			return "", ctx.Err()
		}
	}
	return r.Runtime.GitContext(ctx, limit, dir, args...)
}
func TestIdentityCollectorRefusesIncompleteObservation(t *testing.T) {
	for _, mode := range []string{"failure", "oversize", "move", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			rt := newFake()
			initIdentityFake(rt, "review/doc")
			started := time.Now()
			got := resolveIdentity(&disturbedIdentityRuntime{Runtime: rt, mode: mode}, "/repo", "", "")
			if got.Status != "invalid" || got.Diagnostic == "" {
				t.Fatalf("incomplete observation: %+v", got)
			}
			if time.Since(started) > 3*time.Second {
				t.Fatal("resolver exceeded bounded deadline")
			}
			if len(rt.wrote) > 0 || rt.spawn != nil {
				t.Fatal("resolver mutated runtime")
			}
		})
	}
}
func TestIdentityHistoryLimitAndFraming(t *testing.T) {
	record := "\x00sha\x00review(doc): human r1\x00\x00\ndoc.md\x00"
	if _, err := parseIdentityHistory(strings.Repeat(record, identityCommitLimit)); err != nil {
		t.Fatal(err)
	}
	if _, err := parseIdentityHistory(strings.Repeat(record, identityCommitLimit+1)); err == nil {
		t.Fatal("accepted excess history")
	}
	for _, bad := range []string{"bad", record[:len(record)-1], "\x00\x00subject\x00body\x00"} {
		if _, err := parseIdentityHistory(bad); err == nil {
			t.Fatalf("accepted malformed %q", bad)
		}
	}
}
func TestOSGitContextOutputCapAndCancellation(t *testing.T) {
	dir := identityRepo(t)
	rt := NewOSRuntime()
	if _, err := rt.GitContext(context.Background(), 8, dir, "log", "-1"); err == nil {
		t.Fatal("accepted excessive output")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := rt.GitContext(ctx, identityLimit, dir, "log"); err == nil {
		t.Fatal("accepted canceled read")
	}
}

func TestDefinitionContextValidationAndEcho(t *testing.T) {
	contextJSON := `{"repo":"/repo","branch":"review/doc","file":"doc.md","activation":"nonce"}`
	rt := newFake()
	var out bytes.Buffer
	code := RunDefinition(DefinitionOptions{RequestID: "request", Definition: "definition", DataDir: "/dd", Tag: "t", Context: json.RawMessage(contextJSON)}, rt, &out, &out)
	if code != 0 {
		t.Fatalf("context rejected: %s", out.String())
	}
	got := definitionOf(t, rt, "t")
	if string(got.Context) != contextJSON {
		t.Fatalf("context changed: %s", got.Context)
	}
	for _, bad := range []string{`null`, `[]`, `{}`, `{"repo":"/repo","branch":"review/doc","file":"doc.md","activation":""}`, `bad`} {
		rt = newFake()
		code = RunDefinition(DefinitionOptions{RequestID: "r", Definition: "d", DataDir: "/dd", Context: json.RawMessage(bad)}, rt, &out, &out)
		if code == 0 || len(rt.wrote) > 0 {
			t.Fatalf("invalid context published: %s", bad)
		}
	}
}

func TestDefinitionCLIContextFlags(t *testing.T) {
	data := t.TempDir()
	env := func(key string) string {
		switch key {
		case "PAIR_DATA_DIR":
			return data
		case "PAIR_TAG":
			return "t"
		}
		return ""
	}
	ctx := `{"repo":"/repo","branch":"review/doc","file":"doc.md","activation":"nonce"}`
	for _, args := range [][]string{{"--context", ctx, "--term", "word", "r", "meaning"}, {"--term", "word", "--context", ctx, "r", "meaning"}} {
		var out bytes.Buffer
		if code := RunDefinitionCLI(args, env, &out, &out); code != 0 {
			t.Fatalf("flags rejected: %d %s", code, out.String())
		}
		raw, err := os.ReadFile(filepath.Join(data, "review-definition-result-t.json"))
		if err != nil {
			t.Fatal(err)
		}
		var d definitionDoc
		json.Unmarshal(raw, &d)
		if string(d.Context) != ctx || d.Term != "word" {
			t.Fatalf("context lost: %s", raw)
		}
	}
	for _, args := range [][]string{{"--context"}, {"--context", "", "r", "d"}, {"--context", "bad", "r", "d"}, {"--context", ctx, "--context", ctx, "r", "d"}} {
		var out bytes.Buffer
		if RunDefinitionCLI(args, env, &out, &out) == 0 {
			t.Fatalf("accepted bad flags: %v", args)
		}
	}
}

func TestPrepareRejectsAmbiguousBeforeTracking(t *testing.T) {
	rt := newFake()
	initIdentityFake(rt, "review/doc")
	rt.classify = "track"
	rt.identityHistory = "\x00sha\x00review(doc): human r1\x00\x00\ndoc.md\x00other.md\x00"
	rt.gitFn = gitScript(map[string]struct {
		out string
		err error
	}{"rev-parse": {out: "/repo\n"}, "branch": {out: "review/doc\n"}, "ls-files": {err: fmt.Errorf("untracked")}})
	var out bytes.Buffer
	code := RunReadiness(ReadinessOptions{File: "/repo/new.md", Prepare: true, PairHome: "/h"}, rt, &out, &out)
	if code == 0 || gitCalled(rt, "add") || gitCalled(rt, "commit") || gitCalled(rt, "checkout") {
		t.Fatalf("ambiguous prepare mutated: code=%d calls=%v", code, rt.gitCalls)
	}
}
