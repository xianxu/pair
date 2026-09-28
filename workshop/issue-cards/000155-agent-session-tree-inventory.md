---
id: '000155'
status: done
started: 2026-08-28T10:54:39-07:00
created: 2026-08-28
updated: 2026-08-28
estimate_hours: 7.85
actual_hours: 14.01
---

# deterministic agent session-tree inventory

## Problem

Pair's session probing is optimized to discover one native root session for one
new launch. It walks the current agent process tree and returns the first
authorized open transcript; fallbacks select a newly created file or the newest
birth-time candidate. After one ID is written to the tag config and ledger, the
watcher exits. Codex and Muse subagent transcripts are deliberately rejected or
ignored.

That is adequate for a happy-path resume token, but it cannot reconstruct the
world after an unclean shutdown. It does not model the agent's complete session
directory, preserve root-to-subagent relationships, explain competing
candidates, or reliably answer which native session tree belongs to which Pair
tag and in what order. Different Pair consumers consequently grow partial
point-lookups and recency heuristics around the same incomplete evidence.
