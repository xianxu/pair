package couchcore

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testgit"
)

var provisionFixtureBudget = testgit.FixtureTreeBudget{Files: 24, Dirs: 48, LogicalBytes: 262144, SampleHooks: 0}
var prepareWorldBudget = testgit.FixtureTreeBudget{Files: 64, Dirs: 128, LogicalBytes: 1048576, SampleHooks: 0}

func checkFixtureTree(t *testing.T, root string, budget testgit.FixtureTreeBudget) testgit.FixtureTreeStats {
	t.Helper()
	inventory, err := testgit.AuditFixtureTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := budget.Verdict(inventory); err != nil {
		t.Fatal(err)
	}
	return inventory
}

func TestSharedGitFixtureBudgetsAndIndependence(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			var provisions []*ProvisionFixture
			var worlds []prepareWorld
			var provisionTotal, prepareTotal testgit.FixtureTreeStats
			for i := 0; i < n; i++ {
				f := newProvisionFixture(t)
				w := newPrepareWorld(t)
				// Read the inventory captured by the real shared constructor, never a
				// synthetic substitute or a post-mutation filesystem sample.
				if f.constructorTree.Files == 0 || w.constructorTree.Files == 0 {
					t.Fatal("shared constructors did not audit their retained trees")
				}
				var err error
				provisionTotal, err = provisionTotal.Add(f.constructorTree)
				if err != nil {
					t.Fatal(err)
				}
				prepareTotal, err = prepareTotal.Add(w.constructorTree)
				if err != nil {
					t.Fatal(err)
				}
				provisions = append(provisions, f)
				worlds = append(worlds, w)
			}
			for _, tt := range []struct {
				name string
				got  testgit.FixtureTreeStats
				cap  testgit.FixtureTreeBudget
			}{
				{"provision", provisionTotal, provisionFixtureBudget}, {"prepare", prepareTotal, prepareWorldBudget},
			} {
				cap := testgit.FixtureTreeBudget{Files: tt.cap.Files * uint64(n), Dirs: tt.cap.Dirs * uint64(n), LogicalBytes: tt.cap.LogicalBytes * uint64(n)}
				if err := cap.Verdict(tt.got); err != nil {
					t.Fatalf("%s: %v", tt.name, err)
				}
				t.Logf("%s N=%d inventory=%+v", tt.name, n, tt.got)
			}
			for i := 0; i < n; i++ {
				f := provisions[i]
				w := worlds[i]
				if f.git(f.Primary, "branch", "--list", "only-here") != "" {
					t.Fatal("provision refs shared")
				}
				if prepareGit(t, w.host, "branch", "--list", "only-here") != "" {
					t.Fatal("prepare refs shared")
				}
				f.git(f.Primary, "branch", "only-here")
				prepareGit(t, w.host, "branch", "only-here")
				// Each clone must retain its own index/worktree and independent remote.
				writePrepareFile(t, filepath.Join(w.host, "local"), fmt.Sprint(i))
				if got := prepareGit(t, w.host, "status", "--porcelain"); got != "?? local" {
					t.Fatalf("prepare worktree status=%q", got)
				}
				if got := f.git(f.Primary, "status", "--porcelain"); got != "" {
					t.Fatalf("provision worktree status=%q", got)
				}
				if got := f.git(f.Remote, "branch", "--list", "only-here"); got != "" {
					t.Fatal("bare clone refs shared")
				}
			}
		})
	}
}
