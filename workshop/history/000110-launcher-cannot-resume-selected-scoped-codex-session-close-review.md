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
| timestamp | 2026-07-07T22:00:00-07:00 |
| verdict | SHIP |

## Verdict

```verdict
verdict: SHIP
confidence: high
```

## Summary

The diff satisfies pair#110's Spec/Plan: Codex nested rollout files are
recognized through the shared transcript resolver, Codex
`codex [OPTIONS] resume <sid>` is detected and stripped in the saved-config path,
and the atlas consumer that previously described args[0..1]-only behavior is
updated. No blocking findings were reported.

## Findings

- Critical: none.
- Important: none.
- Minor: `cmd/internal/launcher/createlogic.go` still says Codex uses the
  "leading" `resume <id>` subcommand. Non-blocking wording only.

## Strengths

- `cmd/internal/launcher/osruntime.go` reuses `transcript.Resolve` for Codex
  session existence (`ARCH-DRY`, `ARCH-PURE`).
- `cmd/internal/launcher/agentargs.go` centralizes Codex resume command parsing
  and reuses it for explicit-resume extraction and persisted-arg stripping.
- `cmd/internal/launcher/createflow_test.go` covers the polluted saved-config
  shape end to end through picker composition.
- `cmd/internal/launcher/osruntime_test.go` pins nested rollout discovery with
  a real filesystem regression test.

## Verification

- `go test ./cmd/internal/launcher -count=1`
- `go test ./cmd/internal/transcript -count=1`
- `go test ./...`
- `git diff --check 9213a7329049a81cb457a20d6dcad03fd0dfad23..HEAD`

## Architecture

- `ARCH-DRY`: pass. No remaining changed-code duplicate of the Codex rollout
  lookup or resume parser.
- `ARCH-PURE`: pass. Arg grammar stays pure; filesystem lookup remains at
  `OSRuntime`.
- `ARCH-PURPOSE`: pass. The fix covers both stated root causes and the relevant
  atlas consumer.

## Plan Revisions

None.
