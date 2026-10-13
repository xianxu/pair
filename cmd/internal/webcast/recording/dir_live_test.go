package recording

import (
	"os"
	"path/filepath"
	"testing"
)

// pair#493: a record is on disk when Put returns, not at Close. A recording is
// read while it is written (the live replay, the notes query), and buffered
// files held a cast's notes until it ended.
func TestDirWriterRecordIsOnDiskBeforeClose(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "rec")
	w, err := NewDirWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { w.Close() })
	b := NewBuilder("rec")
	p := b.El("p", []string{"id", "p"}, b.Txt("Hello"))
	b.Page(SegLive, "http://app/", p)
	for _, r := range b.Recs {
		if err := w.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Put(&Note{ID: "n1", Author: Author{ID: "ana"}, Text: "hi", At: NoteAt{Node: p.ID}}); err != nil {
		t.Fatal(err)
	}
	rec, err := ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := rec.Records[len(rec.Records)-1].(*Note); !ok || n.ID != "n1" {
		t.Fatalf("before Close, the last record read back is %#v", rec.Records[len(rec.Records)-1])
	}
}

// A reader can race a write and see a line half written (close review): the
// live reader decodes the complete lines, and the validator still names the
// torn tail.
func TestReadDirDecodesCompleteLinesOnly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "rec")
	w, err := NewDirWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	b := NewBuilder("rec")
	p := b.El("p", nil, b.Txt("Hello"))
	b.Page(SegLive, "http://app/", p)
	for _, r := range b.Recs {
		if err := w.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	w.Close()
	whole := len(b.Recs)
	f, err := os.OpenFile(filepath.Join(dir, FileEvents), os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`{"t":5,"k":"scr`) // a write in progress
	f.Close()

	rec, err := ReadDir(dir)
	if err != nil {
		t.Fatalf("a torn tail failed the whole recording: %v", err)
	}
	if len(rec.Records) != whole {
		t.Fatalf("%d records read, want the %d complete ones", len(rec.Records), whole)
	}
	if fs, _, err := ValidateDir(dir); err != nil || len(Errors(fs)) == 0 {
		t.Fatalf("the validator passed a torn tail: %v %v", fs, err)
	}
}
