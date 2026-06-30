package dispatcher

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
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
		{Name: "launch", Summary: "session lifecycle and public pair launcher flow", Status: "prototype"},
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
	case "launch":
		return DispatchWithLauncherRuntime(args, osLauncherRuntime())
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

type LauncherRuntime struct {
	Env      launcher.Env
	Sessions launcher.SessionSource
	History  launcher.HistoricalScanner
}

type StaticSessions struct {
	Sessions []launcher.Session
	Err      error
}

func (s StaticSessions) Snapshot() ([]launcher.Session, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	return s.Sessions, nil
}

type StaticHistory struct {
	Tags []launcher.HistoricalTag
	Err  error
}

func (h StaticHistory) Scan(_ string, _ time.Time) ([]launcher.HistoricalTag, error) {
	if h.Err != nil {
		return nil, h.Err
	}
	return h.Tags, nil
}

func DispatchWithLauncherRuntime(args []string, rt LauncherRuntime) Result {
	launchArgs := []string(nil)
	if len(args) > 1 {
		launchArgs = args[1:]
	}
	if len(launchArgs) > 0 && (launchArgs[0] == "help" || launchArgs[0] == "--help" || launchArgs[0] == "-h") {
		return Result{Stdout: LaunchHelp(programName), ExitCode: 0}
	}
	outcome, err := launcher.Run(launchArgs, rt.Env, rt.Sessions, rt.History)
	if err != nil {
		return Result{Stderr: fmt.Sprintf("pair-go launch: %v\n", err), ExitCode: 2}
	}
	decision := outcome.Decision
	return Result{
		Stderr: fmt.Sprintf(
			"pair-go launch: prototype decision action=%s tag=%s session=%s; real zellij launch remains shell-owned\n",
			decision.Action,
			decision.Tag,
			decision.SessionName,
		),
		ExitCode: 3,
	}
}

func LaunchHelp(program string) string {
	return fmt.Sprintf(`Usage: %s launch [agent] [-- agent-args...]
       %s launch resume <tag>

Guarded decision-phase prototype. Public sessions still start through bin/pair.
This command parses launch inputs and computes the create/attach/picker decision,
then stops before invoking zellij.
`, program, program)
}

func LauncherEnv(home, xdgDataHome, cwd string) launcher.Env {
	return launcher.Env{
		Home:     home,
		XDGData:  xdgDataHome,
		Cwd:      cwd,
		Now:      time.Now(),
		HistoryD: 14,
	}
}

func osLauncherRuntime() LauncherRuntime {
	home := os.Getenv("HOME")
	xdg := os.Getenv("XDG_DATA_HOME")
	cwd, _ := os.Getwd()
	env := LauncherEnv(home, xdg, cwd)
	dataDir := launcher.ResolveDataDir(home, xdg)
	return LauncherRuntime{
		Env:      env,
		Sessions: launcher.ZellijSource{},
		History:  launcher.HistorySource{DataDir: dataDir},
	}
}

// Help renders the development-only dispatcher usage text.
func Help(program string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Usage: %s <command> [args]\n\n", program)
	b.WriteString("Development dispatcher skeleton. Public sessions still start through bin/pair.\n\n")
	b.WriteString("Implemented prototype commands:\n")
	for _, family := range Families() {
		if family.Status == "prototype" {
			fmt.Fprintf(&b, "  %-17s %s (prototype; decision-phase only)\n", family.Name, family.Summary)
		}
	}
	b.WriteString("\nPlanned command families (not implemented in this skeleton):\n")
	for _, family := range Families() {
		if family.Status != "prototype" {
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
