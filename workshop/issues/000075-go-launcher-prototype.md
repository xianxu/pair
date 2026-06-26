---
id: 000075
status: open
deps: [000073, 000074]
github_issue:
created: 2026-06-26
updated: 2026-06-26
estimate_hours:
---

# pair Go launcher prototype

## Problem

The launcher is the largest remaining shell surface and the most important packaging target, but it owns many behavioral edges: session picker, tag normalization, resume/continue/rename, zellij lifecycle, quit/restart markers, data-dir migrations, orphan cleanup, cmux title ownership, and dev rebuild behavior. Porting it must not break normal Pair usage.

## Spec

Prototype the launcher core in Go behind an alternate or guarded path. The prototype should implement a coherent vertical slice of `bin/pair` behavior while the shell launcher remains the public default.

Expected scope should be chosen from the inventory, but a good first vertical slice is:

- argv parsing and help/list dispatch;
- tag normalization;
- data-dir resolution;
- session listing/picker data model using injected zellij/fzf seams;
- create/attach decision model without launching real zellij in unit tests.

Process-level tests should use fake `zellij`, `fzf`, `nvim`, and filesystem state where needed. Business decisions should be pure functions with thin command-exec seams (`ARCH-PURE`).

## Done when

- [ ] A guarded Go launcher path can exercise a documented subset of launcher behavior.
- [ ] Existing `bin/pair` remains the default public launcher.
- [ ] Tests cover the ported decision logic and at least one process-level fake for external commands.
- [ ] Any behavior not yet ported fails explicitly rather than silently diverging.
- [ ] Pair remains usable after merge through the existing public command.

## Plan

- [ ] Select the launcher slice from #73.
- [ ] Extract pure decision models and tests.
- [ ] Add fake-command process tests for the selected slice.
- [ ] Implement the guarded Go path.
- [ ] Document remaining shell-owned launcher behavior.

## Log

### 2026-06-26

Created from #72. This issue is deliberately a prototype/vertical-slice milestone, not the public switch.
