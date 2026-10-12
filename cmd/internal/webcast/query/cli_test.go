package query

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func runMain(argv ...string) (code int, out, errOut string) {
	var o, e bytes.Buffer
	code = Run(argv, nil, &o, &e)
	return code, o.String(), e.String()
}

func TestMainRefusesBadArguments(t *testing.T) {
	dir := recording.Fixture("rec-app")
	for _, argv := range [][]string{
		{},
		{dir},
		{dir, "page"},
		{dir, "events", "--from", "1"},
		{dir, "events", "--from", "x", "--to", "2"},
		{dir, "dom", "--at", "-5"},
		{dir, "dom", "--from", "1"},
		{dir, "dom", "--json", "--json"},
		{dir, "notes", "--before"},
		{dir, "dom", "--max-lines", "0"},
		{dir, "dom", "--json", "--max-lines", "5"},
		{dir, "events", "--from", "0", "--to", "1", "--max-lines", "5", "--json"},
	} {
		code, _, errOut := runMain(argv...)
		if code != 2 || !strings.Contains(errOut, "usage:") {
			t.Errorf("%q: exit %d, %q", argv, code, errOut)
		}
	}
}

func TestMainFailsWhereTheRecordingCantAnswer(t *testing.T) {
	dir := recording.Fixture("rec-app")
	broken := t.TempDir()
	data, err := os.ReadFile(filepath.Join(dir, recording.FileEvents))
	if err != nil {
		t.Fatal(err)
	}
	// A change to a node that never existed: the reducer refuses it.
	bad := append(data, []byte(`{"t":99999,"k":"rm","id":999999}`+"\n")...)
	if err := os.WriteFile(filepath.Join(broken, recording.FileEvents), bad, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		argv []string
		want string
	}{
		{[]string{t.TempDir(), "dom"}, "holds no records"},
		{[]string{filepath.Join(dir, "nope"), "dom"}, "holds no records"},
		{[]string{dir, "dom", "--sel", "#nope"}, `nothing matches "#nope"`},
		{[]string{dir, "dom", "--sel", "a:hover"}, "selectors are"},
		{[]string{dir, "events", "--from", "9", "--to", "2"}, "is empty"},
		{[]string{dir, "notes", "--note", "zz"}, `no note "zz"`},
		{[]string{broken, "notes"}, "doesn't read: record"},
	} {
		code, _, errOut := runMain(c.argv...)
		if code != 1 || !strings.Contains(errOut, c.want) || strings.Contains(errOut, "query: query:") {
			t.Errorf("%q: exit %d, %q; want %q", c.argv, code, errOut, c.want)
		}
	}
}

// --json answers decode, and carry what the text leaves out.
func TestMainJSON(t *testing.T) {
	dir := recording.Fixture("rec-app")
	code, out, errOut := runMain(dir, "dom", "--at", "6000", "--sel", "#search", "--json")
	var dom struct {
		T       int64
		URL     string
		Matches []struct {
			ID    int64
			A     map[string]string
			State *struct{ V *string }
		}
	}
	if code != 0 || json.Unmarshal([]byte(out), &dom) != nil || len(dom.Matches) != 1 || dom.Matches[0].A["class"] == "" || dom.Matches[0].State == nil {
		t.Errorf("dom --json: %d %s %s", code, out, errOut)
	}

	code, out, _ = runMain(dir, "events", "--from", "4780", "--to", "4920", "--json")
	ls := strings.Split(strings.TrimSpace(out), "\n")
	for _, l := range ls {
		if _, err := recording.Decode([]byte(l)); err != nil || code != 0 {
			t.Errorf("events --json line %q: %v", l, err)
		}
	}
	if len(ls) < 3 {
		t.Errorf("events --json gave %d records", len(ls))
	}

	code, out, _ = runMain(dir, "notes", "--json")
	var ns []struct {
		Note    map[string]any
		State   string
		Element string
		Context struct{ Node, Landmark int64 }
		Before  []map[string]any
	}
	if code != 0 || json.Unmarshal([]byte(out), &ns) != nil || len(ns) != 3 || ns[1].State != "resolved" || ns[1].Context.Node != 233 || len(ns[1].Before) == 0 {
		t.Errorf("notes --json: %d %s", code, out)
	}
}

// Recorded text is the page's: a control sequence in it reaches the
// agent's terminal as U+FFFD, never as an escape.
func TestMainScrubsControlCharacters(t *testing.T) {
	b := recording.NewBuilder("esc")
	b.Page(recording.SegLive, "http://app/", b.El("p", []string{"title", "a\x1b]0;pwned\x07b"}, b.Txt("x\x1b[2Jy")))
	b.At(5).Add(&recording.Note{ID: "n\x1b", Author: recording.Author{ID: "v1", Name: "Eve\x1b[31m"}, Text: "t", At: recording.NoteAt{Node: 1}})
	dir := filepath.Join(t.TempDir(), "rec")
	w, err := recording.NewDirWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range b.Recs {
		if err := w.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"dom", "notes"} {
		code, out, errOut := runMain(dir, q)
		if code != 0 || strings.ContainsRune(out, 0x1b) || strings.ContainsRune(out, 0x07) || !strings.Contains(out, "\uFFFD") {
			t.Errorf("%s: exit %d %q %q", q, code, out, errOut)
		}
	}
}

// When matches outrun the budget, the answer says how many were left out.
func TestDOMSaysWhenMatchesAreCut(t *testing.T) {
	code, out, _ := runMain(recording.Fixture("rec-stress"), "dom", "--at", "3000", "--sel", "td", "--max-lines", "4")
	if code != 0 || !strings.Contains(out, "more matches; narrow the selector") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}
