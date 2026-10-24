package agentfixture

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestWorkloadValidation(t *testing.T) {
	for _, raw := range []string{
		`{"id":"x","kind":"state","state":"working","workload":{"action":"start","scrollback_lines":1}}`,
		`{"id":"x","kind":"state","state":"idle","workload":{"action":"start"}}`,
		`{"id":"x","kind":"exec","workload":{"action":"start"}}`,
		`{"id":"x","kind":"state","state":"working","workload":{"action":"stop","scrollback_lines":2000}}`,
		`{"id":"x","kind":"state","state":"working","workload":{"action":"unknown"}}`,
	} {
		if _, err := DecodeCommand([]byte(raw)); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	for _, lines := range []int{0, 2000} {
		if err := ValidateCommand(Command{ID: "x", Kind: SetState, State: Working, Workload: &WorkloadRequest{Action: WorkloadStart, ScrollbackLines: lines}}); err != nil {
			t.Fatal(err)
		}
	}
}
func TestWorkloadBoundedReduction(t *testing.T) {
	start := time.Unix(1000, 0)
	for _, count := range []int{100, 200} {
		s, _, err := StepWorkload(WorkloadState{}, WorkloadEvent{Kind: WorkloadBegin, At: start, Request: WorkloadRequest{Action: WorkloadStart, ScrollbackLines: 2000}})
		if err != nil {
			t.Fatal(err)
		}
		prefill := WorkloadPrefill(2000)
		if strings.Count(prefill, "\r\n") != 2000 || len(prefill) > 256<<10 {
			t.Fatal("prefill bound")
		}
		s, _, err = StepWorkload(s, WorkloadEvent{Kind: WorkloadWritten, Bytes: uint64(len(prefill))})
		for i := 1; i <= count; i++ {
			var effect bool
			s, effect, err = StepWorkload(s, WorkloadEvent{Kind: WorkloadTick, At: start.Add(time.Duration(i) * 100 * time.Millisecond)})
			if err != nil || !effect {
				t.Fatal("tick missing", err)
			}
			patch := WorkloadPatch(s.Observation.Ticks)
			if len(patch) > 128 || strings.Contains(patch, "[2J") {
				t.Fatal("unbounded repaint")
			}
			s, _, err = StepWorkload(s, WorkloadEvent{Kind: WorkloadWritten, Tick: true, Bytes: uint64(len(patch))})
		}
		if s.Observation.Ticks != uint64(count) || s.Observation.Bytes > uint64(len(prefill)+count*128) {
			t.Fatal(s)
		}
		before := s
		if _, _, err = StepWorkload(s, WorkloadEvent{Kind: WorkloadBegin, At: start, Request: WorkloadRequest{Action: WorkloadStart}}); err == nil {
			t.Fatal("replacement accepted")
		}
		s, _, _ = StepWorkload(before, WorkloadEvent{Kind: WorkloadEnd, At: start.Add(time.Minute)})
		s, effect, err := StepWorkload(s, WorkloadEvent{Kind: WorkloadTick, At: start.Add(2 * time.Minute)})
		if effect || err != nil || s.Observation.Ticks != uint64(count) {
			t.Fatal("late tick")
		}
	}
	s, _, _ := StepWorkload(WorkloadState{}, WorkloadEvent{Kind: WorkloadBegin, At: start, Request: WorkloadRequest{Action: WorkloadStart}})
	s, effect, _ := StepWorkload(s, WorkloadEvent{Kind: WorkloadTick, At: start.Add(15 * time.Minute)})
	if effect || s.Observation.Status != WorkloadExpired {
		t.Fatal("expiry", s)
	}
}

func TestWorkloadAdversarialEvents(t *testing.T) {
	start := time.Unix(1000, 0)
	for trial := uint64(1); trial <= 32; trial++ {
		seed := trial
		s, _, _ := StepWorkload(WorkloadState{}, WorkloadEvent{Kind: WorkloadBegin, At: start, Request: WorkloadRequest{Action: WorkloadStart}})
		s, _, _ = StepWorkload(s, WorkloadEvent{Kind: WorkloadWritten})
		var committed uint64
		now := start
		for i := 0; i < 200; i++ {
			seed = seed*6364136223846793005 + 1
			now = now.Add(time.Duration(seed%150) * time.Millisecond)
			var effect bool
			s, effect, _ = StepWorkload(s, WorkloadEvent{Kind: WorkloadTick, At: now})
			if effect {
				s, _, _ = StepWorkload(s, WorkloadEvent{Kind: WorkloadWritten, Tick: true, Bytes: 12})
				committed++
			}
			// Duplicate completion must never inflate counters.
			previous := s.Observation
			next, _, err := StepWorkload(s, WorkloadEvent{Kind: WorkloadWritten, Tick: true, Bytes: 12})
			if err == nil && next.Observation != previous {
				t.Fatal("unowned completion changed counters")
			}
			if s.Observation.Ticks != committed || s.Observation.Ticks > uint64(now.Sub(start)/WorkloadInterval) {
				t.Fatal("rate/count bound", s)
			}
		}
	}
}

func TestWorkloadFailureAndByteCeilings(t *testing.T) {
	start := time.Unix(1000, 0)
	s, _, _ := StepWorkload(WorkloadState{}, WorkloadEvent{Kind: WorkloadBegin, At: start, Request: WorkloadRequest{Action: WorkloadStart}})
	if _, _, err := StepWorkload(s, WorkloadEvent{Kind: WorkloadWritten, Bytes: 256<<10 + 1}); err == nil {
		t.Fatal("overbound initial output accepted")
	}
	s, _, _ = StepWorkload(s, WorkloadEvent{Kind: WorkloadWritten, Bytes: 100})
	s, _, _ = StepWorkload(s, WorkloadEvent{Kind: WorkloadTick, At: start.Add(WorkloadInterval)})
	if _, _, err := StepWorkload(s, WorkloadEvent{Kind: WorkloadWritten, Tick: true, Bytes: 129}); err == nil {
		t.Fatal("overbound tick output accepted")
	}
	first := errors.New("partial output")
	s, _, _ = StepWorkload(s, WorkloadEvent{Kind: WorkloadFailed, At: start.Add(WorkloadInterval), Err: first})
	s, _, _ = StepWorkload(s, WorkloadEvent{Kind: WorkloadFailed, Err: errors.New("later")})
	s, _, _ = StepWorkload(s, WorkloadEvent{Kind: WorkloadWritten, Tick: true, Bytes: 12})
	if s.Failure != first || s.Observation.Ticks != 0 || s.Observation.Bytes != 100 {
		t.Fatal("failed output fabricated completion", s)
	}
}

func TestWorkloadPrefillGeometry(t *testing.T) {
	prefill := WorkloadPrefill(2000)
	if len(prefill) != 240000 {
		t.Fatalf("prefill bytes=%d want240000", len(prefill))
	}
	lines := strings.Split(prefill, "\r\n")
	if len(lines) != 2001 || lines[2000] != "" {
		t.Fatal("prefill CRLF framing")
	}
	for i, line := range lines[:2000] {
		if len(line) != 118 || !strings.HasPrefix(line, fmt.Sprintf("fixture scrollback %04d", i)) {
			t.Fatalf("line%d width=%d", i, len(line))
		}
		for _, r := range line {
			if r < 32 || r > 126 {
				t.Fatalf("line%d non-printable column", i)
			}
		}
	}
	for _, agent := range []string{"claude", "codex"} {
		initial := prefill + fmt.Sprintf("\x1b[%d;1H", 38) + Paint(AgentState{Agent: agent, Phase: Working}, 120, 38)
		if len(initial) > 256<<10 {
			t.Fatalf("%s initial bytes=%d", agent, len(initial))
		}
	}
}
