package query

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// The scripted agent check: given only a recording and the CLI, the answer
// to "what does this note refer to?" is in the note's output: the element
// (its id and tag), the time, and the action just before it, named by its
// target. The expectations are the fixtures' own records, found here
// independently of the query code.
func TestAgentCanTellWhatEachNoteRefersTo(t *testing.T) {
	for _, name := range recording.Fixtures() {
		dir := recording.Fixture(name)
		r, err := recording.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for i, x := range r.Records {
			n, ok := x.(*recording.Note)
			if !ok {
				continue
			}
			code, out, errOut := runMain(dir, "notes", "--note", n.ID)
			if code != 0 {
				t.Fatalf("%s %s: %s", name, n.ID, errOut)
			}
			st, err := recording.Reduce(r.Records[:i+1], n.T)
			if err != nil {
				t.Fatal(err)
			}
			el := st.Node(n.At.Node)
			for _, want := range []string{
				fmt.Sprintf("note %s  t=%d", n.ID, n.T),
				fmt.Sprintf("on @%d", n.At.Node),
				fmt.Sprintf("@%d <%s", el.ID, el.N),
				fmt.Sprintf("%q", n.Text),
			} {
				if !strings.Contains(out, want) {
					t.Errorf("%s %s: want %q in\n%s", name, n.ID, want, out)
				}
			}
			// The last user action before the note, if any in the window.
			for j := i - 1; j >= 0 && r.Records[j].Time() >= n.T-DefaultBefore; j-- {
				var want string
				switch a := r.Records[j].(type) {
				case *recording.Click:
					want = fmt.Sprintf("click @%d", a.ID)
				case *recording.Focus:
					want = fmt.Sprintf("focus @%d", a.ID)
				case *recording.Input:
					want = fmt.Sprintf("input @%d", a.ID)
				case *recording.Scroll:
					want = fmt.Sprintf("scroll @%d", a.ID)
				case *recording.Selection:
					// The real recorder reports selection (#472). A caret
					// after typing folds into the typed value's line.
					if a.Anchor != nil && a.Focus != nil && *a.Anchor == *a.Focus && typedJustBefore(r.Records[:j], a.By) {
						continue
					}
					switch {
					case a.Anchor != nil:
						want = fmt.Sprintf("selection @%d", a.Anchor[0])
					case a.Focus != nil:
						want = "selection ? … "
					default:
						want = "selection cleared"
					}
				case *recording.Nav:
					want = "navigate (same document) to " + a.URL
				case *recording.Seg:
					want = fmt.Sprintf("segment %d starts (%s) on %s", a.Seg, a.Why, a.Tab.URL)
				case *recording.Stop:
					want = "stop (" + a.Why + ")"
				default:
					continue
				}
				if last := lastLine(out); !strings.Contains(last, want) {
					t.Errorf("%s %s: the last action before is %q; output ends %q", name, n.ID, want, last)
				}
				break
			}
		}
	}
}

// typedJustBefore says whether the last record of recs that isn't a value
// attribute is a typed value by by.
func typedJustBefore(recs []recording.Record, by string) bool {
	for j := len(recs) - 1; j >= 0; j-- {
		switch a := recs[j].(type) {
		case *recording.Attr:
			if a.Name == "value" {
				continue
			}
		case *recording.Input:
			return a.V != nil && a.By == by
		}
		return false
	}
	return false
}

func lastLine(s string) string {
	ls := strings.Split(strings.TrimSpace(s), "\n")
	return ls[len(ls)-1]
}
