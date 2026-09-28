---
id: 000336
status: open
deps: []
github_issue:
created: 2026-09-28
updated: 2026-09-28
estimate_hours:
card_mirror: '88cc677688cca3b62f136ae8a4d38dba4216aef3' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch slot shells: home alias to checkout root

## Problem

Shells started within a Couch slot can navigate away from its checkout. The
operator needs a short command to return to that slot's worktree checkout root,
especially from the right pane, without remembering the slot's filesystem path.

## Spec

Provide a shell alias named `home` in all interactive shells started within a
Couch slot, including the right-pane shell and newly opened shell tabs. Running
`home` changes the current shell's working directory to that slot's worktree
checkout root, regardless of its current directory.

Resolve the destination from the owning slot's checkout identity, not the
current directory, the user's home directory, or another slot's checkout.
Handle checkout paths containing spaces or shell metacharacters safely. Keep
the alias scoped to slot shells; do not change `$HOME` or global shell startup
files. If the checkout is unavailable, report a normal directory-change error
without silently selecting another destination.

During design, enumerate supported shells and startup paths so coverage is not
limited to the initial right-pane shell. Identify how existing user definitions
of `home` are handled before implementation.

## Done when

- `home` returns an interactive shell to its owning slot's checkout root from
  a subdirectory or an unrelated directory.
- The alias is available in the initial right-pane shell and subsequent slot
  shells/tabs across supported startup paths.
- Primary and numbered slots resolve independently to their own checkout roots;
  paths with spaces/metacharacters work without unintended shell evaluation.
- Missing destinations fail visibly without changing to another directory;
  `$HOME` and shells outside Couch slots remain unchanged.
- Tests cover shell startup integration and actual directory changes; usage and
  supported-shell behavior are documented.

## Plan

- [ ] Identify slot-root authority and enumerate interactive shell startup paths.
- [ ] Add slot-scoped alias initialization and regression coverage.
- [ ] Smoke-test `home` in primary/numbered slots and new right-pane tabs.

## Log

### 2026-09-28

Operator requested that every shell started in a Couch slot, such as the right
pane, have a `home` alias returning to that slot's worktree checkout root.
Ticket only; no implementation requested in this turn.
