---
id: 000360
status: working
deps: []
github_issue:
created: 2026-09-30
updated: 2026-09-30
estimate_hours:
card_mirror: 'f8508a15751ae82456bde5da7314b3ac2c5a5d2f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-30T22:55:02-07:00
flow: {kind: quick, provenance: inferred, spec: "8b97fb72", done: "a2a7c388"}
---

# Address couch slots by repository prefix, alias, or agent

## Problem

Slot references must use the exact repository directory name: `parley.nvim:1`,
not `parley:1`. `ParseWorkspaceReference` (`couchcore/workspaceref.go`) splits
`repo:N`, and `resolveSlotInput` (`couchcore/slotcontext.go`) joins `repo` onto
the fleet root, so anything that isn't the exact directory name fails. People
type the short form naturally, and an agent in an LLM session understands it,
but couch does not.

Slots also carry attributes, such as which coding agent they run. There is no
way to address "a free slot running claude", which cross-slot dispatch (#353)
will want.

## Spec

A slot is a repository plus its Nth checkout, addressed `repo:N` and shown that
way on the tab bar. Every surface that takes a slot reference — operation refs
(`couch resume parley:1`), the switcher filter, and `couch --send-to` — resolves
the repository part with one shared rule. Resolution stays deterministic and
inside couch; no model is involved.

1. **Repository name resolution** (one pure function, used by both the
   operation resolver `couchcore.resolveSlotInput`/`enrolledPrimary` and the
   message resolver `couchmessage.ResolveRecipient`):
   - exact directory name or exact alias wins;
   - otherwise a unique prefix of a directory name or alias resolves
     (`parley:1` → `parley.nvim:1`);
   - a prefix matching several repositories is refused, listing them;
   - no match is refused with a bounded candidate list (repositories, and for
     a slot miss the existing slot numbers), so the caller corrects in one step.
   For operations the candidate set is the enrolled repositories; an exact
   sibling directory under the fleet root keeps working as today, and the
   prefix fallback applies only when that directory does not exist. For
   messaging the candidate set is the live bindings' repositories.
2. **Repository alias.** One optional alias per enrolled repository
   (`xianxu.dev` → `blog`), set from the live `:0` row's "alias" action (the
   slot-world replacement for rename, #363). (No CLI form: switcher operations
   are not argv-reachable; see the plan's Revisions.) It is a valid
   repository token, unique across enrolled repositories and their directory
   names; empty clears it. It works anywhere the repository name does
   (`blog:1`, `--send-to blog`), and the tab bar and switcher label slots
   `alias:N`.
3. **Agent filter for messaging.** `couch --send-to REPO --agent NAME` picks a
   free slot of that repository running that agent. Repository identity is
   checked against all live bindings before the agent filter (#353 BR-5). With
   an exact `repo:N` target, a mismatched agent refuses. An agent filter
   requires a repository; there is no cross-repository "any slot".
4. **Inventory.** `couch --actors --json` already lists live slots with their
   agent; it gains the alias. Senders never need it to route: only a miss
   returns candidates.
5. **Out of scope.** Thread names as addresses (the operator decided slots,
   not threads, are the addressing unit; rename itself retires in #363), an
   LLM resolver, the switcher's action model (#363), and slot removal (#364).

## Done when

- `parley:1` resolves to `parley.nvim:1` for operations and for `--send-to`,
  with a test that delivers to the resolved slot; an ambiguous prefix is
  refused with the candidates listed.
- After aliasing `xianxu.dev` to `blog`, `blog:1` and `--send-to blog` reach
  xianxu.dev's slots, `xianxu.dev:1` still works, and the tab bar and switcher
  show `blog:1`.
- An alias colliding with another repository's name or alias is refused.
- A reference that resolves to nothing lists bounded candidates.
- `--send-to pair --agent codex` picks the free codex slot when two free slots
  run different agents, and refuses when none runs it.
- The couch skill, README and atlas describe prefixes, aliases and `--agent`;
  `couch --help` shows `--agent`.

## Plan

- [ ] Durable plan: `workshop/plans/000360-address-couch-slots-by-short-repo-name-thread-name-or-attributes-plan.md`.

## Log

### 2026-09-30

- Filed at the operator's request. Related to #353 (cross-slot dispatch
  addresses `repo:N` and "any free slot of `repo`"), and the attribute
  selection is what #353's "any free slot" would use. Not a dependency: short
  names help every command that takes a slot reference today.
- Current resolution: `ParseWorkspaceReference` plus `resolveSlotInput`, exact
  directory name under the fleet root. Thread names already resolve as plain
  refs (`ref` is "tag, path, or name"), but not inside `repo:N`.
- Operator review: the reason for an in-couch resolver is to keep slot
  inventory out of the sending agent's context on every route. Folded in as
  a requirement; deterministic filters resolved inside couch meet it, with
  candidates returned only on a miss.
- Design talk with the operator: `parley:1` failed in `--send-to` too (exact
  family compare in `ResolveRecipient`). Thread names are not the addressing
  unit: a slot is repo + Nth checkout. Rename becomes a repository alias on the
  live `:0` row. The switcher action cleanup moved to #363, slot removal to
  #364.

## Revisions

### 2026-09-30 — rescoped after design talk

- Dropped: thread names addressing slots (`name:N`), and the LLM resolver
  alternative (decided: out of routing; no switcher fallback planned).
- Added: per-repository alias, shared by addressing and the tab bar label.
- Kept: unique-prefix fallback, candidates on miss, `--agent` filter, live
  inventory (already `couch --actors --json`).
