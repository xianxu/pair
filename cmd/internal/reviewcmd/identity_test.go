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

func TestIdentitySnapshotBindsWorkingBytes(t *testing.T) {
	dir := identityRepo(t)
	rt := NewOSRuntime()
	head := identityGit(t, dir, "rev-parse", "HEAD")
	for _, body := range []string{"uncommitted\n", "no final newline", ""} {
		if err := os.WriteFile(filepath.Join(dir, "doc.md"), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		got := resolveIdentityWithSnapshot(rt, dir, "doc.md", head, true)
		if got.Status != "resolved" || got.Snapshot == nil || *got.Snapshot != body {
			t.Fatalf("snapshot: %+v", got)
		}
	}
}

type movingSnapshotRuntime struct{ Runtime }

func (r movingSnapshotRuntime) ReadIdentityFile(ctx context.Context, root, rel string, limit int) (string, error) {
	result, err := r.Runtime.ReadIdentityFile(ctx, root, rel, limit)
	r.Runtime.(*fakeRuntime).identityHead = "moved"
	return result, err
}
func TestIdentitySnapshotRejectsMovementAndSize(t *testing.T) {
	fake := newFake()
	initIdentityFake(fake, "review/doc")
	fake.files["/repo/doc.md"] = "A bytes"
	got := resolveIdentityWithSnapshot(movingSnapshotRuntime{fake}, "/repo", "", "", true)
	if got.Status != "invalid" || got.Snapshot != nil {
		t.Fatalf("moved snapshot published: %+v", got)
	}
	dir := identityRepo(t)
	head := identityGit(t, dir, "rev-parse", "HEAD")
	os.WriteFile(filepath.Join(dir, "doc.md"), []byte(strings.Repeat("x", identityLimit+1)), 0600)
	got = resolveIdentityWithSnapshot(NewOSRuntime(), dir, "doc.md", head, true)
	if got.Status != "invalid" || got.Snapshot != nil {
		t.Fatalf("oversize snapshot: %+v", got)
	}
}

func TestIdentitySnapshotRejectsNonTextAndEncodedOverflow(t *testing.T) {
	dir := identityRepo(t)
	head := identityGit(t, dir, "rev-parse", "HEAD")
	for _, body := range [][]byte{{0xff}, {'x', 0, 'y'}} {
		os.WriteFile(filepath.Join(dir, "doc.md"), body, 0600)
		got := resolveIdentityWithSnapshot(NewOSRuntime(), dir, "doc.md", head, true)
		if got.Status != "invalid" || got.Snapshot != nil {
			t.Fatalf("nontext accepted: %+v", got)
		}
	}
	os.WriteFile(filepath.Join(dir, "doc.md"), []byte(strings.Repeat("\x01", 2<<20)), 0600)
	var output, diagnostic bytes.Buffer
	code := RunReadinessCLI([]string{"--resolve", dir, "--selected", "doc.md", "--head", head, "--snapshot"}, func(string) string { return "" }, &output, &diagnostic)
	var got ReviewIdentity
	if code != 0 || json.Unmarshal(output.Bytes(), &got) != nil || got.Status != "invalid" || output.Len() > identityLimit {
		t.Fatalf("encoded overflow: code=%d len=%d result=%+v", code, output.Len(), got)
	}
}

func TestIdentityRejectsPausedCheckout(t *testing.T) {
	dir := identityRepo(t)
	filter := filepath.Join(t.TempDir(), "smudge.sh")
	script := "#!/bin/sh\nif [ -n \"$PAIR_TEST_PAUSE_DIR\" ]; then\n : > \"$PAIR_TEST_PAUSE_DIR/ready\"\n while [ ! -f \"$PAIR_TEST_PAUSE_DIR/release\" ]; do sleep 0.01; done\nfi\ncat\n"
	if err := os.WriteFile(filter, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	identityGit(t, dir, "config", "filter.pause.smudge", filter)
	identityGit(t, dir, "config", "filter.pause.clean", "cat")
	os.WriteFile(filepath.Join(dir, ".gitattributes"), []byte("zzz.txt filter=pause\n"), 0600)
	os.WriteFile(filepath.Join(dir, "zzz.txt"), []byte("A filter\n"), 0600)
	identityGit(t, dir, "add", ".")
	identityGit(t, dir, "commit", "-qm", "filter setup")
	os.WriteFile(filepath.Join(dir, "doc.md"), []byte("A bytes\n"), 0600)
	identityGit(t, dir, "commit", "-qam", "review(doc): human r1")
	identityGit(t, dir, "checkout", "-qb", "review/b")
	os.WriteFile(filepath.Join(dir, "doc.md"), []byte("B bytes\n"), 0600)
	os.WriteFile(filepath.Join(dir, "zzz.txt"), []byte("B filter\n"), 0600)
	identityGit(t, dir, "commit", "-qam", "other branch")
	identityGit(t, dir, "checkout", "-q", "review/doc")
	pause := t.TempDir()
	command := exec.Command("git", "-C", dir, "checkout", "-q", "review/b")
	command.Env = append(os.Environ(), "PAIR_TEST_PAUSE_DIR="+pause)
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.WriteFile(filepath.Join(pause, "release"), nil, 0600)
		if err := command.Wait(); err != nil {
			t.Errorf("checkout: %v %s", err, output.String())
		}
	})
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(pause, "ready")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("smudge did not pause")
		}
		time.Sleep(10 * time.Millisecond)
	}
	body, _ := os.ReadFile(filepath.Join(dir, "doc.md"))
	if string(body) != "B bytes\n" || identityGit(t, dir, "branch", "--show-current") != "review/doc" {
		t.Fatalf("fixture did not expose intermediate checkout: %q", body)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "index.lock")); err != nil {
		t.Fatalf("expected checkout-owned index lock: %v", err)
	}
	for _, snapshot := range []bool{false, true} {
		got := resolveIdentityWithSnapshot(NewOSRuntime(), dir, "", "", snapshot)
		if got.Status != "invalid" || got.Snapshot != nil {
			t.Fatalf("in-progress checkout admitted: %+v", got)
		}
	}
}

func TestIdentityRejectsPublishedIndexBeforeHEAD(t *testing.T) {
	dir := identityRepo(t)
	os.WriteFile(filepath.Join(dir, "doc.md"), []byte("A bytes\n"), 0600)
	identityGit(t, dir, "commit", "-qam", "review(doc): human r1")
	identityGit(t, dir, "checkout", "-qb", "review/b")
	os.WriteFile(filepath.Join(dir, "doc.md"), []byte("B bytes\n"), 0600)
	identityGit(t, dir, "commit", "-qam", "other branch")
	identityGit(t, dir, "checkout", "-q", "review/doc")
	// checkout publishes its destination index before updating HEAD. read-tree
	// reproduces that intermediate state with real Git and no remaining lock.
	identityGit(t, dir, "read-tree", "-u", "--reset", "review/b")
	if identityGit(t, dir, "branch", "--show-current") != "review/doc" {
		t.Fatal("HEAD changed")
	}
	got := resolveIdentityWithSnapshot(NewOSRuntime(), dir, "", "", true)
	if got.Status != "invalid" || got.Snapshot != nil {
		t.Fatalf("destination index with old HEAD admitted: %+v", got)
	}
}

func TestIdentityIndexAdmissionKeepsUnstagedEdits(t *testing.T) {
	dir := identityRepo(t)
	head := identityGit(t, dir, "rev-parse", "HEAD")
	os.WriteFile(filepath.Join(dir, "doc.md"), []byte("unstaged document\n"), 0600)
	got := resolveIdentityWithSnapshot(NewOSRuntime(), dir, "doc.md", head, true)
	if got.Status != "resolved" || got.Snapshot == nil || *got.Snapshot != "unstaged document\n" {
		t.Fatalf("unstaged edit rejected: %+v", got)
	}
	for _, path := range []string{"doc.md", "unrelated.txt", ".gitattributes"} {
		if path != "doc.md" {
			os.WriteFile(filepath.Join(dir, path), []byte("staged\n"), 0600)
		}
		identityGit(t, dir, "add", "--", path)
		got = resolveIdentityWithSnapshot(NewOSRuntime(), dir, "doc.md", head, true)
		if got.Status != "invalid" || got.Snapshot != nil || !strings.Contains(got.Diagnostic, "commit/unstage") {
			t.Fatalf("staged %s admitted: %+v", path, got)
		}
		identityGit(t, dir, "reset", "-q", "HEAD", "--", path)
	}
	got = resolveIdentityWithSnapshot(NewOSRuntime(), dir, "doc.md", head, true)
	if got.Status != "resolved" {
		t.Fatalf("retry after unstage: %+v", got)
	}
}

type changingIndexRuntime struct {
	Runtime
	checks       int
	lockOnSecond bool
}

func (r *changingIndexRuntime) IndexGeneration(ctx context.Context, path string) (string, error) {
	r.checks++
	if r.checks > 1 {
		if r.lockOnSecond {
			return "", fmt.Errorf("index operation in progress")
		}
		return "new-index-generation", nil
	}
	return "old-index-generation", nil
}
func TestIdentityRejectsIndexMovementDuringCollection(t *testing.T) {
	for _, locked := range []bool{false, true} {
		fake := newFake()
		initIdentityFake(fake, "review/doc")
		fake.files["/repo/doc.md"] = "A bytes"
		got := resolveIdentityWithSnapshot(&changingIndexRuntime{Runtime: fake, lockOnSecond: locked}, "/repo", "", "", true)
		if got.Status != "invalid" || got.Snapshot != nil {
			t.Fatalf("index movement admitted: %+v", got)
		}
	}
}

func TestIdentityUsesWorktreeLocalIndex(t *testing.T) {
	dir := identityRepo(t)
	linked := filepath.Join(t.TempDir(), "linked")
	identityGit(t, dir, "worktree", "add", "-q", "-b", "review/linked", linked)
	head := identityGit(t, linked, "rev-parse", "HEAD")
	got := resolveIdentityWithSnapshot(NewOSRuntime(), linked, "doc.md", head, true)
	if got.Status != "resolved" || got.Snapshot == nil {
		t.Fatalf("linked identity: %+v", got)
	}
	index := identityGit(t, linked, "rev-parse", "--path-format=absolute", "--git-path", "index")
	if err := os.WriteFile(index+".lock", []byte("writer"), 0600); err != nil {
		t.Fatal(err)
	}
	got = resolveIdentityWithSnapshot(NewOSRuntime(), linked, "doc.md", head, true)
	if got.Status != "invalid" || got.Snapshot != nil {
		t.Fatalf("linked index lock ignored: %+v", got)
	}
}
