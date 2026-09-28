---
id: 000173
status: open
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
github_issue:
---

# Publish and display the actor description

## Problem

couch's per-actor description is fully built and entirely inert. Three layers,
none connected:

**Nothing writes it.** `Couch.PublishDescription` (`couchcore/couch.go:911`)
writes the sidecar, exposed as `couch --internal publish-description <text>`.
Grep finds **no caller outside couch's own package** — so the sidecar is
populated only if the LLM chooses to run an internal command nothing prompts it
to run. In practice every description is empty or hand-typed.

**Nothing displays it.** `couchtty` — console, status row, switcher menu —
contains **zero** references to `Describe`, `Description`, or `Desc`. The only
mention in the package is the comment at `reserve.go:82` explaining how
untrusted description text is sanitized, documenting a data flow that does not
exist. `StatusActor.Label` is `p.label` (`console.go:902`), the
operator-assigned short name. `Desc` reaches `TreeSummary` (`couch.go:827`) and
the actor summary (`:766`) and so serializes into `couch`'s JSON, but nothing
human-facing renders it.

**Yet it is matched against.** `couch.go:718` substring-matches
`c.Describe(w)` inside tree lookup. couch will resolve an actor by description
text — against a field that is always empty.

Meanwhile the string already exists. `pair-slug` runs at turn-end via
`pair-wrap` — agent-agnostic across claude/codex/agy — deriving a left segment
from the git branch and a `<focus>` right segment from a small model over the
recent transcript, with a KEEP gate and validation, writing a candidate to the
proposed-slug binding. nvim applies it to the draft pane's first line, where the
operator reads it today (e.g. `queue.md implementation`). It is generated every
turn, already sanitized, already non-fatal on failure.

So the work is a wire, not a feature.
