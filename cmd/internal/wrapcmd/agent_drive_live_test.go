package wrapcmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	modelCaptureStartup = iota
	modelCaptureCommand
	modelCaptureSubmitted
	modelCaptureModelSelected
	modelCaptureEffortSelected
	modelCaptureApplied
	modelCaptureRecoveryCommand
	modelCaptureRecoverySubmitted
	modelCaptureCancelled
	modelCaptureRecovered
)

// TestAgentDriveLiveCapture probes only fresh PTYs. No generated agent turn is
// submitted; the sole command is the built-in /model control.
func TestAgentDriveLiveCapture(t *testing.T) {
	agent := os.Getenv("PAIR_LIVE_MODEL_HARNESS")
	if agent == "" {
		t.Skip("set PAIR_LIVE_MODEL_HARNESS for isolated model capture")
	}
	if agent != "codex" {
		t.Fatal("live model capture supports isolated Codex only; Claude conformance is hub-owned")
	}
	tempRoot, err := os.MkdirTemp("/tmp", "pair-model-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tempRoot) })
	root, err := filepath.EvalSymlinks(tempRoot)
	if err != nil {
		t.Fatal(err)
	}
	env := peerLiveEnvironment(os.Environ(), root)
	for _, dir := range []string{"home", "codex", "claude", "config", "cache", "data", "state", "repo"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	cwd := filepath.Join(root, "repo")
	env = peerLiveLocalAuth(t, "codex", root, cwd, env)
	if err := exec.Command("git", "init", "-q", cwd).Run(); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(root, "codex", "config.toml")
	before, err := os.ReadFile(config)
	if err != nil {
		t.Fatal("disposable Codex config missing")
	}
	args := []string{"--no-daemon", "--no-alt-screen", "--sandbox", "read-only", "--ask-for-approval", "never", "-c", "check_for_update_on_startup=false"}
	executable, err := exec.LookPath(agent)
	if err != nil {
		t.Fatal(err)
	}
	vc := exec.Command(executable, "--version")
	vc.Env = env
	version, err := vc.Output()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("version %s", strings.TrimSpace(string(version)))
	classifier := newHarnessTTYLiveClassifier(t, agent)
	defer classifier.Close()
	phase := modelCaptureStartup
	var pending []byte
	captures := map[string][]byte{}
	var retainedOutput []byte
	var styledScreen string
	save := func(name string) {
		captures[name+".raw"] = []byte(strings.ReplaceAll(string(retainedOutput), root, "/scratch"))
		captures[name+".screen"] = []byte(strings.ReplaceAll(styledScreen, root, "/scratch"))
	}
	_, state, captureErr := captureHarnessTTYClassified(harnessTTYCaptureRequest{Executable: executable, Args: args, Env: env, Dir: cwd, StartupTimeout: 25 * time.Second,
		Classify: func(chunk, retained []byte) harnessTTYConformanceState {
			retainedOutput = retained
			observed := classifier.Observe(chunk, retained)
			snap := classifier.proxy.terminal.Snapshot()
			var lines []string
			for y := 0; y < snap.Height; y++ {
				marker := -1
				if snap.CursorVisible && snap.Cursor.Y == y {
					marker = snap.Cursor.X
				}
				lines = append(lines, renderTailRow(snap.Cells[y*snap.Width:(y+1)*snap.Width], marker))
			}
			styled := strings.Join(lines, "\n")
			styledScreen = styled
			screen := strings.NewReplacer("‹dim›", "", "‹/dim›", "", "‹rev›", "", "‹/rev›", "", "‹cursor›", "").Replace(styled)
			screen = strings.ReplaceAll(screen, root, "/scratch")
			if observed == harnessTTYUnauthenticated {
				return observed
			}
			if phase == modelCaptureStartup && peerComposerState(agent, snap) == PeerComposerEmpty && !strings.Contains(screen, "loading") && strings.Contains(screen, "GPT-6.1-Sol default") {
				save("startup")
				pending = []byte("/model")
				var readErr error
				before, readErr = os.ReadFile(config)
				if readErr != nil {
					return harnessTTYRecognizerDrift
				}
				phase = modelCaptureCommand
			}
			if phase == modelCaptureCommand && strings.Contains(screen, "/model") {
				phase = modelCaptureSubmitted
				pending = []byte("\r")
			}
			if phase == modelCaptureSubmitted && strings.Contains(screen, "Select Model and Effort") && strings.Contains(styled, "‹rev›› 1. GPT-6.1-Sol (current)") && strings.Contains(screen, "esc back") {
				save("model-picker")
				phase = modelCaptureModelSelected
				pending = []byte("\r")
			}
			if phase == modelCaptureModelSelected && strings.Contains(screen, "Select Reasoning Level for GPT-6.1-Sol") && strings.Contains(styled, "‹rev›› 1. Low (default)") && strings.Contains(screen, "s session") {
				save("effort-picker")
				phase = modelCaptureEffortSelected
				pending = []byte("s")
			}
			if phase == modelCaptureEffortSelected && strings.Contains(screen, "Model changed to gpt-6.1-sol low for this session only") && strings.Contains(screen, "GPT-6.1-Sol low") && peerComposerState(agent, snap) == PeerComposerEmpty {
				save("session-status")
				phase = modelCaptureApplied
				pending = []byte("/model")
			}
			if phase == modelCaptureApplied && strings.Contains(screen, "/model") {
				phase = modelCaptureRecoverySubmitted
				pending = []byte("\r")
			}
			if phase == modelCaptureRecoverySubmitted && strings.Contains(screen, "Select Model") {
				save("recovery-picker")
				phase = modelCaptureCancelled
				pending = []byte("\x1b")
			}
			if phase == modelCaptureCancelled && peerComposerState(agent, snap) == PeerComposerEmpty && strings.Contains(screen, "GPT-6.1-Sol low") {
				save("recovery-prompt")
				phase = modelCaptureRecovered
				return harnessTTYRecognized
			}
			return harnessTTYWaiting
		}, Input: func([]byte) []byte {
			b := pending
			pending = nil
			if len(b) > 0 {
				t.Logf("input phase=%d styled=%s", phase, strings.ReplaceAll(strings.Join(classifier.proxy.agentTail(38).Lines, "\n"), root, "/scratch"))
				time.Sleep(1500 * time.Millisecond)
			}
			return b
		}})
	after, err := os.ReadFile(config)
	if err != nil {
		t.Fatal("disposable Codex config missing after apply")
	}
	t.Logf("defaults before=%x after=%x", sha256.Sum256(before), sha256.Sum256(after))
	if captureErr != nil {
		t.Fatalf("isolated model capture blocked: state=%s phase=%d: %v", state, phase, captureErr)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("session selection changed disposable defaults")
	}
	if phase != modelCaptureRecovered {
		t.Fatal("session/recovery phases unconfirmed")
	}
	if dir := *peerLiveCaptureFlag; dir != "" {
		if !filepath.IsAbs(dir) {
			t.Fatal("capture directory must be absolute")
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		for name, data := range captures {
			if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
		metadata := map[string]any{"agent": agent, "version": strings.TrimSpace(string(version)), "model": "gpt-6.1-sol", "effort": "low", "scope": "session", "defaults_before_sha256": fmt.Sprintf("%x", sha256.Sum256(before)), "defaults_after_sha256": fmt.Sprintf("%x", sha256.Sum256(after)), "recovery": "separate ESC returned fresh empty composer", "capture": "fresh isolated PTY; /model only; auth copies removed on cleanup"}
		b, err := json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "metadata.json"), append(b, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

// Replay raw bytes through the shared terminal emulator: the recorded paint,
// rather than a hand-authored screen, must supply recipe evidence.
func TestAgentDriveRecordedCaptureReplay(t *testing.T) {
	expected := map[string]string{"startup": "Ask Codex to do anything", "model-picker": "Select Model and Effort", "effort-picker": "Select Reasoning Level for GPT-6.1-Sol", "session-status": "Model changed to gpt-6.1-sol low for this session only", "recovery-picker": "Select Model and Effort", "recovery-prompt": "GPT-6.1-Sol low"}
	for phase, marker := range expected {
		t.Run(phase, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", "drive", "codex", "0.162.1", phase+".raw"))
			if err != nil {
				t.Fatal(err)
			}
			for _, private := range []string{"/Users/", "/private/tmp/pair-model-", "access_token", "refresh_token", "id_token"} {
				if bytes.Contains(raw, []byte(private)) {
					t.Fatal("capture contains private state")
				}
			}
			classifier := newHarnessTTYLiveClassifier(t, "codex")
			defer classifier.Close()
			classifier.Observe(raw, raw)
			screen := strings.Join(classifier.proxy.agentTail(38).Lines, "\n")
			if !strings.Contains(screen, marker) {
				t.Fatalf("replayed %s did not paint expected recipe evidence", phase)
			}
			if phase == "recovery-prompt" && peerComposerState("codex", classifier.proxy.terminal.Snapshot()) != PeerComposerEmpty {
				t.Fatal("replay did not recover empty composer")
			}
		})
	}
}
