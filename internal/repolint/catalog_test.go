// Package repolint qualifies the repository boundary around the upstream catalog.
package repolint

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/ariadne/pkg/golint"
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
			nodes = rule.Match(file)
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
	decoder := json.NewDecoder(bytes.NewReader(data))
	var value func() error
	value = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return fmt.Errorf("duplicate or invalid JSON field %v", key)
				}
				seen[name] = true
				if err := value(); err != nil {
					return err
				}
			}
		case '[':
			for decoder.More() {
				if err := value(); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("invalid JSON delimiter %q", delimiter)
		}
		_, err = decoder.Token()
		return err
	}
	return value()
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

func gitOutput(root string, args ...string) ([]byte, error) {
	command := exec.Command("git", args...)
	command.Dir = root
	var stderr bytes.Buffer
	command.Stderr = &stderr
	out, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git %v: %w: %s", args, err, stderr.String())
	}
	return out, nil
}
func eligible(filename string) bool {
	return strings.HasSuffix(filename, ".go") && !strings.HasPrefix(filename, "workshop/") && !strings.Contains("/"+filename, "/testdata/")
}
func scanTracked(root string, readFile func(string) ([]byte, error)) ([]locatedFinding, []sleepAt, int, error) {
	inventory, err := gitOutput(root, "ls-files", "-z", "--cached")
	if err != nil {
		return nil, nil, 0, err
	}
	var findings []locatedFinding
	var sites []sleepAt
	count := 0
	for _, filename := range strings.Split(string(inventory), "\x00") {
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
		fileFindings, err := golint.Check(filename, source, golint.Catalog)
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

// mainRef follows the repository's configured main tracking branch. A missing
// configuration refuses rather than guessing an authority from candidate HEAD.
func mainRef(root string) (string, error) {
	config, err := gitOutput(root, "config", "--get-regexp", `^branch\.main\.(remote|merge)$`)
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 {
			return "", err
		}
	}
	values := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(config)), "\n") {
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, " ")
		if !ok || values[key] != "" {
			return "", fmt.Errorf("ambiguous main tracking configuration")
		}
		values[key] = strings.TrimSpace(value)
	}
	if len(values) > 0 {
		remote, merge := values["branch.main.remote"], values["branch.main.merge"]
		if remote == "" || !strings.HasPrefix(merge, "refs/heads/") {
			return "", fmt.Errorf("invalid main tracking configuration")
		}
		if remote == "." {
			return merge, nil
		}
		return "refs/remotes/" + remote + "/" + strings.TrimPrefix(merge, "refs/heads/"), nil
	}
	// Detached CI checkouts need not create a local main branch. Full-history
	// checkout still provides its remote main ref; ambiguity is an error.
	refs, err := gitOutput(root, "for-each-ref", "--format=%(refname)", "refs/remotes/")
	if err != nil {
		return "", err
	}
	var candidates []string
	for _, ref := range strings.Fields(string(refs)) {
		if strings.HasSuffix(ref, "/main") {
			candidates = append(candidates, ref)
		}
	}
	if len(candidates) != 1 {
		return "", fmt.Errorf("need exactly one remote main ref without main tracking config, found %d", len(candidates))
	}
	return candidates[0], nil
}

func debtBaseline(root string) (Debt, error) {
	ref, err := mainRef(root)
	if err != nil {
		return Debt{}, err
	}
	base, err := gitOutput(root, "merge-base", "HEAD", ref)
	if err != nil {
		return Debt{}, err
	}
	commit := strings.TrimSpace(string(base))
	tree, err := gitOutput(root, "ls-tree", "--name-only", commit, "--", debtPath)
	if err != nil {
		return Debt{}, err
	}
	if strings.TrimSpace(string(tree)) != "" {
		data, err := gitOutput(root, "show", commit+":"+debtPath)
		if err != nil {
			return Debt{}, err
		}
		return decodeDebt(data)
	}
	// Bootstrap is legal only before this feature reaches main and only from the
	// immutable pre-implementation source anchor, never the candidate inventory.
	history, err := gitOutput(root, "log", "-1", "--format=%H", commit, "--", debtPath)
	if err != nil {
		return Debt{}, err
	}
	if len(bytes.TrimSpace(history)) != 0 {
		return Debt{}, fmt.Errorf("main previously had debt file but baseline blob is missing")
	}
	if _, err := gitOutput(root, "merge-base", "--is-ancestor", bootstrapAnchor, "HEAD"); err != nil {
		return Debt{}, fmt.Errorf("bootstrap source anchor not in candidate ancestry: %w", err)
	}
	return anchorDebt(root, bootstrapAnchor)
}
func anchorDebt(root, anchor string) (Debt, error) {
	inventory, err := gitOutput(root, "ls-tree", "-r", "--name-only", "-z", anchor)
	if err != nil {
		return Debt{}, err
	}
	var names []string
	for _, name := range strings.Split(string(inventory), "\x00") {
		if eligible(name) && strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
	}
	var requests strings.Builder
	for _, name := range names {
		fmt.Fprintln(&requests, anchor+":"+name)
	}
	command := exec.Command("git", "cat-file", "--batch")
	command.Dir = root
	command.Stdin = strings.NewReader(requests.String())
	data, err := command.Output()
	if err != nil {
		return Debt{}, fmt.Errorf("read bootstrap source blobs: %w", err)
	}
	reader := bufio.NewReader(bytes.NewReader(data))
	debt := Debt{Version: 1, Sites: []SleepSite{}}
	for _, name := range names {
		header, err := reader.ReadString('\n')
		if err != nil {
			return Debt{}, err
		}
		fields := strings.Fields(header)
		if len(fields) != 3 || fields[1] != "blob" {
			return Debt{}, fmt.Errorf("invalid bootstrap blob header %q", header)
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil || size < 0 || size > len(data) {
			return Debt{}, fmt.Errorf("invalid bootstrap blob size")
		}
		source := make([]byte, size)
		if _, err := io.ReadFull(reader, source); err != nil {
			return Debt{}, err
		}
		separator, err := reader.ReadByte()
		if err != nil || separator != '\n' {
			return Debt{}, fmt.Errorf("invalid bootstrap blob separator")
		}
		sites, err := sleepSites(name, source)
		if err != nil {
			return Debt{}, err
		}
		for _, s := range sites {
			debt.Sites = append(debt.Sites, s.Site)
		}
	}
	if _, err := reader.ReadByte(); err != io.EOF {
		return Debt{}, fmt.Errorf("unexpected trailing bootstrap bytes")
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
	t.Logf("checked %d tracked Go sources, %d rules; %d exact sleep debt sites", count, len(golint.Catalog), len(debt.Sites))
	for _, f := range remaining {
		t.Errorf("%s:%s", f.Path, f.Finding.String())
	}
}
