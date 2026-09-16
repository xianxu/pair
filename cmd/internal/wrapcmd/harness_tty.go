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
		composerGate: composerGatePositive,
		recognize:    claudeComposerActive,
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
	// Muse's composer submits on bare CR and inserts a newline on Shift+Return
	// — the inverse of the other three — so the remap translates plain Return
	// into Shift+Return rather than into a literal newline byte. That reading
	// of Muse comes from its own shortcut sheet (captured as shortcuts.raw:
	// "shift + enter for newline", "enter to submit message") and from live
	// operator smoke; no test presses the key, which ttyFixtureReactionGaps
	// records rather than leaving this comment to imply otherwise.
	//
	// PRECONDITION: `\x1b[13;2u` is a Kitty keyboard protocol key, parseable
	// only while Muse keeps progressive enhancement pushed. Muse 1.3.0 does push
	// it (a `CSI > ... u` push, in every captured muse fixture). If that stops,
	// Return would insert those bytes as literal text while return-remap
	// telemetry still reported `fired` — so the push is asserted against the
	// capture itself by assertKittyKeyboardPrecondition, not trusted from here.
	"muse": {
		keymap: sendKeymap{
			plainCR: []byte("\x1b[13;2u"),
			altCR:   []byte{'\r'},
			altBS:   []byte{0x15},
		},
		overlay:      detectMuseOverlayOpen,
		composerGate: composerGatePositive,
		recognize:    museComposerActive,
	},
}

// profileForHarness returns a copy whose mutable keymap slices are the
// caller's own. The func and enum fields are values shared by copy, so callers
// must not treat the result as deeply isolated.
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
	// submits reports that these bytes reach the AGENT as a turn-opening
	// submission, so the notification floor may open a turn on them (#171).
	// It is not the same as `outcome == adapt.Bypass`: the overlay-active
	// bypass confirms a pair-local picker and submits nothing to the agent,
	// and remapping to a composer newline is not a submission either. Keying
	// the floor on Bypass alone would open a turn on a picker confirm and
	// then raise a spurious "no agent output" alert against it.
	submits bool
}

func decidePlainReturn(profile harnessTTYProfile, overlayActive bool, snapshot *terminalSnapshot) returnDecision {
	if overlayActive {
		// The caller clears the overlay under overlayMu before deciding, so
		// the decision reports only what to emit.
		return returnDecision{
			bytes:   []byte{'\r'},
			outcome: adapt.Bypass,
			reason:  "plain Enter → bare CR (overlay active)",
			submits: false, // pair-local picker confirm; never reaches the agent
		}
	}
	// An empty plainCR would report Fired while emitting nothing, swallowing
	// the user's Enter. The gate enum fails closed on its zero value; so does
	// the keymap.
	if len(profile.keymap.plainCR) == 0 {
		return unknownComposerDecision()
	}
	remap := func() returnDecision {
		return returnDecision{
			bytes:   append([]byte(nil), profile.keymap.plainCR...),
			outcome: adapt.Fired,
			reason:  "plain Enter → newline remap",
			submits: false, // a newline inside the composer, not a send
		}
	}
	switch profile.composerGate {
	case composerGateLegacy:
		return remap()
	case composerGatePositive:
		if snapshot == nil || profile.recognize == nil {
			return unknownComposerDecision()
		}
		if !profile.recognize(*snapshot) {
			return returnDecision{
				bytes:   []byte{'\r'},
				outcome: adapt.Bypass,
				reason:  "plain Enter → bare CR (composer inactive)",
				submits: true, // the CR reaches the agent (menu answer)
			}
		}
		return remap()
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
		submits: true, // the CR reaches the agent; fail open for the floor
	}
}
