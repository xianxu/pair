package launcher

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// The acceptor agrees with REAL zellij, not just with fakeRuntime (#215 BR-2).
//
// Gated on PAIR_LIVE_COUCH=1 with t.Skip and deliberately no build tag, matching
// the other live conformance tests.
//
// WHY THIS SEAM NEEDS A LIVE CHECK. #215 made a candidate's acceptability an
// arithmetic question answered from observations of this machine. Before, every
// candidate was probed, so a missed rejection mis-accepted ONE name; now one
// wrong observation decides every candidate of that length or longer. The whole
// chain rests on sessionNameRejected (zellijparse.go) matching one substring of
// zellij's stderr, and a zellij release rewording that message would silently
// turn the oracle into "accept everything" with every unit test still green.
//
// It drives the PRODUCTION acceptor rather than a helper, so what is pinned is
// what runs.
func TestTheAcceptorAgreesWithRealZellijLive(t *testing.T) {
	if os.Getenv("PAIR_LIVE_COUCH") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1 to check the acceptor against real zellij")
	}
	rt := OSRuntime{}
	accepts, refusedAt := sessionNameAcceptor(rt)

	const marker = "pair-probe-zz"
	pad := func(n int) string { return marker + strings.Repeat("z", n-len(marker)) }

	// A name no socket path can hold must be refused. If sessionNameRejected has
	// stopped recognising zellij's message, ProbeSessionName returns nil and this
	// is the assertion that catches it.
	if accepts(pad(200)) {
		out, _ := exec.Command("zellij", "--session", pad(200), "action", "list-clients").CombinedOutput()
		t.Fatalf("a 200-byte session name was ACCEPTED, so over-long names now read as "+
			"usable and the arithmetic oracle accepts everything.\nzellij said: %q",
			strings.TrimSpace(string(out)))
	}
	limit, ok := refusedAt()
	if !ok {
		t.Fatal("a refusal happened but the acceptor reports none observed")
	}
	t.Logf("shortest refusal observed on this machine: %d bytes", limit)

	// A short name must be accepted -- the positive control. Without it, an
	// acceptor that refused everything would pass the check above.
	if !accepts(marker) {
		t.Fatalf("real zellij refused even a %d-byte name; there is no usable budget "+
			"on this machine at all", len(marker))
	}

	// And the acceptor must agree with a raw probe at every length, which is the
	// property the bracket rests on (acceptance is monotone in length).
	fresh, _ := sessionNameAcceptor(rt)
	for n := len(marker); n <= 60; n++ {
		want := rt.ProbeSessionName(pad(n)) == nil
		if got := fresh(pad(n)); got != want {
			t.Fatalf("acceptor said %v for a %d-byte name, a direct probe says %v",
				got, n, want)
		}
	}
}
