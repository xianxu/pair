package launcher

import (
	"reflect"
	"strings"
	"testing"
)

func TestNativeResumeSelectorClassifiesAndPins(t *testing.T) {
	cases := []struct {
		agent  string
		args   []string
		kind   NativeResumeKind
		id     string
		pinned []string
	}{
		{"codex", []string{"--model", "m", "resume", "--last", "fix this"}, NativeResumeLatest, "", []string{"--model", "m", "resume", "canonical", "fix this"}},
		{"muse", []string{"--provider", "echo", "resume", "--last", "--workspace", "/work"}, NativeResumeLatest, "", []string{"--provider", "echo", "resume", "canonical", "--workspace", "/work"}},
		{"claude", []string{"--model", "m", "-c", "--", "literal --resume"}, NativeResumeLatest, "", []string{"--model", "m", "--resume", "canonical", "--", "literal --resume"}},
		{"qoder", []string{"-rabc", "--model", "m"}, NativeResumeExplicit, "abc", []string{"--model", "m", "--resume", "canonical"}},
		{"grok", []string{"--resume"}, NativeResumeLatest, "", []string{"--resume", "canonical"}},
		{"agy", []string{"--conversation", "id"}, NativeResumeExplicit, "id", []string{"--conversation", "canonical"}},
		{"codex", []string{"exec", "resume", "id"}, NativeResumeExplicit, "id", []string{"exec", "resume", "canonical"}},
		{"claude", []string{"-pc"}, NativeResumeLatest, "", []string{"-p", "--resume", "canonical"}},
		{"claude", []string{"--resume"}, NativeResumePicker, "", []string{"--resume", "canonical"}},
	}
	for _, tc := range cases {
		t.Run(tc.agent+"/"+string(tc.kind), func(t *testing.T) {
			s, err := ParseNativeResumeSelector(tc.agent, tc.args)
			if err != nil || s.Kind != tc.kind || s.NativeID != tc.id {
				t.Fatalf("selector=%+v err=%v", s, err)
			}
			got, err := PinNativeResumeSelector(s, "canonical")
			if err != nil || !reflect.DeepEqual(got, tc.pinned) {
				t.Fatalf("pin=%v err=%v want=%v", got, err, tc.pinned)
			}
			if !reflect.DeepEqual(s.Args, tc.args) {
				t.Fatal("parser changed caller argv")
			}
		})
	}
}

func TestNativeResumeSelectorRespectsValuesAndFreshBranches(t *testing.T) {
	for _, tc := range []struct {
		agent string
		args  []string
		kind  NativeResumeKind
	}{
		{"claude", []string{"--model", "--resume"}, NativeResumeNone},
		{"claude", []string{"--system-prompt", "--continue"}, NativeResumeNone},
		{"codex", []string{"-c", "resume"}, NativeResumeNone},
		{"muse", []string{"--provider=resume"}, NativeResumeNone},
		{"claude", []string{"--", "--resume", "id"}, NativeResumeNone},
		{"claude", []string{"--session-id", "new"}, NativeResumeNew},
		{"grok", []string{"--resume", "old", "--fork-session", "--session-id", "new"}, NativeResumeFork},
		{"codex", []string{"fork", "--last"}, NativeResumeFork},
		{"sh", []string{"-c", "echo hi"}, NativeResumeNone},
	} {
		s, err := ParseNativeResumeSelector(tc.agent, tc.args)
		if err != nil || s.Kind != tc.kind {
			t.Errorf("%s %v: %+v %v", tc.agent, tc.args, s, err)
		}
	}
}

func TestNativeResumeSelectorRejectsContradictoryOrMalformedSelection(t *testing.T) {
	for _, tc := range []struct {
		agent string
		args  []string
	}{
		{"claude", []string{"--continue", "--resume", "id"}},
		{"claude", []string{"--resume", "id", "--session-id", "new"}},
		{"codex", []string{"resume", "--last=bad"}},
		{"codex", []string{"--last"}},
		{"grok", []string{"--resume", "a", "--resume", "b"}},
	} {
		if _, err := ParseNativeResumeSelector(tc.agent, tc.args); err == nil {
			t.Errorf("accepted %s %v", tc.agent, tc.args)
		}
	}
}

func TestNativeResumeSelectorAdditionalOptionBoundaries(t *testing.T) {
	for _, tc := range []struct {
		agent string
		args  []string
		kind  NativeResumeKind
		id    string
	}{
		{"muse", []string{"--worktree", "off", "resume", "id"}, NativeResumeExplicit, "id"},
		{"claude", []string{"-prsid"}, NativeResumeExplicit, "sid"},
		{"grok", []string{"-mc"}, NativeResumeNone, ""},
		{"codex", []string{"resume", "id", "--all"}, NativeResumeExplicit, "id"},
	} {
		s, err := ParseNativeResumeSelector(tc.agent, tc.args)
		if err != nil || s.Kind != tc.kind || s.NativeID != tc.id {
			t.Errorf("%s %v => %+v %v", tc.agent, tc.args, s, err)
		}
	}
	if _, err := ParseNativeResumeSelector("claude", []string{"-crsid"}); err == nil {
		t.Error("contradictory cluster accepted")
	}
}

func TestNativeResumeSelectorCommandOptionsBeforeIDAndCWD(t *testing.T) {
	for _, tc := range []struct {
		agent string
		args  []string
		cwd   string
	}{
		{"codex", []string{"resume", "--model", "m", "id"}, ""},
		{"muse", []string{"resume", "--provider", "echo", "id"}, ""},
		{"codex", []string{"-C/work", "resume", "id"}, "/work"},
		{"qoder", []string{"-w/work", "--resume", "id"}, "/work"},
	} {
		s, err := ParseNativeResumeSelector(tc.agent, tc.args)
		if err != nil || s.Kind != NativeResumeExplicit || s.NativeID != "id" || s.CWD != tc.cwd {
			t.Errorf("%s %v => %+v %v", tc.agent, tc.args, s, err)
		}
		if err == nil {
			args, err := PinNativeResumeSelector(s, "canonical")
			if err != nil {
				t.Error(err)
			} else {
				next, err := ParseNativeResumeSelector(tc.agent, args)
				if err != nil || next.NativeID != "canonical" {
					t.Errorf("pin failed: %v %+v %v", args, next, err)
				}
			}
		}
	}
}

func TestNativeResumeSelectorAgyGoFlagForms(t *testing.T) {
	for _, args := range [][]string{{"-conversation", "id"}, {"--conversation=id"}, {"-conversation=id"}, {"--resume=id"}} {
		s, err := ParseNativeResumeSelector("agy", args)
		if err != nil || s.Kind != NativeResumeExplicit || s.NativeID != "id" {
			t.Errorf("%v => %+v %v", args, s, err)
		}
	}
	for _, args := range [][]string{{"--continue=true"}, {"-c=true"}} {
		s, err := ParseNativeResumeSelector("agy", args)
		if err != nil || s.Kind != NativeResumeLatest {
			t.Errorf("%v => %+v %v", args, s, err)
		}
	}
	s, err := ParseNativeResumeSelector("agy", []string{"--continue=false"})
	if err != nil || s.Kind != NativeResumeNone {
		t.Errorf("false continue=%+v %v", s, err)
	}
}

func TestCodexIndirectResumeCompatibilityKeepsFullUUIDGuarded(t *testing.T) {
	for _, tc := range []struct {
		agent string
		args  []string
		want  bool
	}{
		{"codex", []string{"resume", "--last"}, true},
		{"codex", []string{"resume"}, true},
		{"codex", []string{"resume", "session-name"}, true},
		{"codex", []string{"resume", "019d1111"}, true},
		{"codex", []string{"resume", "search words"}, true},
		{"codex", []string{"resume", "019d1111-1111-7111-8111-111111111111"}, false},
		{"codex", []string{"resume", "FFFFFFFF-FFFF-4FFF-8FFF-FFFFFFFFFFFF"}, false},
		{"codex", []string{"fork", "--last"}, false},
		{"codex", []string{"hello"}, false},
		{"codex", []string{"--session-id", "new"}, false},
		{"claude", []string{"--continue"}, false},
		{"qoder", []string{"--continue"}, false},
		{"muse", []string{"resume", "--last"}, false},
		{"grok", []string{"--resume", "session-name"}, false},
		{"agy", []string{"--continue"}, false},
	} {
		s, err := ParseNativeResumeSelector(tc.agent, tc.args)
		if err != nil {
			t.Fatal(err)
		}
		if got := CodexIndirectResumeCompatibility(s); got != tc.want {
			t.Errorf("%s %v => compatibility=%v want=%v", tc.agent, tc.args, got, tc.want)
		}
		if !reflect.DeepEqual(s.Args, tc.args) {
			t.Fatal("compatibility changed caller argv")
		}
	}
	if strings.ContainsAny(CodexIndirectResumeCompatibilityNotice, "\r\n") || !strings.Contains(CodexIndirectResumeCompatibilityNotice, "not guarded") {
		t.Fatalf("notice=%q", CodexIndirectResumeCompatibilityNotice)
	}
}

func TestCodexCompatibilityLatestNeverBecomesExplicitIdentity(t *testing.T) {
	for _, args := range [][]string{{"resume", "--last"}, {"resume", "--last", "prompt"}} {
		if id := extractExplicitResume("codex", args); id != "" {
			t.Fatalf("latest became explicit admission key %q for %v", id, args)
		}
	}
}

func TestNativeResumeSelectorRetainsNewDestinationID(t *testing.T) {
	for _, agent := range []string{"claude", "qoder", "grok"} {
		forms := [][]string{{"--session-id", "destination"}, {"--session-id=destination"}}
		if agent == "grok" {
			forms = append(forms, []string{"-s", "destination"}, []string{"-s=destination"}, []string{"-sdestination"})
		}
		for _, args := range forms {
			for _, fork := range []bool{false, true} {
				t.Run(agent+"/"+strings.Join(args, " ")+"/"+map[bool]string{false: "new", true: "fork"}[fork], func(t *testing.T) {
					invocation := append([]string{}, args...)
					wantKind, source := NativeResumeNew, ""
					if fork {
						invocation = append(resumeToken(agent, "source"), append([]string{"--fork-session"}, invocation...)...)
						wantKind, source = NativeResumeFork, "source"
					}
					s, err := ParseNativeResumeSelector(agent, invocation)
					if err != nil || s.Kind != wantKind || s.NativeID != source || s.NewNativeID != "destination" {
						t.Fatalf("source/destination changed: %+v err=%v", s, err)
					}
					if !reflect.DeepEqual(s.Args, invocation) {
						t.Fatal("new destination changed invocation")
					}
				})
			}
		}
	}
}

func TestNativeResumeSelectorNewDestinationRespectsOptionAndPromptRoles(t *testing.T) {
	for _, agent := range []string{"claude", "qoder", "grok"} {
		for _, args := range [][]string{
			{"--model", "--session-id=literal"},
			{"--model", "--session-id", "literal"},
			{"--", "--session-id", "literal"},
			{"--", "--session-id=literal"},
		} {
			s, err := ParseNativeResumeSelector(agent, args)
			if err != nil || s.NewNativeID != "" || s.Kind != NativeResumeNone {
				t.Errorf("%s %q promoted a literal: %+v err=%v", agent, args, s, err)
			}
		}
	}
	for _, args := range [][]string{{"--model", "-sliteral"}, {"--", "-sliteral"}, {"--", "-s", "literal"}} {
		s, err := ParseNativeResumeSelector("grok", args)
		if err != nil || s.NewNativeID != "" || s.Kind != NativeResumeNone {
			t.Errorf("Grok %q promoted literal: %+v err=%v", args, s, err)
		}
	}
}
