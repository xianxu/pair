package wrapcmd

import (
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/pairlog"
)

// peerTurnRecorder returns the recorder a submitted peer delivery calls
// (d.mu held, so it only starts the write). The session watcher confirms a
// thread's binding from Pair-log turns it finds in the agent's transcript;
// without this a slot driven only by peer messages never confirms and stays
// unrelaunchable (pair#618). The entry is the exact envelope pasted, so it
// hashes like the agent's copy of the turn. A failed write costs one
// correlation fact, never the delivery, so it is only logged.
func peerTurnRecorder(path string, getenv func(string) string, debug func(string, string), workers *sync.WaitGroup) func(couchmessage.Message) {
	return func(m couchmessage.Message) {
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
