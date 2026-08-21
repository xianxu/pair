// Package couchcmd is couch's CLI surface.
//
// It follows the house convention: one exported Run per command package with
// the signature Run(args, stdin, stdout, stderr) int, errors printed to the
// injected stderr rather than returned, and no process globals touched inside
// the package (env arrives through Runtime). Compare termcmd/run.go:46-50.
package couchcmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

// Runtime is the seam for everything ambient: env lookup and where the store
// lives. Tests supply their own so they never read the developer's real
// ~/.local/share/pair.
type Runtime interface {
	Getenv(string) string
	StoreDir() string
}

type OSRuntime struct{}

var _ Runtime = OSRuntime{}

func (OSRuntime) Getenv(k string) string { return os.Getenv(k) }

func (r OSRuntime) StoreDir() string {
	if dir := r.Getenv("COUCH_STORE_DIR"); dir != "" {
		return dir
	}
	// Unscoped on purpose: the registry spans every worktree, so a per-tree
	// scope directory would mean spawning in /a and listing from /b read
	// different files.
	return filepath.Join(launcher.ResolveDataDir(r.Getenv("HOME"), r.Getenv("XDG_DATA_HOME")), "couch")
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return RunWithRuntime(args, stdin, stdout, stderr, OSRuntime{})
}

// Dispatch is the operation table, built FROM couchcore.Operations(). There is
// deliberately no argv switch: the audit asserts this table's key set is
// identical to the declared operation set, so an operation reachable here but
// never declared cannot exist.
func Dispatch() map[string]couchcore.Operation {
	out := map[string]couchcore.Operation{}
	for _, op := range couchcore.Operations() {
		out[op.Name] = op
	}
	return out
}

func RunWithRuntime(args []string, stdin io.Reader, stdout, stderr io.Writer, rt Runtime) int {
	table := Dispatch()
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		usage(stdout, table)
		return 0
	}
	op, ok := table[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "couch: unknown operation %q\n\n", args[0])
		usage(stderr, table)
		return 2
	}

	parsed, err := bindArgs(op, args[1:])
	if err != nil {
		fmt.Fprintf(stderr, "couch %s: %v\n", op.Name, err)
		return 2
	}

	c, err := couchcore.New(
		couchcore.ExecRunner{}, couchcore.OSPathOps{}, couchcore.ExecGit{},
		couchcore.OSProcOps{}, couchcore.NewStore(rt.StoreDir()),
		couchcore.SystemClock{}, couchcore.NewRandomIDGen(),
	)
	if err != nil {
		fmt.Fprintf(stderr, "couch: %v\n", err)
		return 1
	}

	result, err := op.Invoke(c, parsed)
	if err != nil {
		renderError(stderr, err)
		return 1
	}
	return render(stdout, op, result)
}

// bindArgs maps positional argv onto the operation's declared ArgSpecs, plus
// --flag=value form for the optional ones.
func bindArgs(op couchcore.Operation, argv []string) (map[string]string, error) {
	out := map[string]string{}
	var positional []string
	for _, a := range argv {
		if strings.HasPrefix(a, "--") {
			name, value, found := strings.Cut(strings.TrimPrefix(a, "--"), "=")
			if !found {
				value = "true"
			}
			out[name] = value
			continue
		}
		positional = append(positional, a)
	}
	i := 0
	for _, spec := range op.Args {
		if _, already := out[spec.Name]; already {
			continue
		}
		if i < len(positional) {
			out[spec.Name] = positional[i]
			i++
			continue
		}
		if spec.Required {
			return nil, fmt.Errorf("missing required argument %q", spec.Name)
		}
	}
	return out, nil
}

func render(w io.Writer, op couchcore.Operation, result any) int {
	switch v := result.(type) {
	case couchcore.StartResult:
		fmt.Fprintf(w, "started %s on %s (pid %d)\n", v.Record.ID, v.Record.Args.Worktree, v.Record.PID)
		if v.Handle != nil {
			// couch start blocks for the child's lifetime: this milestone has
			// no pty, so the child owns the terminal until it exits (#146).
			return v.Handle.Wait()
		}
		return 0
	case []couchcore.ActorView:
		renderViews(w, v)
	case couchcore.Worktree:
		fmt.Fprintf(w, "%s\n", v)
	case couchcore.ActorRecord:
		fmt.Fprintf(w, "stopped %s on %s\n", v.ID, v.Args.Worktree)
	case string:
		if v == "" {
			fmt.Fprintln(w, "(no description)")
		} else {
			fmt.Fprintln(w, v)
		}
	default:
		fmt.Fprintf(w, "%v\n", v)
	}
	return 0
}

func renderViews(w io.Writer, views []couchcore.ActorView) {
	if len(views) == 0 {
		fmt.Fprintln(w, "no actors")
		return
	}
	for _, v := range views {
		state := "dead"
		if v.Live {
			state = "live"
		}
		label := v.Name
		if label == "" {
			label = v.Record.Args.Worktree.Repo()
		}
		fmt.Fprintf(w, "%-14s %-5s %-22s %s\n", v.Record.ID, state, label, v.Record.Args.Worktree)
		if v.Desc != "" {
			fmt.Fprintf(w, "%-14s       %s\n", "", v.Desc)
		}
	}
}

// renderError gives the tree-occupied refusal the shape the project asks for:
// a decision at the moment the operator has context, with the offer chosen by
// the repo's recorded concurrency policy.
func renderError(w io.Writer, err error) {
	var occ *couchcore.TreeOccupiedError
	if !asTreeOccupied(err, &occ) {
		fmt.Fprintf(w, "couch: %v\n", err)
		return
	}
	fmt.Fprintf(w, "%s already has an agent:\n", occ.Tree)
	for _, a := range occ.Incumbents {
		fmt.Fprintf(w, "  %s (pid %d)\n", a.ID, a.PID)
	}
	fmt.Fprintf(w, "They would share a branch and index.\n")
	switch occ.Mode {
	case couchcore.WorktreeParallel:
		fmt.Fprintf(w, "  -> new worktree (cheap here), or switch to it, or --same-tree\n")
	case couchcore.HeavyLocalState:
		fmt.Fprintf(w, "  -> switch to it, or --same-tree (worktrees are expensive in this repo)\n")
	default:
		fmt.Fprintf(w, "  -> switch to it, or --same-tree (this repo runs one agent at a time)\n")
	}
}

func usage(w io.Writer, table map[string]couchcore.Operation) {
	names := make([]string, 0, len(table))
	for n := range table {
		names = append(names, n)
	}
	sort.Strings(names)
	fmt.Fprintln(w, "couch - supervise agent actors, one per working tree")
	fmt.Fprintln(w)
	for _, n := range names {
		op := table[n]
		fmt.Fprintf(w, "  %-10s %s\n", op.Name, op.Summary)
		for _, a := range op.Args {
			req := ""
			if a.Required {
				req = " (required)"
			}
			fmt.Fprintf(w, "  %-10s   <%s>%s -- %s\n", "", a.Name, req, a.Summary)
		}
	}
}
