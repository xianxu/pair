# Frame-Title Terminal Tab Rename Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the right terminal's content-area rename prompt with a
native-feeling, Unicode-aware editor rendered in the Zellij pane frame.

**Architecture:** A pure rune/cursor editor and a deterministic streaming byte
decoder form the core. `pumpStdin` becomes the single event loop that owns the
50ms Escape-prefix timer; `terminalMux` remains the thin IO shell that renders
preview/ordinary pane titles and commits tab names. Rename-mode input is always
consumed before the child PTY boundary.

**Tech Stack:** Go, PTYs, terminal escape sequences, Zellij pane-title actions.

---

## Revisions

### 2026-07-24 — Reconcile first plan review

Enumerate every accepted control encoding and split boundary, define malformed
and EOF behavior, add an injected reset/stop/drain timer plus one lifetime
reader-result channel, and pin same-read Alt+r transition semantics. Add the
missing task-to-estimate mapping. The implementation scope is unchanged.

### 2026-07-24 — Reconcile estimate vocabulary

The implementation gate rejected the non-canonical `ux-iteration` estimate
label. Map that unchanged live terminal/title risk to a second
`api-integration` row from the closed vocabulary; no hours or implementation
scope change.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `RenameEditor` | `cmd/internal/termcmd/rename.go` | new |
| `RenameEvent` / `RenameOutcome` | `cmd/internal/termcmd/rename.go` | new |
| `RenameDecoderState` | `cmd/internal/termcmd/rename_input.go` | new |
| `DecodeRenameInput` | `cmd/internal/termcmd/rename_input.go` | new |

- **`RenameEditor`** — immutable-by-contract rune text, cursor, and original
  name transformed by editor events.
  - **Relationships:** 1:1 with the active terminal tab while rename mode is
    active; 1:N from editor state to input events.
  - **DRY rationale:** One transition function owns insertion, movement,
    deletion, commit, and cancel rather than scattering string surgery through
    the PTY loop (`ARCH-DRY`, `ARCH-PURE`).
  - **Future extensions:** Word movement or selection adds events without
    changing terminal IO.

- **`RenameEvent` / `RenameOutcome`** — semantic input and transition result:
  insert rune, left/right/home/end, backspace/delete, commit/cancel, or consume.
  - **Relationships:** N:1 decoded events to one editor; each transition yields
    zero or one terminal outcome.
  - **DRY rationale:** Decoder and editor communicate in one vocabulary rather
    than sharing escape-byte knowledge.
  - **Future extensions:** Additional keys widen the event enum.

- **`RenameDecoderState`** — buffered escape/UTF-8 prefix plus bracketed-paste
  state.
  - **Relationships:** 1:1 with a live rename mode; N:1 stdin reads to one
    decoder state.
  - **DRY rationale:** One state owns every incomplete byte boundary.
  - **Future extensions:** Kitty-keyboard encodings can be added as recognized
    sequences.

- **`DecodeRenameInput`** — pure `(state, bytes, flushEscape) → state, events,
  exit` transition. A commit/cancel consumes the remainder of its input batch.
  - **Relationships:** feeds `RenameEditor`; never touches a reader, timer, PTY,
    or Zellij.
  - **DRY rationale:** Every split-boundary test exercises the same production
    decoder (`ARCH-PURE`).
  - **Future extensions:** None anticipated beyond recognized key sequences.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Rename-aware stdin pump | `cmd/internal/termcmd/run.go` | modified | stdin reads and 50ms timer |
| `RenameTimer` / reader-result channel | `cmd/internal/termcmd/run.go` | new | timer and blocking reader |
| Frame-title renderer | `cmd/internal/termcmd/run.go` | modified | `zellij action rename-pane` |
| Terminal stream fake | `cmd/internal/termcmd/run_test.go` | modified | chunked stdin and injected Zellij runtime |

- **Rename-aware stdin pump** — owns one reader goroutine for the lifetime of
  `pumpStdin`, selects between input and the rename Escape timer, and never
  starts a nested reader that could steal post-rename bytes.
  - **Injected into:** pure decoder/editor calls; tests use `splitReader`.
  - **Future extensions:** The same timer loop can eventually consolidate
    normal-mode incomplete shortcut flushing.

- **`RenameTimer` / reader-result channel** — injected reset/stop/drain timer
  and one copied-buffer result channel around blocking `stdin.Read`.
  - **Injected into:** `pumpStdinWithTimer`; production uses `time.Timer`, tests
    use a manually fired fake.
  - **Future extensions:** Other prefix decoders can share the event loop
    without adding competing readers.

- **Frame-title renderer** — derives ordinary or rename-preview inventory under
  the mux lock, then calls the existing `RunZellijAction("rename-pane", …)`
  boundary. Entry failure aborts rename; refresh failure retains editor state;
  commit/cancel exit deterministically even if restoration fails.
  - **Injected into:** rename-mode orchestration through the existing fakeable
    `Runtime`.
  - **Future extensions:** Other transient terminal modes can render through
    the same title override.

- **Terminal stream fake** — observes child writes, mux state, title actions,
  and reported failures without a real shell.
  - **Injected into:** production `pumpStdin`.
  - **Future extensions:** Additional pane-local modes reuse the fixture.

## Estimate mapping

- Issue authoring/spec work maps to `issue-spec`.
- Task 1's rune/cursor state machine maps to `tui-screen`.
- Task 2's pure streaming decoder maps to `smaller-go-module`.
- Task 3's single-reader/timer loop, Zellij title boundary, and live
  feel/revision risk map to two `api-integration` rows.
- Task 4 maps to `atlas-docs`; the one issue-close fresh review maps to
  `milestone-review`.
- The v3.1 total is `Σdesign×1.30 + Σimpl×0.90 = 2.476`, rounded to 2.48.

## Chunk 1: Pure rename model

### Task 1: Build the rune/cursor editor

**Files:**
- Create: `cmd/internal/termcmd/rename.go`
- Create: `cmd/internal/termcmd/rename_test.go`

- [x] **Step 1: Write failing table-driven editor tests**

Cover insertion at the cursor, Unicode rune movement, Home/End,
Backspace/Delete boundaries, whitespace-trimmed non-empty commit,
empty-commit retention, cancel, and consumed no-op events. Assert the original
name is never mutated before commit.

- [x] **Step 2: Verify RED**

Run:

```bash
go test ./cmd/internal/termcmd -run 'TestRenameEditor' -count=1
```

Expected: build failure because the editor types/functions do not exist.

- [x] **Step 3: Implement the minimal pure editor**

Use `[]rune` and a cursor index. Keep transition input/output value-like; do not
call terminal or Zellij code.

- [x] **Step 4: Verify GREEN**

Run the focused command above; expected PASS.

- [x] **Step 5: Commit**

```bash
git add cmd/internal/termcmd/rename.go cmd/internal/termcmd/rename_test.go
git commit -m "#118: model terminal tab rename editing"
```

### Task 2: Build the streaming rename decoder

**Files:**
- Create: `cmd/internal/termcmd/rename_input.go`
- Create: `cmd/internal/termcmd/rename_input_test.go`

- [x] **Step 1: Write failing decoder matrices**

Accept Enter (`CR`, `LF`), Backspace (`DEL`, `BS`), Left (`ESC [ D`, `ESC O D`),
Right (`ESC [ C`, `ESC O C`), Home (`ESC [ H`, `ESC O H`, `ESC [ 1 ~`), End
(`ESC [ F`, `ESC O F`, `ESC [ 4 ~`), Delete (`ESC [ 3 ~`), and bare Escape.
Consume SGR mouse (`ESC [ < … M/m`), bracketed-paste start/end
(`ESC [ 200 ~` / `ESC [ 201 ~`) and all enclosed payload, and every sequence in
the authoritative `cmd/internal/workbenchshortcut/shortcut.go` registry through
`FindChord`/`IsChordPrefix` (including `ESC`+letter and KKP forms).

Split every recognized multi-byte control sequence at every byte boundary and
assert equivalence with unsplit input. Do the same for representative 2-, 3-,
and 4-byte UTF-8 (`é`, `界`, `🙂`). Cover bare Escape before/after
`flushEscape=true`, invalid UTF-8, and both `edit+Enter+suffix` /
`edit+Escape+suffix`. At EOF, a held bare Escape cancels; any other incomplete
control/UTF-8 prefix is consumed and EOF cancels/restores the rename mode. For
an unknown/malformed control sequence, consume only the longest prefix proven
invalid, then reprocess the first later byte that can begin printable input so
a following rune is not lost.

- [x] **Step 2: Verify RED**

```bash
go test ./cmd/internal/termcmd -run 'TestDecodeRenameInput' -count=1
```

Expected: build failure for the missing decoder.

- [x] **Step 3: Implement the pure streaming transition**

Recognize complete sequences before treating Escape as cancel; buffer every
valid prefix and incomplete UTF-8. Flush a lone Escape only when explicitly
requested by the caller. Once commit/cancel occurs, consume the batch suffix.

- [x] **Step 4: Verify GREEN**

Run the focused decoder suite; expected PASS.

- [x] **Step 5: Commit**

```bash
git add cmd/internal/termcmd/rename_input.go cmd/internal/termcmd/rename_input_test.go
git commit -m "#118: decode streaming rename input"
```

## Chunk 2: Terminal integration

### Task 3: Replace the content prompt with frame editing

**Files:**
- Modify: `cmd/internal/termcmd/run.go`
- Modify: `cmd/internal/termcmd/run_test.go`
- Modify: `tests/term-pane-shortcuts-test.sh`

- [x] **Step 1: Write failing production-stream tests**

Drive Alt+r through `pumpStdinWithTimer` with a fake mux/runtime and manually
fired timer. Assert:

- the initial title includes the existing active name and cursor marker;
- edits refresh only the frame title and write no child bytes;
- Enter commits and Escape cancels;
- same-read suffixes, shortcuts, mouse input, and bracketed paste never leak;
- child output can continue without replacing the rename frame title;
- entry-title failure aborts mode; refresh failure retains it; commit/cancel
  restoration failures preserve the specified name/mode outcome and report.
- bare Escape cancels on timeout; a recognized continuation before timeout
  edits without cancellation; a continuation after timeout is handled by the
  now-normal stream; EOF flushes a held prefix; and the sole reader remains
  active without stealing bytes across rename exit.
- one read containing `Alt+r + edits + Enter/Escape + suffix` feeds the bytes
  after Alt+r directly into rename decoding and consumes the suffix.

- [x] **Step 2: Verify RED**

```bash
go test ./cmd/internal/termcmd -run 'TestPumpStdinRename|TestRenameTitleFailure' -count=1
```

Expected: failures because Alt+r still calls `readRawPrompt`.

- [x] **Step 3: Implement one rename-aware input loop**

Remove `readRawPrompt`. Add rename state to `pumpStdin`; use one lifetime
reader goroutine that copies each read into:

```go
type stdinResult struct { data []byte; err error }
```

and select against an injected:

```go
type RenameTimer interface {
    C() <-chan time.Time
    Reset(time.Duration)
    StopAndDrain()
}
```

The production adapter wraps `time.Timer`; tests manually fire the channel and
assert reset/stop/drain calls without wall-clock sleeps. When Alt+r is decoded,
feed its already-read suffix to rename decoding before requesting another
result. On commit/cancel discard that result's remainder and resume normal
forwarding only with the next channel result. Extend the narrow mux interface with
active-name, preview-title, commit, and restore operations. Keep
`RunZellijAction` as the only title IO boundary and report errors through
`ReportShortcutError`.

- [x] **Step 4: Verify GREEN**

```bash
go test ./cmd/internal/termcmd -count=1
make build
bash tests/term-pane-shortcuts-test.sh
```

Expected: PASS; the shell inventory still classifies Alt+r as terminal-local.

- [x] **Step 5: Commit**

```bash
git add cmd/internal/termcmd/run.go cmd/internal/termcmd/run_test.go tests/term-pane-shortcuts-test.sh
git commit -m "#118: rename terminal tabs in the pane frame"
```

### Task 4: Document, verify, and smoke

**Files:**
- Modify: `README.md`
- Modify: `atlas/architecture.md`
- Modify: `workshop/issues/000118-rename-terminal-tabs-in-the-pane-frame.md`

- [ ] **Step 1: Update docs and issue evidence**

Document Alt+r's frame editor and its child-independent input ownership. Check
completed plan rows and record red/green evidence.

- [ ] **Step 2: Run complete verification**

```bash
go test ./... -count=1
make test-lua
bash tests/term-pane-shortcuts-test.sh
bash tests/review-toggle-test.sh
make runtimebundle-drift-check
zellij --config-dir zellij setup --check
git diff --check
```

Expected: all commands exit 0.

- [ ] **Step 3: Manual smoke**

In a fresh layout-3 `pair-dev`, run Neovim in the right terminal, press Alt+r,
edit a Unicode tab name with cursor/Delete/Backspace, cancel once, then commit.
Confirm the Neovim screen receives no bytes and the frame returns to ordinary
inventory. Operator verification is required before close/landing.

- [ ] **Step 4: Commit**

```bash
git add README.md atlas/architecture.md workshop/issues/000118-rename-terminal-tabs-in-the-pane-frame.md workshop/plans/000118-rename-terminal-tabs-in-the-pane-frame-plan.md
git commit -m "#118: document frame-title tab rename"
```
