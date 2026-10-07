package couchtty

import (
	"os"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// The reattach pass and the queue workers take their bound from
// couchcore.LifecycleParallelism (pair#205). Pin it to 1 so tests written for
// one attempt in flight keep their meaning on any host; tests of the bounded
// behaviour set ReattachPass.Limit explicitly.
func TestMain(m *testing.M) {
	couchcore.LifecycleParallelism = 1
	os.Exit(m.Run())
}

// soleLoading is the one thread in flight, for tests that run with Limit 1;
// zero when none is.
func (pass ReattachPass) soleLoading() couchcore.ThreadAddress {
	for _, address := range pass.InFlight {
		return address
	}
	return couchcore.ThreadAddress{}
}

// soleLoadingAttempt is that thread's attempt; zero when none is in flight.
func (pass ReattachPass) soleLoadingAttempt() uint64 {
	for attempt := range pass.InFlight {
		return attempt
	}
	return 0
}
