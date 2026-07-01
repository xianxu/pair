---
id: 000093
status: working
deps: [000092]
github_issue:
created: 2026-07-01
updated: 2026-07-01
estimate_hours: 17.4
started: 2026-07-01T14:39:06-07:00
---

# port stateful shell orchestrators to Go

Tracking: #91 (native single binary) — step 3 of 5. Depends on #92.

## Problem

The runtime's stateful orchestration still lives in shell. `bin/pair-shell`
(the launcher) owns the zellij lifecycle, prompt UI, restart/quit cleanup, cmux
ownership, dev rebuild, continuation, rename, config/session migration, and the
title poller; `bin/pair-title.sh`, `bin/pair-scrollback-open`, the
`bin/pair-review-*` helpers, and the clipboard helpers
(`clipboard-to-pane.sh`, `copy-on-select.sh`, `flash-pane.sh`) are all shell
orchestrators. Until this logic moves into Go, the binary can never stop
extracting a shell tree (#94), so this is the load-bearing step toward a native
single binary. `atlas/go-migration-inventory.md` already flags each surface with
a migration priority (P0 launcher, P1 title/scrollback, P2 review helpers).

## Spec

Port the stateful shell orchestrators into Go **one at a time**, each behind a
merge-safe compatibility shim, following the #78 precedent (session watcher
ported to `cmd/pair-session-watch` with `bin/pair-session-watch.sh` retained as
a shim). Reuse the fakeable pure decision core from #75 (`cmd/internal/launcher`)
for the launcher work.

- Each surface is an independent, merge-safe milestone: land it, keep the old
  shell name working as a shim, verify the flow, then move to the next. Pair
  stays usable throughout.
- Keep native assets native: `nvim/*.lua` and `zellij/*.kdl` are NOT ported to
  Go (that boundary is owned by #95). This issue audits their shell-outs but
  leaves them as native assets.
- `ARCH-PURE`: extract pure decision logic (lifecycle state, poller cadence,
  target resolution) into unit-tested packages; keep zellij/nvim/filesystem/cmux
  interaction in thin, process-tested seams.
- `ARCH-DRY`: reuse the existing internal packages (`launcher`, `entrypoint`,
  `sessionwatch`, dispatcher runners) rather than reimplementing.

Ordering rationale: the launcher is the last and largest surface because it owns
the most state; the leaf orchestrators (title poller, scrollback/review/clipboard
openers) are ported first to shrink `bin/pair-shell`'s dependency set before it
is itself replaced.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.4 impl=1.6
item: smaller-go-module design=0.3 impl=1.2
item: smaller-go-module design=0.4 impl=1.6
item: smaller-go-module design=0.2 impl=0.9
item: smaller-go-module design=1.0 impl=6.0
item: milestone-review design=0.0 impl=1.5
item: atlas-docs design=0.2 impl=0.6
total: 17.4
```

Whole-issue estimate across M1–M5 (the five `smaller-go-module` items, in
milestone order; `milestone-review` covers the five boundary reviews;
design hours are weighted 1.6× per the model). The **M5 launcher** item
(design=1.0 impl=6.0 → ~7.6h) is the dominant uncertainty — `bin/pair-shell`
is ~2200 lines; per the Plan's granularity note it may split into its own
ticket, which would re-scope this estimate. M1–M4 (the leaf orchestrators,
~7.4h) are well-understood ports on the verified #78 sessionwatch template.
Durable plan:
`workshop/plans/000093-port-shell-orchestrators-plan.md` (M1 detailed; M2–M5
milestone-level, detailed as reached).

## Done when

- [ ] Each listed shell orchestrator has a Go owner; the shell name survives only
      as a thin re-exec shim (or is removed where no caller needs it).
- [ ] Pure lifecycle/poller/target decision logic is unit-tested; zellij/nvim/
      cmux/filesystem interaction is behind process-tested seams.
- [ ] `nvim/*.lua` / `zellij/*.kdl` remain native assets; their shell-outs are
      audited and repointed to Go owners where applicable.
- [ ] `pair`, `pair-dev`, keybindings, scrollback, changelog, continuation,
      restart/quit, rename, and review flows work after each milestone.

## Plan

Each `Mx` is a merge-safe review boundary closed on its own (`sdlc
milestone-close`); the surfaces are independent enough to port and verify one at
a time.

- [ ] M1 — title poller: port `bin/pair-title.sh` to Go (the explicit #78
      next-candidate), keep the `.sh` name as a shim.
- [ ] M2 — scrollback/changelog openers: port `bin/pair-scrollback-open` (and the
      changelog opener) to Go orchestration; `nvim/*.lua` viewers stay native.
- [ ] M3 — review helpers: port `bin/pair-review-target` / `pair-review-open` /
      `pair-review-readiness` orchestration to Go.
- [ ] M4 — clipboard helpers: port `clipboard-to-pane.sh`, `copy-on-select.sh`,
      `flash-pane.sh` to Go (or fold behind the dispatcher).
- [ ] M5 — launcher / session lifecycle: port `bin/pair-shell`'s orchestration to
      Go on the `cmd/internal/launcher` core, retaining a compatibility shim;
      zellij/nvim stay external.

## Log

### 2026-07-01

Created as step 3 of the native-single-binary tracker (#91) — the load-bearing
port. Surfaces and priorities drawn from `atlas/go-migration-inventory.md`;
milestone ordering puts the leaf orchestrators before the launcher so
`bin/pair-shell` shrinks before it is replaced. Per the tracker's granularity
decision this stays one ticket with per-surface milestones (M1–M5); a milestone
can be split into its own ticket later if its scope grows.
