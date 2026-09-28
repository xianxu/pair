---
id: '000170'
status: done
started: 2026-09-02T11:03:39-07:00
created: 2026-09-02
updated: 2026-09-13
estimate_hours: 10.69
actual_hours: 11.01
---

# Rescope couch to couch-lite

## Problem

couch has consumed ~172 measured hours across twelve closed issues (#145, #146,
#149, #151, #152, #154, #155, #156, #158, #159, #161, #167), of which ~139 went
into the switcher and identity layers. The operator has driven it for a day or
two and it is currently generating papercuts (#163, #164, #165, #168, #169)
faster than value. It replaced the substrate — tabs became a switcher — without
yet adding a capability.

**Root cause: no razor-clear view of what couch is.** That gap got filled by
generality. `cmd/couch` + `cmd/internal/couchcore` is ~22k lines carrying
admission control, supervisor leases, start grants, park transactions, a
write-ahead journal and fail-closed projections — distributed-systems machinery
defending one operator on one host. The oscillation showed up as repeated
redesign of threading structure and agent selection. pair avoided the same
ambiguity by exposing CLI options and letting usage reveal the right
configuration; couch decided in advance and encoded the decision in types, so
every later discovery meant changing the ontology instead of adding a flag.

The estimate ratios separate cleanly along that seam:

| shape unknown (deciding while building) | shape known (building a behavior) |
| --- | --- |
| #146 0.28x, #149 0.32x, #154 0.27x, #151 0.38x | #156 2.27x, #158 1.72x, #159 1.19x, #167 1.95x |

Secondary: ephemeral runtime state is harder to get right (timing-dependent)
and more opaque to a coding agent than repo state. Failures like #169 are
transient — by the time anyone inspects, the subprocess error is gone.
