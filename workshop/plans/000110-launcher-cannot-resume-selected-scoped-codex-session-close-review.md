# Boundary Review — pair#110 (whole-issue close)

| field | value |
|-------|-------|
| issue | 110 — launcher cannot resume selected scoped codex session |
| repo | pair |
| issue file | workshop/issues/000110-launcher-cannot-resume-selected-scoped-codex-session.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9213a7329049a81cb457a20d6dcad03fd0dfad23..HEAD |
| command | sdlc close --issue 110 |
| reviewer | codex |
| timestamp | 2026-07-07T21:21:15-07:00 |
| verdict | SHIP |

## Verdict

```verdict
verdict: SHIP
confidence: high
```

## Summary

The diff delivers the issue's stated purpose: Codex session existence now reuses
the shared transcript resolver, which finds the nested
`~/.codex/sessions/YYYY/MM/DD/rollout-*<sid>*.jsonl` shape, and the launcher
resume picker already consumes `AgentSessionExists` to expose the saved-session
option. No blocking or non-blocking findings were reported.

## Findings

- Critical: none.
- Important: none.
- Minor: none.

## Strengths

- `cmd/internal/launcher/osruntime.go` reuses `transcript.Resolve` instead of
  adding a second Codex path convention (`ARCH-DRY`).
- `cmd/internal/launcher/osruntime_test.go` pins the real nested rollout path
  that caused the regression.
- Existing picker tests confirm resumable sessions produce `use saved params +
  session` and compose Codex `resume <sid>` correctly.

## Verification

- `go test ./cmd/internal/launcher -count=1`
- `go test ./...`
- `git diff --check 9213a7329049a81cb457a20d6dcad03fd0dfad23..HEAD`

## Architecture

- `ARCH-DRY`: pass. Codex path discovery is centralized through
  `cmd/internal/transcript`.
- `ARCH-PURE`: pass. The IO remains in the runtime boundary; picker logic stays
  behind the existing `AgentSessionExists` seam.
- `ARCH-PURPOSE`: pass. The fix addresses the stated launcher resume failure
  without adding a new ledger or picker path.

## Plan Revisions

None.
