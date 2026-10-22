---
id: 000804
status: open
deps: [pair#803]
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '95aee0a12f5e811e5aaddc241937402b6914cf28' # card fields mirrored from issue-cards; edit via sdlc
---

# Route Lua Zellij reads through the shared seam

## Problem

#803 centralizes bounded Zellij read recovery for Go consumers, but the Neovim
Lua consumers still execute Zellij directly and can receive the same empty or
crossed reply. The remaining exclusion must not become another copied retry loop.

## Spec

Add a thin Pair CLI adapter over #803's shared Go read seam, and route production
Lua reads through it: nvim/workbench_route.lua, nvim/dom_transport.lua,
nvim/pair_poke.lua, and nvim/review/restore_client.lua. Preserve the existing
operation-wide deadlines, cancellation/join, asynchronous restore behavior,
per-attempt diagnostic evidence, and two independently valid agreeing observations.
Pin an allowlisted argv and stdout/stderr/exit contract before implementation.
Mutations and intentionally raw diagnostic probes never gain automatic retries.

The first three callers query list-panes --json --command --state --tab through
zellij_observation.query/zellij_trace.action. Review restore also queries
are-floating-panes-visible --tab-id ID. An empty result is never absence proof;
malformed same-kind data remains a strict refusal. Do not invent a second Lua
retry policy. Regenerate the shipped runtime bundle from its source files.

## Done when

- Every listed Lua production query derives recovery policy from the Go seam.
- Stateful fake schedules cover empty/crossed recovery, exhaustion, cancellation,
  malformed replies, and no mutation replay through the actual Lua adapter.
- Existing confirmation, async restoration and caller deadlines remain pinned.
- The atlas removes the explicit Lua exclusion after qualified integration.

## Plan

- [ ]

## Log

### 2026-10-22

Split from #803 by TL delivery a27b640f-2e13-4c62-b6d8-b2faf5b422e0. Unclaimed
follow-up; no stacked development before the Go seam lands.
