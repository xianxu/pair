---
id: 000163
status: done
created: 2026-09-01
updated: 2026-09-30
estimate_hours:
github_issue:
started: 2026-09-30T12:34:02-07:00
actual_hours: 0.18
tracker:
    version: 1
    completion:
        token: close-861e36cdde66
        repository: github.com/xianxu/pair
        reviewed_head: 5e585cac82a1fdb23fca060dca3829f0a272a7f3
        evidence_commit: 7e093879334e43f8ffdfadf2410ad938cab29ee2
        landed_commit: 1103a6b773509cc51ec920a889c1267ce308c873
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
