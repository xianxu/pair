package repolint

import (
	"strings"
	"testing"

	"github.com/xianxu/ariadne/pkg/golint"
)

func TestModuleCacheOwnerRule(t *testing.T) {
	for _, tc := range []struct {
		name, imports, body string
		want                int
	}{
		{"raw cache", ``, `root := t.TempDir(); env := []string{"GOMODCACHE="+root}; _ = env`, 1},
		{"inline raw cache", ``, `_ = []string{"GOMODCACHE="+t.TempDir()}`, 1},
		{"raw cache field", ``, `root := t.TempDir(); _ = caches{GOMODCACHE:root}`, 1},
		{"raw cache setenv", ``, `root := t.TempDir(); t.Setenv("GOMODCACHE",root)`, 1},
		{"raw os setenv", `import "os"`, `root := t.TempDir(); _ = os.Setenv("GOMODCACHE",root)`, 1},
		{"joined raw cache", `import "path/filepath"`, `root := t.TempDir(); cache := filepath.Join(root,"modules"); _ = []string{"GOMODCACHE="+cache}`, 1},
		{"managed allocation", `import "github.com/xianxu/pair/cmd/internal/testpair"`, `root := testpair.CacheTempDir(t); _ = []string{"GOMODCACHE="+root}`, 0},
		{"aliased managed allocation", `import owned "github.com/xianxu/pair/cmd/internal/testpair"`, `root := owned.CacheTempDir(t); _ = caches{GOMODCACHE:root}`, 0},
		{"registered exact cleanup", `import "github.com/xianxu/pair/cmd/internal/testpair"`, `root := t.TempDir(); t.Cleanup(func(){if err := testpair.RemoveCacheTree(root); err != nil {t.Error(err)}}); _ = []string{"GOMODCACHE="+root}`, 0},
		{"deferred exact cleanup", `import "github.com/xianxu/pair/cmd/internal/testpair"`, `root := t.TempDir(); defer testpair.RemoveCacheTree(root); t.Setenv("GOMODCACHE",root)`, 0},
		{"unrelated cleanup", `import "github.com/xianxu/pair/cmd/internal/testpair"`, `root := t.TempDir(); other := t.TempDir(); t.Cleanup(func(){_ = testpair.RemoveCacheTree(other)}); _ = []string{"GOMODCACHE="+root}`, 1},
		{"unregistered cleanup", `import "github.com/xianxu/pair/cmd/internal/testpair"`, `root := t.TempDir(); cleanup := func(){_ = testpair.RemoveCacheTree(root)}; _ = cleanup; _ = []string{"GOMODCACHE="+root}`, 1},
		{"unexecuted deferred cleanup", `import "github.com/xianxu/pair/cmd/internal/testpair"`, `root := t.TempDir(); cleanup := func(){defer testpair.RemoveCacheTree(root)}; _ = cleanup; _ = []string{"GOMODCACHE="+root}`, 1},
		{"unexecuted registered cleanup", `import "github.com/xianxu/pair/cmd/internal/testpair"`, `root := t.TempDir(); cleanup := func(){t.Cleanup(func(){_ = testpair.RemoveCacheTree(root)})}; _ = cleanup; _ = []string{"GOMODCACHE="+root}`, 1},
		{"wrong helper import", `import testpair "example.com/testpair"`, `root := t.TempDir(); defer testpair.RemoveCacheTree(root); _ = []string{"GOMODCACHE="+root}`, 1},
		{"shadowed helper", `import "github.com/xianxu/pair/cmd/internal/testpair"`, `root := t.TempDir(); testpair := fake; defer testpair.RemoveCacheTree(root); _ = []string{"GOMODCACHE="+root}`, 1},
		{"shadowed root", `import "github.com/xianxu/pair/cmd/internal/testpair"`, `root := t.TempDir(); t.Cleanup(func(){root := other; _ = testpair.RemoveCacheTree(root)}); _ = []string{"GOMODCACHE="+root}`, 1},
		{"joined managed root", `import ("path/filepath"; "github.com/xianxu/pair/cmd/internal/testpair")`, `root := testpair.CacheTempDir(t); _ = caches{GOMODCACHE:filepath.Join(root,"modules")}`, 0},
		{"cleanup parent", `import ("path/filepath"; "github.com/xianxu/pair/cmd/internal/testpair")`, `root := t.TempDir(); t.Cleanup(func(){_ = testpair.RemoveCacheTree(root)}); _ = caches{GOMODCACHE:filepath.Join(root,"modules")}`, 0},
		{"cleanup child does not own parent", `import ("path/filepath"; "github.com/xianxu/pair/cmd/internal/testpair")`, `root := t.TempDir(); t.Cleanup(func(){_ = testpair.RemoveCacheTree(filepath.Join(root,"other"))}); _ = caches{GOMODCACHE:root}`, 1},
		{"caller cache", `import "os"`, `_ = []string{"GOMODCACHE="+os.Getenv("GOMODCACHE")}`, 0},
		{"other cache key", ``, `_ = []string{"GOCACHE="+t.TempDir()}`, 0},
		{"home raw cleanup", `import "os"`, `root,_ := os.MkdirTemp("","test-"); defer os.RemoveAll(root); _ = os.Setenv("HOME",root)`, 1},
		{"home aliased os cleanup", `import sys "os"`, `root,_ := sys.MkdirTemp("","test-"); defer sys.RemoveAll(root); _ = sys.Setenv("HOME",root)`, 1},
		{"home managed cleanup", `import ("os"; "github.com/xianxu/pair/cmd/internal/testpair")`, `root,_ := os.MkdirTemp("","test-"); defer testpair.RemoveCacheTree(root); _ = os.Setenv("HOME",root)`, 0},
		{"home unrelated socket cleanup", `import "os"`, `root,_ := os.MkdirTemp("","test-"); sockets,_ := os.MkdirTemp("","socket-"); defer os.RemoveAll(sockets); _ = os.Setenv("HOME",root)`, 0},
		{"home unrelated helper no blessing", `import ("os"; "github.com/xianxu/pair/cmd/internal/testpair")`, `root,_ := os.MkdirTemp("","test-"); defer os.RemoveAll(root); defer testpair.RemoveCacheTree(other); _ = os.Setenv("HOME",root)`, 1},
		{"home shadowed cleanup root", `import "os"`, `root,_ := os.MkdirTemp("","test-"); t.Cleanup(func(){root := other; _ = os.RemoveAll(root)}); _ = os.Setenv("HOME",root)`, 0},
		{"testing home raw cleanup", `import "os"`, `root := t.TempDir(); t.Cleanup(func(){_ = os.RemoveAll(root)}); t.Setenv("HOME",root)`, 1},
		{"shadowed os", `import "os"`, `os := fake; root,_ := os.MkdirTemp("","test-"); defer os.RemoveAll(root); _ = os.Setenv("HOME",root)`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package fixture\nimport \"testing\"\n" + tc.imports + "\nfunc fixture(t *testing.T){" + tc.body + "}")
			findings, err := golint.Check("cmd/example/cache_test.go", source, pairCatalogFor("cmd/example/cache_test.go"))
			if err != nil {
				t.Fatal(err)
			}
			var matching []golint.Finding
			for _, finding := range findings {
				if finding.Rule == "test-module-cache-owner" {
					matching = append(matching, finding)
				}
			}
			if len(matching) != tc.want {
				t.Fatalf("cache owner findings=%v want=%d", matching, tc.want)
			}
			for _, finding := range matching {
				if finding.Line <= 0 || !strings.Contains(finding.Message, "pair#887") {
					t.Fatalf("unexpected finding %+v", finding)
				}
			}
		})
	}
}

func TestModuleCacheOwnerCatalogScope(t *testing.T) {
	source := []byte(`package fixture; import "testing"; func fixture(t *testing.T){root := t.TempDir(); _ = []string{"GOMODCACHE="+root}}`)
	for _, tc := range []struct {
		path string
		want int
	}{
		{"cmd/example/cache_test.go", 1},
		{"cmd/example/cache.go", 0},
	} {
		t.Run(tc.path, func(t *testing.T) {
			repository := newFakeRepository()
			repository.index[tc.path] = source
			findings, _, _, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
			if err != nil || len(findings) != tc.want {
				t.Fatalf("findings=%v err=%v want=%d", findings, err, tc.want)
			}
		})
	}
}
