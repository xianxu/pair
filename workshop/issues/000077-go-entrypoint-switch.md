---
id: 000077
status: open
deps: [000074, 000075, 000076]
github_issue:
created: 2026-06-26
updated: 2026-06-26
estimate_hours:
---

# pair Go entrypoint switch

## Problem

At some point the public `pair` command must become Go-owned. That switch is the riskiest packaging step because every normal session starts there.

## Spec

Switch the public `pair` entrypoint to the Go dispatcher/launcher after the dispatcher, helper routes, and launcher prototype have enough coverage. Preserve a compatibility escape hatch during the transition, such as `pair-legacy`, an env-selected fallback, or a retained script path chosen by the preceding issues.

The switch must include a rollback story and a dogfood verification path. If the installed binary is stale, the diagnostics should point to `make install` or `pair-dev` consistently with existing lessons.

## Done when

- [ ] Running `pair` uses the Go entrypoint by default.
- [ ] A documented fallback or rollback path exists for one migration window.
- [ ] `pair-dev` still rebuilds and launches the working tree behavior.
- [ ] Existing create, attach, resume, continue, rename/list, quit, and restart flows are verified or explicitly out of scope with a fallback.
- [ ] Pair remains usable after merge; no keybinding workflow regresses.

## Plan

- [ ] Confirm prerequisites from earlier Go migration issues.
- [ ] Add or update installation wiring.
- [ ] Preserve fallback behavior.
- [ ] Run process-level launcher tests and live dogfood checks.
- [ ] Update README/atlas packaging notes.

## Log

### 2026-06-26

Created from #72 as the public switch milestone. This should not be claimed until the earlier dispatcher/helper/launcher milestones have landed.
