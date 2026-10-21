package launcher

import (
	"bytes"
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
