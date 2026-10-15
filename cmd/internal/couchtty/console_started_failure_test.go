package couchtty

import (
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestConsoleReattachEmptyStartedResultKeepsFailureOrigin(t *testing.T) {
	f := newFixtureBeforeRun(t, 24, 100, func(c *Console) {
		root := c.panes["c1"].thread
		c.SetOperationDispatcher(func(couchcore.OperationCall) (any, error) {
			// A typed zero result must not erase the attempt's exact address.
			return couchcore.StartResult{}, errors.New("worker attachment unavailable")
		})
		c.ArmReattachPass(root)
		c.SetActionableProvider(detachedFleet(root, 1))
	})
	waitFor(t, "empty started result completes the failed Continue attempt", func() bool {
		state := f.con.menuSnapshot()
		if state.Reattach.Phase != ReattachDone || len(state.Reattach.InFlight) != 0 || len(state.Reattach.Failed) != 1 {
			return false
		}
		for address, reason := range state.Reattach.Failed {
			return address != (couchcore.ThreadAddress{}) && reason == "worker attachment unavailable"
		}
		return false
	})
}
