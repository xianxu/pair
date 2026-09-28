---
id: 000212
status: open
created: 2026-09-07
updated: 2026-09-07
estimate_hours:
github_issue:
---

# spike: Gleam relay to inform 121 M3

## Problem

`#121 M3` is *"implement the Oracle-friendly relay as a dumb authenticated
mailbox, local daemon long-poll connection"*. That component is small, stateless,
network-facing, connection-concurrency-heavy, and holds **no domain logic** — the
whole security posture of `#121` is that the local machine stays authoritative
and the relay widens no trust boundary.

That shape is an unusually good candidate for Gleam/BEAM, and an unusually good
first Gleam project: the thin-corpus penalty scales with domain complexity, and a
dumb mailbox has none. High concurrency value, low domain risk, small blast
radius.

But the useful question is not "would it work" — it is **what is it like to
operate**, which cannot be answered by reading. Hence a spike whose deliverable
is a decision, not a component.

**This issue does not implement `#121 M3` and does not commit it to Gleam.** It
is time-boxed, throwaway-by-default, and exists to make that later choice
evidence-based. Reasoning captured in
`brain/workshop/pensive/2026-09-07-01-pensive-agentic-web-language.md`.
