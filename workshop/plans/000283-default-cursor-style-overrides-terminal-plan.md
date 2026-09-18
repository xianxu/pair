# Default Cursor Style Implementation Plan (#283)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A child that never sets a cursor style, or resets it (`ESC[0 q` / `ESC[ q`),
reaches the parent terminal as `ESC[0 q` ("use your configured default") instead of
pair's explicit `ESC[1 q`, while every explicit `ESC[1..6 q` survives verbatim.

**Architecture:** Make "terminal default" a first-class value at each seam, carried as
the zero value: the pair-owned vt fork gains `CursorDefault` as the zero `CursorStyle`
(renumbering the explicit styles onto DECSCUSR's shape families 1/2/3); the endpoint
publishes `Frame.Cursor.Shape` as that value unchanged (the `+1` that lost the default
disappears); `cursorEpilogue` emits `ESC[0 q` for Shape 0. RIS, a fresh screen and a
fresh alternate screen already reset the cursor to its zero value, so they reach the
default with no extra code.

**Tech Stack:** Go; `third_party/vt` (pair-owned fork of charmbracelet/x/vt);
`cmd/internal/terminal`; `cmd/internal/terminalqualify`.

---

## Design decision: fix it in vt, not at the endpoint

The issue asks how much of `third_party/vt` to touch versus tracking the default at
the endpoint through the `CursorStyle` callback. The callback cannot do it:

- it fires only on change, so `ESC[0 q` after an explicit blinking block is invisible;
- RIS replaces the cursor with `Cursor{}` without calling it;
- the endpoint stopped using cursor callbacks in #255 M2 BR8 — it reads the
  authoritative `Emulator.Cursor()` state because callbacks were incomplete.

The fork is pair-owned (`PAIR_PATCHES.md`), so the state belongs where the other
cursor state lives. The vt diff is two lines of semantics: the enum gains a zero
member, and the DECSCUSR handler stops mapping 0/absent to 1.

**Why renumber instead of a `StyleSet bool` (ARCH-ORDER).** A separate flag makes
`Style × Steady × StyleSet` = 12 representable states for 7 legal ones (default +
six explicit), and every reader has to know to consult the flag first. A zero-valued
`CursorDefault` gives 8 representable states. The one leftover, `(CursorDefault,
Steady=true)`, is never produced: the handler gives the default blink=true, which is
the zero value's `Steady=false`. So `ESC[0 q`, RIS and a fresh screen all leave the
same `Cursor` value. The renumbering also makes vt's style numbers equal to
`Frame.Cursor.Shape`, so the endpoint mapping becomes the identity. The `+1` there was
the site that lost the default. Upstream code refers to these constants only by name,
except for the one handler we are rewriting. That keeps the fork's rebase surface
unchanged.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `vt.CursorStyle` (`CursorDefault` zero member) | `third_party/vt/cursor.go` | modified |
| DECSCUSR handler | `third_party/vt/handlers.go` | modified |
| `terminal.Cursor` (Shape 0 = parent default; Blink only for explicit shapes) | `cmd/internal/terminal/frame.go` | modified |
| `Frame.Validate` (rejects Shape 0 with Blink) | `cmd/internal/terminal/frame.go` | modified |
| `cursorEpilogue` | `cmd/internal/terminal/render.go` | modified |

- **`vt.CursorStyle`** — DECSCUSR shape family: `CursorDefault`=0, `CursorBlock`=1,
  `CursorUnderline`=2, `CursorBar`=3. `Cursor.Steady` is meaningful only for an
  explicit style.
  - **Relationships:** one per `Screen.cur`/`Screen.saved`. Primary and alternate
    screens each hold one, as they already do.
  - **DRY rationale:** one numbering shared by vt state and frame publication, so the
    `vt → Frame` shape translation disappears instead of being duplicated.
  - **Future extensions:** a terminal-global cursor style (xterm keeps one style
    across buffers; this fork keeps one per screen) would move this field, not
    change its values. Out of scope.
- **DECSCUSR handler** — absent/0 → `(CursorDefault, blink)`; n∈1..6 →
  `(CursorStyle((n+1)/2), n odd)`; n>6 → ignored (state unchanged, unchanged from
  today).
- **`terminal.Cursor` / `Frame.Validate`** — Shape 0 means the parent's configured
  shape *and* blink, so Blink is false for it. `Validate` rejects `Shape 0 && Blink`,
  so each visible state has one representation, and `Render`'s `prev.Cursor !=
  next.Cursor` dirtiness check stays exact.
- **`cursorEpilogue`** — Shape 0 → `ESC[0 q`; Shape s → `ESC[2s-blink q`.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Endpoint.capture` cursor publication | `cmd/internal/terminal/endpoint.go` | modified | vt emulator state |
| `Presenter.Select` → parent writer | `cmd/internal/terminal/presenter.go` | unchanged (acceptance seam) | parent TTY (`ttyio.Writer`) |
| qualifier candidate observation | `cmd/internal/terminalqualify/candidate.go` | unchanged code, literals change | vt emulator |

- **`Endpoint.capture`** — `Shape: int(cur.Style)`, `Blink: cur.Style !=
  vt.CursorDefault && !cur.Steady`.
- **`Presenter.Select`** — the acceptance test writes to it through the existing
  stateful parent fake, `ttyio.NewFake()`. The independent xterm oracle
  (`tests/terminal-oracle`, @xterm/headless 5.5) cannot observe the default: its
  `setCursorStyle` maps 0 to 1 and overwrites `options.cursorStyle`. The live
  conformance check is therefore the operator's Ghostty smoke.

### Consumer enumeration (ARCH-PURPOSE shadow-sweep)

Consumers of `Frame.Cursor.Shape/Blink`:

- `cursorEpilogue`. Covers both `Render` (`render.go:111`) and `HistoryRender.Emit`
  (`history_render.go:409`).
- `Frame.Validate`.
- Producers: `Endpoint.capture`, and `couchtty/console_menu.go:214-216`. The
  console menu publishes `Cursor{Visible: true}` (Shape 0). Today that renders
  `ESC[2 q` (a steady block). After this change it renders `ESC[0 q`, the operator's
  configured cursor. The menu relied on the documented "0/default" meaning, so this
  is the intended consequence, not a regression. It gets a line in the Log.

Consumers of `vt.Cursor.Style/Steady`:

- `Endpoint.capture`.
- `terminalqualify` candidate (`"cursor-style": "%d,%t"`, raw state).
- `Callbacks.CursorStyle`. No pair caller registers it. It now reports
  `CursorDefault`, and it fires on explicit→default because that is a value change.
- The vt test `TestPairCursorStyleRejectsInvalid`.

### Operating envelope and other lenses

- **ARCH-CONSTRAINTS.** Per-frame path, keystroke latency class. `ESC[0 q` is the
  same length as `ESC[n q`. There is no new work per frame, and the #262
  per-frame re-assert policy is unchanged.
- **ARCH-SECURE.** DECSCUSR parameters come from untrusted child bytes. The existing
  n>6 rejection stays, and a missing parameter parses to 0. No secrets are involved.
- **ARCH-ORDER.** The cursor-style state (7 legal values) changes on these events:
  - DECSCUSR n sets it; an n>6 request is ignored.
  - RIS sets the default.
  - DECSC/DECRC save and restore it.
  - 1049 entry copies the primary cursor; 47/1047 entry keeps the alternate
    screen's own.

  The endpoint test replays each stream at every byte split. The event most likely
  to be mishandled is `ESC[0 q` while the state is already an explicit blinking
  block — the callback-invisible case, which gets its own test.
- **ARCH-FUNERAL.** N/A. This creates no artifact, handle or record; it changes the
  value of an existing in-memory field.
- **ARCH-MOCK.** Parent terminal: `ttyio.NewFake` (stateful writer), plus the
  operator's Ghostty smoke as live conformance (see above).

---

## Chunk 1: vt, endpoint, frame, renderer

### Task 1: vt records the default

**Files:**
- Modify: `third_party/vt/cursor.go:5-24`
- Modify: `third_party/vt/handlers.go:845-861`
- Modify: `third_party/vt/pair_edges_test.go:258-265`
- Create: `third_party/vt/pair_cursor_style_test.go`
- Modify: `third_party/vt/PAIR_PATCHES.md` (patch-family entry)

- [ ] **Step 1: Write the failing test** (`pair_cursor_style_test.go`)

```go
package vt

import "testing"

// DECSCUSR 0 or an absent parameter hands the cursor back to the host
// terminal's configured default; RIS and a fresh screen start there (#283).
func TestPairCursorStyleCarriesHostDefault(t *testing.T) {
	cases := []struct {
		name, stream string
		style        CursorStyle
		steady       bool
	}{
		{"fresh", "", CursorDefault, false},
		{"zero after bar", "\x1b[6 q\x1b[0 q", CursorDefault, false},
		{"absent after underline", "\x1b[4 q\x1b[ q", CursorDefault, false},
		{"zero after blinking block", "\x1b[1 q\x1b[0 q", CursorDefault, false},
		{"reset", "\x1b[6 q\x1bc", CursorDefault, false},
		{"blinking block", "\x1b[1 q", CursorBlock, false},
		{"steady block", "\x1b[2 q", CursorBlock, true},
		{"blinking underline", "\x1b[3 q", CursorUnderline, false},
		{"steady underline", "\x1b[4 q", CursorUnderline, true},
		{"blinking bar", "\x1b[5 q", CursorBar, false},
		{"steady bar", "\x1b[6 q", CursorBar, true},
		{"unknown ignored", "\x1b[3 q\x1b[7 q", CursorUnderline, false},
	}
	for _, tc := range cases {
		e := NewEmulator(4, 2)
		e.WriteString(tc.stream)
		if c := e.Cursor(); c.Style != tc.style || c.Steady != tc.steady {
			t.Errorf("%s: style=%d steady=%v want %d/%v", tc.name, c.Style, c.Steady, tc.style, tc.steady)
		}
		e.Close()
	}
}

// The callback fires on every change of state, including an explicit blinking
// block returning to the default, which the old mapping made indistinguishable.
func TestPairCursorStyleCallbackReportsDefault(t *testing.T) {
	e := NewEmulator(4, 2)
	defer e.Close()
	var got []CursorStyle
	e.SetCallbacks(Callbacks{CursorStyle: func(s CursorStyle, _ bool) { got = append(got, s) }})
	e.WriteString("\x1b[1 q\x1b[0 q")
	if len(got) != 2 || got[0] != CursorBlock || got[1] != CursorDefault {
		t.Fatalf("callbacks %v", got)
	}
}
```

- [ ] **Step 2: Run it and confirm that it fails.**
  `(cd third_party/vt && go test -run 'TestPairCursorStyle' ./...)` should fail to
  compile with `undefined: CursorDefault`.

- [ ] **Step 3: Implement**

`cursor.go`:

```go
// CursorStyle is a DECSCUSR shape family. The zero value leaves shape and blink
// to the host terminal's configured default (DECSCUSR 0, RIS; pair #283).
type CursorStyle int

// Cursor styles.
const (
	CursorDefault CursorStyle = iota
	CursorBlock
	CursorUnderline
	CursorBar
)
```

and `Steady bool // Not blinking; meaningful only for an explicit Style`.

`handlers.go` DECSCUSR:

```go
	e.RegisterCsiHandler(ansi.Command(0, ' ', 'q'), func(params ansi.Params) bool {
		// Set Cursor Style [ansi.DECSCUSR]. Absent or 0 is the host default,
		// not a blinking block (pair #283); 1,2 block, 3,4 underline, 5,6 bar,
		// odd codes blink.
		n, _, _ := params.Param(0, 0)
		if n > 6 {
			return false
		}
		e.scr.setCursorStyle(CursorStyle((n+1)/2), n == 0 || n%2 == 1)
		return true
	})
```

`pair_edges_test.go` `TestPairCursorStyleRejectsInvalid`: write `\x1b[3 q` first,
then the invalid code, and assert the style is still `CursorUnderline`. This is
stronger than comparing against a fresh value, which now coincides with the default.

- [ ] **Step 4: Run the tests and confirm that they pass.**
  `(cd third_party/vt && go test ./... && go test -race ./...)` should PASS.

- [ ] **Step 5: `PAIR_PATCHES.md`.** Add a bullet under Patch families:
  `cursor.go`, `handlers.go`: DECSCUSR 0/absent records `CursorDefault` (the
  zero value, so RIS/fresh screens start there). The explicit styles are
  renumbered onto DECSCUSR shape families 1/2/3, and the callback reports the
  default.

### Task 2: endpoint publishes Shape 0; frame rejects a blinking default

**Files:**
- Modify: `cmd/internal/terminal/endpoint.go:227-228`
- Modify: `cmd/internal/terminal/frame.go:15-21,91`
- Modify: `cmd/internal/terminal/endpoint_test.go:294-331`
- Modify: `cmd/internal/terminal/frame_test.go` (one Validate case)

- [ ] **Step 1: Write the failing tests.** In
  `TestEndpointAuthoritativeCursorAcrossResetRestoreAndBuffers`, the cases that
  used to encode the bug now expect the default: `reset`, `alternate-entry` and
  `reset-held` become `Cursor{Visible: true}`. Add these cases:

```go
		{"never-set", "", Cursor{Visible: true}},
		{"zero-after-blinking-block", "\x1b[1 q\x1b[0 q", Cursor{Visible: true}},
		{"absent-after-steady-underline", "\x1b[4 q\x1b[ q", Cursor{Visible: true}},
		{"explicit-blinking-block", "\x1b[0 q\x1b[1 q", Cursor{Visible: true, Blink: true, Shape: 1}},
		{"unknown-ignored", "\x1b[3 q\x1b[7 q", Cursor{Visible: true, Blink: true, Shape: 2}},
```

(`saved`, `alternate-return` and `alternate-retained` keep their literals. Their
Shape values do not change under the renumbering.) The `for split := 0; split <=
len(tc.stream)` loop already replays every byte split, and `never-set` runs one
split.

`frame_test.go`: a frame whose cursor is `Cursor{Visible: true, Blink: true}`
(Shape 0) fails `Validate()`, and the same frame with Shape 1 passes.

- [ ] **Step 2: Run the tests and confirm that they fail.**
  `go test ./cmd/internal/terminal -run 'TestEndpointAuthoritativeCursor|Validate'`
  should FAIL with `cursor{... Shape:1}` where `{... Shape:0}` was expected.

- [ ] **Step 3: Implement.** `endpoint.go`:

```go
	cur := e.backend.Cursor()
	// vt's CursorStyle is the frame's shape family; the default carries no blink.
	f.Cursor = Cursor{X: cur.X, Y: cur.Y, Visible: !cur.Hidden, Blink: cur.Style != vt.CursorDefault && !cur.Steady, Shape: int(cur.Style)}
```

`frame.go` doc:

```go
// Cursor coordinates are zero based. Shape is DECSCUSR's shape family: 1 block,
// 2 underline, 3 bar, or 0 for the parent terminal's configured default shape
// and blink. Blink applies only to an explicit shape: a default cursor carries
// Blink false, so each visible state has one representation (#283).
```

`Validate`: append `|| f.Cursor.Shape == 0 && f.Cursor.Blink` to the cursor clause.

- [ ] **Step 4: Run the tests and confirm that they pass.** `go test ./cmd/internal/terminal`.

### Task 3: renderer emits `ESC[0 q`; acceptance at the endpoint → parent seam

**Files:**
- Modify: `cmd/internal/terminal/render.go:22-38`
- Modify: `cmd/internal/terminal/render_test.go` (epilogue table)
- Modify: `cmd/internal/terminal/presenter_test.go` (endpoint → parent acceptance)

- [ ] **Step 1: Write the failing tests.**

```go
func TestCursorEpilogueCarriesDECSCUSR(t *testing.T) {
	for _, tc := range []struct {
		c    Cursor
		code int
	}{
		{Cursor{}, 0}, {Cursor{Shape: 1, Blink: true}, 1}, {Cursor{Shape: 1}, 2},
		{Cursor{Shape: 2, Blink: true}, 3}, {Cursor{Shape: 2}, 4},
		{Cursor{Shape: 3, Blink: true}, 5}, {Cursor{Shape: 3}, 6},
	} {
		if got := cursorEpilogue(tc.c); !strings.Contains(got, fmt.Sprintf("\x1b[%d q", tc.code)) {
			t.Errorf("%+v: %q lacks DECSCUSR %d", tc.c, got, tc.code)
		}
	}
}

var decscusr = regexp.MustCompile(`\x1b\[(\d*) q`)

// A child's DECSCUSR reaches the parent verbatim through the production
// endpoint and presenter; a default (never set, 0, absent, RIS) arrives as
// ESC[0 q, never an explicit block (#283).
func TestChildCursorStyleReachesParentVerbatim(t *testing.T) {
	for _, tc := range []struct{ name, child, want string }{
		{"never-set", "", "0"}, {"zero", "\x1b[0 q", "0"}, {"absent", "\x1b[5 q\x1b[ q", "0"},
		{"reset", "\x1b[6 q\x1bc", "0"}, {"zero-after-block", "\x1b[1 q\x1b[0 q", "0"},
		{"1", "\x1b[1 q", "1"}, {"2", "\x1b[2 q", "2"}, {"3", "\x1b[3 q", "3"},
		{"4", "\x1b[4 q", "4"}, {"5", "\x1b[5 q", "5"}, {"6", "\x1b[6 q", "6"},
	} {
		parent := ttyio.NewFake()
		e, err := NewEndpoint(tc.name, Geometry{8, 4}, ttyio.NewFake())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := e.Feed([]byte(tc.child), time.Now()); err != nil {
			t.Fatal(err)
		}
		p := NewPresenter(parent, CouchAnyMotion)
		if err := p.Select(context.Background(), e, Geometry{8, 5}, make([]Cell, 8)); err != nil {
			t.Fatal(err)
		}
		all := decscusr.FindAllStringSubmatch(string(parent.Bytes()), -1)
		if len(all) == 0 || all[len(all)-1][1] != tc.want {
			t.Errorf("%s: parent DECSCUSR %q want %q", tc.name, all, tc.want)
		}
		e.Close()
	}
}
```

(`TestCursorEpilogueCarriesDECSCUSR` goes in `render_test.go`; the acceptance test
goes in `presenter_test.go`, which already imports `context`, `ttyio` and `time` —
add `regexp`. The `NewEndpoint`/`NewPresenter`/`Select` shape is exactly
`render_oracle_test.go:185-200`'s. It reads the parent bytes before `Release`,
which writes its own `ESC[0 q`, so only the frame epilogue is observed.)

- [ ] **Step 2: Run the tests and confirm that they fail.**
  `go test ./cmd/internal/terminal -run 'CursorEpilogue|ReachesParent'`. `{Cursor{}, 0}`
  should fail (the epilogue emits `ESC[2 q`), and so should the four default
  acceptance cases.

- [ ] **Step 3: Implement.** `render.go`:

```go
// cursorEpilogue places the cursor and restores its shape and visibility after a
// frame painted with the cursor hidden. Shape 0 hands shape and blink back to the
// parent's configured default (DECSCUSR 0, #283).
func cursorEpilogue(c Cursor) string {
	code := 0
	if c.Shape != 0 {
		code = c.Shape * 2
		if c.Blink {
			code--
		}
	}
	s := fmt.Sprintf("\x1b[%d;%dH\x1b[%d q", c.Y+1, c.X+1, code)
	...
```

- [ ] **Step 4: Run the tests and confirm that they pass.**
  `go test ./cmd/internal/terminal`.

### Task 4: qualifier literals, sweep, docs, full verification

**Files:**
- Modify: `cmd/internal/terminalqualify/input_cases.go:42` (`"2,true"` → `"3,true"`)
- Modify: `cmd/internal/terminalqualify/screen_cases.go:17-18` (`"1,true"` →
  `"2,true"`, `"2,false"` → `"3,false"`). Line 16's RIS literal `"0,true"` keeps its
  text but now means the default, which was the point of that predicate.
- Modify: `atlas/terminal.md` (cursor-style sentence in the #262 M2 paragraph)

- [ ] **Step 1: Update the literals.** Run `go test ./cmd/internal/terminalqualify/...`
  and expect PASS. Then run `go run ./cmd/probes/terminalqualify` and confirm that
  it still reports all 68 executable cases passing. Its exit status 1, for the
  separately tracked integration obligations, is expected (PAIR_PATCHES
  Verification).
- [ ] **Step 2: Sweep.** Run `grep -rn 'Shape\b\|CursorBlock\|\[1 q\|blinking block'`
  over `cmd/ third_party/vt atlas/` and confirm that no consumer still assumes
  "never 0" or "fresh = block".
- [ ] **Step 3: Atlas.** In `atlas/terminal.md`, after "placing the cursor with its
  shape (DECSCUSR)", add: "A child that never set a shape, or reset it, is
  published as Shape 0 and re-asserted as `ESC[0 q`, so the parent's configured
  cursor survives (#283)".
- [ ] **Step 4: Full suite.** Run `make test > $TMPDIR/mt.log 2>&1; echo $?` (per
  memory: no pipe to head, scrub the session env) and `go test ./...`. Expect
  green.
- [ ] **Step 5: Commit.** `#283: carry the terminal-default cursor style through vt → frame → parent`.
- [ ] **Step 6: Operator smoke.** Ask the operator to run a plain shell under couch
  with Ghostty `cursor-style = bar` and confirm that it shows a bar. Also check that
  the couch switcher's text-input caret now follows the configured style.
