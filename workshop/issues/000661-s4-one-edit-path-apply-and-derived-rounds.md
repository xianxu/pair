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

- [ ] M1 — Couch: `apply` (atomic resolution, `edits-invalid`), the round derived on every write, the round sidecar in both layouts, `round --json` with `cleared`, the propose converter, and the shared Go/Lua fixtures.
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
item: lua-neovim             design=1.0 impl=0.48
item: smaller-go-module      design=0.2 impl=0.16
item: skill-or-dispatcher    design=0.3 impl=0.12
item: atlas-docs             design=0.05 impl=0.1
item: milestone-review       design=0.0 impl=0.16
design-buffer: 0.15
total: 6.48
```

The items are, in order:
- M1: the resolver and fixtures; the line diff and round; the propose converter; the write path and sidecar; the round verb and CLI; the Lua resolver on the fixtures; atlas; the review.
- M2: the pane on the round, with nvim apply and reconcile retired; register on first apply; the review agent's instructions; atlas; the review.

Design is the low end of each range, because the durable plan already resolved the decisions; the buffer is 15% for a thorough plan. Impl is 40% of the v2 table's figure.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

## Log

### 2026-10-18
