package couchschedule

import (
	"strings"
	"testing"
	"time"
)

func TestSpecDefaultsAndExplicitFields(t *testing.T) {
	s, err := ParseTaskSpec([]byte(`{"version":1,"cron":"0 7 * * *","timeout":"10m"}`))
	if err != nil || s.Version != 1 || s.Cron != "0 7 * * *" || s.Timeout != 10*time.Minute || s.Scope != "machine" || s.Missed != "skip" || s.ScriptRun || s.Heavy {
		t.Fatalf("defaults: %+v, %v", s, err)
	}
	s, err = ParseTaskSpec([]byte(`{"version":1,"cron":"*/5 * * * *","timeout":"24h","scope":"team","owner":"xian-xu","missed":"once","script_run":true,"heavy":true}`))
	if err != nil || s.Scope != "team" || s.Owner != "xian-xu" || s.Missed != "once" || !s.ScriptRun || !s.Heavy || s.Timeout != 24*time.Hour {
		t.Fatalf("explicit: %+v, %v", s, err)
	}
}

func TestSpecRefusesInvalidDeclarations(t *testing.T) {
	base := `{"version":1,"cron":"0 7 * * *","timeout":"10m"}`
	bad := []string{"", "null", "[]", base + "{}", strings.Repeat(" ", 32769), string([]byte{0xff}),
		`{"version":1,"version":1,"cron":"* * * * *","timeout":"1m"}`,
		`{"version":1,"cron":"* * * * *"}`,
		`{"cron":"* * * * *","timeout":"1m"}`,
		`{"version":1,"timeout":"1m"}`,
	}
	for _, field := range []string{"version", "cron", "timeout"} {
		for _, value := range []string{"null", "[]", "{}", "false"} {
			start := map[string]string{"version": "1", "cron": `"0 7 * * *"`, "timeout": `"10m"`}
			bad = append(bad, strings.Replace(base, `"`+field+`":`+start[field], `"`+field+`":`+value, 1))
		}
	}
	for _, extra := range []string{`"unknown":true`, `"Scope":"team"`, `"scope":null`, `"scope":2`, `"scope":"fleet"`, `"scope":"team"`, `"scope":"team","owner":"../x"`, `"owner":""`, `"missed":"all"`, `"missed":null`, `"script_run":null`, `"script_run":"true"`, `"heavy":null`, `"heavy":1`} {
		bad = append(bad, strings.TrimSuffix(base, "}")+","+extra+"}")
	}
	for _, value := range []string{`"0s"`, `"-1m"`, `"24h1ns"`, `"forever"`, `10`} {
		bad = append(bad, strings.Replace(base, `"10m"`, value, 1))
	}
	for _, value := range []string{"0", "2", "1.0", `"1"`} {
		bad = append(bad, strings.Replace(base, `"version":1`, `"version":`+value, 1))
	}
	bad = append(bad, strings.Replace(base, "0 7 * * *", "@daily", 1))
	for _, raw := range bad {
		if s, err := ParseTaskSpec([]byte(raw)); err == nil {
			t.Errorf("accepted %q: %+v", raw, s)
		}
	}
}

func TestOwnerOperatorsAndAdmission(t *testing.T) {
	raw := "# Team\n## Operators\n\n- xian-xu (Xian Xu): maintainer\n- a-2 (A Two): backup\n### Notes\nSome prose.\n## Other\n- malformed\n"
	ops, err := ParseOperators([]byte(raw))
	if err != nil || strings.Join(ops, ",") != "xian-xu,a-2" {
		t.Fatalf("operators: %v %v", ops, err)
	}
	for _, raw := range []string{"", "# Team\n## Something\n- malformed"} {
		ops, err := ParseOperators([]byte(raw))
		if err != nil || len(ops) != 0 {
			t.Fatalf("absent: %v %v", ops, err)
		}
	}
	for _, raw := range []string{"## Operators\n- x (X): yes\n- x (Other): no", "## Operators\n- x: no", "## Operators\n* x (X): no", "## Operators\n- X (X): no", "## Operators\n- x (): no", "## Operators\n- x (X):", "## Operators\n## Other\n## Operators\n- x (X): yes", string([]byte{255})} {
		if _, err := ParseOperators([]byte(raw)); err == nil {
			t.Errorf("accepted malformed operators %q", raw)
		}
	}
	for _, tc := range []struct {
		spec    TaskSpec
		ops     []string
		current string
		allowed bool
		reason  string
	}{
		{TaskSpec{Scope: "machine"}, nil, "", true, ""},
		{TaskSpec{Scope: "team", Owner: "xian-xu"}, ops, "xian-xu", true, ""},
		{TaskSpec{Scope: "team", Owner: "xian-xu"}, ops, "a-2", false, "not-owner"},
		{TaskSpec{Scope: "team", Owner: "xian-xu"}, nil, "xian-xu", false, "owner-not-listed"},
		{TaskSpec{Scope: "team", Owner: "xian-xu"}, ops, "", false, "missing-operator"},
	} {
		allowed, reason := OwnerAdmission(tc.spec, tc.ops, tc.current)
		if allowed != tc.allowed || reason != tc.reason {
			t.Errorf("admission %+v: %v %q", tc, allowed, reason)
		}
	}
}

func TestOwnerSlugAndTaskNames(t *testing.T) {
	for input, want := range map[string]string{"Xian Xu": "xian-xu", "  A__B.. C  ": "a-b-c", "É / 中": "", "A--B": "a-b", "UPPER2": "upper2"} {
		if got := OperatorSlug(input); got != want {
			t.Errorf("slug(%q)=%q want %q", input, got, want)
		}
	}
	for _, name := range []string{"x", "task-2", "2", "a_2", "a-", "a--b", strings.Repeat("a", 128)} {
		if !ValidTaskName(name) {
			t.Errorf("refused %q", name)
		}
	}
	for _, name := range []string{"", ".", "..", "a/b", "a\\b", "/tmp", "-a", "_a", "A", "é", "a b", "a\n", "a\x00", strings.Repeat("a", 129)} {
		if ValidTaskName(name) {
			t.Errorf("accepted %q", name)
		}
	}
}

func FuzzSpecAndOperators(f *testing.F) {
	f.Add([]byte(`{"version":1,"cron":"* * * * *","timeout":"1m"}`))
	f.Add([]byte("## Operators\n- x (X): owner"))
	f.Fuzz(func(t *testing.T, raw []byte) {
		s, err := ParseTaskSpec(raw)
		if err == nil {
			if s.Timeout <= 0 || s.Timeout > 24*time.Hour {
				t.Fatalf("invalid accepted timeout: %+v", s)
			}
		}
		ops, err := ParseOperators(raw)
		if err == nil {
			seen := map[string]bool{}
			for _, op := range ops {
				if !ValidTaskName(op) || seen[op] {
					t.Fatalf("invalid accepted operators %v", ops)
				}
				seen[op] = true
			}
		}
	})
}
