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

func TestExpandPeekReferences(t *testing.T) {
	for raw, want := range map[string][]string{
		"pair:3":                   {"pair:3"},
		"pair:1:2:3,ariadne:0:1:2": {"pair:1", "pair:2", "pair:3", "ariadne:0", "ariadne:1", "ariadne:2"},
		"pair:1, brain:0":          {"pair:1", "brain:0"},
		"my-thread":                {"my-thread"},
		"a:b:c":                    {"a:b:c"},
	} {
		got, err := ExpandPeekReferences(raw)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("%q = %q %v, want %q", raw, got, err, want)
		}
	}
	for _, raw := range []string{"pair:1,", ",pair:1", "pair:1,,pair:2"} {
		if _, err := ExpandPeekReferences(raw); err == nil {
			t.Errorf("%q accepted", raw)
		}
	}
}

// The live tail answers without touching the recording; when it fails, the
// recording answers and the live reason is named (pair#425).
func TestPeekPrefersTheLiveTail(t *testing.T) {
	env, record := peekEnv(t)
	recordingRead := false
	env.Couch.SlotTerminal = func(ThreadAddress, string, int) ([]string, error) {
		recordingRead = true
		return []string{"plain"}, nil
	}
	var asked int
	env.Couch.SlotTail = func(_ context.Context, address ThreadAddress, n int) (TerminalTail, error) {
		if address != record.Address {
			t.Fatalf("tail of %v", address)
		}
		asked = n
		return TerminalTail{Lines: []string{"old", "❯ ‹cursor›‹dim›Try it‹/dim›"}, Cursor: "2,3 default", Truncated: 4}, nil
	}
	r, err := env.Couch.PeekThread(context.Background(), "pair:1", record.Address, 1)
	if err != nil {
		t.Fatal(err)
	}
	if recordingRead || asked != 1 || r.Source != "live" || r.Cursor != "2,3 default" || r.Truncated != 4 ||
		!reflect.DeepEqual(r.Lines, []string{"❯ ‹cursor›‹dim›Try it‹/dim›"}) || len(r.Unavailable) != 0 {
		t.Fatalf("recording read %v asked %d: %+v", recordingRead, asked, r)
	}

	env.Couch.SlotTail = func(context.Context, ThreadAddress, int) (TerminalTail, error) {
		return TerminalTail{}, errors.New("no running couch answered")
	}
	r, err = env.Couch.PeekThread(context.Background(), "pair:1", record.Address, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !recordingRead || r.Source != "recording" || r.Cursor != "" || !reflect.DeepEqual(r.Lines, []string{"plain"}) ||
		!containsPrefix(r.Unavailable, "live tail: no running couch answered") {
		t.Fatalf("%+v", r)
	}
}

// A multi-slot peek is one snapshot in request order; a slot that does not
// resolve reports why in its own section, through the operation dispatcher.
func TestPeekSnapshotOverSeveralSlots(t *testing.T) {
	env, record := peekEnv(t)
	env.Couch.SlotTail = func(_ context.Context, address ThreadAddress, _ int) (TerminalTail, error) {
		return TerminalTail{Lines: []string{"tail of " + string(address.Tag)}}, nil
	}
	tag := string(record.Address.Tag)
	value, err := dispatchTestOperation(env.Couch, "peek", map[string]string{"repo-scope": record.Address.RepoScope, "ref": tag + ",nope," + tag, "lines": "5"})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, ok := value.(PeekSnapshot)
	if !ok || len(snapshot.Slots) != 3 {
		t.Fatalf("%#v", value)
	}
	for _, i := range []int{0, 2} {
		if s := snapshot.Slots[i]; s.Ref != tag || s.Source != "live" || s.Lines[0] != "tail of "+tag {
			t.Fatalf("slot %d: %+v", i, s)
		}
	}
	if s := snapshot.Slots[1]; s.Ref != "nope" || len(s.Unavailable) != 1 || s.Lines == nil {
		t.Fatalf("unresolved slot: %+v", s)
	}

	// One reference keeps the single-slot answer.
	value, err = dispatchTestOperation(env.Couch, "peek", map[string]string{"repo-scope": record.Address.RepoScope, "ref": tag})
	if _, ok := value.(PeekResult); err != nil || !ok {
		t.Fatalf("single: %#v %v", value, err)
	}
}
