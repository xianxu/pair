package vmcmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVMCLIHelpAndInvalidArgsAvoidRuntime(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"--checkout", "relative", "--", "true"}} {
		var out, errOut bytes.Buffer
		constructed := false
		code := runCLIWithRuntime(context.Background(), args, &out, &errOut, func() (Runtime, error) { constructed = true; return nil, fmt.Errorf("unexpected runtime") })
		if constructed {
			t.Fatal("help/parse failure touched runtime")
		}
		if args[0] == "--help" {
			if code != 0 || !strings.Contains(out.String(), "cleanup up to 10 s") {
				t.Fatal(code, out.String())
			}
		} else if code != 2 {
			t.Fatal(code, errOut.String())
		}
	}
}
func TestVMCLIReturnsCommandFailureAndPublishesEnvelope(t *testing.T) {
	rt, driver, req := hostFixture(t)
	driver.commandExit = 7
	var out, errOut bytes.Buffer
	code := runCLIWithRuntime(context.Background(), []string{"--checkout", req.Checkout, "--", "false"}, &out, &errOut, func() (Runtime, error) { return rt, nil })
	if code != 7 || !strings.Contains(out.String(), "timeout 20m0s + cleanup up to 10 s") {
		t.Fatal(code, out.String(), errOut.String())
	}
	raw, err := os.ReadFile(filepath.Join(rt.artifacts, "result.json"))
	if err != nil || !strings.Contains(string(raw), "timeout 20m0s + cleanup up to 10 s") {
		t.Fatal(string(raw), err)
	}
}
