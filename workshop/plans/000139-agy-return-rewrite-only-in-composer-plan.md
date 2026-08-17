# Agy Return Rewrite Only in Composer Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Gate Antigravity (`agy`) plain Return rewriting behind positive composer detection with visible cursor, falling back to bare CR on inactive states and overlays, backed by a unified terminal state tracker across Codex, Muse, and Agy.

**Architecture:** A pure VT100/ANSI screen state machine (`terminalTracker`) in `cmd/internal/wrapcmd/terminal_tracker.go` parses terminal escape sequences, tracks cursor coordinates (including relative moves like `CUU`, `CHA`, `RI`), visibility (`DECTCEM`), and per-row painted chrome attributes. Agent-specific predicates (`codex`, `muse`, `agy`) query this unified model, and `wrap.go` gates plain Return newline remap accordingly (ARCH-DRY, ARCH-PURE).

**Tech Stack:** Go, standard library (`sync`, `strconv`, `bytes`, `strings`), `cmd/internal/ansi`, `cmd/internal/adapt`.

---

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `terminalTracker` | `cmd/internal/wrapcmd/terminal_tracker.go` | new |
| `terminalState` | `cmd/internal/wrapcmd/terminal_tracker.go` | new |

- **`terminalTracker`** — Concurrency-safe terminal grid model tracking dimensions, cursor coordinates (row, col), visibility, SGR styles, line/display erases, and agent chrome markers across chunks.
  - **Relationships:** Owned by `proxy` (1:1 per wrapped agent session).
  - **DRY rationale (ARCH-DRY):** Replaces duplicated VT escape parsing in `codexComposerTracker` and `museComposerTracker` with one shared parser handling absolute and relative cursor positioning (`CUP`, `CHA`, `VPA`, `CUU`, `CUD`, `CUF`, `CUB`, `RI`), visibility, and erases.
  - **Purity (ARCH-PURE):** Pure deterministic state machine tested via byte feeds without OS or PTY mocks.
  - **Future extensions:** Support for additional harnesses (e.g. Claude positive composer detection or new terminal agents).

- **`terminalState`** — Immutable snapshot of terminal dimensions, cursor position, visibility, and row-level chrome flags.
  - **Predicates:** `CodexActive() bool`, `MuseActive() bool`, `AgyActive() bool`.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `proxy.emitPlainCR` | `cmd/internal/wrapcmd/wrap.go` | modified | Stdin translation pump |
| `proxy.handleChunk` | `cmd/internal/wrapcmd/wrap.go` | modified | Master PTY output stream |
| `proxy.resizeWinsize` | `cmd/internal/wrapcmd/wrap.go` | modified | Terminal resize signals |

- **`proxy.emitPlainCR`** — Stdin translation function for Enter keystrokes.
  - **Injected into:** `proxy.translateChunk`.
  - **Behavior:** Checks `pickerActive` first (bare `\r`), then `agentBasename` composer active status (`codex`, `muse`, `agy`). Returns bare `\r` if inactive, or `p.sendKM.plainCR` (`\n`) if active.

---

## Tasks

### Task 1: Pure Terminal State Tracker (`terminalTracker`)

**Files:**
- Create: `cmd/internal/wrapcmd/terminal_tracker.go`
- Test: `cmd/internal/wrapcmd/terminal_tracker_test.go`

- [ ] **Step 1: Write unit tests for `terminalTracker`**

Create `cmd/internal/wrapcmd/terminal_tracker_test.go` with tests covering:
1. `TestTerminalTracker_CursorPositioning`: Absolute (`H`, `f`, `G`, `d`) and relative (`A`, `B`, `C`, `D`, `\x1bM`, `\r`, `\n`) cursor movements with default parameter `= 1`.
2. `TestTerminalTracker_CursorVisibility`: `\x1b[?25h` (visible) and `\x1b[?25l` (hidden).
3. `TestTerminalTracker_Erases`: `ED` (`J`), `EL` (`K`), `ECH` (`X`).
4. `TestTerminalTracker_CodexPredicate`: BG `48;2;57;57;57` detection with $\ge 2$ rows in `[cursorRow-1, cursorRow+1]`.
5. `TestTerminalTracker_MusePredicate`: Prompt glyph `›` (`\xe2\x9f\xa9`) detection in `[cursorRow-1, cursorRow+1]`.
6. `TestTerminalTracker_AgyPredicate_SingleLine`: Border `───` (`\xe2\x94\x80`) and prompt `>` (`\x1b[94m>`) on prompt row with visible cursor.
7. `TestTerminalTracker_AgyPredicate_MultiLine`: Multi-line composer where cursor is enclosed between top border/prompt and bottom border.
8. `TestTerminalTracker_AgyPredicate_RejectsHiddenCursor`: Returns false when `\x1b[?25l` is active.
9. `TestTerminalTracker_AgyPredicate_RejectsUnrelatedOutput`: Returns false for body text containing `>` away from prompt column or without visible cursor.
10. `TestTerminalTracker_SplitChunks`: Carryover of split ANSI escapes and multi-byte UTF-8 sequences (`\xe2\x94\x80`, `\xe2\x9f\xa9`).

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -v ./cmd/internal/wrapcmd -run TestTerminalTracker`
Expected: FAIL (compilation error: `terminalTracker` undefined).

- [ ] **Step 3: Implement `terminalTracker`**

Create `cmd/internal/wrapcmd/terminal_tracker.go`:
- Implement `terminalTracker` with `sync.Mutex`, `rows`, `cols`, `cursorRow`, `cursorCol`, `cursorVisible`, `bg`, `pending`, and maps:
  - `codexBGRows map[int]bool`
  - `musePromptRows map[int]bool`
  - `agyBorderRows map[int]bool`
  - `agyPromptRows map[int]bool`
- Implement `resize(rows, cols int)` with row/column clamping.
- Implement `feed(data []byte)` parsing ANSI escapes:
  - 2-byte escape: `\x1bM` (`RI` -> `t.cursorRow--`, clamped).
  - CSI sequences: `CUP` (`H`, `f`), `CHA` (`G`), `VPA` (`d`), `CUU` (`A`), `CUD` (`B`), `CUF` (`C`), `CUB` (`D`), `DECTCEM` (`?25h`/`?25l`), `SGR` (`m`), `ED` (`J`), `EL` (`K`), `ECH` (`X`).
  - Text bytes: Carriage return `\r` (`cursorCol = 1`), Newline `\n` (`cursorRow++`, `cursorCol = 1`), printable characters (`cursorCol++`).
  - Detect Chrome:
    - Codex: `bg == "2;57;57;57"` on print or `EL`.
    - Muse: `\xe2\x9f\xa9` prompt glyph.
    - Agy: `\xe2\x94\x80` horizontal border (track count per row $\ge 5$), `>` at `col <= 6` (prompt glyph).
- Implement `state() terminalState` returning snapshot.
- Implement predicates on `terminalState`:
  - `CodexActive() bool`: `cursorVisible && cursorRow > 0 && count(codexBGRows in [cursorRow-1, cursorRow+1]) >= 2`.
  - `MuseActive() bool`: `cursorVisible && cursorRow > 0 && count(musePromptRows in [cursorRow-1, cursorRow+1]) >= 1`.
  - `AgyActive() bool`: `cursorVisible && cursorRow > 0 && (isAgyPromptOrBorderNearby(cursorRow) || isEnclosedInAgyBox(cursorRow))`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -v ./cmd/internal/wrapcmd -run TestTerminalTracker`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/wrapcmd/terminal_tracker.go cmd/internal/wrapcmd/terminal_tracker_test.go
git commit -m "wrap: #139: implement unified terminal state tracker and agent predicates

Co-Authored-By: Antigravity <antigravity@google.com>"
```

---

### Task 2: Refactor Codex & Muse Trackers to derive from `terminalTracker`

**Files:**
- Modify: `cmd/internal/wrapcmd/codex_composer.go`
- Modify: `cmd/internal/wrapcmd/muse_composer.go`
- Test: `cmd/internal/wrapcmd/codex_composer_test.go`
- Test: `cmd/internal/wrapcmd/muse_composer_test.go`

- [ ] **Step 1: Adapt `codexComposerTracker` and `museComposerTracker` to use `terminalTracker`**

Refactor `codexComposerTracker` and `museComposerTracker` to wrap or alias `terminalTracker`, preserving public methods (`resize`, `feed`, `state`, `active`) so existing callers and unit tests continue to compile and pass.

- [ ] **Step 2: Run Codex & Muse composer unit tests**

Run: `go test -v ./cmd/internal/wrapcmd -run "TestCodexComposer|TestMuseComposer"`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add cmd/internal/wrapcmd/codex_composer.go cmd/internal/wrapcmd/muse_composer.go
git commit -m "wrap: #139: derive codex and muse composer trackers from terminalTracker

Co-Authored-By: Antigravity <antigravity@google.com>"
```

---

### Task 3: Wire Agy Positive Composer Gate into `wrap.go` & Add Return Tests

**Files:**
- Modify: `cmd/internal/wrapcmd/wrap.go`
- Create: `cmd/internal/wrapcmd/agy_return_test.go`

- [ ] **Step 1: Write failing Return routing tests for Agy**

Create `cmd/internal/wrapcmd/agy_return_test.go` with tests:
1. `TestEmitPlainCR_AgyComposerActiveRewritesToNewline`: Feed agy startup prompt with visible cursor $\rightarrow$ `emitPlainCR` returns `[]byte{'\n'}`.
2. `TestEmitPlainCR_AgyMultiLineComposerRewritesToNewline`: Multi-line prompt with cursor on line 3 $\rightarrow$ returns `[]byte{'\n'}`.
3. `TestEmitPlainCR_AgyComposerInactiveSendsBareCR`: Inactive composer (no prompt/border) $\rightarrow$ returns `[]byte{'\r'}`.
4. `TestEmitPlainCR_AgyHiddenCursorSendsBareCR`: Composer painted but cursor hidden (`\x1b[?25l`) $\rightarrow$ returns `[]byte{'\r'}`.
5. `TestEmitPlainCR_AgyOverlayBeatsComposer`: `pickerActive` is true $\rightarrow$ returns `[]byte{'\r'}` and clears `pickerActive`.
6. `TestEmitPlainCR_AgyUnknownComposerSendsBareCR`: Zero-state proxy $\rightarrow$ returns `[]byte{'\r'}`.
7. `TestHandleChunk_AgyFeedsComposerTracker`: `handleChunk` with agy prompt activates composer.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -v ./cmd/internal/wrapcmd -run TestEmitPlainCR_Agy`
Expected: FAIL (agy currently rewrites unconditionally).

- [ ] **Step 3: Wire Agy composer gating in `wrap.go`**

In `cmd/internal/wrapcmd/wrap.go`:
- Add `agyComposer *terminalTracker` field to `proxy` (or unify `codexComposer`, `museComposer`, `agyComposer`).
- Add helper `p.agyComposerActive() bool` and `p.ensureAgyComposer() *terminalTracker`.
- In `resizeWinsize`: Call `p.ensureAgyComposer().resize(int(ws.Rows), int(ws.Cols))` when `agentBasename == "agy"`.
- In `handleChunk`: Call `p.ensureAgyComposer().feed(data)` when `agentBasename == "agy"`.
- In `emitPlainCR`:
  ```go
  if p.agentBasename == "agy" && !p.agyComposerActive() {
      p.adapt.Log(1, "return-remap", adapt.Bypass, "plain Enter → bare CR (agy composer inactive)")
      return append(out, '\r')
  }
  ```

- [ ] **Step 4: Run Agy return tests**

Run: `go test -v ./cmd/internal/wrapcmd -run TestEmitPlainCR_Agy`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/wrapcmd/wrap.go cmd/internal/wrapcmd/agy_return_test.go
git commit -m "wrap: #139: gate agy plain return behind positive composer detection

Co-Authored-By: Antigravity <antigravity@google.com>"
```

---

### Task 4: Documentation, Atlas Update & Full Test Suite Verification

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `atlas/how-to-bring-up-a-new-harness-cli.md`

- [ ] **Step 1: Update documentation**

Update `atlas/architecture.md` and `atlas/how-to-bring-up-a-new-harness-cli.md` to document that `agy` uses positive composer detection via `terminalTracker` (matching Codex and Muse), where plain Return rewrites to LF only when the composer box is positively detected with a visible cursor.

- [ ] **Step 2: Run full unit and integration test suite**

Run: `go test ./...`
Expected: PASS across all packages (`cmd/...`, `pkg/...`).

- [ ] **Step 3: Run make test suite**

Run: `make test`
Expected: PASS across all test suites.

- [ ] **Step 4: Commit**

```bash
git add atlas/architecture.md atlas/how-to-bring-up-a-new-harness-cli.md
git commit -m "atlas: #139: document positive composer detection for agy

Co-Authored-By: Antigravity <antigravity@google.com>"
```
