package dispatcher

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/xianxu/pair/cmd/internal/clipcmd"
	"github.com/xianxu/pair/cmd/internal/contextcmd"
	"github.com/xianxu/pair/cmd/internal/opener"
	"github.com/xianxu/pair/cmd/internal/reviewcmd"
	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
	"github.com/xianxu/pair/cmd/internal/slugcmd"
)

const programName = "pair-go"

// CommandFamily names a Pair CLI surface. Name is a space-separated command
// path: flat (e.g. "context") or one level of grouping (e.g. "review open").
// Status is the single source of implemented-ness ("implemented" | "planned" |
// "handoff" | "prototype"); Streaming is the orthogonal axis: a subcommand that
// needs real stdin, a live stdout/stderr consumer, or a long lifetime is routed
// through cmd/pair-go's streaming seam instead of the buffered Dispatch path.
// Alias marks a transitional flat name kept only until its callers migrate to
// the nested form (removed in #104 M3); aliases are hidden from Help.
type CommandFamily struct {
	Name      string
	Summary   string
	Status    string
	Streaming bool
	Alias     bool
}

// Result is the process-facing outcome of a pure dispatch decision.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Families returns the command families for the Go dispatcher — the single
// source of truth for the subcommand surface (#104). Related helpers nest under
// a group (review/scrollback/changelog/clip); the rest stay flat.
func Families() []CommandFamily {
	return []CommandFamily{
		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "handoff"},
		// flat helpers
		{Name: "context", Summary: "agent pane context meter", Status: "implemented"},
		{Name: "slug", Summary: "session orientation slug generation", Status: "implemented"},
		{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "implemented", Streaming: true},
		{Name: "scribe", Summary: "PTY logging wrapper", Status: "implemented", Streaming: true},
		{Name: "session-watch", Summary: "async codex/agy session-id discovery", Status: "implemented", Streaming: true},
		{Name: "title", Summary: "agent pane title poller", Status: "implemented", Streaming: true},
		{Name: "continuation", Summary: "continuation datatype writer", Status: "implemented", Streaming: true},
		// review group
		{Name: "review target", Summary: "record the review-target pane", Status: "implemented"},
		{Name: "review open", Summary: "open/refresh the review pane", Status: "implemented"},
		{Name: "review readiness", Summary: "prepare review-readiness data", Status: "implemented"},
		// scrollback group
		{Name: "scrollback render", Summary: "raw PTY capture to ANSI scrollback", Status: "implemented"},
		{Name: "scrollback open", Summary: "open the scrollback viewer pane", Status: "implemented"},
		// changelog group
		{Name: "changelog render", Summary: "TTY transcript to distilled change log", Status: "implemented", Streaming: true},
		{Name: "changelog open", Summary: "open the changelog viewer pane", Status: "implemented"},
		// clip group
		{Name: "clip copy-on-select", Summary: "copy selection to the clipboard", Status: "implemented", Streaming: true},
		{Name: "clip clipboard-to-pane", Summary: "paste the clipboard into a pane", Status: "implemented"},
		{Name: "clip flash-pane", Summary: "flash a pane border", Status: "implemented"},
	}
}

// DispatchNames returns the top-level tokens the public `pair` entrypoint peels
// off before the launcher (Status == "implemented"). For a nested family the
// token is the group (e.g. "review" for "review open"); tokens are deduped, so
// this is one source (Families), no drift.
func DispatchNames() []string {
	var names []string
	seen := map[string]bool{}
	for _, f := range Families() {
		if f.Status != "implemented" {
			continue
		}
		tok := f.Name
		if i := strings.IndexByte(tok, ' '); i >= 0 {
			tok = tok[:i]
		}
		if !seen[tok] {
			seen[tok] = true
			names = append(names, tok)
		}
	}
	return names
}

// Resolve matches argv against the families, honoring one level of grouping: a
// two-token "group leaf" match wins over a one-token flat match. It returns the
// matched family and the args remaining after the command tokens.
func Resolve(args []string) (CommandFamily, []string, bool) {
	if len(args) == 0 {
		return CommandFamily{}, nil, false
	}
	if len(args) >= 2 {
		if f, ok := familyByName(args[0] + " " + args[1]); ok {
			return f, args[2:], true
		}
	}
	if f, ok := familyByName(args[0]); ok {
		return f, args[1:], true
	}
	return CommandFamily{}, nil, false
}

// IsStreaming reports whether a subcommand needs the streaming seam.
func IsStreaming(name string) bool {
	for _, f := range Families() {
		if f.Name == name {
			return f.Streaming
		}
	}
	return false
}

// Dispatch parses argv and returns the skeleton dispatch result.
func Dispatch(args []string) Result {
	if len(args) == 0 {
		return Result{Stdout: Help(programName), ExitCode: 0}
	}

	switch args[0] {
	case "help", "--help", "-h":
		return Result{Stdout: Help(programName), ExitCode: 0}
	case "version", "--version":
		return Result{
			Stdout:   "pair-go dispatcher skeleton\nlaunch handoff: bin/pair\n",
			ExitCode: 0,
		}
	case "launch":
		return launchHandoffResult()
	}

	family, rest, ok := Resolve(args)
	if !ok {
		return Result{
			Stderr:   fmt.Sprintf("%s: unknown command %q; run %s help\n", programName, args[0], programName),
			ExitCode: 2,
		}
	}
	if family.Status != "implemented" {
		return Result{
			Stderr:   fmt.Sprintf("%s: %s is planned but not implemented in this skeleton; run %s help\n", programName, family.Name, programName),
			ExitCode: 2,
		}
	}
	if family.Streaming {
		// Streaming subcommands (wrap/scribe/changelog render/continuation/
		// session-watch/title/clip copy-on-select) are intercepted by cmd/pair-go's
		// streaming seam before Dispatch; reaching here is a buffered-path mistake.
		return Result{
			Stderr:   fmt.Sprintf("%s: %s is a streaming subcommand; invoke it via cmd/pair-go's streaming seam, not the buffered Dispatch\n", programName, family.Name),
			ExitCode: 2,
		}
	}

	switch family.Name {
	case "context":
		return dispatchContext(rest)
	case "slug":
		return dispatchSlug(rest)
	case "scrollback render":
		return dispatchScrollbackRender(rest)
	case "scrollback open":
		return bufferedStderr(func(stderr *bytes.Buffer) int { return opener.RunScrollbackCLI(rest, os.Getenv, stderr) })
	case "changelog open":
		return bufferedStderr(func(stderr *bytes.Buffer) int { return opener.RunChangelogCLI(rest, os.Getenv, stderr) })
	case "review target":
		return bufferedStdoutStderr(func(stdout, stderr *bytes.Buffer) int { return reviewcmd.RunTargetCLI(rest, os.Getenv, stdout, stderr) })
	case "review open":
		return bufferedStderr(func(stderr *bytes.Buffer) int { return reviewcmd.RunOpenCLI(rest, os.Getenv, stderr) })
	case "review readiness":
		return bufferedStdoutStderr(func(stdout, stderr *bytes.Buffer) int { return reviewcmd.RunReadinessCLI(rest, os.Getenv, stdout, stderr) })
	case "clip clipboard-to-pane":
		return bufferedStderr(func(stderr *bytes.Buffer) int { return clipcmd.RunClipboardToPaneCLI(os.Getenv, stderr) })
	case "clip flash-pane":
		return bufferedStderr(func(stderr *bytes.Buffer) int { return clipcmd.RunFlashPaneCLI(rest, os.Getenv, stderr) })
	}

	return Result{
		Stderr:   fmt.Sprintf("%s: %s has no buffered route wired\n", programName, family.Name),
		ExitCode: 2,
	}
}

func dispatchContext(args []string) Result {
	var stdout bytes.Buffer
	code := contextcmd.Run(args, contextcmd.EnvFromOS(), &stdout)
	return Result{Stdout: stdout.String(), ExitCode: code}
}

func dispatchScrollbackRender(args []string) Result {
	var stdout, stderr bytes.Buffer
	code := scrollbackcmd.Run(args, &stdout, &stderr)
	return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}
}

// bufferedStderr wraps a RunXxxCLI that writes user-facing UI directly to the
// real os stdio (e.g. an interactive nvim viewer, opener/runtime.go:151-153) and
// uses the injected stderr only for its own error text. The buffered Result
// carries that error text + the exit code.
func bufferedStderr(run func(*bytes.Buffer) int) Result {
	var stderr bytes.Buffer
	code := run(&stderr)
	return Result{Stderr: stderr.String(), ExitCode: code}
}

func bufferedStdoutStderr(run func(*bytes.Buffer, *bytes.Buffer) int) Result {
	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr)
	return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}
}

// dispatchSlug routes `pair slug`. slug is env-driven with no args and writes
// only to files + $PAIR_SLUG_LOG (no stdout/stderr), so the buffered Result is
// empty; only the exit code carries. slug.Run always returns 0 (tolerant).
func dispatchSlug([]string) Result {
	return Result{ExitCode: slugcmd.Run()}
}

func launchHandoffResult() Result {
	return Result{
		Stderr:   "pair-go launch is a process handoff implemented by cmd/pair-go; call pair-go launch ... instead of dispatcher.Dispatch\n",
		ExitCode: 2,
	}
}

// Help renders the development-only dispatcher usage text.
func Help(program string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Usage: %s <command> [args]\n\n", program)
	b.WriteString("Development dispatcher skeleton. Public sessions still start through bin/pair.\n\n")
	b.WriteString("Implemented commands:\n")
	for _, family := range Families() {
		if family.Alias {
			continue
		}
		if family.Status == "prototype" {
			fmt.Fprintf(&b, "  %-17s %s (prototype; decision-phase only)\n", family.Name, family.Summary)
		} else if family.Status == "handoff" {
			fmt.Fprintf(&b, "  %-17s %s (compatibility handoff to bin/pair)\n", family.Name, family.Summary)
		} else if family.Status == "implemented" {
			fmt.Fprintf(&b, "  %-17s %s (implemented helper route)\n", family.Name, family.Summary)
		}
	}
	b.WriteString("\nPlanned command families (not implemented in this skeleton):\n")
	for _, family := range Families() {
		if family.Status == "planned" {
			fmt.Fprintf(&b, "  %-17s %s (%s; not implemented in this skeleton)\n", family.Name, family.Summary, family.Status)
		}
	}
	b.WriteString("\nSupported skeleton commands:\n")
	b.WriteString("  help              show this help\n")
	b.WriteString("  version           show dispatcher skeleton metadata\n")
	return b.String()
}

func familyByName(name string) (CommandFamily, bool) {
	for _, family := range Families() {
		if family.Name == name {
			return family, true
		}
	}
	return CommandFamily{}, false
}
