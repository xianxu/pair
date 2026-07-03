---
type: pensive
date: 2026-07-02
topic: multi-milestone SDLC gotchas
mode: thoughts
description: Two workflow gotchas learned closing #99 M3+M4 — sdlc actual is issue-cumulative, and the durable plan must be self-sufficient (not lean on the continuation).
references: [workshop/plans/000099-port-launcher-to-go-plan.md, workshop/lessons.md, workshop/issues/000099-port-launcher-to-go.md]
---

# Pensive: multi-milestone SDLC gotchas

Closing two milestones of #99 back-to-back surfaced two things about the SDLC
machinery that aren't obvious until you hit them on a *multi*-milestone issue.

**`sdlc actual` / `milestone-close --actual` reports issue-cumulative, not the
milestone increment.** The active-time engine windows `<issue-start> → HEAD`, so
at every milestone close it suggests the *running total* for the issue (M3: 4.69h,
M4: 7.21h), and it suggests that number even though it knows `--milestone Mx` was
passed. The per-milestone increment — the actual re-price signal the change-code
estimate judges ask for — is `cumulative_now − cumulative_at_prior_close`, which
you compute yourself (M3 ≈ 2.78h, M4 ≈ 2.52h). Pass the tool's cumulative value to
`--actual` (it's the measured, not-typed number, and the velocity ledger derives
increments from the cumulative series), but state the increment in the close Log so
the re-price is legible. Don't burn a cycle re-deriving "is 1.91h the M2 increment
or cumulative?" each time — it's cumulative; the increment is a subtraction.

**The durable plan is the record of truth; a scope contradiction left in it — even
one you fully understand and explained in the continuation — is a change-code
plan-quality FAILURE.** M4's plan bullet paired "flip the default" with "convert
`bin/pair-shell` to a thin shim," which loops (native → ErrFallbackToShell → shim →
native). I'd flagged exactly this in the M4 *continuation*, but the continuation is
ephemeral; the plan-quality judge reads the *plan* and correctly blocked, because
an agent reading only the plan would walk into the loop. The fix was a plan
`## Revisions` entry (flip-default-only; shim/retirement → M5), then re-run
change-code → INFO. Lesson: when you discover a milestone must narrow/re-sequence
its plan bullet, fix the plan artifact *before* change-code, not just the
continuation — the continuation explains, the plan *commits*.

## Open questions

- Is the `sdlc actual` cumulative-vs-increment behavior intended, or worth an
  upstream ask (ariadne) for a `--since-milestone` window, the way ariadne#162
  covers the milestone-close window bug? For now the subtraction is fine.
