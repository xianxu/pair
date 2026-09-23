package wrapcmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func orientationProxy(t *testing.T) (*proxy, <-chan orientation.DeliveryState) {
	t.Helper()
	states := make(chan orientation.DeliveryState, 8)
	p := &proxy{agentBasename: "claude", stdout: io.Discard, lifecycleEvents: make(chan TurnObservation, 8)}
	p.orientation = newOrientationDelivery(orientation.Request{SchemaVersion: 1, Tag: "work", Agent: "claude", Attempt: "n", Body: "read context"})
	p.orientation.publish = func(s orientation.DeliveryState) { states <- s }
	p.orientation.settleAfter = 20 * time.Millisecond
	if err := p.configureHarnessTTY(false, 120, 38); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.closeTerminal() })
	return p, states
}
func TestOrientationSerializedInputScheduler(t *testing.T) {
	for _, action := range []string{"submit", "input-before", "input-after-paste", "overlay-after-paste", "timeout", "child-exit"} {
		t.Run(action, func(t *testing.T) {
			p, states := orientationProxy(t)
			if action == "timeout" {
				p.orientation.deadlineAfter = 10 * time.Millisecond
			}
			reader, writer := io.Pipe()
			defer reader.Close()
			defer writer.Close()
			out := newDrainBuffer()
			done := make(chan struct{})
			go func() { p.translateStdinFrom(reader, out, time.Millisecond); close(done) }()
			if action == "input-before" {
				writer.Write([]byte("operator"))
				if !waitFor(time.Second, func() bool { return len(out.Bytes()) > 0 }) {
					t.Fatal("input not forwarded")
				}
			}
			var rolling []byte
			if action != "timeout" && action != "child-exit" {
				p.handleChunk([]byte(claudeLiveComposerPaint()), &rolling)
			}
			if action == "input-after-paste" || action == "overlay-after-paste" {
				if !waitFor(time.Second, func() bool { return bytes.Contains(out.Bytes(), []byte("read context")) }) {
					t.Fatal("no paste")
				}
			}
			if action == "input-after-paste" {
				writer.Write([]byte("operator"))
			}
			if action == "overlay-after-paste" {
				p.handleChunk([]byte("\x1b]777;"+pickerOpenOSCBody+"\x07"), &rolling)
			}
			if action == "child-exit" {
				p.orientation.observe(false, false, true)
			}
			var state orientation.DeliveryState
			select {
			case state = <-states:
			case <-time.After(time.Second):
				t.Fatal("no status")
			}
			if action == "submit" {
				if state.Phase != orientation.DeliverySubmitted || !bytes.Equal(out.Bytes(), []byte("\x1b[200~read context\x1b[201~\r")) {
					t.Fatalf("submit %#v bytes %q", state, out.Bytes())
				}
			} else {
				if state.Phase != orientation.DeliveryCancelled || bytes.Contains(out.Bytes(), []byte{'\r'}) {
					t.Fatalf("cancel %#v bytes %q", state, out.Bytes())
				}
			}
			if action == "input-after-paste" && (!state.BodyWritten || !waitFor(time.Second, func() bool { return bytes.HasSuffix(out.Bytes(), []byte("operator")) })) {
				t.Fatalf("operator text/body presence %#v %q", state, out.Bytes())
			}
			writer.Close()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("input owner did not stop")
			}
		})
	}
}

type shortOrientationWriter struct{ calls int }

func (w *shortOrientationWriter) Write(b []byte) (int, error) {
	w.calls++
	return 1, errors.New("partial")
}
func TestOrientationPartialWriteNeverRetries(t *testing.T) {
	p, states := orientationProxy(t)
	reader, writer := io.Pipe()
	defer reader.Close()
	out := &shortOrientationWriter{}
	done := make(chan struct{})
	go func() { p.translateStdinFrom(reader, out, time.Millisecond); close(done) }()
	var rolling []byte
	p.handleChunk([]byte(claudeLiveComposerPaint()), &rolling)
	select {
	case s := <-states:
		if s.Phase != orientation.DeliveryIndeterminate {
			t.Fatal(s)
		}
	case <-time.After(time.Second):
		t.Fatal("no state")
	}
	p.handleChunk([]byte(claudeLiveComposerPaint()), &rolling)
	writer.Close()
	<-done
	if out.calls != 1 {
		t.Fatalf("retried %d writes", out.calls)
	}
}
func TestOrientationRequestConsumedAndBound(t *testing.T) {
	r := orientation.Request{SchemaVersion: 1, Tag: "work", Agent: "claude", Attempt: "n", Body: "read context"}
	raw, _ := json.Marshal(r)
	env := []string{orientation.Env + "=" + string(raw), "PAIR_TAG=work", "PAIR_LAUNCH_NONCE=n"}
	request, childEnv, err := consumeOrientation(env, "claude")
	if err != nil || request == nil || envValue(childEnv, orientation.Env) != "" {
		t.Fatalf("consume %#v %v %v", request, childEnv, err)
	}
	if _, _, err := consumeOrientation(env, "codex"); err == nil {
		t.Fatal("accepted different agent")
	}
}

func TestOrientationCapturedComposersWithoutReturnRemap(t *testing.T) {
	paths, err := filepath.Glob("testdata/tty/*/*/composer.raw")
	if err != nil {
		t.Fatal(err)
	}
	// Historical Codex composer fixtures capture its provisional loading card;
	// the orientation-positive fixture is the resolved live startup frame.
	filtered := paths[:0]
	for _, path := range paths {
		if !strings.Contains(path, "/codex/") {
			filtered = append(filtered, path)
		}
	}
	paths = append(filtered, "testdata/orientation/codex/0.154.0/ready.raw")
	seen := map[string]bool{}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			agent := filepath.Base(filepath.Dir(filepath.Dir(path)))
			seen[agent] = true
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			p := &proxy{agentBasename: agent, stdout: io.Discard, lifecycleEvents: make(chan TurnObservation, 8)}
			p.orientation = newOrientationDelivery(orientation.Request{SchemaVersion: 1, Tag: "work", Agent: agent, Attempt: "n", Body: "read context"})
			states := make(chan orientation.DeliveryState, 2)
			p.orientation.publish = func(s orientation.DeliveryState) { states <- s }
			p.orientation.settleAfter = time.Millisecond
			if err := p.configureHarnessTTY(false, 120, 38); err != nil {
				t.Fatal(err)
			}
			defer p.closeTerminal()
			if p.hasReturnRemap() {
				t.Fatal("orientation enabled Return remapping")
			}
			r, w := io.Pipe()
			defer r.Close()
			out := newDrainBuffer()
			done := make(chan struct{})
			go func() { p.translateStdinFrom(r, out, time.Millisecond); close(done) }()
			var rolling []byte
			p.handleChunk(raw, &rolling)
			select {
			case state := <-states:
				if state.Phase != orientation.DeliverySubmitted {
					t.Fatalf("state %#v", state)
				}
			case <-time.After(time.Second):
				t.Fatal("captured composer did not submit")
			}
			w.Close()
			<-done
			if !bytes.Equal(out.Bytes(), []byte("\x1b[200~read context\x1b[201~\r")) {
				t.Fatalf("input %q", out.Bytes())
			}
			select {
			case event := <-p.lifecycleEvents:
				if event.Kind != ObservationUserSubmission {
					t.Fatalf("notification %#v", event)
				}
			default:
				t.Fatal("missing submission bookkeeping")
			}
		})
	}
	for _, agent := range []string{"claude", "codex", "agy", "muse", "qoder"} {
		if !seen[agent] {
			t.Errorf("missing %s fixture", agent)
		}
	}
}

// TestOrientationRuleCellToleranceStaysPerProfile pins the per-profile half of
// orientationComposerActive's rule-cell tolerance. Qoder parks its hidden
// system cursor on the closing rule after a repaint, so only Qoder may skip a
// rule cell at the prompt column and keep scanning upward; for every sibling a
// rule cell there means the cursor is not in the composer. A shared skip made
// claude report composer-ready with its cursor on the closing rule (measured
// false→true versus base), which would auto-submit an orientation prompt into
// a composer the user was not in.
func TestOrientationRuleCellToleranceStaysPerProfile(t *testing.T) {
	const grey = "136;136;136"
	cases := []struct {
		name   string
		agent  string
		stream string
		want   bool
	}{
		{
			// The measured regression. The claude recognizer accepts the
			// cursor resting on the closing rule, so only the per-profile gate
			// in orientationComposerActive keeps the answer false.
			name:   "claude cursor on the closing rule",
			agent:  "claude",
			stream: claudeBox(5, "❯", grey, "alpha") + "\x1b[?25h\x1b[8;3H",
			want:   false,
		},
		{
			name:  "muse cursor on the closing rule",
			agent: "muse",
			stream: "\x1b[7;1H\x1b[2m────\x1b[8;1H\x1b[22m⟩ work on #140" +
				"\x1b[9;1H\x1b[2m────\x1b[?25h\x1b[9;15H",
			want: false,
		},
		{
			// Agy reaches this gate uncolored only through the ruled-box
			// fallback, which accepts the cursor on the closing rule; the rule
			// cell must still decline for agy. The second rule row and footer
			// below the cursor are what the fallback's footer scan needs, the
			// shape its live capture paints.
			name:  "agy cursor on the closing rule",
			agent: "agy",
			stream: "\x1b[6;1H─────\x1b[7;1H> alpha" +
				"\x1b[8;1H─────\x1b[9;1H─────" +
				"\x1b[10;1H? for shortcuts Claude Sonnet 4.6 (Thinking)" +
				"\x1b[?25h\x1b[8;4H",
			want: false,
		},
		{
			// Codex paints no rules, and its recognizer already declines a
			// rule cell at column 0 wherever the cursor rests, so it should
			// never deliver one to the orientation scan. The row guards that
			// barrier if codex's scan is ever loosened.
			name:   "codex cursor on a divider row",
			agent:  "codex",
			stream: "\x1b[7;1H\x1b[1m›\x1b[22m alpha\x1b[8;1H─────\x1b[?25h\x1b[8;4H",
			want:   false,
		},
		{
			// The tolerance is real and per-profile: qoder's hidden cursor
			// parks on the closing rule of its own composer and the gate must
			// stay true, or the captured composer.raw stops auto-submitting.
			name:   "qoder cursor on the closing rule",
			agent:  "qoder",
			stream: qoderBox(6, ">", "alpha") + "\x1b[?25l\x1b[8;4H",
			want:   true,
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			p := &proxy{agentBasename: test.agent, stdout: io.Discard, lifecycleEvents: make(chan TurnObservation, 8)}
			p.orientation = newOrientationDelivery(orientation.Request{SchemaVersion: 1, Tag: "work", Agent: test.agent, Attempt: "n", Body: "read context"})
			if err := p.configureHarnessTTY(false, 120, 38); err != nil {
				t.Fatal(err)
			}
			defer p.closeTerminal()
			var rolling []byte
			p.handleChunk([]byte(test.stream), &rolling)
			if got := p.orientationComposerActive(p.terminal.Snapshot()); got != test.want {
				t.Fatalf("orientationComposerActive = %t, want %t", got, test.want)
			}
		})
	}
}

// TestOrientationUncoloredAgyRequiresVisibleCursor pins the fail-safe default
// of the spec's allowHiddenCursor field (BR-37): only Qoder's spec sets it, so
// the agy uncolored fallback must keep declining a snapshot whose system cursor
// is hidden even when the box is otherwise complete. The visible-cursor row is
// the control — without it a decline could come from the box shape rather than
// the cursor, and the hidden row would pin nothing.
func TestOrientationUncoloredAgyRequiresVisibleCursor(t *testing.T) {
	// The closing rule must sit directly below the prompt row: the fallback's
	// footer scan reads the row after that closing rule for the shortcuts line.
	box := "\x1b[6;1H─────\x1b[7;1H> alpha\x1b[8;1H─────" +
		"\x1b[9;1H? for shortcuts Claude Sonnet 4.6 (Thinking)"
	cases := []struct {
		name   string
		stream string
		want   bool
	}{
		{name: "visible cursor inside the composer", stream: box + "\x1b[?25h\x1b[7;5H", want: true},
		{name: "hidden cursor inside the composer", stream: box + "\x1b[?25l\x1b[7;5H", want: false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			p := &proxy{agentBasename: "agy", stdout: io.Discard, lifecycleEvents: make(chan TurnObservation, 8)}
			p.orientation = newOrientationDelivery(orientation.Request{SchemaVersion: 1, Tag: "work", Agent: "agy", Attempt: "n", Body: "read context"})
			if err := p.configureHarnessTTY(false, 120, 38); err != nil {
				t.Fatal(err)
			}
			defer p.closeTerminal()
			var rolling []byte
			p.handleChunk([]byte(test.stream), &rolling)
			if got := p.orientationComposerActive(p.terminal.Snapshot()); got != test.want {
				t.Fatalf("orientationComposerActive = %t, want %t", got, test.want)
			}
		})
	}
}
func TestOrientationChildEnvironmentAndReadinessStatus(t *testing.T) {
	isolateNotificationSockets(t)
	t.Setenv("PAIR_SCOPE_KEY", "")
	t.Setenv("PAIR_RETENTION_START_ID", "")
	dir := t.TempDir()
	fake := filepath.Join(dir, "claude")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nif [ -n \"${PAIR_ORIENTATION_REQUEST+x}\" ]; then exit 13; fi\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	request := orientation.Request{SchemaVersion: 1, Tag: "work", Agent: "claude", Attempt: "attempt", Body: "read context"}
	raw, _ := json.Marshal(request)
	t.Setenv(orientation.Env, string(raw))
	t.Setenv("PAIR_TAG", "work")
	t.Setenv("PAIR_SESSION_NAME", "pair-work")
	t.Setenv("PAIR_LAUNCH_NONCE", "attempt")
	t.Setenv("PAIR_DATA_DIR", dir)
	command, _ := launcher.EncodeAgentCommand(launcher.AgentCommand{Executable: fake, Argv: []string{}})
	t.Setenv(launcher.AgentCommandEnv, command)
	master, tty, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer tty.Close()
	var stderr bytes.Buffer
	if code := Run([]string{"--from-launch-env"}, tty, io.Discard, &stderr); code != 0 {
		t.Fatalf("child code %d: %s", code, &stderr)
	}
	recordRaw, err := os.ReadFile(filepath.Join(dir, "agent-ready-work-claude.json"))
	if err != nil {
		t.Fatal(err)
	}
	record, err := readiness.Decode(string(recordRaw))
	if err != nil {
		t.Fatal(err)
	}
	if record.Nonce != "attempt" || record.Tag != "work" || record.Agent != "claude" || record.Session != "pair-work" || record.Orientation == nil || record.Orientation.Phase != orientation.DeliveryCancelled {
		t.Fatalf("ready %#v", record)
	}
	next, err := freshAgentInvocation("/pair", "", []string{"claude"}, []string{orientation.Env + "=" + string(raw)}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if envValue(next.env, orientation.Env) != "" {
		t.Fatal("reexec retained consumed request")
	}
}

func TestOrientationDeclinesCapturedMenusAndNonCodingMode(t *testing.T) {
	paths, _ := filepath.Glob("testdata/tty/*/*/menu.raw")
	overlays, _ := filepath.Glob("testdata/tty/*/*/overlay.raw")
	paths = append(paths, overlays...)
	modes, _ := filepath.Glob("testdata/tty/claude/*/bash-mode.raw")
	paths = append(paths, modes...)
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			agent := filepath.Base(filepath.Dir(filepath.Dir(path)))
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			p := &proxy{agentBasename: agent, stdout: io.Discard}
			p.orientation = newOrientationDelivery(orientation.Request{SchemaVersion: 1, Tag: "t", Agent: agent, Attempt: "n", Body: "read context"})
			states := make(chan orientation.DeliveryState, 2)
			p.orientation.publish = func(s orientation.DeliveryState) { states <- s }
			p.orientation.deadlineAfter = 10 * time.Millisecond
			p.orientation.settleAfter = time.Millisecond
			if err := p.configureHarnessTTY(false, 120, 38); err != nil {
				t.Fatal(err)
			}
			defer p.closeTerminal()
			r, w := io.Pipe()
			defer r.Close()
			out := newDrainBuffer()
			done := make(chan struct{})
			go func() { p.translateStdinFrom(r, out, time.Millisecond); close(done) }()
			var rolling []byte
			p.handleChunk(raw, &rolling)
			select {
			case state := <-states:
				if state.Phase == orientation.DeliverySubmitted {
					t.Errorf("submitted into captured menu/mode %s", path)
				}
			case <-time.After(time.Second):
				t.Error("no timeout status")
			}
			w.Close()
			<-done
			if len(out.Bytes()) != 0 {
				t.Errorf("wrote into menu/mode: %q", out.Bytes())
			}
		})
	}
}

type failOrientationSubmitWriter struct {
	*drainBuffer
	calls int
}

func (w *failOrientationSubmitWriter) Write(b []byte) (int, error) {
	w.calls++
	if w.calls == 2 {
		return 0, errors.New("submit failed")
	}
	return w.drainBuffer.Write(b)
}
func TestOrientationSubmitFailureAndLaterInput(t *testing.T) {
	for _, failSubmit := range []bool{false, true} {
		t.Run(fmt.Sprint(failSubmit), func(t *testing.T) {
			p, states := orientationProxy(t)
			r, w := io.Pipe()
			defer r.Close()
			var out io.Writer
			recorded := newDrainBuffer()
			if failSubmit {
				out = &failOrientationSubmitWriter{drainBuffer: recorded}
			} else {
				out = recorded
			}
			done := make(chan struct{})
			go func() { p.translateStdinFrom(r, out, time.Millisecond); close(done) }()
			var rolling []byte
			p.handleChunk([]byte(claudeLiveComposerPaint()), &rolling)
			var state orientation.DeliveryState
			select {
			case state = <-states:
			case <-time.After(time.Second):
				t.Fatal("no result")
			}
			want := orientation.DeliverySubmitted
			if failSubmit {
				want = orientation.DeliveryIndeterminate
			}
			if state.Phase != want || !state.BodyWritten {
				t.Fatalf("state %#v", state)
			}
			w.Write([]byte("later operator input"))
			w.Close()
			<-done
			if !bytes.HasSuffix(recorded.Bytes(), []byte("later operator input")) {
				t.Fatalf("later input lost %q", recorded.Bytes())
			}
			select {
			case other := <-states:
				t.Fatalf("revised terminal status %#v", other)
			default:
			}
		})
	}
}
