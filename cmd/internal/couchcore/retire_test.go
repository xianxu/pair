package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// retireAuthorityForTest is the reason a store-level test retires address
// with: a proof covering a readable record, the escape for an unreadable one,
// and none for a missing one, so the store's own refusal is what the test
// observes.
func retireAuthorityForTest(t testing.TB, s *ThreadStore, address ThreadAddress) RetireAuthority {
	t.Helper()
	record, err := s.GetThread(address)
	if err == nil {
		return proofForTest(t, record)
	}
	if escape, err := s.UnreadableRecordEscape(address); err == nil {
		return escape
	}
	return nil
}

func proofForTest(t testing.TB, record ThreadRecord) TeardownProof {
	t.Helper()
	proof, err := TestOnlyTeardownProof(record, time.Now())
	if err != nil {
		t.Fatalf("proof for %s: %v", record.Address.Tag, err)
	}
	return proof
}

// slotAuthorityForTest is the reason for replacing an observed slot current.
func slotAuthorityForTest(t testing.TB, observed slotCurrentObservation) RetireAuthority {
	t.Helper()
	switch {
	case observed.Record != nil:
		return proofForTest(t, *observed.Record)
	case observed.Exists:
		return UnreadableRecordEscape{raw: append([]byte(nil), observed.Raw...)}
	}
	return nil
}

// teardownReady gives a hand-built Couch the seams teardown reads: a
// namespace for the lease, an empty process table and a clock.
func teardownReady(t testing.TB, c *Couch) *Couch {
	t.Helper()
	ns, err := ResolveCouchNamespace(t.TempDir(), "/unused")
	if err != nil {
		t.Fatal(err)
	}
	c.Namespace = ns
	c.ProcessTable = func() ([]procutil.Process, error) { return nil, nil }
	if c.Clock == nil {
		c.Clock = FixedClock{T: time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)}
	}
	return c
}

// Every store retirement entry point refuses a retirement that gives no
// reason, and the record stays where it was (pair#695).
func TestEveryRetirementTakesAnAuthority(t *testing.T) {
	t.Parallel()
	store, _ := newTestThreadStore(t)
	record := archivableThread(t, store, "couch-0000000000000001")
	raw, err := os.ReadFile(store.recordPath(record.Address))
	if err != nil {
		t.Fatal(err)
	}
	next := archivableThread(t, store, "couch-0000000000000002")
	if err := store.ArchiveThread(next.Address, retireAuthorityForTest(t, store, next.Address)); err != nil {
		t.Fatal(err)
	}
	next.Revision = 1
	for name, retire := range map[string]func() error{
		"archive":          func() error { return store.ArchiveThread(record.Address, nil) },
		"archive expected": func() error { return store.ArchiveThreadExpected(record.Address, record.Revision, nil) },
		"archive bytes":    func() error { return store.ArchiveThreadExpectedBytes(record.Address, raw, nil) },
		"replace":          func() error { return store.ReplaceThreadExpected(record.Address, record.Revision, next, nil) },
	} {
		if err := retire(); err == nil || !strings.Contains(err.Error(), "needs a reason") {
			t.Errorf("%s without an authority: %v", name, err)
		}
	}
	if _, err := store.GetThread(record.Address); err != nil {
		t.Fatalf("a refused retirement moved the record: %v", err)
	}

	slot := testLocalThreadStore(t)
	current, err := slot.CreateThread(recordAtCheckout(t, slot.slot.WorktreeRoot, slot.slot.WorktreeRoot, "couch-0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	observed, err := slot.observeSlotCurrent()
	if err != nil {
		t.Fatal(err)
	}
	fresh := current
	fresh.Address.Tag, fresh.Incarnations, fresh.Revision = "couch-1111111111111111", nil, 1
	if err := slot.replaceSlotCurrent(observed, fresh, nil); err == nil || !strings.Contains(err.Error(), "needs a reason") {
		t.Fatalf("slot replace without an authority: %v", err)
	}
	// An adoption authorizes publishing its survivor only.
	adoption := AdoptionAuthority{survivor: ThreadAddress{RepoScope: current.Address.RepoScope, Tag: "couch-2222222222222222"}}
	if err := slot.replaceSlotCurrent(observed, fresh, adoption); err == nil {
		t.Fatal("an adoption authority published a record that is not its survivor")
	}
}

// The escape is for undecodable bytes only, and for exactly those bytes.
func TestUnreadableEscapeRefusesADecodableRecord(t *testing.T) {
	t.Parallel()
	store, _ := newTestThreadStore(t)
	healthy := archivableThread(t, store, "couch-0000000000000001")
	if _, err := store.UnreadableRecordEscape(healthy.Address); err == nil {
		t.Fatal("the escape was granted for a record that decodes")
	}
	if err := store.ArchiveThread(healthy.Address, UnreadableRecordEscape{raw: []byte("x")}); err == nil {
		t.Fatal("a readable record left through the unreadable escape")
	}
	corrupt := writeCorruptRecord(t, store, healthy, "couch-0000000000000002")
	escape, err := store.UnreadableRecordEscape(corrupt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.recordPath(corrupt), []byte(`{"other":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.ArchiveThread(corrupt, escape); err == nil {
		t.Fatal("the escape retired bytes it was not granted for")
	}
	if escape, err = store.UnreadableRecordEscape(corrupt); err != nil {
		t.Fatal(err)
	}
	if err := store.ArchiveThread(corrupt, escape); err != nil {
		t.Fatalf("the escape for the current bytes: %v", err)
	}
}

// A proof covers the revision teardown left; a record that moved since may
// carry a launch the proof never saw.
func TestArchiveRefusesProofForMovedRevision(t *testing.T) {
	t.Parallel()
	store, _ := newTestThreadStore(t)
	record := archivableThread(t, store, "couch-0000000000000001")
	proof := proofForTest(t, record)
	label := "moved"
	if _, err := store.ApplyThreadMetadata(record.Address, record.Revision, ThreadMetadataPatch{Name: &label}); err != nil {
		t.Fatal(err)
	}
	if err := store.ArchiveThread(record.Address, proof); err == nil || !strings.Contains(err.Error(), "tear down again") {
		t.Fatalf("a proof for an older revision retired the record: %v", err)
	}
	other := archivableThread(t, store, "couch-0000000000000002")
	if err := store.ArchiveThread(other.Address, proofForTest(t, record)); err == nil {
		t.Fatal("a proof for another thread retired this one")
	}
}

// openParkThread is a record whose park transaction is still open and whose
// park owner and helper are both gone: what a teardown that stopped them
// leaves behind.
func openParkThread(t *testing.T, store *ThreadStore) ThreadRecord {
	t.Helper()
	record := actionableTestThread("couch-e1a31510b7033d08", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	record.Incarnations = []ThreadIncarnation{{PID: 64734, Identity: "1789535173.46673", State: IncarnationLive}}
	record.Park = wedgedParkFixture(record.Address)
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	label := "parked"
	if created, err = store.ApplyThreadMetadata(created.Address, created.Revision, ThreadMetadataPatch{Name: &label}); err != nil {
		t.Fatal(err)
	}
	return created
}

// The store never strands an open park: archive refuses it, whatever the
// authority, and ProveDown (which stops nothing) does not fold it.
func TestArchiveRefusesOpenPark(t *testing.T) {
	t.Parallel()
	store, _ := newTestThreadStore(t)
	record := openParkThread(t, store)
	couch := teardownReady(t, &Couch{Threads: store, Artifacts: NewFakeThreadArtifactCollisionChecker(), Proc: NewFakeProcOps(), Path: NewFakePathOps(nil)})
	proof, err := couch.ProveDown(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	if proof.Revision() != record.Revision {
		t.Fatalf("ProveDown changed the record: revision %d -> %d", record.Revision, proof.Revision())
	}
	if err := store.ArchiveThread(record.Address, proof); err == nil || !strings.Contains(err.Error(), "park in flight") {
		t.Fatalf("archive of an open park: %v", err)
	}
}

// Teardown folds the park its stopped owners left, so the proof's revision
// is the folded record and archiving with it succeeds.
func TestTeardownFoldsCompletedPark(t *testing.T) {
	t.Parallel()
	store, _ := newTestThreadStore(t)
	record := openParkThread(t, store)
	couch := teardownReady(t, &Couch{Threads: store, Artifacts: NewFakeThreadArtifactCollisionChecker(), Proc: NewFakeProcOps(), Path: NewFakePathOps(nil)})
	proof, err := couch.Teardown(context.Background(), record, "archive")
	if err != nil {
		t.Fatal(err)
	}
	folded, err := store.GetThread(record.Address)
	if err != nil {
		t.Fatal(err)
	}
	if folded.Park != nil || proof.Revision() != folded.Revision || folded.Revision == record.Revision {
		t.Fatalf("park after teardown %+v, proof revision %d, record %d -> %d", folded.Park, proof.Revision(), record.Revision, folded.Revision)
	}
	if err := store.ArchiveThreadExpected(record.Address, proof.Revision(), proof); err != nil {
		t.Fatalf("archive with the folded proof: %v", err)
	}
}

// A Fresh source that cannot be proven down is retired as bookkeeping only
// by its retained Fresh selection; any other record must be proven down.
func TestFreshLeavesRemnantsNeedsTheRetainedSelection(t *testing.T) {
	t.Parallel()
	unproven := errors.New("session still present")
	if _, err := newFreshLeavesRemnants(ThreadRecord{Address: ThreadAddress{RepoScope: "s", Tag: "t"}}, unproven); err == nil || !errors.Is(err, unproven) {
		t.Fatalf("a record without a Fresh selection got the remnants authority: %v", err)
	}
}

// An unproven teardown leaves its failure on the slot; the proven retry that
// archives the record stamps its proof time there, and the emptied slot's
// --show no longer shows the earlier failure (pair#695).
func TestProvenTeardownHidesEarlierUnprovenFailure(t *testing.T) {
	env := newTestEnv(t, "/repo")
	failures := NewFakeLaunchFailureStore()
	env.Couch.LaunchFailures = failures
	local := testLocalThreadStore(t)
	slot := *local.slot
	record, err := local.CreateThread(recordAtCheckout(t, slot.WorktreeRoot, slot.WorktreeRoot, "couch-0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	failedAt := time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)
	if err := failures.Save(t.Context(), LaunchFailure{StartingPath: slot.WorktreeRoot, Nonce: "unproven", FailedAt: failedAt}); err != nil {
		t.Fatal(err)
	}
	if got, _ := env.Couch.launchFailuresForShow(t.Context(), "scope", slot.WorktreeRoot, nil, slot); len(got) != 1 {
		t.Fatalf("before the proof: %+v", got)
	}
	proof := proofForTest(t, record)
	proof.provenAt = failedAt.Add(time.Minute)
	if err := local.ArchiveThread(record.Address, proof); err != nil {
		t.Fatal(err)
	}
	archived, err := local.ArchivedThreads()
	if err != nil || len(archived) != 1 || archived[0].ProvenDownAt == nil || !archived[0].ProvenDownAt.Equal(proof.provenAt) {
		t.Fatalf("archived %+v err %v, want the proof time stamped", archived, err)
	}
	if got, detail := env.Couch.launchFailuresForShow(t.Context(), "scope", slot.WorktreeRoot, nil, slot); len(got) != 0 || detail != "" {
		t.Fatalf("after the proven teardown: %+v %s", got, detail)
	}
}

// assertRetiredBytes: a record retired by proof is archived as its exact
// members plus proven_down_at and nothing else (pair#695); every other
// member keeps its bytes.
func assertRetiredBytes(t testing.TB, archived, raw []byte) {
	t.Helper()
	var got, want map[string]json.RawMessage
	if err := json.Unmarshal(archived, &got); err != nil {
		t.Fatalf("archived record: %v", err)
	}
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatalf("retired record: %v", err)
	}
	if _, stamped := got["proven_down_at"]; !stamped {
		t.Fatal("a record retired by proof was archived without its proof time")
	}
	delete(got, "proven_down_at")
	if len(got) != len(want) {
		t.Fatalf("archive members %d, record members %d", len(got), len(want))
	}
	for key, value := range want {
		var a, b bytes.Buffer
		if json.Compact(&a, got[key]) != nil || json.Compact(&b, value) != nil || !bytes.Equal(a.Bytes(), b.Bytes()) {
			t.Fatalf("archive changed member %q: %s -> %s", key, value, got[key])
		}
	}
}

// A minted proof is what lets the store retire a record, so the test-only
// minting path never appears in production source anywhere in the module.
// couchcoretest may call it because only tests may import couchcoretest,
// which this checks too.
func TestOnlyTeardownProofHasNoProductionCaller(t *testing.T) {
	root := filepath.Join("..", "..", "..") // the module root
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fixture := strings.Contains(filepath.ToSlash(path), "/couchcoretest/")
		if bytes.Contains(raw, []byte("TestOnlyTeardownProof(")) && !fixture && filepath.Base(path) != "teardown.go" {
			t.Errorf("%s mints a teardown proof outside a test", path)
		}
		if bytes.Contains(raw, []byte(`"github.com/xianxu/pair/cmd/internal/couchcoretest"`)) && !fixture {
			t.Errorf("%s imports the couchcoretest fixtures from production code", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
