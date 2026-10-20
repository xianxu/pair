package terminal

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/xianxu/pair/cmd/internal/ttyio"
)

func TestBackendCellDegradationPublication(t *testing.T) {
	style := uv.Style{Fg: ansi.IndexedColor(2), Attrs: uv.AttrBold}
	cases := []Cell{
		{Content: "\x1b]52;secret", Width: 1}, {Content: string([]byte{0xff, 0xfe}), Width: 1},
		{Content: "ab", Width: 1}, {Content: "界", Width: 1}, {Content: "A", Width: -1},
		{Content: "A", Width: 3}, {Content: "界", Width: 2}, {Content: "x", Width: 0},
		{Content: strings.Repeat("a", MaxClusterBytes+1), Width: 1},
		{Content: "A", Width: 1, Link: uv.Link{URL: "https://secret\x1b", Params: "private"}},
	}
	for i, bad := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			e, err := NewEndpoint("degraded", Geometry{8, 4}, ttyio.NewFake())
			if err != nil {
				t.Fatal(err)
			}
			defer e.Close()
			bad.Style = style
			*e.backend.CellAt(0, 0) = bad
			*e.backend.CellAt(1, 0) = Cell{Content: "Q", Width: 1}
			e.generation++
			p, err := e.Publication(time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if err := p.Frame.Validate(); err != nil {
				t.Fatalf("backend cell escaped unrepaired: %v", err)
			}
			if got := p.Frame.Cells[0]; got.Content != "�" || got.Width != 1 || got.Link != (uv.Link{}) || !reflect.DeepEqual(got.Style, style) {
				t.Fatalf("replacement=%+v", got)
			}
			if p.Frame.Cells[1].Content != "Q" {
				t.Fatal("unrelated neighbor moved")
			}
			if !reflect.DeepEqual(*e.backend.CellAt(0, 0), bad) {
				t.Fatal("backend mutated")
			}
			direct := p.Frame.Clone()
			direct.Cells[0] = bad
			if direct.Validate() == nil {
				t.Fatal("direct invalid frame accepted")
			}
			if _, err := Render(Frame{}, direct); err == nil {
				t.Fatal("direct invalid render accepted")
			}
		})
	}
}

func TestBackendCellDegradationHistoryAndWideNeighbor(t *testing.T) {
	e, err := NewEndpoint("history", Geometry{8, 4}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	// A valid wide head becomes broken when its unsafe continuation is repaired.
	row := uv.Line{{Content: "界", Width: 2}, {Width: 0, Link: uv.Link{URL: "secret\n"}}, {Content: "Z", Width: 1}}
	e.backend.Scrollback().Push(row)
	before := e.backend.HistorySnapshot()
	p, err := e.Publication(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := p.History.Validate(); err != nil {
		t.Fatalf("unsafe history: %v", err)
	}
	if len(p.History.Rows) != 1 || p.History.Rows[0].Cells[0].Content != "�" || p.History.Rows[0].Cells[1].Content != "�" || p.History.Rows[0].Cells[2].Content != "Z" {
		t.Fatalf("history=%+v", p.History)
	}
	if !reflect.DeepEqual(before, e.backend.HistorySnapshot()) {
		t.Fatal("backend history mutated")
	}
	direct := p.History.Clone()
	direct.Rows[0].Cells[0] = Cell{Content: "\x00", Width: 1}
	if direct.Validate() == nil {
		t.Fatal("direct invalid history accepted")
	}
}

func TestBackendCellDegradationObservationOnceAndBounded(t *testing.T) {
	var observations []Observation
	e, err := NewEndpoint("evidence", Geometry{8, 4}, ttyio.NewFake(), func(o Observation) {
		if o.Kind == "cell-degraded" {
			o.Data = append([]byte(nil), o.Data...)
			observations = append(observations, o)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	content := strings.Repeat("\x1b", 80) + strings.Repeat(string([]byte{0xff}), 80)
	for n := 0; n < 20; n++ {
		*e.backend.CellAt(2, 1) = Cell{Content: content, Width: 99, Link: uv.Link{URL: "DO-NOT-LOG-URL", Params: "DO-NOT-LOG-PARAMS"}}
		e.generation++
		if _, err := e.Publication(time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	if len(observations) != 1 {
		t.Fatalf("degradation observations=%d, want1", len(observations))
	}
	data := observations[0].Data
	if !json.Valid(data) || len(data) > 2048 || strings.Contains(string(data), "DO-NOT-LOG") {
		t.Fatalf("unsafe/unbounded evidence: %q", data)
	}
	var evidence map[string]any
	if err := json.Unmarshal(data, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence["source"] != "screen" || evidence["row"] != float64(1) || evidence["column"] != float64(2) || evidence["width"] != float64(99) || evidence["content_bytes"] != float64(len(content)) || evidence["content_clipped"] != true || evidence["invalid_bytes"] != float64(80) || evidence["invalid_bytes_clipped"] != true {
		t.Fatalf("missing bounded evidence: %s", data)
	}
	prefix, _ := evidence["content_prefix"].(string)
	for _, r := range prefix {
		if unicode.IsControl(r) {
			t.Fatal("decoded evidence contains raw controls")
		}
	}
}

func TestBackendCellDegradationGeneratedRows(t *testing.T) {
	e, err := NewEndpoint("generated", Geometry{16, 4}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	rng := rand.New(rand.NewSource(727))
	for trial := 0; trial < 500; trial++ {
		var originals []Cell
		for x := 0; x < 16; x++ {
			b := make([]byte, rng.Intn(12))
			rng.Read(b)
			c := Cell{Content: string(b), Width: rng.Intn(7) - 2}
			if x%4 == 3 {
				c = Cell{Content: "V", Width: 1, Style: uv.Style{Attrs: uv.AttrItalic}}
			}
			*e.backend.CellAt(x, 0) = c
			originals = append(originals, c)
		}
		e.generation++
		p, err := e.Publication(time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if err := p.Frame.Validate(); err != nil {
			t.Fatalf("trial%d: %v", trial, err)
		}
		row := p.Frame.Cells[:16]
		if len(row) != len(originals) {
			t.Fatal("column count changed")
		}
		for x, c := range row {
			if !utf8.ValidString(c.Content) {
				t.Fatalf("invalid utf8 at%d", x)
			}
			for _, r := range c.Content {
				if unicode.IsControl(r) {
					t.Fatalf("control at%d", x)
				}
			}
			if c.Width < 0 || c.Width > 2 {
				t.Fatalf("bad width at%d", x)
			}
			if c.Width == 2 && (x+1 == len(row) || row[x+1].Width != 0 || row[x+1].Content != "") {
				t.Fatalf("broken pair at%d", x)
			}
			if x%4 == 3 && !reflect.DeepEqual(c, originals[x]) {
				t.Fatalf("valid cell changed at%d", x)
			}
			if !reflect.DeepEqual(*e.backend.CellAt(x, 0), originals[x]) {
				t.Fatal("input mutated")
			}
		}
	}
}

func TestBackendCellDegradationPresenterOracle(t *testing.T) {
	p, parent, e, child := presenterFixture(t, AnyMotion)
	selectPresenter(t, p, e)
	*e.backend.CellAt(0, 0) = Cell{Content: "\x1b]52;bad", Width: 1, Style: uv.Style{Fg: ansi.IndexedColor(2), Attrs: uv.AttrBold}}
	*e.backend.CellAt(1, 0) = Cell{Content: "Q", Width: 1}
	e.generation++
	if err := p.Present(context.Background(), e); err != nil {
		t.Fatalf("backend fault killed presenter: %v", err)
	}
	if err := p.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if p.View().State != Ready {
		t.Fatalf("presenter state=%+v", p.View())
	}
	if err := p.Input(context.Background(), uv.KeyPressEvent{Code: 'x', Text: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := e.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if string(child.Bytes()) != "x" {
		t.Fatalf("input=%q", child.Bytes())
	}
	screens := runOracle(t, 8, 5, []string{string(parent.Bytes())})
	if screens[0].Lines[0] != "�Q      " || screens[0].Cells[0][0].FG != 2 || !screens[0].Cells[0][0].Bold {
		t.Fatalf("oracle=%+v", screens[0])
	}
}

// independentCellSafe spells the external output invariant without calling the
// production validator or repair classifier.
func independentCellSafe(row []Cell, x int) bool {
	c := row[x]
	printable := func(s string) bool {
		if !utf8.ValidString(s) {
			return false
		}
		for _, r := range s {
			if unicode.IsControl(r) {
				return false
			}
		}
		return true
	}
	if !printable(c.Content) || len(c.Content) > MaxClusterBytes || !printable(c.Link.URL) || !printable(c.Link.Params) || len(c.Link.URL) > MaxLinkBytes || len(c.Link.Params) > MaxLinkBytes || c.Width < 0 || c.Width > 2 {
		return false
	}
	if c.Content != "" {
		g, w := ansi.FirstGraphemeCluster(c.Content, ansi.GraphemeWidth)
		if g != c.Content || w != c.Width || c.Width == 0 {
			return false
		}
	}
	if c.Width == 2 {
		if x+1 >= len(row) || row[x+1].Content != "" || row[x+1].Width != 0 {
			return false
		}
		// A head whose continuation must degrade cannot retain width two.
		if !independentCellSafe(row, x+1) {
			return false
		}
	}
	return true
}

func TestRepairBackendRowMixedGeneratedInvariants(t *testing.T) {
	rng := rand.New(rand.NewSource(72716))
	palette := []Cell{{}, {Content: "A", Width: 1}, {Content: "界", Width: 2}, {Content: "👩‍💻", Width: 2}, {Content: "é", Width: 1}, {Content: "界", Width: 1}, {Content: "\u0301", Width: 0}, {Width: 3}, {Content: "x", Width: 0}, {Content: "\xc0\xaf", Width: 1}, {Content: "a\x9b", Width: 1}, {Content: "ab", Width: 1}, {Width: 0, Link: uv.Link{URL: "\x1bsecret"}}, {Content: "L", Width: 1, Link: uv.Link{URL: "https://valid.example", Params: "id=ok"}}}
	for trial := 0; trial < 2000; trial++ {
		row := make([]Cell, 1+rng.Intn(32))
		for i := range row {
			row[i] = palette[rng.Intn(len(palette))]
			row[i].Style = uv.Style{Attrs: uv.AttrBold}
		}
		before := append([]Cell(nil), row...)
		repaired, fault, work := repairBackendRow(row)
		if !reflect.DeepEqual(row, before) {
			t.Fatal("pure repair helper mutated its input")
		}
		row = repaired
		if work != len(row) {
			t.Fatalf("work=%d cells=%d", work, len(row))
		}
		changed := false
		for x, c := range row {
			if !independentCellSafe(row, x) {
				t.Fatalf("trial%d col%d unsafe output %+v", trial, x, row)
			}
			if independentCellSafe(before, x) {
				if !reflect.DeepEqual(c, before[x]) {
					t.Fatalf("trial%d valid cell%d changed: %+v => %+v", trial, x, before[x], c)
				}
			} else {
				changed = true
				if c.Content != "�" || c.Width != 1 || c.Link != (uv.Link{}) || !reflect.DeepEqual(c.Style, before[x].Style) {
					t.Fatalf("trial%d bad replacement%d: %+v", trial, x, c)
				}
			}
		}
		if changed != (fault != nil) {
			t.Fatal("fault presence disagrees with repairs")
		}
	}
}

func TestRepairBackendRowLinearWorkAtEightAndSixteenRows(t *testing.T) {
	for _, rows := range []int{8, 16} {
		work := 0
		for y := 0; y < rows; y++ {
			row := make([]Cell, 80)
			for i := range row {
				row[i] = Cell{Content: "\x1b", Width: 1}
			}
			_, fault, n := repairBackendRow(row)
			if fault == nil {
				t.Fatal("missing fault")
			}
			work += n
		}
		if work != rows*80 {
			t.Fatalf("%d rows visited%d cells, want%d", rows, work, rows*80)
		}
	}
}

func TestDescribeCellFaultBoundsAndHistoryPosition(t *testing.T) {
	c := Cell{Content: strings.Repeat("😀", 65) + string([]byte{0xff, 0xfe, 0xc0, 0xaf}), Width: -2, Link: uv.Link{URL: "private-url", Params: "private-params"}}
	data := describeCellFault("history", 3, 42, backendCellFault{column: 5, cell: c, reason: "invalid-utf8"})
	var evidence map[string]any
	if err := json.Unmarshal(data, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence["history_row_id"] != float64(42) || evidence["row"] != float64(3) || evidence["column"] != float64(5) || evidence["content_codepoints"] != float64(69) || evidence["invalid_bytes_hex"] != "fffec0af" || evidence["content_prefix"] != strings.Repeat("😀", 64) {
		t.Fatalf("evidence=%s", data)
	}
	if len(data) > 2048 || strings.Contains(string(data), "private") {
		t.Fatalf("unbounded/private evidence=%s", data)
	}
}

func TestDescribeCellFaultExplicitCodepointsAndBoundedScan(t *testing.T) {
	content := "\u200d\x1b" + strings.Repeat("😀", 100000)
	data := describeCellFault("screen", 0, 0, backendCellFault{cell: Cell{Content: content, Width: 2}, reason: "cluster-limit"})
	var evidence map[string]any
	if err := json.Unmarshal(data, &evidence); err != nil {
		t.Fatal(err)
	}
	values, ok := evidence["content_codepoint_values"].([]any)
	if !ok || len(values) != 64 || values[0] != "U+200D" || values[1] != "U+001B" || values[2] != "U+1F600" {
		t.Fatalf("invisible codepoints missing: %s", data)
	}
	if evidence["content_scan_clipped"] != true || evidence["content_codepoints"] != float64(256) || evidence["content_scanned_bytes"].(float64) > 1024 || evidence["content_bytes"] != float64(len(content)) {
		t.Fatalf("scan not bounded: %s", data)
	}
	if len(data) > 2048 {
		t.Fatalf("evidence bytes=%d", len(data))
	}
}

// TestDescribeCellFaultFitsDiagnosticAdmission pins the cell evidence against
// termcmd.termDiagnostics.observe's 4096-byte admission limit. Include every
// optional field, maximum-width integer positions, all 64 sampled codepoints,
// and the full 32-byte invalid encoding sample. Controls have a larger escaped
// prefix than four-byte Unicode; exercise both rather than assuming UTF-8 size
// bounds the resulting JSON size.
func TestDescribeCellFaultFitsDiagnosticAdmission(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	for _, sample := range []string{strings.Repeat("\U0010ffff", 64), strings.Repeat("\u009f", 64), strings.Repeat("<", 64)} {
		c := Cell{Content: sample + strings.Repeat(string([]byte{0xff}), 32), Width: -maxInt - 1, Link: uv.Link{URL: strings.Repeat("private-url", 10000), Params: strings.Repeat("private-params", 10000)}}
		data := describeCellFault("history", maxInt, ^uint64(0), backendCellFault{column: maxInt, cell: c, reason: "content-width-mismatch"})
		if len(data) > 4096 {
			t.Fatalf("trigger evidence would be dropped by diagnostic admission: %d bytes", len(data))
		}
		var evidence map[string]any
		if err := json.Unmarshal(data, &evidence); err != nil {
			t.Fatal(err)
		}
		if len(evidence["content_codepoint_values"].([]any)) != 64 || len(evidence["invalid_bytes_hex"].(string)) != 64 || evidence["content_clipped"] != true || evidence["history_row_id"] == nil {
			t.Fatalf("worst-case fixture omitted evidence: %s", data)
		}
		if strings.Contains(string(data), "private-") {
			t.Fatal("raw hyperlink leaked")
		}
		t.Logf("sample U+%04X: bounded evidence %d/4096 bytes", []rune(sample)[0], len(data))
	}
}
