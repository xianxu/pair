---
id: '000051'
status: wontfix
created: 2026-06-11
updated: 2026-06-16
---

# pair-continuation: guard repo-root vs distilled session's repo

## Problem

The continuation writer (`cmd/pair-continuation`, shipped in `#50`) writes to whatever
repo `-repo-root` resolves to — the flag if given, else `git rev-parse --show-toplevel`
of the cwd. It does **not** verify that the target repo is the repo of the *session being
distilled*, nor that the referenced `issues:` exist there. So a live distill agent invoked
from a **different repo root** than the pair tag's repo (e.g. parking the `pair-port`
session while cwd is in `pair-brain`, or a stray `-repo-root`) would silently write +
commit + **push** the continuation into the wrong repo — no error. (Surfaced while
dogfooding `#50`; the writer trusts its inputs.)
