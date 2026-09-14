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
	for _, agent := range []string{"claude", "codex", "agy", "muse"} {
		if !seen[agent] {
			t.Errorf("missing %s fixture", agent)
		}
	}
}
func TestOrientationChildEnvironmentAndReadinessStatus(t *testing.T) {
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
