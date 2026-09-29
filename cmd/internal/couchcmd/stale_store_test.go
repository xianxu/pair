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
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "PAIR_") || strings.HasPrefix(key, "COUCH_") {
			t.Setenv(key, "")
		}
	}
	home, data := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", data)
	t.Chdir(t.TempDir())
	root := launcher.ResolveDataDir(home, data)
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAIR_DATA_DIR", root)
	t.Setenv("COUCH_STORE_DIR", filepath.Join(root, "couch"))
	c, err := storagegc.NewCoordinator(root)
	if err != nil {
		t.Fatal(err)
	}
	auxiliary := t.TempDir()
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
	operatorHome, operatorData := t.TempDir(), t.TempDir()
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
	_, source, _, _ := runtime.Caller(0)
	script := filepath.Join(filepath.Dir(source), "..", "..", "..", "tests", "with-isolated-pair.sh")
	cmd := exec.Command("sh", script, "env", "PAIR346_ISOLATION_CHILD=1", os.Args[0], "-test.run=^TestIsolatedSmokeListChild$", "-test.v")
	cmd.Env = append(os.Environ(), "HOME="+operatorHome, "XDG_DATA_HOME="+operatorData, "PAIR_DATA_DIR="+operatorRoot, "COUCH_STORE_DIR="+filepath.Join(operatorRoot, "couch"), "PAIR_LOG_PATH="+filepath.Join(operatorRoot, "poison-log"))
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
}

func TestIsolatedSmokeListChild(t *testing.T) {
	if os.Getenv("PAIR346_ISOLATION_CHILD") != "1" {
		t.Skip("subprocess helper")
	}
	if os.Getenv("PAIR_LOG_PATH") != "" {
		t.Fatal("inherited artifact override escaped isolation")
	}
	t.Chdir(t.TempDir())
	root := launcher.ResolveDataDir(os.Getenv("HOME"), os.Getenv("XDG_DATA_HOME"))
	if root != os.Getenv("PAIR_DATA_DIR") || filepath.Join(root, "couch") != os.Getenv("COUCH_STORE_DIR") {
		t.Fatal("roots do not agree")
	}
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
