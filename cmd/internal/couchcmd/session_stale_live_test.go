package couchcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchsingleton"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

// This recurring CLI contract is opt-in and never builds binaries inside a test.
// See atlas/couch.md for the command and the changes that require rerunning it.
func TestSessionStaleLiveConformance(t *testing.T) {
	if os.Getenv("PAIR_LIVE_SESSION_STALE") != "1" {
		t.Skip("set PAIR_LIVE_SESSION_STALE=1 and provide already-built Couch and tl-lead binaries")
	}
	couchBin, leadBin := os.Getenv("PAIR_LIVE_COUCH_BIN"), os.Getenv("PAIR_LIVE_TL_LEAD_BIN")
	for _, bin := range []string{couchBin, leadBin} {
		if !filepath.IsAbs(bin) {
			t.Fatalf("already-built binary must be absolute: %q", bin)
		}
		if info, err := os.Stat(bin); err != nil || info.Mode()&0111 == 0 {
			t.Fatalf("executable %q: %v", bin, err)
		}
	}
	root := testpair.ShortRoot(t)
	store, pair, identity := filepath.Join(root, "store"), filepath.Join(root, "pair"), filepath.Join(root, "identity")
	binDir, repo := filepath.Join(root, "bin"), filepath.Join(root, "fleet", "fixture")
	for _, dir := range []string{binDir, filepath.Join(repo, "workshop", "issues")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(couchBin, filepath.Join(binDir, "couch")); err != nil {
		t.Fatal(err)
	}
	env := testpair.Environment(t, root, "PATH="+binDir+":/usr/bin:/bin", "TMPDIR="+root+"/", "XDG_DATA_HOME="+filepath.Join(root, "data"), "COUCH_ISOLATED_ROOT="+root, "COUCH_STORE_DIR="+store, "PAIR_DATA_DIR="+pair, "COUCH_IDENTITY_DIR="+identity, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
	run := func(bin string, args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, bin, args...)
		cmd.Env, cmd.Dir = env, repo
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", bin, args, err, out)
		}
		return out
	}
	adopt := []string{"--adopt-store", store, "--pair-data", pair, "--identity-dir", identity}
	var report couchsingleton.Report
	if err := json.Unmarshal(run(couchBin, adopt...), &report); err != nil || report.Digest == "" {
		t.Fatalf("adoption preview: %+v, %v", report, err)
	}
	run(couchBin, append(adopt, "--apply", report.Digest)...)
	run("/usr/bin/git", "init", "-q")
	run("/usr/bin/git", "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-q", "--allow-empty", "-m", "fixture landing")
	run("/usr/bin/git", "update-ref", "refs/remotes/origin/main", "HEAD")

	r := newSessionStaleRig(t)
	r.watcher.sync = sessionsync.New(store)
	r.file = filepath.Join(repo, "status.md")
	write := func(stamp string) {
		t.Helper()
		doc := "---\ntype: session-sync\ncouch-owner: " + r.hub.Slot + "\n---\n\n## Threads\n\n" + stamp + "unread operator body\n"
		if err := os.WriteFile(r.file, []byte(doc), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("")
	if _, err := r.watcher.sync.Register(r.file, r.hub.Slot); err != nil {
		t.Fatal(err)
	}
	// Establish a baseline, then leave an operator edit unread. Section checks
	// must preserve the existing shadow and state, not just the document bytes.
	if _, err := r.watcher.sync.Read(r.file, ""); err != nil {
		t.Fatal(err)
	}
	write("operator addition\n\n")
	r.watcher.check = func(ctx context.Context, file string) (string, error) {
		r.checks.Add(1)
		cmd := exec.CommandContext(ctx, leadBin, "status", "check", file)
		cmd.Env, cmd.Dir = env, repo
		return runStaleCommand(cmd)
	}
	// The admitted wrapper is real; replace its composer seam with the real
	// endpoint transport and a retained submitted receipt, without native agents.
	endpointNS := t.TempDir()
	socket, err := couchmessage.EndpointSocket(endpointNS, r.hub)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	reserved := ""
	stored := map[string]couchmessage.Message{}
	server, err := couchmessage.StartServer(context.Background(), socket, func(_ context.Context, raw []byte) ([]byte, error) {
		var request couchmessage.EndpointRequest
		if err := json.Unmarshal(raw, &request); err != nil {
			return nil, err
		}
		if err := couchmessage.ValidateEndpointRequest(request); err != nil {
			return nil, err
		}
		mu.Lock()
		defer mu.Unlock()
		response := couchmessage.EndpointResponse{}
		switch request.Op {
		case "observe":
			response.Observation = couchmessage.Observation{Sequence: 1, LastActivity: time.Now().Add(-time.Minute)}
		case "reserve":
			if reserved != "" || request.Sequence != 1 {
				response.Error = "reservation refused"
			} else {
				reserved = request.ID
			}
		case "release":
			if reserved == request.ID {
				reserved = ""
			}
		case "commit":
			if reserved != request.ID || request.Message == nil {
				response.Error = "not reserved"
			} else {
				stored[request.ID] = *request.Message
				response.Receipt = &couchmessage.Receipt{Message: *request.Message, Status: couchmessage.Submitted}
				r.endpoint.deliveries <- *request.Message
			}
		case "status":
			message, ok := stored[request.ID]
			if !ok {
				response.Error = "not retained"
			} else {
				response.Receipt = &couchmessage.Receipt{Message: message, Status: couchmessage.Submitted}
			}
		default:
			return nil, fmt.Errorf("unexpected endpoint operation %q", request.Op)
		}
		return json.Marshal(response)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Close() })
	r.world.endpoints[r.hub] = couchmessage.RemoteEndpoint{Namespace: endpointNS, Binding: r.hub}

	snapshot := func() map[string]string {
		t.Helper()
		result := map[string]string{}
		if err := filepath.WalkDir(filepath.Join(store, "session-sync"), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || strings.HasSuffix(path, ".flock") {
				return nil
			}
			body, err := os.ReadFile(path)
			if err == nil {
				result[path] = string(body)
			}
			return err
		}); err != nil {
			t.Fatal(err)
		}
		body, err := os.ReadFile(r.file)
		if err != nil {
			t.Fatal(err)
		}
		result[r.file] = string(body)
		return result
	}
	assertNeutralTick := func() {
		t.Helper()
		before := snapshot()
		r.watcher.tick(context.Background())
		if after := snapshot(); !reflect.DeepEqual(before, after) {
			t.Fatalf("reminder changed document or unread baseline:\nbefore %v\nafter %v", before, after)
		}
	}
	assertSilent := func() {
		t.Helper()
		select {
		case m := <-r.endpoint.deliveries:
			t.Fatalf("unexpected reminder %+v", m)
		default:
		}
	}
	assertNeutralTick()
	first := r.waitMessage(t)
	if first.From != r.hub || first.To != r.hub || !strings.Contains(first.Body, "stale: ## Threads (unstamped)") {
		t.Fatalf("real CLI reminder %+v", first)
	}
	assertNeutralTick()
	assertSilent()
	if r.checks.Load() != 1 {
		t.Fatal("CLI repeated before half-hour allowance")
	}
	r.now = r.now.Add(30 * time.Minute)
	assertNeutralTick()
	r.waitMessage(t)
	write("_updated " + time.Now().Format("2006-01-02 15:04") + "_\n\n")
	r.now = r.now.Add(30 * time.Minute)
	assertNeutralTick()
	assertSilent()
	write("")
	r.now = r.now.Add(30 * time.Minute)
	if err := os.WriteFile(r.file+".lock", []byte("holder: operator\n"), 0600); err != nil {
		t.Fatal(err)
	}
	checks := r.checks.Load()
	assertNeutralTick()
	assertSilent()
	if r.checks.Load() != checks {
		t.Fatal("checked operator-held file")
	}
	if err := os.Remove(r.file + ".lock"); err != nil {
		t.Fatal(err)
	}
	assertNeutralTick()
	r.waitMessage(t)
	if r.logs.Load() != 0 {
		t.Fatalf("real checker logged %d failures", r.logs.Load())
	}
}
