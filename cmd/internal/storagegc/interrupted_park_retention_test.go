package storagegc

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

// An interrupted park leaves historical protocol artifacts behind. Recovery
// must not turn that evidence into either invisible debris or immortal data.
func TestInterruptedParkBundleUsesOrdinarySessionRetention(t *testing.T) {
	root := t.TempDir()
	owner, err := artifactpath.NewStorageOwner(root, "scope", "interrupted-thread")
	if err != nil {
		t.Fatal(err)
	}
	paths, err := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
	if err != nil {
		t.Fatal(err)
	}
	lifecycle, err := paths.Lifecycle("historical-park")
	if err != nil {
		t.Fatal(err)
	}
	request, err := lifecycle.Request(1)
	if err != nil {
		t.Fatal(err)
	}
	trigger, err := lifecycle.Trigger("old-session", 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(lifecycle.Dir(), 0700); err != nil {
		t.Fatal(err)
	}
	// No completion exists: collection inventories protocol names, not a
	// successful park or repaired lifecycle payload.
	files := map[string]string{
		paths.Draft():            "draft",
		paths.LifecycleJournal(): "lifecycle",
		lifecycle.Lock():         "lifecycle-lock",
		request:                  "lifecycle-request",
		trigger:                  "lifecycle-trigger",
	}
	for path := range files {
		if err := os.WriteFile(path, nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	want := map[string]string{
		filepath.Dir(lifecycle.Dir()): "lifecycle",
		lifecycle.Dir():               "lifecycle",
	}
	for path, family := range files {
		want[path] = family
	}
	inventory, err := InventoryRoot(root, nil, nil, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !inventory.Complete || len(inventory.Unknown) != 0 || len(inventory.Groups) != 1 {
		t.Fatalf("historical bundle lost or incomplete: %+v", inventory)
	}
	group := inventory.Groups[0]
	if group.Owner != owner || len(group.Blockers) != 0 || len(group.Members) != len(want) {
		t.Fatalf("historical owner inventory: %+v", group)
	}
	for _, member := range group.Members {
		if family, ok := want[member.Path]; !ok || member.Family != family || member.Retention != artifactpath.SessionRetention {
			t.Fatalf("historical member is not ordinary session data: %+v", member)
		}
		if artifactpath.GCClassifications[member.Family].Disposition != artifactpath.GCCollectable {
			t.Fatalf("historical member became permanently protected: %+v", member)
		}
		delete(want, member.Path)
	}
	if len(want) != 0 {
		t.Fatalf("historical members omitted: %v", want)
	}

	archivedAt := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)
	oldUse := archivedAt.Add(-2 * RetentionPeriod)
	evidence := Evidence{
		Owner: owner, Complete: inventory.Complete, Live: ProcessDead, Protected: true,
		Activity: &ActivityRecord{Version: 1, Owner: owner, Incarnation: "old-owner", InitializedAt: oldUse, LastUse: oldUse},
	}
	if decision := Decide(archivedAt, evidence); decision.State != Protected {
		t.Fatalf("visible historical bundle is not protected: %+v", decision)
	}
	// Archival removes visible protection but starts the existing archive grace,
	// even though activity was already old enough before the interruption.
	evidence.Protected = false
	evidence.ArchiveTimes = []time.Time{archivedAt}
	expiresAt := archivedAt.Add(RetentionPeriod)
	if decision := Decide(expiresAt.Add(-time.Nanosecond), evidence); decision.State != Grace || !decision.EligibleAt.Equal(expiresAt) {
		t.Fatalf("archive grace not preserved: %+v", decision)
	}
	if decision := Decide(expiresAt, evidence); decision.State != Eligible || !decision.EligibleAt.Equal(expiresAt) {
		t.Fatalf("historical bundle not collectible after ordinary retention: %+v", decision)
	}
}
