package launcher

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// Exercise the real launch boundary: a resumed conversation creates a new
// wrapper, so it needs new readiness evidence even though its native ID stays.
func TestCouchPeerColdResumePublishesLaunchIdentity(t *testing.T) {
	for _, agent := range []string{"codex", "claude"} {
		t.Run(agent, func(t *testing.T) {
			opts := managedSessionOptions("create")
			opts.Args.Agent, opts.Args.AgentExplicit = agent, true
			opts.Args.AgentArgs = []string{}
			opts.Args.AgentArgsExplicit, opts.Args.AgentArgsFromCouch = true, true
			opts.Args.ResumeRequired, opts.Args.RequiredSessionID = true, "native-root-1"
			rt := &couchSessionRuntime{fakeRuntime: newFakeRuntime()}
			key := opts.Args.ForcedTag + "|" + agent
			rt.bindingStatuses[key] = sessioninventory.BindingEstablished
			rt.establishedSessions[key] = "native-root-1"
			rt.env["PAIR_LAUNCH_NONCE"] = "stale-parent"
			var stderr bytes.Buffer
			code, err := RunLaunch(opts, rt, &stderr)
			if err != nil || code != 0 || rt.launchCount != 1 {
				t.Fatalf("launch code=%d err=%v stderr=%s", code, err, &stderr)
			}
			if nonce := rt.env["PAIR_LAUNCH_NONCE"]; nonce == "" || nonce == "stale-parent" {
				t.Fatalf("resumed wrapper has no fresh launch identity: %q", nonce)
			}
			if rt.env["PAIR_SESSION_NAME"] != opts.Args.CouchSession.Name || rt.env["PAIR_SESSION_ID"] != "native-root-1" {
				t.Fatal("resume lost the terminal or native conversation identity")
			}
		})
	}
}

func TestCouchCodexLaunchUsesCurrentShellEnvironment(t *testing.T) {
	for _, hosted := range []bool{false, true} {
		t.Run(map[bool]string{false: "standalone", true: "couch"}[hosted], func(t *testing.T) {
			opts := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: []string{"--model", "model", "--", "prompt"}, AgentArgsExplicit: true, AgentArgsFromCouch: true, FreshRequired: true})
			if hosted {
				scope, _ := ResolveRepoScope(opts.Env.Cwd)
				opts.Env.CouchThreadScope, opts.Env.CouchThreadTag = scope.Key, "work"
			}
			rt := newFakeRuntime()
			code, err := run(t, opts, rt)
			if err != nil || code != 0 {
				t.Fatalf("launch: %d %v", code, err)
			}
			command, err := DecodeAgentCommand(rt.env[AgentCommandEnv])
			if err != nil {
				t.Fatal(err)
			}
			// The inline flag lands before `--`: after it, codex would read
			// the flag as prompt text (#410).
			want := []string{"--model", "model", "--no-alt-screen", "--", "prompt"}
			if hosted {
				want = append([]string{"--disable", "shell_snapshot"}, want...)
			}
			if !reflect.DeepEqual(command.Argv, want) {
				t.Fatalf("argv=%q want=%q", command.Argv, want)
			}
			for _, entry := range rt.ledger["work"] {
				if strings.Contains(strings.Join(entry.Args, " "), "shell_snapshot") {
					t.Fatal("runtime shell policy leaked into saved agent arguments")
				}
			}
		})
	}
}
