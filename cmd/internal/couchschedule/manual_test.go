package couchschedule

import (
	"strings"
	"testing"
)

func TestManualIntentBoundsAndDigest(t *testing.T) {
	good := ManualIntent{ID: "run-1", Hub: "/repo/hub", Task: "check", Args: []string{"a", "b"}}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	same := good
	same.Args = append([]string(nil), good.Args...)
	if good.Digest() != same.Digest() {
		t.Fatal("unstable digest")
	}
	same.Args = []string{"ab"}
	if good.Digest() == same.Digest() {
		t.Fatal("argv collision")
	}
	same = good
	same.ID = "other"
	if good.Digest() != same.Digest() {
		t.Fatal("ID is not intent content")
	}
	for _, bad := range []ManualIntent{{ID: "../id", Hub: good.Hub, Task: good.Task}, {ID: good.ID, Hub: "relative", Task: good.Task}, {ID: good.ID, Hub: good.Hub, Task: "../task"}, {ID: good.ID, Hub: good.Hub, Task: good.Task, Args: []string{strings.Repeat("x", MaxManualArgsBytes+1)}}, {ID: good.ID, Hub: good.Hub, Task: good.Task, Args: []string{"a\x00b"}}} {
		if bad.Validate() == nil {
			t.Fatalf("accepted %#v", bad)
		}
	}
}
func TestManualStatusPageBounds(t *testing.T) {
	s := ManualStatus{ID: "run-1", Hub: "/repo/hub", Task: "check", Stage: "done", Total: MaxManualOutput, Output: make([]byte, MaxManualOutputPage)}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	if !s.Terminal() {
		t.Fatal("done is terminal")
	}
	s.Output = append(s.Output, 0)
	if s.Validate() == nil {
		t.Fatal("oversized page accepted")
	}
}
