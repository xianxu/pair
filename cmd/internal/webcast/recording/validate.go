package recording

import "fmt"

// maxFindings caps a report: past it, the recording is broken enough that
// more lines say nothing new.
const maxFindings = 100

// Validate checks a recording given as its record files (name → bytes, any
// of FileEvents, FileNotes, FilePointer, FileFrames). It decodes every line,
// checks each record's fields and that it sits in its file, then reduces the
// merged stream and reports every record the reducer refuses, at its line.
// Kinds this version doesn't know are warnings. It also returns the final
// state, for callers that want to look further.
func Validate(files map[string][]byte) ([]Finding, *State) {
	var findings []Finding
	add := func(f Finding) {
		if len(findings) < maxFindings {
			findings = append(findings, f)
		}
	}
	located := map[string][]Located{}
	for name, data := range files {
		ls, bad := DecodeFile(name, data)
		for _, f := range bad {
			add(f)
		}
		// Merging sorts by t, so a file's own order is checked first.
		for i := 1; i < len(ls); i++ {
			if ls[i].Time() < ls[i-1].Time() {
				add(Finding{File: name, Line: ls[i].Line, Msg: fmt.Sprintf("t=%d is before the previous record's t=%d", ls[i].Time(), ls[i-1].Time())})
			}
		}
		located[name] = ls
	}
	if ev := located[FileEvents]; len(ev) == 0 || ev[0].Kind() != "head" {
		add(Finding{File: FileEvents, Line: 1, Msg: "the first record must be head"})
		return findings, NewState()
	}
	s := NewState()
	s.whole = true
	for _, l := range Merge(located) {
		if !Known(l.Kind()) {
			add(Finding{File: l.File, Line: l.Line, Warning: true, Msg: fmt.Sprintf("unknown kind %q (ignored)", l.Kind())})
			continue
		}
		if err := s.Apply(l.Record); err != nil {
			add(Finding{File: l.File, Line: l.Line, Msg: fmt.Sprintf("%s: %v", l.Kind(), err)})
		}
	}
	for _, l := range located[FileFrames] {
		if err := l.check(); err != nil {
			add(Finding{File: l.File, Line: l.Line, Msg: err.Error()})
		}
	}
	return findings, s
}

// Errors keeps the findings that aren't warnings.
func Errors(fs []Finding) []Finding {
	var out []Finding
	for _, f := range fs {
		if !f.Warning {
			out = append(out, f)
		}
	}
	return out
}
