# Boundary Review — pair#141 (whole-issue close)

| field | value |
|-------|-------|
| issue | 141 — Alt+n restart should recover live Codex session |
| repo | pair |
| issue file | workshop/issues/000141-alt-n-restart-should-recover-saved-session-from-ledger.md |
| boundary | whole-issue close |
| window | f4d7ab68296c6e8f1fa6d22a223c0322f69100fb..HEAD |
| command | sdlc close --issue 141 |
| reviewer | codex |
| timestamp | 2026-08-16T22:51:20-07:00 |
| verdict | FIX-THEN-SHIP |

## Verdict

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The review found no Critical blockers. The implementation delivered the core
behavior: `pair restart` captures a live Codex session id before kill, serializes
it through the restart marker, and restart re-entry prefers it when composing
`codex resume <id>`.

## Findings

### Important

- `cmd/internal/launcher/osruntime.go` duplicated `processChildren`,
  `descendantPIDs`, and `lsofNames` that already existed in `slugcmd`
  (`ARCH-DRY`). Fix: extract the process-tree / `lsof` helpers behind a shared
  internal helper so slug and launcher do not drift.
- `cmd/internal/launcher/osruntime.go` had no direct OSRuntime test for
  `agent-pid-<tag>` + descendant `lsof` to `session_id` recovery. Fix: add a
  temp-`PATH` fake `ps`/`lsof` test similar to slugcmd's live transcript test.

### Minor

- `cmd/internal/launcher/restart.go` still said no new Runtime seam was added,
  but `LiveAgentSessionID` had been added. Fix: update the comment.

## Resolution

- Moved shared process tree and `lsof` helpers into `cmd/internal/procutil`.
- Updated launcher and slugcmd to reuse `procutil` helpers.
- Added direct `OSRuntime.LiveAgentSessionID` coverage with fake `ps`/`lsof`.
- Updated the stale `runRestart` comment.
- Added a lesson for shared OS command helper seams.

## Verification

- `go test ./cmd/internal/launcher ./cmd/internal/transcript ./cmd/internal/slugcmd ./cmd/internal/procutil -count=1`
- `go test ./... -count=1`
- `git diff --check`
