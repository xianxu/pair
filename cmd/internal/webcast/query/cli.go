package query

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Flag is the couch flag the queries answer to.
const Flag = "--broadcast-web-query"

// Usage is the queries' help, for couch --help and every usage error.
const Usage = `couch ` + Flag + ` DIR dom [--at T] [--sel SELECTOR] [--max-lines N] [--json]
couch ` + Flag + ` DIR events --from T1 --to T2 [--max-lines N] [--json]
couch ` + Flag + ` DIR notes [--note ID] [--before MS] [--json]
couch ` + Flag + ` DIR sketches [--sketch ID] [--at T] [--json]
couch ` + Flag + ` DIR sketch-image --sketch ID --out DIR [--at T]
couch ` + Flag + ` DIR link [--at T]
  DIR is a recording directory. Times are ms since the recording started; --at defaults
  to its end (link's to 0), --before (the actions shown before each note) to 10000.
  link prints the recording's replay link at --at, shared through the running Couch's
  broadcast tunnel, to point a person at a moment. Every element is
  named @N, its node id: pass --sel @N to see it.
  Selectors: ` + SelectorGrammar + `.
`

// DefaultBefore is how far back a note's preceding actions go, in ms.
const DefaultBefore = 10000

type args struct {
	dir, verb, sel, note, sketch, out string
	at, from, to, before              int64
	atSet, fromSet, toSet             bool
	maxLines                          int
	json                              bool
}

func parseArgs(argv []string) (args, error) {
	a := args{before: DefaultBefore}
	if len(argv) < 2 {
		return a, fmt.Errorf("needs a recording directory and a query (dom, events, notes or link)")
	}
	a.dir, a.verb = argv[0], argv[1]
	allowed := map[string]map[string]bool{
		"dom":          {"--at": true, "--sel": true, "--max-lines": true, "--json": true},
		"events":       {"--from": true, "--to": true, "--max-lines": true, "--json": true},
		"notes":        {"--note": true, "--before": true, "--json": true},
		"link":         {"--at": true},
		"sketches":     {"--sketch": true, "--at": true, "--json": true},
		"sketch-image": {"--sketch": true, "--out": true, "--at": true},
	}[a.verb]
	if allowed == nil {
		return a, fmt.Errorf("unknown query %q (dom, events, notes or link)", a.verb)
	}
	seen := map[string]bool{}
	rest := argv[2:]
	for i := 0; i < len(rest); i++ {
		f := rest[i]
		if !allowed[f] {
			return a, fmt.Errorf("%s doesn't take %q", a.verb, f)
		}
		if seen[f] {
			return a, fmt.Errorf("%s given twice", f)
		}
		seen[f] = true
		if f == "--json" {
			a.json = true
			continue
		}
		if i+1 >= len(rest) {
			return a, fmt.Errorf("%s needs a value", f)
		}
		i++
		v := rest[i]
		switch f {
		case "--sel":
			a.sel = v
		case "--out":
			a.out = v
		case "--sketch":
			a.sketch = v
		case "--note":
			a.note = v
		default:
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil || n < 0 {
				return a, fmt.Errorf("%s takes a whole number of ms, not %q", f, v)
			}
			switch f {
			case "--at":
				a.at, a.atSet = n, true
			case "--from":
				a.from, a.fromSet = n, true
			case "--to":
				a.to, a.toSet = n, true
			case "--before":
				a.before = n
			case "--max-lines":
				if n == 0 {
					return a, fmt.Errorf("--max-lines takes at least 1")
				}
				a.maxLines = int(n)
			}
		}
	}
	if a.verb == "sketch-image" && (a.sketch == "" || a.out == "") {
		return a, fmt.Errorf("sketch-image requires --sketch ID and --out DIR")
	}
	if a.json && a.maxLines > 0 {
		return a, fmt.Errorf("--json answers in full; --max-lines budgets the text")
	}
	if a.verb == "events" && (!a.fromSet || !a.toSet) {
		return a, fmt.Errorf("events needs --from and --to")
	}
	return a, nil
}

// Linker is a recording's replay link at t ms: couch's, which asks the
// running Couch to share it (#477). The queries themselves need no Couch.
type Linker func(dir string, at int64) (string, error)

// Run runs one query: argv is what follows the couch flag. It exits 2 on a
// usage error and 1 when the recording can't answer. link asks link; nil
// refuses it.
func Run(argv []string, link Linker, stdout, stderr io.Writer) int {
	return RunWithOptions(argv, RunOptions{Link: link}, stdout, stderr)
}

// RunWithOptions preserves ordinary queries while injecting image publication.
// Query itself never launches a browser or creates image output files.
func RunWithOptions(argv []string, options RunOptions, stdout, stderr io.Writer) int {
	link := options.Link
	a, err := parseArgs(argv)
	if err != nil {
		fmt.Fprintf(stderr, "couch: %s: %v\nusage:\n%s", Flag, err, Usage)
		return 2
	}
	if a.verb == "link" {
		if link == nil {
			link = func(string, int64) (string, error) { return "", fmt.Errorf("no Couch to share the recording through") }
		}
		url, err := link(a.dir, a.at)
		if err != nil {
			fmt.Fprintf(stderr, "couch: %s: %v\n", Flag, err)
			return 1
		}
		fmt.Fprintln(stdout, url)
		return 0
	}
	rec, err := recording.ReadDir(a.dir)
	if err != nil {
		fmt.Fprintf(stderr, "couch: %s: %v\n", Flag, err)
		return 1
	}
	if len(rec.Records) == 0 {
		fmt.Fprintf(stderr, "couch: %s: %s holds no records\n", Flag, a.dir)
		return 1
	}
	q := New(rec.Records, notes.Structural{})
	q.PageLines = a.maxLines
	out := scrubber{stdout}
	if a.verb == "sketch-image" {
		if err := exportSketchImage(q, a, options); err != nil {
			fmt.Fprintf(stderr, "couch: %s: %s\n", Flag, strings.TrimPrefix(err.Error(), "query: "))
			return 1
		}
		fmt.Fprintln(out, a.out)
		return 0
	}
	if err := answer(q, a, out); err != nil {
		fmt.Fprintf(stderr, "couch: %s: %s\n", Flag, strings.TrimPrefix(err.Error(), "query: "))
		return 1
	}
	return 0
}

// scrubber replaces control characters (other than newline and tab) with
// U+FFFD on the way out. Every answer quotes the recorded page, and an
// escape sequence in a page's text must not reach the agent's terminal.
type scrubber struct{ w io.Writer }

func (s scrubber) Write(p []byte) (int, error) {
	clean := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return unicode.ReplacementChar
		}
		return r
	}, string(p))
	if _, err := io.WriteString(s.w, clean); err != nil {
		return 0, err
	}
	return len(p), nil
}

func answer(q *Querier, a args, w io.Writer) error {
	switch a.verb {
	case "sketches":
		if !a.atSet {
			a.at = q.End()
		}
		views, err := q.SketchesAt(a.at)
		if err != nil {
			return err
		}
		if a.sketch != "" {
			found := false
			for _, v := range views {
				if v.History.Begin.ID == a.sketch {
					views = []SketchView{v}
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("query: sketch %q missing at history cutoff", a.sketch)
			}
		}
		if a.json {
			return json.NewEncoder(w).Encode(views)
		}
		for _, v := range views {
			if _, err := io.WriteString(w, SketchText(v)); err != nil {
				return err
			}
		}
		return nil
	case "dom":
		if !a.atSet {
			a.at = q.End()
		}
		if a.json {
			return domJSON(q, a, w)
		}
		page, err := q.DOMAt(a.at, a.sel)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, page)
		return err
	case "events":
		if a.json {
			evs, err := q.Events(a.from, a.to)
			if err != nil {
				return err
			}
			for _, r := range evs {
				line, err := recording.Encode(r)
				if err != nil {
					return err
				}
				fmt.Fprintf(w, "%s\n", line)
			}
			return nil
		}
		lines, err := EventLines(q.recs, a.from, a.to)
		if err != nil {
			return err
		}
		max := a.maxLines
		if max == 0 {
			max = DefaultPageLines
		}
		if len(lines) == 0 {
			fmt.Fprintf(w, "nothing happened from %d to %d\n", a.from, a.to)
		}
		for i, l := range lines {
			if i == max {
				fmt.Fprintf(w, "… %d more lines; narrow the window\n", len(lines)-max)
				break
			}
			fmt.Fprintln(w, l)
		}
		return nil
	default:
		vs, err := q.Notes(a.before)
		if err != nil {
			return err
		}
		if a.note != "" {
			var one []NoteView
			for _, v := range vs {
				if v.Note.ID == a.note {
					one = append(one, v)
				}
			}
			if len(one) == 0 {
				return fmt.Errorf("no note %q in this recording", a.note)
			}
			vs = one
		}
		if a.json {
			return notesJSON(vs, w)
		}
		if len(vs) == 0 {
			fmt.Fprintln(w, "no notes in this recording")
		}
		for i, v := range vs {
			if i > 0 {
				fmt.Fprintln(w)
			}
			io.WriteString(w, NoteText(v, a.before))
		}
		return nil
	}
}

// NoteText is a note as an agent reads it: who said what, when, on which
// element, the context around it, and what the user did just before.
func NoteText(v NoteView, before int64) string {
	var b strings.Builder
	n := v.Note
	fmt.Fprintf(&b, "note %s  t=%d  segment %d  %s  by %s\n", n.ID, n.T, n.Seg, n.State, who(n.Author.ID, n.Author.Name))
	if v.PageLoad != (PageLoadFacts{}) {
		fmt.Fprintf(&b, "  page_load: target=%q loader=%q root=%q\n", v.PageLoad.Target, v.PageLoad.Loader, v.PageLoad.Root)
	}
	if n.Edited {
		b.WriteString("  edited\n")
	}
	if n.ResolvedBy != nil {
		fmt.Fprintf(&b, "  resolved by %s\n", who(n.ResolvedBy.ID, n.ResolvedBy.Name))
	}
	fmt.Fprintf(&b, "  %s\n", strconv.Quote(n.Text))
	fmt.Fprintf(&b, "  on @%d", n.At.Node)
	if n.At.Sel != "" {
		fmt.Fprintf(&b, "  selector %s", n.At.Sel)
	}
	if n.At.Box != nil {
		bx := n.At.Box
		fmt.Fprintf(&b, "  box %s,%s %sx%s", num(bx[0]), num(bx[1]), num(bx[2]), num(bx[3]))
	}
	if n.Region != nil {
		r := n.Region
		fmt.Fprintf(&b, "  region %s,%s %sx%s", num(r[0]), num(r[1]), num(r[2]), num(r[3]))
	}
	b.WriteString("\n")
	if v.Header != "" {
		fmt.Fprintf(&b, "  page: %s\n", v.Header)
	}
	if v.Element == "" {
		b.WriteString("  element: gone at the note's time\n")
	} else {
		b.WriteString("  element:\n")
		b.WriteString(indent(v.Element, "    "))
	}
	if v.Landmark != "" {
		fmt.Fprintf(&b, "  context: within %s\n", v.Landmark)
		if v.Context.Text != "" {
			fmt.Fprintf(&b, "    text: %s\n", strconv.Quote(v.Context.Text))
		}
	}
	fmt.Fprintf(&b, "  before (%d ms):", before)
	lines := v.BeforeLines
	if len(lines) == 0 {
		for _, r := range v.Before { // a surface without words: kinds and times
			lines = append(lines, fmt.Sprintf("%d %s", r.Time(), r.Kind()))
		}
	}
	if len(lines) == 0 {
		b.WriteString(" no user actions\n")
	} else {
		b.WriteString("\n" + indent(strings.Join(lines, "\n")+"\n", "    "))
	}
	return b.String()
}

func indent(s, pad string) string {
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	return pad + strings.Join(lines, "\n"+pad) + "\n"
}

// jsonNode is a live node in the format's Node shape, plus what the moment
// adds: an iframe's document and the observer's state.
type jsonNode struct {
	ID      recording.NodeID  `json:"id"`
	Y       string            `json:"y"`
	N       string            `json:"n,omitempty"`
	NS      string            `json:"ns,omitempty"`
	A       map[string]string `json:"a,omitempty"`
	V       *string           `json:"v,omitempty"`
	Mode    string            `json:"mode,omitempty"`
	URL     string            `json:"url,omitempty"`
	C       []*jsonNode       `json:"c,omitempty"`
	SR      *jsonNode         `json:"sr,omitempty"`
	TC      *jsonNode         `json:"tc,omitempty"`
	Frame   *jsonNode         `json:"frame,omitempty"`
	State   *recording.Input  `json:"state,omitempty"`
	Focused bool              `json:"focused,omitempty"`
	Scroll  *[2]float64       `json:"scroll,omitempty"`
}

func toJSON(st *recording.State, n *recording.DOMNode) *jsonNode {
	if n == nil {
		return nil
	}
	j := &jsonNode{ID: n.ID, Y: n.Y, N: n.N, NS: n.NS, A: n.A, Mode: n.Mode, URL: n.URL,
		SR: toJSON(st, n.Shadow), TC: toJSON(st, n.Content), Frame: toJSON(st, n.Frame),
		State: st.Inputs[n.ID], Focused: st.Focus == n.ID && n.ID != 0}
	if n.Y == recording.YText || n.Y == recording.YComment {
		v := n.V
		j.V = &v
	}
	if sc, ok := st.Scrolls[n.ID]; ok {
		j.Scroll = &sc
	}
	for _, k := range n.Kids {
		j.C = append(j.C, toJSON(st, k))
	}
	return j
}

func domJSON(q *Querier, a args, w io.Writer) error {
	st, roots, err := q.matchAt(a.at, a.sel)
	if err != nil {
		return err
	}
	out := struct {
		T       int64              `json:"t"`
		Seg     int                `json:"seg"`
		Live    bool               `json:"live"`
		URL     string             `json:"url"`
		Title   string             `json:"title,omitempty"`
		VP      recording.Viewport `json:"vp"`
		Control recording.Control  `json:"control"`
		Matches []*jsonNode        `json:"matches"`
	}{T: st.T, Seg: st.Seg, Live: st.Live, URL: st.URL, Title: st.Title, VP: st.VP, Control: st.Control}
	for _, r := range roots {
		out.Matches = append(out.Matches, toJSON(st, r))
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	return enc.Encode(out)
}

func notesJSON(vs []NoteView, w io.Writer) error {
	type ctx struct {
		Node     recording.NodeID `json:"node"`
		Landmark recording.NodeID `json:"landmark"`
		Text     string           `json:"text,omitempty"`
		Box      *recording.Box   `json:"box,omitempty"`
	}
	type view struct {
		PageLoad   PageLoadFacts     `json:"page_load,omitzero"`
		ResolvedBy *recording.Author `json:"resolved_by,omitempty"`
		Note       json.RawMessage   `json:"note"`
		State      string            `json:"state"`
		Seg        int               `json:"seg"`
		URL        string            `json:"url,omitempty"`
		Element    string            `json:"element,omitempty"`
		Context    ctx               `json:"context"`
		Landmark   string            `json:"landmark,omitempty"`
		Before     []json.RawMessage `json:"before"`
	}
	out := []view{}
	for _, v := range vs {
		note, err := recording.Encode(v.Note.Note)
		if err != nil {
			return err
		}
		jv := view{PageLoad: v.PageLoad, ResolvedBy: v.Note.ResolvedBy, Note: note, State: v.Note.State, Seg: v.Note.Seg, URL: v.URL, Element: v.Element,
			Context: ctx{v.Context.Node, v.Context.Landmark, v.Context.Text, v.Context.Box}, Landmark: v.Landmark,
			Before: []json.RawMessage{}}
		for _, r := range v.Before {
			line, err := recording.Encode(r)
			if err != nil {
				return err
			}
			jv.Before = append(jv.Before, line)
		}
		out = append(out, jv)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	return enc.Encode(out)
}
