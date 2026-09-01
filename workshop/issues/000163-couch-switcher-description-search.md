---
id: 000163
status: open
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-01
estimate_hours:
---

# Match and show actor descriptions in Couch switcher

## Problem

Couch's switcher typeahead does not search an actor's assigned description, so
users cannot find an actor using the descriptive context they gave it. The
switcher also omits that description when it is the reason a result matched,
making the match difficult to understand.

## Spec

- Include a non-empty actor description in switcher typeahead matching.
- When an actor matches by description, show the description directly below
  the actor's primary line in the switcher result.
- Preserve the existing presentation for actors without descriptions.

## Done when

- Typing text found only in an actor's description returns that actor.
- A description-matched result renders the matching description beneath the
  actor line.
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
