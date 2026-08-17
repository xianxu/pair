---
id: 000142
status: working
deps: []
github_issue:
created: 2026-08-16
updated: 2026-08-16
estimate_hours: 0.57
started: 2026-08-16T23:04:16-07:00
---

# Codex agent-pane Return should use submit rewrite

## Problem

In the Codex agent pane, plain Return currently submits the composer text. Pair's intended convention is that plain Return acts like `/r` by inserting a newline in the active agent composer, while Alt+Return submits.

## Spec

When Codex is showing its active composer, `pair-wrap` must rewrite plain Return to the Codex newline byte. Detection should anchor on Codex exposing a visible cursor on or next to a recently painted composer surface, not on Pair's outer pane geometry or an inferred logical screen bottom. A visible terminal cursor alone is not sufficient evidence because non-composer terminal UIs can also show cursors. The existing overlay/picker exception remains higher priority: Return must still confirm Pair overlays with a bare CR reliably.

## Done when

- Plain Return in an active Codex composer rewrites to LF for the observed cursor-visible composer state.
- Plain Return still sends bare CR when the Codex composer is not positively detected.
- Existing overlay behavior still beats the composer rewrite.
- Targeted wrapcmd tests pass.

## Plan

- [x] Add a failing Codex composer test for the observed mismatch: outer winsize rows=38, Codex-painted composer at rows 19-21, visible cursor at row 20.
- [x] Extend Codex composer detection to use visible cursor plus nearby composer-background evidence independent of bottom-row geometry.
- [x] Add/extend a Return rewrite test so the cursor-anchored active composer produces LF instead of bare CR.
- [x] Run targeted and package tests.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.

```estimate
model: estimate-logic-v3.1

item: issue-spec design=0.08 impl=0.02
item: smaller-go-module design=0.15 impl=0.25
total: 0.57
```

## Revisions

### 2026-08-16

- Switched the implementation direction from logical-bottom inference to cursor-anchored Codex composer detection after operator review. The new rule requires a visible cursor tied to recent composer-surface evidence and avoids deriving a second screen height. Cursor visibility alone remains rejected because ordinary terminal UIs often show a cursor outside composer state.

## Log

### 2026-08-16

- Claimed issue and entered planning with `sdlc start-plan`.
- Plan-quality gate passed on round 2 after adding non-goals and function-level test strategy.
- Estimate derived via `estimate-logic-v3.1` against the stale repo-local calibration source.
- Root-cause evidence: `adapt-pair.jsonl` recorded `plain Enter -> bare CR (codex composer inactive)` at 2026-08-16T22:35:06-07:00.
- Scrollback at the same offset showed Codex painting an active composer on rows 19-21 with a default status row at 22 and visible cursor at 20;3, while `wrap-events-pair.jsonl` only had an outer winsize of 38 rows until 22:54:41. The strict outer-bottom-band detector therefore missed Codex's logical-screen composer.
- `sdlc change-code --issue 142` passed plan-quality and estimate-quality, then created branch `000142-codex-agent-pane-return-should-use-submit-rewrite`.
- Added failing tests for the observed rows 19-21 cursor-anchored Codex composer and the Return rewrite path; confirmed they failed before production changes.
- Implemented cursor-plus-composer-surface detection: composer rows are tracked independent of bottom geometry, cursor-only states remain inactive, and paint away from the cursor remains inactive.
- Updated Claude/Agy follow-up issues and the harness bring-up atlas page to require each agent's native composer-availability signal instead of copying Codex's cursor/paint heuristic.
- Verified with `go test ./cmd/internal/wrapcmd -run 'TestCodexComposerTracker|TestEmitPlainCR' -count=1`, `go test ./cmd/internal/wrapcmd -count=1`, `go test ./... -count=1`, and `git diff --check`.
- Close review returned REWORK for a sparse false positive: one nearby composer-painted row plus a far painted row counted active. Fixed by counting only rows in the cursor neighborhood and added sparse-paint plus unterminated-CSI regressions.
