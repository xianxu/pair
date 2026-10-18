package repolint

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// Transitions belong only to the conformance driver, not the scanner's read
// interface. Both drivers expose the same bounded, clean-checkout histories.
type repositoryFixture interface {
	Write(string, []byte) error
	Stage(string) error
	Remove(string, bool) error
	Commit() (string, error)
	CommitParents([]string) (string, error)
	Checkout(string) error
	MoveRef(string, string) error
	ConfigureMain(mainTracking) error
}

func TestRepositoryReadContract(t *testing.T) {
	for _, backend := range []string{"stateful-fake", "real-git"} {
		t.Run(backend, func(t *testing.T) {
			var repository repositoryRead
			var fixture repositoryFixture
			if backend == "stateful-fake" {
				fake := newFakeRepository()
				repository, fixture = fake, fake
			} else {
				root := fixtureGit(t)
				real := gitRepository{root: root}
				repository = real
				fixture = &gitFixture{repository: real}
			}
			runRepositoryReadContract(t, repository, fixture)
		})
	}
}

func runRepositoryReadContract(t *testing.T, repository repositoryRead, fixture repositoryFixture) {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	write := func(name string, data []byte) { must(fixture.Write(name, data)); must(fixture.Stage(name)) }
	commit := func() string { t.Helper(); id, err := fixture.Commit(); must(err); return id }
	const name = "line\nbreak.bin"
	original := []byte("package p\n// binary payload: \x00\n")
	write("a.go", []byte("package p\n"))
	write(name, original)
	first := commit()
	must(fixture.Write("untracked.go", []byte("package untracked\n")))
	paths, err := repository.TrackedPaths()
	must(err)
	slices.Sort(paths)
	if !reflect.DeepEqual(paths, []string{"a.go", name}) {
		t.Fatalf("index includes untracked source: %q", paths)
	}
	must(fixture.Remove("a.go", false))
	paths, err = repository.TrackedPaths()
	must(err)
	if !slices.Contains(paths, "a.go") {
		t.Fatal("worktree deletion changed index")
	}
	must(fixture.Write("a.go", []byte("package p\n")))
	must(fixture.Remove("a.go", true))
	paths, err = repository.TrackedPaths()
	must(err)
	if slices.Contains(paths, "a.go") {
		t.Fatal("staged deletion retained path")
	}
	must(fixture.Write(name, []byte("candidate bytes\n")))
	must(fixture.Stage(name))
	second := commit()
	old, err := repository.ReadBlobs(first, []string{name})
	must(err)
	if !bytes.Equal(old[name], original) {
		t.Fatalf("commit snapshot changed: %q", old[name])
	}
	old[name][0] = 'X'
	old, err = repository.ReadBlobs(first, []string{name})
	must(err)
	if !bytes.Equal(old[name], original) {
		t.Fatal("caller mutated committed blob")
	}
	snapshot, err := repository.PathsAt(first)
	must(err)
	if !slices.Contains(snapshot, "a.go") {
		t.Fatal("historical tree lost deleted path")
	}
	had, err := repository.PathExisted(second, "a.go")
	must(err)
	if !had {
		t.Fatal("deleted path lost history")
	}
	had, err = repository.PathExisted(first, "never")
	must(err)
	if had {
		t.Fatal("unobserved path invented")
	}
	must(fixture.MoveRef("refs/heads/topic", first))
	must(fixture.Checkout("refs/heads/topic"))
	write("topic.go", []byte("package topic\n"))
	third := commit()
	base, err := repository.MergeBase(second, third)
	must(err)
	if base != first {
		t.Fatalf("common base=%s want%s", base, first)
	}
	ancestor, err := repository.IsAncestor(first, third)
	must(err)
	if !ancestor {
		t.Fatal("ancestor not recognized")
	}
	ancestor, err = repository.IsAncestor(second, third)
	must(err)
	if ancestor {
		t.Fatal("divergent commit treated as ancestor")
	}
	// Resolving a named ref is observable state, not a canned reply sequence.
	must(fixture.MoveRef("refs/remotes/upstream/main", second))
	remote, err := repository.ReadBlobs("refs/remotes/upstream/main", []string{name})
	must(err)
	if string(remote[name]) != "candidate bytes\n" {
		t.Fatalf("ref did not resolve immutable tree: %q", remote[name])
	}
	must(fixture.ConfigureMain(mainTracking{Configured: true, Remote: "upstream", Merge: "refs/heads/main"}))
	ref, err := mainRefWithRepository(repository)
	must(err)
	if ref != "refs/remotes/upstream/main" {
		t.Fatal(ref)
	}
	must(fixture.ConfigureMain(mainTracking{}))
	ref, err = mainRefWithRepository(repository)
	must(err)
	if ref != "refs/remotes/upstream/main" {
		t.Fatal(ref)
	}
	must(fixture.MoveRef("refs/remotes/another/main", second))
	if _, err := mainRefWithRepository(repository); err == nil {
		t.Fatal("ambiguous detached authority admitted")
	}
	// The same historical-source bootstrap derivation must work against both
	// backends. Its caller still fixes the immutable production anchor.
	seed := []byte("package p\nimport \"time\"\nfunc TestSeed(){\n" + strings.Repeat("time.Sleep(time.Second)\n", 103) + "}\n")
	write("seed_test.go", seed)
	seeded := commit()
	initial, err := anchorDebtWithRepository(repository, seeded)
	must(err)
	if len(initial.Sites) != 103 {
		t.Fatalf("bootstrap sites=%d", len(initial.Sites))
	}
	must(fixture.Write("seed_test.go", []byte("package broken")))
	again, err := anchorDebtWithRepository(repository, seeded)
	must(err)
	if !reflect.DeepEqual(initial, again) {
		t.Fatal("working source authorized historical debt")
	}
	// Crisscross merges have two incomparable best bases; neither backend
	// may arbitrarily choose an authority. A disconnected root has no base.
	left, err := fixture.CommitParents([]string{first})
	must(err)
	right, err := fixture.CommitParents([]string{first})
	must(err)
	mergeLeft, err := fixture.CommitParents([]string{left, right})
	must(err)
	mergeRight, err := fixture.CommitParents([]string{right, left})
	must(err)
	if _, err := repository.MergeBase(mergeLeft, mergeRight); err == nil {
		t.Fatal("ambiguous crisscross base admitted")
	}
	disconnected, err := fixture.CommitParents(nil)
	must(err)
	if _, err := repository.MergeBase(first, disconnected); err == nil {
		t.Fatal("disconnected history given a base")
	}
	for _, check := range []struct {
		name string
		run  func() error
	}{
		{"unknown-inventory-revision", func() error { _, e := repository.PathsAt("unknown-commit"); return e }},
		{"unknown-blob-revision", func() error { _, e := repository.ReadBlobs("unknown-commit", []string{name}); return e }},
		{"unknown-empty-blob-revision", func() error { _, e := repository.ReadBlobs("unknown-commit", nil); return e }},
		{"missing-blob", func() error { _, e := repository.ReadBlobs(first, []string{"missing.go"}); return e }},
		{"unknown-history", func() error { _, e := repository.PathExisted("unknown-commit", "a.go"); return e }},
		{"unknown-common-base", func() error { _, e := repository.MergeBase(first, "unknown-commit"); return e }},
		{"unknown-ancestor", func() error { _, e := repository.IsAncestor("unknown-commit", first); return e }},
	} {
		t.Run(check.name, func(t *testing.T) {
			if err := check.run(); err == nil {
				t.Fatal("unknown state admitted")
			}
		})
	}
}

// gitFixture writes only to an owned disposable repository. Commands share
// gitRepository's sole execution boundary with reads; it is not exposed by the
// production-consumed repositoryRead interface.
type gitFixture struct {
	repository gitRepository
	sequence   int
}

func (f *gitFixture) Write(name string, data []byte) error {
	p := filepath.Join(f.repository.root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0600)
}
func (f *gitFixture) Stage(name string) error {
	_, err := f.repository.run(nil, "add", "--", name)
	return err
}
func (f *gitFixture) Remove(name string, stage bool) error {
	if err := os.Remove(filepath.Join(f.repository.root, filepath.FromSlash(name))); err != nil {
		return err
	}
	if stage {
		return f.Stage(name)
	}
	return nil
}
func (f *gitFixture) Commit() (string, error) {
	f.sequence++
	if _, err := f.repository.run(nil, "commit", "--allow-empty", "-m", fmt.Sprintf("fixture%d", f.sequence)); err != nil {
		return "", err
	}
	data, err := f.repository.run(nil, "rev-parse", "HEAD")
	return string(bytes.TrimSpace(data)), err
}
func (f *gitFixture) Checkout(ref string) error {
	_, err := f.repository.run(nil, "checkout", ref)
	return err
}
func (f *gitFixture) MoveRef(ref, revision string) error {
	_, err := f.repository.run(nil, "update-ref", ref, revision)
	return err
}
func (f *gitFixture) ConfigureMain(config mainTracking) error {
	if !config.Configured {
		_, err := f.repository.run(nil, "config", "--remove-section", "branch.main")
		return err
	}
	for _, setting := range [][2]string{{"branch.main.remote", config.Remote}, {"branch.main.merge", config.Merge}} {
		if _, err := f.repository.run(nil, "config", setting[0], setting[1]); err != nil {
			return err
		}
	}
	return nil
}

func TestRepositoryAdmissionContract(t *testing.T) {
	for _, backend := range []string{"stateful-fake", "real-git"} {
		t.Run(backend, func(t *testing.T) {
			var repository repositoryRead
			var fixture repositoryFixture
			var readFile func(string) ([]byte, error)
			root := t.TempDir()
			if backend == "stateful-fake" {
				fake := newFakeRepository()
				repository, fixture = fake, fake
				readFile = func(name string) ([]byte, error) { return fake.readWorking(root, name) }
			} else {
				root = fixtureGit(t)
				real := gitRepository{root: root}
				repository = real
				fixture = &gitFixture{repository: real}
				readFile = os.ReadFile
			}
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			source := []byte("package p\nimport \"time\"\nfunc TestA(){time.Sleep(time.Second)}\n")
			sites, err := sleepSites("x_test.go", source)
			must(err)
			debt := Debt{Version: 1, Sites: []SleepSite{sites[0].Site}}
			encoded, err := json.Marshal(debt)
			must(err)
			for name, data := range map[string][]byte{debtPath: encoded, "x_test.go": source, "a/testdata/bad_test.go": source, "workshop/bad_test.go": source, "generated_test.go": append([]byte("// Code generated by fixture. DO NOT EDIT.\n"), source...)} {
				must(fixture.Write(name, data))
				must(fixture.Stage(name))
			}
			first, err := fixture.Commit()
			must(err)
			must(fixture.Write("untracked_test.go", source))
			must(fixture.MoveRef("refs/heads/candidate", first))
			must(fixture.Checkout("refs/heads/candidate"))
			findings, currentSites, count, err := scanTrackedWithRepository(repository, root, readFile)
			must(err)
			if len(findings) != 1 || count != 2 {
				t.Fatalf("inventory findings=%v count=%d", findings, count)
			}
			baseline, err := debtBaselineWithRepository(repository)
			must(err)
			remaining, err := checkDebt(findings, currentSites, baseline)
			must(err)
			if len(remaining) != 0 {
				t.Fatalf("exact original site rejected %v", remaining)
			}
			must(fixture.Write("new_test.go", source))
			must(fixture.Stage("new_test.go"))
			findings, currentSites, _, err = scanTrackedWithRepository(repository, root, readFile)
			must(err)
			remaining, err = checkDebt(findings, currentSites, baseline)
			must(err)
			if len(remaining) != 1 || remaining[0].Path != "new_test.go" {
				t.Fatalf("new tracked violation hidden: %v", remaining)
			}
			// A candidate can shrink the file; it cannot move its authority ahead of main.
			empty := []byte(`{"version":1,"sites":[]}`)
			must(fixture.Write(debtPath, empty))
			must(fixture.Stage(debtPath))
			candidate, err := fixture.Commit()
			must(err)
			baseline, err = debtBaselineWithRepository(repository)
			must(err)
			if len(baseline.Sites) != 1 {
				t.Fatal("candidate debt became main authority")
			}
			must(fixture.MoveRef("refs/heads/main", candidate))
			baseline, err = debtBaselineWithRepository(repository)
			must(err)
			if len(baseline.Sites) != 0 {
				t.Fatal("moved main did not retire debt")
			}
			must(fixture.Write(debtPath, []byte(`{"version":99,"sites":[]}`)))
			must(fixture.Stage(debtPath))
			malformed, err := fixture.Commit()
			must(err)
			must(fixture.MoveRef("refs/heads/main", malformed))
			if _, err := debtBaselineWithRepository(repository); err == nil {
				t.Fatal("malformed main debt admitted")
			}
			must(fixture.Remove(debtPath, true))
			deleted, err := fixture.Commit()
			must(err)
			must(fixture.MoveRef("refs/heads/main", deleted))
			if _, err := debtBaselineWithRepository(repository); err == nil {
				t.Fatal("historically deleted debt re-entered bootstrap")
			}
		})
	}
}

func TestFakeRepositoryGeneratedAncestry(t *testing.T) {
	// Enumerate every forward-edge DAG over four commits. The independent
	// transitive-closure matrix is the oracle, not another graph walk.
	const n = 4
	for mask := 0; mask < 1<<6; mask++ {
		fake := newFakeRepository()
		ids := make([]string, n)
		var reachable [n][n]bool
		bit := 0
		for child := 0; child < n; child++ {
			parents := []string{}
			reachable[child][child] = true
			for parent := 0; parent < child; parent++ {
				if mask&(1<<bit) != 0 {
					parents = append(parents, ids[parent])
					reachable[parent][child] = true
				}
				bit++
			}
			id, err := fake.commitWithParents(parents)
			if err != nil {
				t.Fatal(err)
			}
			ids[child] = id
		}
		for via := 0; via < n; via++ {
			for from := 0; from < n; from++ {
				for to := 0; to < n; to++ {
					reachable[from][to] = reachable[from][to] || (reachable[from][via] && reachable[via][to])
				}
			}
		}
		for left := 0; left < n; left++ {
			for right := 0; right < n; right++ {
				actual, err := fake.IsAncestor(ids[left], ids[right])
				if err != nil || actual != reachable[left][right] {
					t.Fatalf("graph%d ancestor%d,%d=%v err%v", mask, left, right, actual, err)
				}
				var expected []string
				for candidate := 0; candidate < n; candidate++ {
					if !reachable[candidate][left] || !reachable[candidate][right] {
						continue
					}
					best := true
					for other := 0; other < n; other++ {
						if candidate != other && reachable[other][left] && reachable[other][right] && reachable[candidate][other] {
							best = false
						}
					}
					if best {
						expected = append(expected, ids[candidate])
					}
				}
				base, err := fake.MergeBase(ids[left], ids[right])
				if len(expected) == 1 {
					if err != nil || base != expected[0] {
						t.Fatalf("graph%d base%d,%d=%s err%v expected%v", mask, left, right, base, err, expected)
					}
				} else if err == nil {
					t.Fatalf("graph%d ambiguous/no base admitted %s", mask, base)
				}
			}
		}
	}
}

func TestFakeScannerOneReadAtNAnd2N(t *testing.T) {
	for _, n := range []int{8, 16} {
		fake := newFakeRepository()
		root := t.TempDir()
		for i := 0; i < n; i++ {
			name := fmt.Sprintf("p%d.go", i)
			if err := fake.Write(name, []byte("package p\n")); err != nil {
				t.Fatal(err)
			}
			if err := fake.Stage(name); err != nil {
				t.Fatal(err)
			}
		}
		reads := 0
		_, _, count, err := scanTrackedWithRepository(fake, root, func(name string) ([]byte, error) { reads++; return fake.readWorking(root, name) })
		if err != nil || count != n || reads != n {
			t.Fatalf("n%d count%d reads%d err%v", n, count, reads, err)
		}
	}
}

func TestFakeRepositoryRefMovementAndUnknownIndex(t *testing.T) {
	fake := newFakeRepository()
	root := t.TempDir()
	if err := fake.Write("x.go", []byte("package p\n")); err != nil {
		t.Fatal(err)
	}
	if err := fake.Stage("x.go"); err != nil {
		t.Fatal(err)
	}
	first, err := fake.Commit()
	if err != nil {
		t.Fatal(err)
	}
	if err := fake.Write("x.go", []byte("package changed\n")); err != nil {
		t.Fatal(err)
	}
	if err := fake.Stage("x.go"); err != nil {
		t.Fatal(err)
	}
	second, err := fake.Commit()
	if err != nil {
		t.Fatal(err)
	}
	if err := fake.MoveRef("refs/heads/main", first); err != nil {
		t.Fatal(err)
	}
	base, err := fake.MergeBase("HEAD", first)
	if err != nil || base != first {
		t.Fatalf("HEAD ignored attached ref move %s %v", base, err)
	}
	if ancestor, err := fake.IsAncestor(second, "HEAD"); err != nil || ancestor {
		t.Fatalf("HEAD still resolves prior ref %v %v", ancestor, err)
	}
	fake.indexError = fmt.Errorf("index inaccessible")
	if _, _, _, err := scanTrackedWithRepository(fake, root, func(name string) ([]byte, error) { return fake.readWorking(root, name) }); err == nil {
		t.Fatal("unknown inventory admitted")
	}
	if _, err := fake.commitWithParents([]string{"unknown"}); err == nil {
		t.Fatal("unknown parent admitted")
	}
}

func (f *gitFixture) CommitParents(parents []string) (string, error) {
	tree, err := f.repository.run(nil, "write-tree")
	if err != nil {
		return "", err
	}
	args := []string{"commit-tree", string(bytes.TrimSpace(tree))}
	for _, parent := range parents {
		args = append(args, "-p", parent)
	}
	f.sequence++
	commit, err := f.repository.run([]byte(fmt.Sprintf("parent-fixture%d\n", f.sequence)), args...)
	if err != nil {
		return "", err
	}
	id := string(bytes.TrimSpace(commit))
	if _, err := f.repository.run(nil, "update-ref", "HEAD", id); err != nil {
		return "", err
	}
	return id, nil
}
