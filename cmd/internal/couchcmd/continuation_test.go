package couchcmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestContinuationEnvironmentRequiresMatchingPaneAddress(t *testing.T) {
	base := map[string]string{
		"COUCH_THREAD_SCOPE": "scope", "PAIR_SCOPE_KEY": "scope",
		"COUCH_THREAD_TAG": "thread", "PAIR_TAG": "thread",
		"PAIR_AGENT": "codex", "PAIR_SESSION_NAME": "session",
		"ZELLIJ_SESSION_NAME": "session", "PAIR_LAUNCH_ORDINAL": "3",
		checkpoint.DigestEnv: strings.Repeat("a", 64),
	}
	for _, field := range []string{"", "PAIR_SCOPE_KEY", "COUCH_THREAD_SCOPE", "PAIR_TAG", "COUCH_THREAD_TAG", "PAIR_SESSION_NAME", "ZELLIJ_SESSION_NAME", "PAIR_AGENT", "PAIR_LAUNCH_ORDINAL", checkpoint.DigestEnv} {
		t.Run(field, func(t *testing.T) {
			getenv := func(key string) string {
				if key == field {
					return ""
				}
				return base[key]
			}
			args := map[string]string{"path": "/checkpoint.md"}
			err := bindContinuationEnvironment(args, getenv)
			if (err == nil) != (field == "") {
				t.Fatalf("field=%s error=%v", field, err)
			}
			if field == "" && (args["repo-scope"] != "scope" || args["tag"] != "thread" || args["launch-ordinal"] != "3" || args["expected-digest"] != base[checkpoint.DigestEnv]) {
				t.Fatalf("bound arguments: %v", args)
			}
		})
	}
	for _, field := range []string{"PAIR_SCOPE_KEY", "PAIR_TAG", "PAIR_SESSION_NAME", "PAIR_LAUNCH_ORDINAL", checkpoint.DigestEnv} {
		getenv := func(key string) string {
			if key == field {
				return "different"
			}
			return base[key]
		}
		if err := bindContinuationEnvironment(map[string]string{}, getenv); err == nil {
			t.Fatalf("accepted mismatched %s", field)
		}
	}
}

// The writer's request never takes a second owner; the exits from a failed
// request are retired (#522) and no longer reach the CLI at all.
func TestContinuationCLIVerbs(t *testing.T) {
	if operationOwnsLive("request-continuation") {
		t.Fatal("writer request must not acquire a second owner")
	}
	for _, retired := range []string{"retry-continuation", "dismiss-continuation", "continuation-status"} {
		if _, ok := Resolve(retired); ok || operationOwnsLive(retired) || operationUsesCurrentRepoScope(retired) {
			t.Errorf("%s still reaches the CLI", retired)
		}
	}
}

func TestContinuationNonConsoleResultWaitsForOwnedChild(t *testing.T) {
	runner := couchcore.NewFakeRunner()
	runner.AutoExit(7)
	handle, err := runner.Start("/repo", []string{"pair"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	result := couchcore.ContinuationResult{Handle: handle, Record: couchcore.ActorRecord{ID: "target"}}
	if code := render(&output, couchcore.Operation{Name: "continue-thread"}, result); code != 7 {
		t.Fatalf("code=%d; owner returned without waiting for target: %s", code, output.String())
	}
}
