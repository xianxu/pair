package couchmessage

import (
	"strings"
	"testing"
)

func TestTailEndpointValidation(t *testing.T) {
	b := protocolBinding("pair:1")
	for _, tc := range []struct {
		name string
		r    EndpointRequest
		ok   bool
	}{
		{"one line", EndpointRequest{Op: "tail", Binding: b, Lines: 1}, true},
		{"max", EndpointRequest{Op: "tail", Binding: b, Lines: MaxTailLines}, true},
		{"zero", EndpointRequest{Op: "tail", Binding: b}, false},
		{"over max", EndpointRequest{Op: "tail", Binding: b, Lines: MaxTailLines + 1}, false},
		{"with ID", EndpointRequest{Op: "tail", Binding: b, Lines: 5, ID: "x"}, false},
		{"no binding", EndpointRequest{Op: "tail", Lines: 5}, false},
		{"lines on observe", EndpointRequest{Op: "observe", Binding: b, Lines: 5}, false},
	} {
		if err := ValidateEndpointRequest(tc.r); (err == nil) != tc.ok {
			t.Errorf("%s: err %v, want ok=%v", tc.name, err, tc.ok)
		}
	}
}

func TestBoundTailDropsOldestLinesAndMovesCursor(t *testing.T) {
	line := strings.Repeat("x", 1000)
	var lines []string
	for range 200 {
		lines = append(lines, line)
	}
	got := BoundTail(Tail{Lines: lines, Cursor: &TailCursor{Row: 200, Col: 3}})
	size := 0
	for _, l := range got.Lines {
		size += len(l) + 1
	}
	if size > MaxTailBytes || got.Truncated == 0 || len(got.Lines)+got.Truncated != 200 {
		t.Fatalf("size %d truncated %d kept %d", size, got.Truncated, len(got.Lines))
	}
	if got.Cursor.Row != 200-got.Truncated {
		t.Fatalf("cursor row %d after dropping %d", got.Cursor.Row, got.Truncated)
	}
	one := BoundTail(Tail{Lines: []string{strings.Repeat("y", MaxTailBytes+10)}})
	if len(one.Lines) != 1 || len(one.Lines[0]) != MaxTailBytes {
		t.Fatalf("single over-long line: %d lines", len(one.Lines))
	}
}

func TestTailCursorString(t *testing.T) {
	for _, tc := range []struct {
		c    TailCursor
		want string
	}{
		{TailCursor{Row: 3, Col: 5, Shape: "bar"}, "3,5 bar"},
		{TailCursor{Row: 3, Col: 5, Shape: "block", Steady: true}, "3,5 block steady"},
		{TailCursor{Row: 3, Col: 5, Hidden: true}, "hidden at 3,5"},
		{TailCursor{Col: 5, Shape: "bar"}, "outside the tail"},
	} {
		if got := tc.c.String(); got != tc.want {
			t.Errorf("%+v = %q, want %q", tc.c, got, tc.want)
		}
	}
}

func TestTailRequestIsIdentityFree(t *testing.T) {
	ok := Request{Op: "tail", TailScope: "scope", TailTag: "tag", Lines: 10}
	if err := ValidateRequest(ok); err != nil {
		t.Fatal(err)
	}
	b := protocolBinding("pair:1")
	for name, r := range map[string]Request{
		"no thread":       {Op: "tail", Lines: 10},
		"no lines":        {Op: "tail", TailScope: "scope", TailTag: "tag"},
		"too many lines":  {Op: "tail", TailScope: "scope", TailTag: "tag", Lines: MaxTailLines + 1},
		"caller identity": {Op: "tail", TailScope: "scope", TailTag: "tag", Lines: 10, Scope: "s", Tag: "t", Session: "x", Nonce: "n"},
		"binding":         {Op: "tail", TailScope: "scope", TailTag: "tag", Lines: 10, Binding: &b},
		"tail on send":    {Op: "send", Scope: "s", Tag: "t", Session: "x", Nonce: "n", ID: "id", Target: "pair", Body: "hi", TailScope: "scope"},
		"lines on actors": {Op: "actors", Scope: "s", Tag: "t", Session: "x", Nonce: "n", Lines: 3},
		"over-long tag":   {Op: "tail", TailScope: "scope", TailTag: strings.Repeat("t", MaxBindingBytes+1), Lines: 10},
	} {
		if ValidateRequest(r) == nil {
			t.Errorf("%s accepted", name)
		}
	}
}
