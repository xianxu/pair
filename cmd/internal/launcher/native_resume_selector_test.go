package launcher

import (
	"reflect"
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
