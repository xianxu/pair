package couchcore

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// RetireAuthority is why the store may retire a thread record: archive it, or
// replace it with another (pair#695 M2). Every retirement funnel
// (archiveThread, ReplaceThreadExpected, replaceSlotCurrent) takes one and
// refuses without it, so no caller retires a record without saying why. It
// is sealed: the three authorities below are the only reasons.
//
//   - TeardownProof: the thread's tree was stopped (or found down) and proven
//     gone, at the revision being retired.
//   - AdoptionAuthority: a slot whose current record is replaced by a live
//     survivor proven its own, every other owner proven dead. A live survivor
//     holds its own lease, so a teardown proof can never exist for it.
//   - UnreadableRecordEscape: the record cannot be decoded, so nothing it
//     names can be stopped; its exact bytes leave, and the caller says the
//     session was not stopped.
//   - FreshLeavesRemnants: a confirmed Fresh recovery whose source could not
//     be proven down. Fresh stops only what the operator confirmed and starts
//     under a new identity beside what is left, so the source is retired as
//     bookkeeping and the remnants are reported, never killed.
type RetireAuthority interface {
	// authorizeRetire checks the authority against what the funnel holds
	// under its lock: the record being retired (nil when undecodable), its
	// exact bytes, and the record replacing it (nil for an archive).
	authorizeRetire(address ThreadAddress, old *ThreadRecord, raw []byte, next *ThreadRecord) error
}

var (
	_ RetireAuthority = TeardownProof{}
	_ RetireAuthority = AdoptionAuthority{}
	_ RetireAuthority = UnreadableRecordEscape{}
	_ RetireAuthority = FreshLeavesRemnants{}
)

func authorizeRetire(authority RetireAuthority, address ThreadAddress, old *ThreadRecord, raw []byte, next *ThreadRecord) error {
	if authority == nil {
		return fmt.Errorf("retiring %s needs a reason: a teardown proof, an adoption, or the unreadable-record escape", address.Tag)
	}
	return authority.authorizeRetire(address, old, raw, next)
}

// A proof covers exactly the record it was taken on: the same thread, the
// same lifecycle address, and the revision teardown left. A record that moved
// since may carry a new launch the proof never saw.
func (p TeardownProof) authorizeRetire(address ThreadAddress, old *ThreadRecord, _ []byte, _ *ThreadRecord) error {
	if old == nil {
		return fmt.Errorf("retiring %s: a teardown proof covers a readable record only", address.Tag)
	}
	if p.provenAt.IsZero() || p.thread != address {
		return fmt.Errorf("retiring %s: the teardown proof is for another thread", address.Tag)
	}
	want, err := threadLifecycleAddress(*old)
	if err != nil {
		return err
	}
	if p.address.Key() != want.Key() {
		return fmt.Errorf("retiring %s: the teardown proof is for %s, not %s", address.Tag, p.address.Key(), want.Key())
	}
	if p.revision != old.Revision {
		return fmt.Errorf("retiring %s: the teardown proof is for revision %d and the record is at %d; tear down again", address.Tag, p.revision, old.Revision)
	}
	return nil
}

// AdoptionAuthority says a slot's current record may be replaced by a live
// survivor: the survivor was proven the slot's own, and every other owner the
// slot names was proven dead. Build it with NewAdoptionAuthority.
type AdoptionAuthority struct {
	survivor ThreadAddress
}

// NewAdoptionAuthority checks the "every other owner is dead" half itself, by
// exact identity; proving the survivor is the caller's (its listing and
// binding evidence differ by path).
func (c *Couch) NewAdoptionAuthority(survivor ThreadAddress, others []ProcessIdentity) (AdoptionAuthority, error) {
	if err := validateThreadAddress(survivor); err != nil {
		return AdoptionAuthority{}, err
	}
	for _, process := range others {
		if c.Proc == nil || observeExactProcess(c.Proc, process) != Dead {
			return AdoptionAuthority{}, fmt.Errorf("adopting %s: another slot owner (pid %d) is not proved stopped", survivor.Tag, process.PID)
		}
	}
	return AdoptionAuthority{survivor: survivor}, nil
}

func (a AdoptionAuthority) authorizeRetire(address ThreadAddress, _ *ThreadRecord, _ []byte, next *ThreadRecord) error {
	if a.survivor == (ThreadAddress{}) {
		return errors.New("adoption authority was not proved")
	}
	if next == nil || next.Address != a.survivor {
		return fmt.Errorf("retiring %s: the adoption authority is for %s, not this replacement", address.Tag, a.survivor.Tag)
	}
	return nil
}

// UnreadableRecordEscape lets an undecodable record leave the working set as
// its exact bytes. Build it with ThreadStore.UnreadableRecordEscape, which
// refuses a record that decodes: a readable record is retired by proof.
type UnreadableRecordEscape struct {
	raw []byte
}

// Raw is the exact bytes the escape covers.
func (e UnreadableRecordEscape) Raw() []byte { return append([]byte(nil), e.raw...) }

// UnreadableRecordEscape reads the record and returns the escape for its
// bytes, refusing when the record decodes.
func (s *ThreadStore) UnreadableRecordEscape(address ThreadAddress) (UnreadableRecordEscape, error) {
	backend, err := s.storeForAddress(address)
	if err != nil {
		return UnreadableRecordEscape{}, err
	}
	var raw []byte
	if err := backend.withReadLockContext(context.Background(), func() error {
		var exists bool
		var e error
		raw, exists, e = backend.readOptionalPayload(backend.recordPath(address))
		if e == nil && !exists {
			e = fmt.Errorf("%w: %+v", ErrThreadNotFound, address)
		}
		return e
	}); err != nil {
		return UnreadableRecordEscape{}, err
	}
	return backend.unreadableEscape(address, raw)
}

func (s *ThreadStore) unreadableEscape(address ThreadAddress, raw []byte) (UnreadableRecordEscape, error) {
	if _, err := s.decodeThreadRaw(address, raw); err == nil {
		return UnreadableRecordEscape{}, fmt.Errorf("thread %s decodes; it is retired by a teardown proof, not the unreadable escape", address.Tag)
	}
	return UnreadableRecordEscape{raw: append([]byte(nil), raw...)}, nil
}

func (e UnreadableRecordEscape) authorizeRetire(address ThreadAddress, old *ThreadRecord, raw []byte, _ *ThreadRecord) error {
	if old != nil {
		return fmt.Errorf("retiring %s: the record decodes now; the unreadable escape no longer applies", address.Tag)
	}
	if e.raw == nil || !bytes.Equal(raw, e.raw) {
		return fmt.Errorf("retiring %s: the record changed since it was found unreadable; inspect and retry", address.Tag)
	}
	return nil
}

// FreshLeavesRemnants is the reason a Fresh recovery retires a source it
// could not prove down: the exact source revision carrying the retained Fresh
// selection, and why the proof failed. Only it may replace a record whose
// park or start claim is still open, which is what Fresh's "leave it
// untouched" means for bookkeeping too.
type FreshLeavesRemnants struct {
	source   ThreadAddress
	revision uint64
	Unproven error
}

func newFreshLeavesRemnants(source ThreadRecord, unproven error) (FreshLeavesRemnants, error) {
	if source.RecoveryIntent == nil || source.RecoveryIntent.Kind() != threadrecord.RecoveryFresh {
		return FreshLeavesRemnants{}, fmt.Errorf("%s carries no retained Fresh selection; its source must be proven down: %w", source.Address.Tag, unproven)
	}
	return FreshLeavesRemnants{source: source.Address, revision: source.Revision, Unproven: unproven}, nil
}

func (f FreshLeavesRemnants) authorizeRetire(address ThreadAddress, old *ThreadRecord, _ []byte, next *ThreadRecord) error {
	if old == nil || next == nil || f.source != address || old.Revision != f.revision {
		return fmt.Errorf("retiring %s: the Fresh selection was made for another record or revision", address.Tag)
	}
	if old.RecoveryIntent == nil || old.RecoveryIntent.Kind() != threadrecord.RecoveryFresh {
		return fmt.Errorf("retiring %s: no retained Fresh selection", address.Tag)
	}
	return nil
}

// retiredBytes is what the archive keeps for a retired record: its exact
// bytes, except that a record retired by proof carries the proof's time as
// its last lifecycle success, so a later failure display can tell an
// unproven attempt from the proven one after it.
func retiredBytes(authority RetireAuthority, old *ThreadRecord, raw []byte) ([]byte, error) {
	proof, ok := authority.(TeardownProof)
	if !ok || old == nil {
		return raw, nil
	}
	stamped := cloneThreadRecord(*old)
	at := proof.provenAt
	stamped.ProvenDownAt = &at
	return encodeThreadRecord(stamped)
}
