package launcher

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/creack/pty"
)

// Crosses the production create boundary, the actual embedded layout's shell
// stanza, a recording pair shim, and the public wrapper into a real child.
func TestCreateLayoutWrapperPreservesAgentCommand(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "pair-real")
	build := exec.Command("go", "build", "-o", binary, "../../pair-go")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build wrapper: %v\n%s", err, out)
	}
	for _, layout := range []string{"main-2.kdl", "main-3.kdl"} {
		t.Run(layout, func(t *testing.T) {
			rt := newFakeRuntime()
			want := []string{"--model", "a b", "", "雪", "*", "$(touch should-not-exist)", "one\ntwo", "a'\"b"}
			code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: want, AgentArgsExplicit: true, AgentArgsFromCouch: true, FreshRequired: true}), rt)
			if err != nil || code != 0 {
				t.Fatalf("launch %d %v", code, err)
			}
			raw, err := os.ReadFile(filepath.Join("../../../zellij/layouts", layout))
			if err != nil {
				t.Fatal(err)
			}
			var stanza string
			re := regexp.MustCompile(`args "-c" (".*")`)
			for _, line := range strings.Split(string(raw), "\n") {
				if strings.Contains(line, "exec pair wrap") {
					m := re.FindStringSubmatch(line)
					if len(m) != 2 {
						t.Fatalf("missing shell stanza: %s", line)
					}
					stanza, err = strconv.Unquote(m[1])
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			if stanza == "" {
				t.Fatal("missing agent pane command")
			}
			for name, body := range map[string]string{
				"pair":   "#!/bin/sh\nprintf '%s\\000' \"$@\" > \"$WRAPPER_ARGS\"\nunset PAIR_TAG PAIR_DATA_DIR PAIR_SESSION_ID PAIR_SCOPE_KEY PAIR_LAUNCH_ORDINAL PAIR_LAUNCH_NONCE\nexec \"$PAIR_REAL\" \"$@\"\n",
				"codex":  "#!/bin/sh\nprintf '%s\\000' \"$@\" > \"$CHILD_ARGS\"\n",
				"zellij": "#!/bin/sh\nexit 0\n",
			} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0755); err != nil {
					t.Fatal(err)
				}
			}
			master, tty, err := pty.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer master.Close()
			defer tty.Close()
			cmd := exec.Command("sh", "-c", stanza)
			cmd.Stdin = tty
			// Use only the launcher's encoded production output as command authority.
			cmd.Env = []string{"PATH=" + dir + ":" + os.Getenv("PATH"), "HOME=" + dir, AgentCommandEnv + "=" + rt.env[AgentCommandEnv], "PAIR_REAL=" + binary, "WRAPPER_ARGS=" + filepath.Join(dir, "wrapper.args"), "CHILD_ARGS=" + filepath.Join(dir, "child.args"), "PAIR_AGENT_PANE_PATH=" + filepath.Join(dir, "pane.json"), "PAIR_SCROLLBACK_RAW_PATH=" + filepath.Join(dir, "scroll.raw")}
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("layout/wrapper: %v\n%s", err, out)
			}
			readArgs := func(name string) []string {
				data, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil {
					t.Fatal(err)
				}
				return strings.Split(string(bytes.TrimSuffix(data, []byte{0})), "\x00")
			}
			if got := readArgs("wrapper.args"); !reflect.DeepEqual(got, []string{"wrap", "--scrollback-log", filepath.Join(dir, "scroll.raw"), "--from-launch-env"}) {
				t.Fatalf("wrapper argv %#v", got)
			}
			if got := readArgs("child.args"); !reflect.DeepEqual(got, append(append([]string{}, want...), "--no-alt-screen")) {
				t.Fatalf("child argv %#v", got)
			}
		})
	}
}
