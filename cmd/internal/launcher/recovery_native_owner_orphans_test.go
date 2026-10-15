package launcher

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func orphanOwnerDiscovery(check RecoveryNativeOwnerCheck) func(context.Context, string) ([]artifactpath.StorageOwner, error) {
	return func(context.Context, string) ([]artifactpath.StorageOwner, error) {
		o, e := artifactpath.NewStorageOwner(check.DataDir, check.RepoScope, "other")
		return []artifactpath.StorageOwner{o}, e
	}
}
func noOrphanOwners(context.Context, string) ([]artifactpath.StorageOwner, error) { return nil, nil }
func TestRecoveryNativeOwnerRetainedChildOutlivesWrapper(t *testing.T) {
	for _, kind := range []string{"same", "different", "gone", "reused", "unknown", "changed-between-observations"} {
		t.Run(kind, func(t *testing.T) {
			id := "selected"
			if kind == "different" || kind == "changed-between-observations" {
				id = "other"
			}
			w, check := nativeOwnerFixture(t, "codex", id)
			snapshot := w.snapshots[0]
			snapshot.Processes = snapshot.Processes[1:]
			snapshot.Processes[0].PPID = 1
			delete(snapshot.Commands, 42)
			switch kind {
			case "gone":
				snapshot.Processes = nil
			case "reused":
				snapshot.Processes[0].Identity = "replacement"
			case "unknown":
				snapshot.Processes[0].Identity = ""
			}
			w.snapshots = []recoveryTerminalSnapshot{snapshot}
			if kind == "changed-between-observations" {
				next := snapshot
				next.Processes = append([]procutil.Process(nil), snapshot.Processes...)
				next.Processes[0].Identity = "replacement"
				w.snapshots = append(w.snapshots, next)
			}
			err := checkRecoveryNativeOwnersWithDiscovery(context.Background(), check, w.Observe, w.Read, orphanOwnerDiscovery(check))
			allowed := kind == "different" || kind == "gone" || kind == "reused"
			if (err == nil) != allowed {
				t.Fatalf("orphan=%s err=%v", kind, err)
			}
		})
	}
}
func TestRecoveryNativeOwnerAdmissionProofAllowsOnlyExactDifferentKeyWriter(t *testing.T) {
	for _, kind := range []string{"spawning", "running", "same-pid-fresh", "wrong-parent", "wrong-wrapper", "unknown-wrapper"} {
		t.Run(kind, func(t *testing.T) {
			legacy, check := nativeOwnerFixture(t, "codex", "other")
			world := newAdmissionWorld()
			world.identities[42] = "wrapper"
			world.identities[43] = "agent-generation"
			world.parents[42] = 1
			world.parents[43] = 42
			world.lives[42] = storagegc.ProcessAlive
			world.lives[43] = storagegc.ProcessAlive
			store := world.store(t.TempDir())
			r := admissionRequest(t, "other")
			grant := reserveAdmission(t, store, r)
			if kind == "unknown-wrapper" {
				world.as(12)
			} else {
				world.as(42)
			}
			claim, e := store.Claim(context.Background(), grant, r.Agent, r.CommandDigest)
			if e != nil {
				t.Fatal(e)
			}
			if kind != "spawning" && kind != "unknown-wrapper" {
				if e = store.PublishChild(context.Background(), claim, 43); e != nil {
					t.Fatal(e)
				}
			}
			switch kind {
			case "same-pid-fresh":
				world.lives[43] = storagegc.ProcessDead
			case "wrong-parent":
				world.parents[43] = 1
			case "wrong-wrapper":
				world.identities[42] = "replacement"
			}
			world.as(7)
			requested := admissionRequest(t, "selected")
			legacy.readErr = errors.New("best effort readiness not yet published")
			check.io = &nativeOwnerCheckIO{store: store, observe: legacy.Observe, read: legacy.Read, discover: noOrphanOwners}
			_, e = store.Reserve(context.Background(), requested, func(ctx context.Context, owners NativeAdmissionOwners) error {
				check.Admissions = owners
				return CheckRecoveryNativeOwner(ctx, check)
			})
			allowed := kind == "spawning" || kind == "running"
			if errors.Is(e, context.DeadlineExceeded) {
				t.Fatal("recursive ownership-view lock", e)
			}
			if (e == nil) != allowed {
				t.Fatalf("admission proof kind=%s err=%v", kind, e)
			}
		})
	}
}

func TestRecoveryNativeOwnerDiscoversRetainedOrphanFromCanonicalFilesystem(t *testing.T) {
	w, check := nativeOwnerFixture(t, "codex", "selected")
	owner, _ := artifactpath.NewStorageOwner(check.DataDir, check.RepoScope, "other")
	paths, _ := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
	if err := os.MkdirAll(owner.Directory(), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Ledger(), w.raw, 0600); err != nil {
		t.Fatal(err)
	}
	raw, err := readiness.Encode(w.ready)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(AgentReadyPath(owner.Directory(), owner.Tag, "codex"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	w.snapshots[0].Processes = w.snapshots[0].Processes[1:]
	w.snapshots[0].Processes[0].PPID = 1
	delete(w.snapshots[0].Commands, 42)
	if err = checkRecoveryNativeOwners(context.Background(), check, w.Observe, osfs.ReadRegularMetadata); err == nil {
		t.Fatal("canonical retained child was not discovered")
	}
	check.NativeID = "different"
	if err = checkRecoveryNativeOwners(context.Background(), check, w.Observe, osfs.ReadRegularMetadata); err != nil {
		t.Fatal("unrelated canonical native key blocked", err)
	}
}
func TestRecoveryNativeOwnerRechecksRetainedOrphanReadiness(t *testing.T) {
	w, check := nativeOwnerFixture(t, "codex", "different")
	w.snapshots[0].Processes = w.snapshots[0].Processes[1:]
	w.snapshots[0].Processes[0].PPID = 1
	delete(w.snapshots[0].Commands, 42)
	reads := 0
	read := func(ctx context.Context, path string, limit int64) ([]byte, error) {
		if strings.Contains(path, "agent-ready-") {
			reads++
			if reads == 2 {
				w.ready.Nonce = "replacement"
			}
		}
		return w.Read(ctx, path, limit)
	}
	if err := checkRecoveryNativeOwnersWithDiscovery(context.Background(), check, w.Observe, read, orphanOwnerDiscovery(check)); err == nil {
		t.Fatal("changed orphan readiness accepted")
	}
}
func TestLegacyNativeOwnerDiscoveryRefusesUnsafeScopeWithoutFollowingIt(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Mkdir(root+"/repos", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, root+"/repos/abcdef1234567890"); err != nil {
		t.Fatal(err)
	}
	if _, err := discoverLegacyNativeOwners(context.Background(), root); err == nil {
		t.Fatal("followed foreign namespace link")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := discoverLegacyNativeOwners(ctx, root); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation=%v", err)
	}
}

func TestRecoveryNativeOwnerPublicPreflightReadsAdmissionWithoutMutation(t *testing.T) {
	legacy, check := nativeOwnerFixture(t, "codex", "other")
	world := newAdmissionWorld()
	world.identities[42] = "wrapper"
	world.identities[43] = "agent-generation"
	world.parents[42] = 1
	world.parents[43] = 42
	world.lives[42] = storagegc.ProcessAlive
	world.lives[43] = storagegc.ProcessAlive
	store := world.store(t.TempDir())
	req := admissionRequest(t, "other")
	grant := reserveAdmission(t, store, req)
	world.as(42)
	claim, err := store.Claim(context.Background(), grant, req.Agent, req.CommandDigest)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.PublishChild(context.Background(), claim, 43); err != nil {
		t.Fatal(err)
	}
	path := admissionStorePath(t, store.Home)
	before, _ := os.ReadFile(path)
	store.io.write = func(string, []byte) error { t.Fatal("public preflight mutated native authority"); return nil }
	store.io.inspect = func(storagegc.ProcessIdentity) storagegc.Liveness {
		t.Fatal("public preflight collected entries")
		return storagegc.ProcessUnknown
	}
	legacy.readErr = errors.New("ready not published")
	check.io = &nativeOwnerCheckIO{store: store, observe: legacy.Observe, read: legacy.Read, discover: noOrphanOwners}
	if err = CheckRecoveryNativeOwner(context.Background(), check); err != nil {
		t.Fatal("public preflight blocked known different key", err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("read-only preflight changed durable store")
	}
	// Losing exact child proof must restore the legacy refusal on this public path.
	world.lives[43] = storagegc.ProcessDead
	if err = CheckRecoveryNativeOwner(context.Background(), check); err == nil {
		t.Fatal("same-PID Fresh borrowed old public admission proof")
	}
}
