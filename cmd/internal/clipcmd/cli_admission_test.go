package clipcmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/notifytransport"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

// This uses the shared suite binary, private fake clipboard tools, the real
// detached copy entry and existing broker. Wrapper-envelope qualification is
// shared with notification presentation tests, not claimed by broker receipt.
func TestDetachedClipboardRefusalUsesExistingBroker(t *testing.T) {
	binary := testpair.Binary(t) // Resolve before the fixture replaces the Go environment.
	f := zellijtest.New(t)
	for _, entry := range f.Env() {
		k, v, _ := strings.Cut(entry, "=")
		t.Setenv(k, v)
	}
	// No connected client: auto-quote must refuse, leaving the OS copy successful.
	f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"test": {Tab: zellijtest.DefaultTab()}}})
	//lint:allow tmp-literal — the notification broker appends a 32-hex digest and PID; macOS TMPDIR can exceed its 100-byte socket limit
	root, err := os.MkdirTemp("/tmp", "pcl-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	t.Setenv("PAIR_NOTIFY_SOCKET_DIR", root)
	binding := filepath.Join(t.TempDir(), "pid")
	broker, err := notifytransport.Start(binding, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()
	t.Setenv("PAIR_PAIR_WRAP_PID_PATH", binding)
	t.Setenv("PAIR_TAG", "t")
	t.Setenv("PAIR_DATA_DIR", t.TempDir())
	clipboard := filepath.Join(f.Root(), "clipboard")
	tool := filepath.Join(filepath.Dir(f.Path()), "pbcopy")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\ncat > \"$PAIR_TEST_CLIPBOARD\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAIR_TEST_CLIPBOARD", clipboard)
	// Keep sibling resolution exact without creating a fresh executable image.
	alias := testpair.At(t, filepath.Join(t.TempDir(), "pair"))
	sourceInfo, err := os.Stat(binary)
	if err != nil {
		t.Fatal(err)
	}
	aliasInfo, err := os.Stat(alias)
	if err != nil || !os.SameFile(sourceInfo, aliasInfo) {
		t.Fatalf("clipboard fixture must execute the shared image (#749): %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, alias, "clip", "copy-on-select")
	cmd.Env = testpair.Environment(t, t.TempDir(), append(f.Env(), "PAIR_NOTIFY_SOCKET_DIR="+root, "PAIR_PAIR_WRAP_PID_PATH="+binding, "PAIR_TAG=t", "PAIR_DATA_DIR="+os.Getenv("PAIR_DATA_DIR"), "PAIR_TEST_CLIPBOARD="+clipboard)...)
	cmd.Stdin = strings.NewReader("private selected body")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("copy failed: %v %s", err, out)
	}
	select {
	case message := <-broker.Messages():
		if message != "Copied; auto-quote refused: uncertain Zellij reply" {
			t.Fatalf("message=%q", message)
		}
	case <-ctx.Done():
		t.Fatal("no detached refusal notification")
	}
	copied, err := os.ReadFile(clipboard)
	if err != nil || !bytes.Equal(copied, []byte("private selected body")) {
		t.Fatalf("clipboard=%q,%v", copied, err)
	}
	if len(f.Effects()) != 0 {
		t.Fatalf("native effects=%v", f.Effects())
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("PAIR_DATA_DIR"), "quote-t")); !os.IsNotExist(err) {
		t.Fatalf("quote staged=%v", err)
	}
}

func TestCopyOnSelectShellFixture(t *testing.T) {
	binary := testpair.Binary(t)
	script, err := filepath.Abs("../../../tests/copy-on-select-test.sh")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "bash", script)
	cmd.Env = testpair.Environment(t, t.TempDir(), "PAIR_TEST_BINARY="+binary)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("shell fixture: %v\n%s", err, out)
	}
}
