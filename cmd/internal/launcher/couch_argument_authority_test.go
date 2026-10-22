package launcher

import (
	"reflect"
	"slices"
	"testing"
)

func TestCouchRepositoryArgumentsOverrideStalePerSlotConfig(t *testing.T) {
	for _, tc := range []struct {
		agent string
		argv  []string
	}{
		{"codex", []string{"--sandbox", "danger-full-access"}},
		{"claude", []string{"--dangerously-skip-permissions"}},
	} {
		t.Run(tc.agent, func(t *testing.T) {
			rt := newFakeRuntime()
			configPath := CanonicalConfigPath("/data", "work", tc.agent)
			rt.files[configPath] = `{"agent":"` + tc.agent + `","args":["--model","stale-session"],"session_id":"OLD"}`
			rt.agentSessions[tc.agent+"|OLD"] = true
			rt.uuids = []string{"NEW"}
			rt.pickFunc = func(string, []string) string { t.Fatal("Couch configuration opened a per-slot picker"); return "" }
			raw, err := BuildCouchLaunchProfile("work", tc.agent, tc.argv, "explicit", "explicit")
			if err != nil {
				t.Fatal(err)
			}
			args, _, err := ApplyCouchLaunchProfile(LaunchArgs{ForcedTag: "work"}, raw)
			if err != nil {
				t.Fatal(err)
			}
			code, err := run(t, baseOpts(args), rt)
			if err != nil || code != 0 {
				t.Fatalf("launch: code %d, %v", code, err)
			}
			command, err := DecodeAgentCommand(rt.env[AgentCommandEnv])
			if err != nil {
				t.Fatal(err)
			}
			if len(command.Argv) < len(tc.argv) || !reflect.DeepEqual(command.Argv[:len(tc.argv)], tc.argv) || slices.Contains(command.Argv, "stale-session") {
				t.Fatalf("per-slot config overrode accepted Couch arguments: %q", command.Argv)
			}
			if rt.defaultReads != 0 {
				t.Fatalf("Couch launch consulted Pair defaults %d times", rt.defaultReads)
			}
		})
	}
}

func TestStandaloneLaunchKeepsPerSlotPickerAndPairDefaults(t *testing.T) {
	for _, agent := range []string{"codex", "claude"} {
		t.Run(agent, func(t *testing.T) {
			rt := newFakeRuntime()
			configPath := CanonicalConfigPath("/data", "work", agent)
			saved := `{"agent":"` + agent + `","args":["--model","saved"],"session_id":"OLD"}`
			rt.files[configPath] = saved
			defaultPath := AgentDefaultPath("/data", agent)
			defaults, err := BuildAgentDefault(agent, []string{"--model", "standalone-default"})
			if err != nil {
				t.Fatal(err)
			}
			rt.files[defaultPath] = defaults
			pickerCalls := 0
			rt.pickFunc = func(string, []string) string { pickerCalls++; return "" }
			code, err := run(t, baseOpts(LaunchArgs{Agent: agent, ForcedTag: "work"}), rt)
			if err != nil || code != 1 || pickerCalls != 1 || rt.defaultReads != 1 {
				t.Fatalf("standalone policy changed: code=%d picker=%d defaults=%d err=%v", code, pickerCalls, rt.defaultReads, err)
			}
			if rt.launchCount != 0 || rt.files[configPath] != saved || rt.files[defaultPath] != defaults {
				t.Fatal("canceling the standalone picker changed its settings or launched")
			}
		})
	}
}
