package sessionwatch

import (
	"os"
	"reflect"
	"testing"
	"time"
)

func TestBuildOptionsFromArgsAndEnv(t *testing.T) {
	env := map[string]string{
		"HOME":                                "/home/me",
		"PAIR_DATA_DIR":                       "/tmp/pair-data",
		"PAIR_SESSION_WATCH_PID_WAIT_SECONDS": "3",
	}
	opts, ok := buildOptions([]string{"codex", "tag", "/repo", "resume", "old", "--flag"}, func(k string) string {
		return env[k]
	})
	if !ok {
		t.Fatalf("buildOptions returned !ok")
	}
	if opts.Agent != "codex" || opts.Tag != "tag" || opts.Cwd != "/repo" {
		t.Fatalf("opts identity = %+v", opts)
	}
	if opts.Home != "/home/me" || opts.DataDir != "/tmp/pair-data" {
		t.Fatalf("opts paths = %+v", opts)
	}
	if opts.PIDWait != 3*time.Second || opts.Timeout != 60*time.Second || opts.Poll != 100*time.Millisecond {
		t.Fatalf("opts durations = %+v", opts)
	}
	if !reflect.DeepEqual(opts.Args, []string{"resume", "old", "--flag"}) {
		t.Fatalf("opts args = %#v", opts.Args)
	}
}

func TestBuildOptionsParsesRepoIdentityBeforeAgentArgs(t *testing.T) {
	opts, ok := buildOptions([]string{"codex", "tag", "/repo/sub", "--repo-root", "/repo", "--repo-name", "pair", "--", "resume", "old", "--repo-root", "agent-value"}, func(k string) string {
		if k == "HOME" {
			return "/home/me"
		}
		return ""
	})
	if !ok {
		t.Fatalf("buildOptions returned !ok")
	}
	if opts.Cwd != "/repo/sub" || opts.RepoRoot != "/repo" || opts.RepoName != "pair" {
		t.Fatalf("opts identity = %+v", opts)
	}
	if !reflect.DeepEqual(opts.Args, []string{"resume", "old", "--repo-root", "agent-value"}) {
		t.Fatalf("opts args = %#v", opts.Args)
	}
}

func TestCommandArgsRoundTripsWatcherMetadataWithoutConsumingAgentArgs(t *testing.T) {
	bound := time.Date(2026, 8, 19, 9, 23, 45, 123456789, time.UTC)
	args := CommandArgs("/pair", "codex", "tag", "/repo/sub", "/repo", "pair", bound,
		[]string{"--pid-not-before", "agent-value", "--repo-root", "agent-root"})
	wantPrefix := []string{"/pair", "session-watch"}
	if !reflect.DeepEqual(args[:2], wantPrefix) {
		t.Fatalf("command prefix = %v, want %v", args[:2], wantPrefix)
	}
	opts, ok := buildOptions(args[2:], func(k string) string { return "" })
	if !ok {
		t.Fatal("buildOptions returned !ok")
	}
	if !opts.PIDNotBefore.Equal(bound) || opts.RepoRoot != "/repo" || opts.RepoName != "pair" {
		t.Fatalf("watcher metadata = %+v", opts)
	}
	wantAgentArgs := []string{"--pid-not-before", "agent-value", "--repo-root", "agent-root"}
	if !reflect.DeepEqual(opts.Args, wantAgentArgs) {
		t.Fatalf("agent args = %v, want %v", opts.Args, wantAgentArgs)
	}
}

func TestBuildOptionsRejectsMalformedPIDGenerationBound(t *testing.T) {
	if _, ok := buildOptions([]string{"codex", "tag", "/repo", "--pid-not-before", "not-a-time", "--"}, func(string) string { return "" }); ok {
		t.Fatal("buildOptions should reject malformed --pid-not-before")
	}
}

func TestBuildOptionsRejectsMissingRequiredArgs(t *testing.T) {
	if _, ok := buildOptions([]string{"codex", "tag"}, func(string) string { return "" }); ok {
		t.Fatalf("buildOptions should reject missing cwd")
	}
}

func TestEnsurePairTagFallback(t *testing.T) {
	t.Setenv("PAIR_TAG", "")
	cleanup := ensurePairTag("from-positional")
	defer cleanup()
	if got := os.Getenv("PAIR_TAG"); got != "from-positional" {
		t.Fatalf("PAIR_TAG = %q, want fallback tag", got)
	}
	cleanup()
	if got := os.Getenv("PAIR_TAG"); got != "" {
		t.Fatalf("PAIR_TAG after cleanup = %q, want empty", got)
	}
}
