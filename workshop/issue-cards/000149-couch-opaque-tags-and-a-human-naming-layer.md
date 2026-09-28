---
id: '000149'
status: done
started: 2026-08-25T14:21:34-07:00
created: 2026-08-21
updated: 2026-08-27
estimate_hours: 17.80
actual_hours: 56.46
---

# couch: opaque tags and a human naming layer

## Problem

pair's **tag does two jobs at once**: it is the durable storage key
(`draft-<tag>.md`, `ledger-<tag>.jsonl`, `log-<tag>.md`,
`scrollback-<tag>-<agent>.raw`) *and* the human handle typed into
`pair <tag>`. Three symptoms follow from that one conflation:

- **Naming is demanded upfront.** A space cannot exist before it has a name,
  so the operator must know what a thread is before starting it. That is
  backwards — the constitution's own flow has an issue crystallise mid-thread,
  which is why `sdlc claim` is cheap and the estimate comes later.
- **Renaming is not offered**, because a rename would be a filesystem move.
- **Tags accumulate with no cleanup.** Nothing distinguishes a thread worth
  keeping from an abandoned one, because everything has a name and nothing
  records whether the name was deliberate.

couch made the same split two days ago for its own identity (opaque system id,
mutable human labels over it — see the project's 2026-08-21 scope event). This
applies it one layer down, to pair.
