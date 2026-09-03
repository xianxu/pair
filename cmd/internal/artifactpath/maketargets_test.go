package artifactpath

import (
	"os"
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
