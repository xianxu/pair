---
id: 000051
status: open
deps: []
github_issue:
created: 2026-06-11
updated: 2026-06-11
estimate_hours:
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

## Spec

A continuation belongs to the repo of the session it distills. Add a guard so a
repo-root mismatch is caught, not silently wrong. Options (pick at design time):

1. **Authoritative home from the session.** pair's park/continue flow already knows the
   tag's worktree/repo (it's a gathered field — `worktree:` is in the frontmatter). Pass
   the session's repo root explicitly and have the writer use it (not cwd), so "wrong cwd"
   can't happen for pair-driven parks.
2. **Cross-check + refuse/warn.** Before writing, verify the referenced `issues:` resolve
   in the target repo's `workshop/issues/` or `workshop/history/`; if none do, refuse (or
   warn + confirm) — "writing a continuation referencing [000050] into <repo>, which has
   no such issue — proceed?". Catches the wrong-repo case content-first.
3. **Both:** prefer the session's repo as the target, and keep the issues cross-check as a
   backstop for hand-invocation.

Keep it cheap and pure where possible (a `repoMatchesSession(...)` / `issuesResolve(...)`
predicate, unit-tested; the IO stays in the thin seam).

## Done when

- The writer (or pair's continue/park flow) detects a repo-root / session-repo mismatch and **refuses or warns+confirms** rather than silently writing to the wrong repo.
- Covered by a unit test (referenced issues don't resolve in target repo → guard fires) and an integration test (writer pointed at a mismatched repo → non-zero / confirm).

## Plan

- [ ] Decide the guard shape (session-repo-as-target vs issues-cross-check vs both)
- [ ] Implement the pure predicate + wire the thin IO seam; test
- [ ] Update `continuation.md` authoring instructions + atlas to state the home-repo rule

## Log

### 2026-06-11
