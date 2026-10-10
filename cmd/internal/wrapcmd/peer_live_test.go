package wrapcmd

import (
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

var (
	peerLiveHarnessFlag  = flag.String("peer-live-harness", "", "isolated live peer receiver: claude or codex")
	peerLiveScenarioFlag = flag.String("peer-live-scenario", "", "startup, paste-short, paste-multiline, paste-wrapped, draft, menu, or image-admission")
	peerLiveSubmitFlag   = flag.Bool("peer-live-submit", false, "permit one harmless no-tools prompt submission in a new isolated PTY")
	peerLiveCaptureFlag  = flag.String("peer-live-capture-dir", "", "absolute directory for successful live captures")
	peerLiveBodyFlag     = flag.String("peer-live-body", "", "submit body: short (default), hyphen-wrap (about 600 chars, rendered wrapped) or collapsed (about 1200 chars, which Claude summarizes)")
	peerLiveAuthFlag     = flag.Bool("peer-live-use-local-auth", false, "use existing login in a fresh session (Claude uses native keychain; Codex copies login into a disposable home)")
)

// peerLiveEnvironment defaults to isolated writable harness state. The explicit
// local-auth mode below opts into native credentials. Existing API-key values may
// authenticate the child; they are never logged or written into fixtures.
func peerLiveEnvironment(inherited []string, root string) []string {
	allowed := map[string]bool{"PATH": true, "USER": true, "LANG": true, "LC_ALL": true, "COLORTERM": true, "ANTHROPIC_API_KEY": true, "OPENAI_API_KEY": true, "HTTPS_PROXY": true, "HTTP_PROXY": true, "NO_PROXY": true, "SSL_CERT_FILE": true, "NODE_EXTRA_CA_CERTS": true}
	var env []string
	for _, entry := range inherited {
		key, _, _ := strings.Cut(entry, "=")
		if allowed[key] {
			env = append(env, entry)
		}
	}
	for key, dir := range map[string]string{"HOME": "home", "CODEX_HOME": "codex", "CLAUDE_CONFIG_DIR": "claude", "XDG_CONFIG_HOME": "config", "XDG_CACHE_HOME": "cache", "XDG_DATA_HOME": "data", "XDG_STATE_HOME": "state"} {
		env = append(env, key+"="+filepath.Join(root, dir))
	}
	return append(env, "TERM=xterm-256color", "DISABLE_AUTOUPDATER=1")
}

func TestPeerLiveEnvironmentIsolated(t *testing.T) {
	env := peerLiveEnvironment([]string{"PATH=/bin", "HOME=/real", "CODEX_HOME=/real/codex", "CLAUDE_CONFIG_DIR=/real/claude", "PAIR_SESSION_NAME=existing", "COUCH_STORE_DIR=/real/couch", "ZELLIJ_SESSION_NAME=existing", "OPENAI_API_KEY=test-only"}, "/isolated")
	for _, key := range []string{"PAIR_SESSION_NAME", "COUCH_STORE_DIR", "ZELLIJ_SESSION_NAME"} {
		if envValue(env, key) != "" {
			t.Fatalf("inherited %s", key)
		}
	}
	if envValue(env, "HOME") != "/isolated/home" || envValue(env, "CODEX_HOME") != "/isolated/codex" || envValue(env, "CLAUDE_CONFIG_DIR") != "/isolated/claude" {
		t.Fatal("harness state not isolated")
	}
	if envValue(env, "PATH") != "/bin" || envValue(env, "OPENAI_API_KEY") != "test-only" {
		t.Fatal("execution/auth environment lost")
	}
}

// TestPeerLiveConformance uses only new PTYs in disposable repositories. It
// never connects to an existing Couch/Pair/native harness conversation. Normal
// tests skip it. Set PAIR_LIVE_PEER_SCENARIO to startup, paste-short,
// paste-multiline, paste-wrapped, draft, menu, image-admission, or submit.
// Default startup performs no input. Submit requires an explicit scenario and
// a harmless no-tools body. Local-auth mode may trust only the empty repository
// this test creates; login and tool-permission dialogs remain blockers.
func TestPeerLiveConformance(t *testing.T) {
	agent := os.Getenv("PAIR_LIVE_PEER_HARNESS")
	if *peerLiveHarnessFlag != "" {
		agent = *peerLiveHarnessFlag
	}
	if agent == "" {
		t.Skip("set PAIR_LIVE_PEER_HARNESS=claude or codex for isolated peer conformance")
	}
	if agent != "claude" && agent != "codex" {
		t.Fatal("unsupported live peer harness")
	}
	scenario := os.Getenv("PAIR_LIVE_PEER_SCENARIO")
	if *peerLiveScenarioFlag != "" {
		scenario = *peerLiveScenarioFlag
	}
	if *peerLiveSubmitFlag {
		scenario = "submit"
	}
	if scenario == "submit" && !*peerLiveSubmitFlag {
		t.Fatal("submit requires explicit -peer-live-submit")
	}
	if scenario == "" {
		scenario = "startup"
	}
	valid := map[string]bool{"startup": true, "paste-short": true, "paste-multiline": true, "paste-wrapped": true, "draft": true, "menu": true, "image-admission": true, "submit": true}
	if !valid[scenario] {
		t.Fatalf("unknown scenario %q", scenario)
	}
	executable, err := exec.LookPath(agent)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"home", "codex", "claude", "config", "cache", "data", "state", "repo"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	cwd := filepath.Join(root, "repo")
	git := exec.Command("git", "init", "-q", cwd)
	if err := git.Run(); err != nil {
		t.Fatalf("initialize isolated repository: %v", err)
	}

	env := peerLiveEnvironment(os.Environ(), root)
	if *peerLiveAuthFlag {
		env = peerLiveLocalAuth(t, agent, root, cwd, env)
	}
	versionCmd := exec.Command(executable, "--version")
	versionCmd.Env = env
	versionOut, err := versionCmd.Output()
	if err != nil {
		t.Fatalf("read version: %v", err)
	}
	version := strings.TrimSpace(string(versionOut))
	args := []string{"--bare", "--tools", "", "--strict-mcp-config", "--setting-sources", ""}
	if *peerLiveAuthFlag {
		// Bare mode intentionally excludes OAuth. Safe mode keeps ordinary
		// authentication while disabling hooks, plugins and project context.
		args[0] = "--safe-mode"
	}
	if agent == "codex" {
		args = []string{"--no-daemon", "--no-alt-screen", "--sandbox", "read-only", "--ask-for-approval", "never", "-c", "check_for_update_on_startup=false"}
	}
	classifier := newHarnessTTYLiveClassifier(t, agent)
	defer classifier.Close()
	p := classifier.proxy
	m := couchmessage.Message{ID: "peer-live-conformance", From: couchmessage.Binding{Slot: "peer:0"}, To: couchmessage.Binding{Slot: "peer:1"}, Body: "Reply PEER_SMOKE_OK only. Do not use tools.", Deadline: time.Now().Add(20 * time.Second)}
	switch scenario {
	case "paste-multiline":
		m.Body = "Reply PEER_SMOKE_OK only.\nDo not use tools.\nThis is harmless test text.\nFourth line.\nFifth line."
	case "paste-wrapped":
		m.Body = "Do not use tools. " + strings.TrimSuffix(strings.Repeat("harmless wrapped text ", 14), " ")
	}
	// pair#418: long single-line bodies with hyphenated tokens, the shape that
	// never submitted (hyphen wrap projection; collapsed paste marker).
	switch *peerLiveBodyFlag {
	case "", "short":
	case "hyphen-wrap", "collapsed":
		repeat := 6
		if *peerLiveBodyFlag == "collapsed" {
			repeat = 13
		}
		m.Body = "Reply PEER_SMOKE_OK only. Do not use tools." + strings.Repeat(" See workshop/projects/ariadne-robustness-1.md and receipt 80a1b3c3-1d5d-478d-b2f6-a440e98adf8f.", repeat)
		m.Deadline = time.Now().Add(30 * time.Second)
	default:
		t.Fatalf("unknown -peer-live-body %q", *peerLiveBodyFlag)
	}
	d := newPeerDelivery(m.To, time.Now)
	ready, sent, finished := false, false, false
	seeded := false
	trustStep := 0
	var trustInput []byte
	var failure error
	renderClass := "unobserved"
	var rendered string
	var expectedWrites int
	queue := func() bool {
		p.peer = d
		if err := d.reserve(m.ID, d.sequence); err != nil {
			failure = err
			return false
		}
		if err := d.enqueue(m); err != nil {
			failure = err
			return false
		}
		return true
	}
	raw, state, captureErr := captureHarnessTTYClassified(harnessTTYCaptureRequest{Executable: executable, Args: args, Env: env, Dir: cwd, StartupTimeout: 15 * time.Second,
		Classify: func(chunk, retained []byte) harnessTTYConformanceState {
			if p.peer != nil {
				p.peer.observeOutput(chunk)
			}
			observed := classifier.Observe(chunk, retained)
			if agent == "claude" && *peerLiveAuthFlag {
				s := p.terminal.Snapshot()
				var screen strings.Builder
				for y := 0; y < s.Height; y++ {
					screen.WriteString(orientationRowText(s, y))
				}
				v := strings.ReplaceAll(screen.String(), " ", "")
				if strings.Contains(v, "Accessingworkspace:"+cwd) {
					if trustStep == 0 && strings.Contains(v, "❯No,exit") {
						trustInput = []byte("\x1b[B")
						trustStep = 1
					}
					if trustStep == 1 && strings.Contains(v, "❯Yes,Itrustthisfolder") {
						trustInput = []byte("\r")
						trustStep = 2
					}
					return harnessTTYWaiting
				}
			}
			if observed == harnessTTYUnauthenticated || (observed == harnessTTYWorkspaceTrust && trustStep == 0) {
				return observed
			}
			snapshot := p.terminal.Snapshot()
			composer := peerComposerState(agent, snapshot)
			if !ready && composer == PeerComposerEmpty {
				ready = true
				if scenario == "startup" {
					finished = true
				}
			}
			if ready && scenario == "image-admission" && !finished {
				if queue() {
					d.admitImage()
					var writes bytes.Buffer
					p.dispatchPeer(&writes)
					if writes.Len() != 0 {
						failure = fmt.Errorf("automatic input after image admission")
					}
				}
				finished = true
			}
			if sent && !finished {
				text, known := peerComposerText(agent, snapshot)
				if scenario == "submit" && expectedWrites == 2 && known && text == "" {
					finished = true
				}
				if known && strings.TrimSpace(text) == strings.TrimSpace(peerEnvelope(m)) {
					renderClass = "exact"
					rendered = text
					if scenario != "submit" {
						finished = true
					}
				}
				if known && (strings.Contains(text, "[Pasted") || strings.Contains(text, "[pasted")) {
					renderClass = "collapsed"
					rendered = text
					if scenario != "submit" {
						finished = true
					}
				}
				if known && renderClass == "unobserved" && peerComposerMatches(agent, snapshot, peerEnvelope(m)) {
					renderClass = "wrapped"
					rendered = text
					if scenario != "submit" {
						finished = true
					}
				}
			}
			if seeded && !sent && (scenario == "draft" || scenario == "menu") && composer != PeerComposerEmpty {
				if scenario == "draft" && !strings.Contains(orientationRowText(snapshot, snapshot.Cursor.Y), "operator draft") {
					return harnessTTYWaiting
				}
				if !queue() {
					finished = true
					return harnessTTYWaiting
				}
				var writes bytes.Buffer
				p.dispatchPeer(&writes)
				if writes.Len() != 0 {
					failure = fmt.Errorf("automatic input over %s", scenario)
				}
				finished = true
			}
			return harnessTTYWaiting
		},
		Startup: func([]byte) bool { return finished },
		Input: func([]byte) []byte {
			if trustInput != nil {
				input := trustInput
				trustInput = nil
				// Test setup only: initial painting precedes the picker's
				// mount effects, which otherwise reset an immediate selection.
				time.Sleep(time.Second)
				return input
			}
			if failure != nil {
				finished = true
				return nil
			}
			if !ready {
				return nil
			}
			if !sent {
				if scenario == "draft" || scenario == "menu" {
					if seeded {
						return nil
					}
					seeded = true
					if scenario == "draft" {
						return []byte("operator draft")
					}
					return []byte("/")
				}
				if !queue() {
					finished = true
					return nil
				}
				var writes bytes.Buffer
				p.dispatchPeer(&writes)
				if !strings.Contains(writes.String(), peerEnvelope(m)) {
					failure = fmt.Errorf("empty composer produced no peer paste")
					finished = true
					return nil
				}
				sent = true
				expectedWrites++
				return writes.Bytes()
			}
			if scenario == "submit" {
				var writes bytes.Buffer
				p.dispatchPeer(&writes)
				if writes.Len() > 0 {
					expectedWrites++
					if d.receipt().Status != couchmessage.Submitted {
						failure = fmt.Errorf("unexpected post-paste write")
					}
					return writes.Bytes()
				}
			}
			return nil
		},
	})
	if captureErr != nil {
		if *peerLiveCaptureFlag != "" && state != harnessTTYUnauthenticated && p.ttyProfile.recognize(p.terminal.Snapshot()) {
			if !filepath.IsAbs(*peerLiveCaptureFlag) {
				t.Fatal("capture directory must be absolute")
			}
			if err := os.MkdirAll(*peerLiveCaptureFlag, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(*peerLiveCaptureFlag, agent+"-"+scenario+"-drift.raw"), raw, 0600); err != nil {
				t.Fatal(err)
			}
		}
		if *peerLiveAuthFlag && state != harnessTTYUnauthenticated {
			s := p.terminal.Snapshot()
			composer, known := peerComposerText(agent, s)
			t.Logf("composer known=%t state=%d cursor=%+v text=%q", known, peerComposerState(agent, s), s.Cursor, composer)
			for row := 0; row < s.Height; row++ {
				line := strings.TrimSpace(orientationRowText(s, row))
				if line != "" && !strings.Contains(line, "http") {
					t.Logf("screen: %s", line)
				}
			}
		}
		// Report only known UI categories, never raw login URLs or credentials.
		visible := strings.ToLower(string(raw))
		for _, marker := range []string{"theme", "onboarding", "welcome", "trust", "login", "sign in", "api key", "terminal setup", "model", "dangerous", "permission"} {
			if strings.Contains(visible, marker) {
				t.Logf("startup contains UI marker %q", marker)
			}
		}
		t.Fatalf("isolated %s %s: state=%s ready=%t bytes=%d: %v; no live receiver qualification", agent, scenario, state, ready, len(raw), captureErr)
	}
	if failure != nil {
		t.Fatal(failure)
	}
	if scenario == "submit" && (expectedWrites != 2 || d.receipt().Status != couchmessage.Submitted) {
		t.Fatal("no confirmed paste and submit")
	}
	if strings.HasPrefix(scenario, "paste-") && !sent {
		t.Fatal("paste was not exercised")
	}
	t.Logf("%s scenario=%s render=%s composer-bytes=%d capture-bytes=%d sha256=%x", version, scenario, renderClass, len(rendered), len(raw), sha256.Sum256(raw))
	destination := os.Getenv("PAIR_LIVE_PEER_CAPTURE_DIR")
	if *peerLiveCaptureFlag != "" {
		destination = *peerLiveCaptureFlag
	}
	if destination != "" {
		if !filepath.IsAbs(destination) {
			t.Fatal("capture directory must be absolute")
		}
		if err := os.MkdirAll(destination, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(destination, agent+"-"+scenario+".raw"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func peerLiveLocalAuth(t *testing.T, agent, root, cwd string, env []string) []string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	copyPrivate := func(source, destination string) {
		b, err := os.ReadFile(source)
		if err != nil {
			t.Fatal("local login file unavailable")
		}
		if err := os.WriteFile(destination, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if agent == "codex" {
		from := os.Getenv("CODEX_HOME")
		if from == "" {
			from = filepath.Join(home, ".codex")
		}
		copyPrivate(filepath.Join(from, "auth.json"), filepath.Join(root, "codex", "auth.json"))
		config := fmt.Sprintf("[projects.%q]\ntrust_level = \"trusted\"\n", cwd)
		if err := os.WriteFile(filepath.Join(root, "codex", "config.toml"), []byte(config), 0600); err != nil {
			t.Fatal(err)
		}
		return env
	}
	// macOS credentials belong to the native config's keychain service.
	// Let the CLI authenticate normally; safe mode disables customizations,
	// and the fresh cwd/no-resume arguments never attach an existing thread.
	var native []string
	for _, entry := range env {
		if strings.HasPrefix(entry, "HOME=") {
			entry = "HOME=" + home
		}
		if strings.HasPrefix(entry, "CLAUDE_CONFIG_DIR=") {
			from := os.Getenv("CLAUDE_CONFIG_DIR")
			if from == "" {
				continue
			}
			entry = "CLAUDE_CONFIG_DIR=" + from
		}
		native = append(native, entry)
	}
	return native
}
