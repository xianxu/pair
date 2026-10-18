---
id: 000676
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
severity: critical
collaboration-mode: collaboration
card_mirror: '772493abd9664ac669254c5fcf5c4c4057de4f6f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T15:12:17-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
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
