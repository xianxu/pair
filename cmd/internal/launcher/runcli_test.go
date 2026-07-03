package launcher

import (
	"bytes"
	"strings"
	"testing"
)

// `pair --help` / `pair help` prints the native usage to stdout and exits 0 (#99
// M5c — the shell that used to own help is retired).
func TestLaunchNativeHelp(t *testing.T) {
	for _, arg := range []string{"--help", "-h", "help"} {
		t.Run(arg, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code, err := LaunchNative([]string{arg}, "/pair", &stdout, &stderr)
			if err != nil || code != 0 {
				t.Fatalf("%s: code=%d err=%v", arg, code, err)
			}
			if !strings.Contains(stdout.String(), "USAGE") || stderr.Len() != 0 {
				t.Fatalf("%s: stdout=%q stderr=%q", arg, stdout.String(), stderr.String())
			}
		})
	}
}

// A leading flag that isn't help is a usage error → stderr + exit 2 (no shell to
// defer to).
func TestLaunchNativeBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := LaunchNative([]string{"--nope"}, "/pair", &stdout, &stderr)
	if err != nil || code != 2 {
		t.Fatalf("code=%d err=%v, want 2", code, err)
	}
	if !strings.Contains(stderr.String(), "not an agent") || stdout.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}
