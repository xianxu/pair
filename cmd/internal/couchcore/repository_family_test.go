package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func savedRepositoryFamiliesForTest(s *ThreadStore) ([]RepositoryFamily, error) {
	var families []RepositoryFamily
	err := s.withPreviewLock(func() error {
		manifest, _, _, err := s.loadManifestLocked()
		families = append(families, manifest.RepositoryFamilies...)
		return err
	})
	return families, err
}

func TestResolveFamilyStartRetainsExplicitDirectory(t *testing.T) {
	base := RepositoryFamily{RepoIdentity: "/repo/.git", PrimaryRoot: "/repo", RelativeStart: "competition/arc-agi-3"}
	for _, relative := range []string{"", "competition/arc-agi-3"} {
		request := base
		request.RelativeStart = relative
		got, err := ResolveFamilyStart(&base, request)
		if err != nil || got != base {
			t.Fatalf("got %+v %v", got, err)
		}
	}
	for _, relative := range []string{".", "competition/arc-agi-2"} {
		request := base
		request.RelativeStart = relative
		if _, err := ResolveFamilyStart(&base, request); err == nil || !strings.Contains(err.Error(), base.RelativeStart) {
			t.Fatalf("missing actionable conflict: %v", err)
		}
	}
	request := base
	request.RelativeStart = ""
	got, err := ResolveFamilyStart(nil, request)
	if err != nil || got.RelativeStart != "." {
		t.Fatal(got, err)
	}
}
func TestFamilyProjectionRejectsEscapesAndPreservesRoot(t *testing.T) {
	for _, bad := range []string{"", "..", "../other", "sub/../other", "/absolute", "sub\\outside", "sub//child"} {
		if _, err := ProjectFamilyPath("/repo", bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	for _, relative := range []string{".", "competition/arc-agi-3"} {
		path, err := ProjectFamilyPath("/repo", relative)
		if err != nil {
			t.Fatal(err)
		}
		got, err := RelativeFamilyPath("/repo", path)
		if err != nil || got != relative {
			t.Fatal(got, err)
		}
	}
	if _, err := RelativeFamilyPath("/repo", "/repo-sibling/sub"); err == nil {
		t.Fatal("sibling prefix escaped")
	}
}
func TestValidateFamilyPathChecksPhysicalContainment(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "inside"), 0700)
	alias := filepath.Join(t.TempDir(), "alias")
	os.Symlink(root, alias)
	physical, _ := filepath.EvalSymlinks(root)
	got, err := ValidateFamilyPath(alias, "inside")
	if err != nil || got != filepath.Join(physical, "inside") {
		t.Fatal(got, err)
	}
	os.Symlink(t.TempDir(), filepath.Join(root, "escape"))
	if _, err := ValidateFamilyPath(root, "escape"); err == nil {
		t.Fatal("symlink escape accepted")
	}
	missing := filepath.Join(root, "missing")
	if _, err := ValidateFamilyPath(root, "missing"); err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("missing path not named: %v", err)
	}
}
func FuzzProjectFamilyPath(f *testing.F) {
	f.Add("competition/arc-agi-3")
	f.Add("../escape")
	f.Fuzz(func(t *testing.T, relative string) {
		path, err := ProjectFamilyPath("/repo", relative)
		if err != nil {
			return
		}
		got, err := RelativeFamilyPath("/repo", path)
		if err != nil || got != relative {
			t.Fatalf("projection roundtrip %q -> %q -> %q: %v", relative, path, got, err)
		}
	})
}
func familyStoreFixture(t *testing.T) (*ThreadStore, SlotRepository, RepositoryFamily) {
	t.Helper()
	f := newProvisionFixture(t)
	repo, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := newTestThreadStore(t)
	return s, repo, RepositoryFamily{RepoIdentity: repo.Identity.RepoIdentity, PrimaryRoot: repo.Identity.PrimaryRoot, RelativeStart: "competition/arc-agi-3"}
}
func TestFamilyReservationConcurrentFirstStartsAndPreview(t *testing.T) {
	s, repo, request := familyStoreFixture(t)
	before, err := s.PreviewRepositoryFamily(context.Background(), repo, request)
	if err != nil || before != request {
		t.Fatal(before, err)
	}
	if _, err = os.Stat(s.manifestPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preview wrote manifest: %v", err)
	}
	other := request
	other.RelativeStart = "competition/arc-agi-2"
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, q := range []RepositoryFamily{request, other} {
		wg.Add(1)
		go func(q RepositoryFamily) {
			defer wg.Done()
			_, err := s.ReserveRepositoryFamily(context.Background(), repo, q)
			results <- err
		}(q)
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("first admissions won %d", successes)
	}
	families, err := savedRepositoryFamiliesForTest(s)
	if err != nil || len(families) != 1 {
		t.Fatal(families, err)
	}
	request.RelativeStart = ""
	inherited, err := s.PreviewRepositoryFamily(context.Background(), repo, request)
	if err != nil || inherited != families[0] {
		t.Fatal(inherited, err)
	}
}
func TestFamilyReservationJournalRecoversAndSurvivesArchive(t *testing.T) {
	s, repo, request := familyStoreFixture(t)
	boom := errors.New("interrupted family publication")
	s.hooks.AfterJournal = func() error { return boom }
	if _, err := s.ReserveRepositoryFamily(context.Background(), repo, request); !errors.Is(err, boom) {
		t.Fatal(err)
	}
	s.hooks = threadStoreHooks{}
	if _, err := s.ReserveRepositoryFamily(context.Background(), repo, request); err != nil {
		t.Fatal(err)
	}
	r := validThreadRecord(t)
	scope, _ := launcher.ResolveRepoScope(repo.Identity.PrimaryRoot)
	r.Address.RepoScope = scope.Key
	r.StartingPath = filepath.Join(repo.Identity.PrimaryRoot, request.RelativeStart)
	r.WorkingPath = r.StartingPath
	r, err := s.CreateThread(r)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.ArchiveThread(r.Address); err != nil {
		t.Fatal(err)
	}
	families, err := savedRepositoryFamiliesForTest(NewThreadStore(s.namespace))
	if err != nil || len(families) != 1 || families[0] != request {
		t.Fatal(families, err)
	}
	changed := request
	changed.RelativeStart = "."
	if _, err = s.ReserveRepositoryFamily(context.Background(), repo, changed); err == nil {
		t.Fatal("archive erased family reservation")
	}
}
func TestFamilyInferencePreservesLegacyPathsAndRefusesAmbiguity(t *testing.T) {
	s, repo, request := familyStoreFixture(t)
	r := validThreadRecord(t)
	scope, _ := launcher.ResolveRepoScope(repo.Identity.PrimaryRoot)
	r.Address.RepoScope = scope.Key
	r.StartingPath = filepath.Join(repo.Identity.PrimaryRoot, request.RelativeStart)
	r.WorkingPath = r.StartingPath
	r, err := s.CreateThread(r)
	if err != nil {
		t.Fatal(err)
	}
	inherited := request
	inherited.RelativeStart = ""
	got, err := s.PreviewRepositoryFamily(context.Background(), repo, inherited)
	if err != nil || got != request {
		t.Fatal(got, err)
	}
	other := r
	other.Address.Tag = "other"
	other.StartingPath = filepath.Join(repo.Identity.PrimaryRoot, "competition/arc-agi-2")
	other.WorkingPath = other.StartingPath
	other.Revision = 1
	if _, err = s.CreateThread(other); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(s.manifestPath())
	if _, err = s.ReserveRepositoryFamily(context.Background(), repo, request); err == nil || !strings.Contains(err.Error(), other.StartingPath) || !strings.Contains(err.Error(), r.StartingPath) {
		t.Fatalf("ambiguous paths not preserved in error: %v", err)
	}
	after, _ := os.ReadFile(s.manifestPath())
	if string(before) != string(after) {
		t.Fatal("ambiguous admission mutated manifest")
	}
	for _, address := range []ThreadAddress{r.Address, other.Address} {
		if _, err = s.GetThread(address); err != nil {
			t.Fatal(err)
		}
	}
}
func TestFamilyManifestRejectsInvalidDescriptors(t *testing.T) {
	s, _, family := familyStoreFixture(t)
	for _, families := range [][]RepositoryFamily{{family, family}, {{RepoIdentity: family.RepoIdentity, PrimaryRoot: family.PrimaryRoot, RelativeStart: "../escape"}}} {
		manifest := threadManifest{SchemaVersion: 2, Threads: []ThreadAddress{}, RepositoryFamilies: families}
		raw, _ := json.Marshal(manifest)
		os.MkdirAll(s.root, 0700)
		os.WriteFile(s.manifestPath(), raw, 0600)
		if _, err := savedRepositoryFamiliesForTest(s); err == nil {
			t.Fatal("accepted invalid family manifest")
		}
	}
}

func TestFamilyPreviewInfersLocalWithoutGlobalStore(t *testing.T) {
	f := newProvisionFixture(t)
	f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
	repo, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := newTestThreadStore(t)
	local := newSlotThreadStore(s.namespace, repo.Slots[0].Identity)
	r := validThreadRecord(t)
	scope, _ := launcher.ResolveRepoScope(f.host(1))
	r.Address.RepoScope = scope.Key
	r.StartingPath = filepath.Join(f.host(1), "sub")
	r.WorkingPath = r.StartingPath
	os.MkdirAll(r.StartingPath, 0700)
	if _, err = local.CreateThread(r); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(s.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("fixture global store unexpectedly exists: %v", err)
	}
	family, err := s.PreviewRepositoryFamily(context.Background(), repo, RepositoryFamily{})
	if err != nil || family.RelativeStart != "sub" {
		t.Fatalf("local retained path lost in preview: %+v %v", family, err)
	}
	if _, err = os.Stat(s.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preview initialized global store: %v", err)
	}
}
func TestFamilyInferenceIncludesArchivedStartingPaths(t *testing.T) {
	s, repo, want := familyStoreFixture(t)
	r := validThreadRecord(t)
	scope, _ := launcher.ResolveRepoScope(repo.Identity.PrimaryRoot)
	r.Address.RepoScope = scope.Key
	r.StartingPath = filepath.Join(repo.Identity.PrimaryRoot, want.RelativeStart)
	r.WorkingPath = r.StartingPath
	r, err := s.CreateThread(r)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.ArchiveThread(r.Address); err != nil {
		t.Fatal(err)
	}
	got, err := s.PreviewRepositoryFamily(context.Background(), repo, RepositoryFamily{})
	if err != nil || got != want {
		t.Fatal(got, err)
	}
}
func TestFamilyIdentityCanonicalizesPhysicalAliases(t *testing.T) {
	s, repo, want := familyStoreFixture(t)
	if _, err := s.ReserveRepositoryFamily(context.Background(), repo, want); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(want.PrimaryRoot, alias); err != nil {
		t.Fatal(err)
	}
	request := RepositoryFamily{RepoIdentity: filepath.Join(alias, ".git"), PrimaryRoot: alias}
	got, err := s.PreviewRepositoryFamily(context.Background(), repo, request)
	if err != nil || got != want {
		t.Fatal(got, err)
	}
}

func TestFamilyInferenceRefusesUnprojectableSameRepositoryWorktree(t *testing.T) {
	_, repo, _ := familyStoreFixture(t)
	r := validThreadRecord(t)
	r.StartingPath = filepath.Join(t.TempDir(), "ordinary-worktree", "sub")
	r.WorkingPath = r.StartingPath
	r.Incarnations = []ThreadIncarnation{{State: IncarnationUnknown, RepoIdentity: repo.Identity.RepoIdentity}}
	if _, _, err := InferRepositoryFamily(repo, []ThreadRecord{r}); err == nil || !strings.Contains(err.Error(), r.StartingPath) {
		t.Fatalf("unprojectable known family silently ignored: %v", err)
	}
}

func TestFamilyInferenceExcludesNestedIndependentRepository(t *testing.T) {
	_, repo, _ := familyStoreFixture(t)
	nested := filepath.Join(repo.Identity.PrimaryRoot, "nested")
	scope, _ := launcher.ResolveRepoScope(nested)
	r := validThreadRecord(t)
	r.Address.RepoScope = scope.Key
	r.StartingPath = filepath.Join(nested, "sub")
	r.WorkingPath = r.StartingPath
	r.Incarnations = []ThreadIncarnation{{State: IncarnationUnknown, RepoIdentity: filepath.Join(nested, ".git")}}
	if family, found, err := InferRepositoryFamily(repo, []ThreadRecord{r}); err != nil || found {
		t.Fatalf("nested independent repository became family evidence: %+v %v %v", family, found, err)
	}
}
