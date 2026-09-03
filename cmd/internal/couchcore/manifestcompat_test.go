package couchcore

import (
	"os"
	"testing"
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
// therefore TOMBSTONES -- decoded, never read, never written -- so the manifest
// sheds them on its next write with no migration pass.
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
