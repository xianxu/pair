package gccmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchcoretest"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/gcruntime"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// Public command composition: real metadata, migration, archive transactions,
// inventory and collection. Global process/open-file inventories are isolated
// inspection fixtures; the live owner's birth identity is checked by the real OS.
func TestPublicApplyPreservesProtectedOwnersAndCollectsIndependentBuckets(t *testing.T) {
	ctx := context.Background()
	c, err := storagegc.NewCoordinator(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	// Older than every retention period, so session, capture and diagnostic
	// data all exceed their age periods; diagnostics still await backup.
	oldest := storagegc.RetentionPeriod
	for _, d := range []time.Duration{storagegc.CaptureRetentionPeriod, diagnosticlog.RetentionPeriod} {
		if d > oldest {
			oldest = d
		}
	}
	old := now.Add(-(oldest + 24*time.Hour))
	c.Now = func() time.Time { return old }
	inspection := &applyInspection{}
	storeRoot, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ns, err := couchcore.ExistingCouchNamespace(storeRoot)
	if err != nil {
		t.Fatal(err)
	}
	store, err := couchcore.NewCoordinatedThreadStore(ns, c)
	if err != nil {
		t.Fatal(err)
	}
	const scope = "816fc349d3faebf8"
	owners := map[string]artifactpath.StorageOwner{}
	payloads := map[string]string{}
	write := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	seed := func(label, tag, repoScope string) {
		t.Helper()
		o, e := artifactpath.NewStorageOwner(c.Root, repoScope, tag)
		if e != nil {
			t.Fatal(e)
		}
		owners[label] = o
		paths, e := artifactpath.ResolveScoped(o.Directory(), o.Tag)
		if e != nil {
			t.Fatal(e)
		}
		payloads[label] = paths.Draft()
		write(payloads[label], "payload-"+label)
		if e := c.Initialize(ctx, o); e != nil {
			t.Fatal(e)
		}
	}
	seed("standalone", "old-standalone", "")
	seed("visible", "couch-0000000000000001", scope)
	seed("new-archive", "couch-0000000000000002", scope)
	seed("old-archive", "couch-0000000000000003", scope)
	seed("live", "live-owner", "")
	record := func(label string) couchcore.ThreadRecord {
		o := owners[label]
		return couchcore.ThreadRecord{SchemaVersion: couchcore.ThreadSchemaVersion, Address: couchcore.ThreadAddress{RepoScope: o.RepoScope, Tag: couchcore.ThreadTag(o.Tag)}, StartingPath: c.Root, WorkingPath: c.Root, CreatedAt: old, Revision: 1, LatestLaunchProfile: &couchcore.LaunchProfile{Agent: "codex", Argv: []string{}}}
	}
	for _, label := range []string{"new-archive", "old-archive"} {
		r := record(label)
		if _, err := store.CreateThread(r); err != nil {
			t.Fatal(err)
		}
		if label == "new-archive" {
			c.Now = func() time.Time { return now }
		} else {
			c.Now = func() time.Time { return old }
		}
		if err := store.ArchiveThread(r.Address, couchcoretest.RetireProof(t, store, r.Address)); err != nil {
			t.Fatal(err)
		}
	}
	// A genuine persisted parked record, created through the store's lifecycle
	// transitions. The helper identity is fixture metadata, never a process target.
	c.Now = func() time.Time { return old }
	r := record("visible")
	r.Incarnations = []couchcore.ThreadIncarnation{{PID: 42, Identity: "fixture-previous-helper", State: couchcore.IncarnationLive, LaunchProfile: r.LatestLaunchProfile}}
	r, err = store.CreateThread(r)
	if err != nil {
		t.Fatal(err)
	}
	identity := couchcore.ParkIdentity{Nonce: "park-0123456789abcdef", Address: r.Address, PID: 42, ProcessIdentity: "fixture-previous-helper"}
	r, err = store.BeginPark(r.Address, r.Revision, identity)
	if err != nil {
		t.Fatal(err)
	}
	r, err = store.AdvancePark(context.Background(), r.Address, r.Revision, couchcore.ParkEvent{Kind: couchcore.ParkRequestCommitted, Identity: identity, Attempt: 1})
	if err != nil {
		t.Fatal(err)
	}
	r, err = store.FinalizePark(context.Background(), r.Address, r.Revision, identity, 1, old)
	if err != nil {
		t.Fatal(err)
	}
	if r.VerifiedPark == nil {
		t.Fatal("fixture is not parked")
	}
	process, err := storagegc.CurrentProcessIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.RegisterProcess(ctx, owners["live"], process, "public-apply-fixture"); err != nil {
		t.Fatal(err)
	}
	p, err := artifactpath.ResolveScoped(owners["visible"].Directory(), owners["visible"].Tag)
	if err != nil {
		t.Fatal(err)
	}
	capture, err := p.ParkedScrollbackArtifacts(old.Format("20060102T150405"))
	if err != nil {
		t.Fatal(err)
	}
	write(capture.Raw, "old capture")
	write(capture.Events, "old offsets")
	debug := filepath.Join(owners["visible"].Directory(), "wrap-events-"+owners["visible"].Tag+".jsonl")
	write(debug, "old diagnostic")
	for _, path := range []string{capture.Raw, capture.Events, debug} {
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
	}
	run := func(args ...string) string {
		t.Helper()
		var out, errout bytes.Buffer
		args = append([]string{"--root", c.Root}, args...)
		if code := runWithService(args, func(string) string { return "" }, &out, &errout, func(root string) (*gcruntime.Service, error) { return gcruntime.NewWithInspection(root, inspection) }); code != 0 {
			t.Fatalf("pair gc %v: code=%d %s", args, code, &errout)
		}
		return out.String()
	}
	report := func(args ...string) gcruntime.Report {
		t.Helper()
		var result gcruntime.Report
		if err := json.Unmarshal([]byte(run(append(args, "--json")...)), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	for _, label := range []string{"new-archive", "old-archive"} {
		if _, err := os.Stat(filepath.Join(ns.Dir(), "threadstore", "archive", scope, owners[label].Tag+".json")); err != nil {
			t.Fatal(err)
		}
	}
	// Session collection stays fenced; diagnostics are independent.
	before := report("--apply")
	if before.Storage.MigrationComplete || before.Storage.Collected != 0 || before.DiagnosticCollectedBytes != int64(len("old diagnostic")) {
		t.Fatalf("collected before migration: %+v", before)
	}
	for _, path := range payloads {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
	run("--complete-migration", "--store", ns.Dir())
	preview := report()
	if !preview.Storage.MigrationComplete {
		t.Fatal("public acknowledgement not observed")
	}
	decisions := map[string]storagegc.RetentionState{}
	for _, item := range preview.Storage.Items {
		if item.Bucket == artifactpath.SessionRetention {
			decisions[item.Owner.Tag] = item.Decision.State
		}
	}
	for label, want := range map[string]storagegc.RetentionState{"standalone": storagegc.Eligible, "visible": storagegc.Protected, "new-archive": storagegc.Grace, "old-archive": storagegc.Eligible, "live": storagegc.Live} {
		if got := decisions[owners[label].Tag]; got != want {
			t.Fatalf("preview %s=%s want %s; %+v", label, got, want, preview.Storage)
		}
	}
	// One external legacy writer must veto both storage and diagnostic cleanup.
	inspection.runtimes = []diagnosticlog.Runtime{{Process: diagnosticlog.Registration{PID: 999999, Birth: "fixture-legacy", Root: c.Root}}}
	blocked := report("--apply")
	if blocked.Storage.Collected != 0 || blocked.DiagnosticCollectedBytes != 0 {
		t.Fatalf("legacy writer failed to protect both inventories: %+v", blocked)
	}
	inspection.runtimes = nil
	applied := report("--apply")
	if applied.Storage.Collected != 3 || applied.DiagnosticCollectedBytes != 0 {
		t.Fatalf("expected two owners and one capture collected, diagnostic retained: %+v", applied)
	}
	absent := []string{payloads["standalone"], payloads["old-archive"], capture.Raw, capture.Events, filepath.Join(ns.Dir(), "threadstore", "archive", scope, owners["old-archive"].Tag+".json")}
	for _, path := range absent {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Errorf("expected collected %s: %v", path, err)
		}
	}
	if _, err := os.Stat(debug); !os.IsNotExist(err) {
		t.Fatal("expired diagnostic remains", err)
	}
	for _, row := range report().Diagnostics {
		if row.Path == debug {
			t.Fatal("removed diagnostic still reported", row)
		}
	}
	for _, label := range []string{"visible", "new-archive", "live"} {
		b, err := os.ReadFile(payloads[label])
		if err != nil || string(b) != "payload-"+label {
			t.Fatalf("changed survivor %s: %q %v", label, b, err)
		}
	}
	recordPath, err := store.RecordPath(r.Address)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{recordPath, filepath.Join(ns.Dir(), "threadstore", "archive", scope, owners["new-archive"].Tag+".json")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
	again := report("--apply")
	if again.Storage.Collected != 0 || again.DiagnosticCollectedBytes != 0 {
		t.Fatalf("repeat apply not idempotent: %+v", again)
	}
	t.Log(fmt.Sprintf("public preview/migration/apply: old standalone+archive+capture collected; diagnostic awaits backup; parked/new archive/live retained; repeat collected=%d", again.Storage.Collected))
}

// Mutable inventories model external writers without querying unrelated host processes.
type applyInspection struct {
	holders  []diagnosticlog.Registration
	runtimes []diagnosticlog.Runtime
}

func (i *applyInspection) OpenFiles(context.Context, string) ([]diagnosticlog.Registration, error) {
	return i.holders, nil
}
func (i *applyInspection) Runtimes(context.Context) ([]diagnosticlog.Runtime, error) {
	return i.runtimes, nil
}
