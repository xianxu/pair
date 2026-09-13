package launcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
)

// Invoked only by couchcmd's acceptance test with the producer's original JSON.
func TestCouchOrientationTransportHelper(t *testing.T) {
	raw := os.Getenv("PAIR184_ACCEPT_PROFILE")
	if raw == "" {
		t.Skip("cross-package transport helper")
	}
	var identity struct {
		Tag string `json:"tag"`
	}
	if err := json.Unmarshal([]byte(raw), &identity); err != nil {
		t.Fatal(err)
	}
	args, _, err := ApplyCouchLaunchProfile(LaunchArgs{ForcedTag: identity.Tag}, raw)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	rt := newFakeRuntime()
	opts := baseOpts(args)
	opts.Env.DataDir = dir
	opts.Env.Home = dir
	if code, err := run(t, opts, rt); err != nil || code != 0 {
		t.Fatalf("create %d %v", code, err)
	}
	binary := os.Getenv("PAIR184_ACCEPT_BINARY")
	if binary == "" {
		binary = filepath.Join(dir, "pair-real")
		if out, err := exec.Command("go", "build", "-o", binary, "../../pair-go").CombinedOutput(); err != nil {
			t.Fatalf("build %v %s", err, out)
		}
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"pair":   "#!/bin/sh\nexec \"$PAIR_REAL\" \"$@\"\n",
		"codex":  "#!/bin/sh\nstty raw -echo\nexec \"$PAIR184_TEST_BINARY\" -test.run '^TestOrientationCaptureChild$'\n",
		"zellij": "#!/bin/sh\nexit 0\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0755); err != nil {
			t.Fatal(err)
		}
	}
	layout, err := os.ReadFile("../../../zellij/layouts/main-2.kdl")
	if err != nil {
		t.Fatal(err)
	}
	var stanza string
	re := regexp.MustCompile(`args "-c" (".*")`)
	for _, line := range strings.Split(string(layout), "\n") {
		if strings.Contains(line, "exec pair wrap") {
			m := re.FindStringSubmatch(line)
			if len(m) != 2 {
				t.Fatal("missing stanza")
			}
			stanza, err = strconv.Unquote(m[1])
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if stanza == "" {
		t.Fatal("missing agent pane")
	}
	master, tty, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer tty.Close()
	if err := pty.Setsize(tty, &pty.Winsize{Rows: 24, Cols: 80}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", stanza)
	cmd.Stdin = tty
	env := map[string]string{}
	for k, v := range rt.env {
		env[k] = v
	}
	env["PATH"] = dir + ":" + os.Getenv("PATH")
	env["HOME"] = dir
	env["PAIR_REAL"] = binary
	env["PAIR184_TEST_BINARY"] = executable
	env["PAIR184_ACCEPT_OUTPUT"] = os.Getenv("PAIR184_ACCEPT_OUTPUT")
	env["PAIR184_CAPTURE_CHILD"] = "1"
	env["ZELLIJ_SESSION_NAME"] = rt.launched
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("layout/wrapper %v\n%s", err, out)
	}
}

func TestOrientationCaptureChild(t *testing.T) {
	if os.Getenv("PAIR184_CAPTURE_CHILD") != "1" {
		t.Skip("wrapper child helper")
	}
	// Resolved Codex startup card and empty composer, cursor after the prompt.
	fmt.Print("\x1b[2J\x1b[1;1HOpenAI Codex (v0.154.0)\r\nmodel: test-model\r\ndirectory: /synthetic\x1b[20;1H\x1b[1m›\x1b[22m \x1b[?25h\x1b[20;3H")
	var received bytes.Buffer
	one := make([]byte, 1)
	for {
		if _, err := io.ReadFull(os.Stdin, one); err != nil {
			os.Exit(2)
		}
		received.Write(one)
		if bytes.HasSuffix(received.Bytes(), []byte("\x1b[201~\r")) {
			break
		}
	}
	if err := os.WriteFile(os.Getenv("PAIR184_ACCEPT_OUTPUT"), received.Bytes(), 0600); err != nil {
		os.Exit(3)
	}
	os.Exit(0)
}

// Only host-process edges are faked. The normal cleanup constructor chooses
// paths, archives through OSRuntime, constructs the descriptor, and publishes
// the actual durable completion consumed by Couch's controller.
type liveAcceptanceCleanupRuntime struct {
	*fakeRuntime
	real          *OSRuntime
	intent        QuitIntent
	removalErrors []error
}

func (r *liveAcceptanceCleanupRuntime) TakeQuitIntent(string) (QuitIntent, bool, error) {
	return r.intent, true, nil
}
func (r *liveAcceptanceCleanupRuntime) WriteQuitIntent(_ string, intent QuitIntent) error {
	r.intent = intent
	return nil
}
func (r *liveAcceptanceCleanupRuntime) InferAgent(string) string { return "claude" }
func (r *liveAcceptanceCleanupRuntime) ParkScrollback(tag, agent string, move bool) (string, bool) {
	if len(r.sessions) != 0 || len(r.reaped) != 1 || r.reaped[0] != tag {
		return "", false
	}
	return r.real.ParkScrollback(tag, agent, move)
}
func (r *liveAcceptanceCleanupRuntime) FileSize(path string) (int64, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, false
	}
	return info.Size(), true
}
func (r *liveAcceptanceCleanupRuntime) Remove(path string) {
	r.fakeRuntime.Remove(path)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		r.removalErrors = append(r.removalErrors, err)
	}
}

func TestCouchLiveCleanupHelper(t *testing.T) {
	raw := os.Getenv("PAIR184_CLEANUP_INTENT")
	if raw == "" {
		t.Skip("cross-package live cleanup helper")
	}
	var intent QuitIntent
	if err := json.Unmarshal([]byte(raw), &intent); err != nil {
		t.Fatal(err)
	}
	if intent.Request == nil {
		t.Fatal("missing actual quit reference")
	}
	ref := intent.Request
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: ref.DataDir, RepoScope: ref.RepoScope, Tag: ref.Tag})
	if err != nil {
		t.Fatal(err)
	}
	runtime := &liveAcceptanceCleanupRuntime{fakeRuntime: newFakeRuntime(), real: NewOSRuntime(paths.ScopeDir(), "/pair"), intent: intent}
	runtime.sessions = []Session{{Name: os.Getenv("PAIR184_CLEANUP_SESSION")}}
	result, handled := runCleanupContext(context.Background(), Env{DataDir: paths.ScopeDir()}, runtime, launchStep{tag: ref.Tag, session: os.Getenv("PAIR184_CLEANUP_SESSION"), agent: "claude"}, ref.RepoScope, 0, io.Discard)
	if !handled || result.Outcome != pairlifecycle.CompletionSuccess || result.Scrollback == nil || len(runtime.removalErrors) != 0 {
		t.Fatalf("cleanup result %+v, filesystem errors %v", result, runtime.removalErrors)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("PAIR184_CLEANUP_OUTPUT"), encoded, 0600); err != nil {
		t.Fatal(err)
	}
}
