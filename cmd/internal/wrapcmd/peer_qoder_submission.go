package wrapcmd

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// qoderSubmissionEvidence compares the complete qualified composer with the
// expected native word-wrapped envelope. Input admission must independently
// prove no human input since the paste: screen equality is not editor authority.
func qoderSubmissionEvidence(s terminalSnapshot, envelope string) (PeerComposerState, bool) {
	if !s.Complete {
		return PeerComposerUnknown, false
	}
	state := peerComposerState("qoder", s)
	if state != PeerComposerOccupied || !utf8.ValidString(envelope) {
		return state, false
	}
	for _, r := range envelope {
		if r != '\n' && unicode.IsControl(r) {
			return state, false
		}
	}
	for _, line := range strings.Split(envelope, "\n") {
		if strings.TrimRight(line, " ") != line {
			return state, false
		}
	}
	text, known := peerComposerText("qoder", s)
	if !known || s.Width <= peerInputColumn("qoder") {
		return state, false
	}
	expected := ansi.Wrap(envelope, s.Width-peerInputColumn("qoder"), "")
	return state, text == expected
}

// Called at the terminal publication event while the input owner still holds
// admission and delivery locks. Reporting is queued; no IO runs on this path.
func (p *proxy) quarantineQoderInput() {
	d := p.peer
	var snapshot terminalSnapshot
	cause := couchmessage.AutomaticInputStallUnrecognizedComposer
	if p.terminal != nil {
		snapshot = p.terminal.Snapshot()
		_, owned := qoderSubmissionEvidence(snapshot, peerEnvelope(d.current.Message))
		if owned {
			cause = couchmessage.AutomaticInputStallUnsubmittedComposer
		}
	}
	p.applyAutomaticInputEvent(automaticInputEvent{kind: automaticQuarantine, owner: automaticInputPeer, epoch: p.automaticRenderEpoch.Load(), reportCause: cause, wait: d.now().Sub(d.pastedAt)}, snapshot)
}
