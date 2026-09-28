---
id: '000153'
status: punt
created: 2026-08-25
updated: 2026-09-02
---

# couch: managed worktree lifecycle and garbage collection

## Problem

`ariadne#200` can resolve a requested path to an admission policy whose
on-capacity action is `provision-worktree`, but resolution deliberately does not
create paths or manage branches. `#149` consumes that result and must not invent
a checkout path inside its admission guard. Worktree-managed repositories need
an explicit lifecycle owner for allocation, branch tracking, divergence, and
safe garbage collection.
