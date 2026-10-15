package sessionsync

import (
	"errors"
	"strings"
	"testing"
)

func TestRegisteredGapAcceptancePublishesPreparedPrunedState(t *testing.T) {
	r := newRig(t)
	if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
		t.Fatal(err)
	}
	d, err := withRegistry(r.sync, func(tx *registryTxn) (DocumentRegistration, error) {
		canonical, err := canonicalDocument(r.p.File)
		if err != nil {
			return DocumentRegistration{}, err
		}
		return *tx.find(canonical), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	st := handoffSizedState(t, r.now, registeredStateMaxBytes-16, d.Paths)
	p := registeredPaths(t, r.sync, r.p.File)
	if err := saveState(p, st); err != nil {
		t.Fatal(err)
	}
	if err := validateRegisteredMetadata(r.read(p.Shadow), st, d.Paths); err != nil {
		t.Fatalf("fixture exceeds initial bounds: %v", err)
	}
	r.sync.Ledger.Now = r.sync.Now
	if err := r.sync.Ledger.MarkGap("ops:0", errors.New("injected lost delivery")); err != nil {
		t.Fatal(err)
	}
	gap, err := r.sync.Ledger.Gap("ops:0")
	if err != nil || gap == nil {
		t.Fatalf("gap fixture: %+v %v", gap, err)
	}
	before := r.read(r.p.File)
	_, err = r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "prepared write\n", IfHash: r.hash("Trains"), Accepts: Accepts{Gap: true}})
	if err != nil {
		t.Fatalf("valid post-pruning write failed (documentChanged=%t): %v", r.read(r.p.File) != before, err)
	}
	document, shadow := r.read(r.p.File), r.read(p.Shadow)
	if document != shadow || !strings.Contains(document, "prepared write\n") {
		t.Fatal("document and prepared shadow differ")
	}
	persisted, err := LoadState(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(persisted.Folded) != 0 || persisted.GapAccepted == nil || !persisted.GapAccepted.Equal(gap.Last) || persisted.LastWrite == nil || !persisted.LastWrite.Equal(r.now) {
		t.Fatalf("prepared state was not published: folds=%d gap=%v lastWrite=%v", len(persisted.Folded), persisted.GapAccepted, persisted.LastWrite)
	}
	if err := validateRegisteredMetadata(shadow, persisted, d.Paths); err != nil {
		t.Fatalf("published metadata unreadable: %v", err)
	}
	if _, _, err := r.sync.DiffText(r.p.File); err != nil {
		t.Fatalf("subsequent metadata read: %v", err)
	}
}
