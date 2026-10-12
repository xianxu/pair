package recording

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// filesOf encodes records into their files.
func filesOf(t *testing.T, recs []Record) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	for name, rs := range Split(recs) {
		var buf bytes.Buffer
		for _, r := range rs {
			line, err := Encode(r)
			if err != nil {
				t.Fatal(err)
			}
			buf.Write(line)
			buf.WriteByte('\n')
		}
		out[name] = buf.Bytes()
	}
	return out
}

func TestValidateGoodRecording(t *testing.T) {
	p := newPage()
	p.b.Add(&Note{ID: "n", Author: Author{ID: "a"}, At: NoteAt{Node: p.div.ID}})
	p.b.Add(&Stroke{ID: "s", Author: Author{ID: "a"}, Pts: [][3]float64{{0, 0, 0}}})
	p.b.Add(&Stop{Why: StopEnd})
	fs, st := Validate(filesOf(t, p.b.Recs))
	if len(fs) != 0 {
		t.Fatalf("findings: %v", fs)
	}
	if len(st.Notes) != 1 || st.Live {
		t.Fatalf("final state: %+v", st)
	}
}

func TestValidateFindings(t *testing.T) {
	type tc struct {
		name  string
		edit  func(files map[string][]byte)
		file  string
		line  int
		match string
	}
	appendLine := func(name, line string) func(map[string][]byte) {
		return func(f map[string][]byte) { f[name] = append(f[name], []byte(line+"\n")...) }
	}
	for _, c := range []tc{
		{"head not first", func(f map[string][]byte) {
			f[FileEvents] = append([]byte(`{"t":0,"k":"title","v":"x"}`+"\n"), f[FileEvents]...)
		}, FileEvents, 1, "first record must be head"},
		{"bad version", func(f map[string][]byte) {
			f[FileEvents] = bytes.Replace(f[FileEvents], []byte(`"v":1`), []byte(`"v":9`), 1)
		}, FileEvents, 1, "version 9"},
		{"time goes back", appendLine(FileEvents, `{"t":-1,"k":"title","v":"x"}`), FileEvents, 0, "before the previous"},
		{"missing field", appendLine(FileEvents, `{"t":0,"k":"attr","id":3}`), FileEvents, 0, "missing name"},
		{"dangling reference", appendLine(FileEvents, `{"t":0,"k":"rm","id":4242}`), FileEvents, 0, "node 4242 is unknown"},
		{"tag change", appendLine(FileEvents, `{"t":0,"k":"rm","id":3}`+"\n"+`{"t":0,"k":"ins","p":7,"after":0,"node":{"id":3,"y":"el","n":"p"}}`), FileEvents, 0, "names one thing"},
		{"wrong file", appendLine(FileEvents, `{"t":0,"k":"note","id":"n","author":{"id":"a"},"at":{"node":3}}`), FileEvents, 0, "belongs in notes.jsonl"},
		{"not json", appendLine(FileNotes, `{nope`), FileNotes, 1, "invalid"},
		{"note-state for no note", appendLine(FileNotes, `{"t":0,"k":"note-state","id":"x","state":"open"}`), FileNotes, 1, "unknown"},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := newPage()
			files := filesOf(t, p.b.Recs)
			c.edit(files)
			fs, _ := Validate(files)
			errs := Errors(fs)
			if len(errs) == 0 {
				t.Fatalf("no finding")
			}
			f := errs[0]
			if f.File != c.file || !strings.Contains(f.Msg, c.match) || (c.line != 0 && f.Line != c.line) {
				t.Fatalf("got %v, want %s:%d ~ %q", f, c.file, c.line, c.match)
			}
			if c.line == 0 && f.Line != strings.Count(string(files[c.file]), "\n") {
				t.Fatalf("finding at line %d, want the last line (%d)", f.Line, strings.Count(string(files[c.file]), "\n"))
			}
		})
	}
}

func TestUnknownKindIsAWarning(t *testing.T) {
	p := newPage()
	files := filesOf(t, p.b.Recs)
	files[FileEvents] = append(files[FileEvents], []byte(`{"t":0,"k":"wiggle","id":3}`+"\n")...)
	fs, _ := Validate(files)
	if len(fs) != 1 || !fs[0].Warning || len(Errors(fs)) != 0 {
		t.Fatalf("findings %v", fs)
	}
}

func TestMergeOrder(t *testing.T) {
	mk := func(file string, line int, r Record) Located { return Located{Record: r, File: file, Line: line} }
	n := &Note{Header: Header{T: 5}, ID: "n"}
	s := &Stroke{Header: Header{T: 5}, ID: "s"}
	e1 := &Title{Header: Header{T: 5}, V: "1"}
	e2 := &Title{Header: Header{T: 5}, V: "2"}
	e0 := &Title{Header: Header{T: 1}, V: "0"}
	got := Records(Merge(map[string][]Located{
		FilePointer: {mk(FilePointer, 1, s)},
		FileNotes:   {mk(FileNotes, 1, n)},
		FileEvents:  {mk(FileEvents, 1, e0), mk(FileEvents, 2, e1), mk(FileEvents, 3, e2)},
	}))
	want := []Record{e0, e1, e2, n, s}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("position %d: got %#v", i, got[i])
		}
	}
}

func TestDirRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "rec")
	w, err := NewDirWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	p := newPage()
	body := []byte("body{color:red}")
	hash, err := w.PutAsset(body)
	if err != nil {
		t.Fatal(err)
	}
	p.b.Add(&Asset{URL: "http://x/a.css", Hash: hash, MIME: "text/css", Size: int64(len(body))})
	p.b.Add(&Note{ID: "n", Author: Author{ID: "a"}, At: NoteAt{Node: p.div.ID}})
	for _, r := range p.b.Recs {
		if err := w.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.PutFrame(&Frame{File: "frames/0.png", W: 2, H: 2, Scale: 1}, []byte("png")); err != nil {
		t.Fatal(err)
	}
	if err := w.PutFrame(&Frame{File: "../x.png", W: 2, H: 2}, nil); err == nil {
		t.Fatal("a frame outside frames/ must be refused")
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	rec, err := ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Records) != len(p.b.Recs) || len(rec.Frames) != 1 {
		t.Fatalf("read %d records, %d frames", len(rec.Records), len(rec.Frames))
	}
	if got, _ := rec.Asset(hash); !bytes.Equal(got, body) {
		t.Fatal("asset body")
	}
	if _, err := rec.Asset("../events.jsonl"); err == nil {
		t.Fatal("an asset name that isn't a hash must be refused")
	}
	fs, _, err := ValidateDir(dir)
	if err != nil || len(fs) != 0 {
		t.Fatalf("ValidateDir: %v %v", fs, err)
	}
	// Tamper: a body that no longer hashes to its name, a stray file, a lost frame.
	os.WriteFile(filepath.Join(dir, AssetName(hash)), []byte("changed"), 0o600)
	os.WriteFile(filepath.Join(dir, "assets", "stray"), nil, 0o600)
	os.Remove(filepath.Join(dir, "frames/0.png"))
	fs, _, _ = ValidateDir(dir)
	var msgs []string
	for _, f := range fs {
		msgs = append(msgs, f.String())
	}
	all := strings.Join(msgs, "\n")
	for _, want := range []string{"does not hash", "not a content hash", "missing frames/0.png"} {
		if !strings.Contains(all, want) {
			t.Errorf("ValidateDir lacks %q:\n%s", want, all)
		}
	}
	if again, err := w.PutAsset(body); err != nil || again != hash {
		t.Errorf("a stored body put again: %s %v", again, err)
	}
	if left, _ := filepath.Glob(filepath.Join(dir, "assets", ".tmp-*")); len(left) != 0 {
		t.Errorf("temporary files left: %v", left)
	}
	if _, err := NewDirWriter(dir); err == nil {
		t.Error("a recording directory is written once")
	}
}
