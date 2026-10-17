package wrapcmd

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/pairlog"
)

// peerTurnRecordLimit bounds what one wrapper launch adds to the Pair log:
// at most this many envelopes, each at most couchmessage.MaxBodyBytes plus its
// header (about 64 KB per launch). Correlation needs one completed round after
// the launch to confirm a binding, and a binding is confirmed per launch, so
// the first few submitted turns serve; later ones would only grow a log that
// every append and the draft pane's status line re-parse.
const peerTurnRecordLimit = 8

// peerTurnRecorder returns the recorder a submitted peer delivery calls
// (d.mu held, so it only starts the write). The session watcher confirms a
// thread's binding from Pair-log turns it finds in the agent's transcript;
// without this a slot driven only by peer messages never confirms and stays
// unrelaunchable (pair#618). The entry is the exact envelope pasted, so it
// hashes like the agent's copy of the turn. A failed write costs one
// correlation fact, never the delivery, so it is only logged.
func peerTurnRecorder(path string, getenv func(string) string, debug func(string, string), workers *sync.WaitGroup) func(couchmessage.Message) {
	var recorded atomic.Int32
	return func(m couchmessage.Message) {
		if recorded.Add(1) > peerTurnRecordLimit {
			return
		}
		envelope := []byte(peerEnvelope(m))
		workers.Add(1)
		go func() {
			defer workers.Done()
			if err := pairlog.PersistPeerTurn(getenv, path, envelope, time.Now(), m.ID); err != nil {
				debug("PEER-turn-log-fail", m.ID+": "+err.Error())
			}
		}()
	}
}
