package wrapcmd

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/adapt"
)

func TestHarnessTTYProfileRegistry(t *testing.T) {
	type wantProfile struct {
		plainCR, altCR, altBS []byte
		overlay               overlayDetector
		gate                  composerGatePolicy
		captureSetsOverlay    bool
	}

	ctrlU := []byte{0x15}
	tests := map[string]wantProfile{
		"claude": {[]byte{'\\', '\r'}, []byte{'\r'}, ctrlU, detectClaudeOverlayOpen, composerGateLegacy, false},
		"codex":  {[]byte{'\n'}, []byte{'\r'}, ctrlU, detectCodexOverlayOpen, composerGatePositive, true},
		"agy":    {[]byte{'\n'}, []byte{'\r'}, ctrlU, detectAgyOverlayOpen, composerGatePositive, false},
		"muse":   {[]byte{'\n'}, []byte{'\r'}, ctrlU, detectMuseOverlayOpen, composerGatePositive, false},
	}

	for harness, want := range tests {
		t.Run(harness, func(t *testing.T) {
			got, ok := profileForHarness(harness, true)
			if !ok {
				t.Fatalf("profileForHarness(%q, true) did not find profile", harness)
			}
			if !bytes.Equal(got.keymap.plainCR, want.plainCR) {
				t.Errorf("plainCR = %q, want %q", got.keymap.plainCR, want.plainCR)
			}
			if !bytes.Equal(got.keymap.altCR, want.altCR) {
				t.Errorf("altCR = %q, want %q", got.keymap.altCR, want.altCR)
			}
			if !bytes.Equal(got.keymap.altBS, want.altBS) {
				t.Errorf("altBS = %q, want %q", got.keymap.altBS, want.altBS)
			}
			if reflect.ValueOf(got.overlay).Pointer() != reflect.ValueOf(want.overlay).Pointer() {
				t.Errorf("overlay detector does not match %q registration", harness)
			}
			if got.composerGate != want.gate {
				t.Errorf("composerGate = %v, want %v", got.composerGate, want.gate)
			}
			if got.recognize != nil {
				t.Error("recognize must remain nil until the harness recognizer tasks")
			}
			if got.captureSetsOverlay != want.captureSetsOverlay {
				t.Errorf("captureSetsOverlay = %t, want %t", got.captureSetsOverlay, want.captureSetsOverlay)
			}
		})
	}
}

func TestHarnessTTYProfileSelection(t *testing.T) {
	if _, ok := profileForHarness("unknown", true); ok {
		t.Fatal("unknown harness unexpectedly selected a profile")
	}
	if _, ok := profileForHarness("codex", false); ok {
		t.Fatal("remap-disabled harness unexpectedly selected a profile")
	}
}

func TestDecidePlainReturn(t *testing.T) {
	activeSnapshot := terminalSnapshot{Width: 1, Height: 1, CursorVisible: true}
	inactiveSnapshot := terminalSnapshot{Width: 1, Height: 1}
	positive := harnessTTYProfile{
		keymap:       sendKeymap{plainCR: []byte{'\n'}},
		composerGate: composerGatePositive,
		recognize: func(snapshot terminalSnapshot) bool {
			return snapshot.CursorVisible
		},
	}
	legacy := harnessTTYProfile{
		keymap:       sendKeymap{plainCR: []byte{'\\', '\r'}},
		composerGate: composerGateLegacy,
	}
	withoutRecognizer := harnessTTYProfile{
		keymap:       sendKeymap{plainCR: []byte{'\n'}},
		composerGate: composerGatePositive,
	}

	tests := []struct {
		name        string
		profile     harnessTTYProfile
		overlay     bool
		snapshot    *terminalSnapshot
		wantBytes   []byte
		wantOutcome adapt.Outcome
		wantReason  string
		wantClear   bool
	}{
		{"overlay wins over active composer", positive, true, &activeSnapshot, []byte{'\r'}, adapt.Bypass, "plain Enter → bare CR (overlay active)", true},
		{"active positive composer", positive, false, &activeSnapshot, []byte{'\n'}, adapt.Fired, "plain Enter → newline remap", false},
		{"inactive positive composer", positive, false, &inactiveSnapshot, []byte{'\r'}, adapt.Bypass, "plain Enter → bare CR (composer inactive)", false},
		{"unknown positive composer state", positive, false, nil, []byte{'\r'}, adapt.Bypass, "plain Enter → bare CR (composer unknown)", false},
		{"positive profile without registered recognizer", withoutRecognizer, false, &activeSnapshot, []byte{'\r'}, adapt.Bypass, "plain Enter → bare CR (composer unknown)", false},
		{"legacy profile preserves remap", legacy, false, nil, []byte{'\\', '\r'}, adapt.Fired, "plain Enter → newline remap", false},
		{"all-zero profile fails closed", harnessTTYProfile{}, false, nil, []byte{'\r'}, adapt.Bypass, "plain Enter → bare CR (composer unknown)", false},
		{"invalid gate policy fails closed", harnessTTYProfile{composerGate: composerGatePolicy(255)}, false, &activeSnapshot, []byte{'\r'}, adapt.Bypass, "plain Enter → bare CR (composer unknown)", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := decidePlainReturn(test.profile, test.overlay, test.snapshot)
			if !bytes.Equal(got.bytes, test.wantBytes) {
				t.Errorf("bytes = %q, want %q", got.bytes, test.wantBytes)
			}
			if got.outcome != test.wantOutcome {
				t.Errorf("outcome = %q, want %q", got.outcome, test.wantOutcome)
			}
			if got.reason != test.wantReason {
				t.Errorf("reason = %q, want %q", got.reason, test.wantReason)
			}
			if got.clearOverlay != test.wantClear {
				t.Errorf("clearOverlay = %t, want %t", got.clearOverlay, test.wantClear)
			}
		})
	}
}
