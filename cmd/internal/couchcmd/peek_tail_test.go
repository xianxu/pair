package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

type fakeTailEndpoint struct {
	couchmessage.DeliveryEndpoint
	tail  couchmessage.Tail
	err   error
	lines int
}

func (f *fakeTailEndpoint) Tail(_ context.Context, n int) (couchmessage.Tail, error) {
	f.lines = n
	return f.tail, f.err
}

// The service reads a tail from the one connected wrapper running the thread,
// for a caller with no slot identity (pair#425).
func TestTailReadsTheThreadsConnectedWrapper(t *testing.T) {
	b := couchmessage.Binding{Slot: "pair:1", Scope: "scope", Tag: "tag", PID: 7}
	other := couchmessage.Binding{Slot: "pair:2", Scope: "scope", Tag: "other", PID: 8}
	endpoint := &fakeTailEndpoint{tail: couchmessage.Tail{Lines: []string{"‹dim›hi‹/dim›"}}}
	s := &messageService{}
	s.authority.endpoint = func(got couchmessage.Binding) couchmessage.DeliveryEndpoint {
		if got != b {
			t.Fatalf("endpoint for %+v", got)
		}
		return endpoint
	}
	connected := map[couchmessage.Binding]bool{b: true, other: true}
	s.connected.Store(&connected)
	request := couchmessage.Request{Op: "tail", TailScope: "scope", TailTag: "tag", Lines: 7}
	got := s.handle(context.Background(), request)
	if got.Code != "ok" || got.Tail == nil || got.Tail.Lines[0] != "‹dim›hi‹/dim›" || endpoint.lines != 7 {
		t.Fatalf("tail: %+v (lines %d)", got, endpoint.lines)
	}

	endpoint.err = errors.New("wrapper gone")
	if got := s.handle(context.Background(), request); got.Code != "unavailable" || got.Error != "wrapper gone" {
		t.Fatalf("endpoint error: %+v", got)
	}
	request.TailTag = "absent"
	if got := s.handle(context.Background(), request); got.Code != "unavailable" {
		t.Fatalf("no wrapper: %+v", got)
	}
	request.Lines = 0
	if got := s.handle(context.Background(), request); got.Code != "invalid-request" {
		t.Fatalf("invalid: %+v", got)
	}
}

func TestRenderLivePeekAndSnapshot(t *testing.T) {
	live := couchcore.PeekResult{Ref: "pair:1", Tag: "t1", Agent: "claude", Source: "live",
		Lines: []string{"❯ ‹cursor›‹dim›Try it‹/dim›"}, Cursor: "1,3 default", Truncated: 2}
	unread := couchcore.PeekResult{Ref: "pair:9", Lines: []string{}, Unavailable: []string{"workspace pair:9 has no readable current thread"}}
	var out bytes.Buffer
	if code := render(&out, couchcore.Operation{Name: "peek"}, couchcore.PeekSnapshot{Slots: []couchcore.PeekResult{live, unread}}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	want := "peek pair:1  agent claude  tag t1  source live\n--- recent terminal ---\n❯ ‹cursor›‹dim›Try it‹/dim›\n---\n" +
		"cursor: 1,3 default\ntruncated: 2 older lines\n\n" +
		"peek pair:9  agent unknown  tag   source none\n--- recent terminal ---\n---\nunavailable: workspace pair:9 has no readable current thread\n"
	if out.String() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out.String(), want)
	}
}

// The peek's broker request names the thread and no caller; each answer
// shape maps to a tail or a reason.
func TestReadSlotTail(t *testing.T) {
	address := couchcore.ThreadAddress{RepoScope: "scope", Tag: "tag"}
	answer := func(r couchmessage.Response, err error) messageCall {
		return func(_ context.Context, _ string, request any, response any) error {
			got := request.(couchmessage.Request)
			if got.Op != "tail" || got.TailScope != "scope" || got.TailTag != "tag" || got.Lines != 30 || couchmessage.ValidateRequest(got) != nil {
				t.Fatalf("request %+v", got)
			}
			*response.(*couchmessage.Response) = r
			return err
		}
	}
	ok := couchmessage.Response{Code: "ok", Tail: &couchmessage.Tail{Lines: []string{"x"}, Cursor: &couchmessage.TailCursor{Row: 1, Col: 2, Shape: "bar"}, Truncated: 1}}
	tail, err := readSlotTail(context.Background(), answer(ok, nil), "/store", address, 30)
	if err != nil || tail.Cursor != "1,2 bar" || tail.Truncated != 1 || tail.Lines[0] != "x" {
		t.Fatalf("%+v %v", tail, err)
	}
	for _, tc := range []struct {
		r    couchmessage.Response
		want string
	}{
		{couchmessage.Response{Code: "invalid-request", Error: "operation requires the calling conversation identity"}, "predates live tails"},
		{couchmessage.Response{Code: "invalid-request", Error: `json: unknown field "TailScope"`}, "predates live tails"},
		{couchmessage.Response{Code: "unavailable", Error: "no wrapper"}, "unavailable: no wrapper"},
		{couchmessage.Response{Code: "ok"}, "returned no tail"},
	} {
		if _, err := readSlotTail(context.Background(), answer(tc.r, nil), "/store", address, 30); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%+v: %v, want %q", tc.r, err, tc.want)
		}
	}
	// More than a live tail holds is refused before any call, so the peek
	// falls back to the recording, which honours the count.
	refused := func(context.Context, string, any, any) error { t.Fatal("called"); return nil }
	if _, err := readSlotTail(context.Background(), refused, "/store", address, couchmessage.MaxTailLines+1); err == nil {
		t.Error("over-long tail accepted")
	}
	if _, err := readSlotTail(context.Background(), answer(couchmessage.Response{}, errors.New("dial: refused")), "/store", address, 30); err == nil || !strings.Contains(err.Error(), "no running couch answered") {
		t.Errorf("transport: %v", err)
	}
}

// The whole router, from argv: a multi-slot reference must not be validated
// as one repo:N before the executor expands it (the first live run refused
// "pair:0:3,ops:0" exactly there).
func TestMultiSlotPeekRunsThroughTheRouter(t *testing.T) {
	rt := newRT(t, "/repo")
	seedThread(t, rt, "/repo")
	var out, errw bytes.Buffer
	ref := "couch-0102030405060708,pair:1:2,couch-0102030405060708"
	if code := RunWithRuntime([]string{"--peek", ref, "--lines", "3", "--json"}, strings.NewReader(""), &out, &errw, rt); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errw.String())
	}
	var snapshot couchcore.PeekSnapshot
	if err := json.Unmarshal(out.Bytes(), &snapshot); err != nil {
		t.Fatalf("%v: %s", err, out.String())
	}
	var refs []string
	for _, slot := range snapshot.Slots {
		refs = append(refs, slot.Ref)
	}
	if strings.Join(refs, " ") != "couch-0102030405060708 pair:1 pair:2 couch-0102030405060708" {
		t.Fatalf("sections %q", refs)
	}
	if snapshot.Slots[0].Tag != "couch-0102030405060708" || len(snapshot.Slots[1].Unavailable) == 0 {
		t.Fatalf("%+v", snapshot.Slots)
	}
}
