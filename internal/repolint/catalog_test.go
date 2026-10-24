// Package repolint qualifies the repository boundary around the upstream catalog.
package repolint

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/ariadne/pkg/golint"
	"github.com/xianxu/pair/internal/jsonstrict"
	"github.com/xianxu/pair/internal/reporoot"
)

const debtPath = "internal/repolint/sleep-debt.json"
const bootstrapAnchor = "0f8e23d0a142f962fc011875c92da68216762058"
const sleepRule = "test-fixed-sleep"

type SleepSite struct {
	Path     string `json:"path"`
	Function string `json:"function"`
	Call     string `json:"call"`
	Ordinal  int    `json:"ordinal"`
}
type Debt struct {
	Version int         `json:"version"`
	Sites   []SleepSite `json:"sites"`
}
type sleepAt struct {
	Site SleepSite
	Line int
}
type locatedFinding struct {
	Path    string
	Finding golint.Finding
}

// sleepSites uses the upstream matcher itself, so admitted sites cannot drift
// from the catalog's threshold, import handling or generated-source policy.
func sleepSites(filename string, source []byte) ([]sleepAt, error) {
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, filename, source, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	if ast.IsGenerated(file) || !strings.HasSuffix(filename, "_test.go") {
		return nil, nil
	}
	var nodes []ast.Node
	for _, rule := range golint.Catalog {
		if rule.Name == sleepRule {
			nodes = rule.Match(file, &golint.CheckContext{Path: filename, Package: path.Dir(filename), PackageName: file.Name.Name})
		}
	}
	owners := map[ast.Node]string{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		name := fn.Name.Name
		if fn.Recv != nil {
			var buf bytes.Buffer
			if err := format.Node(&buf, token.NewFileSet(), fn.Recv.List[0].Type); err != nil {
				return nil, err
			}
			name = buf.String() + "." + name
		}
		ast.Inspect(fn, func(n ast.Node) bool {
			if n != nil {
				owners[n] = name
			}
			return true
		})
	}
	counts := map[string]int{}
	var out []sleepAt
	for _, node := range nodes {
		var buf bytes.Buffer
		if err := format.Node(&buf, token.NewFileSet(), node); err != nil {
			return nil, err
		}
		function := owners[node]
		if function == "" {
			function = "<package>"
		}
		call := buf.String()
		key := function + "\x00" + call
		counts[key]++
		out = append(out, sleepAt{SleepSite{filename, function, call, counts[key]}, fs.Position(node.Pos()).Line})
	}
	return out, nil
}

func decodeDebt(data []byte) (Debt, error) {
	var d Debt
	if err := uniqueJSONKeys(data); err != nil {
		return d, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return d, err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return d, fmt.Errorf("debt has trailing data")
	}
	if d.Version != 1 || d.Sites == nil {
		return d, fmt.Errorf("debt requires version 1 and a sites array")
	}
	seen := map[SleepSite]bool{}
	for _, s := range d.Sites {
		if s.Path == "" || path.Clean(s.Path) != s.Path || strings.ContainsAny(s.Path, "\\\x00") || strings.HasPrefix(s.Path, "/") || strings.HasPrefix(s.Path, "../") || !eligible(s.Path) || !strings.HasSuffix(s.Path, "_test.go") || s.Function == "" || s.Ordinal < 1 {
			return d, fmt.Errorf("invalid debt site %+v", s)
		}
		expression, err := parser.ParseExpr(s.Call)
		if err != nil {
			return d, fmt.Errorf("invalid sleep call: %w", err)
		}
		var canonical bytes.Buffer
		if err := format.Node(&canonical, token.NewFileSet(), expression); err != nil {
			return d, err
		}
		call, ok := expression.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 || canonical.String() != s.Call {
			return d, fmt.Errorf("invalid canonical sleep call %q", s.Call)
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Sleep" {
			return d, fmt.Errorf("non-sleep debt call %q", s.Call)
		}
		pkg, ok := selector.X.(*ast.Ident)
		if !ok || pkg.Name == "_" {
			return d, fmt.Errorf("invalid sleep package in %q", s.Call)
		}
		source := []byte("package p\nimport " + pkg.Name + " \"time\"\nfunc F(){" + s.Call + "}")
		sites, err := sleepSites(s.Path, source)
		if err != nil || len(sites) != 1 {
			return d, fmt.Errorf("debt call is not an upstream fixed sleep: %q", s.Call)
		}
		if seen[s] {
			return d, fmt.Errorf("duplicate debt site %+v", s)
		}
		seen[s] = true
	}
	return d, nil
}

// Encoding/json deliberately accepts repeated keys; debt is an authority, so
// repeated fields must refuse rather than let the last spelling win.
func uniqueJSONKeys(data []byte) error {
	return jsonstrict.Scan(data, jsonstrict.Options{})
}

func checkDebt(findings []locatedFinding, sites []sleepAt, debt Debt) ([]locatedFinding, error) {
	allowed := map[SleepSite]bool{}
	used := map[SleepSite]bool{}
	for _, s := range debt.Sites {
		if allowed[s] {
			return nil, fmt.Errorf("duplicate debt site %+v", s)
		}
		allowed[s] = true
	}
	byLine := map[string][]SleepSite{}
	for _, s := range sites {
		key := s.Site.Path + ":" + strconv.Itoa(s.Line)
		byLine[key] = append(byLine[key], s.Site)
	}
	var remaining []locatedFinding
	for _, f := range findings {
		if f.Finding.Rule != sleepRule {
			remaining = append(remaining, f)
			continue
		}
		key := f.Path + ":" + strconv.Itoa(f.Finding.Line)
		candidates := byLine[key]
		if len(candidates) == 0 {
			return nil, fmt.Errorf("sleep finding has no AST site: %s", key)
		}
		site := candidates[0]
		byLine[key] = candidates[1:]
		if allowed[site] {
			used[site] = true
		} else {
			remaining = append(remaining, f)
		}
	}
	for s := range allowed {
		if !used[s] {
			return nil, fmt.Errorf("unused sleep debt %+v", s)
		}
	}
	return remaining, nil
}
func checkShrink(previous, current Debt) error {
	allowed := map[SleepSite]bool{}
	for _, s := range previous.Sites {
		allowed[s] = true
	}
	for _, s := range current.Sites {
		if !allowed[s] {
			return fmt.Errorf("sleep debt may only shrink; unadmitted site %+v", s)
		}
	}
	return nil
}

func eligible(filename string) bool {
	return strings.HasSuffix(filename, ".go") && !strings.HasPrefix(filename, "workshop/") && !strings.Contains("/"+filename, "/testdata/")
}
func scanTracked(root string, readFile func(string) ([]byte, error)) ([]locatedFinding, []sleepAt, int, error) {
	return scanTrackedWithRepository(gitRepository{root: root}, root, readFile)
}
func scanTrackedWithRepository(repository repositoryRead, root string, readFile func(string) ([]byte, error)) ([]locatedFinding, []sleepAt, int, error) {
	inventory, err := repository.TrackedPaths()
	if err != nil {
		return nil, nil, 0, err
	}
	var findings []locatedFinding
	var sites []sleepAt
	count := 0
	for _, filename := range inventory {
		if !eligible(filename) {
			continue
		}
		source, err := readFile(filepath.Join(root, filepath.FromSlash(filename)))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, nil, count, fmt.Errorf("read %s: %w", filename, err)
		}
		count++
		// The staged debt ratchet is the only sleep exception mechanism. Even a
		// reasoned inline sleep suppression would conceal a new site from it.
		parsed, err := parser.ParseFile(token.NewFileSet(), filename, source, parser.ParseComments)
		if err != nil {
			return nil, nil, count, err
		}
		if !ast.IsGenerated(parsed) {
			for _, group := range parsed.Comments {
				for _, c := range group.List {
					// Match Check's directive grammar exactly: the prefix includes
					// one ASCII space, the remainder is TrimSpace'd, and the
					// rule ends at the next ASCII space (not arbitrary fields).
					text, directive := strings.CutPrefix(c.Text, "//lint:allow ")
					name, _, _ := strings.Cut(strings.TrimSpace(text), " ")
					if directive && name == sleepRule {
						return nil, nil, count, fmt.Errorf("%s: inline sleep suppression bypasses tracked debt", filename)
					}
				}
			}
		}
		fileFindings, err := golint.Check(filename, source, pairCatalogFor(filename))
		if err != nil {
			return nil, nil, count, err
		}
		for _, f := range fileFindings {
			findings = append(findings, locatedFinding{filename, f})
		}
		fileSites, err := sleepSites(filename, source)
		if err != nil {
			return nil, nil, count, err
		}
		sites = append(sites, fileSites...)
	}
	if count == 0 {
		return nil, nil, 0, fmt.Errorf("no tracked eligible Go sources")
	}
	return findings, sites, count, nil
}

// mainRefWithRepository follows configured main tracking, or the unique remote
// main ref in a detached checkout. Missing or ambiguous authority refuses.
func mainRefWithRepository(repository repositoryRead) (string, error) {
	config, err := repository.MainTracking()
	if err != nil {
		return "", err
	}
	if config.Configured {
		if config.Remote == "" || !strings.HasPrefix(config.Merge, "refs/heads/") {
			return "", fmt.Errorf("invalid main tracking configuration")
		}
		if config.Remote == "." {
			return config.Merge, nil
		}
		return "refs/remotes/" + config.Remote + "/" + strings.TrimPrefix(config.Merge, "refs/heads/"), nil
	}
	refs, err := repository.RemoteMainRefs()
	if err != nil {
		return "", err
	}
	if len(refs) != 1 {
		return "", fmt.Errorf("need exactly one remote main ref without main tracking config, found %d", len(refs))
	}
	return refs[0], nil
}
func debtBaseline(root string) (Debt, error) {
	return debtBaselineWithRepository(gitRepository{root: root})
}
func debtBaselineWithRepository(repository repositoryRead) (Debt, error) {
	ref, err := mainRefWithRepository(repository)
	if err != nil {
		return Debt{}, err
	}
	commit, err := repository.MergeBase("HEAD", ref)
	if err != nil {
		return Debt{}, err
	}
	paths, err := repository.PathsAt(commit)
	if err != nil {
		return Debt{}, err
	}
	for _, name := range paths {
		if name == debtPath {
			blobs, err := repository.ReadBlobs(commit, []string{debtPath})
			if err != nil {
				return Debt{}, err
			}
			data, ok := blobs[debtPath]
			if !ok {
				return Debt{}, fmt.Errorf("missing debt baseline blob")
			}
			return decodeDebt(data)
		}
	}
	existed, err := repository.PathExisted(commit, debtPath)
	if err != nil {
		return Debt{}, err
	}
	if existed {
		return Debt{}, fmt.Errorf("main previously had debt file but baseline blob is missing")
	}
	ancestor, err := repository.IsAncestor(bootstrapAnchor, "HEAD")
	if err != nil {
		return Debt{}, err
	}
	if !ancestor {
		return Debt{}, fmt.Errorf("bootstrap source anchor not in candidate ancestry")
	}
	return anchorDebtWithRepository(repository, bootstrapAnchor)
}
func anchorDebt(root, anchor string) (Debt, error) {
	return anchorDebtWithRepository(gitRepository{root: root}, anchor)
}
func anchorDebtWithRepository(repository repositoryRead, anchor string) (Debt, error) {
	inventory, err := repository.PathsAt(anchor)
	if err != nil {
		return Debt{}, err
	}
	var names []string
	for _, name := range inventory {
		if eligible(name) && strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
	}
	blobs, err := repository.ReadBlobs(anchor, names)
	if err != nil {
		return Debt{}, err
	}
	debt := Debt{Version: 1, Sites: []SleepSite{}}
	for _, name := range names {
		source, ok := blobs[name]
		if !ok {
			return Debt{}, fmt.Errorf("missing bootstrap source blob %q", name)
		}
		sites, err := sleepSites(name, source)
		if err != nil {
			return Debt{}, err
		}
		for _, site := range sites {
			debt.Sites = append(debt.Sites, site.Site)
		}
	}
	sort.Slice(debt.Sites, func(i, j int) bool { return fmt.Sprint(debt.Sites[i]) < fmt.Sprint(debt.Sites[j]) })
	if len(debt.Sites) != 103 {
		return Debt{}, fmt.Errorf("bootstrap anchor has %d sleep sites, expected 103", len(debt.Sites))
	}
	return debt, nil
}

func TestTrackedGoFilesPassCatalog(t *testing.T) {
	root, err := reporoot.Current()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, debtPath))
	if err != nil {
		t.Fatal(err)
	}
	debt, err := decodeDebt(data)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := debtBaseline(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkShrink(baseline, debt); err != nil {
		t.Fatal(err)
	}
	findings, sites, count, err := scanTracked(root, os.ReadFile)
	if err != nil {
		t.Fatal(err)
	}
	remaining, err := checkDebt(findings, sites, debt)
	if err != nil {
		t.Fatal(err)
	}
	localFindings, err := readNativeFixtureRoots(root, os.ReadFile)
	if err != nil {
		t.Fatal(err)
	}
	remaining = append(remaining, localFindings...)
	t.Logf("checked %d tracked Go sources, %d catalog rules plus Pair native-fixture-roots; %d exact sleep debt sites", count, len(pairCatalog), len(debt.Sites))
	for _, f := range remaining {
		t.Errorf("%s:%s", f.Path, f.Finding.String())
	}
}
