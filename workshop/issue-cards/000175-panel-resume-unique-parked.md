---
id: 000175
status: open
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
github_issue:
---

# Resume a unique parked thread from the panel

## Problem

`pair#167` resumes a unique parked thread on **`couch` startup** — "on
interactive `couch [<repo>]` startup, before preparing a new root" — installing
it as home. Starting an actor from the **panel** takes a different path and
always creates a new thread, even when exactly one verified-parked thread sits
at that same normalized path.

Observed 2026-09-02 in `/Users/xianxu/workspace/tools`, capacity `bounded,
limit 1`:

```
couch-21baa48c3a7f009b  created 09-01 20:34  verified_park 09-02 08:38  no incarnations
couch-64bbe04986164fae  created 09-02 12:27  incarnation: live
```

Identical `starting_path`, `working_path` and `repo_scope`. Admission is correct
to allow it — it counts live/unknown/creating incarnations, and a parked thread
has none, so park genuinely releases the tree. The gap is that the panel never
asks whether the thread being created already exists parked.

**The ratchet.** `#167` resolves ambiguity by creating: "more than one matching
candidate → treat the set as non-resumable and create a new root." Under bounded
capacity that compounds — two parked threads make a third, three make a fourth,
each one making the next start *more* ambiguous. The state that most needs
resolving is the state the rule most reliably worsens.
