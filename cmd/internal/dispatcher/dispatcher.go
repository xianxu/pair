package dispatcher

import (
	"fmt"
	"strings"
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
		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "planned"},
		{Name: "wrap", Summary: "PTY proxy around a TUI agent", Status: "planned"},
		{Name: "slug", Summary: "session orientation slug generation", Status: "planned"},
		{Name: "context", Summary: "agent pane context meter", Status: "planned"},
		{Name: "scrollback-render", Summary: "raw PTY capture to ANSI scrollback", Status: "planned"},
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
			Stdout:   "pair-go dispatcher skeleton\npublic launcher: bin/pair\n",
			ExitCode: 0,
		}
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

// Help renders the development-only dispatcher usage text.
func Help(program string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Usage: %s <command> [args]\n\n", program)
	b.WriteString("Development dispatcher skeleton. Public sessions still start through bin/pair.\n\n")
	b.WriteString("Planned command families (not implemented in this skeleton):\n")
	for _, family := range Families() {
		fmt.Fprintf(&b, "  %-17s %s (%s; not implemented in this skeleton)\n", family.Name, family.Summary, family.Status)
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
