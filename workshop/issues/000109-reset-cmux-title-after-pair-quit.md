---
id: 000109
status: working
deps: []
github_issue:
created: 2026-07-07
updated: 2026-07-07
estimate_hours: 0.25
started: 2026-07-07T17:48:57-07:00
---

# Reset cmux title after pair quit

## Problem

After exiting pair from inside cmux, the cmux workspace title sometimes remains
on the pair session title instead of resetting to the shell cwd. The native
launcher cleanup is supposed to reset the title when this pair owns the cmux
workspace.

## Spec

- Quit cleanup must recognize both cmux owner record formats:
  - legacy launcher format: `tag`
  - title-poller format: `tag<TAB>public-session`
- Keep the existing cleanup flow: only reset the cmux workspace title when this
  pair owns the workspace, then clear the owner record (ARCH-DRY, ARCH-PURE).
- Do not loosen ownership to "any cmux owner file exists": a foreign live pair
  must still prevent this session from resetting a shared workspace title.
- The reset title remains the current launcher behavior: the basename of the
  shell cwd, with the existing fallback for empty/root paths.

## Done when

- A regression test proves `PairOwnsCmuxWorkspace` returns true for the
  title-poller two-field owner format.
- `pair quit` cleanup can reset cmux titles after the title poller has updated
  the owner file.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
the stale repo-local calibration source reported by `sdlc start-plan`.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.05 impl=0.15
design-buffer: 0.05
total: 0.25
```

## Plan

- [ ] Add a failing OSRuntime cmux ownership test for a two-field
      `tag<TAB>public-session` owner file.
- [ ] Update the launcher ownership check to compare the parsed owner tag, while
      preserving legacy single-field support.
- [ ] Run focused launcher tests and formatting checks.

## Log

### 2026-07-07
- Root cause: `cmd/internal/titlepoller` writes cmux owner records as
  `tag<TAB>public-session`, but `cmd/internal/launcher.OSRuntime`
  `PairOwnsCmuxWorkspace` only accepted `strings.TrimSpace(raw) == tag`. Once
  the poller touched the owner file, quit cleanup skipped the cmux reset.
