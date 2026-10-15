package wrapcmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Execute the shipped pane commands: inspecting layout text cannot establish
// which environment the shell actually passes to each helper and final exec.
func TestLayoutNativeAdmissionReachesOnlyAgentWrapper(t *testing.T) {
	for _, layout := range []string{"main-2.kdl", "main-3.kdl"} {
		t.Run(layout, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("../../../zellij/layouts", layout))
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			bin := filepath.Join(root, "bin")
			if err := os.Mkdir(bin, 0700); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"pair", "nvim", "zellij"} {
				body := "#!/bin/sh\nprintf '%s|%s|%s\\n' \"${0##*/}\" \"$*\" \"${PAIR_NATIVE_ADMISSION-unset}\" >> \"$RECORDING\"\n"
				if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0700); err != nil {
					t.Fatal(err)
				}
			}
			commandRE := regexp.MustCompile(`^\s*args "-c" (".*")\s*$`)
			var commands []string
			for _, line := range strings.Split(string(raw), "\n") {
				if match := commandRE.FindStringSubmatch(line); match != nil {
					command, err := strconv.Unquote(match[1])
					if err != nil {
						t.Fatal(err)
					}
					commands = append(commands, command)
				}
			}
			wantPanes := 2
			if layout == "main-3.kdl" {
				wantPanes = 3
			}
			if len(commands) != wantPanes {
				t.Fatalf("layout pane commands=%d want %d", len(commands), wantPanes)
			}
			recording := filepath.Join(root, "recording")
			for _, command := range commands {
				cmd := exec.Command("sh", "-c", command)
				cmd.Env = []string{
					"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH"),
					"HOME=" + root, "RECORDING=" + recording,
					"PAIR_NATIVE_ADMISSION=exact-single-use-grant", "ZELLIJ_PANE_ID=7", "PAIR_PANE_TITLE=codex",
					"PAIR_HOME=" + filepath.Join(root, "home with spaces"),
					"PAIR_AGENT_PANE_PATH=" + filepath.Join(root, "agent-pane"),
					"PAIR_SCROLLBACK_RAW_PATH=" + filepath.Join(root, "scrollback"),
					"PAIR_NVIM_DRAFT_PID_PATH=" + filepath.Join(root, "draft-pid"),
					"PAIR_DRAFT_PATH=" + filepath.Join(root, "draft.md"),
				}
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("pane shell failed: %v: %s", err, out)
				}
			}
			recorded, err := os.ReadFile(recording)
			if err != nil {
				t.Fatal(err)
			}
			wrappers, drafts, terminals, helpers := 0, 0, 0, 0
			for _, line := range strings.Split(strings.TrimSpace(string(recorded)), "\n") {
				parts := strings.SplitN(line, "|", 3)
				if len(parts) != 3 {
					t.Fatalf("bad recording: %q", line)
				}
				if parts[0] == "pair" && strings.HasPrefix(parts[1], "wrap ") {
					wrappers++
					if parts[2] != "exact-single-use-grant" {
						t.Fatalf("agent wrapper lost original grant: %q", line)
					}
					continue
				}
				if parts[2] != "" && parts[2] != "unset" {
					t.Errorf("non-agent command inherited authority: %q", line)
				}
				switch parts[0] {
				case "nvim":
					drafts++
				case "pair":
					if parts[1] != "term" {
						t.Fatalf("unexpected non-agent Pair command: %q", line)
					}
					terminals++
				case "zellij":
					helpers++
				default:
					t.Fatalf("unexpected layout process: %q", line)
				}
			}
			if wrappers != 1 || drafts != 1 || terminals != wantPanes-2 || helpers != wantPanes-1 {
				t.Fatalf("missing layout boundary: wrappers=%d drafts=%d terminals=%d helpers=%d", wrappers, drafts, terminals, helpers)
			}
		})
	}
}
