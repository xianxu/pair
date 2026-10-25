package contextcmd

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Execute the real layout bootstrap with a protocol fake of Pair's internal
// publisher and guarded-command adapter. The publisher's codec has its own
// panebirth tests; this checks that both layouts publish before guarded rename,
// preserve native argv and pass the legacy pane fields to existing consumers.

// agentPaneShellFromKDL extracts the agent pane's shell command from a layout.
// The agent pane is identified by its pair wrap command, which distinguishes
// it from the draft and terminal panes' own `args "-c"` lines.
func agentPaneShellFromKDL(t *testing.T, layout string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "zellij", "layouts", layout))
	if err != nil {
		t.Fatal(err)
	}
	const prefix = `args "-c" "`
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, prefix) || !strings.Contains(trimmed, `pair wrap`) {
			continue
		}
		body := strings.TrimSuffix(strings.TrimPrefix(trimmed, prefix), `"`)
		return unescapeKDL(body)
	}
	t.Fatalf("%s: no agent pane `args \"-c\"` line containing pair wrap", layout)
	return ""
}

// unescapeKDL resolves the only two escapes the layouts use: \" and \\. It must
// leave `\n` alone — that backslash-n is printf's escape, not the KDL string's,
// and collapsing it here would make the extracted command write a literal newline
// instead of asking printf to.
func unescapeKDL(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\\') {
			b.WriteByte(s[i+1])
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// stubBin writes a no-op executable so `exec pair wrap …` resolves without
// launching anything real.
func stubBin(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// recordingStub writes a `zellij` that APPENDS its argv to $ZELLIJ_ARGV_LOG.
//
// A no-op stub would let the layout's `rename-pane --pane-id "$ZELLIJ_PANE_ID"
// "${PAIR_PANE_TITLE:-agent}"` run with its arguments vanishing — so mangling that
// expansion, or the --pane-id flag, would keep every test in the tree green. The
// startup pane title is never observed live (that needs a fresh session), so this
// is the only thing standing behind it. A stateful fake, not a stateless mock.
func recordingStub(t *testing.T, dir, name, log string) {
	t.Helper()
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" >> \"" + log + "\"\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

// TestAgentPaneJSONRoundTripsThroughKDL executes the real layout command and
// asserts its output decodes through the real consumer.
func TestAgentPaneJSONRoundTripsThroughKDL(t *testing.T) {
	for _, layout := range []string{"main-2.kdl", "main-3.kdl"} {
		t.Run(layout, func(t *testing.T) {
			shell := agentPaneShellFromKDL(t, layout)

			dataDir := t.TempDir()
			paneCwdDir := t.TempDir()
			binDir := t.TempDir()
			argvLog := filepath.Join(dataDir, "zellij-argv")
			panePath := filepath.Join(dataDir, "pane-t-claude.json")
			recordingStub(t, binDir, "zellij", argvLog)
			pairScript := `#!/bin/sh
case "$1 $2" in
  'internal pane-ready')
    printf '{"pane_id":"%s","cwd":"%s"}\n' "$ZELLIJ_PANE_ID" "$PWD" > "$PAIR_AGENT_PANE_PATH" ;;
  'internal zellij-command')
    test -s "$PAIR_AGENT_PANE_PATH" || exit 91
    shift 3
    exec zellij "$@" ;;
  'wrap --scrollback-log') exit 0 ;;
  *) exit 92 ;;
esac
`
			if err := os.WriteFile(filepath.Join(binDir, "pair"), []byte(pairScript), 0o755); err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(shell, "pair internal pane-ready;") || !strings.Contains(shell, "pair internal zellij-command -- action rename-pane") {
				t.Fatalf("pane readiness must publish before guarded rename: %s", shell)
			}

			cmd := exec.Command("sh", "-c", shell)
			cmd.Dir = paneCwdDir
			cmd.Env = []string{
				"PATH=" + binDir,
				"PAIR_DATA_DIR=" + dataDir,
				"PAIR_TAG=t",
				"PAIR_AGENT=claude",
				"PAIR_AGENT_PANE_PATH=" + panePath,
				"PAIR_SCROLLBACK_RAW_PATH=" + filepath.Join(dataDir, "scrollback-t-claude.raw"),
				"ZELLIJ_PANE_ID=7",
				// The startup title createflow exports; the KDL reads it as
				// ${PAIR_PANE_TITLE:-agent} and hands it to rename-pane.
				"PAIR_PANE_TITLE=claude",
				// PWD is what the pane publisher records as "cwd"; a non-interactive sh
				// inherits it rather than recomputing it, so set it explicitly.
				"PWD=" + paneCwdDir,
				"HOME=" + t.TempDir(),
			}
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("layout command failed: %v\noutput: %s", err, out)
			}

			raw, err := os.ReadFile(panePath)
			if err != nil {
				t.Fatal(err)
			}

			// Exactly one sidecar object with the legacy pane fields.
			dec := json.NewDecoder(strings.NewReader(string(raw)))
			var first struct {
				PaneID string `json:"pane_id"`
				Cwd    string `json:"cwd"`
			}
			if err := dec.Decode(&first); err != nil {
				t.Fatalf("first JSON value: %v\nfile: %s", err, raw)
			}
			if first.PaneID != "7" {
				t.Errorf("pane_id = %q, want 7", first.PaneID)
			}
			if first.Cwd != paneCwdDir {
				t.Errorf("cwd = %q, want %q", first.Cwd, paneCwdDir)
			}
			var extra json.RawMessage
			if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
				t.Fatalf("expected exactly one JSON object, got a second (%v): %s", err, raw)
			}

			// (c) The startup-title hop: createflow's PAIR_PANE_TITLE must reach
			// rename-pane for THIS pane. Asserting argv, not just that it ran.
			argv, err := os.ReadFile(argvLog)
			if err != nil {
				t.Fatalf("zellij was never invoked: %v", err)
			}
			got := strings.Split(strings.TrimSpace(string(argv)), "\n")
			want := []string{"action", "rename-pane", "--pane-id", "7", "claude"}
			if len(got) != len(want) {
				t.Fatalf("zellij argv = %q, want %q", got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("zellij argv = %q, want %q", got, want)
				}
			}
		})
	}
}
