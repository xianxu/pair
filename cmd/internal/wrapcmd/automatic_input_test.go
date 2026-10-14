package wrapcmd

import (
	"bytes"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"strings"
	"testing"
	"time"
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

func automaticWaiting(t *testing.T, submitted bool) automaticInputTransaction {
	t.Helper()
	tx, _ := advanceAutomaticInput(automaticInputTransaction{}, automaticInputEvent{kind: automaticBegin, owner: automaticInputPeer})
	tx, _ = advanceAutomaticInput(tx, automaticInputEvent{kind: automaticFinish, owner: automaticInputPeer, epoch: 7, submitted: submitted})
	return tx
}
func TestAutomaticInputTripwireSequence(t *testing.T) {
	start := time.Unix(100, 0)
	for _, tc := range []struct {
		name      string
		submitted bool
		composer  PeerComposerState
		released  bool
	}{
		{"safe stale fence", true, PeerComposerEmpty, true},
		{"unknown is never permission", true, PeerComposerUnknown, false},
		{"uncertain is never permission", false, PeerComposerEmpty, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := automaticWaiting(t, tc.submitted)
			e := automaticInputEvent{kind: automaticObserve, owner: automaticInputPeer, now: start, epoch: 7, composer: tc.composer, complete: true, eligible: true, recoveryAfter: 30 * time.Second}
			tx, d := advanceAutomaticInput(tx, e)
			if d.available || d.report != nil {
				t.Fatal("early availability/report")
			}
			e.now = start.Add(30*time.Second - time.Nanosecond)
			tx, d = advanceAutomaticInput(tx, e)
			if d.report != nil {
				t.Fatal("early tripwire")
			}
			e.now = start.Add(30 * time.Second)
			tx, d = advanceAutomaticInput(tx, e)
			if d.report == nil || d.report.Released != tc.released || d.available != tc.released || d.report.Sequence != 1 {
				t.Fatalf("tripwire=%+v available=%v", d.report, d.available)
			}
			e.now = e.now.Add(time.Hour)
			tx, d = advanceAutomaticInput(tx, e)
			if d.report != nil {
				t.Fatal("tripwire repeated")
			}
			// Any still-blocked episode may recover normally from fresh empty evidence.
			e.epoch = 8
			e.composer = PeerComposerEmpty
			tx, d = advanceAutomaticInput(tx, e)
			if !d.available {
				t.Fatal("normal recovery refused")
			}
			tx, _ = advanceAutomaticInput(tx, automaticInputEvent{kind: automaticBegin, owner: automaticInputPeer})
			tx, _ = advanceAutomaticInput(tx, automaticInputEvent{kind: automaticFinish, owner: automaticInputPeer, epoch: 8, submitted: true})
			e.epoch = 8
			tx, _ = advanceAutomaticInput(tx, e)
			e.now = e.now.Add(30 * time.Second)
			_, d = advanceAutomaticInput(tx, e)
			if d.report == nil || d.report.Sequence != 2 {
				t.Fatalf("sequence not monotonic: %+v", d.report)
			}
		})
	}
}
func TestAutomaticInputTripwireDoesNotBorrowHumanTime(t *testing.T) {
	start := time.Unix(100, 0)
	tx := automaticWaiting(t, true)
	e := automaticInputEvent{kind: automaticObserve, owner: automaticInputPeer, now: start, epoch: 7, composer: PeerComposerEmpty, complete: true, eligible: true, recoveryAfter: 30 * time.Second}
	tx, _ = advanceAutomaticInput(tx, e)
	e.now = start.Add(29 * time.Second)
	e.eligible = false
	tx, d := advanceAutomaticInput(tx, e)
	if d.report != nil {
		t.Fatal("reported over human input")
	}
	e.now = start.Add(time.Hour)
	e.eligible = true
	tx, d = advanceAutomaticInput(tx, e)
	if d.available || d.report != nil {
		t.Fatal("human time counted as blocked time")
	}
	e.now = e.now.Add(29 * time.Second)
	tx, d = advanceAutomaticInput(tx, e)
	if d.report != nil {
		t.Fatal("renewed interval fired early")
	}
	e.now = e.now.Add(time.Second)
	_, d = advanceAutomaticInput(tx, e)
	if d.report == nil {
		t.Fatal("missing bounded report")
	}
}
func TestAutomaticInputDisabledRecoveryAndUnsafeObservations(t *testing.T) {
	for _, composer := range []PeerComposerState{PeerComposerEmpty, PeerComposerUnknown, PeerComposerOccupied} {
		for _, complete := range []bool{false, true} {
			tx := automaticWaiting(t, true)
			e := automaticInputEvent{kind: automaticObserve, owner: automaticInputPeer, now: time.Unix(1, 0), epoch: 7, composer: composer, complete: complete, eligible: true, recoveryAfter: -1}
			tx, _ = advanceAutomaticInput(tx, e)
			e.now = e.now.Add(24 * time.Hour)
			_, d := advanceAutomaticInput(tx, e)
			if d.available || d.report != nil {
				t.Fatalf("disabled recovery granted permission: composer=%v complete=%v", composer, complete)
			}
		}
	}
	// A new epoch inside an unfinished frame is not fresh empty evidence.
	tx := automaticWaiting(t, true)
	_, d := advanceAutomaticInput(tx, automaticInputEvent{kind: automaticObserve, owner: automaticInputPeer, epoch: 8, composer: PeerComposerEmpty, complete: false})
	if d.available {
		t.Fatal("incomplete frame released lease")
	}
}

func TestAutomaticInputTripwireDispatchPreservesAdmission(t *testing.T) {
	for _, guard := range []string{"none", "unknown", "turn", "picker", "image", "held-input", "buffered-input", "forwarded-input", "partial-frame", "uncertain"} {
		t.Run(guard, func(t *testing.T) {
			f, d := peerIntegrationFixture(t)
			p := f.proxy
			now := time.Unix(100, 0)
			d.now = func() time.Time { return now }
			d.state.Phase = couchmessage.PeerDeliverySubmitted
			p.beginAutomaticInput(automaticInputPeer)
			p.finishAutomaticInput(automaticInputPeer, false)
			if guard == "uncertain" {
				p.automaticInput.submitted = false
			}
			m := peerTestMessage(now)
			m.Deadline = now.Add(time.Minute)
			if err := d.reserve(m.ID, d.sequence); err != nil {
				t.Fatal(err)
			}
			if err := d.enqueue(m); err != nil {
				t.Fatal(err)
			}
			switch guard {
			case "unknown":
				f.output("\x1b[2J")
			case "turn":
				p.turnActive.Store(true)
			case "picker":
				p.pickerActive.Store(true)
			case "image":
				d.image = true
			case "held-input":
				d.holding = true
			case "buffered-input":
				d.inputBuffered = true
			case "forwarded-input":
				d.replies.inFlight = 1
			case "partial-frame":
				f.output("\x1b[?2026h")
			}
			var reports []couchmessage.AutomaticInputStall
			p.automaticReport = func(r couchmessage.AutomaticInputStall, _ terminalSnapshot) { reports = append(reports, r) }
			var out bytes.Buffer
			p.dispatchPeer(&out)
			if out.Len() != 0 {
				t.Fatal("early paste")
			}
			now = now.Add(30 * time.Second)
			p.dispatchPeer(&out)
			if guard == "none" {
				if out.Len() == 0 || len(reports) != 1 || !reports[0].Released {
					t.Fatalf("missing safe release: out=%q reports=%+v", out.String(), reports)
				}
				return
			}
			if out.Len() != 0 {
				t.Fatalf("unsafe paste with %s", guard)
			}
			if guard == "unknown" || guard == "uncertain" {
				if len(reports) != 1 || reports[0].Released {
					t.Fatalf("missing blocked alert: %+v", reports)
				}
			} else if len(reports) != 0 {
				t.Fatalf("normal hold became tripwire: %+v", reports)
			}
		})
	}
}
