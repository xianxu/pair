package couchcore

import (
	"os"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// liveManifest is the operator's real thread-store manifest with its thread
// list emptied -- the subject here is the ENVELOPE's key set, and a populated
// list would only make the test depend on record fixtures that say nothing
// about it. Every other key is verbatim; the two legacy ones are why this
// test exists.
//
// The manifest is decoded with strictThreadStoreJSON, which sets
// DisallowUnknownFields. Deleting a field from threadManifest therefore makes
// every manifest still carrying it undecodable -- and an undecodable manifest
// is not one lost thread, it is the WHOLE STORE: loadManifestLocked fails, so
// no thread can be listed, resumed, or reattached. That is a strictly worse
// failure than the per-record one threadrecord's compat_test guards, which is
// why the manifest gets its own.
//
// Measured before pair#170 M4 deleted the legacy cutover: `legacy_cutover` and
// `legacy_migration_version` were both present in the live manifest. They are
// therefore TOMBSTONES -- decoded and never read. They are, however, WRITTEN:
// see TestManifestTombstonesSurviveAWrite for why that is the right behaviour
// rather than an oversight.
const liveManifest = `{
  "schema_version": 1,
  "generation": 57,
  "threads": [],
  "legacy_cutover": true,
  "legacy_migration_version": 1
}`

func TestPreM4ManifestStillLoads(t *testing.T) {
	store, _ := newTestThreadStore(t)
	if err := os.MkdirAll(store.root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.manifestPath(), []byte(liveManifest), 0o600); err != nil {
		t.Fatal(err)
	}

	snapshot, err := store.Snapshot()
	if err != nil {
		t.Fatalf("a manifest written before M4 no longer loads: %v\n"+
			"every thread in the store would vanish at once, not just one", err)
	}
	if snapshot.Generation != 57 {
		t.Fatalf("generation = %d, want 57", snapshot.Generation)
	}
}

// livePreM4RecordWithOpenStart is a pre-pair#170 record carrying an INTERRUPTED
// start: a `creating` incarnation whose repository identity lives inside the
// `policy` object, because the top-level `repo_identity` field did not exist
// yet.
//
// This is the shape that turns a decode-only tombstone into an outage.
// reconcileInterruptedStarts runs inside New(), promotes this incarnation, and
// advanceSuccessfulStart refuses an empty repository identity -- so if the
// tombstone is merely tolerated rather than READ, couch refuses to start at
// all after the upgrade. The operator's store has 5 records with `policy` and
// none with an open start today, which is luck, not a guarantee.
const livePreM4RecordWithOpenStart = `{
  "schema_version": 2,
  "address": {"repo_scope": "816fc349d3faebf8", "tag": "couch-0102030405060708"},
  "starting_path": "/repo",
  "working_path": "/repo",
  "created_at": "2026-09-02T14:25:51.487991-07:00",
  "revision": 3,
  "claim_generation": 55,
  "incarnations": [
    {
      "pid": 22870,
      "identity": "1788384351.597473",
      "state": "creating",
      "started_at": "2026-09-02T14:25:51.487991-07:00",
      "policy": {
        "policy_version": 1,
        "policy_digest": "b446286124e39db3c6117fe7b3f29aa4c43d266ff96eccfa475a84f1f8ba7844",
        "repo_identity": "/repo/.git",
        "admission_key": "/repo/.git",
        "capacity": {"kind": "bounded", "limit": 1},
        "on_capacity": "reject"
      },
      "start": {
        "nonce": "start-0102030405060708",
        "owner_pid": 4242,
        "owner_identity": "1788384351.000001"
      }
    }
  ],
  "last_active_at": "0001-01-01T00:00:00Z"
}`

func TestPreM4StartCarriesItsRepositoryIdentityForward(t *testing.T) {
	address := ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}
	persisted, err := threadrecord.DecodePersisted([]byte(livePreM4RecordWithOpenStart),
		threadrecord.Address{RepoScope: address.RepoScope, Tag: string(address.Tag)},
		threadRecordValidators)
	if err != nil {
		t.Fatalf("a pre-M4 record with an open start no longer decodes: %v", err)
	}
	record := fromPersistedThreadRecord(persisted)
	if len(record.Incarnations) != 1 {
		t.Fatalf("incarnations = %+v, want the one in the fixture", record.Incarnations)
	}
	// Not merely "it decoded": the VALUE has to survive, or promoting this
	// start refuses and New() fails for the whole store.
	if got := record.Incarnations[0].RepoIdentity; got != "/repo/.git" {
		t.Fatalf("repository identity = %q, want it recovered from the policy tombstone", got)
	}
}

// The manifest keeps its tombstones; a record sheds them. The asymmetry is
// easy to misread as a bug -- an earlier comment here claimed both shed -- so
// it is pinned rather than described.
//
// A record is rebuilt from a domain type that has no deprecated field, so a
// write drops it. The manifest has no such domain type: it is decoded and
// re-marshalled through threadManifest, which carries the raw keys forward.
//
// Keeping them is what we want. `legacy_cutover` records that the one-time
// registry import already ran; clearing it would tell a rolled-back pre-M4
// binary that it never had, and that binary would import the registry a second
// time.
func TestManifestTombstonesSurviveAWrite(t *testing.T) {
	store, _ := newTestThreadStore(t)
	if err := os.MkdirAll(store.root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.manifestPath(), []byte(liveManifest), 0o600); err != nil {
		t.Fatal(err)
	}

	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = store.namespace.Dir(), store.namespace.Dir()
	if _, err := store.CreateThread(record); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(store.manifestPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"legacy_cutover", "legacy_migration_version"} {
		if !strings.Contains(string(raw), key) {
			t.Fatalf("a manifest write dropped %q:\n%s\n"+
				"a rolled-back pre-M4 binary would then re-run the registry cutover", key, raw)
		}
	}
}
