package couchcore

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// detachedLayoutRecord is a thread with a live client-less session -- the
// session-holding shape the guard cares about -- carrying `layout` verbatim as
// a persisted record would.
func detachedLayoutRecord(t *testing.T, layout Layout) (ThreadRecord, map[ThreadAddress]ThreadEvidence) {
	t.Helper()
	active := time.Unix(1000, 0).UTC()
	record := actionableTestThread("couch-0000000000000003", active)
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	record.Layout = layout
	evidence := map[ThreadAddress]ThreadEvidence{record.Address: {
		Detached: []DetachedSessionObservation{{
			Address: record.Address, SessionName: "pair-three", Agent: "claude", NativeID: "native-3",
		}},
		ParkedStatus: ProofResolved, DetachedStatus: ProofResolved,
	}}
	return record, evidence
}

func projectOne(t *testing.T, record ThreadRecord, evidence map[ThreadAddress]ThreadEvidence) ActionableThreadSummary {
	t.Helper()
	rows := ProjectActionableThreads(ThreadProjectionInput{
		Records: []ThreadRecord{record}, Evidence: evidence,
	})
	if len(rows) != 1 || rows[0].State != ThreadDetached {
		t.Fatalf("rows = %+v; want exactly one detached row", rows)
	}
	return rows[0]
}

// The Critical the plan gate caught (PQ-1). Every record written before #198 has
// no layout field. If the projection carried the raw Layout("") through, it
// would not equal Layout2 and EVERY existing thread would block a default
// `couch` startup -- turning this feature into an outage on first run.
func TestProjectionNormalizesAbsentLayoutAndDoesNotBlockDefaultStartup(t *testing.T) {
	record, evidence := detachedLayoutRecord(t, "")
	row := projectOne(t, record, evidence)
	if row.Layout != Layout2 {
		t.Fatalf("row layout = %q; want Layout2 normalized from an absent field", row.Layout)
	}
	if got := ResolveLayoutConflicts(Layout2, []ActionableThreadSummary{row}); len(got) != 0 {
		t.Fatalf("a pre-#198 record blocked a default startup: %+v", got)
	}
}

func TestProjectionCarriesARecordedLayout(t *testing.T) {
	record, evidence := detachedLayoutRecord(t, Layout3)
	if row := projectOne(t, record, evidence); row.Layout != Layout3 {
		t.Fatalf("row layout = %q; want Layout3", row.Layout)
	}
}

// ARCH-SECURE: an unreadable value must not become a plausible default. It
// surfaces as LayoutUnknown, which refuses visibly against either request.
func TestProjectionMarksUnreadableLayoutUnknown(t *testing.T) {
	record, evidence := detachedLayoutRecord(t, "layout9")
	row := projectOne(t, record, evidence)
	if row.Layout != LayoutUnknown {
		t.Fatalf("row layout = %q; want LayoutUnknown", row.Layout)
	}
	for _, requested := range []Layout{Layout2, Layout3} {
		if got := ResolveLayoutConflicts(requested, []ActionableThreadSummary{row}); len(got) != 1 {
			t.Fatalf("requested %v: an unreadable layout did not refuse: %+v", requested, got)
		}
	}
}

// The witness has to survive the persistence hop, or the guard reads a field
// that park/resume silently dropped.
func TestThreadRecordLayoutRoundTripsThroughPersistence(t *testing.T) {
	active := time.Unix(1000, 0).UTC()
	record := actionableTestThread("couch-0000000000000005", active)
	record.Layout = Layout3
	if got := fromPersistedThreadRecord(toPersistedThreadRecord(record)); got.Layout != Layout3 {
		t.Fatalf("layout after round trip = %q; want Layout3", got.Layout)
	}
}

// A record predating #198 carries no layout key at all; it must survive the hop
// as empty rather than acquiring a value nothing wrote.
func TestAbsentLayoutSurvivesPersistenceAsEmpty(t *testing.T) {
	active := time.Unix(1000, 0).UTC()
	record := actionableTestThread("couch-0000000000000006", active)
	if got := fromPersistedThreadRecord(toPersistedThreadRecord(record)); got.Layout != "" {
		t.Fatalf("layout after round trip = %q; want empty", got.Layout)
	}
}

// Persistence was tested at the Go-struct level only, which cannot see the
// on-disk key: renaming the `layout` json tag in threadrecord would have kept
// the round-trip green while silently dropping every witness on disk. This
// decodes real bytes.
func TestLayoutWitnessPersistsUnderItsOnDiskKey(t *testing.T) {
	active := time.Unix(1000, 0).UTC()
	record := actionableTestThread("couch-0000000000000007", active)
	record.Layout = Layout3

	raw, err := json.Marshal(toPersistedThreadRecord(record))
	if err != nil {
		t.Fatal(err)
	}
	var onDisk map[string]any
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		t.Fatal(err)
	}
	if got := onDisk["layout"]; got != "layout3" {
		t.Fatalf("on-disk key \"layout\" = %v; want \"layout3\" (a renamed tag drops every witness)", got)
	}

	// And a record written before #198 carries no such key at all, rather than
	// an empty one -- strictjson is unforgiving, so the absence is the contract.
	old := actionableTestThread("couch-0000000000000008", active)
	rawOld, err := json.Marshal(toPersistedThreadRecord(old))
	if err != nil {
		t.Fatal(err)
	}
	var onDiskOld map[string]any
	if err := json.Unmarshal(rawOld, &onDiskOld); err != nil {
		t.Fatal(err)
	}
	if _, present := onDiskOld["layout"]; present {
		t.Fatalf("a record with no layout emitted the key anyway: %s", rawOld)
	}
}

// A real pre-#198 record, decoded through the production path, must load
// cleanly and read as layout2.
func TestPre198RecordDecodesThroughTheProductionPath(t *testing.T) {
	active := time.Unix(1000, 0).UTC()
	record := actionableTestThread("couch-0000000000000009", active)
	raw, err := json.Marshal(toPersistedThreadRecord(record))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := threadrecord.DecodePersisted(raw, toPersistedThreadAddress(record.Address), threadRecordValidators)
	if err != nil {
		t.Fatalf("a pre-#198 record no longer decodes: %v", err)
	}
	if got := NormalizeLayout(decoded.Layout); got != Layout2 {
		t.Fatalf("pre-#198 record normalized to %q; want Layout2", got)
	}
}
