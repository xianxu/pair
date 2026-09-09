package mouseinput

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The package shipped with its tests living in termcmd, which is where they were
// before the move. That is coverage by inheritance: a change here is only caught
// if termcmd happens to exercise it, and termcmd cannot exercise what only couch
// calls.
func TestParseDecodesBothTerminators(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Event
	}{
		{"\x1b[<0;12;34M", Event{Button: 0, X: 12, Y: 34}},
		{"\x1b[<0;12;34m", Event{Button: 0, X: 12, Y: 34, Release: true}},
		{"\x1b[<64;1;1M", Event{Button: WheelUp, X: 1, Y: 1}},
		// Above 223, which is the reason SGR is required: the legacy X10
		// encoding caps there and fails SILENTLY on a wide or tall terminal.
		{"\x1b[<0;500;400M", Event{Button: 0, X: 500, Y: 400}},
	} {
		got, ok := Parse([]byte(tc.in))
		if !ok || got != tc.want {
			t.Errorf("Parse(%q) = (%+v,%v), want (%+v,true)", tc.in, got, ok, tc.want)
		}
	}
}

func TestParseRefusesWhatItCannotDecode(t *testing.T) {
	for _, in := range []string{
		"", "\x1b[<", "\x1b[<0;12;34", // no terminator
		"\x1b[M abc",   // legacy X10 -- refused, not guessed at
		"\x1b[<a;b;cM", // non-numeric
		"hello",        //
	} {
		if _, ok := Parse([]byte(in)); ok {
			t.Errorf("Parse(%q) decoded something", in)
		}
	}
}

// ParsePrefix returns the RAW bytes, and a forwarder must write those rather
// than a re-encoding -- otherwise the wire format has two sources of truth.
func TestParsePrefixReturnsTheWireBytes(t *testing.T) {
	const report = "\x1b[<0;12;34M"
	event, raw, rest, ok := ParsePrefix([]byte(report + "tail"))
	if !ok {
		t.Fatal("ParsePrefix refused a complete report")
	}
	if string(raw) != report {
		t.Errorf("raw = %q, want %q", raw, report)
	}
	if string(rest) != "tail" {
		t.Errorf("rest = %q, want %q", rest, "tail")
	}
	if event.X != 12 || event.Y != 34 {
		t.Errorf("event = %+v", event)
	}
}

func TestFindLocatesAReportAfterOtherBytes(t *testing.T) {
	before, event, raw, rest, ok := Find([]byte("abc\x1b[<0;7;8Mdef"))
	if !ok {
		t.Fatal("Find missed an embedded report")
	}
	if string(before) != "abc" || string(raw) != "\x1b[<0;7;8M" || string(rest) != "def" {
		t.Errorf("before=%q raw=%q rest=%q", before, raw, rest)
	}
	if event.X != 7 || event.Y != 8 {
		t.Errorf("event = %+v", event)
	}
}

// IsPrefix says "could still become a report". A caller holding on it MUST bound
// the wait -- MaxReport is that bound, and without one a stray introducer parks
// every following keystroke.
func TestIsPrefixCoversEveryGenuinePrefixAndNothingElse(t *testing.T) {
	const report = "\x1b[<0;12;34M"
	for split := 1; split < len(report); split++ {
		if !IsPrefix([]byte(report[:split])) {
			t.Errorf("IsPrefix(%q) = false, but it is a genuine prefix", report[:split])
		}
	}
	if IsPrefix([]byte(report)) {
		t.Error("a COMPLETE report is not a prefix")
	}
	for _, in := range []string{"a", "\x1b[A", "\x1b[200~"} {
		if IsPrefix([]byte(in)) {
			t.Errorf("IsPrefix(%q) = true", in)
		}
	}
	// The bound is what a caller uses to stop waiting.
	if len("\x1b[<"+strings.Repeat("9", MaxReport)) <= MaxReport {
		t.Fatal("MaxReport does not bound anything")
	}
}

// The risky class for WithButton is arbitrary bytes, not a list of shapes: it is
// a splice into a wire format, so the property that matters is that it changes
// the button and NOTHING else, for any input and any button.
func FuzzWithButtonChangesOnlyTheButton(f *testing.F) {
	f.Add([]byte("\x1b[<0;7;9M"), 64)
	f.Add([]byte("\x1b[<80;120;44m"), 65)
	f.Add([]byte("\x1b[<84;1;1M"), 68)
	f.Add([]byte("\x1b[<"), 0)
	f.Add([]byte("\x1b[<abc;1;1M"), 3)
	f.Add([]byte("\x1b[<0;7;9M"), -1)
	f.Add([]byte(""), 0)
	f.Fuzz(func(t *testing.T, raw []byte, button int) {
		out, ok := WithButton(raw, button)
		before, parsed := Parse(raw)

		if !parsed || button < 0 {
			if ok {
				t.Fatalf("WithButton(%q, %d) accepted input Parse rejects", raw, button)
			}
			return
		}
		if !ok {
			t.Fatalf("WithButton(%q, %d) refused a parseable report", raw, button)
		}
		after, ok := Parse(out)
		if !ok {
			t.Fatalf("spliced report no longer parses: %q", out)
		}
		want := before
		want.Button = button
		if after != want {
			t.Fatalf("WithButton(%q, %d) = %q -> %+v, want %+v", raw, button, out, after, want)
		}
		// Everything from the first separator on is the terminal's own bytes.
		if got, expected := out[bytes.IndexByte(out, ';'):], raw[bytes.IndexByte(raw, ';'):]; !bytes.Equal(got, expected) {
			t.Fatalf("bytes after the button field changed: %q, want %q", got, expected)
		}
	})
}

// The rule, not the two instances that motivated it (#213 BR-1): an SGR button
// field carries modifier bits, so a consumer must compare BaseButton(b) and
// never the raw value. Comparing raw silently misses every modified variant —
// in couch it made the ctrl-strip a no-op, and in termcmd it dropped modified
// wheel ticks into the pass-through arm, leaking SGR bytes to a child that never
// enabled tracking. Both read as correct code.
//
// AST-based and rooted at the MODULE, not this package's parent (BR-7): a
// grep for ".Button ==" on one line misses `switch event.Button { case WheelUp:`
// and a button copied into a local first — the same class shape, one refactor
// away — and a walk from "../.." would never reach probes/.
func TestNoConsumerComparesARawButtonAgainstAWheelConstant(t *testing.T) {
	const moduleRoot = "../../.."
	var offenders []string

	err := filepath.Walk(moduleRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if name := info.Name(); name == ".git" || name == "testdata" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil // not our business to fail on unparseable sources
		}
		text := func(node ast.Node) string {
			var buf bytes.Buffer
			if printer.Fprint(&buf, fset, node) != nil {
				return ""
			}
			return buf.String()
		}
		// A raw button is `<expr>.Button`, or a local initialised straight from
		// one — which is the copy-into-a-variable evasion.
		copies := map[string]bool{}
		ast.Inspect(file, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
				return true
			}
			if isRawButton(assign.Rhs[0], nil) {
				if name, ok := assign.Lhs[0].(*ast.Ident); ok {
					copies[name.Name] = true
				}
			}
			return true
		})
		mentionsWheel := func(node ast.Node) bool { return strings.Contains(text(node), "Wheel") }
		report := func(node ast.Node, form string) {
			offenders = append(offenders, fmt.Sprintf("%s:%d: %s (%s)",
				path, fset.Position(node.Pos()).Line, text(node), form))
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.BinaryExpr:
				if node.Op != token.EQL && node.Op != token.NEQ {
					return true
				}
				if isRawButton(node.X, copies) && mentionsWheel(node.Y) ||
					isRawButton(node.Y, copies) && mentionsWheel(node.X) {
					report(node, "comparison")
				}
			case *ast.SwitchStmt:
				if node.Tag == nil || !isRawButton(node.Tag, copies) {
					return true
				}
				for _, stmt := range node.Body.List {
					clause, ok := stmt.(*ast.CaseClause)
					if !ok {
						continue
					}
					for _, value := range clause.List {
						if mentionsWheel(value) {
							report(node.Tag, "switch tag")
						}
					}
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(offenders) > 0 {
		t.Errorf("raw button reaches a wheel constant — use mouseinput.BaseButton:\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

// isRawButton reports whether expr is `<something>.Button`, or an identifier
// copied from one. A BaseButton(...) call is a CallExpr, so it never matches --
// which is the whole point.
func isRawButton(expr ast.Expr, copies map[string]bool) bool {
	switch node := expr.(type) {
	case *ast.SelectorExpr:
		return node.Sel.Name == "Button"
	case *ast.Ident:
		return copies[node.Name]
	}
	return false
}
