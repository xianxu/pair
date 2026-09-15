package terminalqualify

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestSplitInputsLiteralBytes(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  [][]string
	}{
		{"", [][]string{{""}}},
		{"A", [][]string{{"A"}}},
		{"\xc3\xa9X", [][]string{{"\xc3\xa9X"}, {"\xc3", "\xa9X"}, {"\xc3\xa9", "X"}, {"\xc3", "\xa9", "X"}}},
		{"\x1b[A", [][]string{{"\x1b[A"}, {"\x1b", "[A"}, {"\x1b[", "A"}, {"\x1b", "[", "A"}}},
	} {
		if got := SplitInputs(tc.input); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("input=%q: got %q want %q", tc.input, got, tc.want)
		}
	}
}

func assertBytePartitions(t *testing.T, input string, variants [][]string) {
	t.Helper()
	boundaries := make(map[int]bool)
	whole, singleBytes := false, false
	for _, chunks := range variants {
		if strings.Join(chunks, "") != input {
			t.Fatalf("bytes changed: %q != %q", chunks, input)
		}
		if len(chunks) == 1 {
			whole = true
		}
		if len(chunks) == 2 {
			boundaries[len(chunks[0])] = true
		}
		singles := len(chunks) == len(input)
		for _, chunk := range chunks {
			if len(chunk) != 1 {
				singles = false
			}
		}
		singleBytes = singleBytes || singles
	}
	if !whole || !singleBytes {
		t.Fatalf("missing whole/byte-at-time: whole=%t singles=%t", whole, singleBytes)
	}
	for boundary := 1; boundary < len(input); boundary++ {
		if !boundaries[boundary] {
			t.Fatalf("missing byte boundary %d", boundary)
		}
	}
}

func TestSplitInputsPreservesAllBytesAndBoundaries(t *testing.T) {
	raw := make([]byte, 256)
	for i := range raw {
		raw[i] = byte(i)
	}
	input := string(raw) + "é中👩‍💻\x1b[?1002h"
	assertBytePartitions(t, input, SplitInputs(input))
}

func TestRunCaseDeliversEveryBytePartition(t *testing.T) {
	input := "é中\x1b[?1002h"
	var delivered [][]string
	got, err := RunCase(context.Background(), Case{ID: "partition", Input: input, Split: true, Expected: Observation{"ok": "yes"}}, func(_ context.Context, _ Case, chunks []string) (Observation, error) {
		delivered = append(delivered, append([]string(nil), chunks...))
		return Observation{"ok": "yes"}, nil
	})
	if err != nil || got.Status != Pass {
		t.Fatalf("%+v %v", got, err)
	}
	assertBytePartitions(t, input, delivered)
}
