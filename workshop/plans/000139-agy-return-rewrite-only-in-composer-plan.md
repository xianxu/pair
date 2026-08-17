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
  painted-background locality rule; Muse preserves its `›` locality rule; Agy
  requires two overlapping contiguous `─` runs (minimum five), an anchored
  prompt in columns 0-5 between them, maximum height 25, and a visible cursor
  inside the same box.

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
  `io.Copy(io.Discard, emulator)`; shutdown closes the retained input-pipe
  writer, joins the drainer, then closes the emulator. Visibility starts false
  and only an explicit chunk-framed `?25h` authorizes it; hide, reset, and
  alt-screen transitions clear it.
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

- [ ] **Step 1: Resolve #140 through its own SDLC run**

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

- [ ] **Step 2: Re-enter #139 implementation through the gate**

Run: `sdlc change-code --issue 139`

Expected: revised plan-quality runs before estimation. After it passes, derive
and install the expanded estimate requested by the command; never reuse the
superseded 0.57h estimate.

### Task 1: Atomic shared terminal snapshots

**Files:**
- Create: `cmd/internal/wrapcmd/terminal_model.go`
- Create: `cmd/internal/wrapcmd/terminal_model_test.go`

- [ ] **Step 1: RED/GREEN constructor and empty snapshot**

Define the desired API:

```go
model := newTerminalModel(80, 24)
t.Cleanup(func() { _ = model.Close() })
_ = model.Feed(raw)
snapshot := model.Snapshot()
```

Run the focused test and observe the undefined constructor failure. Implement
only construction, a 80x24 empty snapshot, and snapshot bounds checks; rerun to
GREEN.

- [ ] **Step 2: RED/GREEN feed and immutable cell copying**

Add tests for printable cells, SGR styles, CUP/relative moves, ECH/EL/ED,
scrolling, wrapping, and every split point of a representative stream. Observe
the expected assertion failures, then minimally wire `Emulator.Write` and deep
cell clones to GREEN.

- [ ] **Step 3: RED/GREEN explicit control observation**

Add a chunk-safe `terminalControlObserver` using `cmd/internal/ansi.Frame` with
bounded pending bytes. It alone owns observed cursor visibility: explicit
`?25h` sets true, `?25l`, RIS (`ESC c`), and alt-screen enter/leave
(`?1047`/`?1049`) set false. Emulator callbacks never authorize visibility.
Test every split point, malformed/incomplete recovery, repeated `?25h`, RIS,
and both alt-screen directions; each reset/switch stays false until a later
explicit `?25h`. Observe RED before implementing each transition.

- [ ] **Step 4: RED/GREEN resize and active-screen snapshot**

Add resize and alt-screen cell replacement tests, observe RED, then implement
`Resize(cols, rows)` and snapshot `AltScreen` tracking without weakening the
explicit-visibility observer.

- [ ] **Step 5: RED/GREEN reply draining and deterministic close**

Add DSR/device-attribute tests that prove `Feed` cannot block. Retain the
closable writer from `Emulator.InputPipe()`. `Close` marks the wrapper closed,
closes that writer, joins the drainer, and only then calls `Emulator.Close`, so
the drainer never races the emulator's unsynchronized `closed` field. Repeated
Close is a no-op; after close, Feed/Resize return `io.ErrClosedPipe` and Snapshot
returns the final immutable state. Observe each failure before implementing.

- [ ] **Step 6: RED/GREEN concurrent shutdown and race verification**

Race Close against Feed, Resize, and Snapshot. Ensure teardown cannot race the
SIGWINCH goroutine: proxy shutdown must stop signal delivery before closing its
terminal, while wrapper post-close behavior remains safe as a backstop.

Run after every slice:

`go test -v ./cmd/internal/wrapcmd -run '^TestTerminalModel' -count=1`

Then run: `go test -race ./cmd/internal/wrapcmd -run '^TestTerminalModel' -count=1`

Expected: PASS without races, leaks, or blocked goroutines.

- [ ] **Step 7: Keep the implementation within this shape**

```go
type terminalModel struct {
    mu sync.Mutex
    emulator *vt.Emulator
    observer terminalControlObserver
    altScreen bool
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

- [ ] **Step 8: Commit**

```bash
git add cmd/internal/wrapcmd/terminal_model.go cmd/internal/wrapcmd/terminal_model_test.go
git commit -m "wrapcmd: #139: add shared terminal snapshot model" -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 2: Harness profiles and pure Return decisions

**Files:**
- Create: `cmd/internal/wrapcmd/harness_tty.go`
- Create: `cmd/internal/wrapcmd/harness_tty_test.go`
- Modify: `cmd/internal/wrapcmd/keymap_registry_test.go`

- [ ] **Step 1: Write failing registry and routing tests**

Table-test exact Claude/Codex/Agy/Muse key bytes, overlay detector identity,
gate policy, Codex capture capability, unknown lookup, and remap-disabled
selection. Recognizer registration is deliberately deferred until Tasks 3/4,
so Chunk 1 commits compile without stubs. Test `decidePlainReturn` across overlay, active,
inactive/unknown, and legacy profiles, including telemetry and overlay clear.

- [ ] **Step 2: Verify RED**

Run: `go test -v ./cmd/internal/wrapcmd -run 'TestHarnessTTYProfile|TestDecidePlainReturn' -count=1`

Expected: FAIL because the unified registry and decision do not exist.

- [ ] **Step 3: Implement the profile registry**

```go
type composerGatePolicy uint8
const (
    composerGateLegacy composerGatePolicy = iota
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
and unknown harness pass-through.

- [ ] **Step 4: Verify GREEN**

Run: `go test -v ./cmd/internal/wrapcmd -run 'TestHarnessTTYProfile|TestDecidePlainReturn|TestSendKeymap' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/wrapcmd/harness_tty.go cmd/internal/wrapcmd/harness_tty_test.go cmd/internal/wrapcmd/keymap_registry_test.go
git commit -m "wrapcmd: #139: centralize harness Return profiles" -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

## Chunk 2: Recognizers and proxy migration

### Task 3: Preserve Codex and Muse recognition on snapshots

**Files:**
- Create: `cmd/internal/wrapcmd/composer_recognizers.go`
- Create: `cmd/internal/wrapcmd/composer_recognizers_test.go`
- Modify: `cmd/internal/wrapcmd/codex_composer_test.go`
- Modify: `cmd/internal/wrapcmd/muse_composer_test.go`

- [ ] **Step 1: Add characterization tests**

Drive current trackers with every existing tracker test plus overwrite, EL,
scroll, resize/reflow, alt-screen, and RIS transitions. Run this
characterization suite GREEN before introducing the new API. Record a decision
matrix: active composer, hidden cursor, overlay, and unknown states must remain
identical; stale evidence may only change `true -> false`, must be individually
allowlisted, and may never change `false -> true`. Include one local weak signal
plus far-away evidence from `lessons.md`.

- [ ] **Step 2: Write failing snapshot recognizer tests**

Feed identical streams through `terminalModel`; assert
`codexComposerActive(snapshot)` and `museComposerActive(snapshot)` match the
characterized decision table except for individually named stale-evidence
allowlist rows. Freeze the old tracker results as table data before rewriting
tests: every allowlisted exception must be old `true` / new `false`, and the
oracle must reject every `false` / new `true` transition.

- [ ] **Step 3: Verify RED**

Run: `go test -v ./cmd/internal/wrapcmd -run 'Test(Codex|Muse)Composer.*Snapshot' -count=1`

Expected: FAIL because snapshot recognizers do not exist.

- [ ] **Step 4: Implement snapshot recognizers alongside existing parsers**

Implement and register pure grid predicates alongside the old tracker
implementations. Retain the trackers and their parsing helpers through Task 4;
Task 5 removes them only after the inherited Agy prototype and proxy no longer
depend on them.

- [ ] **Step 5: Verify preservation and commit**

```bash
go test -v ./cmd/internal/wrapcmd -run 'Test(Codex|Muse)Composer|TestEmitPlainCR_(Codex|Muse)' -count=1
git add cmd/internal/wrapcmd/composer_recognizers.go cmd/internal/wrapcmd/composer_recognizers_test.go cmd/internal/wrapcmd/codex_composer_test.go cmd/internal/wrapcmd/muse_composer_test.go
git commit -m "wrapcmd: #139: derive Codex and Muse gates from terminal snapshots" -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

Expected: tests PASS with equality except for the named stale-evidence rows;
each exception is old `true` / new `false`, and there are no `false` / new
`true` changes. The frozen table remains the differential oracle after the live
tracker implementations are deleted in Task 5; Task 5 updates
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
- Remove inherited untracked prototype after porting unique cases:
  `cmd/internal/wrapcmd/agy_composer.go`,
  `cmd/internal/wrapcmd/agy_composer_test.go`

- [ ] **Step 1: Write failing geometry tests**

Generate snapshots varying border span/length, prompt column, cursor
position/visibility, height, and local-vs-distant evidence. Include every
weak/stale negative from the issue revision.

- [ ] **Step 2: Verify RED**

Run: `go test -v ./cmd/internal/wrapcmd -run '^TestAgyComposerActive' -count=1`

Expected: FAIL because `agyComposerActive` is absent.

- [ ] **Step 3: Implement the minimal predicate**

Pair only overlapping contiguous `─` runs of length >=5 and height <=25; require
the anchored prompt and visible cursor inside the same pair. Retain no evidence
across snapshots.

- [ ] **Step 4: Verify GREEN and retire prototype**

Run: `go test -v ./cmd/internal/wrapcmd -run '^TestAgyComposerActive' -count=1`

Expected: PASS. Port unique inherited cases, then remove the untracked
partial-parser files so they cannot be staged.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/wrapcmd/composer_recognizers.go cmd/internal/wrapcmd/composer_recognizers_test.go
git commit -m "wrapcmd: #139: recognize the coherent Agy composer box" -m "Co-Authored-By: OpenAI Codex <noreply@openai.com>"
```

### Task 5: Migrate proxy routing to profiles

**Files:**
- Modify: `cmd/internal/wrapcmd/wrap.go`
- Modify: `cmd/internal/wrapcmd/codex_return_test.go`
- Modify: `cmd/internal/wrapcmd/muse_return_test.go`
- Create: `cmd/internal/wrapcmd/agy_return_test.go`
- Create: `cmd/internal/wrapcmd/harness_tty_integration_test.go`
- Modify: `cmd/internal/wrapcmd/overlay_test.go`
- Modify: `cmd/internal/wrapcmd/picker_overlay_test.go`
- Modify: `cmd/internal/wrapcmd/adapt_drift_test.go`
- Modify: `cmd/internal/wrapcmd/translate_test.go`
- Modify: `cmd/internal/wrapcmd/keymap_registry_test.go`
- Delete: `cmd/internal/wrapcmd/codex_composer.go`
- Delete: `cmd/internal/wrapcmd/muse_composer.go`

- [ ] **Step 1: Write failing integration tests**

A stateful `harnessSessionFake` sends lifecycle chunks through `handleChunk`.
Assert selection, feed/resize, overlay precedence/clear, composer multiline,
hidden/busy/unknown bare CR, Alt+Return, remap disabled, unknown pass-through,
Codex capture overlay, and terminal close.

- [ ] **Step 2: Verify RED**

Run: `go test -v ./cmd/internal/wrapcmd -run 'TestHarnessTTYIntegration|TestEmitPlainCR_Agy' -count=1`

Expected: FAIL because proxy still has per-agent branches.

- [ ] **Step 3: Replace proxy branches**

Resolve the profile beside remap setup; create/feed/resize/close one terminal for
positive-gated profiles; use `decidePlainReturn`; make `armCapture` consult the
profile; stop/join signal handling before terminal close; remove parallel
registries and composer fields. Only now delete Codex/Muse trackers and their
orphaned helpers, after the Agy prototype has been retired.

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
git add cmd/internal/wrapcmd/wrap.go cmd/internal/wrapcmd/codex_return_test.go cmd/internal/wrapcmd/muse_return_test.go cmd/internal/wrapcmd/agy_return_test.go cmd/internal/wrapcmd/harness_tty_integration_test.go cmd/internal/wrapcmd/overlay_test.go cmd/internal/wrapcmd/picker_overlay_test.go cmd/internal/wrapcmd/adapt_drift_test.go cmd/internal/wrapcmd/translate_test.go cmd/internal/wrapcmd/keymap_registry_test.go cmd/internal/wrapcmd/codex_composer.go cmd/internal/wrapcmd/muse_composer.go
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

- [ ] **Step 3: RED/GREEN a capture-capable live PTY helper**

Before capturing fixtures, implement a test-only helper using
`pty.StartWithSize(..., &pty.Winsize{Rows: 38, Cols: 120})`. A single read
goroutine owns the PTY and appends at most 1 MiB while feeding the profile after
each chunk. Use a named 15-second startup timeout. On every exit path: close the
PTY, send interrupt, allow a two-second grace period, kill if necessary, and
bound `cmd.Wait` so the child is always reaped. Unit-test normal recognition,
missing executable, timeout, unauthenticated/login UI, workspace-trust UI, and
recognizer drift using controlled child commands. Failure output is stripped,
bounded, and never dumps environment or input bytes.

- [ ] **Step 4: Capture literal real-harness bytes**

Use the live capture path, never fake-generated bytes. Keep the smallest raw
prefix painting the composer; record version/command/digest. If unavailable or
unauthenticated, stop and report the blocker rather than inventing a fixture.
Use argv `agy --dangerously-skip-permissions`, `codex --no-alt-screen`, and
`muse` unless the installed CLI requires a documented correction; obtain each
version with the same executable's `--version` and preserve the actual argv in
metadata. `PAIR_LIVE_CAPTURE_OUT=<destination>` makes the live test write the
captured prefix atomically.

- [ ] **Step 5: Replay through the production seam**

Make `TestHarnessTTYFixtureConformance` instantiate the production
proxy/profile, feed each fixture through `proxy.handleChunk` at every split
point, and assert final Return bytes plus overlay clearing. With no
`PAIR_LIVE_HARNESS`, the live test skips; otherwise it uses the helper, compares
installed version to metadata, and prints the exact recapture destination on
version or recognition drift.

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
- Modify: `workshop/issues/000139-agy-return-rewrite-only-in-composer.md`

- [ ] **Step 1: Update atlas**

Map profile ownership, terminal lifecycle, Return precedence, recognizer
boundary, fixture layout, live cadence, and new-harness opt-in.

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
git add atlas/architecture.md atlas/how-to-bring-up-a-new-harness-cli.md workshop/issues/000139-agy-return-rewrite-only-in-composer.md workshop/plans/000139-agy-return-rewrite-only-in-composer-plan.md
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
