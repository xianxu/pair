---
id: 000643
status: done
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours: 0.76
collaboration-mode: collaboration
card_mirror: 'f82db2cd5222bf92665feab9cf551a714e51254e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T04:11:07-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 0.90
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

- [x] Add the explicit frontmatter scope and regression tests for byte preservation, scoped shadow acknowledgement, stale hashes and operator locks.
- [x] Verify the real CLI on disposable files in an isolated namespace, update help and atlas, and run the recorded suite.

After acceptance checks, close review and landing use the SDLC gates.

Implementation detail: [durable plan](../plans/000643-frontmatter-sync-plan.md).

## Log

### 2026-10-17
- 2026-10-17: closed — First-attempt recorded full suite on pushed candidate passed in 11m26s, log 7e108fb4eadf; both affected packages, focused race and 751343 fuzz executions passed. Real new CLI on isolated disposable legacy and registered files preserved exact CRLF body and Updated bytes and unread diff, stale exit 6 and operator-lock exit 3. Help, shipped reference and atlas updated. TL approved this live route and owns ariadne#424 adoption after landing.; review verdict: SHIP

- Claimed and started planning in pair:6. Recovery evidence: #632 already landed; no previous TTY capture is available. Current branch and tracker identify #643 as the active item. No implementation has started.
- Existing section reads do not advance the shadow; whole-file reads do. Frontmatter reads will explicitly acknowledge only their block. Existing section writes without a shadow copy the whole document; the new frontmatter scope must instead start with a frontmatter-only shadow.
- ARCH-DRY/ARCH-PURE: extend the existing guarded document transaction and atomic shadow/state publication; isolate byte splitting and replacement as pure functions. No new lock, timer, or metadata family.
- TL approval receipt fcd4f3fa-4875-4626-93c6-fb486da60eb0 verified submitted: scope approved, full flow if needed; real new CLI from this slot against a disposable file in an isolated namespace is sufficient, no pair:0 staging. TL owns routing ariadne#424 consumer adoption after landing.

- Plan review requested function-level test strategies and an explicit partial-publication/retry contract (PQ-1/PQ-2); durable plan revised before implementation. Per-file atomic replacement is distinct from the sequential document/shadow/state publications.

- TDD: initial sessionsync run failed on the missing scoped API; CLI parser/real-runtime/help tests then failed on unsupported --frontmatter before wiring. Targeted package runs passed (sessionsync 9.466s, couchcmd 57.457s), focused race runs passed, and the 15-second byte projection/replacement fuzz run passed 751,343 executions. Added direct-first-write, ordinary registered baseline, malformed-shadow and partial-publication recovery coverage.
- Shipped reference baseline check found no documented body-preserving metadata route. Repeating the reference retrieval after the update selected --frontmatter, the complete-block payload and hash boundary, body/Updated preservation, and safe publication-error reconciliation correctly.

- Real built CLI conformance passed for legacy and registered disposable files: scoped read/write retained the exact CRLF body and Updated bytes, unread operator addition stayed in diff, stale returned 6, operator lock returned 3. Isolated namespace was initialized through normal adopt-store preview/apply; the initial smoke correctly refused an unmigrated disposable namespace. No operator store or pair:0 was touched. Scratch evidence: /private/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair643-live.G8sg4M; driver /tmp/pair643-live.py.

- Recorded full suite passed first attempt on be287c12c50c27bc8f1c3d7fb73d1f04b33769ca: 11m26s runtime after 1406s queue wait; full log has no failure markers. Log SHA256 7e108fb4eadfb5407f59179361349ce58e9aa5eae22faf679b40bb1d90c6684d. No retry. Close review is next; landing is pending.

- Close review round 1 returned SHIP/high, zero findings; reviewer independently passed both affected packages. PR #398 opened. Main integrated cleanly; targeted integration checks and normal landing checks follow. TL still owns ariadne#424 adoption.

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

## Revisions

- 2026-10-17: separate completed verification from the subsequent SDLC close/landing verbs in the acceptance checklist, so a checked verification row does not claim that landing already happened. No behavior or acceptance contract changed.
