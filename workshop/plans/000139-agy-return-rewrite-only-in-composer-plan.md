# Agy Return Rewrite Only in Composer Implementation Plan

> **Superseded plan record:** The original Agy-only design below is retained as
> history. Its steps are non-executable; the 2026-08-17 revision at the end is
> authoritative.

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Gate Antigravity (`agy`) plain Return rewriting behind positive composer detection with visible cursor, falling back to bare CR on inactive states and overlays, backed by a dedicated pure tracker and stateful protocol fake.

**Architecture:** A pure VT100/ANSI screen state machine (`agyComposerTracker`) in `cmd/internal/wrapcmd/agy_composer.go` parses terminal escape sequences, tracks cursor coordinates (including relative moves like `CUU`, `CHA`, `RI`), visibility (`DECTCEM`), and per-row painted chrome attributes with explicit screen-mutation invalidation rules (`ED`, `EL`, non-border text overwrite, `setWinsize` row pruning). `wrap.go` gates plain Return newline remap accordingly (ARCH-DRY, ARCH-PURE, ARCH-MOCK).

**Tech Stack:** Go, standard library (`sync`, `strconv`, `bytes`, `strings`), `cmd/internal/ansi`, `cmd/internal/adapt`.

---

## Core concepts

### Historical conceptual core (superseded; not implementation entities)

| Name | Lives in | Status |
|------|----------|--------|
| `agyComposerTracker` | `cmd/internal/wrapcmd/agy_composer.go` | new |
| `agyComposerState` | `cmd/internal/wrapcmd/agy_composer.go` | new |
| `agySessionFake` | `cmd/internal/wrapcmd/agy_return_test.go` | new |

- **`agyComposerTracker`** — Concurrency-safe terminal grid model tracking dimensions, cursor coordinates (row, col), visibility, line/display erases, and Agy chrome markers across chunks.
  - **Relationships:** Owned by `proxy` (1:1 per wrapped Agy agent session).
  - **DRY rationale (ARCH-DRY):** Mirrors the proven structure of `codexComposerTracker` and `museComposerTracker` while providing full relative cursor movement (`CUU`, `CHA`, `RI`) and screen mutation invalidation rules required by React-Ink / Agy.
  - **Purity (ARCH-PURE):** Pure deterministic state machine tested via byte feeds without OS or PTY mocks.
  - **Invalidation rules:** `ED` (`J`) deletes prompt/border rows in erased ranges; `EL` (`K`) clears prompt/border marks for `cursorRow`; non-border text printing at `cursorRow` clears `agyBorderRows[cursorRow]`; `setWinsize` prunes row keys $> rows$.
  - **Locality & Height Bounds:** `active()` checks `cursorVisible && cursorRow > 0 && (isNearby(cursorRow) || isEnclosed(cursorRow))` with strict vertical height limit $\le 25$ rows.

- **`agyComposerState`** — Immutable snapshot of terminal dimensions, cursor position, visibility, and row-level chrome flags.
  - **Predicates:** `active() bool`.

- **`agySessionFake`** — Stateful external terminal-protocol double modeling Agy's 4 UI lifecycle phases (ARCH-MOCK):
  1. Startup banner and initial composer box (`───\n>\n───\n? for shortcuts\x1b[2A\x1b[2C\x1b[?25h`).
  2. Multi-line input editing with cursor repositioning.
  3. Output streaming / thinking state with hidden cursor (`\x1b[?25l`).
  4. Permission / picker overlay prompts ("Do you want to proceed?").

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `proxy.emitPlainCR` | `cmd/internal/wrapcmd/wrap.go` | modified | Stdin translation pump |
| `proxy.handleChunk` | `cmd/internal/wrapcmd/wrap.go` | modified | Master PTY output stream |
| `proxy.setWinsize` | `cmd/internal/wrapcmd/wrap.go` | modified | Terminal resize signals |

- **`proxy.emitPlainCR`** — Stdin translation function for Enter keystrokes.
  - **Injected into:** `proxy.translateChunk`.
  - **Behavior:** Checks `pickerActive` first (bare `\r`), then `p.agyComposerActive()`. Returns bare `\r` if inactive, or `p.sendKM.plainCR` (`\n`) if active.

- **`proxy.setWinsize`** — Real resize handler at `wrap.go:2000`.
  - **Behavior:** Calls `p.ensureAgyComposer().resize(int(ws.Rows), int(ws.Cols))` when `agentBasename == "agy"`.

---

## Tasks

### Task 1: Pure Agy Composer Tracker (`agyComposerTracker`)

**Files:**
- Create: `cmd/internal/wrapcmd/agy_composer.go`
- Test: `cmd/internal/wrapcmd/agy_composer_test.go`

- Historical step: Write unit tests for `agyComposerTracker` using risky-function strategies.

Create `cmd/internal/wrapcmd/agy_composer_test.go` testing three named risky functions:
1. `agyComposerTracker.feed`: Test parser resilience over arbitrary malformed, split UTF-8 (`\xe2\x94\x80`), and unterminated ANSI sequences (`\x1b[...`) with bounded carryover.
2. `agyComposerTracker.applyEscape`: Test cursor movement (`CUP`, `CHA`, `VPA`, `CUU`, `CUD`, `CUF`, `CUB`, `RI`), visibility toggles (`?25h`/`?25l`), and screen/line erase invalidations (`ED`, `EL`).
3. `agyComposerState.active`: Test positive and negative boundary conditions across single-line prompt, multi-line enclosed composer box, hidden cursor, stale distant chrome, and erased display.

- Historical step: Run tests to verify they fail.

Run: `go test -v ./cmd/internal/wrapcmd -run TestAgyComposer`
Expected: FAIL (compilation error: `agyComposerTracker` undefined).

- Historical step: Implement `agyComposerTracker`.

Create `cmd/internal/wrapcmd/agy_composer.go`:
- Implement `agyComposerTracker` with `sync.Mutex`, `rows`, `cols`, `cursorRow`, `cursorCol`, `cursorVisible`, `pending`, and row maps:
  - `promptRows map[int]bool`
  - `borderRows map[int]bool`
- Implement `resize(rows, cols int)` with row/column clamping and row pruning.
- Implement `feed(data []byte)` parsing ANSI escapes:
  - 2-byte escape: `\x1bM` (`RI` -> `t.cursorRow--`, clamped).
  - CSI sequences: `CUP` (`H`, `f`), `CHA` (`G`), `VPA` (`d`), `CUU` (`A`), `CUD` (`B`), `CUF` (`C`), `CUB` (`D`) — all defaulting omitted parameter to 1.
  - Cursor visibility: `DECTCEM` (`?25h` -> true, `?25l` -> false).
  - Erase invalidations: `ED` (`J` mode 0/1/2/3 deletes prompt/border rows in range), `EL` (`K` deletes prompt/border marks on cursorRow).
  - Chrome detection: `\xe2\x94\x80` horizontal border rule ($\ge 5$ count per row), `>` anchored at `col <= 6`.
- Implement `state() agyComposerState` returning snapshot.
- Implement `agyComposerState.active() bool`: `cursorVisible && cursorRow > 0 && (isNearby(cursorRow) || isEnclosed(cursorRow))`.

- Historical step: Run tests to verify they pass.

Run: `go test -v ./cmd/internal/wrapcmd -run TestAgyComposer`
Expected: PASS.

- Historical step: Commit.

```bash
git add cmd/internal/wrapcmd/agy_composer.go cmd/internal/wrapcmd/agy_composer_test.go
git commit -m "wrap: #139: implement agy composer tracker and mutation invalidation rules

Co-Authored-By: Antigravity <antigravity@google.com>"
```
## Revisions

### 2026-08-17T13:50:00-07:00 — Unified harness TTY Return routing

**Reason.** The inherited Agy prototype duplicated terminal parsing already
present in the Codex and Muse trackers and admitted stale/weak evidence. This
revision is authoritative: one `x/vt` screen model, one harness profile
registry, harness-specific pure recognizers, and one fail-safe Return decision.

# Unified Harness TTY Return Routing Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Return customization derive from one harness TTY profile and one
shared terminal snapshot while preserving Codex/Muse behavior and adding a
strict positive Agy composer gate.

**Architecture:** `terminalModel` wraps the existing
`github.com/charmbracelet/x/vt` emulator and publishes atomic immutable
snapshots. A `harnessTTYProfile` registry owns all Return customization; pure
recognizers interpret snapshots and `decidePlainReturn` applies overlay,
positive-gate, and legacy precedence. `proxy` remains a thin feed/resize/input
shell (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

**Tech Stack:** Go, `github.com/charmbracelet/x/vt`,
`github.com/charmbracelet/ultraviolet`, `github.com/creack/pty`, standard
library synchronization/IO, and existing `adapt` telemetry.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `terminalSnapshot` | `cmd/internal/wrapcmd/terminal_model.go` | new |
| `harnessTTYProfile` | `cmd/internal/wrapcmd/harness_tty.go` | new |
| `returnDecision` / `decidePlainReturn` | `cmd/internal/wrapcmd/harness_tty.go` | new |
| `codexComposerActive` | `cmd/internal/wrapcmd/composer_recognizers.go` | new |
| `museComposerActive` | `cmd/internal/wrapcmd/composer_recognizers.go` | new |
| `agyComposerActive` | `cmd/internal/wrapcmd/composer_recognizers.go` | new |
| `codexComposerTracker` | `cmd/internal/wrapcmd/codex_composer.go` | deleted |
| `museComposerTracker` | `cmd/internal/wrapcmd/muse_composer.go` | deleted |

- **`terminalSnapshot`** — immutable dimensions, zero-based cursor position,
  cursor visibility, active-screen identity, and deep-copied `uv.Cell` grid
  captured under one lock. Recognizers never touch the emulator.
- **`harnessTTYProfile`** — one immutable registration per harness: complete
  `sendKeymap`, overlay detector, gate policy, optional recognizer, and
  image-capture overlay capability. It replaces the parallel registries and
  agent-name routing branches (ARCH-DRY).
- **`returnDecision` / `decidePlainReturn`** — pure routing result containing
  bytes, adaptation outcome/reason, and whether overlay state clears. Overlay
  sends bare CR; a positive composer sends `plainCR`; an inactive/unknown
  positive gate sends bare CR; a legacy profile retains `plainCR`. Alt+Return
  remains `altCR`.
- **Recognizers** — pure predicates over one snapshot. Codex preserves its
  painted-background locality rule. Muse requires a qualified current-screen
  composer signature derived from checked-in literal capture evidence; an
  unrelated nearby `›` is never sufficient. Agy requires two overlapping
  contiguous `─` runs (minimum five), an anchored prompt in columns 0-5 between
  them, maximum height 25, and a visible cursor inside the same box.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `terminalModel` | `cmd/internal/wrapcmd/terminal_model.go` | new | `x/vt.Emulator` and reply pipe |
| `proxy.ttyProfile` / `proxy.terminal` | `cmd/internal/wrapcmd/wrap.go` | modified | PTY output, resize, stdin routing |
| `harnessSessionFake` | `cmd/internal/wrapcmd/harness_tty_integration_test.go` | new | stateful harness byte stream |
| `TestHarnessTTYLiveConformance` | `cmd/internal/wrapcmd/harness_tty_live_test.go` | new | installed harness CLI in a PTY |

- **`terminalModel`** owns one mutex, the emulator, a chunk-safe explicit
  control observer, a reply-drainer goroutine, and idempotent shutdown. `Feed`,
  `Resize`, `Snapshot`, and `Close` serialize on the same lock. Creation starts
  `io.Copy(io.Discard, emulator)` only after checking that the `io.Writer`
  returned by `Emulator.InputPipe()` also implements `io.Closer`; construction
  fails if the pinned dependency loses that capability. Shutdown closes the
  retained checked closer, joins the drainer, then closes the emulator.
  Visibility starts false and only an explicit chunk-framed `?25h` authorizes
  it; hide, reset, and alt-screen transitions clear it.
- **`proxy` integration** resolves a profile once when remapping is enabled,
  creates a terminal only for positive-gated profiles, feeds raw output, resizes
  with `(cols, rows)`, and closes it in the existing teardown defer.
- **`harnessSessionFake`** drives startup, multiline edit, hidden/busy, overlay,
  and reset transitions through production `handleChunk` (ARCH-MOCK).
- **Live conformance** is opt-in and launches `PAIR_LIVE_HARNESS` under
  `creack/pty`; ordinary unit CI never contacts an external harness.

## Chunk 1: Dependency and shared substrate

### Task 0: Clear the Muse ownership dependency

**Files:**
- Existing issue: `workshop/issues/000140-muse-return-rewrite-only-in-composer.md`

- [x] **Step 1: Resolve #140 through its own SDLC run**

Do not edit #140 inside the #139 transaction. Its implementation landed before
claim as `ab736d1`, so first run the binary-owned fresh review over the exact
lost window:

```bash
sdlc judge milestone-review --issue 140 --base ab736d1^ --head ab736d1 --agent codex
```

Record the verdict and exact window in #140's Log. Fix any Critical/Important
finding through #140 before proceeding. Then, in a clean checkout or isolated
worktree, follow `sdlc claim --issue 140`, reconcile its checklist to
`ab736d1`, rerun its focused tests, and invoke `sdlc close --issue 140
--verified '<behavior evidence plus ab736d1^..ab736d1 review>'`. Use only the
precise actual-hours waiver named by `sdlc` if the pre-claim implementation
cannot be measured. Resume #139 only after #140 is `codecomplete` or `done`.

- [x] **Step 2: Re-enter #139 implementation through the gate**

Run: `sdlc change-code --issue 139`

Expected: revised plan-quality runs before estimation. After it passes, derive
and install the expanded estimate requested by the command; never reuse the
superseded 0.57h estimate.

### Task 1: Atomic shared terminal snapshots

**Files:**
- Create: `cmd/internal/wrapcmd/terminal_model.go`
- Create: `cmd/internal/wrapcmd/terminal_model_test.go`

- [x] **Step 1: RED/GREEN constructor and empty snapshot**

Define the desired API:

```go
model, err := newTerminalModel(80, 24)
if err != nil { t.Fatal(err) }
t.Cleanup(func() { _ = model.Close() })
_ = model.Feed(raw)
snapshot := model.Snapshot()
```

`newTerminalModel`: vary dimensions and the injected input-pipe capability; the
guard is either one immutable in-bounds empty snapshot or a constructor error
before any drainer starts. Validate dimensions with one pure overflow-safe
helper before calling x/vt: both axes must be positive and no greater than 4096,
and area must be no greater than 262,144 cells (`width > maxCells/height`, never
unchecked multiplication). Observe RED, then implement only that contract.

- [x] **Step 2: RED/GREEN feed and immutable cell copying**

`terminalModel.Feed` / `Snapshot`: fuzz arbitrary byte streams and chunk
partitions seeded from supported harness captures; the guard is bounded,
nonblocking, panic-free, bounds-safe snapshots plus deep-cloned cells that
cannot mutate a later snapshot. Do not require whole-grid chunk equivalence:
x/vt flushes extended graphemes at each `Write`, so even valid ZWJ sequences can
render differently across caller chunks. Remove the fuzz oracle that compares
whole snapshots whenever `utf8.Valid(raw)`, retain its safety/coherence/deep-copy
checks for every partition, and add a deterministic `👩‍💻` regression proving
the permitted grid divergence while both snapshots remain coherent and
in-bounds. Task 6 instead requires recognizer and Return-decision equivalence at
every split of literal harness streams. Observe RED, then minimally wire
`Emulator.Write` and cell cloning.

- [x] **Step 3: RED/GREEN explicit control observation**

`terminalControlObserver.Feed`: fuzz arbitrary malformed/split escape streams
seeded with explicit show/hide, reset, alternate-screen, UTF-8, C1 CSI, and
OSC/DCS controls; the guard is bounded parser storage, chunking equivalence, and
fail-closed visibility until a complete top-level explicit show sequence. Use a
bounded `github.com/charmbracelet/x/ansi.Parser`, matching x/vt state semantics;
emulator callbacks never authorize production visibility. Observe RED, then
implement the minimal observer.

- [x] **Step 4: RED/GREEN resize and active-screen snapshot**

`terminalModel.Resize` / `Snapshot`: generate dimensions and screen-identity
transitions; the guard is one atomic bounds-safe snapshot whose visibility
remains fail-closed across replacement. Reuse the constructor's dimension
validator before x/vt; rejected resizes preserve dimensions, cells, cursor,
visibility, and active-screen identity. Observe RED, then implement resize and
`AltScreen` tracking.

- [x] **Step 5: RED/GREEN reply draining and deterministic close**

`newTerminalModel` / `Close`: generate emulator reply-producing streams and
repeated shutdown schedules under a timeout; the guard is nonblocking Feed,
idempotent close, and deterministic post-close errors/final snapshot. Check
`replyWriter := Emulator.InputPipe()` with `replyCloser, ok :=
replyWriter.(io.Closer)` before starting the drainer; on `!ok`, close the unused
emulator and return a constructor error. `Close` marks the wrapper closed,
closes `replyCloser`, joins the drainer, and only then calls `Emulator.Close`, so
the drainer never races the emulator's unsynchronized `closed` field. Observe
RED, then implement.

- [x] **Step 6: RED/GREEN concurrent shutdown and race verification**

`terminalModel` concurrent API: generate interleavings of Feed, Resize,
Snapshot, and Close under deadlines and the race detector; the guard is no race,
leak, panic, or blocked goroutine. Proxy teardown stops and joins signal
delivery before terminal close.

Run after every slice:

`go test -v ./cmd/internal/wrapcmd -run '^TestTerminalModel' -count=1`

Then run: `go test -race ./cmd/internal/wrapcmd -run '^TestTerminalModel' -count=1`

Expected: PASS without races, leaks, or blocked goroutines.

- [x] **Step 7: Keep the implementation within this shape**

```go
type terminalModel struct {
    mu sync.Mutex
    emulator *vt.Emulator
    observer terminalControlObserver
    altScreen bool
    replyCloser io.Closer
    drainDone chan struct{}
    closed bool
}

type terminalSnapshot struct {
    Width, Height int
    Cursor uv.Position
    CursorVisible bool
    AltScreen bool
    Cells []uv.Cell
}
```

Copy cells via `Clone` under the model lock and expose a bounds-safe snapshot
`CellAt`. `x/vt` owns screen semantics; the observer owns only explicit control
evidence that the emulator API cannot distinguish safely.

- [x] **Step 8: Commit**

```bash
git add cmd/internal/wrapcmd/terminal_model.go cmd/internal/wrapcmd/terminal_model_test.go
git commit -m "wrapcmd: #139: add shared terminal snapshot model" -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 2: Harness profiles and pure Return decisions

**Files:**
- Create: `cmd/internal/wrapcmd/harness_tty.go`
- Create: `cmd/internal/wrapcmd/harness_tty_test.go`
- Modify: `cmd/internal/wrapcmd/keymap_registry_test.go`

- [x] **Step 1: Write failing registry and routing tests**

Table-test exact Claude/Codex/Agy/Muse key bytes, overlay detector identity,
gate policy, Codex capture capability, unknown lookup, and remap-disabled
selection. Recognizer registration is deliberately deferred until Tasks 3/4,
so Chunk 1 commits compile without stubs. Test `decidePlainReturn` across overlay, active,
inactive/unknown, and legacy profiles, including telemetry and overlay clear.

- [x] **Step 2: Verify RED**

Run: `go test -v ./cmd/internal/wrapcmd -run 'TestHarnessTTYProfile|TestDecidePlainReturn' -count=1`

Expected: FAIL because the unified registry and decision do not exist.

- [x] **Step 3: Implement the profile registry**

```go
type composerGatePolicy uint8
const (
    composerGateUnknown composerGatePolicy = iota
    composerGateLegacy
    composerGatePositive
)
type composerRecognizer func(terminalSnapshot) bool
type harnessTTYProfile struct {
    keymap sendKeymap
    overlay overlayDetector
    composerGate composerGatePolicy
    recognize composerRecognizer
    captureSetsOverlay bool
}
```

Add `profileForHarness`; preserve Claude legacy behavior, exact byte mappings,
and unknown harness pass-through. `decidePlainReturn` switches exhaustively on
the gate policy: zero/unknown and out-of-range values fail closed to bare CR,
`adapt.Bypass`, and composer-unknown telemetry. Only explicit legacy and
positive policies may authorize profile key bytes.

- [x] **Step 4: Verify GREEN**

Run: `go test -v ./cmd/internal/wrapcmd -run 'TestHarnessTTYProfile|TestDecidePlainReturn|TestSendKeymap' -count=1`

Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add cmd/internal/wrapcmd/harness_tty.go cmd/internal/wrapcmd/harness_tty_test.go cmd/internal/wrapcmd/keymap_registry_test.go
git commit -m "wrapcmd: #139: centralize harness Return profiles" -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 2A: Capture Muse evidence before defining its recognizer

**Files:**
- Create: `cmd/internal/wrapcmd/harness_tty_live_test.go`
- Create: `cmd/internal/wrapcmd/testdata/tty/muse/<captured-version>/metadata.json`
- Create: `cmd/internal/wrapcmd/testdata/tty/muse/<captured-version>/composer.raw`

- [x] **Step 1: RED/GREEN the bounded PTY capture seam**

Implement the test-only `creack/pty` capture helper before any Muse snapshot
predicate exists. Use `pty.StartWithSize(..., &pty.Winsize{Rows: 38, Cols:
120})`; one read goroutine owns the PTY, retains at most 1 MiB, and reports
bounded output. Use a named 15-second startup timeout. On every exit path close
the PTY, interrupt the child, allow two seconds, kill if necessary, and bound
`cmd.Wait`. First write controlled-child tests for normal capture, missing
executable, timeout, and child cleanup; observe RED before implementing each
lifecycle behavior. Teardown must close/cancel before signaling, always attempt
the bounded wait/kill/reap sequence even after signal or kill errors, and join
the one reader goroutine. Combine cleanup failure with any primary capture
failure using `errors.Join`; injected failure-path tests must prove later
cleanup operations still execute.

- [x] **Step 2: Capture literal Muse startup bytes**

Run the installed `muse` executable in the helper. Obtain exact trimmed
`muse --version` output, retain the smallest literal startup prefix that paints
the composer, and write metadata using the Task 6 schema (`agent`, `version`,
RFC3339 `captured_at`, `command`, and filename-to-SHA-256 `files`). Never
synthesize bytes from a fake or from the old tracker. If Muse is unavailable,
unauthenticated, or blocked by workspace trust, stop and report the blocker.

- [x] **Step 3: Pin the qualified signature as evidence, not an assumption**

Inspect the captured raw SGR, prompt coordinates/shape, and final x/vt snapshot.
Add a fixture sanity test proving the prefix contains the observed qualified
composer signature and that the same nearby `›` with the observed qualifying
attributes removed is distinguishable. Task 3 must consume exactly this
evidence; if the capture does not expose a stable qualifier beyond the glyph,
stop and re-plan rather than inventing one.

- [x] **Step 4: Verify and commit**

```bash
go test -v ./cmd/internal/wrapcmd -run 'TestHarnessTTYCapture|TestMuseFixtureEvidence' -count=1
git add cmd/internal/wrapcmd/harness_tty_live_test.go cmd/internal/wrapcmd/testdata/tty/muse
git commit -m "wrapcmd: #139: capture Muse composer evidence" -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

## Chunk 2: Recognizers and proxy migration

### Task 3: Preserve Codex and Muse recognition on snapshots

**Files:**
- Create: `cmd/internal/wrapcmd/composer_recognizers.go`
- Create: `cmd/internal/wrapcmd/composer_recognizers_test.go`
- Modify: `cmd/internal/wrapcmd/harness_tty.go`
- Modify: `cmd/internal/wrapcmd/harness_tty_test.go`
- Modify: `cmd/internal/wrapcmd/codex_composer_test.go`
- Modify: `cmd/internal/wrapcmd/muse_composer_test.go`

- [x] **Step 1: Freeze the differential oracle**

`codexComposerActive` / `museComposerActive`: generate current-screen snapshots
around literal captured signatures and arbitrary mutation/locality transforms;
the guard is equality with frozen old-tracker decisions except the named Muse
stale-mutation and unqualified-glyph old-`true` / new-`false` safety corrections,
with every old-`false` / new-`true` rejected. Seed the local-weak-plus-distant
evidence rule from `lessons.md`. Run the old-tracker oracle GREEN before adding
the new API.

- [x] **Step 2: Write failing snapshot recognizer tests**

Drive the frozen oracle through `terminalModel` and the absent snapshot
recognizers, preserving the strategy and guard above. Derive Muse's qualified
prompt signature only from Task 2A's literal capture; do not invent styling or
geometry absent from observed bytes.

- [x] **Step 3: Verify RED**

Run: `go test -v ./cmd/internal/wrapcmd -run 'Test(Codex|Muse)Composer.*Snapshot' -count=1`

Expected: FAIL because snapshot recognizers do not exist.

- [x] **Step 4: Implement snapshot recognizers alongside existing parsers**

Implement and register pure grid predicates alongside the old tracker
implementations. Retain the trackers and their parsing helpers through Task 4;
Task 5 removes them only after the inherited Agy prototype and proxy no longer
depend on them.

- [x] **Step 5: Verify preservation and commit**

```bash
go test -v ./cmd/internal/wrapcmd -run 'Test(Codex|Muse)Composer|TestEmitPlainCR_(Codex|Muse)' -count=1
git add cmd/internal/wrapcmd/composer_recognizers.go cmd/internal/wrapcmd/composer_recognizers_test.go cmd/internal/wrapcmd/harness_tty.go cmd/internal/wrapcmd/harness_tty_test.go cmd/internal/wrapcmd/codex_composer_test.go cmd/internal/wrapcmd/muse_composer_test.go
git commit -m "wrapcmd: #139: derive Codex and Muse gates from terminal snapshots" -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

Expected: tests PASS with equality except for the named safety-correction rows,
including stale mutation evidence and the unqualified Muse glyph; each exception
is old `true` / new `false`, and there are no old `false` / new `true` changes.
The frozen table remains the differential oracle after the live tracker
implementations are deleted in Task 5; Task 5 updates
`composer_recognizers_test.go` only to remove live-tracker plumbing, not the
recorded expectations.

Do not delete the old tracker implementations in this task: the inherited Agy
prototype still compiles against their helpers, and proxy still consumes them.
Deletion happens only in Task 5 after the prototype and proxy dependencies are
gone.

### Task 4: Strict Agy coherent-box recognition

**Files:**
- Modify: `cmd/internal/wrapcmd/composer_recognizers.go`
- Modify: `cmd/internal/wrapcmd/composer_recognizers_test.go`
- Modify: `cmd/internal/wrapcmd/harness_tty.go`
- Modify: `cmd/internal/wrapcmd/harness_tty_test.go`
- Remove inherited untracked prototype after porting unique cases:
  `cmd/internal/wrapcmd/agy_composer.go`,
  `cmd/internal/wrapcmd/agy_composer_test.go`

- [x] **Step 1: Write failing geometry tests**

Generate snapshots varying border span/length, prompt column, cursor
position/visibility, height, and local-vs-distant evidence. Include every
weak/stale negative from the issue revision.

- [x] **Step 2: Verify RED**

Run: `go test -v ./cmd/internal/wrapcmd -run '^TestAgyComposerActive' -count=1`

Expected: FAIL because `agyComposerActive` is absent.

- [x] **Step 3: Implement the minimal predicate**

Pair only overlapping contiguous `─` runs of length >=5 and height <=25; require
the anchored prompt and visible cursor inside the same pair. Retain no evidence
across snapshots.

- [x] **Step 4: Verify GREEN and retire prototype**

Run: `go test -v ./cmd/internal/wrapcmd -run '^TestAgyComposerActive' -count=1`

Expected: PASS. Port unique inherited cases, then remove the untracked
partial-parser files so they cannot be staged.

- [x] **Step 5: Commit**

```bash
git add cmd/internal/wrapcmd/composer_recognizers.go cmd/internal/wrapcmd/composer_recognizers_test.go cmd/internal/wrapcmd/harness_tty.go cmd/internal/wrapcmd/harness_tty_test.go
git commit -m "wrapcmd: #139: recognize the coherent Agy composer box" -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 5: Migrate proxy routing to profiles

**Files:**
- Modify: `cmd/internal/wrapcmd/wrap.go`
- Modify: `cmd/internal/wrapcmd/terminal_model.go`
- Modify: `cmd/internal/wrapcmd/terminal_model_test.go`
- Modify: `cmd/internal/wrapcmd/composer_recognizers.go`
- Modify: `cmd/internal/wrapcmd/composer_recognizers_test.go`
- Modify: `cmd/internal/wrapcmd/codex_return_test.go`
- Modify: `cmd/internal/wrapcmd/muse_return_test.go`
- Create: `cmd/internal/wrapcmd/agy_return_test.go`
- Create: `cmd/internal/wrapcmd/harness_tty_integration_test.go`
- Modify: `cmd/internal/wrapcmd/overlay_test.go`
- Modify: `cmd/internal/wrapcmd/picker_overlay_test.go`
- Modify: `cmd/internal/wrapcmd/adapt_drift_test.go`
- Modify: `cmd/internal/wrapcmd/translate_test.go`
- Modify: `cmd/internal/wrapcmd/translate_stdin_test.go`
- Modify: `cmd/internal/wrapcmd/keymap_registry_test.go`
- Delete: `cmd/internal/wrapcmd/codex_composer.go`
- Delete: `cmd/internal/wrapcmd/codex_composer_test.go`
- Delete: `cmd/internal/wrapcmd/muse_composer.go`
- Delete: `cmd/internal/wrapcmd/muse_composer_test.go`

- [ ] **Step 1: Write failing integration tests**

A stateful `harnessSessionFake` sends lifecycle chunks through `handleChunk`.
Assert selection, feed/resize, overlay precedence/clear, composer multiline,
hidden/busy/unknown bare CR, Alt+Return, remap disabled, unknown pass-through,
Codex capture overlay, and terminal close.
Add deterministic production-path regressions proving overlay re-arm cannot be
consumed by an older Enter and `setWinsize` (not only `resizeTerminal`) provides
a latched resize transaction: invalid sizes mutate neither side; PTY set
failure stays inactive after later `?25h` plus positive composer bytes; no
concurrent snapshot authorizes the prepared intermediate state; and a later
successful resize re-enables only after fresh explicit visibility evidence.
Also overlap two resize transactions deterministically to prove the second
cannot prepare before the first commits or aborts, and inject a panicking
overlay detector through `handleChunk` before proving the next Return completes.

- [ ] **Step 2: Verify RED**

Run: `go test -v ./cmd/internal/wrapcmd -run 'TestHarnessTTYIntegration|TestEmitPlainCR_Agy' -count=1`

Expected: FAIL because proxy still has per-agent branches.

- [ ] **Step 3: Replace proxy branches**

Resolve the profile beside remap setup; create/feed/resize/close one terminal for
positive-gated profiles; use `decidePlainReturn`; make `armCapture` consult the
profile; stop/join signal handling before terminal close; remove parallel
registries and composer fields. Overlay detection, text-tail update, arming,
and Return consumption share `overlayMu`; Return uses `Swap(false)` while
holding it so a concurrent new overlay cannot be lost. Resize is a latched
transaction owned by `terminalModel`: prepare validates then masks authorization
before changing model geometry and returns an exclusive token; after PTY resize
succeeds, token commit clears old visibility and restores synchronization.
Every failure aborts the token, releasing transaction ownership while leaving
synchronization false across all later Feed/show-cursor traffic until a later
successful transaction. Overlay detection uses a defer-unlocked helper so a
recovered detector panic cannot strand `overlayMu`.
Only now delete Codex/Muse trackers and their orphaned helpers, after the Agy
prototype has been retired.

- [ ] **Step 4: Verify GREEN and shadow sweep**

```bash
go test -v ./cmd/internal/wrapcmd -run 'TestHarnessTTYIntegration|TestEmitPlainCR_(Codex|Muse|Agy)|TestHarnessTTYProfile' -count=1
rg -n 'sendKeymapByAgent|overlayDetectorByAgent|codexComposer|museComposer|agentBasename == "(codex|muse|agy)"' cmd/internal/wrapcmd
```

Expected: tests PASS and no Return/composer shadow source remains. Explain
unrelated agent-specific behavior rather than deleting it mechanically. Also
search constructors and consumers of `checkOverlayOpen`, `armCapture`,
`hasReturnRemap`, and translation setup so no test or partial proxy reconstructs
a second profile.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/wrapcmd/wrap.go cmd/internal/wrapcmd/terminal_model.go cmd/internal/wrapcmd/terminal_model_test.go cmd/internal/wrapcmd/composer_recognizers.go cmd/internal/wrapcmd/composer_recognizers_test.go cmd/internal/wrapcmd/codex_return_test.go cmd/internal/wrapcmd/muse_return_test.go cmd/internal/wrapcmd/agy_return_test.go cmd/internal/wrapcmd/harness_tty_integration_test.go cmd/internal/wrapcmd/overlay_test.go cmd/internal/wrapcmd/picker_overlay_test.go cmd/internal/wrapcmd/adapt_drift_test.go cmd/internal/wrapcmd/translate_test.go cmd/internal/wrapcmd/translate_stdin_test.go cmd/internal/wrapcmd/keymap_registry_test.go cmd/internal/wrapcmd/codex_composer.go cmd/internal/wrapcmd/codex_composer_test.go cmd/internal/wrapcmd/muse_composer.go cmd/internal/wrapcmd/muse_composer_test.go
git commit -m "wrapcmd: #139: route Return through harness TTY profiles" -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

## Chunk 3: Conformance, documentation, and close evidence

### Task 6: Checked-in fixtures and live conformance

**Files:**
- Create: `cmd/internal/wrapcmd/testdata/tty/<agent>/<version>/metadata.json`
- Create: `cmd/internal/wrapcmd/testdata/tty/<agent>/<version>/composer.raw`
- Create when captured: matching `overlay.raw`
- Create: `cmd/internal/wrapcmd/harness_tty_fixture_test.go`
- Create: `cmd/internal/wrapcmd/harness_tty_live_test.go`

- [ ] **Step 1: Write failing fixture inventory/schema tests**

Define literal metadata:

```go
type ttyFixtureMetadata struct {
    Agent string `json:"agent"`
    Version string `json:"version"` // trimmed exact --version output
    CapturedAt string `json:"captured_at"`
    Command []string `json:"command"`
    Files map[string]string `json:"files"` // filename -> lowercase SHA-256
}
```

Require one `composer.raw` fixture for every positive-gated profile (Codex,
Muse, Agy), exactly one metadata digest per raw file and no dangling digest,
valid RFC3339 capture time, non-empty argv, and a directory version segment
derived by one tested filename-safe normalizer. The inventory must fail rather
than vacuously pass when a required profile/directory is absent.

- [ ] **Step 2: Verify RED**

Run: `go test -v ./cmd/internal/wrapcmd -run '^TestHarnessTTYFixtureConformance' -count=1`

Expected: FAIL because fixtures do not exist.

- [ ] **Step 3: Extend the capture helper into live conformance**

Extend Task 2A's bounded test-only PTY helper so each chunk feeds the production
profile and supports conformance classification. Add controlled-child tests for
normal recognition, unauthenticated/login UI, workspace-trust UI, and recognizer
drift. Preserve its 120x38 size, 1 MiB cap, 15-second timeout, deterministic
interrupt/kill/reap lifecycle, and stripped bounded failure output that never
dumps environment or input bytes.

- [ ] **Step 4: Capture literal real-harness bytes**

Use the live capture path, never fake-generated bytes. Keep the smallest raw
prefix painting the composer; record version/command/digest. Retain and
revalidate Task 2A's Muse fixture, and capture the missing Codex and Agy
fixtures. If any harness is unavailable or unauthenticated, stop and report the
blocker rather than inventing a fixture. Use argv `agy
--dangerously-skip-permissions`, `codex --no-alt-screen`, and `muse` unless the
installed CLI requires a documented correction; obtain each version with the
same executable's `--version` and preserve the actual argv in metadata.
`PAIR_LIVE_CAPTURE_OUT=<destination>` makes the live test write the captured
prefix atomically.

- [ ] **Step 5: Replay through the production seam**

Make `TestHarnessTTYFixtureConformance` instantiate the production
proxy/profile and establish an unsplit baseline for each fixture. For every
split from zero through `len(raw)`, feed the fixture through `proxy.handleChunk`
and assert equality with that baseline for (a) the recognizer result and (b) the
complete `returnDecision`/observable decision result, including the expected
positive composer classification, final Return bytes, and overlay clearing.
With no `PAIR_LIVE_HARNESS`, the live test skips; otherwise it uses the helper,
compares installed version to metadata, and prints the exact recapture
destination on version or recognition drift.

- [ ] **Step 6: Verify every supported harness exactly**

```bash
go test -v ./cmd/internal/wrapcmd -run '^TestHarnessTTYFixtureConformance' -count=1
PAIR_LIVE_HARNESS=agy go test -v ./cmd/internal/wrapcmd -run '^TestHarnessTTYLiveConformance' -count=1
PAIR_LIVE_HARNESS=codex go test -v ./cmd/internal/wrapcmd -run '^TestHarnessTTYLiveConformance' -count=1
PAIR_LIVE_HARNESS=muse go test -v ./cmd/internal/wrapcmd -run '^TestHarnessTTYLiveConformance' -count=1
```

Expected: fixture replay and all three installed live harness checks PASS.
Authentication/trust failures are legitimate blockers to close, not silent
passes.

- [ ] **Step 7: Commit**

```bash
git add cmd/internal/wrapcmd/testdata/tty cmd/internal/wrapcmd/harness_tty_fixture_test.go cmd/internal/wrapcmd/harness_tty_live_test.go
git commit -m "wrapcmd: #139: pin harness TTY conformance fixtures" -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 7: Atlas and full verification

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `atlas/how-to-bring-up-a-new-harness-cli.md`
- Modify: `README.md`
- Modify: `doctor/README.md`
- Modify: `workshop/issues/000139-agy-return-rewrite-only-in-composer.md`

- [ ] **Step 1: Update atlas and user-facing guidance**

Map profile ownership, terminal lifecycle, Return precedence, recognizer
boundary, fixture layout, live cadence, and new-harness opt-in. Update the
README keybinding guidance with Claude, Codex, Muse, and Agy plain/Alt+Return
behavior and the positive-gate fail-safe. Replace doctor guidance for the
deleted keymap/overlay registries with `harnessTTYProfiles` and
`profileForHarness`.

- [ ] **Step 2: Run focused, race, and repository verification**

```bash
go test ./cmd/internal/wrapcmd -count=1
go test -race ./cmd/internal/wrapcmd -count=1
go test ./... -count=1
make test
git diff --check
```

Expected: every command PASS.

- [ ] **Step 3: Update durable state and commit**

Tick rows only after evidence exists. Log the shadow sweep, fixture/live results,
and all four ARCH decisions. Leave the canonical issue Plan pointer unchecked
until every implementation/verification row is complete, then tick it before
calling `sdlc close`. Confirm with `rg -n '^- \[ \]'` over the authoritative
issue and plan sections.

```bash
git add atlas/architecture.md atlas/how-to-bring-up-a-new-harness-cli.md README.md doctor/README.md workshop/issues/000139-agy-return-rewrite-only-in-composer.md workshop/plans/000139-agy-return-rewrite-only-in-composer-plan.md
git commit -m "atlas: #139: map unified harness Return routing" -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

- [ ] **Step 4: Close through SDLC and persist its anchor**

Run `sdlc close --issue 139 --verified '<focused, race, full-suite, fixture,
and live evidence>'`. Follow its next-action output; it measures actual time,
runs the mandatory fresh boundary review, and moves the issue to
`codecomplete`. Do not run a redundant ad-hoc review. If it reports
FIX-THEN-SHIP, apply every Critical/Important fix before committing and follow
the command's post-verdict protocol. After successful close, tick this durable
plan row and commit the `codecomplete` issue mutation, plan, generated review
sidecar, and any `workshop/lessons.md` rule together so the reviewed anchor is
durable.

---

### Historical Task 2: Wire Agy Positive Composer Gate into `wrap.go` & Add Stateful Protocol Replay Tests

**Files:**
- Modify: `cmd/internal/wrapcmd/wrap.go`
- Create: `cmd/internal/wrapcmd/agy_return_test.go`

- Historical step: Write Return routing and protocol replay tests.

Create `cmd/internal/wrapcmd/agy_return_test.go`:
1. `proxy.emitPlainCR`: Matrix testing across active composer (`\n`), inactive composer (`\r`), unknown state (`\r`), and overlay precedence (`pickerActive` returns `\r` and clears flag).
2. `proxy.handleChunk`: Stateful lifecycle replay using `agySessionFake` through startup, multi-line typing, thinking/generation (`\x1b[?25l`), and permission picker overlay.

- Historical step: Run tests to verify they fail.

Run: `go test -v ./cmd/internal/wrapcmd -run TestEmitPlainCR_Agy`
Expected: FAIL (agy currently rewrites unconditionally).

- Historical step: Wire Agy composer gating in `wrap.go`.

In `cmd/internal/wrapcmd/wrap.go`:
- Add `agyComposer *agyComposerTracker` field to `proxy`.
- Add helper `p.agyComposerActive() bool` and `p.ensureAgyComposer() *agyComposerTracker`.
- In `setWinsize` (`wrap.go:2000`): Call `p.ensureAgyComposer().resize(int(ws.Rows), int(ws.Cols))` when `agentBasename == "agy"`.
- In `handleChunk` (`wrap.go:2677`): Call `p.ensureAgyComposer().feed(data)` when `agentBasename == "agy"`.
- In `emitPlainCR` (`wrap.go:1737`):
  ```go
  if p.agentBasename == "agy" && !p.agyComposerActive() {
      p.adapt.Log(1, "return-remap", adapt.Bypass, "plain Enter → bare CR (agy composer inactive)")
      return append(out, '\r')
  }
  ```

- Historical step: Run Agy return tests.

Run: `go test -v ./cmd/internal/wrapcmd -run TestEmitPlainCR_Agy`
Expected: PASS.

- Historical step: Commit.

```bash
git add cmd/internal/wrapcmd/wrap.go cmd/internal/wrapcmd/agy_return_test.go
git commit -m "wrap: #139: gate agy plain return behind positive composer detection

Co-Authored-By: Antigravity <antigravity@google.com>"
```

---

### Historical Task 3: Documentation, Atlas Update & Full Test Suite Verification

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `atlas/how-to-bring-up-a-new-harness-cli.md`

- Historical step: Update documentation.

Update `atlas/architecture.md` and `atlas/how-to-bring-up-a-new-harness-cli.md` to document that `agy` uses positive composer detection via `agyComposerTracker` (matching Codex and Muse), where plain Return rewrites to LF only when the composer box is positively detected with a visible cursor.

- Historical step: Run full unit and integration test suite.

Run: `go test ./...`
Expected: PASS across all packages (`cmd/...`, `pkg/...`).

- Historical step: Run make test suite.

Run: `make test`
Expected: PASS across all test suites.

- Historical step: Commit.

```bash
git add atlas/architecture.md atlas/how-to-bring-up-a-new-harness-cli.md
git commit -m "atlas: #139: document positive composer detection for agy

Co-Authored-By: Antigravity <antigravity@google.com>"
```

## Revisions

### 2026-08-17T14:17:00-07:00 — Replace Task 0 after #140 boundary REWORK

The exact review of `ab736d1^..ab736d1` found that #140 cannot cross its
boundary as landed: its Muse tracker retains stale screen evidence, accepts an
unqualified `›`, duplicates terminal tracking, lacks stateful/live protocol
evidence, and omits README coverage. Those are not separable dependency fixes;
they are the unified substrate and conformance work in Tasks 1, 3, 5, 6, and 7.
Implementing them first under #140 would create the same architecture twice and
immediately delete one copy (ARCH-DRY, ARCH-PURPOSE).

This revision supersedes only Chunk 1 Task 0 above:

- [x] Record the `REWORK` verdict and `ab736d1^..ab736d1` window in #140,
      mark #140 `wontfix` as superseded by #139 through its own clean worktree,
      and commit that tracker-only transaction. Do not claim or close the
      known-broken implementation as codecomplete.
- [x] Remove #139's dependency on #140 and preserve all #140 acceptance
      criteria in #139. Treat the old Muse tracker as characterization data;
      the new snapshot recognizer must correct the two reviewed false positives
      and all named stale-state cases without introducing any `false -> true`.
- [x] Add `README.md` to Task 7 and document the user-facing Return behavior for
      Claude, Codex, Muse, and Agy alongside the atlas updates.
- [x] After this revision's fresh review passes, run
      `sdlc change-code --issue 139`; let the gate derive the expanded estimate.

All remaining tasks and their TDD order are unchanged.

#### Evidence-order correction

Task 2A now precedes Task 3 and captures the literal Muse bytes and metadata
through the same bounded PTY seam Task 6 later extends. Task 3 may define the
qualified Muse signature only from that checked-in evidence. Task 6 retains the
Muse fixture, adds Codex/Agy inventory, and performs live drift validation; it
no longer retroactively supplies evidence for an already-committed predicate.

#### Plan-quality round 5 corrections

PQ-3 is addressed by replacing the terminal and recognizer case inventories
with one adversarial-input strategy plus mechanical guard for each named risky
function. The RED/GREEN checkpoints remain only as execution gates, not prose
copies of test tables. PQ-6 is addressed by changing `newTerminalModel` to
return an error and checking the dynamic `io.Closer` capability of the
`io.Writer` returned by pinned `x/vt.Emulator.InputPipe()` before starting the
reply drainer. The retained `io.Closer` is now an explicit model field and the
shutdown/property tests exercise capability failure, reply production, and
concurrent close (ARCH-PURE).

#### Task 1 parser and malformed-stream correction

Quality review replaced the Frame-only observer with a bounded stateful x/ansi
parser because C1 CSI is a control only in ground state, not inside UTF-8 or
control-string payloads. This removes a parallel partial DFA. Fuzzing plus the
valid `👩‍💻` split then proved that pinned x/vt flushes extended graphemes at
`Write` boundaries. Task 1 therefore carries safety/bounds/fail-closed observer
invariants for arbitrary streams, not whole-grid chunk equality. Task 6 owns the
purpose-level invariant: recognizer and Return decisions must match at every
split of each literal harness fixture. Pair does not add a second grapheme
renderer around x/vt (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

Fresh review made both consequences executable: Task 1 removes the disproved
valid-UTF-8 snapshot-equality fuzz oracle and adds a deterministic ZWJ safety
regression; Task 6 compares every split with an unsplit baseline for both the
recognizer and the complete Return decision, rather than allowing final bytes
alone to mask classification drift.

#### Task 1 allocation-bound correction

Quality review found that the positive-dimension check still permitted
unbounded allocations in x/vt and `Snapshot`. Task 1 now defines one pure,
overflow-safe validator shared by construction and resize: each axis is at most
4096 and total area at most 262,144 cells. RED tests cover oversized axes,
oversized area, and integer-overflow-shaped inputs; rejected resize must leave
the full prior snapshot unchanged. No allocation or x/vt mutation may occur
before validation (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

#### Task 2 fail-safe enum correction

Quality review found that the planned `composerGateLegacy = iota` made the zero
value fail open. Task 2 reserves zero for `composerGateUnknown` and implements
the pure decision as an exhaustive policy switch. RED tests cover an all-zero
profile and an out-of-range policy; both must return bare CR, `adapt.Bypass`,
composer-unknown telemetry, and no overlay clear. This prevents a missed
profile initialization from swallowing or rewriting Return when Task 5 moves
the profile onto `proxy` (ARCH-PURE, ARCH-PURPOSE).

#### Task 2A coordinated-teardown correction

Quality review found that the first capture helper suppressed cleanup errors
behind primary errors, returned early on unexpected kill failure, and never
joined its PTY reader. Task 2A now treats teardown as one bounded coordinated
operation: cancel and close the PTY, interrupt, wait, kill if needed, always
attempt the final reap wait, and join the reader. It returns `errors.Join` of
the capture and teardown failures. A narrow injected process/PTY operation seam
tests signal and kill failures while asserting later wait/reap/join steps still
occur; controlled-child tests continue to exercise the real seam
(ARCH-PURE, ARCH-MOCK, ARCH-PURPOSE).

#### Task 3 registry-file correction

Task 3 already requires registering the Codex and Muse snapshot recognizers,
but its original file and commit lists omitted `harness_tty.go`, the profile
registry's source of truth, and `harness_tty_test.go`, whose Task 2
characterization intentionally pinned nil recognizers. Add both files to the
task so the new pure predicates are actual tested profile consumers rather than
unused parallel APIs
(ARCH-DRY, ARCH-PURPOSE).

#### Task 4 Agy registry-file correction

Like Task 3, Task 4 must register and characterize the Agy predicate in the
profile source of truth. Add `harness_tty.go` and `harness_tty_test.go` to its
file/commit set so Task 5 consumes a complete positive-gated profile rather
than performing an implicit late registration (ARCH-DRY, ARCH-PURPOSE).

#### Task 5 tracker-retirement file correction

Deleting `codex_composer.go` also removes `codexComposerMinRows`, still consumed
by the snapshot predicate, and deleting both tracker implementations leaves
their implementation-specific test files uncompilable. Add
`composer_recognizers.go`/`composer_recognizers_test.go` to Task 5: move the
constant into the pure recognizer, remove only old-tracker plumbing while
retaining the frozen 15-row expectations, and delete both old tracker test
files with their implementations. This makes tracker retirement executable
without losing the durable differential oracle (ARCH-DRY, ARCH-PURPOSE).
The shadow sweep also includes `translate_stdin_test.go`, whose shared Claude
proxy helper hand-built the retired keymap field; migrate it to profile lookup
so test construction does not preserve a second profile source.
