package launcher

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// codexResumeWithVersion runs the exact-root resume launch from
// TestRequiredNativeResumeBindingLaunchesExactRootWithoutDefaults with the
// fake CLI reporting version ("" = probe unavailable).
func codexResumeWithVersion(t *testing.T, version string, agentArgs []string) (*fakeRuntime, int, error) {
	t.Helper()
	rt := newFakeRuntime()
	rt.bindingStatuses["work|codex"] = sessioninventory.BindingEstablished
	rt.establishedSessions["work|codex"] = "native-root-1"
	if version != "" {
		rt.agentVersions["codex"] = version
	}
	args := LaunchArgs{
		Agent: "codex", AgentExplicit: true, ForcedTag: "work",
		AgentArgs: agentArgs, AgentArgsExplicit: true, AgentArgsFromCouch: true,
		ResumeRequired: true, RequiredSessionID: "native-root-1",
	}
	opts := baseOpts(args)
	scope, err := ResolveRepoScope(opts.Env.Cwd)
	if err != nil {
		t.Fatal(err)
	}
	opts.Env.CouchThreadScope, opts.Env.CouchThreadTag = scope.Key, "work"
	code, err := run(t, opts, rt)
	return rt, code, err
}

func TestCodexLaunchOptsOutOfTheSharedServer(t *testing.T) {
	rt, code, err := codexResumeWithVersion(t, "0.164.0", []string{"--sandbox", "workspace-write"})
	if err != nil || code != 0 {
		t.Fatalf("RunLaunch = %d, %v", code, err)
	}
	if got, want := launchArgsText(t, rt.env), "--no-daemon --disable shell_snapshot resume native-root-1 --sandbox workspace-write --no-alt-screen"; got != want {
		t.Fatalf("argv = %q want %q", got, want)
	}
	if got := rt.env[AgentVersionEnv]; got != "0.164.0" {
		t.Fatalf("PAIR_AGENT_VERSION = %q", got)
	}
}

func TestCodexLaunchBeforeTheGateIsUnchanged(t *testing.T) {
	rt, code, err := codexResumeWithVersion(t, "0.163.0", []string{"--sandbox", "workspace-write"})
	if err != nil || code != 0 {
		t.Fatalf("RunLaunch = %d, %v", code, err)
	}
	if got, want := launchArgsText(t, rt.env), "--disable shell_snapshot resume native-root-1 --sandbox workspace-write --no-alt-screen"; got != want {
		t.Fatalf("argv = %q want %q", got, want)
	}
	if got := rt.env[AgentVersionEnv]; got != "0.163.0" {
		t.Fatalf("PAIR_AGENT_VERSION = %q", got)
	}
}

func TestCodexLaunchHonorsAnExplicitNoDaemon(t *testing.T) {
	rt, code, err := codexResumeWithVersion(t, "0.164.0", []string{"--no-daemon", "--sandbox", "workspace-write"})
	if err != nil || code != 0 {
		t.Fatalf("RunLaunch = %d, %v", code, err)
	}
	argv := strings.Fields(launchArgsText(t, rt.env))
	count := 0
	for _, arg := range argv {
		if arg == "--no-daemon" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("--no-daemon appears %d times in %v", count, argv)
	}
}

// A prompt word after `--` is not a flag: the launcher must still opt out of
// the shared server.
func TestCodexLaunchTreatsPromptTextAsNoFlag(t *testing.T) {
	rt, code, err := codexResumeWithVersion(t, "0.164.0", []string{"--sandbox", "workspace-write", "--", "--no-daemon"})
	if err != nil || code != 0 {
		t.Fatalf("RunLaunch = %d, %v", code, err)
	}
	if got, want := launchArgsText(t, rt.env), "--no-daemon --disable shell_snapshot resume native-root-1 --sandbox workspace-write --no-alt-screen -- --no-daemon"; got != want {
		t.Fatalf("argv = %q want %q", got, want)
	}
}

func TestCodexLaunchWithoutAVersionProbeIsUnchanged(t *testing.T) {
	rt, code, err := codexResumeWithVersion(t, "", []string{"--sandbox", "workspace-write"})
	if err != nil || code != 0 {
		t.Fatalf("RunLaunch = %d, %v", code, err)
	}
	if got, want := launchArgsText(t, rt.env), "--disable shell_snapshot resume native-root-1 --sandbox workspace-write --no-alt-screen"; got != want {
		t.Fatalf("argv = %q want %q", got, want)
	}
	if got, ok := rt.env[AgentVersionEnv]; !ok || got != "" {
		t.Fatalf("PAIR_AGENT_VERSION = %q (present=%t), want cleared", got, ok)
	}
}

// The gate covers every codex launch, not only Couch's.
func TestStandaloneCodexLaunchOptsOutOfTheSharedServer(t *testing.T) {
	rt := newFakeRuntime()
	rt.agentVersions["codex"] = "0.164.0"
	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "bugfix"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("RunLaunch = %d, %v", code, err)
	}
	if got, want := launchArgsText(t, rt.env), "--no-daemon --no-alt-screen"; got != want {
		t.Fatalf("argv = %q want %q", got, want)
	}
	if got := rt.env[AgentVersionEnv]; got != "0.164.0" {
		t.Fatalf("PAIR_AGENT_VERSION = %q", got)
	}
}

// A launch from a pane whose env already carries a version must overwrite it:
// the ready record describes this launch, not the caller's (pair#891).
func TestAgentVersionEnvOverwritesAnInheritedValue(t *testing.T) {
	for _, tc := range []struct{ agent, inherited string }{
		{"codex", "0.163.0"},
		{"claude", "0.164.0"},
	} {
		rt := newFakeRuntime()
		rt.env[AgentVersionEnv] = tc.inherited
		code, err := run(t, baseOpts(LaunchArgs{Agent: tc.agent, ForcedTag: "bugfix"}), rt)
		if err != nil || code != 0 {
			t.Fatalf("%s: RunLaunch = %d, %v", tc.agent, code, err)
		}
		if got, ok := rt.env[AgentVersionEnv]; !ok || got != "" {
			t.Fatalf("%s: PAIR_AGENT_VERSION = %q (present=%t), want cleared", tc.agent, got, ok)
		}
	}
}
