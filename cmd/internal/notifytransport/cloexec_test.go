package notifytransport

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

func TestNamespaceLockCloseOnExec(t *testing.T) {
	root := filepath.Join(t.TempDir(), "namespace")
	unlock, err := lockNamespace(root)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	target, err := os.Stat(filepath.Join(root, "lock"))
	if err != nil {
		t.Fatal(err)
	}
	// The lock's descriptor is deliberately encapsulated. Inspect the live
	// kernel table by inode rather than adding a production seam just for a test.
	want := target.Sys().(*syscall.Stat_t)
	var limit unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &limit); err != nil {
		t.Fatal(err)
	}
	// macOS /dev/fd directory entries do not enumerate all live descriptors.
	// This isolated, nonparallel test opens one lock; bound the kernel scan.
	for fd := 0; fd < int(min(limit.Cur, 4096)); fd++ {
		var st unix.Stat_t
		if unix.Fstat(fd, &st) != nil || uint64(st.Dev) != uint64(want.Dev) || uint64(st.Ino) != uint64(want.Ino) {
			continue
		}
		flags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0)
		if err != nil {
			t.Fatal(err)
		}
		if flags&unix.FD_CLOEXEC == 0 {
			t.Error("held namespace lock descriptor can survive exec")
		}
		return
	}
	t.Fatal("namespace lock descriptor not found within the bounded kernel scan")
}

// A post-open fcntl would pass a final FD flag check but still race fork/exec.
// Require the atomic flag at every raw open in the complete two-module family,
// including short-lived readPID/directoryNames descriptors hidden from callers.
func TestRawOpensAreAtomicCloseOnExec(t *testing.T) {
	for _, dir := range []string{".", "../diagnosticlog"} {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		found := 0
		for _, path := range files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			fs := token.NewFileSet()
			file, err := parser.ParseFile(fs, path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			packages := map[string]bool{}
			for _, im := range file.Imports {
				importPath, err := strconv.Unquote(im.Path.Value)
				if err != nil {
					t.Fatal(err)
				}
				if importPath != "syscall" && importPath != "golang.org/x/sys/unix" {
					continue
				}
				name := filepath.Base(importPath)
				if im.Name != nil {
					name = im.Name.Name
				}
				packages[name] = true
			}
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok || !packages[pkg.Name] {
					return true
				}
				index := 1
				switch sel.Sel.Name {
				case "Open":
				case "Openat":
					index = 2
				case "Creat":
					index = -1
				default:
					return true
				}
				found++
				if index < 0 || len(call.Args) <= index || !atomicCloseOnExecFlag(call.Args[index], packages) {
					t.Errorf("%s: raw %s must include O_CLOEXEC at open time", fs.Position(call.Pos()), sel.Sel.Name)
				}
				return true
			})
		}
		if found == 0 {
			t.Errorf("no raw opens inspected in %s", dir)
		}
	}
}

func atomicCloseOnExecFlag(expr ast.Expr, packages map[string]bool) bool {
	switch e := expr.(type) {
	case *ast.ParenExpr:
		return atomicCloseOnExecFlag(e.X, packages)
	case *ast.BinaryExpr:
		return e.Op == token.OR && (atomicCloseOnExecFlag(e.X, packages) || atomicCloseOnExecFlag(e.Y, packages))
	case *ast.SelectorExpr:
		pkg, ok := e.X.(*ast.Ident)
		return ok && packages[pkg.Name] && e.Sel.Name == "O_CLOEXEC"
	}
	return false
}
