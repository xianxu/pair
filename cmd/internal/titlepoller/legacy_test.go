package titlepoller

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func legacyTestPolicy() LegacyPolicy {
	return LegacyPolicy{Executable: "/pair/bin/pair", DataRoot: "/data", Cutoff: LegacyCutoff{Platform: "darwin", Start: "200.0"}}
}
func legacyTestTitle(pid int, tag string) procutil.Observation {
	return procutil.Observation{Process: procutil.Process{PID: pid, PPID: 1, Identity: "100.1", Name: "pair"}, CommandInfo: procutil.CommandInfo{Executable: "/pair/bin/pair", Args: []string{"/pair/bin/pair", "title", tag, "codex", "session"}}}
}
func legacyTestSample(rows ...procutil.Observation) LegacySample {
	return LegacySample{Complete: true, Platform: "darwin", At: time.Unix(300, 0), Rows: rows}
}
func legacyTestWrapper(path string) procutil.Observation {
	return procutil.Observation{Process: procutil.Process{PID: 99, PPID: 7, Identity: "110.0", Name: "pair"}, CommandInfo: procutil.CommandInfo{Executable: "/pair/bin/pair", Args: []string{"/pair/bin/pair", "wrap", "--scrollback-log", path, "--from-launch-env"}}}
}
func TestDecideLegacyTitlesUsesExactWrapperOwner(t *testing.T) {
	for _, tc := range []struct {
		name    string
		wrapper []procutil.Observation
		want    LegacyDisposition
	}{
		{"orphan", nil, LegacyEligible},
		{"prefix collision", []procutil.Observation{legacyTestWrapper("/data/scrollback-work-10-codex.raw")}, LegacyEligible},
		{"exact", []procutil.Observation{legacyTestWrapper("/data/scrollback-work-1-codex.raw")}, LegacyKeep},
		{"same tag other agent", []procutil.Observation{legacyTestWrapper("/data/scrollback-work-1-claude.raw")}, LegacyKeep},
		{"scoped exact", []procutil.Observation{legacyTestWrapper("/data/repos/0123456789abcdef/scrollback-work-1-codex.raw")}, LegacyKeep},
		{"unparseable wrapper", []procutil.Observation{legacyTestWrapper("/other/scrollback-work-1-codex.raw")}, LegacyUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sample := legacyTestSample(append([]procutil.Observation{legacyTestTitle(42, "work-1")}, tc.wrapper...)...)
			got := DecideLegacyTitles(sample, legacyTestPolicy())
			if len(got) != 1 || got[0].Disposition != tc.want {
				t.Fatalf("decisions=%+v want %s", got, tc.want)
			}
		})
	}
}
func TestDecideLegacyTitlesKeepsUncertainEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*LegacySample, *LegacyPolicy)
		want   LegacyDisposition
	}{
		{"incomplete", func(s *LegacySample, _ *LegacyPolicy) { s.Complete = false }, LegacyUnknown},
		{"live parent", func(s *LegacySample, _ *LegacyPolicy) { s.Rows[0].PPID = 12 }, LegacyKeep},
		{"new generation", func(s *LegacySample, _ *LegacyPolicy) { s.Rows[0].Identity = "200.0" }, LegacyKeep},
		{"microsecond ordering", func(s *LegacySample, p *LegacyPolicy) { s.Rows[0].Identity = "100.10"; p.Cutoff.Start = "100.9" }, LegacyKeep},
		{"malformed identity", func(s *LegacySample, _ *LegacyPolicy) { s.Rows[0].Identity = "100.1000000" }, LegacyUnknown},
		{"missing identity", func(s *LegacySample, _ *LegacyPolicy) { s.Rows[0].Identity = "" }, LegacyUnknown},
		{"unreadable possible wrapper", func(s *LegacySample, _ *LegacyPolicy) {
			s.Rows = append(s.Rows, procutil.Observation{Process: procutil.Process{PID: 99, Name: "pair"}, Err: errors.New("denied")})
		}, LegacyUnknown},
		{"unreadable unknown name", func(s *LegacySample, _ *LegacyPolicy) {
			s.Rows = append(s.Rows, procutil.Observation{Process: procutil.Process{PID: 99}, Err: errors.New("denied")})
		}, LegacyUnknown},
		{"foreign known name", func(s *LegacySample, _ *LegacyPolicy) {
			s.Rows = append(s.Rows, procutil.Observation{Process: procutil.Process{PID: 99, Name: "postgres"}, Err: errors.New("denied")})
		}, LegacyEligible},
		{"invalid cutoff", func(_ *LegacySample, p *LegacyPolicy) { p.Cutoff.Start = "not a stamp" }, LegacyUnknown},
		{"boot mismatch", func(s *LegacySample, p *LegacyPolicy) {
			s.Platform = "linux"
			s.BootID = "boot-a"
			s.Rows[0].Identity = "100"
			p.Cutoff = LegacyCutoff{Platform: "linux", BootID: "boot-b", Start: "200"}
		}, LegacyUnknown},
		{"linux ticks", func(s *LegacySample, p *LegacyPolicy) {
			s.Platform = "linux"
			s.BootID = "boot-a"
			s.Rows[0].Identity = "100"
			p.Cutoff = LegacyCutoff{Platform: "linux", BootID: "boot-a", Start: "200"}
		}, LegacyEligible},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, p := legacyTestSample(legacyTestTitle(42, "work")), legacyTestPolicy()
			tc.mutate(&s, &p)
			got := DecideLegacyTitles(s, p)
			if len(got) != 1 || got[0].Disposition != tc.want {
				t.Fatalf("decisions=%+v want %s", got, tc.want)
			}
		})
	}
}
func TestDecideLegacyTitlesRefusesNearMatches(t *testing.T) {
	for _, args := range [][]string{{"/pair/bin/pair", "title", "work", "codex"}, {"/pair/bin/pair", "title", "work", "codex", "session", "extra"}, {"/pair/bin/pair", "subtitle", "work", "codex", "session"}} {
		row := legacyTestTitle(42, "work")
		row.Args = args
		if got := DecideLegacyTitles(legacyTestSample(row), legacyTestPolicy()); len(got) != 0 {
			t.Fatalf("near match reported: %+v", got)
		}
	}
	row := legacyTestTitle(42, "work")
	row.Executable = "/unrelated/pair"
	if got := DecideLegacyTitles(legacyTestSample(row), legacyTestPolicy()); len(got) != 0 {
		t.Fatalf("foreign executable reported: %+v", got)
	}
}

type legacyCleanupFake struct {
	identities []string
	events     []string
	sleepErr   error
}

func (f *legacyCleanupFake) Identity(pid int) (string, error) {
	f.events = append(f.events, "identity")
	if len(f.identities) == 0 {
		return "", errors.New("unknown")
	}
	s := f.identities[0]
	f.identities = f.identities[1:]
	return s, nil
}
func (f *legacyCleanupFake) Signal(pid int, sig syscall.Signal) error {
	if sig == syscall.SIGTERM {
		f.events = append(f.events, "TERM")
	} else {
		f.events = append(f.events, "KILL")
	}
	return nil
}
func (f *legacyCleanupFake) Sleep(context.Context, time.Duration) error {
	f.events = append(f.events, "wait")
	return f.sleepErr
}
func TestCleanupLegacyTitleRechecksEverySignal(t *testing.T) {
	candidate := LegacyTitleDecision{PID: 42, Start: "100.1", Disposition: LegacyEligible}
	for _, tc := range []struct {
		name string
		ids  []string
		want []string
	}{
		{"unchanged", []string{"100.1", "100.1"}, []string{"identity", "TERM", "wait", "identity", "KILL"}},
		{"reused before term", []string{"200.0"}, []string{"identity"}},
		{"reused after term", []string{"100.1", "200.0"}, []string{"identity", "TERM", "wait", "identity"}},
		{"unknown after term", []string{"100.1"}, []string{"identity", "TERM", "wait", "identity"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &legacyCleanupFake{identities: tc.ids}
			CleanupLegacyTitle(context.Background(), candidate, fake, time.Millisecond)
			if !reflect.DeepEqual(fake.events, tc.want) {
				t.Fatalf("effects=%v want %v", fake.events, tc.want)
			}
		})
	}
	for _, status := range []LegacyDisposition{LegacyKeep, LegacyUnknown} {
		fake := &legacyCleanupFake{}
		candidate.Disposition = status
		CleanupLegacyTitle(context.Background(), candidate, fake, time.Millisecond)
		if len(fake.events) > 0 {
			t.Fatalf("kept decision had effects %v", fake.events)
		}
	}
}

func TestCleanupApprovedLegacyTitlesResamplesAfterApproval(t *testing.T) {
	policy := legacyTestPolicy()
	approved := DecideLegacyTitles(legacyTestSample(legacyTestTitle(42, "work")), policy)
	fresh := legacyTestSample(legacyTestTitle(42, "work"), legacyTestWrapper("/data/scrollback-work-codex.raw"))
	calls := 0
	fake := &legacyCleanupFake{identities: []string{"100.1", "100.1"}}
	results, err := CleanupApprovedLegacyTitles(context.Background(), approved, policy, func(context.Context) (LegacySample, error) { calls++; return fresh, nil }, fake, time.Millisecond)
	if err != nil || calls != 1 || len(results) != 1 || len(fake.events) != 0 {
		t.Fatalf("new wrapper not preserved: results=%+v err=%v samples=%d effects=%v", results, err, calls, fake.events)
	}
	fresh = legacyTestSample(legacyTestTitle(42, "other"))
	calls = 0
	_, err = CleanupApprovedLegacyTitles(context.Background(), approved, policy, func(context.Context) (LegacySample, error) { calls++; return fresh, nil }, fake, time.Millisecond)
	if err != nil || calls != 1 || len(fake.events) != 0 {
		t.Fatalf("approval widened to another tag: %v %v", err, fake.events)
	}
}
