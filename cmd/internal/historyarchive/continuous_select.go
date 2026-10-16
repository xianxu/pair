package historyarchive

import "time"

// Candidate is one discovered source file before archiving. Size and Mtime
// come from the walk; the content hash is computed only when the archived
// index cannot answer from them.
// pair:m5-concept pure
type Candidate struct {
	Path    string
	Size    int64
	Mtime   time.Time
	Regular bool
}

// SelectQuiet keeps regular files left unmodified for at least quiet: a file
// still being written is archived once it settles, not mid-change.
// pair:m5-concept pure
func SelectQuiet(cs []Candidate, now time.Time, quiet time.Duration) []Candidate {
	cutoff := now.Add(-quiet)
	out := []Candidate{}
	for _, c := range cs {
		if c.Regular && !c.Mtime.After(cutoff) {
			out = append(out, c)
		}
	}
	return out
}
