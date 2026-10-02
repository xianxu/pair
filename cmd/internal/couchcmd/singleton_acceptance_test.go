package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/couchsingleton"
)

type adoptionFixture struct {
	runtime                                       OSRuntime
	root, sources, store, pair, identity, account string
}

func newAdoptionFixture(t *testing.T) adoptionFixture {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := adoptionFixture{root: root, sources: filepath.Join(root, "sources"), account: filepath.Join(root, "account")}
	f.store, f.pair, f.identity = filepath.Join(f.sources, "store"), filepath.Join(f.sources, "pair"), filepath.Join(f.sources, "identity")
	env := map[string]string{"HOME": filepath.Join(root, "ambient-home"), "COUCH_STORE_DIR": f.store, "PAIR_DATA_DIR": f.pair, "COUCH_IDENTITY_DIR": f.identity}
	f.runtime = OSRuntime{accountHome: func() (string, error) { return f.account, nil }, env: func(k string) string { return env[k] }}
	return f
}

func (f adoptionFixture) args(extra ...string) []string {
	return append([]string{"--adopt-store", f.store, "--pair-data", f.pair, "--identity-dir", f.identity}, extra...)
}

func (f adoptionFixture) run(args ...string) (int, string, string) {
	var out, stderr bytes.Buffer
	code := RunWithRuntime(args, strings.NewReader(""), &out, &stderr, f.runtime)
	return code, out.String(), stderr.String()
}

// Snapshot includes names, kinds, permissions and content, so preview cannot
// quietly create directories or coordination files while preserving payloads.
func adoptionSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if os.IsNotExist(err) && path == root {
			return nil
		}
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		value := info.Mode().String()
		if !d.IsDir() {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value += ":" + string(raw)
		}
		result[path] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func decodeAdoptionReport(t *testing.T, raw string) couchsingleton.Report {
	t.Helper()
	var report couchsingleton.Report
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("report %q: %v", raw, err)
	}
	return report
}

func seedAdoptionStore(t *testing.T, f adoptionFixture, store string) {
	t.Helper()
	_, err := (couchidentity.IdentityStore{HostDir: f.identity, StoreDir: store}).Allocate(context.Background(), couchidentity.AllocationRequest{Conversation: true, Terminal: true, RepositoryToken: "repo"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAdoptionCommandPreviewApplyAndRetryPreserveIdentity(t *testing.T) {
	f := newAdoptionFixture(t)
	seedAdoptionStore(t, f, f.store)
	before := adoptionSnapshot(t, f.root)
	code, output, stderr := f.run(f.args()...)
	if code != 0 {
		t.Fatalf("preview: %d %s", code, stderr)
	}
	report := decodeAdoptionReport(t, output)
	if report.Status != "READY" || len(report.Digest) != 64 || len(report.Blockers) != 0 {
		t.Fatalf("report: %+v", report)
	}
	if after := adoptionSnapshot(t, f.root); !reflect.DeepEqual(before, after) {
		t.Fatal("preview changed source or authority files")
	}
	identityBefore := adoptionSnapshot(t, f.identity)
	localBefore, err := os.ReadFile(filepath.Join(f.store, "identities.json"))
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		code, output, stderr = f.run(f.args("--apply", report.Digest)...)
		if code != 0 {
			t.Fatalf("apply attempt %d: %d %s", attempt, code, stderr)
		}
		var selected struct {
			Status    string                   `json:"status"`
			Selection couchsingleton.Selection `json:"selection"`
		}
		if err := json.Unmarshal([]byte(output), &selected); err != nil {
			t.Fatal(err)
		}
		want := couchsingleton.Roots{StoreDir: f.store, PairDataDir: f.pair, IdentityDir: f.identity}
		if selected.Status != "SELECTED" || selected.Selection.Roots != want {
			t.Fatalf("selection: %+v", selected)
		}
	}
	if after := adoptionSnapshot(t, f.identity); !reflect.DeepEqual(identityBefore, after) {
		t.Fatal("apply or retry changed host allocation authority")
	}
	localAfter, err := os.ReadFile(filepath.Join(f.store, "identities.json"))
	if err != nil || !bytes.Equal(localBefore, localAfter) {
		t.Fatalf("apply or retry changed local counters: %v", err)
	}
	code, output, stderr = f.run(f.args()...)
	if code != 0 || decodeAdoptionReport(t, output).Status != "SELECTED" {
		t.Fatalf("selected preview: %d %s %s", code, output, stderr)
	}
}

func TestAdoptionCommandStaleDigestDoesNotSelectOrMutateSources(t *testing.T) {
	f := newAdoptionFixture(t)
	seedAdoptionStore(t, f, f.store)
	code, output, stderr := f.run(f.args()...)
	if code != 0 {
		t.Fatalf("preview: %d %s", code, stderr)
	}
	report := decodeAdoptionReport(t, output)
	// A later allocation is valid new state, not malformed data that would
	// independently refuse even if the receipt comparison were removed.
	seedAdoptionStore(t, f, f.store)
	before := adoptionSnapshot(t, f.sources)
	code, _, stderr = f.run(f.args("--apply", report.Digest)...)
	if code == 0 || !strings.Contains(stderr, "adoption evidence changed") {
		t.Fatalf("stale receipt: %d %s", code, stderr)
	}
	if after := adoptionSnapshot(t, f.sources); !reflect.DeepEqual(before, after) {
		t.Fatal("stale apply changed sources")
	}
	if _, err := os.Stat(filepath.Join(f.account, ".local", "share", "pair-host", "singleton", "selection.json")); !os.IsNotExist(err) {
		t.Fatalf("stale apply published selection: %v", err)
	}
	code, output, stderr = f.run(f.args()...)
	if code != 0 || decodeAdoptionReport(t, output).Digest == report.Digest {
		t.Fatalf("fresh preview did not reflect new allocation: %d %s", code, stderr)
	}
}

func TestAdoptionCommandMultipleStoresRemainUnmigratedWithoutMutation(t *testing.T) {
	f := newAdoptionFixture(t)
	other := filepath.Join(f.sources, "other-store")
	seedAdoptionStore(t, f, f.store)
	seedAdoptionStore(t, f, other)
	before := adoptionSnapshot(t, f.root)
	code, output, stderr := f.run(f.args()...)
	report := decodeAdoptionReport(t, output)
	if code == 0 || report.Status != "UNMIGRATED" || len(report.Blockers) == 0 || !strings.Contains(stderr, "UNMIGRATED") {
		t.Fatalf("ambiguous preview: %d %+v %s", code, report, stderr)
	}
	seen := map[string]bool{}
	for _, c := range report.Candidates {
		if c.State == "populated" {
			seen[c.Roots.StoreDir] = true
		}
	}
	if !seen[f.store] || !seen[other] {
		t.Fatalf("report lost populated source: %+v", report)
	}
	if after := adoptionSnapshot(t, f.root); !reflect.DeepEqual(before, after) {
		t.Fatal("refused preview mutated state")
	}
	sources := adoptionSnapshot(t, f.sources)
	code, _, stderr = f.run(f.args("--apply", report.Digest)...)
	if code == 0 || !strings.Contains(stderr, "UNMIGRATED") {
		t.Fatalf("ambiguous apply: %d %s", code, stderr)
	}
	if after := adoptionSnapshot(t, f.sources); !reflect.DeepEqual(sources, after) {
		t.Fatal("refused apply mutated sources")
	}
}

func TestAdoptionCommandExclusionPreservesRetiredStoreAndCannotBypassOwner(t *testing.T) {
	f := newAdoptionFixture(t)
	other := filepath.Join(f.sources, "retired-store")
	seedAdoptionStore(t, f, f.store)
	seedAdoptionStore(t, f, other)
	ns, err := couchcore.ExistingCouchNamespace(other)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := couchcore.AcquireSupervisorLease(ns, couchcore.OSProcOps{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { owner.Close() })
	before := adoptionSnapshot(t, f.root)
	args := f.args("--exclude-store", other)
	code, output, stderr := f.run(args...)
	report := decodeAdoptionReport(t, output)
	if code == 0 || report.Status != "UNMIGRATED" {
		t.Fatalf("exclusion bypassed live owner: %d %+v %s", code, report, stderr)
	}
	ownerReported := false
	for _, c := range report.Candidates {
		if c.Roots.StoreDir == other && c.Owner != "" {
			ownerReported = true
		}
	}
	if !ownerReported {
		t.Fatal("exclusion report omitted competing live owner")
	}
	if after := adoptionSnapshot(t, f.root); !reflect.DeepEqual(before, after) {
		t.Fatal("live exclusion preview changed state")
	}
	if err := owner.Close(); err != nil {
		t.Fatal(err)
	}
	retiredBefore := adoptionSnapshot(t, other)
	code, output, stderr = f.run(args...)
	if code != 0 {
		t.Fatalf("retired exclusion preview: %d %s", code, stderr)
	}
	report = decodeAdoptionReport(t, output)
	code, _, stderr = f.run(append(args, "--apply", report.Digest)...)
	if code != 0 {
		t.Fatalf("retired exclusion apply: %d %s", code, stderr)
	}
	if after := adoptionSnapshot(t, other); !reflect.DeepEqual(retiredBefore, after) {
		t.Fatal("excluding retired inventory modified its files")
	}
	manager, request, _, err := f.runtime.singletonManager()
	if err != nil {
		t.Fatal(err)
	}
	selected, err := manager.Read(request)
	if err != nil || !reflect.DeepEqual(selected.Excluded, []string{other}) {
		t.Fatalf("excluded disposition not retained: %+v %v", selected, err)
	}
}

func TestSingletonCommandReadBeforeAdoptionIsActionableAndReadOnly(t *testing.T) {
	for _, args := range [][]string{{"--list"}, {"--archived"}, {"--actors"}, {"--internal", "publish-description", "status"}} {
		f := newAdoptionFixture(t)
		before := adoptionSnapshot(t, f.root)
		code, _, stderr := f.run(args...)
		if code == 0 || !strings.Contains(stderr, "couch --adopt-store") || !strings.Contains(stderr, f.store) {
			t.Errorf("%q missing concrete adoption command: %d %s", args, code, stderr)
		}
		if after := adoptionSnapshot(t, f.root); !reflect.DeepEqual(before, after) {
			t.Errorf("%q initialized state", args)
		}
	}
}

func TestSingletonCommandHeldHostLeaseRefusesBeforeStoreConstruction(t *testing.T) {
	f := newAdoptionFixture(t)
	m, _, _, err := f.runtime.singletonManager()
	if err != nil {
		t.Fatal(err)
	}
	ns, err := couchcore.ResolveCouchNamespace(m.AuthorityDir, f.root)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := couchcore.AcquireSupervisorLease(ns, couchcore.OSProcOps{})
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	before := adoptionSnapshot(t, f.root)
	code, _, stderr := f.run("--internal", "retry-continuation", "missing-thread")
	if code == 0 || !strings.Contains(stderr, "supervised by pid") {
		t.Fatalf("owner contention: %d %s", code, stderr)
	}
	if after := adoptionSnapshot(t, f.root); !reflect.DeepEqual(before, after) {
		t.Fatal("contender constructed a store or modified held authority")
	}
}
