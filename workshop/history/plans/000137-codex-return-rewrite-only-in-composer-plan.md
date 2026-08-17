# Codex Composer-Gated Return Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite plain Return in the Codex pane only when the live bottom-anchored Codex composer is positively detected.

**Architecture:** Add a small pure Codex composer tracker that consumes raw PTY bytes and tracks only the terminal facts needed for this decision: pane size, cursor position/visibility, current SGR background, and bottom-band erase-line paints. `pair-wrap` remains the IO shell that feeds the tracker and uses its state in `emitPlainCR`; existing overlay detection still takes precedence. This follows ARCH-PURE and ARCH-DRY by reusing the wrapper path and avoiding another menu-marker family.

**Tech Stack:** Go, existing `cmd/internal/wrapcmd` tests, raw ANSI/CSI bytes already available in `pair-wrap`.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `codexComposerTracker` | `cmd/internal/wrapcmd/codex_composer.go` | new |
| `codexComposerState` | `cmd/internal/wrapcmd/codex_composer.go` | new |

- **`codexComposerTracker`** — minimal terminal-state parser for Codex composer detection.
  - **Relationships:** 1:1 with a `proxy` for Codex sessions only.
  - **DRY rationale:** Centralizes the positive-composer signal instead of growing `codexPickerMarkers` for every non-composer UI.
  - **Future extensions:** Add agent-specific trackers later behind the same "positive composer" contract.

- **`codexComposerState`** — immutable snapshot used by Return routing.
  - **Relationships:** owned by `codexComposerTracker`, read by `proxy.emitPlainCR`.
  - **DRY rationale:** Separates "parse raw terminal state" from "route Return".
  - **Future extensions:** Include confidence/debug fields for adapt telemetry if drift appears.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `pair-wrap` Codex output pump | `cmd/internal/wrapcmd/wrap.go` | modified | Codex PTY output |
| `pair-wrap` Return translator | `cmd/internal/wrapcmd/wrap.go` | modified | User keyboard input |

- **`pair-wrap` Codex output pump** — feeds raw chunks and window size into the tracker from `handleChunk` / `setWinsize`.
  - **Injected into:** `codexComposerTracker` as raw bytes; tests use byte fixtures, not a live Codex process.
  - **Future extensions:** Add adapt `composer-detect` fired/near-miss telemetry after the signal proves useful.

- **`pair-wrap` Return translator** — uses tracker state to decide whether Codex plain Return rewrites to LF.
  - **Injected into:** `emitPlainCR`; overlay-active bypass remains first.
  - **Future extensions:** Other agents can opt into positive-composer gating once they have a tracker.

## Tasks

### Task 1: Add Pure Codex Composer Tracker

**Files:**
- Create: `cmd/internal/wrapcmd/codex_composer.go`
- Create: `cmd/internal/wrapcmd/codex_composer_test.go`

**Unit-tested functions and strategy:**
- `(*codexComposerTracker).resize(rows, cols int)` — adversarial pane geometry: zero sizes, shrink after active state, and normal 38-row pane; guard that impossible geometry clears active state.
- `(*codexComposerTracker).feed(data []byte)` — adversarial raw PTY chunks: split CSI/SGR sequences, unrelated cursor moves, hidden cursor, stale bottom-row clears, and bottom-band composer paints; guard that only the Codex composer background plus visible bottom-band cursor activates.
- `(*codexComposerTracker).state() codexComposerState` / `codexComposerState.active()` — adversarial stale/incomplete snapshots; guard that active requires visible cursor plus enough bottom-band painted rows.

- [x] **Step 1: Write failing composer-positive test**

Test raw bytes shaped like observed Codex output: a 38-row pane, `CSI 35;1H`, `CSI 48;2;57;57;57m`, `CSI K` on rows 35-37, `CSI ?25h`, and `CSI 36;3H`. Expected: composer active.

- [x] **Step 2: Write failing negative tests**

Cover hidden cursor (`CSI ?25l`), composer background away from the bottom band, and missing composer background. Expected: inactive.

- [x] **Step 3: Implement minimal parser**

Parse only:
- `CSI <row>;<col> H` / `f` cursor position,
- `CSI ?25h` / `CSI ?25l` cursor visibility,
- `CSI ... m` SGR reset/background including `0`, `49`, and `48;2;57;57;57`,
- `CSI K` erase-line, recorded as "current row was painted with composer background".

Ignore other CSI/OSC/text except advancing cursor columns for printable bytes. Clamp rows/cols to current pane size.

- [x] **Step 4: Define active predicate**

Active iff:
- pane has non-zero rows/cols,
- cursor is visible,
- cursor row is within the bottom composer band, default last 4 rows excluding the status row,
- at least two rows in that same bottom band were recently erased/painted with `48;2;57;57;57`,
- cursor row is one of those bottom-band rows or directly inside the painted band.

### Task 2: Gate Codex Plain Return on Composer State

**Files:**
- Modify: `cmd/internal/wrapcmd/wrap.go`
- Modify: `cmd/internal/wrapcmd/overlay_test.go` or new focused Return test file

**Unit-tested functions and strategy:**
- `(*proxy).emitPlainCR(out []byte)` — adversarial state combinations: Codex composer active/inactive, `pickerActive` set, and non-Codex agents; guard that overlay bypass wins, Codex rewrites only with active composer, and other agents keep existing remaps.
- `(*proxy).handleChunk(data []byte, rolling *[]byte)` — raw-output integration seam using a controlled proxy and byte fixture; guard that Codex chunks feed the tracker before Return translation without requiring a live Codex process.

- [x] **Step 1: Write failing Return tests**

Add tests proving:
- Codex plain Return with composer active emits LF.
- Codex plain Return with composer inactive emits bare CR.
- `pickerActive` still emits bare CR and clears even when composer is active.
- Claude/agy/muse keep existing behavior.

- [x] **Step 2: Wire tracker into proxy**

Add a Codex-only tracker field to `proxy`. Initialize lazily when `agentBasename == "codex"` and Return remap is enabled. Feed `resize(rows, cols)` from `setWinsize` and raw chunks from `handleChunk` before `checkOverlayOpen`.

- [x] **Step 3: Change `emitPlainCR` routing**

Order:
1. If `pickerActive`, send bare CR and clear existing overlay state.
2. If agent is Codex and composer tracker is absent/inactive, send bare CR.
3. Otherwise use existing `sendKM.plainCR`.

Do not change Alt+Return.

### Task 3: Update Harness Guide and Architecture Notes

**Files:**
- Modify: `atlas/how-to-bring-up-a-new-harness-cli.md`
- Modify: `atlas/architecture.md`

- [x] **Step 1: Update the guide**

Aspect 1/2 should say agent-pane Return rewriting is a composer-only behavior. New integrations should positively detect their composer/input box; overlay/menu marker lists are fallback/override signals, not the primary strategy.

- [x] **Step 2: Update architecture**

Document Codex's bottom-anchored composer signal: visible cursor near bottom plus `48;2;57;57;57` composer background on bottom-band rows.

### Task 4: Verify and Close

**Files:**
- Modify: `workshop/issues/000137-codex-return-rewrite-only-in-composer.md`

- [x] **Step 1: Run focused tests**

Run: `go test ./cmd/internal/wrapcmd -run 'TestCodexComposer|Test.*PlainEnter|TestTranslateChunk_Codex|TestEmitPlainCR|TestHandleChunk_CodexFeedsComposerTracker' -count=1`

- [x] **Step 2: Run package and repo tests**

Run: `go test ./cmd/internal/wrapcmd -count=1` and `go test ./...`

- [x] **Step 3: Validate issue and whitespace**

Run: `sdlc issue validate --issue 137` and `git diff --check`.

- [ ] **Step 4: Close with evidence**

Use `sdlc close --issue 137 --verified '<commands and behavior evidence>'`.
