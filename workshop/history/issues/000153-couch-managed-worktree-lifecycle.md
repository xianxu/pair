---
id: 000153
status: punt
deps: [149, 152]
github_issue:
created: 2026-08-25
updated: 2026-09-02
estimate_hours:
---

# couch: managed worktree lifecycle and garbage collection

## Problem

`ariadne#200` can resolve a requested path to an admission policy whose
on-capacity action is `provision-worktree`, but resolution deliberately does not
create paths or manage branches. `#149` consumes that result and must not invent
a checkout path inside its admission guard. Worktree-managed repositories need
an explicit lifecycle owner for allocation, branch tracking, divergence, and
safe garbage collection.

## Spec

Provisioning is a typed response to a full admission key, not a fallback based
on repository name. The lifecycle owner creates a worktree using the owning
repo's declared strategy, returns its canonical path for ordinary TUI start,
and records enough provenance to distinguish couch-managed trees from user-made
ones.

The durable thread preserves immutable `starting_path` and `created_at`, while
its current `working_path` may be rebound only by this lifecycle owner after a
deterministic reprovision succeeds. A cleanup that would strand a retained
thread without a reproducible checkout is refused. Resume can therefore ask
this owner to recreate and rebind a managed tree without falsifying where the
thread began or silently making historical threads unresumable.

Managed missing-path resume is one optimistic transaction. It first asks #152
to preflight the composite thread as uniquely resolved, verified parked, and
profile-valid and captures the thread revision; no worktree is created for a
live, unknown, partially parked, ambiguous, or invalid-profile thread. The
lifecycle owner then reprovisions outside the store lock. Rebinding takes the
lock and compare-and-swaps the captured revision after repeating #152's state
checks and current policy/provenance validation. A concurrent mutation or failed
recheck leaves the new checkout unbound and reports it for conservative cleanup;
only a successful rebind calls #152's unchanged launch phase. Thus provisioning
cannot turn a refusal into a side effect on the durable thread.

Cleanup is conservative and evidence-driven. A managed worktree is never
removed while it has a live or unknown thread, dirty files, unpushed commits,
an unmerged branch, or unresolved divergence. Garbage collection reports those
facts and either performs a recoverable cleanup or delegates complex cases to
an agent skill; it never guesses from age alone. Exact branch naming and merge
policy are designed with the first consuming application repo, not hard-coded
from Pair's local-tool workflow (ARCH-PURPOSE, ARCH-MOCK).

## Done when

- A `provision-worktree` policy result can create an isolated canonical path and
  feed it back through the normal #149 start path.
- Managed provenance survives couch restart and never claims ownership of a
  user-created worktree.
- A missing managed `working_path` is deterministically reprovisioned and
  rebound before invoking #152's unchanged verified resume; cleanup refuses
  when that recovery contract cannot be preserved.
- Cleanup refuses live/unknown, dirty, divergent, unmerged, or unpushed work and
  reports exact evidence.
- A stateful Git fake and real-repository conformance cover create, inspect, and
  cleanup behavior.

## Plan

- [ ] Design the per-repo provisioning declaration and managed provenance.
- [ ] Implement allocation behind a stateful Git/worktree seam.
- [ ] Model cleanup eligibility from measured repository and actor facts.
- [ ] Add conservative garbage collection plus agent-assisted escalation.

## Log

### 2026-08-25

Split from #149 so consuming `ariadne#200`'s normalized policy cannot quietly
grow a second, Pair-specific worktree strategy.

### 2026-09-02

Punted by the couch-lite rescope (`pair#170`). couch narrows to a switcher over
a group of live coding sessions — one LLM stack, one path, no cluster transport,
no advisor, no managed worktrees. This is deferral, not rejection: the spec above
stands and the scope event in `workshop/projects/couch.md` records why it stopped
being the next thing to build.
