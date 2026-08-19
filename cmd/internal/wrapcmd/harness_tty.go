package wrapcmd

import "github.com/xianxu/pair/cmd/internal/adapt"

type composerGatePolicy uint8

const (
	composerGateUnknown composerGatePolicy = iota
	composerGateLegacy
	composerGatePositive
)

type composerRecognizer func(terminalSnapshot) bool

type harnessTTYProfile struct {
	keymap             sendKeymap
	overlay            overlayDetector
	composerGate       composerGatePolicy
	recognize          composerRecognizer
	captureSetsOverlay bool
}

var harnessTTYProfiles = map[string]harnessTTYProfile{
	"claude": {
		keymap: sendKeymap{
			plainCR: []byte{'\\', '\r'},
			altCR:   []byte{'\r'},
			altBS:   []byte{0x15},
		},
		overlay:      detectClaudeOverlayOpen,
		composerGate: composerGateLegacy,
	},
	"codex": {
		keymap: sendKeymap{
			plainCR: []byte{'\n'},
			altCR:   []byte{'\r'},
			altBS:   []byte{0x15},
		},
		overlay:            detectCodexOverlayOpen,
		composerGate:       composerGatePositive,
		recognize:          codexComposerActive,
		captureSetsOverlay: true,
	},
	"agy": {
		keymap: sendKeymap{
			plainCR: []byte{'\n'},
			altCR:   []byte{'\r'},
			altBS:   []byte{0x15},
		},
		overlay:      detectAgyOverlayOpen,
		composerGate: composerGatePositive,
		recognize:    agyComposerActive,
	},
	"muse": {
		keymap: sendKeymap{
			plainCR: []byte{'\n'},
			altCR:   []byte{'\r'},
			altBS:   []byte{0x15},
		},
		overlay:      detectMuseOverlayOpen,
		composerGate: composerGatePositive,
		recognize:    museComposerActive,
	},
}

func profileForHarness(harness string, remapEnabled bool) (harnessTTYProfile, bool) {
	if !remapEnabled {
		return harnessTTYProfile{}, false
	}
	profile, ok := harnessTTYProfiles[harness]
	if !ok {
		return harnessTTYProfile{}, false
	}
	profile.keymap.plainCR = append([]byte(nil), profile.keymap.plainCR...)
	profile.keymap.altCR = append([]byte(nil), profile.keymap.altCR...)
	profile.keymap.altBS = append([]byte(nil), profile.keymap.altBS...)
	return profile, true
}

type returnDecision struct {
	bytes   []byte
	outcome adapt.Outcome
	reason  string
}

func decidePlainReturn(profile harnessTTYProfile, overlayActive bool, snapshot *terminalSnapshot) returnDecision {
	if overlayActive {
		// The caller clears the overlay under overlayMu before deciding, so
		// the decision reports only what to emit.
		return returnDecision{
			bytes:   []byte{'\r'},
			outcome: adapt.Bypass,
			reason:  "plain Enter → bare CR (overlay active)",
		}
	}
	switch profile.composerGate {
	case composerGateLegacy:
		return returnDecision{
			bytes:   append([]byte(nil), profile.keymap.plainCR...),
			outcome: adapt.Fired,
			reason:  "plain Enter → newline remap",
		}
	case composerGatePositive:
		if snapshot == nil || profile.recognize == nil {
			return unknownComposerDecision()
		}
		if !profile.recognize(*snapshot) {
			return returnDecision{
				bytes:   []byte{'\r'},
				outcome: adapt.Bypass,
				reason:  "plain Enter → bare CR (composer inactive)",
			}
		}
		return returnDecision{
			bytes:   append([]byte(nil), profile.keymap.plainCR...),
			outcome: adapt.Fired,
			reason:  "plain Enter → newline remap",
		}
	case composerGateUnknown:
		return unknownComposerDecision()
	default:
		return unknownComposerDecision()
	}
}

func unknownComposerDecision() returnDecision {
	return returnDecision{
		bytes:   []byte{'\r'},
		outcome: adapt.Bypass,
		reason:  "plain Enter → bare CR (composer unknown)",
	}
}
