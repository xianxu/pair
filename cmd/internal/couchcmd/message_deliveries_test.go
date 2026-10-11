package couchcmd

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

// answeringEndpoint is a wrapper endpoint whose delivery outcome a test picks.
type answeringEndpoint struct {
	serviceEndpointFake
	receipt couchmessage.Receipt
	err     error
}

func (e answeringEndpoint) Deliver(context.Context, couchmessage.Message, func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	return e.receipt, e.err
}

// TestADeliveryToTheOwnerSlotIsOwedByItsSessionSyncFile is pair#437's
// end-to-end row: a real send through the message service lands a ledger row,
// and the owner's session-sync file reports it owed until a write folds it.
func TestADeliveryToTheOwnerSlotIsOwedByItsSessionSyncFile(t *testing.T) {
	r := newServiceRig(t)
	store := t.TempDir()
	r.s.recordDeliveries(store)
	from, to := r.connect(1), r.connect(2)

	file := filepath.Join(t.TempDir(), "tl.md")
	doc := "---\ntype: session-sync\nowner: " + to.Slot + "\n---\n\n## Needs you\n\nold\n"
	os.WriteFile(file, []byte(doc), 0o644)
	// The agent's last write; the shadow's mtime is the owed horizon.
	os.WriteFile(sessionsync.PathsOf(file).Shadow, []byte(doc), 0o644)
	os.Chtimes(sessionsync.PathsOf(file).Shadow, time.Now().Add(-time.Minute), time.Now().Add(-time.Minute))

	send := callerRequest(from, "send")
	send.ID, send.Target, send.Body = "id-owed", to.Slot, "pair#437 done\ndetails"
	if resp := r.s.handle(context.Background(), send); resp.Code != "accepted" && resp.Code != "ok" {
		t.Fatalf("send %+v", resp)
	}
	s := sessionsync.New(store)
	deadline := time.Now().Add(3 * time.Second)
	var st sessionsync.Status
	for {
		var err error
		if st, err = s.Status(file); err != nil {
			t.Fatal(err)
		}
		if len(st.Owed) > 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(st.Owed) != 1 || st.Owed[0].ID != "id-owed" || st.Owed[0].From != from.Slot || st.Owed[0].Kind != sessionsync.KindDelivery {
		t.Fatalf("owed %+v", st.Owed)
	}
	read, err := s.Read(file, "Needs you")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Write(sessionsync.WriteRequest{File: file, Section: "Needs you", Body: "noted", IfHash: read.Hash, Fold: []string{"id-owed"}}); err != nil {
		t.Fatal(err)
	}
	if st, _ = s.Status(file); len(st.Owed) != 0 {
		t.Fatalf("folded delivery still owed: %+v", st.Owed)
	}
}

// Only an explicit cancel or expiry proves nothing was pasted; an error or a
// non-terminal answer (which the broker reads as indeterminate) is recorded,
// marked uncertain.
func TestDeliveryRecordingKeepsEverythingThatMayHaveLanded(t *testing.T) {
	r := newServiceRig(t)
	store := t.TempDir()
	r.s.recordDeliveries(store)
	b := r.connect(1)
	for _, c := range []struct {
		id      string
		receipt couchmessage.Receipt
		err     error
		want    bool
		unsure  bool
	}{
		{"submitted", couchmessage.Receipt{Status: couchmessage.Submitted}, nil, true, false},
		{"indeterminate", couchmessage.Receipt{Status: couchmessage.Indeterminate}, nil, true, true},
		{"error", couchmessage.Receipt{}, errors.New("endpoint gone"), true, true},
		// The broker reads any error as indeterminate, whatever the receipt says.
		{"submitted-but-error", couchmessage.Receipt{Status: couchmessage.Submitted}, context.DeadlineExceeded, true, true},
		{"nonterminal", couchmessage.Receipt{Status: couchmessage.Delivering}, nil, true, true},
		{"cancelled", couchmessage.Receipt{Status: couchmessage.Cancelled}, nil, false, false},
		{"expired", couchmessage.Receipt{Status: couchmessage.Expired}, nil, false, false},
	} {
		e := messageEndpoint{service: r.s, binding: b, endpoint: answeringEndpoint{receipt: c.receipt, err: c.err}}
		_, _ = e.Deliver(context.Background(), couchmessage.Message{ID: c.id, From: b, To: b, Body: "x"}, nil)
		rows, _ := (&sessionsync.Ledger{Dir: sessionsync.LedgerDir(store)}).Read(b.Slot)
		found := false
		for _, row := range rows {
			if row.ID == c.id {
				found = true
				if row.Uncertain != c.unsure {
					t.Fatalf("%s: uncertain=%v", c.id, row.Uncertain)
				}
			}
		}
		if found != c.want {
			t.Fatalf("%s: recorded=%v want %v", c.id, found, c.want)
		}
	}
}

// A failed append keeps the delivery's outcome and records a gap for that
// slot only; other slots' files keep working (BR-2), and a restart does not
// clear it (BR-3).
func TestALedgerAppendFailureRecordsAGapForThatSlotOnly(t *testing.T) {
	r := newServiceRig(t)
	store := t.TempDir()
	r.s.recordDeliveries(store)
	b, other := r.connect(1), r.connect(2)
	ledgerDir := sessionsync.LedgerDir(store)
	ledger := &sessionsync.Ledger{Dir: ledgerDir}
	// The slot's file becomes a directory: appends to it fail.
	os.MkdirAll(filepath.Join(ledgerDir, url.QueryEscape(b.Slot)+".jsonl"), 0o700)
	e := messageEndpoint{service: r.s, binding: b, endpoint: answeringEndpoint{receipt: couchmessage.Receipt{Status: couchmessage.Submitted}}}
	if receipt, err := e.Deliver(context.Background(), couchmessage.Message{ID: "x", From: other, To: b}, nil); err != nil || receipt.Status != couchmessage.Submitted {
		t.Fatalf("the delivery outcome must survive: %+v %v", receipt, err)
	}
	if g, err := ledger.Gap(b.Slot); err != nil || g == nil {
		t.Fatalf("gap %+v %v", g, err)
	}
	r.s.recordDeliveries(store) // a Couch restart
	if g, _ := ledger.Gap(b.Slot); g == nil {
		t.Fatal("a restart recovers no row, so it must not clear the gap")
	}
	if g, _ := ledger.Gap(other.Slot); g != nil || ledger.Recording() != nil {
		t.Fatalf("another slot is unaffected: %+v", g)
	}
}
