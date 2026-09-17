---
id: 000276
status: open
deps: [pair#256]
github_issue:
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
---

# Surface couch-tagged agents that have no thread record

## Problem

`ReasonUnrecordedChild` ("running but unrecorded") exists in the classifier's
vocabulary, but it is reached **per record** — so an agent running with *no*
record at all is invisible rather than reported. `pair#272` named the case and
its `## Done when` carries the bullet; split out of `pair#256` on 2026-09-16 when
that plan was re-cut, because this is an **additive discovery feature**, not a
lifecycle-authority fix, and it was the largest remaining piece of scope that did
not repair a broken state.

Live fixtures, running since 2026-08-30 and still present after the operator's
2026-09-16 cleanup:

| Session | `pair wrap` | Conversation | Tag |
|---|---|---|---|
| `📁parley-couch` | 84488 | `claude --session-id 44e6ec1b…` | `couch-797c45e8e649a9bb` |
| `📁parley-couch-2` | 1130 | `claude --session-id 9a0a197f…` | `couch-2583ed61c0ab6ebe` |

Both hold intact conversations couch cannot see. Also present: `87464`
`📁brain-couch-2`, a zellij server whose `pair wrap` child is gone — the
agent-died-but-shell-survived shape, which any listing here must not present as
reattachable work.

## Spec

Three things this needs, none of which the current code has:

- **A `repos/*` enumeration.** `DetachedSessionResolver`'s doc
  (`couchcore/artifactcollision.go:245-249`) states the checker deliberately
  lacks one and takes addresses instead, because the session-name index is per
  repo scope. Reporting a session that maps to no address requires walking the
  scope directories — a new capability with its own seam and stateful fake
  (ARCH-MOCK), not a tweak to the existing resolver.
- **A named field on `ThreadProjectionInput`.** Its doc
  (`couchcore/actionableinventory.go:169-177`) exists so an addition is visible
  at every construction site rather than silently omitted; add the unrecorded set
  there and let `FromSnapshot` carry it.
- **A row with no actions, this issue.** There is no record to archive, name or
  describe. Report-only: it says a couch-tagged agent is running that couch does
  not track, and names its session. **Adoption** ("make this a thread again") and
  **forced stop** are separable follow-ons, deliberately out of scope here —
  adoption in particular needs to mint a record and bind a native session id,
  which is its own design.

Depends on `pair#256`, which establishes the session-first classification and the
session-evidence seam this reads.

## Done when

- A couch-tagged zellij session with no thread record appears in the switcher and
  in `couch --list`, naming its session.
- The two `parley` fixtures above are visible; a session whose agent is gone is
  not presented as reattachable work.
- A session that *does* map to a record produces no duplicate row.
- The scope enumeration runs behind a seam with a stateful fake, and the listing
  is a pure projection over it.
- `pair#272`'s "a running couch-tagged agent with no record is visible somewhere"
  is satisfied here; `pair#256` records the transfer.

## Plan

- [ ] Land `pair#256` first.
- [ ] Claim/start-plan; design the enumeration seam and its fake.
- [ ] Implement the projection and the report-only row.
- [ ] Verify against the two live fixtures; atlas; close through SDLC.

## Log

### 2026-09-16

- Split out of `pair#256` during the operator-directed over-engineering audit:
  a new seam, a new fake, a scope enumeration and a projection field, all for a
  report-only row, inside an issue whose purpose is repairing lifecycle
  authority. Additive, separable, and cleanly testable on its own.
