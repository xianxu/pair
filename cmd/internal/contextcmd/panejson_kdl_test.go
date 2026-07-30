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

// The agent pane's `args "-c"` line in the zellij layouts is the PRODUCER of
// pane-<tag>-<agent>.json; contextcmd.paneCwd and titlepoller are its consumers.
// Every other test in the tree hand-writes that JSON, so nothing executed the
// producer and nothing would notice it breaking (#133).
//
// The failure mode this guards is specific and silent: shell printf RECYCLES its
// format string while arguments remain. Drop a `%s` and leave its argument and
// printf emits the format twice — two concatenated JSON objects in one file, which
// fails json.Unmarshal, so paneCwd returns "" and the poller skips the pane. No
// test goes red, because none of them run this line.
//
// ARCH-MOCK: `zellij` and `pair` are faked on PATH — the same seam the real launch
// resolves them through — so `exec pair wrap` is a no-op while the JSON-writing
// prefix runs for real. `zellij` is a RECORDING fake, not a no-op: it captures
// argv so the `rename-pane` half of the line is asserted too, since that hop
// carries the startup pane title and is never observed live.

// agentPaneShellFromKDL extracts the agent pane's shell command from a layout.
// The agent pane is identified by the pane_id JSON it writes, which distinguishes
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
		if !strings.HasPrefix(trimmed, prefix) || !strings.Contains(trimmed, `pane_id`) {
			continue
		}
		body := strings.TrimSuffix(strings.TrimPrefix(trimmed, prefix), `"`)
		return unescapeKDL(body)
	}
	t.Fatalf("%s: no agent pane `args \"-c\"` line containing pane_id", layout)
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
			recordingStub(t, binDir, "zellij", argvLog)
			stubBin(t, binDir, "pair")

			cmd := exec.Command("sh", "-c", shell)
			cmd.Dir = paneCwdDir
			cmd.Env = []string{
				"PATH=" + binDir,
				"PAIR_DATA_DIR=" + dataDir,
				"PAIR_TAG=t",
				"PAIR_AGENT=claude",
				"ZELLIJ_PANE_ID=7",
				// The startup title createflow exports; the KDL reads it as
				// ${PAIR_PANE_TITLE:-agent} and hands it to rename-pane.
				"PAIR_PANE_TITLE=claude",
				// PWD is what the printf records as "cwd"; a non-interactive sh
				// inherits it rather than recomputing it, so set it explicitly.
				"PWD=" + paneCwdDir,
				"HOME=" + t.TempDir(),
			}
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("layout command failed: %v\noutput: %s", err, out)
			}

			// (a) The real consumer resolves the cwd from what the producer wrote.
			if got := paneCwd(dataDir, "t", "claude"); got != paneCwdDir {
				t.Errorf("paneCwd = %q, want %q", got, paneCwdDir)
			}

			raw, err := os.ReadFile(filepath.Join(dataDir, "pane-t-claude.json"))
			if err != nil {
				t.Fatal(err)
			}

			// (b) Exactly ONE JSON object with a non-empty pane_id. A second
			// decodable value means printf recycled its format.
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

// A session started BEFORE #133 has a pane-<tag>-<agent>.json carrying the extra
// "cwd_display" key, and it stays on disk when the binary updates underneath it.
// The consumer must still resolve its cwd — this is the upgrade path, not a
// hypothetical. (Several fixtures elsewhere in the tree happen to carry the legacy
// shape; this makes that coverage intentional rather than incidental.)
func TestPaneCwdToleratesLegacyCwdDisplayField(t *testing.T) {
	dataDir := t.TempDir()
	legacy := `{"pane_id":"7","cwd":"/home/u/work","cwd_display":"~/work"}` + "\n"
	if err := os.WriteFile(filepath.Join(dataDir, "pane-t-claude.json"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := paneCwd(dataDir, "t", "claude"); got != "/home/u/work" {
		t.Errorf("paneCwd on a pre-#133 record = %q, want /home/u/work", got)
	}
}
