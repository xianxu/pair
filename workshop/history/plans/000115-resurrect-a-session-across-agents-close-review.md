# Boundary Review - pair#115

| field | value |
|-------|-------|
| issue | 115 - Switch the agent driving existing work |
| repo | pair |
| boundary | whole-issue close |
| window | 2e232a2..HEAD |
| reviewer | codex |
| timestamp | 2026-08-16T19:39:45-07:00 |
| verdict | FIX-THEN-SHIP |

## Verdict

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The first close review returned `REWORK`; the second close review accepted the
scope as `FIX-THEN-SHIP` after the critical correctness and tracker findings
were addressed.

## Findings

- Critical fixed before the second close review: an agent-mismatched
  `config-<tag>-<agent>.json` could still reach the restart picker. Fixed by
  validating the embedded config agent before treating the sidecar as usable,
  warning, and falling through to ledger/defaults.
- Critical fixed before the second close review: #115 still carried the
  abandoned live-handoff `M5` as an unchecked close item. Fixed by revising #115
  to the revived M1/M2 close scope and creating #135 for the deferred live
  cross-agent handoff.
- Important fixed under `FIX-THEN-SHIP`: stale saved native-session IDs were
  silently downgraded to fresh sessions. Fixed by warning in `runConfigPicker`
  when a saved `session_id` is not resumable, with an integrated create-flow
  regression.

## Verification

- `go test ./cmd/internal/launcher -run TestRunLaunchIgnoresMismatchedTagConfigWithWarning -count=1`
- `go test ./cmd/internal/launcher -run 'TestRunLaunch(TagRestartPickerWarnsWhenSavedSessionIsStale|IgnoresMismatchedTagConfigWithWarning)' -count=1`
- `go test ./cmd/internal/launcher -count=1`
- `go test ./... -count=1`
- `sdlc issue validate --issue 115`
- `sdlc issue validate --issue 135`
- `git diff --check`

## Close Trailers

Review-Verdict: FIX-THEN-SHIP
Review-Window: 2e232a2..HEAD
