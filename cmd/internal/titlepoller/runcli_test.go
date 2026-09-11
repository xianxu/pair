package titlepoller

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/contextcmd"
)

// The contract is tied to the reads, not to a list. Every Pair-session variable
// the poller reads -- OBSERVED through contextcmd.EnvFrom and the CLI parse --
// must be one SessionEnv supplies, so a new read fails here until the contract
// carries it; NewSessionEnv then gains a parameter, and both launcher call sites
// stop compiling until they fill it (pair#183). PAIR_ is the repo's prefix for
// session variables; the rest of what the poller reads (HOME, XDG_DATA_HOME,
// CMUX_WORKSPACE_ID) belongs to the operator's terminal and is inherited.
func TestSessionEnvSuppliesEveryPairVariableThePollerReads(t *testing.T) {
	const dataDir, scopeKey = "/data/repos/k", "k"
	supplied := map[string]string{}
	for _, kv := range NewSessionEnv(dataDir, scopeKey).Environ() {
		name, value, _ := strings.Cut(kv, "=")
		supplied[name] = value
	}
	var read []string
	getenv := func(name string) string {
		read = append(read, name)
		return supplied[name]
	}

	context := contextcmd.EnvFrom(getenv)
	opts, ok := optionsFromCLI([]string{"T", "claude", "📁repo-T"}, getenv)
	if !ok {
		t.Fatal("optionsFromCLI refused a complete argv")
	}

	for _, name := range read {
		if strings.HasPrefix(name, "PAIR_") && supplied[name] == "" {
			t.Errorf("the poller reads %s but SessionEnv does not supply it", name)
		}
	}
	// The round trip, so a supplied-but-misnamed variable cannot pass.
	if context.PairDataDir != dataDir || context.PairScopeKey != scopeKey || opts.DataDir != dataDir {
		t.Fatalf("round trip lost the contract: context=%+v opts.DataDir=%q", context, opts.DataDir)
	}
}

// The parse's refusal: a short argv starts no poller, and RunCLI exits 0
// without a word -- the same silent class as pair#183, so it is pinned rather
// than assumed unchanged by the extraction (plan gate PQ-1).
func TestOptionsFromCLIRefusesAnArgvWithoutTagAndAgent(t *testing.T) {
	getenv := func(string) string { return "" }
	for _, args := range [][]string{nil, {"T"}} {
		if _, ok := optionsFromCLI(args, getenv); ok {
			t.Errorf("optionsFromCLI(%q) accepted an argv with no agent", args)
		}
	}
}
