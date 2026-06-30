package dispatcher

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/contextcmd"
	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
)

const programName = "pair-go"

// CommandFamily names a future Pair CLI surface without claiming it works yet.
type CommandFamily struct {
	Name    string
	Summary string
	Status  string
}

// Result is the process-facing outcome of a pure dispatch decision.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Families returns the planned command families for the Go dispatcher.
func Families() []CommandFamily {
	return []CommandFamily{
		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "handoff"},
		{Name: "context", Summary: "agent pane context meter", Status: "implemented"},
		{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "implemented"},
		{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
		{Name: "slug", Summary: "session orientation slug generation", Status: "planned"},
		{Name: "changelog", Summary: "TTY transcript to distilled change log", Status: "planned"},
		{Name: "continuation", Summary: "continuation datatype writer", Status: "planned"},
		{Name: "scribe", Summary: "PTY logging wrapper", Status: "planned"},
	}
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
	case "context":
		return dispatchContext(args[1:])
	case "scrollback-render":
		return dispatchScrollbackRender(args[1:])
	}

	if family, ok := familyByName(args[0]); ok {
		return Result{
			Stderr:   fmt.Sprintf("%s: %s is planned but not implemented in this skeleton; run %s help\n", programName, family.Name, programName),
			ExitCode: 2,
		}
	}

	return Result{
		Stderr:   fmt.Sprintf("%s: unknown command %q; run %s help\n", programName, args[0], programName),
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
