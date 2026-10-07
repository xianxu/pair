package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func singletonTestRuntime(t *testing.T, env map[string]string) OSRuntime {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return OSRuntime{accountHome: func() (string, error) { return home, nil }, env: func(k string) string { return env[k] }}
}

func TestProductionSingletonContendsAcrossAmbientStores(t *testing.T) {
	root := t.TempDir()
	first := singletonTestRuntime(t, map[string]string{"HOME": root, "COUCH_STORE_DIR": filepath.Join(root, "first"), "PAIR_DATA_DIR": filepath.Join(root, "data")})
	_, lease, err := first.prepareSingleton(true)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	second := first
	second.env = func(k string) string {
		return map[string]string{"HOME": filepath.Join(root, "other-home"), "COUCH_STORE_DIR": filepath.Join(root, "second"), "XDG_DATA_HOME": filepath.Join(root, "other-data")}[k]
	}
	_, other, err := second.prepareSingleton(true)
	if other != nil {
		other.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "supervis") {
		t.Fatalf("second owner = %v; expected ownership refusal", err)
	}
	if _, err := os.Stat(filepath.Join(root, "second")); !os.IsNotExist(err) {
		t.Fatalf("loser created store: %v", err)
	}
}

func TestSelectedRuntimeRootsSurviveHomeChange(t *testing.T) {
	root := t.TempDir()
	env := map[string]string{"HOME": root, "PAIR_DATA_DIR": filepath.Join(root, "legacy-data"), "COUCH_STORE_DIR": filepath.Join(root, "legacy-store"), "COUCH_IDENTITY_DIR": filepath.Join(root, "legacy-identity")}
	rt := singletonTestRuntime(t, env)
	selected, lease, err := rt.prepareSingleton(true)
	if err != nil {
		t.Fatal(err)
	}
	lease.Close()
	env = map[string]string{"HOME": filepath.Join(root, "changed-home"), "COUCH_THREAD_SCOPE": "scope", "COUCH_THREAD_TAG": "tag", "PAIR_SESSION_NAME": "session", "PAIR_LAUNCH_NONCE": "nonce"}
	rt.env = func(k string) string { return env[k] }
	resumed, release, err := rt.prepareSingleton(false)
	if err != nil {
		t.Fatal(err)
	}
	defer release.Close()
	if resumed.StoreDir() != selected.StoreDir() || runtimePairDataDir(resumed) != runtimePairDataDir(selected) {
		t.Fatal("selected runtime drifted")
	}
	ns, err := resumed.ResolveNamespace()
	if err != nil {
		t.Fatal(err)
	}
	c, err := resumed.NewCouchWith(couchcore.NewFakeRunner(), ns)
	if err != nil {
		t.Fatal(err)
	}
	if c.Artifacts.(interface{ PairLifecycleDataDir() string }).PairLifecycleDataDir() != selected.selection.Roots.PairDataDir {
		t.Fatal("artifact reader ignored selection")
	}
	if tracesForRuntime(resumed).root != selected.selection.Roots.PairDataDir {
		t.Fatal("trace root ignored selection")
	}
	var out, stderr bytes.Buffer
	code := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "actors"}, resumed, &out, &stderr, func(_ context.Context, socket string, _ any, response any) error {
		want, _ := couchmessage.SocketPath(selected.StoreDir(), "broker")
		if socket != want {
			t.Fatalf("socket %q != %q", socket, want)
		}
		raw := []byte(`{"code":"ok"}`)
		return json.Unmarshal(raw, response)
	})
	if code != 0 {
		t.Fatalf("message %d: %s", code, &stderr)
	}
}

func TestIsolatedRuntimeRejectsRootEscape(t *testing.T) {
	for _, key := range []string{"COUCH_STORE_DIR", "PAIR_DATA_DIR", "COUCH_IDENTITY_DIR"} {
		t.Run(key, func(t *testing.T) {
			root := t.TempDir()
			outside := t.TempDir()
			rt := singletonTestRuntime(t, map[string]string{"COUCH_ISOLATED_ROOT": root, key: outside})
			_, lease, err := rt.prepareSingleton(true)
			if lease != nil {
				lease.Close()
			}
			if err == nil {
				t.Fatal("escaped isolation")
			}
			entries, _ := os.ReadDir(root)
			if len(entries) != 0 {
				t.Fatal("refusal mutated isolation root")
			}
		})
	}
}

func TestSingletonReadDoesNotAdopt(t *testing.T) {
	root := t.TempDir()
	rt := singletonTestRuntime(t, map[string]string{"COUCH_ISOLATED_ROOT": root})
	_, lease, err := rt.prepareSingleton(false)
	if lease != nil {
		lease.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "adopt") {
		t.Fatalf("read before adoption: %v", err)
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatal("read initialized state")
	}
}

func TestSingletonChildUsesSelectedRoots(t *testing.T) {
	root := isolatedCouchTestEnvironment(t)
	t.Setenv("PAIR_LOG_PATH", filepath.Join(root, "poison"))
	t.Setenv("COUCH_CAPTURE_DIR", filepath.Join(root, "capture"))
	rt, lease, err := (OSRuntime{}).prepareSingleton(true)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	output := filepath.Join(root, "child-env")
	h, err := rt.runtimeRunner(couchcore.ExecRunner{}).Start(root, []string{"/bin/sh", "-c", `printf '%s\n' "$PAIR_DATA_DIR" "$COUCH_STORE_DIR" "$COUCH_IDENTITY_DIR" "$COUCH_ISOLATED_ROOT" "$PAIR_LOG_PATH" "$COUCH_PAIR_DATA_DIR" "$COUCH_CAPTURE_DIR" > "$1"`, "sh", output}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if h.Wait() != 0 {
		t.Fatal("child failed")
	}
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	if len(lines) != 7 || lines[6] != "" || lines[0] != "" || lines[1] != rt.StoreDir() || lines[2] != rt.selection.Roots.IdentityDir || lines[3] != root || lines[4] != "" || lines[5] != rt.selection.Roots.PairDataDir {
		t.Fatalf("wrong descendant environment: %q", raw)
	}
}

func TestSelectedRuntimeDoesNotConferSupervisorAuthority(t *testing.T) {
	isolatedCouchTestEnvironment(t)
	rt, lease, err := (OSRuntime{}).prepareSingleton(true)
	if err != nil {
		t.Fatal(err)
	}
	ns, err := rt.ResolveNamespace()
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if unauthorized, err := rt.AcquireSupervisor(ns); err == nil {
		unauthorized.Close()
		t.Fatal("released runtime retained supervisor authority")
	}
	read, release, err := (OSRuntime{}).prepareSingleton(false)
	if err != nil {
		t.Fatal(err)
	}
	defer release.Close()
	if unauthorized, err := read.AcquireSupervisor(ns); err == nil {
		unauthorized.Close()
		t.Fatal("selected inventory authorized an owner")
	}
}

func TestInitialExplicitPairRootLocatesItsDefaultCouchStore(t *testing.T) {
	root := t.TempDir()
	rt := singletonTestRuntime(t, map[string]string{"PAIR_DATA_DIR": filepath.Join(root, "custom-pair")})
	manager, request, _, err := rt.singletonManager()
	if err != nil {
		t.Fatal(err)
	}
	if manager.Defaults.StoreDir != filepath.Join(request.Roots.PairDataDir, "couch") {
		t.Fatalf("default Couch store %q detached from requested Pair root %q", manager.Defaults.StoreDir, request.Roots.PairDataDir)
	}
}

func TestLegacyHostedScopedPairRootUsesSelectedInventory(t *testing.T) {
	isolatedCouchTestEnvironment(t)
	rt, lease, err := (OSRuntime{}).prepareSingleton(true)
	if err != nil {
		t.Fatal(err)
	}
	lease.Close()
	t.Setenv("COUCH_THREAD_SCOPE", "scope123")
	t.Setenv("PAIR_DATA_DIR", filepath.Join(rt.selection.Roots.PairDataDir, "repos", "scope123"))
	read, release, err := (OSRuntime{}).prepareSingleton(false)
	if err != nil {
		t.Fatal(err)
	}
	defer release.Close()
	if read.selection.Roots != rt.selection.Roots {
		t.Fatal("hosted helper changed runtime roots")
	}
}

func TestSingletonBlockedChildRetainsActorPathsAndSelectedRoots(t *testing.T) {
	root := isolatedCouchTestEnvironment(t)
	rt, lease, err := (OSRuntime{}).prepareSingleton(true)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	runner := couchcore.NewFakeRunner()
	actorPath := filepath.Join(root, "actor.log")
	child, err := rt.runtimeRunner(runner).StartBlocked(context.Background(), root, []string{"pair", "resume", "tag"}, []string{"PAIR_LOG_PATH=" + actorPath, "PAIR_DATA_DIR=/wrong"}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer child.Cancel()
	env := map[string]string{}
	for _, entry := range runner.Child(child.ID()).Env {
		key, value, _ := strings.Cut(entry, "=")
		env[key] = value
	}
	if env["PAIR_DATA_DIR"] != "" || env["COUCH_PAIR_DATA_DIR"] != rt.selection.Roots.PairDataDir || env["PAIR_LOG_PATH"] != actorPath || env["COUCH_STORE_DIR"] != rt.StoreDir() {
		t.Fatalf("blocked child environment: %v", env)
	}
	if runner.Child(child.ID()).ExecCount != 0 {
		t.Fatal("blocked child executed before acknowledgement")
	}
}
