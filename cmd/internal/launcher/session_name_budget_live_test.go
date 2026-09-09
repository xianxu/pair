package launcher

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// The byte budget is measured against REAL zellij, not a fake (#215 BR-2).
//
// Gated on PAIR_LIVE_COUCH=1 with t.Skip and deliberately no build tag, matching
// the other live conformance tests.
//
// WHY THIS SEAM NEEDS A LIVE CHECK NOW. #215 made a candidate's acceptability an
// arithmetic question answered from one measured number. Before, a missed
// rejection mis-accepted ONE candidate and the ladder kept walking; now a wrong
// budget decides EVERY remaining candidate without asking. The whole chain rests
// on sessionNameRejected (zellijparse.go) matching one substring of zellij's
// stderr, and a zellij release that rewords that message would silently disable
// the oracle with every unit test still green.
//
// So this asserts the BOUNDARY rather than a value: a name at exactly `budget`
// bytes is accepted and one byte more is refused. The number itself is a
// property of this machine's socket directory and is not worth pinning.
func TestSessionNameBudgetMatchesRealZellijLive(t *testing.T) {
	if os.Getenv("PAIR_LIVE_COUCH") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1 to measure the budget against real zellij")
	}
	rt := OSRuntime{}
	probe := func(name string) bool { return rt.ProbeSessionName(name) == nil }

	budget, measured := discoverSessionNameBudget(probe)
	if !measured {
		t.Skipf("real zellij refuses even a %d-byte name here, so there is no boundary "+
			"to check; the acceptor correctly falls back to probing per candidate",
			len(sessionNameProbeMarker))
	}
	t.Logf("measured budget on this machine: %d bytes", budget)

	pad := func(n int) string {
		return sessionNameProbeMarker + strings.Repeat("z", n-len(sessionNameProbeMarker))
	}
	if !probe(pad(budget)) {
		t.Errorf("real zellij REFUSED a %d-byte name at the measured budget; the "+
			"arithmetic oracle would accept every candidate this size", budget)
	}
	if probe(pad(budget + 1)) {
		t.Errorf("real zellij ACCEPTED a %d-byte name, one past the measured budget of "+
			"%d; the oracle is rejecting names zellij would take, and the ladder will "+
			"hand back a shorter name than it needed to", budget+1, budget)
	}

	// And the classifier must still recognise zellij's OWN refusal text.
	//
	// Against zellij's raw output, not against ProbeSessionName's error: that
	// error is pair's wrapper ("session name too long: ..."), and feeding it back
	// to the classifier tests the wrapper's spelling rather than zellij's. The
	// first cut of this test did exactly that and reported a defect that was not
	// there.
	//
	// This is the assertion that catches a zellij release rewording its message:
	// sessionNameRejected matches ONE substring, and if it stops matching, every
	// over-long candidate reads as acceptable and the arithmetic budget is
	// measured as 64 -- with every unit test in this package still green.
	out, _ := exec.Command("zellij", "--session", pad(budget+1),
		"action", "list-clients").CombinedOutput()
	if !sessionNameRejected(string(out)) {
		t.Errorf("zellij refused a %d-byte name but sessionNameRejected did not "+
			"recognise its message, so over-long candidates now read as ACCEPTABLE.\n"+
			"zellij said: %q\nthe classifier looks for: %q",
			budget+1, strings.TrimSpace(string(out)), "session name must be less than")
	}
}
