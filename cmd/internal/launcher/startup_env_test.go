package launcher

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/startupbudget"
)

func TestNativeStartupDeadlineConsumedAndRejectsUnmanagedInvocation(t *testing.T) {
	t.Setenv(startupbudget.DeadlineEnv, "not-a-deadline")
	var out, stderr bytes.Buffer
	code, err := LaunchNative([]string{"--help"}, "", &out, &stderr)
	if code != 2 || err != nil || !strings.Contains(stderr.String(), "startup") {
		t.Fatalf("code=%d err=%v stderr=%q", code, err, &stderr)
	}
	if got := os.Getenv(startupbudget.DeadlineEnv); got != "" {
		t.Fatalf("deadline leaked into descendants: %q", got)
	}
}

func TestNativeStartupSuppliedZeroNeverSelectsFallback(t *testing.T) {
	opts := managedSessionOptions("create")
	raw, err := json.Marshal(opts.Args.CouchSession)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(CouchSessionIntentEnv, string(raw))
	t.Setenv(CouchLaunchProfileEnv, "")
	t.Setenv(startupbudget.DeadlineEnv, "0001-01-01T00:00:00Z")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", "")
	var out, stderr bytes.Buffer
	code, err := LaunchNative([]string{CouchSessionFlag, "resume", opts.Args.CouchSession.Tag}, "", &out, &stderr)
	if code != 2 || err != nil || !strings.Contains(stderr.String(), "startup deadline") {
		t.Fatalf("code=%d error=%v stderr=%s", code, err, &stderr)
	}
	if os.Getenv(startupbudget.DeadlineEnv) != "" {
		t.Fatal("deadline leaked")
	}
}
