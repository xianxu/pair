package diagnosticcmd

import (
	"bytes"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendCommandUsesExactPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace")
	var stderr bytes.Buffer
	if code := Run([]string{"--path", path}, func(string) string { return "" }, strings.NewReader("record\n"), &stderr); code != 0 {
		t.Fatalf("%d %s", code, &stderr)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "record\n" {
		t.Fatalf("%q %v", got, err)
	}
	for _, args := range [][]string{nil, {"--path"}, {"--path", path, "extra"}} {
		if Run(args, func(string) string { return "" }, strings.NewReader("bad"), &stderr) == 0 {
			t.Fatal("accepted invalid command")
		}
	}
}

// The command is the common append boundary used by shell/Lua emitters.
func TestAppendCommandHonorsPersistedLimitAcrossProcess(t *testing.T) {
	if path := os.Getenv("PAIR579_CHILD_APPEND_PATH"); path != "" {
		if rc := Run([]string{"--path", path}, func(string) string { return "" }, strings.NewReader("de"), os.Stderr); rc != 0 {
			t.Fatalf("append rc=%d", rc)
		}
		if rc := Run([]string{"--path", path}, func(string) string { return "" }, strings.NewReader("foreign-overflow"), os.Stderr); rc != 0 {
			t.Fatalf("optional refusal rc=%d", rc)
		}
		return
	}
	path := filepath.Join(t.TempDir(), "shared.log")
	w, err := diagnosticlog.Open(path, diagnosticlog.Options{MaxBytes: 5})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestAppendCommandHonorsPersistedLimitAcrossProcess$", "-test.count=1")
	child.Env = append(os.Environ(), "PAIR579_CHILD_APPEND_PATH="+path)
	if out, err := child.CombinedOutput(); err != nil {
		t.Fatalf("child append: %v\n%s", err, out)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "abcde" {
		t.Fatalf("short-lived writer bypassed persisted limit: %q %v", got, err)
	}
}
