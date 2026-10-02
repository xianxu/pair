package couchcmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func TestListWithMissingAuxiliaryStoreUsesIsolatedRoots(t *testing.T) {
	fixture := isolatedCouchTestEnvironment(t)
	t.Chdir(fixture)
	root := os.Getenv("PAIR_DATA_DIR")
	seedIsolatedCouchSelection(t)
	c, err := storagegc.NewCoordinator(root)
	if err != nil {
		t.Fatal(err)
	}
	auxiliary := filepath.Join(fixture, "auxiliary")
	if err := os.Mkdir(auxiliary, 0700); err != nil {
		t.Fatal(err)
	}
	if err := c.RegisterStore(context.Background(), auxiliary); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(auxiliary); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if code := Run([]string{"--list"}, strings.NewReader(""), &out, &stderr); code != 0 {
		t.Fatalf("production --list rejected intact store: %d %s", code, stderr.String())
	}
	if _, err := c.ReadRegistry(); err == nil {
		t.Fatal("listing silently removed missing registration")
	}
}

func TestIsolatedSmokeEnvironmentProtectsAmbientRegistry(t *testing.T) {
	operatorFixture, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	operatorHome, operatorData := filepath.Join(operatorFixture, "home"), filepath.Join(operatorFixture, "data")
	if err := os.MkdirAll(operatorHome, 0700); err != nil {
		t.Fatal(err)
	}
	operatorRoot := launcher.ResolveDataDir(operatorHome, operatorData)
	if err := os.MkdirAll(operatorRoot, 0700); err != nil {
		t.Fatal(err)
	}
	coordinator, err := storagegc.NewCoordinator(operatorRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := coordinator.RegisterStore(context.Background(), t.TempDir()); err != nil {
		t.Fatal(err)
	}
	registryPath := filepath.Join(operatorRoot, ".retention", "stores.json")
	before, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	identityDir := filepath.Join(operatorFixture, "identity")
	if err := os.Mkdir(identityDir, 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(identityDir, "couch-identities.json")
	poison := []byte("ambient identity must remain untouched\n")
	if err := os.WriteFile(sentinel, poison, 0600); err != nil {
		t.Fatal(err)
	}
	_, source, _, _ := runtime.Caller(0)
	script := filepath.Join(filepath.Dir(source), "..", "..", "..", "tests", "with-isolated-pair.sh")
	cmd := exec.Command("sh", script, "env", "PAIR346_ISOLATION_CHILD=1", os.Args[0], "-test.run=^TestIsolatedSmokeListChild$", "-test.v")
	cmd.Env = append(os.Environ(), "COUCH_ISOLATED_ROOT="+operatorFixture, "COUCH_IDENTITY_DIR="+identityDir, "HOME="+operatorHome, "TMPDIR="+operatorHome+"/", "XDG_DATA_HOME="+operatorData, "PAIR_DATA_DIR="+operatorRoot, "COUCH_STORE_DIR="+filepath.Join(operatorRoot, "couch"), "PAIR_LOG_PATH="+filepath.Join(operatorRoot, "poison-log"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isolated child: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "isolated registration verified") {
		t.Fatalf("child did not verify registration: %s", output)
	}
	after, err := os.ReadFile(registryPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("ambient operator registry changed: %s %v", after, err)
	}
	identityAfter, err := os.ReadFile(sentinel)
	if err != nil || !bytes.Equal(poison, identityAfter) {
		t.Fatalf("ambient identity changed: %q %v", identityAfter, err)
	}
}

func TestIsolatedSmokeListChild(t *testing.T) {
	if os.Getenv("PAIR346_ISOLATION_CHILD") != "1" {
		t.Skip("subprocess helper")
	}
	if temp := os.Getenv("TMPDIR"); temp != filepath.Clean(temp) || !strings.HasPrefix(temp, filepath.Dir(os.Getenv("HOME"))+string(filepath.Separator)) {
		t.Fatalf("temporary storage is not canonical and isolated: %q", temp)
	}
	isolated := os.Getenv("COUCH_ISOLATED_ROOT")
	if !filepath.IsAbs(isolated) || isolated != filepath.Dir(os.Getenv("HOME")) {
		t.Fatalf("missing or inherited isolated root: %q", isolated)
	}
	for _, key := range []string{"HOME", "TMPDIR", "XDG_DATA_HOME", "PAIR_DATA_DIR", "COUCH_STORE_DIR"} {
		if !strings.HasPrefix(os.Getenv(key), isolated+string(filepath.Separator)) {
			t.Fatalf("%s escaped isolated root: %q", key, os.Getenv(key))
		}
	}
	if os.Getenv("COUCH_IDENTITY_DIR") != "" {
		t.Fatal("inherited identity override escaped isolation")
	}
	if os.Getenv("PAIR_LOG_PATH") != "" {
		t.Fatal("inherited artifact override escaped isolation")
	}
	t.Chdir(t.TempDir())
	root := launcher.ResolveDataDir(os.Getenv("HOME"), os.Getenv("XDG_DATA_HOME"))
	if root != os.Getenv("PAIR_DATA_DIR") || filepath.Join(root, "couch") != os.Getenv("COUCH_STORE_DIR") {
		t.Fatal("roots do not agree")
	}
	seedIsolatedCouchSelection(t)
	var out, stderr bytes.Buffer
	if code := Run([]string{"--list"}, strings.NewReader(""), &out, &stderr); code != 0 {
		t.Fatalf("list: %d %s", code, stderr.String())
	}
	c, err := storagegc.NewCoordinator(root)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := c.ReadRegistry()
	if err != nil || len(registry.Stores) == 0 {
		t.Fatalf("isolated store not registered: %+v %v", registry, err)
	}
	t.Log("isolated registration verified")
}

// Actual-process fixtures must opt into a confined authority; HOME alone does
// not redirect production singleton ownership or durable identity selection.
func isolatedCouchTestEnvironment(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "PAIR_") || strings.HasPrefix(key, "COUCH_") || strings.HasPrefix(key, "XDG_") {
			t.Setenv(key, "")
		}
	}
	for key, value := range map[string]string{
		"COUCH_ISOLATED_ROOT": root,
		"HOME":                filepath.Join(root, "home"),
		"XDG_DATA_HOME":       filepath.Join(root, "data"),
		"PAIR_DATA_DIR":       filepath.Join(root, "data", "pair"),
		"COUCH_STORE_DIR":     filepath.Join(root, "data", "pair", "couch"),
		"COUCH_IDENTITY_DIR":  filepath.Join(root, "identity"),
	} {
		t.Setenv(key, value)
	}
	return root
}

func seedIsolatedCouchSelection(t *testing.T) {
	t.Helper()
	_, lease, err := (OSRuntime{}).prepareSingleton(true)
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
}
