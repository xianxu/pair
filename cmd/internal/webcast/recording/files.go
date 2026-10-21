package recording

import (
	"bytes"
	"fmt"
	"math"
	"slices"
)

// Located is a record with where it was read from.
type Located struct {
	Record
	File string
	Line int // 1-based
}

// mergeOrder: at equal t, events first, then notes, then pointer strokes.
var mergeOrder = []string{FileEvents, FileNotes, FilePointer}

// Finding is one problem in a recording, at a file and line (0: the
// recording as a whole).
type Finding struct {
	File    string
	Line    int
	Warning bool
	Msg     string
}

func (f Finding) String() string {
	level := "error"
	if f.Warning {
		level = "warning"
	}
	if f.Line == 0 {
		return fmt.Sprintf("%s: %s: %s", f.File, level, f.Msg)
	}
	return fmt.Sprintf("%s:%d: %s: %s", f.File, f.Line, level, f.Msg)
}

// DecodeFile reads one record file. A line that doesn't decode, or a kind
// that belongs to another file, is a finding; blank lines are skipped.
func DecodeFile(name string, data []byte) ([]Located, []Finding) {
	var out []Located
	var bad []Finding
	for i, line := range bytes.Split(data, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		r, err := Decode(line)
		if err != nil {
			bad = append(bad, Finding{File: name, Line: i + 1, Msg: err.Error()})
			continue
		}
		if want := FileOf(r.Kind()); want != name {
			bad = append(bad, Finding{File: name, Line: i + 1, Msg: fmt.Sprintf("a %s record belongs in %s", r.Kind(), want)})
			continue
		}
		out = append(out, Located{Record: r, File: name, Line: i + 1})
	}
	return out, bad
}

// Merge makes one stream of a recording's record files: by t, and at equal
// t in file order (events, notes, pointer), each file keeping its own order.
func Merge(files map[string][]Located) []Located {
	var all []Located
	for _, name := range mergeOrder {
		all = append(all, files[name]...)
	}
	slices.SortStableFunc(all, func(a, b Located) int {
		if a.Time() != b.Time() {
			return cmpInt(a.Time(), b.Time())
		}
		return fileRank(a.File) - fileRank(b.File)
	})
	return all
}

// OrderedTime stamps a generated live record so splitting and merging its
// files preserves emission order. Within a millisecond Merge orders files;
// crossing backward in that order must begin a new millisecond. Raw readers
// still validate supplied timestamps directly, without rewriting history.
// Sketch sources have strict fences on both sides so their timestamp names
// exactly the preceding source prefix. Clears strictly follow annotations.
// An impossible fence refuses rather than wrapping the timeline.
func OrderedTime(t, lastTime int64, lastKind, kind string) (int64, error) {
	t = max(t, lastTime)
	fence := kind == "sketch-begin" || lastKind == "sketch-begin" || kind == "annotation-clear" || fileRank(FileOf(kind)) < fileRank(FileOf(lastKind))
	if t == lastTime && lastKind != "" && fence {
		if t == math.MaxInt64 {
			return 0, fmt.Errorf("recording timestamp overflow before %s", kind)
		}
		t++
	}
	return t, nil
}

func fileRank(file string) int { return slices.Index(mergeOrder, file) }

func cmpInt(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// Records strips locations.
func Records(ls []Located) []Record {
	out := make([]Record, len(ls))
	for i, l := range ls {
		out[i] = l.Record
	}
	return out
}

// Split groups records by the file each kind lives in, keeping order.
func Split(recs []Record) map[string][]Record {
	out := map[string][]Record{}
	for _, r := range recs {
		f := FileOf(r.Kind())
		out[f] = append(out[f], r)
	}
	return out
}
