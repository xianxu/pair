package couchcore

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testpair"
)

var recoveryLiveRootKeys = []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_RUNTIME_DIR", "ZELLIJ_SOCKET_DIR", "ZELLIJ_CONFIG_DIR"}

func setupRecoveryLiveEnvironment(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	//lint:allow tmp-literal — native sockets append version and session names; the private root must fit the tested 103-byte AF_UNIX budget even with long TMPDIR
	short, err := os.MkdirTemp("/tmp", "p636-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(short); err != nil {
			t.Errorf("clean private recovery runtime/socket roots: %v", err)
		}
	})
	home, config, data, cache := filepath.Join(root, "home"), filepath.Join(root, "config"), filepath.Join(root, "data"), filepath.Join(root, "cache")
	runtime, sockets, zellijConfig := filepath.Join(short, "r"), filepath.Join(short, "s"), filepath.Join(config, "zellij")
	for _, dir := range []string{home, config, data, cache, runtime, sockets, zellijConfig} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	// Keep default XDG Zellij configuration discovery, explicitly confined to
	// this empty directory. No configuration or setup-wizard behavior is changed.
	testpair.ApplyEnvironment(t, home, "TMPDIR=/tmp", "SHELL=/bin/sh",
		"XDG_CONFIG_HOME="+config, "XDG_DATA_HOME="+data, "XDG_CACHE_HOME="+cache,
		"XDG_RUNTIME_DIR="+runtime, "ZELLIJ_SOCKET_DIR="+sockets, "ZELLIJ_CONFIG_DIR="+zellijConfig)
}

// This bounded source guard protects the two actual live entrypoints and the
// nested helper handoff. The subprocess test above remains the behavior proof.
func recoveryLiveEnvironmentViolations(sources map[string]string) []string {
	var violations []string
	functions := map[string]*ast.FuncDecl{}
	for name, source := range sources {
		file, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
		if err != nil {
			violations = append(violations, name+": "+err.Error())
			continue
		}
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				functions[name+":"+fn.Name.Name] = fn
			}
		}
	}
	hasCall := func(fn *ast.FuncDecl, name string) bool {
		found := false
		if fn != nil {
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if ok {
					id, ok := call.Fun.(*ast.Ident)
					found = found || (ok && id.Name == name)
				}
				return true
			})
		}
		return found
	}
	for _, entry := range []string{
		"recovery_conformance_live_test.go:TestRecoveryRealHelperAndSessionConformanceLive",
		"park_lifecycle_live_test.go:newRealParkConformance",
	} {
		if !hasCall(functions[entry], "setupRecoveryLiveEnvironment") {
			violations = append(violations, entry+": missing private environment setup")
		}
	}
	child := functions["park_lifecycle_live_test.go:startParkLifecycleHelper"]
	propagated := false
	if child != nil {
		ast.Inspect(child.Body, func(node ast.Node) bool {
			assign, ok := node.(*ast.AssignStmt)
			if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
				return true
			}
			field, ok := assign.Lhs[0].(*ast.SelectorExpr)
			if !ok || field.Sel.Name != "Env" {
				return true
			}
			command, ok := field.X.(*ast.Ident)
			if !ok || command.Name != "cmd" {
				return true
			}
			call, ok := assign.Rhs[0].(*ast.CallExpr)
			if ok {
				name, ok := call.Fun.(*ast.Ident)
				propagated = ok && name.Name == "recoveryLiveChildEnvironment"
			}
			return true
		})
	}
	if !propagated {
		violations = append(violations, "startParkLifecycleHelper: missing explicit child environment handoff")
	}
	reads := map[string]bool{}
	if fn := functions["recovery_live_environment_test.go:recoveryLiveChildEnvironment"]; fn != nil {
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Getenv" {
				return true
			}
			pkg, ok := selector.X.(*ast.Ident)
			if !ok || pkg.Name != "os" {
				return true
			}
			if literal, ok := call.Args[0].(*ast.BasicLit); ok {
				key, err := strconv.Unquote(literal.Value)
				if err == nil {
					reads[key] = true
				}
			}
			return true
		})
	}
	for _, key := range recoveryLiveRootKeys {
		if !reads[key] {
			violations = append(violations, "recoveryLiveChildEnvironment: missing explicit "+key)
		}
	}
	return violations
}

func TestRecoveryLiveEnvironmentSourceGuard(t *testing.T) {
	sources := map[string]string{}
	for _, name := range []string{"recovery_conformance_live_test.go", "park_lifecycle_live_test.go", "recovery_live_environment_test.go"} {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		sources[name] = string(raw)
	}
	if violations := recoveryLiveEnvironmentViolations(sources); len(violations) != 0 {
		t.Fatalf("live isolation source guard: %v", violations)
	}
	for _, mutation := range []struct{ name, file, old, replacement string }{
		{"warm setup", "recovery_conformance_live_test.go", "setupRecoveryLiveEnvironment(t)", "_ = t"},
		{"park setup", "park_lifecycle_live_test.go", "setupRecoveryLiveEnvironment(t)", "_ = t"},
		{"nested handoff", "park_lifecycle_live_test.go", "cmd.Env = recoveryLiveChildEnvironment(t,", "cmd.Env = omittedEnvironment(t,"},
		{"runtime root", "recovery_live_environment_test.go", `os.Getenv("XDG_RUNTIME_DIR")`, `os.Getenv("OMITTED_RUNTIME")`},
		{"socket root", "recovery_live_environment_test.go", `os.Getenv("ZELLIJ_SOCKET_DIR")`, `os.Getenv("OMITTED_SOCKET")`},
		{"config root", "recovery_live_environment_test.go", `os.Getenv("ZELLIJ_CONFIG_DIR")`, `os.Getenv("OMITTED_CONFIG")`},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			changed := map[string]string{}
			for name, source := range sources {
				changed[name] = source
			}
			source := changed[mutation.file]
			start := 0
			if mutation.file == "recovery_live_environment_test.go" {
				// Mutate the actual builder, excluding the mutation-case strings.
				start = strings.Index(source, "func recoveryLiveChildEnvironment(")
				if start < 0 {
					t.Fatal("child environment builder missing")
				}
			}
			changed[mutation.file] = source[:start] + strings.Replace(source[start:], mutation.old, mutation.replacement, 1)
			if changed[mutation.file] == sources[mutation.file] {
				t.Fatal("source mutation did not apply")
			}
			if violations := recoveryLiveEnvironmentViolations(changed); len(violations) == 0 {
				t.Fatal("missing isolation boundary escaped source guard")
			}
		})
	}
}

func recoveryLiveChildEnvironment(t *testing.T, explicit ...string) []string {
	t.Helper()
	base := []string{"TMPDIR=/tmp", "SHELL=/bin/sh", "TERM=xterm-256color",
		"XDG_CONFIG_HOME=" + os.Getenv("XDG_CONFIG_HOME"),
		"XDG_DATA_HOME=" + os.Getenv("XDG_DATA_HOME"),
		"XDG_CACHE_HOME=" + os.Getenv("XDG_CACHE_HOME"),
		"XDG_RUNTIME_DIR=" + os.Getenv("XDG_RUNTIME_DIR"),
		"ZELLIJ_SOCKET_DIR=" + os.Getenv("ZELLIJ_SOCKET_DIR"),
		"ZELLIJ_CONFIG_DIR=" + os.Getenv("ZELLIJ_CONFIG_DIR")}
	return testpair.Environment(t, os.Getenv("HOME"), append(base, explicit...)...)
}

func TestRecoveryLiveEnvironmentOwnsRoots(t *testing.T) {
	ambient := t.TempDir()
	for _, key := range recoveryLiveRootKeys {
		t.Setenv(key, filepath.Join(ambient, key))
	}
	var home, sockets string
	t.Run("fixture", func(t *testing.T) {
		setupRecoveryLiveEnvironment(t)
		home, sockets = os.Getenv("HOME"), os.Getenv("ZELLIJ_SOCKET_DIR")
		if got, want := os.Getenv("ZELLIJ_CONFIG_DIR"), filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "zellij"); got != want {
			t.Fatalf("changed default config location: %q want %q", got, want)
		}
		entries, err := os.ReadDir(os.Getenv("ZELLIJ_CONFIG_DIR"))
		if err != nil || len(entries) != 0 {
			t.Fatalf("fixture must not install config or suppress setup wizard: entries=%v err=%v", entries, err)
		}
		// Exercise the longest current fixture name against the local Unix
		// socket API, including the version directory and macOS /private alias.
		resolved, err := filepath.EvalSymlinks(sockets)
		if err != nil {
			t.Fatal(err)
		}
		versionDir := filepath.Join(resolved, "0.45.1")
		if err := os.Mkdir(versionDir, 0700); err != nil {
			t.Fatal(err)
		}
		socketPath := filepath.Join(versionDir, "pair-recovery-"+strings.Repeat("f", 20))
		if len(socketPath) > 103 {
			t.Fatalf("fixture exceeds conservative Unix socket budget: bytes=%d path=%q", len(socketPath), socketPath)
		}
		listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
		if err != nil {
			t.Fatal(err)
		}
		if err := listener.Close(); err != nil {
			t.Fatal(err)
		}
		for _, child := range []struct {
			name string
			env  []string
		}{
			{"direct", os.Environ()},
			{"nested helper", recoveryLiveChildEnvironment(t)},
		} {
			// A harmless child reports the effective environment seen by native
			// commands; no installed Zellij or operator configuration is opened.
			cmd := exec.Command("/bin/sh", "-c", `printf '%s\000' "$HOME" "$XDG_CONFIG_HOME" "$XDG_DATA_HOME" "$XDG_CACHE_HOME" "$XDG_RUNTIME_DIR" "$ZELLIJ_SOCKET_DIR" "$ZELLIJ_CONFIG_DIR" "$TMPDIR" "$SHELL"`)
			cmd.Env = child.env
			raw, err := cmd.Output()
			if err != nil {
				t.Fatal(err)
			}
			values := strings.Split(strings.TrimSuffix(string(raw), "\x00"), "\x00")
			if len(values) != len(recoveryLiveRootKeys)+2 {
				t.Fatalf("child environment=%q", raw)
			}
			for i, key := range recoveryLiveRootKeys {
				value := values[i]
				if value == "" || strings.HasPrefix(value, ambient+string(os.PathSeparator)) || value != os.Getenv(key) {
					t.Errorf("%s child %s=%q does not preserve explicit private root %q", child.name, key, value, os.Getenv(key))
					continue
				}
				if info, err := os.Stat(value); err != nil || !info.IsDir() {
					t.Errorf("%s child %s root is not a directory: %v", child.name, key, err)
				}
			}
			if values[len(recoveryLiveRootKeys)] != "/tmp" || values[len(recoveryLiveRootKeys)+1] != "/bin/sh" {
				t.Errorf("changed TMPDIR/SHELL: %q", values)
			}
		}
	})
	if sockets != "" {
		if _, err := os.Stat(filepath.Dir(sockets)); !os.IsNotExist(err) {
			t.Errorf("owned short socket root survived fixture cleanup: %v", err)
		}
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Errorf("owned HOME survived fixture cleanup: %v", err)
	}
	for _, key := range recoveryLiveRootKeys {
		if got := os.Getenv(key); got != filepath.Join(ambient, key) {
			t.Errorf("ambient %s not restored: %q", key, got)
		}
	}
}
