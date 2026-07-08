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
| timestamp | 2026-07-07T21:55:00-07:00 |
| verdict | FIX-THEN-SHIP |

## Verdict

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

## Summary

The code change delivers the issue behavior: Codex nested rollout session files
are recognized through the shared transcript resolver, and
`codex [OPTIONS] resume <sid>` is detected and stripped for persisted config
args. The review found one docs-gate miss: `atlas/architecture.md` still
documented the old args[0..1]-only Codex strip rule.

## Findings

- Critical: none.
- Important: `atlas/architecture.md` needed to document that Pair strips
  `resume <X>` after recognized Codex global options, then prepends the
  canonical `resume <session_id>`.
- Minor: none.

## Resolution

Updated `atlas/architecture.md` to describe `codex [OPTIONS] resume <id>` and
the corresponding strip-then-prepend behavior.

## Verification

- `go test ./cmd/internal/launcher -count=1`
- `go test ./cmd/internal/transcript -count=1`
- `go test ./...`
- `git diff --check 9213a7329049a81cb457a20d6dcad03fd0dfad23..HEAD`

## Architecture

- `ARCH-DRY`: pass in code; Codex path discovery now uses the shared transcript
  resolver and Codex resume parsing is shared by explicit-resume detection and
  persisted-arg stripping.
- `ARCH-PURE`: pass; argv grammar logic stays pure and filesystem checks remain
  at the runtime boundary.
- `ARCH-PURPOSE`: pass after the atlas update; the documented consumer now
  matches the delivered resume behavior.

## Plan Revisions

None.
