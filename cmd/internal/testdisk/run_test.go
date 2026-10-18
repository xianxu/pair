package testdisk

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestMeasuredRun(t *testing.T) {
	for _, tt := range []struct {
		name        string
		supported   bool
		uncovered   string
		beforeError bool
		afterError  bool
		values      []uint64
		callback    int
		want        int
		reads       int
		message     string
	}{
		{"pass", true, "", false, false, []uint64{5, 105}, 0, 0, 2, "bytes=100 cap=100"},
		{"over", true, "", false, false, []uint64{5, 106}, 0, 1, 2, "exceeded"},
		{"failed tests", true, "", false, false, []uint64{5, 105}, 7, 7, 2, "bytes=100"},
		{"failed tests and over", true, "", false, false, []uint64{5, 106}, 7, 7, 2, "exceeded"},
		{"unsupported", false, "", false, false, nil, 0, 0, 0, "unsupported"},
		{"unsupported failed tests", false, "", false, false, nil, 9, 9, 0, "unsupported"},
		{"before failure", true, "", true, false, nil, 0, 1, 1, "before sample"},
		{"after failure", true, "", false, true, []uint64{5}, 0, 1, 2, "after sample"},
		{"counter regression", true, "", false, false, []uint64{5, 4}, 0, 1, 2, "decreased"},
		{"benchmark", true, "benchmark", false, false, nil, 0, 0, 0, "outside ordinary"},
		{"fuzz", true, "fuzz", false, false, nil, 0, 0, 0, "outside ordinary"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reads, runs := 0, 0
			read := func() (uint64, error) {
				reads++
				if tt.beforeError && reads == 1 || tt.afterError && reads == 2 {
					return 0, errors.New("counter unavailable")
				}
				if reads > len(tt.values) {
					t.Fatalf("unexpected read %d", reads)
				}
				return tt.values[reads-1], nil
			}
			var out bytes.Buffer
			options := runOptions{name: "owned", perRun: 100, repeats: 1, supported: tt.supported, uncovered: tt.uncovered}
			got := runMeasured(options, read, func() int { runs++; return tt.callback }, &out)
			if got != tt.want || runs != 1 || reads != tt.reads || !strings.Contains(out.String(), tt.message) {
				t.Fatalf("status=%d runs=%d reads=%d output=%q", got, runs, reads, out.String())
			}
		})
	}
}

func TestSupportedCounterDoesNotSilentlyBecomeUnsupported(t *testing.T) {
	var out bytes.Buffer
	opts := runOptions{name: "owned", perRun: 100, repeats: 1, supported: true}
	if got := runMeasured(opts, func() (uint64, error) { return 0, fmt.Errorf("unexpected native failure") }, func() int { return 0 }, &out); got != 1 {
		t.Fatalf("status=%d output=%s", got, out.String())
	}
}

func TestMeasuredRunPreservesPanic(t *testing.T) {
	defer func() {
		if recover() != "body panic" {
			t.Error("callback panic not preserved")
		}
	}()
	runMeasured(runOptions{name: "owned", perRun: 100, repeats: 1, supported: true}, func() (uint64, error) { return 0, nil }, func() int { panic("body panic") }, &bytes.Buffer{})
}
