package artifactpath

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// A `go test -run <regex>` that matches NOTHING prints "ok … [no tests to run]"
// and exits 0. So a live-conformance target whose tests have been deleted does
// not fail -- it reports green, forever, which is strictly worse than the
// target being gone.
//
// pair#170 M4 deleted TestFleetPolicyResolverConformance with admission and
// left `make test-couch-policy-live` behind; it passed silently. The CI
// workflow that invoked it WAS noticed and deleted, so the loud half went and
// the quiet half stayed -- exactly backwards.
//
// This pins the class: every `-run` regex in the Makefiles must name at least
// one test that still exists.
var goTestRunPattern = regexp.MustCompile(`go test\b[^\n]*?-run\s+'([^']*)'([^\n]*)`)

func TestEveryMakefileTestSelectorMatchesALiveTest(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	names := goTestNames(t, repoRoot)

	for _, makefile := range []string{"Makefile", "Makefile.local"} {
		path := filepath.Join(repoRoot, makefile)
		raw, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range goTestRunPattern.FindAllStringSubmatch(string(raw), -1) {
			// Make doubles a literal `$` for the shell; undo it before
			// compiling the anchor.
			selector := strings.ReplaceAll(match[1], "$$", "$")
			expression, err := regexp.Compile(selector)
			if err != nil {
				t.Errorf("%s: selector %q does not compile: %v", makefile, selector, err)
				continue
			}
			if !matchesAny(expression, names) {
				t.Errorf("%s: `go test -run %s` matches no test that exists.\n"+
					"go test exits 0 with \"[no tests to run]\", so this target reports green while checking nothing.\n"+
					"Delete the target, or point it at a test that is still here.", makefile, selector)
			}
		}
	}
}

func matchesAny(expression *regexp.Regexp, names []string) bool {
	for _, name := range names {
		if expression.MatchString(name) {
			return true
		}
	}
	return false
}

// goTestNames collects every top-level Test function in the tree. Cheap textual
// scan on purpose: parsing every _test.go to answer "does this name exist"
// buys nothing, and a subtest name is not addressable by -run's first segment
// anyway.
func goTestNames(t *testing.T, repoRoot string) []string {
	t.Helper()
	declaration := regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]*)\(`)
	seen := map[string]bool{}
	err := filepath.WalkDir(repoRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "workshop", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		for _, match := range declaration.FindAllStringSubmatch(string(raw), -1) {
			seen[match[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatal("found no Test declarations; the scan is broken, not the Makefiles")
	}
	return names
}

// `go build ./cmd/foo` writes its output into the CURRENT directory, so a stray
// build from the repo root drops a binary next to the source and `git add -A`
// sweeps it in. pair#170 M4 committed a 4.5 MB Mach-O this way, and this repo
// is the ariadne base layer, so the blob would have propagated to every
// dependent repo and been permanent once merged.
//
// The .gitignore fix for that was a hand-written list, which already missed two
// of the eight main packages the day it landed -- the same recall failure one
// level down. Derived here instead.
func TestEveryMainPackageIsIgnoredAtTheRepoRoot(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	ignored, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	rules := map[string]bool{}
	for _, line := range strings.Split(string(ignored), "\n") {
		rules[strings.TrimSpace(line)] = true
	}

	for _, directory := range mainPackageDirs(t, repoRoot) {
		// `go build` names the binary after the package's DIRECTORY.
		binary := "/" + filepath.Base(directory)
		if !rules[binary] {
			t.Errorf("main package %s builds to %q at the repo root, which .gitignore does not cover.\n"+
				"A stray `go build` there is one `git add -A` from committing a multi-megabyte binary.",
				directory, strings.TrimPrefix(binary, "/"))
		}
	}
}

// mainPackageDirs asks the Go toolchain which packages are `main`. The first
// version of this tested whether a file's bytes began with "package main\n",
// which 5 of the 8 main packages in this tree fail because they open with a doc
// comment -- including cmd/probes/couchstartrecovery, the package whose stray
// binary is the entire reason the guard exists. It reported green with
// /couchstartrecovery deleted from .gitignore.
//
// The rule that came out of it: every axis of a mechanical guard's input must
// come from an oracle that already owns the answer, and the guard must be
// mutation-checked against the artifact that MOTIVATED it, not an arbitrary
// member of the set. `go list` owns the answer to "is this package main".
func mainPackageDirs(t *testing.T, repoRoot string) []string {
	t.Helper()
	// go list reports ABSOLUTE directories, so the base for filepath.Rel has to
	// be absolute too -- with a relative base, Rel returns an error for every
	// entry and the guard silently sees an empty set.
	absoluteRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "list", "-f", "{{if eq .Name \"main\"}}{{.Dir}}{{end}}", "./...")
	command.Dir = absoluteRoot
	out, listErr := command.Output()
	if listErr != nil {
		t.Skipf("go list unavailable: %v", listErr)
	}
	var dirs []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		relative, relErr := filepath.Rel(absoluteRoot, strings.TrimSpace(line))
		if relErr != nil {
			continue
		}
		dirs = append(dirs, relative)
	}
	sort.Strings(dirs)
	if len(dirs) == 0 {
		t.Fatal("go list found no main packages; the scan is broken, not the tree")
	}
	// The motivating artifact must be in scope, or the guard is green for the
	// wrong reason -- which is exactly how the first version passed.
	found := false
	for _, directory := range dirs {
		if strings.HasSuffix(directory, "couchstartrecovery") {
			found = true
		}
	}
	if !found {
		t.Fatalf("main packages = %v; the package this guard was written for is not among them", dirs)
	}
	return dirs
}
