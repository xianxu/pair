package terminal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

type historyOracleRow struct {
	Text    string
	Wrapped bool
	Cells   []oracleCell
}
type historyOracleScreen struct {
	oracleScreen
	History []historyOracleRow
	Wraps   []bool
}

func runHistoryOracle(t *testing.T, cols, rows int, chunks ...string) []historyOracleScreen {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "../../../tests/terminal-oracle")
	if _, err := os.Stat(filepath.Join(dir, "node_modules/@xterm/headless/package.json")); err != nil {
		if os.Getenv("PAIR_TERMINAL_ORACLE") == "1" {
			t.Fatal(err)
		}
		t.Skip("npm ci in tests/terminal-oracle required")
	}
	data, _ := json.Marshal(map[string]any{"Cols": cols, "Rows": rows, "Scrollback": 1000, "Chunks": chunks})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", "driver.cjs")
	cmd.Dir = dir
	cmd.Stdin = bytes.NewReader(data)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("history oracle %v: %s", err, output)
	}
	var screens []historyOracleScreen
	if err := json.Unmarshal(output, &screens); err != nil {
		t.Fatal(err)
	}
	return screens
}
func nativeHistoryDump(t *testing.T, cols, rows int, wire string) string {
	t.Helper()
	if os.Getenv("PAIR_TERMINAL_NATIVE") != "1" {
		t.Skip("PAIR_TERMINAL_NATIVE=1 enables disposable Zellij oracle")
	}
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "../../../tests/terminal-oracle/discovery")
	data, _ := json.Marshal(map[string]any{"cols": cols, "rows": rows, "wire": wire})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", "zellij_oracle.py")
	cmd.Dir = dir
	cmd.Stdin = bytes.NewReader(data)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native history oracle %v: %s", err, output)
	}
	var result struct {
		Stdout, Stderr string
		Returncode     int
	}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatal(err)
	}
	if result.Returncode != 0 {
		t.Fatalf("native oracle exit%d: %s", result.Returncode, result.Stderr)
	}
	return result.Stdout
}
func TestHistoryWireIndependentOracle(t *testing.T) {
	source := "AAA界Z\r\nAA  BZ\r\nHARD\r\nONE\r\nTWO\r\nTHR\r\nFOUR"
	p, f := historyFixture(t, source)
	wire, state := renderHistoryBytes(t, Frame{}, f, p.History, HistoryState{})
	direct := "\x1b[1;4r" + source + "\x1b[5;1HCHR"
	expected := runHistoryOracle(t, 4, 5, direct)[0]
	got := runHistoryOracle(t, 4, 5, string(wire))[0]
	if !reflect.DeepEqual(got.History, expected.History) {
		t.Fatalf("history\ngot %+v\nwant %+v", got.History, expected.History)
	}
	if !reflect.DeepEqual(got.Wraps, expected.Wraps) || !reflect.DeepEqual(got.Lines, expected.Lines) {
		t.Fatalf("viewport %+v wraps%v want%+v wraps%v", got.Lines, got.Wraps, expected.Lines, expected.Wraps)
	}
	if len(got.History) != 5 || got.History[0].Text != "AAA" || !got.History[1].Wrapped || got.History[2].Text != "AA  " {
		t.Fatal("literal history expectations failed")
	}
	// Append output with the same owner, then rebuild after an owner switch.
	p2, f2 := historyFixture(t, source+"\r\nNEXT")
	appendWire, state2 := renderHistoryBytes(t, f, f2, p2.History, state)
	appended := runHistoryOracle(t, 4, 5, string(wire), string(appendWire))[1]
	wantAppend := runHistoryOracle(t, 4, 5, "\x1b[1;4r"+source+"\r\nNEXT\x1b[5;1HCHR")[0]
	if !reflect.DeepEqual(appended.History, wantAppend.History) {
		t.Fatal("incremental history mismatch")
	}
	other := f2
	other.EndpointID = "other"
	rebuilt, _ := renderHistoryBytes(t, f2, other, p2.History, state2)
	again := runHistoryOracle(t, 4, 5, string(wire), string(appendWire), string(rebuilt))[2]
	if !reflect.DeepEqual(again.History, wantAppend.History) {
		t.Fatal("selection duplicated history")
	}
}
func TestHistoryWireNativeOracle(t *testing.T) {
	source := "AAA界Z\r\nAA  BZ\r\nHARD\r\nONE\r\nTWO\r\nTHR\r\nFOUR"
	p, f := historyFixture(t, source)
	wire, _ := renderHistoryBytes(t, Frame{}, f, p.History, HistoryState{})
	direct := "\x1b[1;4r" + source + "\x1b[5;1HCHR"
	want := nativeHistoryDump(t, 4, 5, direct)
	got := nativeHistoryDump(t, 4, 5, string(wire))
	if got != want {
		t.Fatalf("native got%q want%q", got, want)
	}
	if got != "AAA界Z\nAA  BZ\nHARD\nONE\nTWO\nTHR\nFOUR\nCHR\n" {
		t.Fatalf("literal native history %q", got)
	}
}

func TestHistoryViewportWrapAndOneColumnGrowth(t *testing.T) {
	for _, cols := range []int{1, 3, 4} {
		t.Run(fmt.Sprint(cols), func(t *testing.T) {
			e, err := NewEndpoint("growth", Geometry{cols, 2}, ttyio.NewFake())
			if err != nil {
				t.Fatal(err)
			}
			defer e.Close()
			source := strings.Repeat("A", cols) + strings.Repeat("B", cols) + "C"
			e.Feed([]byte(source), time.Time{})
			if err := e.Resize(Geometry{8, 2}, func(Geometry) error { return nil }); err != nil {
				t.Fatal(err)
			}
			p, err := e.Publication(time.Time{})
			if err != nil {
				t.Fatal(err)
			}
			chrome, _ := StyledRows("CHR", 8, 1)
			f, err := Compose(p.Frame, Geometry{8, 3}, chrome)
			if err != nil {
				t.Fatal(err)
			}
			wire, _ := renderHistoryBytes(t, Frame{}, f, p.History, HistoryState{})
			// Diagnostic scroll admits only the two child rows, exposing the complete
			// history-to-viewport logical connection without admitting chrome.
			wire = append(wire, []byte("\x1b[r\x1b[3;1H\n\n")...)
			screens := runHistoryOracle(t, 8, 3, string(wire))
			h := screens[0].History
			var logical string
			for _, row := range h {
				logical += row.Text
			}
			if logical != source {
				t.Fatalf("xterm logical %q want%q", logical, source)
			}
			if len(h) != 3 || h[0].Wrapped || !h[1].Wrapped || h[2].Wrapped {
				t.Fatalf("wrap flags %+v", h)
			}
			got := nativeHistoryDump(t, 8, 3, string(wire))
			if got != source+"\n\nCHR\n\n\n" {
				t.Fatalf("native logical %q wantprefix%q", got, source)
			}
		})
	}
}

func TestHistoryOneHostRowOracle(t *testing.T) {
	source := "AAA界Z\r\nB\r\nC"
	e, err := NewEndpoint("one-row", Geometry{4, 1}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	e.Feed([]byte(source), time.Time{})
	p, err := e.Publication(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	f, err := Compose(p.Frame, Geometry{4, 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	wire, _ := renderHistoryBytes(t, Frame{}, f, p.History, HistoryState{})
	direct := runHistoryOracle(t, 4, 1, source)[0]
	got := runHistoryOracle(t, 4, 1, string(wire))[0]
	if !reflect.DeepEqual(got.History, direct.History) || !reflect.DeepEqual(got.Lines, direct.Lines) || !reflect.DeepEqual(got.Wraps, direct.Wraps) {
		t.Fatalf("one-row typed=%+v direct=%+v", got, direct)
	}
	expected := nativeHistoryDump(t, 4, 1, source)
	native := nativeHistoryDump(t, 4, 1, string(wire))
	if native != expected || native != "AAA界Z\nB\nC\n" {
		t.Fatalf("native one-row got%q direct%q", native, expected)
	}
}

func TestHistoryED2NativeBlankAndSpaceRows(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
		g                  Geometry
	}{
		{"hard-empty-row", "A\r\n\r\nB\x1b[2J", "A\n\nB\n\n\n\n\n", Geometry{4, 4}},
		{"soft-space-row", "A   B\x1b[2J", "A   B\n\n\n\n", Geometry{2, 3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, err := NewEndpoint("ed2", tc.g, ttyio.NewFake())
			if err != nil {
				t.Fatal(err)
			}
			defer e.Close()
			e.Feed([]byte(tc.source), time.Time{})
			p, err := e.Publication(time.Time{})
			if err != nil {
				t.Fatal(err)
			}
			wire, _ := renderHistoryBytes(t, Frame{}, p.Frame, p.History, HistoryState{})
			// ED2 intentionally follows native Zellij's preserve-to-history semantics.
			// Pinned xterm itself erases without saving the viewport to history.
			directX := runHistoryOracle(t, tc.g.Cols, tc.g.Rows, tc.source)[0]
			typedX := runHistoryOracle(t, tc.g.Cols, tc.g.Rows, string(wire))[0]
			if len(directX.History) != 0 || len(typedX.History) != 3 {
				t.Fatalf("ED2 baseline history direct=%d typed=%d", len(directX.History), len(typedX.History))
			}
			direct := nativeHistoryDump(t, tc.g.Cols, tc.g.Rows, tc.source)
			typed := nativeHistoryDump(t, tc.g.Cols, tc.g.Rows, string(wire))
			if direct != tc.want || typed != tc.want {
				t.Fatalf("ED2 logical direct%q typed%q want%q", direct, typed, tc.want)
			}
		})
	}
}
