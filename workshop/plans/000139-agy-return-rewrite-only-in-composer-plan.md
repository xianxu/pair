# Agy Return Rewrite Only in Composer Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Gate Antigravity (`agy`) plain Return rewriting behind positive composer detection with visible cursor, falling back to bare CR on inactive states and overlays, backed by a dedicated pure tracker and stateful protocol fake.

**Architecture:** A pure VT100/ANSI screen state machine (`agyComposerTracker`) in `cmd/internal/wrapcmd/agy_composer.go` parses terminal escape sequences, tracks cursor coordinates (including relative moves like `CUU`, `CHA`, `RI`), visibility (`DECTCEM`), and per-row painted chrome attributes with explicit screen-mutation invalidation rules (`ED`, `EL`, non-border text overwrite, `setWinsize` row pruning). `wrap.go` gates plain Return newline remap accordingly (ARCH-DRY, ARCH-PURE, ARCH-MOCK).

**Tech Stack:** Go, standard library (`sync`, `strconv`, `bytes`, `strings`), `cmd/internal/ansi`, `cmd/internal/adapt`.

---

## Core concepts

### Pure entities (the conceptual core)

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

- [ ] **Step 1: Write unit tests for `agyComposerTracker` using risky-function strategies**

Create `cmd/internal/wrapcmd/agy_composer_test.go` testing three named risky functions:
1. `agyComposerTracker.feed`: Test parser resilience over arbitrary malformed, split UTF-8 (`\xe2\x94\x80`), and unterminated ANSI sequences (`\x1b[...`) with bounded carryover.
2. `agyComposerTracker.applyEscape`: Test cursor movement (`CUP`, `CHA`, `VPA`, `CUU`, `CUD`, `CUF`, `CUB`, `RI`), visibility toggles (`?25h`/`?25l`), and screen/line erase invalidations (`ED`, `EL`).
3. `agyComposerState.active`: Test positive and negative boundary conditions across single-line prompt, multi-line enclosed composer box, hidden cursor, stale distant chrome, and erased display.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -v ./cmd/internal/wrapcmd -run TestAgyComposer`
Expected: FAIL (compilation error: `agyComposerTracker` undefined).

- [ ] **Step 3: Implement `agyComposerTracker`**

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

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -v ./cmd/internal/wrapcmd -run TestAgyComposer`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/wrapcmd/agy_composer.go cmd/internal/wrapcmd/agy_composer_test.go
git commit -m "wrap: #139: implement agy composer tracker and mutation invalidation rules

Co-Authored-By: Antigravity <antigravity@google.com>"
```

---

### Task 2: Wire Agy Positive Composer Gate into `wrap.go` & Add Stateful Protocol Replay Tests

**Files:**
- Modify: `cmd/internal/wrapcmd/wrap.go`
- Create: `cmd/internal/wrapcmd/agy_return_test.go`

- [ ] **Step 1: Write Return routing & protocol replay tests using risky-function strategies**

Create `cmd/internal/wrapcmd/agy_return_test.go`:
1. `proxy.emitPlainCR`: Matrix testing across active composer (`\n`), inactive composer (`\r`), unknown state (`\r`), and overlay precedence (`pickerActive` returns `\r` and clears flag).
2. `proxy.handleChunk`: Stateful lifecycle replay using `agySessionFake` through startup, multi-line typing, thinking/generation (`\x1b[?25l`), and permission picker overlay.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -v ./cmd/internal/wrapcmd -run TestEmitPlainCR_Agy`
Expected: FAIL (agy currently rewrites unconditionally).

- [ ] **Step 3: Wire Agy composer gating in `wrap.go`**

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

### Task 3: Documentation, Atlas Update & Full Test Suite Verification

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `atlas/how-to-bring-up-a-new-harness-cli.md`

- [ ] **Step 1: Update documentation**

Update `atlas/architecture.md` and `atlas/how-to-bring-up-a-new-harness-cli.md` to document that `agy` uses positive composer detection via `agyComposerTracker` (matching Codex and Muse), where plain Return rewrites to LF only when the composer box is positively detected with a visible cursor.

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
