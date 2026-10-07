package couchcore

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/orientation"
)

func TestPeekTail(t *testing.T) {
	lines := []string{"a", "b", "c"}
	if got := peekTail(lines, 2); !reflect.DeepEqual(got, []string{"b", "c"}) {
		t.Fatal(got)
	}
	if got := peekTail(lines, 0); !reflect.DeepEqual(got, lines) {
		t.Fatal(got)
	}
	got := peekTail(lines, 5)
	got[0] = "changed"
	if lines[0] != "a" {
		t.Fatal("peekTail aliases its input")
	}
}

func peekEnv(t *testing.T) (*testEnv, ThreadRecord) {
	t.Helper()
	env := newTestEnv(t, "/repo")
	record := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Couch.SwitchContext = switchContextFunc(func(_ context.Context, r ThreadRecord) (orientation.OrientationContext, error) {
		return orientation.OrientationContext{Tag: string(r.Address.Tag), SourceAgent: "claude",
			PairLog: "/data/log.md", NativeTranscripts: []string{"/home/.claude/projects/x/s.jsonl"}}, nil
	})
	return env, record
}

// TestPeekShowsTheSlotsRecentTerminal: the rendered recording's tail, capped,
// with the envelope a coordinator searches for.
func TestPeekShowsTheSlotsRecentTerminal(t *testing.T) {
	env, record := peekEnv(t)
	var asked int
	env.Couch.SlotTerminal = func(address ThreadAddress, agent string, maxLines int) ([]string, error) {
		if address != record.Address || agent != "claude" {
			t.Fatalf("read %v %s", address, agent)
		}
		asked = maxLines
		return []string{"old", "[Couch peer from pair:0; delivery abc]", "please pick up pair#9", "❯ "}, nil
	}
	r, err := env.Couch.PeekThread(context.Background(), "pair:1", record.Address, 3)
	if err != nil {
		t.Fatal(err)
	}
	if asked != 3 || !reflect.DeepEqual(r.Lines, []string{"[Couch peer from pair:0; delivery abc]", "please pick up pair#9", "❯ "}) {
		t.Fatalf("asked %d, lines %q", asked, r.Lines)
	}
	if r.Ref != "pair:1" || r.Agent != "claude" || r.SentPrompts != "/data/log.md" || len(r.Transcripts) != 1 || len(r.Unavailable) != 0 {
		t.Fatalf("%+v", r)
	}
}

// TestPeekNamesEveryUnreadableSource: a failed read is named with its reason
// and never returned as an empty answer.
func TestPeekNamesEveryUnreadableSource(t *testing.T) {
	env, record := peekEnv(t)
	env.Couch.SlotTerminal = func(ThreadAddress, string, int) ([]string, error) {
		return nil, errors.New("read raw: no such file")
	}
	r, err := env.Couch.PeekThread(context.Background(), "pair:1", record.Address, 0)
	if err != nil {
		t.Fatal(err)
	}
	if r.Lines == nil || len(r.Lines) != 0 || !containsPrefix(r.Unavailable, "terminal recording: read raw: no such file") {
		t.Fatalf("%+v", r)
	}

	env.Couch.SwitchContext = switchContextFunc(func(context.Context, ThreadRecord) (orientation.OrientationContext, error) {
		return orientation.OrientationContext{Unavailable: []string{"native session has no exact established outgoing binding"}}, nil
	})
	r, err = env.Couch.PeekThread(context.Background(), "pair:1", record.Address, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !containsPrefix(r.Unavailable, "native session has no exact") || !containsPrefix(r.Unavailable, "terminal recording: the thread's agent is unknown") {
		t.Fatalf("%+v", r)
	}

	env.Couch.SwitchContext, env.Couch.SlotTerminal = nil, nil
	r, err = env.Couch.PeekThread(context.Background(), "pair:1", record.Address, 0)
	if err != nil || !containsPrefix(r.Unavailable, "transcript resolver is not configured") {
		t.Fatalf("%+v %v", r, err)
	}

	if _, err := env.Couch.PeekThread(context.Background(), "pair:9", ThreadAddress{RepoScope: record.Address.RepoScope, Tag: "missing"}, 0); err == nil {
		t.Fatal("an unknown thread was peeked")
	}
}

// TestPeekIsReadOnly: peeking changes no thread record.
func TestPeekIsReadOnly(t *testing.T) {
	env, record := peekEnv(t)
	env.Couch.SlotTerminal = func(ThreadAddress, string, int) ([]string, error) { return []string{"x"}, nil }
	before, err := env.Couch.Threads.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.PeekThread(context.Background(), "pair:1", record.Address, 0); err != nil {
		t.Fatal(err)
	}
	after, err := env.Couch.Threads.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("peek changed the thread store")
	}
}

func containsPrefix(list []string, prefix string) bool {
	for _, s := range list {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}
