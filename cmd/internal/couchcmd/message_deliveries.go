package couchcmd

import (
	"os"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

// deliveryRecorder appends each landed delivery to the session-sync ledger
// (pair#437), so `couch --session-sync status` can list what a file's owner
// still owes it.
type deliveryRecorder struct {
	ledger *sessionsync.Ledger
	marker sessionsync.Recording
	now    func() time.Time
}

// recordDeliveries starts the ledger for this Couch: the marker tells
// readers deliveries are being recorded from now on.
func (s *messageService) recordDeliveries(storeDir string) {
	r := &deliveryRecorder{ledger: &sessionsync.Ledger{Dir: sessionsync.LedgerDir(storeDir)},
		marker: sessionsync.Recording{PID: os.Getpid(), Started: time.Now()}, now: time.Now}
	if r.ledger.MarkRecording(r.marker) != nil {
		// No marker: status refuses not-recording, which is the truth.
		return
	}
	s.deliveries.Store(r)
}

// record keeps every delivery that may have reached the agent: submitted,
// indeterminate, an endpoint error or a non-terminal answer (the broker
// reads the last two as indeterminate). Only an explicit cancel or expiry
// proves nothing was pasted. A spurious owed row costs one fold; a missing
// one is the liveness bug the ledger exists for.
//
// The owner's own session-sync ping is not recorded (pair#441): it asks for
// a fold, so recording it would seed the next ping. It is recognized by
// sender identity, as the breaker recognizes the owner's submit (#433), so a
// peer imitating the prefix stays owed.
func (r *deliveryRecorder) record(m couchmessage.Message, receipt couchmessage.Receipt, err error) {
	if err == nil && (receipt.Status == couchmessage.Cancelled || receipt.Status == couchmessage.Expired) {
		return
	}
	if m.From == m.To && strings.HasPrefix(m.Body, sessionsync.PingPrefix) {
		return
	}
	e := sessionsync.Entry{ID: m.ID, From: m.From.Slot, At: r.now(), Head: m.Body,
		AddressedAt: m.AcceptedAt,
		Origin:      m.Origin, RemoteEdited: m.RemoteEdited, RemoteSource: m.RemoteSource, DocumentID: m.DocumentID,
		Uncertain: err != nil || receipt.Status != couchmessage.Submitted}
	if aerr := r.ledger.Append(m.To.Slot, e); aerr != nil {
		// Delivery succeeded; the record did not. Record the gap for that
		// slot's readers instead of failing the message.
		_ = r.ledger.MarkGap(m.To.Slot, aerr)
	}
}
