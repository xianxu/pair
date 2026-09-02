---
id: 000173
status: open
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
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

## Spec

### 1. Publish

When `pair-slug` has a validated slug **and** it is running inside a couch
actor, publish it as that tree's description.

- **Detect via the environment couch already exports**: `COUCH_TREE` is set for
  a couch-hosted session (alongside `COUCH_THREAD_TAG`, `COUCH_STORE_DIR`).
  Unset ⇒ standalone pair ⇒ no couch call at all, and no error.
- **Go through the declared operation** — exec `couch --internal
  publish-description <text>` — rather than writing couch's store file
  directly. A second writer into couch's store from another binary breaks the
  single-authority rule the store is built around, and the operation already
  carries validation and dispatch (`ARCH-DRY`).
- **Inherit pair-slug's failure posture verbatim**: any error logs to
  `$PAIR_SLUG_LOG` when set and exits 0 without writing. A couch publish
  failure must never disturb the agent or the draft — pair-slug is spawned
  backgrounded at turn-end precisely so it cannot.

**Decision to make in the plan: whole slug or focus segment?** The slug is
`<branch>/<focus>`. The switcher row and status chip already carry identity
(tree, name, branch), so the branch half is redundant there and eats scarce
columns; the `<focus>` segment alone is the part that answers "what is this
actor doing". Recommend publishing the focus segment, but confirm against how
it reads in the row before fixing it.

### 2. Display on the status row

The row is one line and chips are per-actor, so every description cannot fit.
The description that matters is the **attached** actor's.

- Give the row's free-text area a priority order: **the latest notice, falling
  back to the active actor's description when no notice is pending.** One slot,
  two sources, explicit precedence — rather than a second region competing for
  the same columns (`ARCH-CONSTRAINTS`: the row is a fixed single line and
  overflow is a truncation decision, not a wrap).
- `RenderStatusRow` already strips control bytes from untrusted label and
  notice text (`reserve.go:82-87`) — reuse that path rather than adding a
  second sanitizer. The hazard it names is real here: a description is
  whatever a child wrote.

### 3. Not in scope

The **switcher** half stays `pair#163` (match descriptions in typeahead, show
them on a description-matched row). That issue currently specifies a search over
a field with no source; it gains `deps: [pair#173]` and becomes buildable once
this lands. Division of labour: this issue owns the source plus the status row,
#163 owns the switcher.

Hover-to-reveal is explicitly out. It needs `?1003` any-motion reporting, which
emits on every cell of pointer movement through couch's input loop — a
different order of volume from the `?1000` click tracking `pair#172` specifies.
Wire the description first; hover was standing in for "I cannot see what these
actors are doing", and may not survive the row and switcher carrying it.

## Done when

- A couch-hosted pair session publishes its description automatically at
  turn-end, for claude and for codex — verified by reading the sidecar (or
  `couch`'s JSON) after a turn, with no manual command run.
- A standalone pair session (`COUCH_TREE` unset) makes no couch call and logs
  no error.
- A failing or missing `couch` binary leaves pair-slug exiting 0 with the draft
  untouched — asserted, since this runs on every turn-end and a regression here
  degrades every session.
- The status row shows the active actor's description when no notice is
  pending, and the notice when one is.
- A description containing `\x1b[2J` cannot clear the operator's screen —
  covered by the existing sanitize path, asserted at the row.
- `pair#163` carries `deps: [pair#173]`.

## Plan

- [ ] Decide whole-slug vs focus segment against a real row render.
- [ ] Publish from `pair-slug` behind a `COUCH_TREE` check, via `couch
      --internal publish-description`, preserving the non-fatal posture.
- [ ] Status row: notice-else-description precedence in the free-text area.
- [ ] Sanitize + truncation tests at the row for untrusted description text.
- [ ] Update `pair#163` with the dependency.

## Log

### 2026-09-02

Found while asking where couch's description is displayed. Answer: nowhere.

Worth recording as evidence rather than argument: this is a precise artifact of
the couch overrun. `pair#149` ("opaque tags and a human naming layer", 17.80h
estimated → 56.46h actual) built the sidecar store, the fallback resolution
order, the untrusted-text sanitization, and the fuzzy matching — and connected
neither end. The ontology was built; the wire in and the wire out were not.
That is `pair#170`'s rescope premise showing up as a measured instance.
