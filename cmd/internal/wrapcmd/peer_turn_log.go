package wrapcmd

import (
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/pairlog"
)

// peerTurnRecordLimit bounds what one wrapper launch adds to the Pair log:
// at most this many exact submitted turns, additionally bounded by
// peerTurnRecordBytes aggregate payload bytes per launch. Correlation needs one completed round after
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
	single, _ := peerTurnRecorders(path, getenv, debug, workers)
	return single
}

// Both callbacks charge one shared launch budget; a batch is one exact turn.
const peerTurnRecordBytes = 256 * 1024

func peerTurnRecorders(path string, getenv func(string) string, debug func(string, string), workers *sync.WaitGroup) (func(couchmessage.Message), func([]couchmessage.Message)) {
	var mu sync.Mutex
	recorded, payloadBytes := 0, 0
	record := func(id, envelope string) {
		mu.Lock()
		if recorded >= peerTurnRecordLimit || len(envelope) > peerTurnRecordBytes-payloadBytes {
			mu.Unlock()
			return
		}
		recorded++
		payloadBytes += len(envelope)
		mu.Unlock()
		workers.Add(1)
		go func() {
			defer workers.Done()
			if err := pairlog.PersistPeerTurn(getenv, path, []byte(envelope), time.Now(), id); err != nil {
				debug("PEER-turn-log-fail", id+": "+err.Error())
			}
		}()
	}
	return func(m couchmessage.Message) { record(m.ID, couchmessage.Envelope(m)) }, func(messages []couchmessage.Message) {
		if len(messages) > 0 {
			record(messages[0].ID, couchmessage.BatchEnvelope(messages))
		}
	}
}
