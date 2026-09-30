package main

import (
	"fmt"
	"strings"
	"testing"
)

// compare must name a hole as a hole, and a send still in flight as incomplete
// rather than as a loss or a pass (pair#208's rule for probes).
func TestCompareReadsHolesAndInFlightSends(t *testing.T) {
	want := payload(3, 2447)
	framed := func(body []byte) []byte { return []byte(fmt.Sprintf("<<B03>>%s<<E03>>", body)) }
	if r := compare(framed(want), 3, want); !r.complete || !r.exact {
		t.Fatalf("whole send: %+v", r)
	}
	// Where the cut lands is ambiguous within a line (neighbouring numbered
	// lines share most bytes); how much is missing is not.
	holed := append(append([]byte{}, want[:952]...), want[1977:]...)
	r := compare(framed(holed), 3, want)
	if !r.complete || r.exact || !strings.Contains(r.detail, "(missing 1025, extra 0)") {
		t.Fatalf("#211's hole: %+v", r)
	}
	if r := compare([]byte("<<B03>>"+string(want[:100])), 3, want); r.complete {
		t.Fatalf("in-flight send read as finished: %+v", r)
	}
	if len(payload(0, 1025)) != 1025 {
		t.Fatal("payload must be exactly the requested size")
	}
}
