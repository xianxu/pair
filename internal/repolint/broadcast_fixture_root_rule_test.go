package repolint

import (
	"strings"
	"testing"
)

func TestBroadcastSocketFixtureRootRule(t *testing.T) {
	for _, tc := range []struct {
		name, imports, body string
		want                int
	}{
		{"shared", `import "github.com/xianxu/pair/cmd/internal/testpair"`, `return testpair.ShortRoot(t)`, 0},
		{"aliased", `import owned "github.com/xianxu/pair/cmd/internal/testpair"`, `return owned.ShortRoot(t)`, 0},
		{"inherited parent", `import "os"`, `dir, _ := os.MkdirTemp("", "cb"); return dir`, 1},
		{"long temp dir", ``, `return t.TempDir()`, 1},
		{"unused shared root", `import "github.com/xianxu/pair/cmd/internal/testpair"`, `_ = testpair.ShortRoot(t); return t.TempDir()`, 1},
		{"wrong import", `import testpair "example.com/testpair"`, `return testpair.ShortRoot(t)`, 1},
		{"shadowed", `import "github.com/xianxu/pair/cmd/internal/testpair"`, `testpair := fake; return testpair.ShortRoot(t)`, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package broadcast\n" + tc.imports + "\nfunc shortDir(t *testing.T) string {" + tc.body + "}")
			repository := newFakeRepository()
			repository.index["cmd/internal/broadcast/cloudflared_test.go"] = source
			findings, _, _, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
			if err != nil || len(findings) != tc.want {
				t.Fatalf("findings=%v err=%v want=%d", findings, err, tc.want)
			}
			for _, finding := range findings {
				if finding.Finding.Rule != "broadcast-socket-fixture-root" || !strings.Contains(finding.Finding.Message, "pair#881") {
					t.Fatalf("unexpected finding %+v", finding)
				}
			}
		})
	}
}
