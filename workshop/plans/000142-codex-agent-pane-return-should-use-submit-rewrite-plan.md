# Codex Agent-Pane Return Rewrite Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make plain Return in the Codex agent pane insert a newline whenever Codex's active composer is visible.

**Architecture:** Keep the existing positive composer detector and overlay precedence. Extend the Codex detector so active composer state is anchored on a visible cursor on or next to recently painted composer-background rows, rather than on Pair's outer PTY screen geometry. A visible cursor alone is deliberately insufficient because normal terminal surfaces often show cursors outside composer state. This satisfies ARCH-DRY and ARCH-PURE by keeping the behavior inside `codexComposerTracker`.

**Tech Stack:** Go, `cmd/internal/wrapcmd`, existing Go unit tests.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `codexComposerTracker` | `cmd/internal/wrapcmd/codex_composer.go` | modified |

**`codexComposerTracker`** — Tracks enough terminal cursor and paint state to decide whether Codex's composer is active before rewriting Return.

- **Relationships:** 1:1 with a running Codex `proxy`; owns cursor visibility, cursor position, and observed composer-background rows.
- **DRY rationale:** Centralizes Codex composer detection so `emitPlainCR` only asks for active/inactive state.
- **Future extensions:** Other Codex composer paint patterns can add evidence to the same tracker without changing input translation.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `proxy.emitPlainCR` | `cmd/internal/wrapcmd/wrap.go` | unchanged behavior, tested | user Return input to agent PTY |
| `proxy.handleChunk` | `cmd/internal/wrapcmd/wrap.go` | unchanged behavior, tested | agent PTY output stream |

**`proxy.emitPlainCR`** — Applies Pair's Return policy after checking overlay and Codex composer state.

- **Injected into:** Input translation path.
- **Future extensions:** Other agents can add composer detectors behind the same policy.

**`proxy.handleChunk`** — Feeds Codex PTY output into the composer tracker.

- **Injected into:** Master PTY pump.
- **Future extensions:** More output-derived state can be attached here if it belongs to live terminal state.

## Non-Goals

- Do not change overlay precedence: Pair overlays still receive bare CR.
- Do not change Alt+Return submit semantics.
- Do not generalize the Codex cursor/paint heuristic to Claude, Agy, or Muse without first identifying each agent's native composer-availability signal.
- Do not infer a second logical screen height or depend on default-cleared rows below the composer.
- Do not treat a visible cursor alone as sufficient composer evidence without nearby composer-background rows.

## Test Strategy

- `codexComposerTracker.feed` over split, partial, and malformed ANSI paint streams -> keep pending CSI handling intact and never panic or invent a cursor row.
- `codexComposerState.active` over ambiguous cursor placement -> require cursor visibility plus at least two composer-background rows adjacent to the cursor row.
- `proxy.emitPlainCR` over overlay-vs-composer precedence -> assert overlay remains a bare-CR bypass and inferred active Codex composer emits LF.

## Chunk 1: Cursor-Anchored Composer Detection

### Task 1: Capture the Regression as a Failing Test

**Files:**
- Modify: `cmd/internal/wrapcmd/codex_composer_test.go`
- Modify: `cmd/internal/wrapcmd/codex_return_test.go`

- [x] Add the smallest failing unit coverage for `codexComposerState.active` using the observed cursor-visible stream.
- [x] Add the smallest failing Return-path coverage for `proxy.emitPlainCR` using that cursor-anchored active composer state.
- [x] Run `go test ./cmd/internal/wrapcmd -run 'TestCodexComposerTracker|TestEmitPlainCR' -count=1` and confirm the new tests fail for inactive composer detection.

### Task 2: Anchor Codex Composer Detection on Cursor Evidence

**Files:**
- Modify: `cmd/internal/wrapcmd/codex_composer.go`

- [x] Continue tracking composer-background `CSI K` rows even when they are outside the outer winsize-derived composer band.
- [x] Track printable bytes written while the Codex composer background is active so rows do not depend on `CSI K` alone.
- [x] In `codexComposerState.active`, require cursor visibility, a valid cursor row, at least two observed composer-background rows, and composer-background evidence on the cursor row or an adjacent row.
- [x] Remove the bottom-band geometry requirement from Codex active-composer detection.
- [x] Clear impossible tracked rows on resize and erase-display operations.

### Task 3: Verify and Commit

**Files:**
- Modify: `workshop/issues/000142-codex-agent-pane-return-should-use-submit-rewrite.md`
- Modify: `workshop/plans/000142-codex-agent-pane-return-should-use-submit-rewrite-plan.md`

- [x] Run the targeted wrapcmd tests.
- [x] Run `go test ./cmd/internal/wrapcmd -count=1`.
- [x] Run `git diff --check`.
- [x] Tick completed issue and plan steps.
- [x] Commit with issue reference after verification.

## Revisions

### 2026-08-16

- Replaced the logical-bottom inference design with cursor-anchored detection after operator review. The implementation now avoids deriving Codex's logical screen geometry and instead treats a visible cursor near observed composer paint as the positive composer signal; cursor-only detection remains out of scope.
