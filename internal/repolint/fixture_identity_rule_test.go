package repolint

import (
	"fmt"
	"strings"
	"testing"
)

func TestFixtureIdentityRule(t *testing.T) {
	for _, constructor := range []struct{ pkg, name string }{
		{"main", "installedEnv"},
		{"testcouch", "allocateRoots"},
	} {
		t.Run(constructor.pkg+"/"+constructor.name, func(t *testing.T) {
			for _, tc := range []struct {
				name, imports, declaration string
				want                       int
			}{
				{"missing call", "", "func %s() {}", 1},
				{"bodyless", "", "func %s()", 1},
				{"seeded", `import "github.com/xianxu/pair/cmd/internal/testpair"`, "func %s() { testpair.SeedCouchIdentity() }", 0},
				{"import alias", `import fixture "github.com/xianxu/pair/cmd/internal/testpair"`, "func %s() { fixture.SeedCouchIdentity() }", 0},
				{"wrong import", `import testpair "example.com/testpair"`, "func %s() { testpair.SeedCouchIdentity() }", 1},
				{"local same name", "", "func %s() { testpair := struct{ SeedCouchIdentity func() }{}; testpair.SeedCouchIdentity() }", 1},
				{"shadowed import", `import "github.com/xianxu/pair/cmd/internal/testpair"`, "func %s() { testpair := struct{ SeedCouchIdentity func() }{}; testpair.SeedCouchIdentity() }", 1},
				{"comment", "", "func %s() { /* testpair.SeedCouchIdentity() */ }", 1},
				{"string", "", `func %s() { _ = "testpair.SeedCouchIdentity()" }`, 1},
				{"function reference", `import "github.com/xianxu/pair/cmd/internal/testpair"`, "func %s() { _ = testpair.SeedCouchIdentity }", 1},
				{"other function seeded", `import "github.com/xianxu/pair/cmd/internal/testpair"`, "func %s() {}\nfunc other() { testpair.SeedCouchIdentity() }", 1},
				{"unrelated function", "", "func other() {} // %s", 0},
				{"method", "", "type fixture struct{}\nfunc (fixture) %s() {}", 0},
			} {
				t.Run(tc.name, func(t *testing.T) {
					source := []byte("package " + constructor.pkg + "\n" + tc.imports + "\n" + fmt.Sprintf(tc.declaration, constructor.name))
					repository := newFakeRepository()
					repository.index["cmd/internal/example/fixture_test.go"] = source
					findings, _, _, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
					if err != nil || len(findings) != tc.want {
						t.Fatalf("findings=%v err=%v want=%d", findings, err, tc.want)
					}
					for _, finding := range findings {
						if finding.Finding.Rule != "fixture-couch-identity" || !strings.Contains(finding.Finding.Message, "pair#778") {
							t.Fatalf("unexpected finding %+v", finding)
						}
					}
				})
			}
		})
	}
}

func TestFixtureIdentityRuleIgnoresOtherPackages(t *testing.T) {
	source := []byte("package unrelated\nfunc installedEnv() {}\nfunc allocateRoots() {}")
	repository := newFakeRepository()
	repository.index["cmd/internal/example/fixture_test.go"] = source
	findings, _, _, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
	if err != nil || len(findings) != 0 {
		t.Fatalf("findings=%v err=%v", findings, err)
	}
}
