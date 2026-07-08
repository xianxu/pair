# Boundary Review — pair#103 (whole-issue close)

| field | value |
|-------|-------|
| issue | 103 — consolidate the offset→(row,col) helper family in nvim/review |
| repo | pair |
| issue file | workshop/issues/000103-consolidate-the-offset-row-col-helper-family-in-nvim-review.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1c4bd640377d8bb57f773fd27ed9cb4022f3ae24..HEAD |
| command | sdlc close --issue 103 --actual 0.27 --no-atlas --verified '<evidence>' |
| reviewer | codex |
| timestamp | 2026-07-07T22:26:44-07:00 |
| verdict | SHIP |

## Verdict

```verdict
verdict: SHIP
confidence: high
```

The diff fulfills #103's stated refactor: duplicated offset/position helpers are
consolidated into `nvim/review/reconstruct.lua`, existing consumers derive from
that source, and the requested Lua/review test suites pass. No blocking or
non-blocking findings were reported.

## Strengths

- `nvim/review/reconstruct.lua` now owns `line_starts`, `pos_of`, and
  `occurrence_at` as pure helpers.
- `nvim/review/markers.lua` derives both `parse_markers` and `spans_multiline`
  line starts/positions from `reconstruct`, removing the two local binary-search
  twins.
- `nvim/review/reconcile.lua` derives hunk offsets from
  `reconstruct.line_starts` and reuses the shared occurrence counter.

## Findings

- Critical: none.
- Important: none.
- Minor: none.

## Verification

- `make test-lua` passed.
- `make test-review` passed.
- `git diff --check` passed.
- Added helper coverage in `nvim/review/reconstruct_test.lua`; existing
  marker/reconcile tests exercise the moved behavior.

## Architecture

- `ARCH-DRY`: pass. The duplicated offset-to-position implementations are
  consolidated.
- `ARCH-PURE`: pass. The helper home remains pure Lua with no vim/IO dependency.
- `ARCH-PURPOSE`: pass. The listed consumers were repointed; the shadow sweep
  found no remaining second implementation of the same binary-search/line-start
  math outside `reconstruct.lua`.

## Plan Revisions

None.
