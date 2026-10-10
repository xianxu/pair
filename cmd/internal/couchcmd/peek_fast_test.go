package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// The broker reads a tail by slot, as a send names one, and says whose it
// is, so a peek needs no store (pair#429).
func TestTailBySlotReadsThatSlotsWrapper(t *testing.T) {
	b := couchmessage.Binding{Slot: "pair:1", Scope: "scope", Tag: "tag", Agent: "codex", Session: "secret", Nonce: "nonce", PID: 7}
	other := couchmessage.Binding{Slot: "pair:2", Scope: "scope", Tag: "other", PID: 8}
	endpoint := &fakeTailEndpoint{tail: couchmessage.Tail{Lines: []string{"❯ "}}}
	s := &messageService{}
	s.authority.endpoint = func(got couchmessage.Binding) couchmessage.DeliveryEndpoint {
		if got != b {
			t.Fatalf("endpoint for %+v", got)
		}
		return endpoint
	}
	s.authority.families = func(context.Context) (map[string]string, error) { return map[string]string{"pair": "p"}, nil }
	connected := map[couchmessage.Binding]bool{b: true, other: true}
	s.connected.Store(&connected)
	got := s.handle(context.Background(), couchmessage.Request{Op: "tail", Target: "p:1", Lines: 5})
	if got.Code != "ok" || got.Tail == nil || endpoint.lines != 5 ||
		got.TailThread == nil || *got.TailThread != (couchmessage.TailThread{Slot: "pair:1", Tag: "tag", Agent: "codex"}) {
		t.Fatalf("%+v", got)
	}
	if got := s.handle(context.Background(), couchmessage.Request{Op: "tail", Target: "pair:9", Lines: 5}); got.Code != "unavailable" {
		t.Fatalf("missing slot: %+v", got)
	}
	s.authority.families = func(context.Context) (map[string]string, error) { return nil, errors.New("store locked") }
	if got := s.handle(context.Background(), couchmessage.Request{Op: "tail", Target: "pair:1", Lines: 5}); got.Code != "unavailable" || !strings.Contains(got.Error, "store locked") {
		t.Fatalf("families error: %+v", got)
	}
}

// fakeTails answers tail requests by slot and records them.
type fakeTails struct {
	mu    sync.Mutex
	asked []string
	fail  map[string]bool
	// old answers as a Couch from before the by-slot form does.
	old bool
}

func (f *fakeTails) call(_ context.Context, _ string, request any, response any) error {
	r := request.(couchmessage.Request)
	f.mu.Lock()
	f.asked = append(f.asked, r.Target)
	f.mu.Unlock()
	if couchmessage.ValidateRequest(r) != nil || r.TailScope != "" {
		return errors.New("bad request")
	}
	if f.old {
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "invalid-request", Error: "tail takes only a thread and a line count"}
		return nil
	}
	if f.fail[r.Target] {
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "unavailable", Error: "no wrapper"}
		return nil
	}
	*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok",
		Tail:       &couchmessage.Tail{Lines: []string{"tail of " + r.Target}, Cursor: &couchmessage.TailCursor{Row: 1, Col: 3, Shape: "bar"}},
		TailThread: &couchmessage.TailThread{Slot: r.Target, Tag: "tag-" + r.Target, Agent: "claude"}}
	return nil
}

// fastPeek answers only when every slot's live tail does; anything else
// leaves the typed peek to answer, with nothing written.
func TestFastPeekAnswersOnlyWhenEverySlotIsLive(t *testing.T) {
	peek := func(ref string, args []string, tails *fakeTails) (bool, string) {
		var out bytes.Buffer
		ok := fastPeek(cliInvocation{kind: cliPeek, ref: ref, args: args}, t.TempDir(), &out, tails.call)
		return ok, out.String()
	}
	tails := &fakeTails{}
	ok, out := peek("pair:1:2,ops:0", []string{"--json", "--lines=7"}, tails)
	var snapshot couchcore.PeekSnapshot
	if !ok || json.Unmarshal([]byte(out), &snapshot) != nil || len(snapshot.Slots) != 3 {
		t.Fatalf("%v %q", ok, out)
	}
	for i, ref := range []string{"pair:1", "pair:2", "ops:0"} {
		if s := snapshot.Slots[i]; s.Ref != ref || s.Tag != "tag-"+ref || s.Agent != "claude" || s.Source != "live" || s.Cursor != "1,3 bar" || s.Lines[0] != "tail of "+ref {
			t.Fatalf("slot %d: %+v", i, s)
		}
	}

	// One slot keeps the single-slot shape, as text.
	ok, out = peek("pair:1", nil, &fakeTails{})
	if !ok || !strings.HasPrefix(out, "peek pair:1  agent claude  tag tag-pair:1  source live\n") {
		t.Fatalf("%v %q", ok, out)
	}

	for name, tc := range map[string]struct {
		ref    string
		args   []string
		fail   map[string]bool
		old    bool
		called bool
	}{
		"a slot fails":     {"pair:1:2", nil, map[string]bool{"pair:2": true}, false, true},
		"an older couch":   {"pair:1", nil, nil, true, true},
		"not a slot":       {"pair:1,couch-0102030405060708", nil, nil, false, false},
		"transcripts":      {"pair:1", []string{"--transcripts"}, nil, false, false},
		"over a live tail": {"pair:1", []string{"--lines=" + "201"}, nil, false, false},
		"bad lines":        {"pair:1", []string{"--lines=x"}, nil, false, false},
	} {
		tails := &fakeTails{fail: tc.fail, old: tc.old}
		ok, out := peek(tc.ref, tc.args, tails)
		if ok || out != "" || (len(tails.asked) > 0) != tc.called {
			t.Errorf("%s: answered %v, wrote %q, asked %q", name, ok, out, tails.asked)
		}
	}
}

// noCouchRT fails any attempt to build a Couch: the fast path must not.
type noCouchRT struct{ testRT }

func (r noCouchRT) NewCouchWith(couchcore.Runner, couchcore.CouchNamespace) (*couchcore.Couch, error) {
	return nil, errors.New("the fast peek built a Couch")
}

// From argv, over a real broker socket: a slot peek is answered by the
// running Couch without building one (pair#429).
func TestPeekAnswersFromTheRunningCouch(t *testing.T) {
	rt := noCouchRT{newRT(t, "/repo")}
	socket, err := couchmessage.SocketPath(rt.StoreDir(), "broker")
	if err != nil {
		t.Fatal(err)
	}
	tails := &fakeTails{}
	server, err := couchmessage.StartServer(context.Background(), socket, func(ctx context.Context, raw []byte) ([]byte, error) {
		var request couchmessage.Request
		if err := strictjson.Decode(raw, &request); err != nil {
			return nil, err
		}
		var response couchmessage.Response
		if err := tails.call(ctx, socket, request, &response); err != nil {
			return nil, err
		}
		return json.Marshal(response)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	var out, errw bytes.Buffer
	if code := RunWithRuntime([]string{"--peek", "pair:1:2", "--lines", "3"}, strings.NewReader(""), &out, &errw, rt); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errw.String())
	}
	if !strings.Contains(out.String(), "peek pair:1  agent claude  tag tag-pair:1  source live") || !strings.Contains(out.String(), "tail of pair:2") {
		t.Fatalf("%s", out.String())
	}
}

// A fast peek answers in the typed peek's shape (pair#429): the same thread
// peeked both ways gives the same JSON, agent and working path included.
func TestFastPeekMatchesTheTypedPeek(t *testing.T) {
	rt := newRT(t, "/repo")
	record := seedVerifiedPark(t, rt, "/repo")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	tail := couchmessage.Tail{Lines: []string{"❯ ‹cursor›"}, Cursor: &couchmessage.TailCursor{Row: 1, Col: 3, Shape: "bar"}, Truncated: 2}
	c.SlotTail = func(context.Context, couchcore.ThreadAddress, int) (couchcore.TerminalTail, error) {
		return couchcore.TerminalTail{Lines: tail.Lines, Cursor: tail.Cursor.String(), Truncated: tail.Truncated}, nil
	}
	typed, err := c.PeekThread(context.Background(), "pair:1", record.Address, 10, false)
	if err != nil {
		t.Fatal(err)
	}

	b := couchmessage.Binding{Slot: "pair:1", Scope: record.Address.RepoScope, Tag: string(record.Address.Tag), Agent: "wrapper-basename", PID: 7}
	s := &messageService{}
	s.authority.endpoint = func(couchmessage.Binding) couchmessage.DeliveryEndpoint { return &fakeTailEndpoint{tail: tail} }
	s.authority.record = c.Threads.GetThread
	connected := map[couchmessage.Binding]bool{b: true}
	s.connected.Store(&connected)
	call := func(ctx context.Context, _ string, request any, response any) error {
		*response.(*couchmessage.Response) = s.handle(ctx, request.(couchmessage.Request))
		return nil
	}
	var out bytes.Buffer
	if !fastPeek(cliInvocation{kind: cliPeek, ref: "pair:1", args: []string{"--json", "--lines=10"}}, t.TempDir(), &out, call) {
		t.Fatal("fast peek did not answer")
	}
	want, err := json.Marshal(typed)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != string(want) || typed.WorkingPath == "" || typed.Agent != "claude" {
		t.Fatalf("fast:  %s\ntyped: %s", out.String(), want)
	}
}
