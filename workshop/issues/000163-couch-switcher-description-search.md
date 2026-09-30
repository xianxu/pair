---
id: 000163
status: working
deps: [pair#173]
github_issue:
created: 2026-09-01
updated: 2026-09-30
estimate_hours:
card_mirror: '06e2ccb5a65cbaf666dad4e8605edbf457f3fdee' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-30T12:34:02-07:00
---

# Match and show actor descriptions in Couch switcher

## Problem

**Blocked on `pair#173`: the description has no source today.** Nothing outside
couch's own package calls `publish-description`, so every description is empty
or hand-typed, and this issue as written would ship a typeahead over empty
strings. #173 wires `pair-slug`'s turn-end output into the sidecar and takes the
status-row display; this issue keeps the switcher half.

Couch's switcher typeahead does not search an actor's assigned description, so
users cannot find an actor using the descriptive context they gave it. The
switcher also omits that description when it is the reason a result matched,
making the match difficult to understand.

## Spec

- Both default and focus switcher views match the sanitized displayed description
  (published summary takes precedence over operator description), case-insensitively.
- Preserve exact opaque-tag precedence and explicit repo:N slot selection; CLI
  references retain their current matching fields. Reuse the shared pure matcher
  with an optional description field populated by the menu (ARCH-DRY).
- When an actor matches by description, show the description directly below
  the actor's primary line in the switcher result.
- Preserve the existing presentation for actors without descriptions.

## Done when

- Typing text found only in the displayed description returns the ordinary or
  numbered-slot row in both default and focus views, case-insensitively.
- Default-view description matches render the description beneath the row;
  focus results keep their existing inline description. Exact tags/slot references
  retain precedence and focus still excludes parked or undescribed rows.
- Actors with no description continue to search and render as before.
- Automated tests cover description matching, display, and the no-description
  case.

## Plan

- [ ] Add failing tests for description typeahead matching and result display.
- [ ] Include assigned descriptions in the switcher's matching data.
- [ ] Render matched descriptions beneath actor rows.
- [ ] Verify existing actor matching and description-free rows do not regress.

## Log

### 2026-09-01

Captured during Couch dogfood testing. Intended layout: when a description is
assigned and matches the query, display it immediately below the actor's main
line.

## Revisions

### 2026-09-30 — Requested in both switcher views

Descriptions now have a source and focus view already renders them inline, so
the old source-blocker note is historical. User requested description matching
in both default and focus views. Match the existing sanitized DisplaySummary
projection, preserve exact-reference behavior, and show matching descriptions
in default results using existing row rendering/extents. This fits the quick
flow; no durable plan or new IO is needed.
