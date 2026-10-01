---
id: 000360
status: done
deps: []
github_issue:
created: 2026-09-30
updated: 2026-10-01
estimate_hours:
card_mirror: '2fed499008f7aa01db237bdbf12d4236abfe1ef6' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-30T22:55:02-07:00
flow: {kind: full, provenance: inferred}
actual_hours: 1.09
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

- [x] Durable plan: `workshop/plans/000360-address-couch-slots-by-short-repo-name-thread-name-or-attributes-plan.md`.

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

### 2026-10-01
- 2026-10-01: closed — Operator smoke on pair:0 at b8ca068c: alias xianxu.dev->blog shows blog in tab+switcher; couch --actors instant, lists xianxu.dev:0 (blog); --send-to parley:1 delivered to parley.nvim:1, pong receipt 0c516971 verified parley.nvim:1->pair:1 submitted. Round-1 review fixes (5f6efb17) unit-tested with mutation checks: offline enrolled exact name refuses instead of prefix-routing (control case shows old reroute), :0-only alias labels, fleet-root shadow anchor, shared bounded list. couchmessage/couchcmd/couchcore/couchtty/wrapcmd pass; artifactpath violation list byte-identical to merge base; full suite at b8ca068c: make -k test green except test-changelog (green under scratchpad TMPDIR), go test ./... 77 ok with 3 failures identical on merge base.; review verdict: SHIP
- 2026-10-01: flow upgraded quick → full — 918 added lines in code files (limit 100); an earlier round of this close already ran the full review

- Implemented per the plan (commits `#360: …`). Operator smoke on pair:0:
  alias `xianxu.dev` → `blog` was stored, but the switcher kept the thread's
  old operator name and the tab kept the pane label `xianxu.dev`: xianxu.dev
  has only `:0`, and the alias only renamed slot groups. Fixed by letting
  `RepositoryAlias` outrank the thread name and pane label in
  `ActionableThreadSummary.Label`, with a regression test through the tab-bar
  model. `couch actors` (no `--`) is the launch form, not `--actors`, hence the
  supervisor-lease error.
- Second smoke: `couch --actors` timed out on every call. Cause (#353 code):
  each listing re-probed every slot serially (zellij, process, git, wrapper
  RPC) inside the same 2s the client waits, while one-second heartbeats and
  reconciliation ran ~3 full liveness checks per wrapper per second over a
  binding map that never shrank. Fixed here at the operator's request:
  `--actors` reads broker memory (last heartbeat + resting probe, stale →
  unknown); a full check is reused for 10s on observe-only paths (sends,
  operator-submit, Reserve, Deliver still re-check); dead bindings are
  dropped. Also fixed: alias reads failed outright on a busy store lock (now
  retried within the caller's deadline).
- `parley:1` resolved correctly but parley.nvim:1 never registered: Claude Code
  auto-updated to 2.1.287 and #353's exact-version allowlist silently skipped
  it. Allowlist removed at the operator's direction (`peerReceiverAgents`);
  evidence-based upgrade validation filed as #368.
- Final smoke (operator): alias shows `blog` in tab and switcher; `--actors`
  instant and lists `xianxu.dev:0 (blog)`; ping to `parley:1` delivered to
  parley.nvim:1, and its pong receipt `0c516971` verified `parley.nvim:1 ->
  pair:1 submitted` with matching body.
- Full suite on pair:0 at `b8ca068c`: `make -k test` green except
  `test-changelog` (green under scratchpad TMPDIR); `go test ./...` 77 ok, three
  failures identical on the merge base (`TestBareCouchInstalledCommand`,
  `TestCouchReferencesLocalArchiveLocatorRoundTrip`,
  `TestProductionArtifactReferencesAreExactlyClassified`, violation list
  byte-identical to base).

## Revisions

### 2026-09-30 — rescoped after design talk

- Dropped: thread names addressing slots (`name:N`), and the LLM resolver
  alternative (decided: out of routing; no switcher fallback planned).
- Added: per-repository alias, shared by addressing and the tab bar label.
- Kept: unique-prefix fallback, candidates on miss, `--agent` filter, live
  inventory (already `couch --actors --json`).
