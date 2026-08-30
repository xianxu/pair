package launcher

import (
	"reflect"
	"testing"
)

func TestTrustedResumeProfileValidation(t *testing.T) {
	validResume := TrustedLaunchProfile{
		SchemaVersion: 1, Tag: "couch-0102030405060708", Agent: "codex", Argv: []string{},
		AgentSource: "saved", ArgvSource: "saved", ResumeRequired: true, RequiredSessionID: "native-root-1",
	}
	if err := ValidateTrustedLaunchProfile(validResume); err != nil {
		t.Fatalf("valid resume profile: %v", err)
	}
	if err := ValidateTrustedLaunchProfile(TrustedLaunchProfile{
		SchemaVersion: 1, Tag: validResume.Tag, Agent: "claude", Argv: []string{},
		AgentSource: "explicit", ArgvSource: "repo-default",
	}); err != nil {
		t.Fatalf("valid ordinary profile: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*TrustedLaunchProfile)
	}{
		{name: "required ID empty", mutate: func(p *TrustedLaunchProfile) { p.RequiredSessionID = "" }},
		{name: "unsupported agent", mutate: func(p *TrustedLaunchProfile) { p.Agent = "other" }},
		{name: "null argv", mutate: func(p *TrustedLaunchProfile) { p.Argv = nil }},
		{name: "explicit agent source", mutate: func(p *TrustedLaunchProfile) { p.AgentSource = "explicit" }},
		{name: "path agent source", mutate: func(p *TrustedLaunchProfile) { p.AgentSource = "path" }},
		{name: "root agent source", mutate: func(p *TrustedLaunchProfile) { p.AgentSource = "root" }},
		{name: "path argv source", mutate: func(p *TrustedLaunchProfile) { p.ArgvSource = "path" }},
		{name: "repo default argv source", mutate: func(p *TrustedLaunchProfile) { p.ArgvSource = "repo-default" }},
		{name: "ordinary carries required ID", mutate: func(p *TrustedLaunchProfile) {
			p.ResumeRequired = false
			p.AgentSource = "explicit"
			p.ArgvSource = "path"
		}},
		{name: "ordinary carries saved sources", mutate: func(p *TrustedLaunchProfile) {
			p.ResumeRequired = false
			p.RequiredSessionID = ""
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := validResume
			profile.Argv = append([]string(nil), validResume.Argv...)
			test.mutate(&profile)
			if err := ValidateTrustedLaunchProfile(profile); err == nil {
				t.Fatalf("accepted %+v", profile)
			}
		})
	}
}

func TestBuildAndApplyTrustedResumeProfileCarriesRequiredIdentity(t *testing.T) {
	raw, err := BuildCouchResumeLaunchProfile(
		"couch-0102030405060708", "muse", []string{"--model", "spark"}, "native-root-1",
	)
	if err != nil {
		t.Fatal(err)
	}
	args, err := ParseArgs([]string{"resume", "couch-0102030405060708", "--layout2"})
	if err != nil {
		t.Fatal(err)
	}
	got, source, err := ApplyCouchLaunchProfile(args, raw)
	if err != nil {
		t.Fatal(err)
	}
	if !got.ResumeRequired || got.RequiredSessionID != "native-root-1" || source != "saved" ||
		got.Agent != "muse" || !reflect.DeepEqual(got.AgentArgs, []string{"--model", "spark"}) {
		t.Fatalf("applied resume profile = %+v, source=%q", got, source)
	}
}

func TestDecideLaunchArgsExplicitArgsWinAndPersist(t *testing.T) {
	got := DecideLaunchArgs(LaunchArgInputs{
		Agent: "codex",
		Args: LaunchArgs{
			AgentArgs:         []string{"--sandbox", "workspace-write"},
			AgentArgsExplicit: true,
		},
		Saved:        savedConfig{Agent: "codex", Args: []string{"--saved"}, SessionID: "SID"},
		SavedResumes: true,
		Default:      AgentDefault{Agent: "codex", Args: []string{"--default"}},
		DefaultFound: true,
	})
	if !reflect.DeepEqual(got.Args, []string{"resume", "SID", "--sandbox", "workspace-write"}) {
		t.Fatalf("Args = %#v, want explicit args composed with saved resume once", got.Args)
	}
	if !got.PersistDefault || got.ClearDefault {
		t.Fatalf("persist/clear = %v/%v, want persist only", got.PersistDefault, got.ClearDefault)
	}
}

func TestDecideLaunchArgsEmptyExplicitSeparatorClearsDefault(t *testing.T) {
	got := DecideLaunchArgs(LaunchArgInputs{
		Agent: "claude",
		Args: LaunchArgs{
			AgentArgsExplicit: true,
		},
		Default:      AgentDefault{Agent: "claude", Args: []string{"--model", "opus"}},
		DefaultFound: true,
	})
	if len(got.Args) != 0 {
		t.Fatalf("Args = %#v, want empty explicit args", got.Args)
	}
	if !got.PersistDefault || !got.ClearDefault {
		t.Fatalf("persist/clear = %v/%v, want clear default after readiness", got.PersistDefault, got.ClearDefault)
	}
}

func TestDecideLaunchArgsSavedConfigWinsOverRepoDefault(t *testing.T) {
	got := DecideLaunchArgs(LaunchArgInputs{
		Agent:        "claude",
		Saved:        savedConfig{Agent: "claude", Args: []string{"--saved"}, SessionID: "SID"},
		SavedResumes: true,
		Default:      AgentDefault{Agent: "claude", Args: []string{"--default"}},
		DefaultFound: true,
	})
	if !reflect.DeepEqual(got.Args, []string{"--saved", "--resume", "SID"}) {
		t.Fatalf("Args = %#v, want saved config args with resume", got.Args)
	}
	if got.PersistDefault || got.ClearDefault {
		t.Fatalf("persist/clear = %v/%v, want neither for implicit saved config", got.PersistDefault, got.ClearDefault)
	}
}

func TestDecideLaunchArgsRepoDefaultWinsWhenNoSavedConfig(t *testing.T) {
	got := DecideLaunchArgs(LaunchArgInputs{
		Agent:        "codex",
		Default:      AgentDefault{Agent: "codex", Args: []string{"--model", "gpt-5"}},
		DefaultFound: true,
	})
	if !reflect.DeepEqual(got.Args, []string{"--model", "gpt-5"}) {
		t.Fatalf("Args = %#v, want repo default args", got.Args)
	}
}

func TestDecideLaunchArgsWarnsAndDropsStaleSavedResume(t *testing.T) {
	got := DecideLaunchArgs(LaunchArgInputs{
		Agent:        "codex",
		Saved:        savedConfig{Agent: "codex", Args: []string{"resume", "OLD", "--search"}, SessionID: "OLD"},
		SavedResumes: false,
	})
	if !reflect.DeepEqual(got.Args, []string{"--search"}) {
		t.Fatalf("Args = %#v, want cleaned saved args without stale resume token", got.Args)
	}
	if len(got.Warnings) == 0 {
		t.Fatal("Warnings empty, want stale resume warning")
	}
}

func TestApplyCouchLaunchProfilePreservesExactPathArgs(t *testing.T) {
	args, err := ParseArgs([]string{"resume", "couch-0102030405060708", "--layout2"})
	if err != nil {
		t.Fatal(err)
	}
	raw := `{"schema_version":1,"tag":"couch-0102030405060708","agent":"codex","argv":["--sandbox","workspace-write"],"agent_source":"explicit","argv_source":"path"}`
	got, source, err := ApplyCouchLaunchProfile(args, raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Agent != "codex" || !got.AgentExplicit || !got.AgentArgsExplicit ||
		!reflect.DeepEqual(got.AgentArgs, []string{"--sandbox", "workspace-write"}) || source != "path" {
		t.Fatalf("ApplyCouchLaunchProfile = %+v, source=%q", got, source)
	}
}

func TestApplyCouchLaunchProfileLeavesRepoDefaultArgsToPair(t *testing.T) {
	args, err := ParseArgs([]string{"resume", "couch-0102030405060708", "--layout2"})
	if err != nil {
		t.Fatal(err)
	}
	raw := `{"schema_version":1,"tag":"couch-0102030405060708","agent":"muse","argv":["--model","spark"],"agent_source":"root","argv_source":"repo-default"}`
	got, source, err := ApplyCouchLaunchProfile(args, raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Agent != "muse" || !got.AgentExplicit || !got.AgentArgsExplicit ||
		!reflect.DeepEqual(got.AgentArgs, []string{"--model", "spark"}) || !got.AgentArgsFromCouch || source != "repo-default" {
		t.Fatalf("ApplyCouchLaunchProfile = %+v, source=%q", got, source)
	}
}

func TestDecideLaunchArgsDoesNotRewriteRepoDefaultForCouchResolvedArgs(t *testing.T) {
	got := DecideLaunchArgs(LaunchArgInputs{
		Agent: "codex",
		Args: LaunchArgs{
			AgentArgs:          []string{"--search"},
			AgentArgsExplicit:  true,
			AgentArgsFromCouch: true,
		},
	})
	if !reflect.DeepEqual(got.Args, []string{"--search"}) || got.PersistDefault || got.ClearDefault {
		t.Fatalf("DecideLaunchArgs = %+v", got)
	}
}

func TestApplyCouchLaunchProfileRejectsWrongTagAndUnknownAgent(t *testing.T) {
	args, err := ParseArgs([]string{"resume", "couch-0102030405060708"})
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		`{"schema_version":1,"tag":"couch-1112131415161718","agent":"codex","argv":[],"agent_source":"path","argv_source":"path"}`,
		`{"schema_version":1,"tag":"couch-0102030405060708","agent":"gemini","argv":[],"agent_source":"path","argv_source":"path"}`,
	} {
		if _, _, err := ApplyCouchLaunchProfile(args, raw); err == nil {
			t.Fatalf("invalid couch launch profile accepted: %s", raw)
		}
	}
}
