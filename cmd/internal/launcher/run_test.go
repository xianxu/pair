package launcher

import (
	"errors"
	"testing"
	"time"
)

type fakeSessions struct {
	sessions []Session
	err      error
}

func (f fakeSessions) Snapshot() ([]Session, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.sessions, nil
}

type fakeHistory struct {
	tags []HistoricalTag
	err  error
}

func (f fakeHistory) Scan(base string, cutoff time.Time) ([]HistoricalTag, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.tags, nil
}

func TestRunLaunchUsesSuppliedEnvironment(t *testing.T) {
	outcome, err := Run([]string{"codex"}, Env{
		Home:     "/home/me",
		Cwd:      "/work/pair",
		Now:      time.Unix(1000, 0),
		HistoryD: 14,
	}, fakeSessions{}, fakeHistory{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if outcome.Decision.Action != ActionCreate || outcome.Decision.Tag != "pair" || !outcome.Decision.PromptName {
		t.Fatalf("Decision = %#v, want create pair with prompt", outcome.Decision)
	}
	if outcome.Env.DataDir != "/home/me/.local/share/pair" {
		t.Fatalf("DataDir = %q, want home-derived data dir", outcome.Env.DataDir)
	}
}

func TestRunLaunchTurnsFakeSessionsIntoPickerDecision(t *testing.T) {
	outcome, err := Run([]string{"claude"}, Env{
		Home:     "/home/me",
		Cwd:      "/work/pair",
		Now:      time.Unix(1000, 0),
		HistoryD: 14,
	}, fakeSessions{sessions: []Session{{Name: "pair-demo", State: SessionDetached}}}, fakeHistory{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if outcome.Decision.Action != ActionPick {
		t.Fatalf("Decision = %#v, want picker", outcome.Decision)
	}
}

func TestRunLaunchTurnsFakeHistoryIntoPickerDecision(t *testing.T) {
	outcome, err := Run([]string{"claude"}, Env{
		Home:     "/home/me",
		Cwd:      "/work/pair",
		Now:      time.Unix(1000, 0),
		HistoryD: 14,
	}, fakeSessions{}, fakeHistory{tags: []HistoricalTag{{Tag: "pair-old"}}})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if outcome.Decision.Action != ActionPick {
		t.Fatalf("Decision = %#v, want picker", outcome.Decision)
	}
}

func TestRunLaunchReturnsTypedUsageError(t *testing.T) {
	_, err := Run([]string{"codex", "extra"}, Env{Home: "/home/me", Cwd: "/work/pair"}, fakeSessions{}, fakeHistory{})
	if err == nil {
		t.Fatal("Run returned nil error")
	}
	var usage UsageError
	if !errors.As(err, &usage) {
		t.Fatalf("error = %T, want UsageError", err)
	}
}
