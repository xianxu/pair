package wrapcmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEmitOuterDoesNotCommitRateLimitAfterShortWrite(t *testing.T) {
	dir := t.TempDir()
	tty := filepath.Join(dir, "tty")
	if err := os.WriteFile(tty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	sidecar := filepath.Join(dir, "outer")
	if err := os.WriteFile(sidecar, []byte(tty+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	p := &proxy{
		outerTTYFile: sidecar,
		lastSlug:     time.Now(),
		writeTTY: func(_ int, p []byte) (int, error) {
			return len(p) - 1, nil
		},
	}
	p.emitOuter("ready")
	if !p.lastEmit.IsZero() {
		t.Fatal("short write was committed as a successful notification")
	}
}

func TestNotificationRewriterEverySplit(t *testing.T) {
	cases := []struct {
		name         string
		input        []byte
		wantOutput   []byte
		wantMessages []string
	}{
		{"osc9", []byte("a\x1b]9;needs input\x07b"), []byte("ab"), []string{"needs input"}},
		{"osc777", []byte("a\x1b]777;notify;Claude;build; finished\x1b\\b"), []byte("ab"), []string{"build; finished"}},
		{"title fallback", []byte("\x1b]777;notify;Claude;\x07"), nil, []string{"Claude"}},
		{"empty fallback", []byte("\x1b]777;notify;;\x07"), nil, []string{"agent attention"}},
		{"progress", []byte("\x1b]9;4;75\x07"), []byte("\x1b]9;4;75\x07"), nil},
		{"unknown", []byte("\x1b]52;c;data\x07"), []byte("\x1b]52;c;data\x07"), nil},
		{"malformed 777", []byte("\x1b]777;notify;title\x07"), []byte("\x1b]777;notify;title\x07"), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for split := 0; split <= len(tc.input); split++ {
				var r NotificationRewriter
				var output []byte
				var messages []string
				for _, chunk := range [][]byte{tc.input[:split], tc.input[split:]} {
					result := r.Feed(chunk, true)
					output = append(output, result.Passthrough...)
					for _, n := range result.Notifications {
						messages = append(messages, n.Message)
					}
				}
				if !bytes.Equal(output, tc.wantOutput) || !equalStrings(messages, tc.wantMessages) {
					t.Fatalf("split %d: output=%q messages=%q; want %q %q", split, output, messages, tc.wantOutput, tc.wantMessages)
				}
			}
		})
	}
}

func TestNotificationRewriterProgressEverySplitPreservesBytesAndObservesState(t *testing.T) {
	cases := []struct {
		name  string
		input []byte
		want  ObservationKind
	}{
		{"working", []byte("a\x1b]9;4;3;\x07b"), ObservationWorking},
		{"stopped", []byte("a\x1b]9;4;0;\x1b\\b"), ObservationStopped},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, normalize := range []bool{false, true} {
				for split := 0; split <= len(tc.input); split++ {
					var r NotificationRewriter
					var output []byte
					var observations []TurnObservation
					for _, chunk := range [][]byte{tc.input[:split], tc.input[split:]} {
						got := r.Feed(chunk, normalize)
						output = append(output, got.Passthrough...)
						observations = append(observations, got.Observations...)
					}
					if !bytes.Equal(output, tc.input) {
						t.Fatalf("normalize %v split %d changed bytes: got %q want %q", normalize, split, output, tc.input)
					}
					if len(observations) != 1 || observations[0].Kind != tc.want {
						t.Fatalf("normalize %v split %d observations = %+v, want one %v", normalize, split, observations, tc.want)
					}
				}
			}
		})
	}
}

func TestNotificationRewriterDisabledIsTransparent(t *testing.T) {
	input := []byte("a\x1b]9;needs input\x07b")
	var r NotificationRewriter
	got := r.Feed(input, false)
	if !bytes.Equal(got.Passthrough, input) || len(got.Notifications) != 0 {
		t.Fatalf("Feed(disabled) = %+v", got)
	}
}

func TestNotificationRewriterOverlongIsBoundedAndTransparent(t *testing.T) {
	var r NotificationRewriter
	start := append([]byte("\x1b]9;"), bytes.Repeat([]byte("x"), notificationRewriteMaxPending+1)...)
	got := r.Feed(start, true)
	if !bytes.Equal(got.Passthrough, start) || len(r.pending) != 0 || !r.skippingOSC {
		t.Fatalf("overlong result=%+v pending=%d skipping=%v", got, len(r.pending), r.skippingOSC)
	}
	end := []byte("tail\x07after")
	got = r.Feed(end, true)
	if !bytes.Equal(got.Passthrough, end) || r.skippingOSC || len(got.Notifications) != 0 {
		t.Fatalf("terminator result=%+v skipping=%v", got, r.skippingOSC)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestProxyNativeNotificationCanonicalEmission(t *testing.T) {
	for _, harness := range []string{"codex", "claude"} {
		t.Run(harness, func(t *testing.T) {
			dir := t.TempDir()
			outer := filepath.Join(dir, "outer")
			if err := os.WriteFile(outer, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			sidecar := filepath.Join(dir, "outer-path")
			if err := os.WriteFile(sidecar, []byte(outer+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			var stdout bytes.Buffer
			now := time.Unix(1_800_000_000, 0)
			f := newHarnessSessionFake(t, harness, true)
			t.Cleanup(f.close)
			p := f.proxy
			p.notifyModeActive = "native"
			p.outerTTYFile = sidecar
			p.stdoutPump = newStdoutPump(&stdout)
			p.lastSlug = now
			p.now = func() time.Time { return now }
			p.lifecycleEvents = make(chan TurnObservation, 2)
			if harness == "codex" {
				f.output(codexLiveComposerPaint())
			} else {
				f.output(claudeLiveComposerPaint())
			}
			_ = f.altEnter()
			p.processLifecycleObservation(<-p.lifecycleEvents)

			p.handleChunk([]byte("a\x1b]777;notify;Agent;needs input\x07b"), &f.rolling)
			p.flushStdout("test")
			written, err := os.ReadFile(outer)
			if err != nil {
				t.Fatal(err)
			}
			if string(written) != "\x1b]777;notify;pair;needs input\x07" {
				t.Fatalf("outer = %q", written)
			}
		})
	}
}

func TestProxyProgressOpenedNativeNotificationCanonicalEmission(t *testing.T) {
	dir := t.TempDir()
	outer := filepath.Join(dir, "outer")
	if err := os.WriteFile(outer, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	sidecar := filepath.Join(dir, "outer-path")
	if err := os.WriteFile(sidecar, []byte(outer+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0)
	p := &proxy{notifyModeActive: "native", outerTTYFile: sidecar, stdoutPump: newStdoutPump(io.Discard), lastSlug: now, now: func() time.Time { return now }}
	rolling := []byte(nil)
	p.handleChunk([]byte("\x1b]9;4;3;\x07\x1b]777;notify;Agent;done\x07"), &rolling)
	p.flushStdout("test")
	written, err := os.ReadFile(outer)
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != "\x1b]777;notify;pair;done\x07" {
		t.Fatalf("outer = %q", written)
	}
}

func TestProxyUnknownOSCPassthrough(t *testing.T) {
	input := []byte("a\x1b]52;c;data\x07b")
	var stdout bytes.Buffer
	p := &proxy{
		agentBasename:    "claude",
		notifyModeActive: "native",
		stdoutPump:       newStdoutPump(&stdout),
		now:              time.Now,
	}
	rolling := []byte(nil)
	p.handleChunk(input, &rolling)
	p.flushStdout("test")
	if !bytes.Equal(stdout.Bytes(), input) {
		t.Fatalf("stdout = %q, want %q", stdout.Bytes(), input)
	}
}
