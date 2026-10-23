package couchcore

import (
	"context"
	"testing"
)

func TestSwitchAgentDispatchRejectsNullArgumentElements(t *testing.T) {
	for _, target := range []string{"claude", "codex"} {
		for _, raw := range []string{`null`, `[null]`, `["--model",null]`} {
			t.Run(target+"/"+raw, func(t *testing.T) {
				env := newTestEnv(t, "/repo")
				sourceAgent := "claude"
				if target == sourceAgent {
					sourceAgent = "codex"
				}
				source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: sourceAgent, Argv: []string{}})
				_, err := DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch)}, OperationCall{Name: "prepare-switch-agent", Implicit: true, Context: context.Background(), Args: map[string]string{"repo-scope": source.Address.RepoScope, "tag": string(source.Address.Tag), "agent": target, "argv": raw}})
				if err == nil {
					t.Fatalf("accepted null startup argument as empty string: %s", raw)
				}
			})
		}
	}
}
