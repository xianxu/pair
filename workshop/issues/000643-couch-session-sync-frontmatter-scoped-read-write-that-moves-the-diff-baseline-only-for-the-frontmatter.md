---
id: 000643
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours: 0.76
collaboration-mode: collaboration
card_mirror: 'bb0f84a67cd7d43a5b1f0fb75c8034cb0d29f40a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T04:11:07-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# couch session-sync: frontmatter-scoped read/write that moves the diff baseline only for the frontmatter

## Problem

## Spec

From ariadne#424 M2 (ariadne:2), 2026-10-17. `couch --session-sync read --all` and `write --all` reset the diff baseline for the whole file. Measured with the live conformance test: after a `read --all`, `couch diff` prints only `lock: none`, even for an operator edit made just before. A writer that needs to change only the frontmatter (ariadne#424 maintains the `discussions:` index there) therefore consumes the owner's unread operator edits in every section.

Proposal: frontmatter-scoped verbs that move the baseline only for the frontmatter, the way the section verbs do for a section:
- `couch --session-sync read --frontmatter FILE`
- `couch --session-sync write --frontmatter BODYFILE FILE --if-hash H`

The hash is the frontmatter's; a stale hash refuses. The operator's lock refuses as other writes do.

## Done when

- Reading or writing the frontmatter leaves the diff baseline of the body unchanged. A test makes an operator edit in a section, writes the frontmatter, and the edit still shows in `couch diff`.
- A stale frontmatter hash refuses, and the operator's lock refuses.
- The verbs are in `couch --help` with their cost tags. tl-lead (ariadne#424 or a follow-up) switches to them.

## Plan

- [ ] Add the explicit frontmatter scope and regression tests for byte preservation, scoped shadow acknowledgement, stale hashes and operator locks.
- [ ] Verify the real CLI on disposable files in an isolated namespace, update help and atlas, run the recorded suite, close review and land.

Implementation detail: [durable plan](../plans/000643-frontmatter-sync-plan.md).

## Log

### 2026-10-17

- Claimed and started planning in pair:6. Recovery evidence: #632 already landed; no previous TTY capture is available. Current branch and tracker identify #643 as the active item. No implementation has started.
- Existing section reads do not advance the shadow; whole-file reads do. Frontmatter reads will explicitly acknowledge only their block. Existing section writes without a shadow copy the whole document; the new frontmatter scope must instead start with a frontmatter-only shadow.
- ARCH-DRY/ARCH-PURE: extend the existing guarded document transaction and atomic shadow/state publication; isolate byte splitting and replacement as pure functions. No new lock, timer, or metadata family.
- TL approval receipt fcd4f3fa-4875-4626-93c6-fb486da60eb0 verified submitted: scope approved, full flow if needed; real new CLI from this slot against a disposable file in an isolated namespace is sufficient, no pair:0 staging. TL owns routing ariadne#424 consumer adoption after landing.

- Plan review requested function-level test strategies and an explicit partial-publication/retry contract (PQ-1/PQ-2); durable plan revised before implementation. Per-file atomic replacement is distinct from the sequential document/shadow/state publications.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; calibration is marked stale by estimate-source, so the units are provisional. Two smaller-go-module extensions cover byte projection/guarded stores and CLI wiring/tests: each base design 0.2 hours discounted by 0.2 for the settled plan, implementation 0.5 hours scaled by 0.4. Existing strings/file APIs and the document transaction suffice; no greenfield library discovery. Atlas/docs uses base design 0.1 discounted by 0.2, implementation 0.1 scaled by 0.4. One review uses implementation 0.5 scaled by 0.4. Familiarity 1.0; thorough-plan design buffer 15%. Derived total 0.755, rounded 0.76 hours.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.04 impl=0.2
item: smaller-go-module design=0.04 impl=0.2
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0 impl=0.2
design-buffer: 0.15
total: 0.76
```
