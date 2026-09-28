---
id: 000304
status: open
created: 2026-09-21
updated: 2026-09-21
estimate_hours:
github_issue:
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
