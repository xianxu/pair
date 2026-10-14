package wrapcmd

import (
	"strings"
	"testing"
)

// After an automatic write is submitted, the next one waits for a repaint
// that shows the composer empty: the stale snapshot that preceded the submit
// is not evidence. Peer delivery is the one automatic writer since #522 retired
// the orientation paste, and consecutive deliveries share this lease.
func TestAutomaticInputSubmittedOwnerWaitsForEmptyRepaint(t *testing.T) {
	f, _ := peerIntegrationFixture(t)
	p := f.proxy
	if !p.automaticInputAvailable(automaticInputPeer) {
		t.Fatal("an idle lease refused its writer")
	}
	p.automaticInput.phase = automaticInputPeer
	p.finishAutomaticInput(automaticInputPeer, false)
	if p.automaticInputAvailable(automaticInputPeer) {
		t.Fatal("the next write ran before any repaint")
	}
	f.output("\x1b[2J" + claudeBox(5, "❯", "136;136;136", "leftover") + "\x1b[?25h\x1b[7;3H")
	if p.automaticInputAvailable(automaticInputPeer) {
		t.Fatal("the next write ran over a composer that still holds text")
	}
	f.output("\x1b[2J" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H")
	if !p.automaticInputAvailable(automaticInputPeer) {
		t.Fatal("an empty repaint did not release the lease")
	}
}
