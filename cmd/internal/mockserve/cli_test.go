package mockserve

import (
	"bytes"
	"os/exec"
	"testing"
)

func TestParseCommand(t *testing.T) {
	for _, args := range [][]string{{"serve", "dir"}, {"serve", "--layout2", "--port", "1234"}, {"use", "dir", "mock-2"}, {"--help"}, {}} {
		if _, err := ParseCommand(args); err != nil {
			t.Errorf("%v: %v", args, err)
		}
	}
	for _, args := range [][]string{{"serve"}, {"use", "dir"}, {"use", "dir", "mock-0"}, {"serve", "dir", "--port", "65536"}, {"serve", "dir", "--port", "x"}, {"serve", "dir", "--port", "1", "--port", "2"}, {"--help", "x"}} {
		if _, err := ParseCommand(args); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
	var out, errs bytes.Buffer
	if rc := Run(t.Context(), []string{"use", t.TempDir(), "mock-1"}, &out, &errs); rc != 1 {
		t.Fatalf("rc=%d err=%s", rc, &errs)
	}
}
func FuzzParseCommand(f *testing.F) {
	f.Add("serve", "x", "mock-2")
	f.Fuzz(func(t *testing.T, a, b, c string) {
		cmd, err := ParseCommand([]string{a, b, c})
		if err == nil && cmd.Mode != "use" {
			t.Fatalf("escaped closed argv: %+v", cmd)
		}
	})
}

func TestReloadJavaScript(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is unavailable; run reload_test.js when installed")
	}
	out, err := exec.CommandContext(t.Context(), node, "--test", "reload_test.js").CombinedOutput()
	if err != nil {
		t.Fatalf("reload client: %v\n%s", err, out)
	}
}
