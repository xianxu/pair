---
id: 000139
status: working
deps: []
github_issue:
created: 2026-08-16
updated: 2026-08-17
estimate_hours: 0.57
started: 2026-08-17T10:59:29-07:00
---

# Agy Return rewrite only in composer

## Problem

Agy currently participates in the pair Return remap convention, while overlay
handling depends on known visible prompt markers. Codex now uses a safer rule:
rewrite plain Return only when Pair positively identifies the live
composer/input box. Agy should follow the same contract so permission pickers
and future UI variants keep plain Return as confirm instead of receiving an
accidental newline.

## Spec

### Goal & Problem
Antigravity (`agy`) previously participated in un-gated Return remapping (mapping plain Enter to `\n` without checking composer presence), relying solely on `agyPickerMarkers` to catch overlay dialogs. Codex (#137) and Muse (#140) established that plain Return must rewrite to newline (`\n`) **only** when Pair positively identifies that the agent is in an active input composer with a visible cursor. If the composer state is inactive, unknown, or if an overlay is active, plain Return must pass through as bare `\r` (CR) so confirmation pickers, shortcut dialogs, and external tools receive normal confirmation instead of an unintended newline.

To preserve strict workstream isolation with open issue #140 (which tracks Muse), `agyComposerTracker` lives in `cmd/internal/wrapcmd/agy_composer.go`, following the established per-agent tracker pattern (`codex_composer.go`, `muse_composer.go`) while providing comprehensive VT escape handling and positive composer gating.

### Architecture & Components

1. **Pure Terminal State Machine (`cmd/internal/wrapcmd/agy_composer.go`)**:
   - `agyComposerTracker` maintains screen dimensions (`rows`, `cols`), cursor coordinates (`cursorRow`, `cursorCol`), cursor visibility (`cursorVisible`), and per-row painted features (`agyBorderRows`, `agyPromptRows`).
   - ANSI escape parsing handles:
     - Absolute cursor positioning: `CUP` (`\x1b[<r>;<c>H`, `\x1b[<r>;<c>f`), `CHA` (`\x1b[<c>G`), `VPA` (`\x1b[<r>d`) — parameter defaults to 1.
     - Relative cursor positioning: `CUU` (`\x1b[<n>A`), `CUD` (`\x1b[<n>B`), `CUF` (`\x1b[<n>C`), `CUB` (`\x1b[<n>D`) — parameter `<n>` defaults to 1.
     - 2-byte escapes: `RI` (`\x1bM` -> move cursor up 1 line, clamped to 1).
     - Carriage control: `\r` (col 1), `\n` (row+1, col 1).
     - Cursor visibility: `DECTCEM` (`\x1b[?25h` -> visible, `\x1b[?25l` -> hidden).
     - Screen mutation & invalidation:
       - `ED` (`\x1b[<n>J`): mode 0 (cursor to end), mode 1 (start to cursor), mode 2/3 (entire screen) clears `agyBorderRows` and `agyPromptRows` in the erased row range.
       - `EL` (`\x1b[<n>K`): erases prompt and border marks for `cursorRow`.
       - Non-border text print: printing non-border text at `cursorRow` clears `agyBorderRows[cursorRow]`.
       - `setWinsize`: prunes row entries $> rows$.
   - Chrome attribute tracking:
     - `agyBorderRows[row]`: set when Agy's horizontal border rule `─` (`\xe2\x94\x80`) is painted at `cursorRow` (requires $\ge 5$ consecutive `─` or full-line rule).
     - `agyPromptRows[row]`: set when Agy's prompt glyph `>` (`0x3e` / `\x1b[94m>`) is painted at `cursorRow` anchored at prompt column (`col <= 6`).

2. **Agy Composer Predicate**:
   - `agyComposerState.active() bool`: `cursorVisible && cursorRow > 0 && (isNearby(cursorRow) || isEnclosed(cursorRow))`, where:
     - `isNearby(r)`: checks if `agyPromptRows` or `agyBorderRows` is active on `r` or `r±1`.
     - `isEnclosed(r)`: checks if `r` is enclosed between an active top border / prompt row and bottom border row (`top <= r <= bottom` with `bottom - top <= 25`).

3. **Proxy Integration (`cmd/internal/wrapcmd/wrap.go`)**:
   - `proxy` holds `agyComposer *agyComposerTracker`.
   - `proxy.setWinsize` (at `wrap.go:2000`) calls `p.ensureAgyComposer().resize(int(ws.Rows), int(ws.Cols))`.
   - `proxy.handleChunk` feeds raw PTY stream chunk into `p.ensureAgyComposer().feed(data)`.
   - `proxy.emitPlainCR`:
     - If `pickerActive` is set $\rightarrow$ clear flag, log `adapt.Bypass` (overlay active), return bare `\r`.
     - If `agentBasename == "agy"` and `!p.agyComposerActive()` $\rightarrow$ log `adapt.Bypass` (agy composer inactive), return bare `\r`.
     - If composer is active $\rightarrow$ log `adapt.Fired` (newline remap), return `p.sendKM.plainCR` (`\n`).

4. **External Terminal Protocol Fake & Conformance (ARCH-MOCK)**:
   - Seam: `proxy.handleChunk(data, rolling)` receiving PTY output.
   - Stateful fake `agySessionFake`: models Agy UI lifecycle transitions (startup prompt, multi-line editing, overlay picker, thinking generation).
   - Integration tests replay these state transitions and verify `emitPlainCR` behavior.

### Invariants & Non-Goals
- **Precedence**: `pickerActive` from `detectAgyOverlayOpen` still takes absolute precedence over composer detection.
- **Alt+Enter**: Alt+Enter always submits bare `\r` (CR) across all agents regardless of composer state.
- **Non-blocking / Pure**: Telemetry and tracker failures are non-fatal and isolated.
- **Workstream Isolation**: Codex and Muse trackers remain untouched; no dependency on open issue #140.

## Done when

- `agyComposerTracker` encapsulates VT escape sequences, relative movements, and screen mutation invalidations cleanly (ARCH-DRY, ARCH-PURE).
- Agy plain Return rewrites to LF (`\n`) only inside a positively detected composer with visible cursor.
- Agy permission prompts/pickers still receive bare CR (`\r`).
- Inactive, hidden-cursor, or unknown Agy UI state receives bare CR (`\r`).
- Stateful fake and regression tests in `cmd/internal/wrapcmd/agy_return_test.go` and `cmd/internal/wrapcmd/agy_composer_test.go` pass (ARCH-MOCK).
- Existing Codex and Muse return remapping tests pass unchanged.
- `atlas/architecture.md` and `atlas/how-to-bring-up-a-new-harness-cli.md` updated with Agy positive composer gating details.

## Plan

- [ ] Implement `agyComposerTracker` in `cmd/internal/wrapcmd/agy_composer.go` with relative/absolute cursor movements and mutation invalidation rules.
- [ ] Add unit tests in `cmd/internal/wrapcmd/agy_composer_test.go`.
- [ ] Wire `agy` composer tracking into `proxy` in `cmd/internal/wrapcmd/wrap.go` (in `setWinsize`, `handleChunk`, and `emitPlainCR`).
- [ ] Add return routing and stateful fake replay tests in `cmd/internal/wrapcmd/agy_return_test.go`.
- [ ] Run full test suite (`go test ./...`) to ensure zero regressions across all agents.
- [ ] Update `atlas/architecture.md` and `atlas/how-to-bring-up-a-new-harness-cli.md`.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.

```estimate
model: estimate-logic-v3.1

item: issue-spec design=0.08 impl=0.02
item: smaller-go-module design=0.15 impl=0.25
total: 0.57
```

## Log

### 2026-08-16

- Follow-up from pair#137: Codex now rewrites Return only under positive live
  composer detection; agy should adopt the same safety model.
- Updated after pair#142 design review: Codex's cursor/paint detector is
  Codex-specific; agy should reuse the positive-detection contract, not the
  exact terminal heuristic.

### 2026-08-17

- Brainstormed positive composer detection for Agy. Captured raw terminal output from live `agy` CLI session.
- Identified that `agy` uses relative cursor movements (`CUU` `\x1b[2A`, `CHA` `\x1b[6G`, `RI` `\x1bM`) to position the cursor at the prompt row between horizontal rules `───` (`\xe2\x94\x80`).
- Sourced plan-quality gate findings: isolated Agy tracker ownership to `agy_composer.go` (avoiding entanglement with open issue #140), specified screen invalidation rules, named `proxy.setWinsize`, and defined stateful fake replay (ARCH-MOCK).
