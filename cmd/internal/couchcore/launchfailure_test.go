package couchcore

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func failureFixture(n int) LaunchFailure {
	return LaunchFailure{Address: ThreadAddress{RepoScope: "repo", Tag: "1-project"}, Nonce: fmt.Sprint(n), FailedAt: time.Unix(int64(n), 0), Agent: "codex", Stage: "registration", LaunchError: "launch failed"}
}
func TestLaunchFailureBoundsAndEnrichment(t *testing.T) {
	f := failureFixture(1)
	f.Output = bytes.Repeat([]byte{0xff}, 20000)
	f.LaunchError = strings.Repeat("x", 5000)
	code := 23
	f.BeforeExit = &code
	h, err := (LaunchFailureHistory{}).Upsert(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(h[0].Output) != 16384 || !bytes.Equal(h[0].Output, f.Output[len(f.Output)-16384:]) {
		t.Fatal("byte tail not preserved")
	}
	if len(h[0].LaunchError) > 4096 || !strings.Contains(h[0].LaunchError, "truncated") {
		t.Fatal("error not bounded with marker")
	}
	later := failureFixture(1)
	later.FailedAt = time.Unix(99, 0)
	later.FinalError = "source parked"
	later.LaunchError = ""
	later.AfterExit = &code
	h, err = h.Upsert(later)
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 1 || !h[0].FailedAt.Equal(f.FailedAt) || h[0].BeforeExit == nil || h[0].AfterExit == nil || h[0].FinalError != "source parked" || len(h[0].Output) != 16384 {
		t.Fatalf("lost enrichment: %+v", h)
	}
	*f.BeforeExit = 99
	f.Output[0] = 0
	if *h[0].BeforeExit != 23 {
		t.Fatal("aliased exit observation")
	}
}
func TestLaunchFailureNewestAttemptsSurviveLateUpdate(t *testing.T) {
	var h LaunchFailureHistory
	for n := 17; n >= 1; n-- {
		var err error
		h, err = h.Upsert(failureFixture(n))
		if err != nil {
			t.Fatal(err)
		}
	}
	h, err := h.Upsert(failureFixture(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 16 || h[0].Nonce != "17" || h[15].Nonce != "2" {
		t.Fatalf("bad retention %+v", h)
	}
	bad := failureFixture(18)
	bad.Nonce = strings.Repeat("x", 1025)
	if _, err = h.Upsert(bad); err == nil {
		t.Fatal("oversize identity accepted")
	}
}

func TestLaunchFailureCleanupEnrichesOutputWithoutInventingEarlyExit(t *testing.T) {
	f := failureFixture(1)
	f.Output = []byte("before")
	f.Operation = "launch"
	h, err := (LaunchFailureHistory{}).Upsert(f)
	if err != nil {
		t.Fatal(err)
	}
	late := f
	late.Output = []byte("before\nafter")
	code := 23
	late.BeforeExit = &code
	late.AfterExit = &code
	late.Operation = "switch-agent"
	h, err = h.Upsert(late)
	if err != nil {
		t.Fatal(err)
	}
	if string(h[0].Output) != "before\nafter" || h[0].BeforeExit != nil || h[0].AfterExit == nil || h[0].Operation != "switch-agent" {
		t.Fatalf("bad cleanup enrichment %+v", h[0])
	}
}

func TestLaunchFailureEncodedCeiling(t *testing.T) {
	var h LaunchFailureHistory
	for i := 1; i <= 16; i++ {
		f := failureFixture(i)
		f.StartingPath = strings.Repeat("\x00", 4096)
		f.Agent = f.StartingPath
		f.Stage = f.StartingPath
		f.Stream = f.StartingPath
		f.Operation = f.StartingPath
		next, err := h.Upsert(f)
		if err != nil {
			if !strings.Contains(err.Error(), "exceeds") {
				t.Fatal(err)
			}
			return
		}
		h = next
	}
	t.Fatal("encoded ceiling not enforced")
}
