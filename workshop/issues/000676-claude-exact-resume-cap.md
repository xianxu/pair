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


- 2026-10-18: change-code full flow admitted after plan-quality disposed PQ-1/PQ-2 and estimate-quality INFO. INFO warns the provisional estimate compresses adversarial fixture/verification effort; retain execution spans in actuals, excluding only queue/operator idle. No gate bypassed.
- TDD: real OS TestClaudeExactOverDescendantCap failed before production edits for both absent and different cwd, returning native root inspection incomplete; `/tmp/pair-676-red.log`, exit 1, SHA256 bd12c27e6945a31da01f54723c2d70df33cbd145e207fed30c5290dbce6da700. Initial green `/tmp/pair-676-green.log`, exit 0, SHA256 a61a18c70288837c3cf9d489251ce142f5bbee5513dde06f338dd68286ee52db. The fixture contains 10,100 unrelated files plus directories. Latest remains incomplete on that same over-cap tree.
- Implemented optional exact-Claude visitor through the actual launcher OSRuntime path. Immediate project entries are bounded; only exact canonical UUID leaves are probed. Existing scanner/EOF/content/role checks remain the authority. Held descriptor and pathname fingerprints include inode/device/mode/size/mtime/ctime. Real filesystem mutations prove replacement/removal, negative insertion, directory metadata mutation, root replacement, changed content, symlink and duplicate refusal. Counted P/2P project and N/2N descendant cases prove independent descendant work and linear immediate-project work. Added descriptor closure and exact returned-ID assertions during main-session review.
- Final focused `go test ./cmd/internal/sessioninventory -run 'NativeSelector|ClaudeExact|DirectoryFingerprint' -count=1` passed 6.269 s; `/tmp/pair-676-focused-final.log` SHA256 a9ae9788d93a2ea41600d9d7885dd05c75a48dabd1534953f65f5ce854bcb130. Same selection with `-race` passed 7.920 s; `/tmp/pair-676-race-final.log` SHA256 7f4eb4497973bd3fd84990a8ebcbc4d3f77d0b044889645e1faa5b04713b091e. Linux amd64 package test compilation passed.
- Full sessioninventory package passed 8.929 s. Initial launcher and artifactpath checks failed because the fresh ordinary clone lacked ignored runtime assets; retained `/tmp/pair-676-packages.log` (SHA256 771ede6af1599542618003a481967309eb1f6c5cba100aa4549845745b066b81) and `/tmp/pair-676-catalog.log` (SHA256 e54b75f232076f4f0ca31a04ae987d4abd7edf7795cdaf0444057236b3932c1d). Canonical `make runtimebundle-generate` fixed setup, `/tmp/pair-676-generate.log` SHA256 5d6bb417bf146c6a22041f56d234e0fb0e1763321e462b3be9577ef9671f0762. Rerun launcher 73.294 s, artifactpath 10.691 s and testpair 7.382 s passed, `/tmp/pair-676-packages-generated.log` SHA256 51a95eae551235ba9bc250788aecf8f76bd0937799c734ebd63976f3e6acc543. No production correction or timeout change was involved in setup recovery.

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
