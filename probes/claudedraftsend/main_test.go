package main

import "testing"

// Claude wraps pasted runs in <pasted_content> tags and inserts blank lines at
// their boundaries — sometimes inside a numbered line. Neither is a loss; the
// first version of this probe read that split as a lost line.
func TestLostIgnoresClaudesPasteBoundariesButNotAHole(t *testing.T) {
	want := payload(1, 2447)
	split := want[:1024] + "\n</pasted_content id=\"ab\">\n\n<pasted_content id=\"cd\">" + want[1024:]
	if n, _ := lost(split, want); n != 0 {
		t.Fatalf("boundary split read as %d lost bytes", n)
	}
	if n, _ := lost(want, want); n != 0 {
		t.Fatalf("whole send read as %d lost bytes", n)
	}
	holed := want[:1024] + want[2048:]
	if n, _ := lost(holed, want); n < 900 {
		t.Fatalf("a dropped ~1 KiB read reported as %d lost bytes", n)
	}
}
