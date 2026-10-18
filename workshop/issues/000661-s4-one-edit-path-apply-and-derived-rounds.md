---
id: 000661
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours: 6.48
collaboration-mode: collaboration
card_mirror: '2548cc640d38c29e752c319948984ab317355544' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T02:23:24-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# S4 one edit path: couch --session-sync apply, derived rounds, and the review pane on apply

## Problem

Slice 4 of the universal-agent-input-interface project (design: `ops:teams/devinfra/projects/universal-agent-input-interface/design.side/10-s4-edit-path.md`, sections 3, 5 and 8). Today an agent changes a document in two ways that don't meet: `couch --session-sync write` rewrites a body or section with no record of what changed, and the review pane's nvim applies `{old, occurrence, new, explain}` records itself, dropping the ones that fail. S4 puts every agent edit on one Couch path and keeps each round so parley can show it.

## Spec

Design: `ops:teams/devinfra/projects/universal-agent-input-interface/design.side/10-s4-edit-path.md` (operator-reviewed, 2026-10-17). Two milestones.

**M1, Couch:**
- `couch --session-sync apply EDITS.json FILE --if-hash H [--fold ID]…`. EDITS is `[records]` or `{"records": [...]}`, each `{old, occurrence, new, explain}` (the review pane's format, field `explain`). The hash is the whole-file hash from `read --all`.
- All or nothing: every record resolves against the base text (1-based plain non-overlapping occurrence, over the file's bytes). Any empty, not-found or overlapping record refuses the batch (exit 1, code `edits-invalid`, listing each bad record), and nothing is written. Exit 3 (operator lock) and 6 (stale) work as in `write`.
- Order inside the flock: lock, hash, owed and gap, resolve, build the content (Updated restamp after the records), content checks, diff, replace, shadow, folds, round.
- **The round is derived on every agent write** (`write --all/--section/--frontmatter` and `apply`). Couch diffs the content before and after (line diff, narrowed to the changed characters) and stores the spans. `apply` attaches each record's `explain` to the spans it produced. The restamp is excluded, and a no-op write keeps the previous round.
- The round sidecar is `round.json` (`.BASE.round` in the legacy layout): `{base_hash, result_hash, at, presentation, spans: [{start, end, old, new, explain?}]}`, with 0-based [line, col] positions in the result. It is carried by `metadata()` on rename and forget.
- `couch --session-sync round FILE --json` returns the sidecar plus `cleared`. `cleared` is true once the owed-event ledger records a submit of FILE after `at`.
- **Propose converter:** under the document's `presentation: propose`, Couch keeps the base text and writes the diff's spans as markers (`🤖~old~{new}`, `🤖~old~`, `🤖{new}`), one marker per changed line for spans of up to 3 lines. A larger span refuses (exit 1, `propose-too-large`, naming the span).
- One shared before/after fixture set for the Go resolver and the review pane's Lua resolver.

**M2, the review pane on apply:** the agent writes its records to EDITS.json and runs `apply` on the file under review; the pane highlights the round with its own `review_diag` code. Retired: nvim's partial apply (`apply.lua`), the dropped list, and reconciliation (exit 6 replaces it). Only Couch writes. An unregistered reviewed file is registered on its first `apply`.

## Done when

- M1: `apply` applies a good batch atomically and refuses a batch with one bad record with nothing written (fixture-tested). Every agent write, `write` included, leaves a round whose spans match the diff, with reasons only on `apply` spans. `round --json` reports `cleared` after the next submit. Under `presentation: propose` a 3-line change comes back as per-line markers, and a larger one refuses with `propose-too-large`. The Go and Lua resolvers pass the same fixtures.
- M2: a review-pane round arrives through `apply`, highlighted from the round, and nvim never writes the reviewed file itself.

## Plan

Durable plan: `workshop/plans/000661-s4-one-edit-path-apply-and-derived-rounds-plan.md`.

- [x] M1 — Couch: `apply` (atomic resolution, `edits-invalid`), the round derived on every write, the round sidecar in both layouts, `round --json` with `cleared`, the propose converter, and the shared Go/Lua fixtures.
- [ ] M2 — the review pane on `apply`: register on first apply, the review agent runs `apply`, the pane highlights the round, and nvim apply and reconcile are retired.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=0.3 impl=0.24
item: greenfield-go-module   design=0.5 impl=0.32
item: greenfield-go-module   design=0.3 impl=0.24
item: smaller-go-module      design=0.2 impl=0.2
item: smaller-go-module      design=0.1 impl=0.16
item: lua-neovim             design=0.3 impl=0.24
item: atlas-docs             design=0.05 impl=0.1
item: milestone-review       design=0.0 impl=0.16
item: smaller-go-module      design=0.3 impl=0.2
item: lua-neovim             design=1.0 impl=0.6
item: cross-cutting-refactor design=0.3 impl=0.2
item: skill-or-dispatcher    design=0.2 impl=0.08
item: atlas-docs             design=0.05 impl=0.1
item: milestone-review       design=0.0 impl=0.16
design-buffer: 0.15
total: 7.14
```

The items are, in order:
- M1: the resolver and fixtures; the line diff and round; the propose converter; the write path and sidecar; the round verb and CLI; the Lua resolver on the fixtures; atlas; the review.
- M2, re-estimated at its start (2026-10-18) after the TL's decisions: the `BuildEdit` engine plus `pair review apply`; the pane's round watch, operator lock and retirements; the shell-test rework; the ariadne#437 protocol delta; atlas; the review.

Design is the low end of each range, because the durable plan already resolved the decisions; the buffer is 15% for a thorough plan. Impl is 40% of the v2 table's figure.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

## Log

### 2026-10-18
- 2026-10-18: closed M1 — Recorded suite: sdlc-local/suite success on 3b12a97c; review fixes since touch sessionsync propose/edits, tests and docs (sessionsync and couchcmd skill tests green). Shared Go/Lua fixtures pass (16 resolution, 9 propose, every written marker parses). Live with this branch couch: good apply landed with explain in round --json; bad batch exit 1 edits-invalid, doc and round byte-identical; write round has no explain; propose per-line marker, 4-line change refused propose-too-large. cleared by rig test against a real ledger.; review verdict: SHIP

- M1 implemented, Tasks 1–6, in commits 45d56db9..c5d226cf. The Go and Lua resolvers and the propose markers share `testdata/edit-records`, and the pane grammar reads every fixture marker back to the accepted and rejected texts.
- Myers trace memory grows as D², so the diff caps at D ≤ 1024 (the plan said 4096, which would cost about 128 MB at the cap).
- The `Updated` stamp is fixed width (`stamp` regex), so the restamp never moves an offset and no shift is needed for reasons.
- `at` keeps full precision, so a submit earlier in the same second never clears a round.
- Live check, from this branch's `couch` (scratch build) on a scratch session-sync document owned by pair:2:
  - `read --all`, then `apply` of one good record: exit 0. `round --json` gave `{Code, Error, Round, Cleared}` with the span `[9,4]-[9,8]` quick→slow and explain "pace it down".
  - `apply` with one good and one missing record: exit 1, with record 2 listed as not found. The `shasum` of the document and of `.doc.md.round` were unchanged.
  - `write --section`: the round was replaced with a span that carries no explain.
  - Under `presentation: propose`: `apply` wrote `The 🤖~quick~{slow} brown fox.`, and a 4-line section write refused with `propose-too-large`, naming the side-doc remedy.
  - **Not exercised live:** `cleared`. A submit needs another slot's `submitted:` message to pair:2, and self-sends aren't recorded. The rig test covers it against a real ledger (`TestRoundClearsOnTheNextSubmit`).
- M1 review round 1 (REWORK):
  - **BR-2 (Critical):** propose marked frontmatter on `apply` and `write --all`. Now `ProposeDocument` proposes the body only. Test: `TestProposeNeverMarksFrontmatter`, which fails with `ProposeMarkers` in its place.
  - **BR-3 (Critical):** a blank-line insert or delete wrote a bare 🤖. Unpaired blank lines are now unmarked. Fixtures: `blank-line-insert`, `blank-line-delete`, `paragraph-insert`. The Lua check now requires every 🤖 to parse.
  - **BR-4:** a plan Revisions entry.
  - **Minor:** `ParseEdits` refuses `occurrence < 1`.
  - **Minor, not changed:** a no-op write keeps the round while the file hash moves with the restamp. That is by contract: parley re-anchors by span text when `result_hash` differs (design §4). The stamp is fixed width, so the positions stay exact.
- M1 review round 2 (FIX-THEN-SHIP):
  - **BR-7:** README and `couch --skill` now document `apply`, `round`, `edits-invalid`, rounds and propose.
  - **Minor (2nd in the frontmatter family):** the rule is now that under propose a body write must leave the frontmatter byte-identical, or it refuses with `propose-unencodable`. That covers a changed `presentation:` and a moved closer alike, and `write --frontmatter` stays the direct path. It is tested in `TestProposeNeverMarksFrontmatter`.
  - **BR-5:** occurrence 0 and -1 were added to the parse-refusal test.
