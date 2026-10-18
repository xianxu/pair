package sessionsync

import "strings"

// Hunk is one changed region of a line diff: old lines [OldLo, OldHi) became
// new lines [NewLo, NewHi). An empty side is a pure insertion or deletion.
type Hunk struct {
	OldLo, OldHi, NewLo, NewHi int
}

// maxDiffEdits bounds Myers' search (pair#661, ARCH-CONSTRAINTS): its trace
// grows as D², so past this many line edits the whole changed middle becomes
// one hunk, keeping a 4 MiB rewrite cheap.
var maxDiffEdits = 1024

// splitLines splits s into lines that keep their newline; a final line
// without one is kept as it is, so a trailing newline is part of the text.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.SplitAfter(s, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// LineDiff returns the hunks that turn a into b, in order. Common leading and
// trailing lines are stripped first, then Myers finds a shortest edit script
// over the middle.
func LineDiff(a, b []string) []Hunk {
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre] == b[pre] {
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf] == b[len(b)-1-suf] {
		suf++
	}
	ma, mb := a[pre:len(a)-suf], b[pre:len(b)-suf]
	if len(ma) == 0 && len(mb) == 0 {
		return nil
	}
	hunks, ok := myers(ma, mb)
	if !ok {
		hunks = []Hunk{{0, len(ma), 0, len(mb)}}
	}
	for i := range hunks {
		hunks[i].OldLo += pre
		hunks[i].OldHi += pre
		hunks[i].NewLo += pre
		hunks[i].NewHi += pre
	}
	return hunks
}

// myers is the greedy O((N+M)D) shortest-edit-script search, keeping one
// frontier per edit count for the backtrack. It gives up past maxDiffEdits.
func myers(a, b []string) ([]Hunk, bool) {
	n, m := len(a), len(b)
	limit := min(n+m, maxDiffEdits)
	var trace [][]int
	v := map[int]int{1: 0}
	found := -1
	for d := 0; d <= limit && found < 0; d++ {
		frontier := make([]int, 2*d+1)
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[k-1] < v[k+1]) {
				x = v[k+1]
			} else {
				x = v[k-1] + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x, y = x+1, y+1
			}
			v[k] = x
			frontier[k+d] = x
			if x >= n && y >= m {
				found = d
			}
		}
		trace = append(trace, frontier)
	}
	if found < 0 {
		return nil, false
	}
	// Backtrack from (n, m): each edit step is one insertion or deletion
	// after which a diagonal of equal lines leads to (x, y).
	var hunks []Hunk
	x, y := n, m
	for d := found; d > 0; d-- {
		prev := trace[d-1]
		at := func(k int) int { return prev[k+d-1] }
		k := x - y
		pk := k - 1
		if k == -d || (k != d && at(k-1) < at(k+1)) {
			pk = k + 1
		}
		px := at(pk)
		py := px - pk
		if pk == k+1 {
			hunks = addHunk(hunks, Hunk{px, px, py, py + 1})
		} else {
			hunks = addHunk(hunks, Hunk{px, px + 1, py, py})
		}
		x, y = px, py
	}
	for i, j := 0, len(hunks)-1; i < j; i, j = i+1, j-1 {
		hunks[i], hunks[j] = hunks[j], hunks[i]
	}
	return hunks, true
}

// addHunk prepends one single-line move (backtracking runs end to start),
// merging it with the previous hunk when the two touch.
func addHunk(hunks []Hunk, h Hunk) []Hunk {
	if len(hunks) > 0 {
		last := &hunks[len(hunks)-1]
		if h.OldHi == last.OldLo && h.NewHi == last.NewLo {
			last.OldLo, last.NewLo = h.OldLo, h.NewLo
			return hunks
		}
	}
	return append(hunks, h)
}
