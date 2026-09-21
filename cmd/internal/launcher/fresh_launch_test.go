package launcher

import (
	"errors"
	"reflect"
	"testing"
)

func TestValidateFreshAgentArgs(t *testing.T) {
	for agent, cases := range map[string][][]string{
		"claude": {{"-c"}, {"-rabc"}, {"-pc"}, {"--resume=x"}, {"--session-id=x"}, {"--fork-session"}, {"--from-pr", "12"}, {"--teleport"}, {"--debug", "--continue"}, {"attach", "x"}},
		"codex":  {{"resume"}, {"--model", "x", "fork", "--last"}, {"-cx=y", "resume"}, {"--approve-for-me", "resume"}, {"exec", "--model", "x", "resume"}, {"--add-dir", "/x", "resume", "abc"}, {"--add-dir", "/x", "fork", "--last"}},
		"agy":    {{"-c"}, {"--continue=true"}, {"--conversation=x"}, {"-conversation=x"}},
		"muse":   {{"--provider", "echo", "resume"}, {"resume", "--last"}, {"exec", "--session-id=x"}, {"--worktree", "create", "resume"}},
		"qoder":  {{"-c"}, {"--continue"}, {"-r", "abc"}, {"--resume", "abc"}, {"--resume=abc"}, {"--session-id", "x"}, {"--session-id=x"}, {"--fork-session"}, {"--remote"}, {"--remote", "task"}, {"--remote-session", "x"}, {"--teleport", "x"}, {"--remote-control", "x"}, {"--list-sessions"}, {"--delete-session", "1"}, {"-dc"}, {"--debug", "--continue"}, {"--worktree", "--resume"}},
	} {
		for _, args := range cases {
			if err := ValidateFreshAgentArgs(agent, args); err == nil {
				t.Errorf("%s accepted %#v", agent, args)
			}
		}
	}
	for agent, cases := range map[string][][]string{
		"claude": {{"--model", "--resume"}, {"--system-prompt", "--continue"}, {"--", "--resume"}, {"--name", "attach"}, {"--allowed-tools", "Bash", "attach"}},
		"codex":  {{"--model", "resume"}, {"-c", "resume"}, {"-mresume"}, {"--", "resume"}, {"a prompt about resume"}},
		"agy":    {{"--model", "--continue"}}, "muse": {{"--model", "resume"}, {"--provider=resume"}},
		"qoder": {{"-m", "some-model", "hello"}, {"--model", "m", "-p"}, {"--model", "--resume"}, {"--worktree", "feat", "hello"}, {"--tools", "a", "b", "--", "hi"}, {"-w", "some/dir", "hello"}, {"-dp"}, {"--", "--resume"}, {"hello", "world"}},
	} {
		for _, args := range cases {
			if err := ValidateFreshAgentArgs(agent, args); err != nil {
				t.Errorf("%s rejected %#v: %v", agent, args, err)
			}
		}
	}
}
func TestFreshVariadicOptionIsPerAgent(t *testing.T) {
	cases := []struct {
		agent, flag string
		want        bool
	}{
		{"claude", "--add-dir", true},
		{"claude", "--file", true},
		{"claude", "--tools", true},
		{"codex", "--add-dir", false},
		{"agy", "--add-dir", false},
		{"qoder", "--add-dir", false},
		{"qoder", "--tools", true},
		{"codex", "--tools", false},
		{"muse", "--tools", false},
		{"agy", "--tools", false},
	}
	for _, tc := range cases {
		if got := freshVariadicOption(tc.agent, tc.flag); got != tc.want {
			t.Errorf("freshVariadicOption(%q, %q) = %v, want %v", tc.agent, tc.flag, got, tc.want)
		}
	}
}

func TestFreshLaunchProfile(t *testing.T) {
	raw, err := BuildCouchFreshLaunchProfile("work", "codex", []string{}, "explicit", "explicit")
	if err != nil {
		t.Fatal(err)
	}
	got, source, err := ApplyCouchLaunchProfile(LaunchArgs{ForcedTag: "work"}, raw)
	if err != nil || !got.FreshRequired || source != "explicit" || got.AgentArgs == nil {
		t.Fatalf("profile %#v %s %v", got, source, err)
	}
	p := TrustedLaunchProfile{SchemaVersion: 1, Tag: "work", Agent: "claude", Argv: []string{}, AgentSource: "explicit", ArgvSource: "explicit", FreshRequired: true, ResumeRequired: true, RequiredSessionID: "x"}
	if ValidateTrustedLaunchProfile(p) == nil {
		t.Fatal("accepted fresh resume")
	}
}
func TestFreshCouchLaunchSkipsDefaultsPickerAndDraft(t *testing.T) {
	rt := newFakeRuntime()
	rt.pickFunc = func(string, []string) string { t.Fatal("fresh launch called picker"); return "" }
	rt.files["/data/config-work-codex.json"] = `{"agent":"codex","args":["resume","old"],"session_id":"old"}`
	rt.files["/data/draft-work.md"] = "operator draft\n"
	args := LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: []string{"--model", "a b", "", "雪"}, AgentArgsExplicit: true, AgentArgsFromCouch: true, FreshRequired: true}
	opts := baseOpts(args)
	opts.ContinueText = "must not touch"
	opts.ContinueDoc = "some.md"
	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("launch %d %v", code, err)
	}
	if rt.defaultReads != 0 || rt.files["/data/agent-default-codex.json"] != "" || len(rt.existingThreads) != 1 || len(rt.threadClaims) != 0 {
		t.Fatalf("defaults/registration: reads %d writes %#v existing %#v claims %#v", rt.defaultReads, rt.files["/data/agent-default-codex.json"], rt.existingThreads, rt.threadClaims)
	}
	if rt.env["PAIR_LAUNCH_NONCE"] == "" {
		t.Fatal("fresh launch missing nonce")
	}
	command, err := DecodeAgentCommand(rt.env[AgentCommandEnv])
	if err != nil {
		t.Fatal(err)
	}
	want := append(append([]string{}, args.AgentArgs...), "--no-alt-screen")
	if !reflect.DeepEqual(command.Argv, want) {
		t.Fatalf("argv %#v want %#v", command.Argv, want)
	}
	if rt.files["/data/draft-work.md"] != "operator draft\n" {
		t.Fatalf("draft changed: %q", rt.files["/data/draft-work.md"])
	}
	for _, content := range rt.files {
		if content == "must not touch" {
			t.Fatal("mutated draft")
		}
	}
}
func TestFreshCouchLaunchRefusesAttach(t *testing.T) {
	rt := newFakeRuntime()
	scope := mustScope(t, "/home/u/work")
	rt.sessionIndex = SessionNameIndex{Entries: []SessionNameEntry{{SessionName: "📁work", ScopeKey: scope.Key, RepoRoot: scope.Root, RepoName: scope.DisplayName, Tag: "work"}}}
	rt.sessions = []Session{{Name: "📁work", Tag: "work", RepoName: "work", Agent: "codex", State: SessionDetached}}
	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", FreshRequired: true, AgentArgsExplicit: true, AgentArgsFromCouch: true}), rt)
	if err != nil || code != 1 || len(rt.attached) != 0 || rt.launchCount != 0 {
		t.Fatalf("attach race %d %v %#v", code, err, rt.attached)
	}
}

func TestFreshCouchFailureNeverPersistsRepoDefault(t *testing.T) {
	for _, stage := range []string{"registration", "launch"} {
		t.Run(stage, func(t *testing.T) {
			rt := newFakeRuntime()
			path := AgentDefaultPath("/data", "codex")
			rt.files[path] = "original default bytes"
			if stage == "registration" {
				rt.existingThreadErr = errors.New("registration failed")
			} else {
				rt.launchErr = errors.New("launch failed")
			}
			args := LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: []string{"--model", "new"}, AgentArgsExplicit: true, AgentArgsFromCouch: true, FreshRequired: true}
			code, err := run(t, baseOpts(args), rt)
			if err != nil || code != 1 {
				t.Fatalf("failure %d %v", code, err)
			}
			if got := rt.files[path]; got != "original default bytes" {
				t.Fatalf("default changed: %q", got)
			}
		})
	}
}
