package threadactivity

import (
	"context"
	"testing"
	"time"
)

type fakeRuntime struct {
	mtimes     map[string]time.Time
	transcript time.Time
	hasSession bool
	asked      []Thread
}

func (f *fakeRuntime) ModTime(path string) (time.Time, bool) {
	m, ok := f.mtimes[path]
	return m, ok
}

func (f *fakeRuntime) SessionActivity(_ context.Context, t Thread) (time.Time, bool) {
	f.asked = append(f.asked, t)
	return f.transcript, f.hasSession
}

var (
	base   = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	thread = Thread{ScopeDir: "/data/repos/scope", Scope: "scope", Tag: "work", Agent: "claude"}
)

const (
	logPath   = "/data/repos/scope/log-work.md"
	panePath  = "/data/repos/scope/pane-work-claude.json"
	draftPath = "/data/repos/scope/draft-work.md"
)

func TestLatestPicksTheNewestSignal(t *testing.T) {
	for name, tc := range map[string]struct {
		rt   *fakeRuntime
		want time.Time
	}{
		"transcript newest": {&fakeRuntime{mtimes: map[string]time.Time{logPath: base.Add(-time.Hour), panePath: base.Add(-2 * time.Hour)}, transcript: base, hasSession: true}, base},
		"log newest":        {&fakeRuntime{mtimes: map[string]time.Time{logPath: base, panePath: base.Add(-2 * time.Hour)}, transcript: base.Add(-time.Hour), hasSession: true}, base},
		"launch newest":     {&fakeRuntime{mtimes: map[string]time.Time{logPath: base.Add(-time.Hour), panePath: base}, transcript: base.Add(-time.Hour), hasSession: true}, base},
		// PQ-1: a session that has not been written to yet still has its
		// launch, so its age is never zero.
		"launch only": {&fakeRuntime{mtimes: map[string]time.Time{panePath: base}}, base},
		"nothing":     {&fakeRuntime{}, time.Time{}},
	} {
		if got := Latest(context.Background(), tc.rt, thread); !got.Equal(tc.want) {
			t.Errorf("%s: Latest = %v, want %v", name, got, tc.want)
		}
	}
}

// Draft autosave writes on every focus loss whether or not anything changed,
// so its mtime means "last left the draft", not input. It must never count.
func TestLatestIgnoresTheDraft(t *testing.T) {
	rt := &fakeRuntime{mtimes: map[string]time.Time{draftPath: base, panePath: base.Add(-48 * time.Hour)}}
	if got := Latest(context.Background(), rt, thread); !got.Equal(base.Add(-48 * time.Hour)) {
		t.Fatalf("Latest = %v, want the launch time; a fresher draft must not count", got)
	}
}

func TestLatestAsksTheSessionForThisThread(t *testing.T) {
	rt := &fakeRuntime{}
	Latest(context.Background(), rt, thread)
	if len(rt.asked) != 1 || rt.asked[0] != thread {
		t.Fatalf("session asked for %+v, want exactly %+v", rt.asked, thread)
	}
}

func TestLatestWithACanceledContextSkipsTheSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rt := &fakeRuntime{mtimes: map[string]time.Time{panePath: base}, transcript: base.Add(time.Hour), hasSession: true}
	if got := Latest(ctx, rt, thread); !got.Equal(base) {
		t.Fatalf("Latest = %v, want the file mtimes' %v", got, base)
	}
	if len(rt.asked) != 0 {
		t.Fatal("a canceled probe still queried the session inventory")
	}
}

func TestLatestWithAnUnresolvablePathStillAsksTheSession(t *testing.T) {
	rt := &fakeRuntime{transcript: base, hasSession: true}
	if got := Latest(context.Background(), rt, Thread{ScopeDir: "", Tag: "", Agent: "claude"}); !got.Equal(base) {
		t.Fatalf("Latest = %v, want the transcript %v", got, base)
	}
}

func TestInScopeResolvesTheRepositoryScopeDirectory(t *testing.T) {
	got, err := InScope("/data", "0123456789abcdef", "work", "codex")
	if err != nil {
		t.Fatal(err)
	}
	if got.ScopeDir != "/data/repos/0123456789abcdef" || got.Scope != "0123456789abcdef" || got.Tag != "work" || got.Agent != "codex" {
		t.Fatalf("InScope = %+v", got)
	}
	if _, err := InScope("/data", "../escape", "work", "codex"); err == nil {
		t.Fatal("an invalid scope resolved")
	}
}
