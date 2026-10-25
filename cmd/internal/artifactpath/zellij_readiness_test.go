package artifactpath

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestZellijReadinessConstructorsUseCanonicalNamespace(t *testing.T) {
	root := t.TempDir()
	cache, err := ResolvePairCache(root)
	if err != nil {
		t.Fatal(err)
	}
	namespace := "/tmp/zellij-test"
	session := "pair-thread"
	namespaceKey, sessionKey := sha256.Sum256([]byte(namespace)), sha256.Sum256([]byte(session))
	lease, err := cache.ZellijNamespaceLease(namespace)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, ".cache", "pair", "zellij-scan-locks-v1", fmt.Sprintf("%02x.lock", namespaceKey[0])); lease != want {
		t.Fatalf("lease=%s want=%s", lease, want)
	}
	directory, err := cache.ZellijReadyDirectory(namespace)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, ".cache", "pair", "zellij-ready-v1", fmt.Sprintf("%x", namespaceKey)); directory != want {
		t.Fatalf("directory=%s want=%s", directory, want)
	}
	marker, err := cache.ZellijReadyMarker(namespace, session)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(directory, fmt.Sprintf("%x.json", sessionKey)); marker != want {
		t.Fatalf("marker=%s want=%s", marker, want)
	}
	if _, err := os.Stat(filepath.Join(root, ".cache")); !os.IsNotExist(err) {
		t.Fatalf("pure constructors touched filesystem: %v", err)
	}
	for _, bad := range []string{"", "relative", "/tmp/../namespace", "/", "/tmp/bad\x00", "/tmp/bad\n", "/tmp/" + strings.Repeat("x", 4096)} {
		if _, err := cache.ZellijNamespaceLease(bad); err == nil {
			t.Errorf("lease accepted namespace %q", bad)
		}
		if _, err := cache.ZellijReadyDirectory(bad); err == nil {
			t.Errorf("directory accepted namespace %q", bad)
		}
		if _, err := cache.ZellijReadyMarker(bad, session); err == nil {
			t.Errorf("marker accepted namespace %q", bad)
		}
	}
	for _, bad := range []string{"", "../session", "a/b", "bad\x00"} {
		if _, err := cache.ZellijReadyMarker(namespace, bad); err == nil {
			t.Errorf("accepted session %q", bad)
		}
	}
	if _, err := (PairCachePaths{}).ZellijNamespaceLease(namespace); err == nil {
		t.Error("unresolved lease cache accepted")
	}
	if _, err := (PairCachePaths{}).ZellijReadyDirectory(namespace); err == nil {
		t.Error("unresolved directory cache accepted")
	}
	if _, err := (PairCachePaths{}).ZellijReadyMarker(namespace, session); err == nil {
		t.Error("unresolved marker cache accepted")
	}
}

func TestZellijReadinessKeysRemainBoundedAndCurrent(t *testing.T) {
	cache, _ := ResolvePairCache("/home/test")
	leases := map[string]bool{}
	for n := 0; n < 10000; n++ {
		namespace := fmt.Sprintf("/tmp/namespace-%d", n)
		lease, err := cache.ZellijNamespaceLease(namespace)
		if err != nil {
			t.Fatal(err)
		}
		again, err := cache.ZellijNamespaceLease(namespace)
		if err != nil || again != lease {
			t.Fatalf("unstable lease: %q %q %v", lease, again, err)
		}
		leases[lease] = true
	}
	if len(leases) != 256 {
		t.Fatalf("lease family=%d keys, want fixed256", len(leases))
	}
	// The session key intentionally has no launch ordinal or retry component.
	// Replacement reuses one current positive slot; exact-generation authority
	// is checked by the lifecycle owner before retiring or publishing its bytes.
	marker, _ := cache.ZellijReadyMarker("/tmp/namespace", "pair-thread")
	again, _ := cache.ZellijReadyMarker("/tmp/namespace", "pair-thread")
	otherNamespace, _ := cache.ZellijReadyMarker("/tmp/other", "pair-thread")
	otherSession, _ := cache.ZellijReadyMarker("/tmp/namespace", "pair-other")
	if marker != again || marker == otherNamespace || marker == otherSession {
		t.Fatalf("current marker keys collide or drift: %q %q %q %q", marker, again, otherNamespace, otherSession)
	}
}

func TestZellijReadinessManifestProtectsLifecycleArtifacts(t *testing.T) {
	tokens := map[string]string{}
	for _, family := range Families {
		tokens[family.Name] = family.Token
	}
	for family, token := range map[string]string{"zellij-namespace-lease": "zellij-scan-locks-v1", "zellij-ready-marker": "zellij-ready-v1"} {
		if tokens[family] != token {
			t.Errorf("family %s token=%q want=%q", family, tokens[family], token)
		}
		policy, ok := GCClassifications[family]
		if !ok || policy.Retention != SessionRetention || policy.Disposition != GCProtected || policy.Authority == "" {
			t.Errorf("%s generic GC must preserve lifecycle-owned artifacts: %+v", family, policy)
		}
	}
	// Generic thread-age collection has neither namespace exclusion nor exact
	// launch-generation proof and must not claim these namespace-wide artifacts.
	cache, _ := ResolvePairCache("/home/test")
	lease, _ := cache.ZellijNamespaceLease("/tmp/namespace")
	marker, _ := cache.ZellijReadyMarker("/tmp/namespace", "pair-thread")
	owner, _ := NewStorageOwner("/home/test/.cache/pair", "", "thread")
	for _, path := range []string{lease, marker} {
		if member, err := MatchArtifact(path, []StorageOwner{owner}, []string{"claude"}); err == nil {
			t.Errorf("generic GC adopted %s as %+v", path, member)
		}
	}
}

func TestZellijReadinessSourcesArePreciselyClassified(t *testing.T) {
	found := map[string]SourceClassification{}
	for _, source := range SourceClassifications {
		found[source.Path] = source
	}
	wantFamilies := []string{"zellij-namespace-lease", "zellij-ready-marker"}
	constructor := found["cmd/internal/artifactpath/zellij_readiness.go"]
	if constructor.Kind != Constructor || !reflect.DeepEqual(constructor.Families, wantFamilies) {
		t.Errorf("constructor classification=%+v", constructor)
	}
	consumer := found["cmd/internal/zellijquery/readiness_os.go"]
	if consumer.Kind != ResolvedConsumer || !reflect.DeepEqual(consumer.Families, wantFamilies) || len(consumer.BindingNames) == 0 {
		t.Errorf("readiness store lacks exact resolved witnesses: %+v", consumer)
	}
	for _, path := range NonArtifactSources {
		if path == constructor.Path || path == consumer.Path {
			t.Errorf("artifact owner also exempted as nonartifact: %s", path)
		}
	}
}

// #866 the production store carries resolved PairCachePaths through a field,
// beyond the generic manifest guard's local-variable witnesses. Keep this
// proof specific to that carrier instead of broadening constructor authority.
func zellijReadinessStoreBindingViolations(body string) []string {
	file, err := parser.ParseFile(token.NewFileSet(), "readiness_os.go", body, 0)
	if err != nil {
		return []string{err.Error()}
	}
	alias := importAliases(file)[artifactpathImportPath]
	resolved := resolverResults(file, alias)["ResolvePairCache"]
	parents := astParents(file)
	typed, initialized := false, false
	members := map[string]bool{"ZellijNamespaceLease": false, "ZellijReadyMarker": false, "ZellijReadyDirectory": false}
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.TypeSpec:
			if n.Name.Name != "readinessStore" {
				return true
			}
			structure, ok := n.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range structure.Fields.List {
				typ, ok := field.Type.(*ast.SelectorExpr)
				owner, ownerOK := selectorOwner(typ)
				if !ok || !ownerOK || owner != alias || typ.Sel.Name != "PairCachePaths" {
					continue
				}
				for _, name := range field.Names {
					if name.Name == "paths" {
						typed = true
					}
				}
			}
		case *ast.CompositeLit:
			typ, ok := n.Type.(*ast.Ident)
			if !ok || typ.Name != "readinessStore" {
				return true
			}
			for _, element := range n.Elts {
				keyed, ok := element.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := keyed.Key.(*ast.Ident)
				if !ok || key.Name != "paths" {
					continue
				}
				value, ok := keyed.Value.(*ast.Ident)
				if ok && resolved[value.Obj] {
					initialized = true
				}
			}
		case *ast.CallExpr:
			method, ok := n.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if _, tracked := members[method.Sel.Name]; !tracked {
				return true
			}
			carrier, ok := method.X.(*ast.SelectorExpr)
			if ok && carrier.Sel.Name == "paths" && callResultConsumed(parents, n) {
				members[method.Sel.Name] = true
			}
		}
		return true
	})
	var violations []string
	if !typed || !initialized {
		violations = append(violations, "readinessStore.paths must retain the canonical ResolvePairCache result")
	}
	for _, member := range []string{"ZellijNamespaceLease", "ZellijReadyMarker", "ZellijReadyDirectory"} {
		if !members[member] {
			violations = append(violations, "readinessStore.paths must consume "+member)
		}
	}
	return violations
}

func TestZellijReadinessStoreRetainsCanonicalResolver(t *testing.T) {
	body, err := os.ReadFile("../zellijquery/readiness_os.go")
	if err != nil {
		t.Fatal(err)
	}
	if violations := zellijReadinessStoreBindingViolations(string(body)); len(violations) != 0 {
		t.Fatalf("canonical readiness path proof: %v", violations)
	}
}

func TestZellijReadinessStoreBindingNegativeControls(t *testing.T) {
	const valid = `package fixture
import ap "github.com/xianxu/pair/cmd/internal/artifactpath"
type readinessStore struct { paths ap.PairCachePaths }
func store(home string) readinessStore {
 paths, _ := ap.ResolvePairCache(home)
 return readinessStore{paths: paths}
}
func paths(s readinessStore) {
 lease, _ := s.paths.ZellijNamespaceLease("/tmp/native")
 marker, _ := s.paths.ZellijReadyMarker("/tmp/native", "session")
 directory, _ := s.paths.ZellijReadyDirectory("/tmp/native")
 _, _, _ = lease, marker, directory
}`
	if violations := zellijReadinessStoreBindingViolations(valid); len(violations) != 0 {
		t.Fatalf("positive control: %v", violations)
	}
	for name, body := range map[string]string{
		"untyped carrier":         strings.Replace(valid, "paths ap.PairCachePaths", "paths interface{}", 1),
		"discarded resolver":      strings.Replace(valid, "paths: paths", "paths: unrelated", 1),
		"reconstructed lock":      strings.Replace(valid, `s.paths.ZellijNamespaceLease("/tmp/native")`, `handmade("/tmp/native")`, 1),
		"reconstructed marker":    strings.Replace(valid, `s.paths.ZellijReadyMarker("/tmp/native", "session")`, `handmade("/tmp/native")`, 1),
		"reconstructed directory": strings.Replace(valid, `s.paths.ZellijReadyDirectory("/tmp/native")`, `handmade("/tmp/native")`, 1),
		"unused witness":          strings.Replace(valid, `lease, _ := s.paths.ZellijNamespaceLease("/tmp/native")`, `s.paths.ZellijNamespaceLease("/tmp/native")`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if len(zellijReadinessStoreBindingViolations(body)) == 0 {
				t.Fatal("broken readiness binding escaped source proof")
			}
		})
	}
}

func TestPaneReadyDiagnosticsHaveExactVocabularyAllowances(t *testing.T) {
	const rel = "cmd/pair-go/main.go"
	var classification SourceClassification
	for _, source := range SourceClassifications {
		if source.Path == rel {
			classification = source
		}
	}
	for _, tc := range []struct {
		name, diagnostic string
		wantViolation    bool
	}{
		{"registered diagnostic", "pair pane-ready: unexpected arguments\n", false},
		{"unregistered diagnostic", "pair pane-ready: arbitrary-new-artifact\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `package main
import "fmt"
func runWithLegacyRuntime(args []string) {
 if args[1] == "pane-ready" { fmt.Fprintf(nil, "DIAGNOSTIC"); fmt.Fprintf(nil, "pair pane-ready: %v\n", nil) }
}`
			body = strings.Replace(body, "DIAGNOSTIC", strings.Trim(fmt.Sprintf("%q", tc.diagnostic), "\""), 1)
			fileSet := token.NewFileSet()
			file, err := parser.ParseFile(fileSet, rel, body, 0)
			if err != nil {
				t.Fatal(err)
			}
			violations := goClassificationViolations(rel, fileSet, file, classification)
			if got := len(violations) != 0; got != tc.wantViolation {
				t.Fatalf("violations=%v want violation=%v", violations, tc.wantViolation)
			}
		})
	}
}
