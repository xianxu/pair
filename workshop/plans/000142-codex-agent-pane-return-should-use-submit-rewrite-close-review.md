# Boundary Review — pair#142 (whole-issue close)

| field | value |
|-------|-------|
| issue | 142 — Codex agent-pane Return should use submit rewrite |
| repo | pair |
| issue file | `workshop/issues/000142-codex-agent-pane-return-should-use-submit-rewrite.md` |
| boundary | whole-issue close |
| window | `f4d7ab68296c6e8f1fa6d22a223c0322f69100fb..HEAD` |
| command | `sdlc close --issue 142` |
| reviewer | codex |
| timestamp | 2026-08-16T23:25:49-07:00 |
| verdict | REWORK |

## Findings

### Critical

- `cmd/internal/wrapcmd/codex_composer.go`: `codexComposerState.active` counted
  all painted rows but only required one row near the cursor. A state such as
  painted rows `{10, 20}` with cursor row `20` could return active without a
  two-row composer surface near the cursor. Fix by counting only rows in the
  cursor neighborhood or requiring a contiguous nearby surface. `ARCH-PURPOSE`.

### Important

- `workshop/plans/000142-codex-agent-pane-return-should-use-submit-rewrite-plan.md`:
  the plan promised `codexComposerTracker.feed` coverage over split, partial,
  and malformed ANSI streams, but the implementation only covered split CSI.
  Add malformed or unterminated escape coverage that proves the tracker does not
  panic or invent composer state.

## Resolution

- Added `TestCodexComposerTrackerRejectsSparsePaintNearCursor`.
- Added `TestCodexComposerTrackerRejectsUnterminatedCSIComposerPaint`.
- Changed `codexComposerState.active` to count only composer-painted rows in
  `cursorRow-1..cursorRow+1`.

## Review Notes

- Strengths called out: observed rows 19-21 regression coverage, Return-path LF
  coverage, overlay precedence coverage, and atlas guidance that other agents
  should not copy Codex's heuristic without evidence.
- No minor findings.
