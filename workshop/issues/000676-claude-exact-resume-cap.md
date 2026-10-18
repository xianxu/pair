---
id: 000676
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours: 1.652
severity: critical
collaboration-mode: collaboration
card_mirror: '9d94bb99921e041d65f2b409b371e0148c17a7b9' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T15:12:17-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Exact Claude resume fails above native tree metadata cap

## Problem

Claude relaunch on the current main build exits before registration with `native resume unavailable (incomplete): native root inspection incomplete`, while Codex relaunch succeeds. The operator cannot restart Claude hubs safely. Both reported requested transcripts still exist; starting fresh abandoned their prior conversations rather than recovering them.

## Spec

Validate an explicitly selected Claude native session from its exact target transcript path with bounded, identity-preserving IO, instead of observing every session in the global projects tree. Keep the global discovery cap and fail-closed identity/role/contradiction checks. Do not convert missing, unreadable, changed, child, or conflicting target evidence into permission to resume. Latest/discovery remains bounded and refuses incomplete evidence.

Hub approved this scope in delivery c4bfa120-b5ef-4b05-aa04-71fb3bdd09c8. Land after #651; the hub owns the landing priority grant and operator restart coordination.

## Done when

- A valid exact Claude root resolves with more than 10,000 unrelated entries in the projects tree, including directories.
- Missing, child, changed, conflicting, and unreadable target evidence continues to refuse; explicit resume never launches a different session.
- Counted target-validation tests reject work growing with unrelated global entries; discovery keeps its existing cap.
- Required suite, fresh close review and landing pass; operator live acceptance is coordinated with the hub.

## Plan

- [ ] Pin the production failure and write a real-filesystem regression with a valid exact root plus over 10,000 unrelated entries; observe the current refusal.
- [ ] Reuse target-validation and bounded IO seams to inspect only the requested transcript, preserving canonical identity and root-role checks.
- [ ] Verify rejection and operation-count coverage, run required gates, and land after #651.

## Log

- 2026-10-18: Read-only production metadata: projects tree has 1,640 directories and 9,091 files, 10,731 entries; no walk error. The two requested transcripts exist as regular files, sizes 38,741,547 and 42,517,023 bytes. Native selector uses the 10,000-entry recovery metadata cap globally even for explicit IDs. Missing scrollback is separate from this launcher error.
- Code chain: ValidateNamedContent calls global Observe; VisitRecoveryFiles counts every directory/file and returns ErrReadLimit at the cap; nativeSelectorRuntime remembers the error; ResolveNativeSelector returns the generic incomplete result; createflow refuses before spawning/registering. Introduced by #556 M2 commit 58cfb5f0a046a9b402ed287a206d80c490caa4b8, already on main before #651. Failed launch diagnostic nonces: start-a1c20b5864903a1f and start-ab0e20819ee90817. No transcript contents copied or modified.
- ARCH-DRY/ARCH-SECURE/ARCH-CONSTRAINTS: reuse existing native root proof, keep identity strict, scope bounded work to the exact target rather than raising the global ceiling.

## Revisions

- 2026-10-18: Recorded the approved exact-target refinement in workshop/plans/000676-claude-exact-resume-cap-plan.md. Shallow project enumeration preserves cross-project/relocated IDs and duplicate-root refusal; deriving only current cwd would change that contract. Ordinary clone under pair-slot5 isolates #676 from #651's still-running merge. Start-plan completed after normal substrate catch-up.

- 2026-10-18: Plan-quality round 1 raised PQ-1 (function-specific adversarial strategies) and PQ-2 (directory evidence cannot use the regular-file sameRecoveryInfo helper). Revised the durable plan with an explicit ctime-bearing directory fingerprint, held descriptor ownership, no-follow current-path comparison, and typed real-filesystem probe seam for deterministic replacement/absence races. No code written before gate acceptance.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only. Calibration is flagged stale, so provisional. Derived only after plan-quality passed round 2.

Issue authoring/investigation uses table design 0.5 at mid-density factor 0.5 and impl 0.3 scaled by 0.4. The exact visitor is one new focused Go concern: existing x/sys/unix and runtime descriptor/stat primitives halve base design 0.5 for library availability, then the approved detailed plan applies factor 0.2; impl 0.8 scales by 0.4. Existing selector dispatch is one smaller-Go extension, and deterministic race/count/conformance fixtures a second, each design 0.3 times 0.2 and impl 0.5 times 0.4. Atlas uses design 0.1 times 0.2, impl 0.15 times 0.4. One review/verification/landing primitive uses design 0.2 times 0.2, impl 0.5 times 0.4. Familiarity 1.0. Design subtotal 0.48, implementation subtotal 1.10; thorough-plan design buffer 15% gives 0.48*1.15+1.10=1.652 hours. No new external API/library or cross-repo propagation; queue and operator wait are excluded from active effort.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.25 impl=0.12
item: greenfield-go-module design=0.05 impl=0.32
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.04 impl=0.2
design-buffer: 0.15
total: 1.652
```
