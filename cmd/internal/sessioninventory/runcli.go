package sessioninventory

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// SessionInventoryCLI is the public stable diagnostic surface.
// pair:155-concept integration new M2 session-inventory
type SessionInventoryCLI struct{}

var supportedAgents = []Agent{AgentAgy, AgentClaude, AgentCodex, AgentMuse}

// RunCLI resolves the production runtime and emits one buffered result.
func RunCLI(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	options, usage := parseCLIOptions(args)
	if usage != "" {
		_, _ = fmt.Fprintln(stderr, usage)
		return 1
	}
	options.currentScopeKey = getenv("PAIR_SCOPE_KEY")
	dataDir := cliPairDataDir(getenv, options.scope)
	home := getenv("HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "pair session-inventory: cannot resolve home")
			return 2
		}
	}
	return runCLIOptions(options, NewOSRuntime(home, dataDir), stdout, stderr)
}

// RunCLIWithRuntime drives the exact production decision over an injected
// native/Pair runtime. Tests use it to exercise partial and failing state.
func RunCLIWithRuntime(args []string, getenv func(string) string, runtime Runtime, stdout, stderr io.Writer) int {
	options, usage := parseCLIOptions(args)
	if usage != "" {
		_, _ = fmt.Fprintln(stderr, usage)
		return 1
	}
	options.currentScopeKey = getenv("PAIR_SCOPE_KEY")
	return runCLIOptions(options, runtime, stdout, stderr)
}

type cliOptions struct {
	agents          []Agent
	scope           string
	json            bool
	conformance     bool
	currentScopeKey string
}

func parseCLIOptions(args []string) (cliOptions, string) {
	flags := flag.NewFlagSet("session-inventory", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	agentName := flags.String("agent", "", "agent")
	scope := flags.String("scope", "current", "scope")
	jsonOutput := flags.Bool("json", false, "JSON output")
	conformance := flags.Bool("conformance", false, "redacted conformance")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return cliOptions{}, "usage: pair session-inventory [--agent claude|codex|agy|muse] [--scope current|all] [--json] [--conformance]"
	}
	if *scope != "current" && *scope != "all" {
		return cliOptions{}, fmt.Sprintf("pair session-inventory: unsupported scope %q", *scope)
	}
	agents := append([]Agent(nil), supportedAgents...)
	if *agentName != "" {
		agent := Agent(*agentName)
		if !validAgent(agent) {
			return cliOptions{}, fmt.Sprintf("pair session-inventory: unsupported agent %q", *agentName)
		}
		agents = []Agent{agent}
	}
	return cliOptions{agents: agents, scope: *scope, json: *jsonOutput, conformance: *conformance}, ""
}

func runCLIOptions(options cliOptions, runtime Runtime, stdout, stderr io.Writer) int {
	return runCLIOptionsWithRenderers(options, runtime, stdout, stderr, defaultCLIRenderers())
}

type cliRenderers struct {
	inventory   func(Inventory, RenderFormat) ([]byte, error)
	conformance func(ConformanceReport) ([]byte, error)
}

func defaultCLIRenderers() cliRenderers {
	return cliRenderers{inventory: RenderV1, conformance: RenderConformance}
}

func runCLIOptionsWithRenderers(options cliOptions, runtime Runtime, stdout, stderr io.Writer, renderers cliRenderers) int {
	if options.conformance {
		report, conformanceErr := RunConformance(runtime, options.agents...)
		rendered, renderErr := renderers.conformance(report)
		if renderErr != nil {
			_, _ = fmt.Fprintln(stderr, "pair session-inventory: render failed")
			return 2
		}
		if !canonicalConformanceRendering(rendered) {
			_, _ = fmt.Fprintln(stderr, "pair session-inventory: conformance privacy check failed")
			return 2
		}
		if err := writeBuffered(stdout, rendered); err != nil {
			_, _ = fmt.Fprintln(stderr, "pair session-inventory: render write failed")
			return 2
		}
		if conformanceErr != nil {
			_, _ = fmt.Fprintln(stderr, "pair session-inventory: conformance failed: schema_near_miss or storage_unreadable")
			return 2
		}
		return 0
	}

	scanners := make([]Scanner, 0, len(options.agents))
	for _, agent := range options.agents {
		scanners = append(scanners, ScannerForAgent(agent))
	}
	inventory := InventoryWithRuntime(runtime, scanners...)
	if everyRequestedScannerFatal(inventory, options.agents) {
		_, _ = fmt.Fprintln(stderr, "pair session-inventory: scan failed: storage_unreadable")
		return 2
	}
	var err error
	inventory, err = RecoverPairBindings(runtime, inventory, options.scope, options.currentScopeKey, options.agents)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "pair session-inventory: Pair artifacts unavailable")
		return 2
	}
	format := RenderHuman
	if options.json {
		format = RenderJSON
	}
	rendered, err := renderers.inventory(inventory, format)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "pair session-inventory: render failed")
		return 2
	}
	if err := writeBuffered(stdout, rendered); err != nil {
		_, _ = fmt.Fprintln(stderr, "pair session-inventory: render write failed")
		return 2
	}
	return 0
}

func canonicalConformanceRendering(rendered []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(rendered))
	decoder.DisallowUnknownFields()
	var report ConformanceReport
	if err := decoder.Decode(&report); err != nil {
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return false
	}
	canonical, err := RenderConformance(report)
	return err == nil && bytes.Equal(rendered, canonical)
}

func everyRequestedScannerFatal(inventory Inventory, agents []Agent) bool {
	withForest := map[Agent]bool{}
	for _, forest := range inventory.Forests {
		withForest[forest.Agent] = true
	}
	fatal := map[Agent]bool{}
	for _, diagnostic := range inventory.Diagnostics {
		if diagnostic.Code == DiagnosticStorageUnreadable {
			fatal[diagnostic.Agent] = true
		}
	}
	for _, agent := range agents {
		if withForest[agent] || !fatal[agent] {
			return false
		}
	}
	return len(agents) != 0
}

func writeBuffered(writer io.Writer, data []byte) error {
	written, err := writer.Write(data)
	if err == nil && written != len(data) {
		err = io.ErrShortWrite
	}
	return err
}

func cliPairDataDir(getenv func(string) string, scope string) string {
	dataDir := getenv("PAIR_DATA_DIR")
	if dataDir == "" {
		base := getenv("XDG_DATA_HOME")
		if base == "" {
			base = filepath.Join(getenv("HOME"), ".local", "share")
		}
		dataDir = filepath.Join(base, "pair")
	}
	dataDir = filepath.Clean(dataDir)
	if scope == "all" && filepath.Base(filepath.Dir(dataDir)) == "repos" {
		return filepath.Dir(filepath.Dir(dataDir))
	}
	return dataDir
}
