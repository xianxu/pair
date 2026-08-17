---
id: 000139
status: working
deps: []
github_issue:
created: 2026-08-16
updated: 2026-08-17
estimate_hours:
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

To implement this robustly across agents while maintaining `ARCH-DRY` and `ARCH-PURE`, we extract a **Unified Terminal State Tracker** (`terminalTracker`) that consolidates ANSI/VT parsing, cursor geometry, and screen attribute tracking across all positive-gated agents (`codex`, `muse`, `agy`).

### Architecture & Components

1. **Pure Terminal State Machine (`cmd/internal/wrapcmd/terminal_tracker.go`)**:
   - `terminalTracker` maintains screen dimensions (`rows`, `cols`), cursor coordinates (`cursorRow`, `cursorCol`), cursor visibility (`cursorVisible`), and per-row painted features.
   - ANSI escape parsing handles:
     - Absolute cursor positioning: `CUP` (`\x1b[<r>;<c>H`, `\x1b[<r>;<c>f`), `CHA` (`\x1b[<c>G`), `VPA` (`\x1b[<r>d`) — parameter defaults to 1.
     - Relative cursor positioning: `CUU` (`\x1b[<n>A`), `CUD` (`\x1b[<n>B`), `CUF` (`\x1b[<n>C`), `CUB` (`\x1b[<n>D`) — parameter `<n>` defaults to 1.
     - 2-byte escapes: `RI` (`\x1bM` -> move cursor up 1 line).
     - Carriage control: `\r` (col 1), `\n` (row+1, col 1).
     - Cursor visibility: `DECTCEM` (`\x1b[?25h` -> visible, `\x1b[?25l` -> hidden).
     - Display and line erases: `ED` (`\x1b[<n>J`), `EL` (`\x1b[<n>K`), `ECH` (`\x1b[<n>X`).
     - SGR styling: RGB background (`48;2;r;g;bm`), 256-color background (`48;5;nm`), default background (`49m`, `0m`).
   - Tracks agent-specific chrome attributes per row:
     - `codexBGRows[row]`: set when Codex's background `48;2;57;57;57` is painted at `cursorRow`.
     - `musePromptRows[row]`: set when Muse's prompt glyph `›` (`\xe2\x9f\xa9`) is painted at `cursorRow`.
     - `agyBorderRows[row]`: set when Agy's horizontal border rule `─` (`\xe2\x94\x80`) is painted at `cursorRow` (requires $\ge 5$ consecutive `─` or full-line rule).
     - `agyPromptRows[row]`: set when Agy's prompt glyph `>` (`0x3e` / `\x1b[94m>`) is painted at `cursorRow` anchored at prompt column (`col <= 6`).

2. **Agent Composer Predicates**:
   - **Codex Predicate**: `cursorVisible && cursorRow > 0 && count(codexBGRows in [cursorRow-1, cursorRow+1]) >= 2`.
   - **Muse Predicate**: `cursorVisible && cursorRow > 0 && count(musePromptRows in [cursorRow-1, cursorRow+1]) >= 1`.
   - **Agy Predicate**: `cursorVisible && cursorRow > 0 && (isAgyPromptOrBorderNearby(cursorRow) || isEnclosedInAgyBox(cursorRow))`, where:
     - `isAgyPromptOrBorderNearby(r)`: checks if `agyPromptRows` or `agyBorderRows` is active on `r` or `r±1`.
     - `isEnclosedInAgyBox(r)`: checks if `r` is enclosed between an active top border / prompt row and bottom border row (`top <= r <= bottom` with `bottom - top <= 25`).

3. **Proxy Integration (`cmd/internal/wrapcmd/wrap.go`)**:
   - `proxy` holds a unified `composerTracker *terminalTracker`.
   - `winsize` updates call `p.composerTracker.resize(ws.Rows, ws.Cols)`.
   - `handleChunk` feeds raw PTY stream chunk into `p.composerTracker.feed(data)`.
   - `emitPlainCR`:
     - If `pickerActive` is set $\rightarrow$ clear flag, log `adapt.Bypass` (overlay active), return bare `\r`.
     - If `agentBasename == "codex"` and `!p.codexComposerActive()` $\rightarrow$ log `adapt.Bypass` (codex composer inactive), return bare `\r`.
     - If `agentBasename == "muse"` and `!p.museComposerActive()` $\rightarrow$ log `adapt.Bypass` (muse composer inactive), return bare `\r`.
     - If `agentBasename == "agy"` and `!p.agyComposerActive()` $\rightarrow$ log `adapt.Bypass` (agy composer inactive), return bare `\r`.
     - If composer is active $\rightarrow$ log `adapt.Fired` (newline remap), return `p.sendKM.plainCR` (`\n`).

### Invariants & Non-Goals
- **Precedence**: `pickerActive` from `detectAgyOverlayOpen` still takes absolute precedence over composer detection.
- **Alt+Enter**: Alt+Enter always submits bare `\r` (CR) across all agents regardless of composer state.
- **Non-blocking / Pure**: Telemetry and tracker failures are non-fatal and isolated.
- **Claude untouched**: Claude continues using portable `\<Enter>` keymap.

## Done when

- Unified `terminalTracker` encapsulates VT escape sequences and cursor geometry cleanly (ARCH-DRY, ARCH-PURE).
- Agy plain Return rewrites to LF (`\n`) only inside a positively detected composer with visible cursor.
- Agy permission prompts/pickers still receive bare CR (`\r`).
- Inactive, hidden-cursor, or unknown Agy UI state receives bare CR (`\r`).
- Existing Codex and Muse return remapping tests pass unchanged against the unified tracker.
- Comprehensive unit tests in `cmd/internal/wrapcmd/agy_return_test.go` and `cmd/internal/wrapcmd/terminal_tracker_test.go`.
- `atlas/architecture.md` and `atlas/how-to-bring-up-a-new-harness-cli.md` updated with Agy positive composer gating details.

## Plan

- [ ] Implement `terminalTracker` in `cmd/internal/wrapcmd/terminal_tracker.go` supporting full relative/absolute cursor movements and per-agent chrome predicates.
- [ ] Migrate `codex` and `muse` trackers to derive from `terminalTracker` (or consolidate `codex_composer.go` and `muse_composer.go`).
- [ ] Wire `agy` composer tracking into `proxy` in `cmd/internal/wrapcmd/wrap.go` and gate `emitPlainCR`.
- [ ] Add unit tests in `cmd/internal/wrapcmd/terminal_tracker_test.go` and `cmd/internal/wrapcmd/agy_return_test.go`.
- [ ] Run full test suite (`go test ./...`) to ensure zero regressions across Codex, Muse, and Claude.
- [ ] Update `atlas/architecture.md` and `atlas/how-to-bring-up-a-new-harness-cli.md`.

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
- Agreed on Approach 2: Unified Terminal State Tracker (`terminalTracker`) consolidating VT escape parsing, cursor geometry, and agent predicates across Codex, Muse, and Agy (ARCH-DRY, ARCH-PURE).
