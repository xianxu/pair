package termcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type admissionSplitRuntime struct {
	*fakeRuntime
	runSplit func([]string) error
}

func (r admissionSplitRuntime) RunZellijActionQuiet(args ...string) error {
	return r.runSplit(args)
}

func TestSplitTerminalDownScrubsNativeAdmission(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"zellij", "pair"} {
		body := "#!/bin/sh\nprintf '%s|%s|%s\\n' \"${0##*/}\" \"$*\" \"${PAIR_NATIVE_ADMISSION-unset}\" >> \"$RECORDING\"\n"
		if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0700); err != nil {
			t.Fatal(err)
		}
	}
	recording := filepath.Join(root, "recording")
	rt := admissionSplitRuntime{
		fakeRuntime: &fakeRuntime{currentPaneID: "4", panesJSON: `[{"id":4,"is_focused":true,"is_floating":false,"pane_x":75,"title":"terminal","terminal_command":"pair term"}]`},
		runSplit: func(args []string) error {
			if len(args) < 4 || args[len(args)-4] != "--" || args[len(args)-3] != "sh" || args[len(args)-2] != "-c" {
				return fmt.Errorf("unexpected terminal split command: %q", args)
			}
			// Zellij creates the new pane from its retained launch environment,
			// even though the invoking terminal itself already cleared the grant.
			cmd := exec.Command(args[len(args)-3], args[len(args)-2:]...)
			cmd.Env = []string{"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH"), "RECORDING=" + recording, "PAIR_NATIVE_ADMISSION=retained-server-grant", "ZELLIJ_PANE_ID=7"}
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("split shell: %w: %s", err, out)
			}
			return nil
		},
	}
	if err := splitTerminalDown(rt); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(recording)
	if err != nil {
		t.Fatal(err)
	}
	want := "zellij|action rename-pane --pane-id 7 terminal|unset\npair|term|unset\n"
	if string(raw) != want {
		t.Fatalf("dynamic terminal inherited native capability:\n%s\nwant:\n%s", raw, want)
	}
}
