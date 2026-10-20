---
id: 000740
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'f44fda469a137f0cb0ccd459af17b2371f625b51' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: global park conformance on a disposable fleet (park all, restart, resume; nothing lost)

## Problem

The operator's shutdown procedure (10-20 drain redesign) ends with a global park before restarting the host. Global
park is rarely used and untested on a full fleet, so the procedure rests on a path nobody has proven.

## Spec

A conformance check, on a disposable Couch (own state folder, socket namespace and throwaway slots; never the live
fleet): park every slot through the operator's global-park action, then restart Couch and resume them. Verify each
slot's conversation, branch and checkout come back, nothing is lost or duplicated, and parked state survives a
supervisor restart. Fix what it finds.

## Done when

- The check passes on the disposable fleet and runs as a named opt-in target.
- Its findings, if any, are fixed or filed.

## Plan

- [ ]

## Log

### 2026-10-20
