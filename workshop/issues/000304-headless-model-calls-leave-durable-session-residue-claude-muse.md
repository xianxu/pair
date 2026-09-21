---
id: 000304
status: open
deps: []
github_issue:
created: 2026-09-21
updated: 2026-09-21
estimate_hours:
---

# headless model calls leave durable session residue (claude, muse)

## Problem

Found by the #300 M4 boundary review (round 11, advisory): the family
`headless-call-leaves-durable-residue` was fixed for qoder (BR-44 added
`--no-session-persistence` to `runQoder`), but its siblings still persist a
transcript per headless call:

- `runClaude` (`cmd/internal/model/model.go`) — `claude --help` lists
  `--no-session-persistence`; `~/.claude/projects` holds residue project dirs
  from `-private-tmp` cwds (measured on this machine).
- `runMuse` — `muse exec --help` lists `--no-session-log`.
- `runCodexCLI` already passes `--ephemeral`, so it is the model row.

The slug/changelog calls fire at every turn end, so the residue grows per
turn with no owner and no sweep. The gap predates #300 and is out of its scope.

## Spec

- Every headless runner in `model.go` passes its agent's no-persistence flag,
  or carries a comment naming why it cannot.
- One table test over the agents `Run` dispatches pins each runner's argv
  (the `wantArgs` pattern from `TestRunQoderDispatchesToQoderCLI`).

## Done when

- `runClaude` passes claude's no-persistence flag (`--no-session-persistence`)
  and a fake-binary dispatch test pins its argv.
- `runMuse` passes muse's no-persistence flag (`--no-session-log`) and a
  fake-binary dispatch test pins its argv (or the comment names why it can't).
- Live spot-check: one headless call per agent leaves no new file under
  `~/.claude/projects` / the muse session dir; evidence in `## Log`.
- Full `go test ./...` green.

## Plan

- [ ] Table-drive the dispatch tests across every headless runner and pin each no-persistence flag.

## Log

### 2026-09-21
- 2026-09-21: created — carried from the #300 M4 boundary review (round 11) as an advisory follow-up; not claimed, not scoped.
