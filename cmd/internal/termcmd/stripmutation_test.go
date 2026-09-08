package termcmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// EVERY SITE THAT MUTATES THE STRIP'S MODEL OWES A REPAINT, and the set of those
// sites is READ OUT OF THE SOURCE rather than remembered.
//
// This exists because remembering failed. #199 M3 swept the three rename methods
// -- because a probe happened to catch them end to end -- and missed removeTab's
// rename branch, where the strip went on listing a tab that had already exited
// (BR-45, reproduced by the boundary review as zero bytes written). The
// representation was defended in comments on both StripModel.Active and
// RenameField.Tab; the TRANSITION was the thing nothing enumerated.
//
// Two halves, and both are needed:
//
//   - TestEveryStripModelMutatorHasARepaintCase reads run.go and fails when a
//     method assigns m.tabs / m.active / m.rename without a case below. That is
//     what makes a method added later announce itself.
//   - TestEveryStripModelMutationRepaintsTheRow drives each case and asserts a
//     repaint carrying POST-mutation state actually reached the pane. A static
//     check alone would have passed the very defect this file exists for:
//     removeTab did call applyTakeover, just not on the branch that mattered.

// stripRepaintCase is one mutator and the proof it repaints.
type stripRepaintCase struct {
	// mutator is the run.go method name this case covers.
	mutator string
	// setup puts the mux into the state the mutation happens FROM. Whatever it
	// paints is discarded before drive runs, so `absent` can assert on the one
	// repaint under test rather than on the whole session.
	setup func(t *testing.T, m *terminalMux)
	// drive performs the mutation on a mux whose writer loop is running.
	drive func(t *testing.T, m *terminalMux)
	// want is a substring the repainted row must carry AFTER the mutation.
	want string
	// absent, when set, must NOT appear in what was written -- the pre-mutation
	// state that a missing repaint would leave on the row.
	absent string
	// needsPty marks a case that starts a real child process. Those fail under
	// a sandbox that forbids fork/exec, which is the documented class in this
	// repo, not a defect in the case.
	needsPty bool
}

func stripRepaintCases() []stripRepaintCase {
	// rename opens the editor and types into it THROUGH the mux, so m.rename
	// holds what the row should show. Typing without refreshRename leaves the
	// mux on the original text, which reads as "the repaint did not happen".
	rename := func(m *terminalMux, typed string) (int, RenameEditor) {
		id, editor, err := m.beginRename()
		if err != nil {
			panic(err)
		}
		for _, r := range typed {
			editor, _ = editor.Apply(RenameEvent{Kind: RenameInsert, Rune: r})
		}
		if typed != "" {
			m.refreshRename(id, editor)
		}
		return id, editor
	}
	return []stripRepaintCase{
		{
			mutator: "newTab",
			drive:   func(t *testing.T, m *terminalMux) { mustNewTab(t, m) },
			want:    "[terminal 1]",
			// "one" and "two" are still on the row; the new tab is what must
			// appear, so there is nothing pre-mutation to assert absent.
			needsPty: true,
		},
		{
			mutator: "switchRelative",
			drive:   func(t *testing.T, m *terminalMux) { m.previousTab() },
			want:    "[one] two",
			absent:  "[two]",
		},
		{
			mutator: "removeTab",
			drive:   func(t *testing.T, m *terminalMux) { onWriterLoop(m, func() { m.removeTab(1) }) },
			want:    "[two]",
			absent:  "one",
		},
		{
			// THE BRANCH BR-45 MISSED. A rename is open, so the screen is not
			// taken over -- and the takeover was the only repaint on this path.
			mutator: "removeTab",
			setup:   func(t *testing.T, m *terminalMux) { rename(m, "x") },
			drive:   func(t *testing.T, m *terminalMux) { onWriterLoop(m, func() { m.removeTab(1) }) },
			want:    "[rename: twox│]",
			absent:  "one",
		},
		{
			mutator: "beginRename",
			drive:   func(t *testing.T, m *terminalMux) { rename(m, "") },
			want:    "[rename: two│]",
			absent:  "[two]",
		},
		{
			mutator: "refreshRename",
			setup:   func(t *testing.T, m *terminalMux) { rename(m, "") },
			drive: func(t *testing.T, m *terminalMux) {
				editor := NewRenameEditor("two")
				editor, _ = editor.Apply(RenameEvent{Kind: RenameInsert, Rune: 'x'})
				m.refreshRename(2, editor)
			},
			want:   "[rename: twox│]",
			absent: "[rename: two│]",
		},
		{
			mutator: "finishRename",
			setup:   func(t *testing.T, m *terminalMux) { rename(m, "") },
			drive: func(t *testing.T, m *terminalMux) {
				if err := m.finishRename(2, RenameOutcome{Kind: RenameOutcomeCommit, Name: "built"}); err != nil {
					t.Fatal(err)
				}
			},
			want:   "[built]",
			absent: "[rename:",
		},
	}
}

// onWriterLoop runs fn ON the writer goroutine, which is where removeTab
// actually runs in production (its own doc says INLINE). Driving it from the
// test goroutine instead races the loop and inverts the order of the two paints,
// which reads as a defect in the code rather than in the harness.
func onWriterLoop(m *terminalMux, fn func()) {
	done := make(chan struct{})
	m.output <- ptyChunk{drained: done, onWriter: fn}
	<-done
}

func mustNewTab(t *testing.T, m *terminalMux) {
	t.Helper()
	if err := m.newTab(); err != nil {
		t.Skipf("newTab needs a real pty: %v", err)
	}
}

func TestEveryStripModelMutationRepaintsTheRow(t *testing.T) {
	for _, tt := range stripRepaintCases() {
		name := tt.mutator
		if tt.want != "" {
			name += " → " + tt.want
		}
		t.Run(name, func(t *testing.T) {
			m, rec := stripMux(t)
			defer close(m.done)
			if tt.setup != nil {
				tt.setup(t, m)
			}
			m.drainForTest() // discard whatever setup painted
			rec.reset()

			tt.drive(t, m)
			m.drainForTest()

			got := rec.String()
			if !strings.Contains(got, tt.want) {
				t.Fatalf("%s did not repaint the row with %q: %q", tt.mutator, tt.want, got)
			}
			if tt.absent != "" && strings.Contains(got, tt.absent) {
				t.Fatalf("%s left the pre-mutation state %q on the row: %q", tt.mutator, tt.absent, got)
			}
		})
	}
}

// TestEveryStripModelMutatorHasARepaintCase is the half that survives someone
// adding a method. It parses run.go and finds every *terminalMux method that
// ASSIGNS to one of the three fields the strip renders from; each must appear in
// the table above.
func TestEveryStripModelMutatorHasARepaintCase(t *testing.T) {
	covered := map[string]bool{}
	for _, c := range stripRepaintCases() {
		covered[c.mutator] = true
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "run.go", nil, 0)
	if err != nil {
		t.Fatalf("parse run.go: %v", err)
	}

	// The fields the strip's model is built from (stripModelLocked). tab.name is
	// reached through finishRename, which is in the table on its own account.
	modelFields := map[string]bool{"tabs": true, "active": true, "rename": true}

	var missing []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Body == nil {
			continue
		}
		star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		if id, ok := star.X.(*ast.Ident); !ok || id.Name != "terminalMux" {
			continue
		}
		mutates := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for _, lhs := range assign.Lhs {
				// m.tabs = …  and  m.active = …  reach here directly; an
				// indexed write (m.tabs[i] = …) arrives as an IndexExpr whose X
				// is the selector, so unwrap one level.
				if idx, ok := lhs.(*ast.IndexExpr); ok {
					lhs = idx.X
				}
				sel, ok := lhs.(*ast.SelectorExpr)
				if !ok {
					continue
				}
				recv, ok := sel.X.(*ast.Ident)
				if !ok || recv.Name != "m" || !modelFields[sel.Sel.Name] {
					continue
				}
				mutates = true
			}
			return true
		})
		if mutates && !covered[fn.Name.Name] {
			missing = append(missing, fn.Name.Name)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("these methods mutate the strip's model with no repaint case: %v\n"+
			"Every mutation owes a repaint (BR-45). Add a stripRepaintCase driving it, "+
			"or state at the call site why the row is repainted elsewhere.", missing)
	}
}

// A FRESH PAINT SUPERSEDES THE OWED ONE (BR-46).
//
// Two slots hold a pending paint -- `owed` (the coalescing slot) and `stripOwed`
// (the row-dirty debt) -- and they used to drain in the wrong order: the debt
// painted current state inline, then flushOwed wrote the OLDER row on top of it.
// The last thing on the wire was stale, contradicting writeOwn's own rule that
// "the freshest paint is the only one worth landing".
func TestAFreshPaintSupersedesTheOwedOne(t *testing.T) {
	m, rec := stripMux(t)
	defer close(m.done)

	// Owe a paint: a console write while the child's stream is mid-sequence.
	m.output <- ptyChunk{id: 2, data: []byte("\x1b[")}
	m.drainForTest()
	m.paintStrip()
	m.drainForTest()

	// Now change the model and let a row-dirty batch close the sequence, which
	// pays the debt inline. The owed row is older than that paint.
	m.mu.Lock()
	m.tabs[1].name = "built"
	m.mu.Unlock()
	rec.reset()
	m.output <- ptyChunk{id: 2, data: []byte("m\x1b[r"), rowDirty: true}
	m.drainForTest()

	got := rec.String()
	if !strings.Contains(got, "[built]") {
		t.Fatalf("the fresh row never landed: %q", got)
	}
	if i, j := strings.LastIndex(got, "[built]"), strings.LastIndex(got, "[two]"); j > i {
		t.Fatalf("a stale owed row landed AFTER the fresh one: %q", got)
	}
}

// The deferred-DIAGNOSTIC queue is bounded (BR-49).
//
// It sits behind a condition the child controls: unlike a mid-sequence chunk
// boundary, a held cursor save persists for as long as the child chooses not to
// restore. An unbounded append behind that is a queue with no ceiling and no
// drain. Bounded by dropping the OLDEST, because the newest report is the one
// that describes the pane's current state.
func TestTheOwedDiagnosticQueueIsBounded(t *testing.T) {
	m, _ := stripMux(t)
	defer close(m.done)

	// A child holding a cursor save: every console write defers from here on.
	m.output <- ptyChunk{id: 2, data: []byte("\x1b7")}
	m.drainForTest()

	for i := 0; i < maxOwedDiag*3; i++ {
		m.output <- ptyChunk{diag: []byte("zellij action failed\r\n")}
	}
	m.drainForTest()

	m.output <- ptyChunk{onWriter: func() {
		if len(m.owedDiag) > maxOwedDiag {
			t.Errorf("owedDiag grew to %d behind a held save; cap is %d", len(m.owedDiag), maxOwedDiag)
		}
	}}
	m.drainForTest()
}

// A repaint reaches the pane only through the gate, so a strip test that writes
// nothing proves nothing. This asserts the harness itself.
func TestStripMuxRecorderSeesTheStartupState(t *testing.T) {
	m, rec := stripMux(t)
	defer close(m.done)
	m.paintStrip()
	m.drainForTest()
	if !strings.Contains(rec.String(), "one [two]") {
		t.Fatalf("the harness recorded no strip at all: %q", rec.String())
	}
	if !strings.Contains(rec.String(), hostty.SetRegion(1, 23)) {
		t.Fatalf("the paint did not re-assert the region: %q", rec.String())
	}
}

var _ = io.Discard
var _ ptychild.Size
