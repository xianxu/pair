package wrapcmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
)

// TestSlashReturnLiveConformance reuses the bounded TTY owner in this trusted
// checkout. Claude opens only its read-only status panel, then cancels it.
// Codex uses TestAgentDriveLiveConformance and its settled admission gate.
func TestSlashReturnLiveConformance(t *testing.T) {
	agent := os.Getenv("PAIR_LIVE_SLASH")
	if agent == "" {
		t.Skip("set PAIR_LIVE_SLASH=claude for command-only native Return conformance")
	}
	if agent != "claude" {
		t.Fatal("this command-only check supports Claude; use TestAgentDriveLiveConformance for Codex")
	}
	executable, err := exec.LookPath(agent)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"home", "codex", "claude", "config", "cache", "data", "state", "repo"} {
		if err := os.Mkdir(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	cwd, command := repoRoot, "/status"
	env := peerLiveLocalAuth(t, agent, root, cwd, peerLiveEnvironment(os.Environ(), root))
	version := exec.Command(executable, "--version")
	version.Env = env
	v, err := version.Output()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(strings.TrimSpace(string(v)))
	args := []string{"--safe-mode", "--tools", "", "--strict-mcp-config", "--setting-sources", ""}
	classifier := newHarnessTTYLiveClassifier(t, agent)
	defer func() {
		if err := classifier.Close(); err != nil {
			t.Error(err)
		}
	}()
	p := classifier.proxy
	stage := 0
	var pending []byte
	out, state, err := captureHarnessTTYConformance(harnessTTYCaptureRequest{
		Executable: executable, Args: args, Env: env, Dir: cwd,
		Input: func([]byte) []byte { data := pending; pending = nil; return data },
	}, func(chunk, retained []byte) harnessTTYConformanceState {
		current := classifier.Observe(chunk, retained)
		if current == harnessTTYUnauthenticated || current == harnessTTYWorkspaceTrust || current == harnessTTYRecognizerDrift {
			return current
		}
		snapshot := p.terminal.Snapshot()
		text, known := qualifiedComposerText(agent, snapshot)
		if !snapshot.Complete {
			return harnessTTYWaiting
		}
		switch stage {
		case 0:
			if known && text == "" {
				pending = []byte(command)
				stage = 1
			}
		case 1:
			if known && text == command {
				decision := decidePlainReturn(*p.ttyProfile, false, &snapshot)
				if !decision.submits || !bytes.Equal(decision.bytes, p.ttyProfile.keymap.altCR) {
					t.Fatalf("native command not proved: %+v", decision)
				}
				p.driveControl.Store(true)
				pending = p.emitPlainCRFromSnapshot(nil, &snapshot)
				p.driveControl.Store(false)
				stage = 2
			}
		case 2:
			screen := p.agentDriveScreen()
			phase := agentdrive.ReadModelScreen(agent, screen).Phase
			if phase == agentdrive.ScreenModel || phase == agentdrive.ScreenCombined || agent == "claude" && agentdrive.DismissablePanel(agent, screen.Lines) {
				pending = []byte{0x1b}
				stage = 3
			}
		case 3:
			if known && text == "" {
				stage = 4
				return harnessTTYRecognized
			}
		}
		return harnessTTYWaiting
	})

	writeHarnessTTYCaptureIfRequested(t, agent, "native slash Return", repoRoot, out)
	if err != nil || state != harnessTTYRecognized || stage != 4 {
		t.Fatalf("stage=%d state=%s err=%v; screen=%s", stage, state, err, strings.ReplaceAll(strings.Join(p.agentDriveScreen().Lines, "\n"), root, "/scratch"))
	}
	t.Logf("native %s opened and canceled in trusted checkout; command-only, no conversational turn; capture_bytes=%d", command, len(out))
}
