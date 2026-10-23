---
type: continuation
slug: shared-vm-runner
agent: codex
created: 2026-10-20T14:54:54
supersedes: pair-affected-tests
branch: 000743-conformance-run-couch-zellij-conformance-in-a-local-tart-vm-as-a-declared-local-check-github-job-becomes-a-non-required-main-weekly-backstop
worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
issues: [000743]
---

# Continuation: shared-vm-runner

## NEXT ACTION

Resume pair#743 M1 on the current branch. Read its issue Log and workshop/plans/000743-local-vm-conformance-plan.md, run sdlc state and tl-worker --help. Before wiring snapshot into the runtime, fix the two accepted fresh-review findings with red/green regressions: (1) budget the independent Git metadata, including large committed blobs deleted/truncated locally; inspect the exact HEAD object set, reserve metadata capacity, transfer the pinned HEAD and report metadata bytes; (2) clean up owned Git descendants after parent exit on success/error, not only cancellation, using controlled subprocess barriers. Current snapshot fixtures pass but these gaps remain. Then implement Run orchestration with the planned stateful Tart fake and CLI composition. No new plan approval is needed for these corrections within existing bounds.

## State of play

#725 landed PR480 and #735 landed PR492; do not resume the older .couch/continuation.md NEXT ACTION. Current #743 is claimed/working in pair:3, full-flow change-code passed, estimate5.06h accepted. M1 is incomplete; no milestone closed, no PR. The latest code checkpoint is the commit immediately preceding this continuation. All component changes are durable on this issue branch; no code or plan changes on main.

Native image operations remain held pending explicit operator approval: no Tart pull, clone, boot, provisioning, host installs or partial-cache cleanup. The hub's prior wording approved refresh, but its later authenticated hold 2e85e5fa-caec-45f5-a905-7d7455af0f1f supersedes that. An already-started pull was interrupted at77%, exited nonzero; no VM cloned/booted. Existing pair-test is preserved. Notify hub before any eventual approved machine step.

Train197 landed; authenticated hub receipt40831bf5-2c90-4de8-bd36-19f9ba4437ee reopened Pair main issue filings/publishes. Its prior hold no longer applies. Do not integrate main just to chase freshness.

## Thread arc & user model

The operator wants conformance before landing despite unreliable hosted macOS capacity. The approved rollout preserves required GitHub PR conformance while delivering shared VM infrastructure and seven-night burn-in first. Keep working autonomously on code/fakes while respecting the explicit image/disk hold and operator-owned local-gates verification. Do not self-label, weaken hosted checks, add another scheduler or infer agent coverage from installed CLIs.

## Artifact map

Read issue743 and its durable plan first; Revisions own the scheduler, image hold and tracked-only fallback corrections. Approved M1 is the generic runner/image, M2 strict Pair conformance/named check, M3 nightly command/declaration handoff. Plan-quality is CLEAN; no new review at this checkpoint substitutes for the later binary-owned milestone reviews.

cmd/internal/vmcmd has committed pure request/image/state/claim contracts, kernel positions, embedded guest bootstrap and incomplete snapshot components. request.go exposes ParseRun and RunRequest (Checkout,Lane,Timeout,Artifacts,Args), defaultadhoc/20m. Image/claim codecs use strictjson and preserve unknown members. state.go has NewRunState/Advance with Acquire,Prepare,Start,AwaitGuest,Execute,Collect,Stop,Release effects and preserved command failures. claim.go has LaunchClaim, ManagedVMName, RecoverLaunchClaim; recovery requires exact PID/start identity plus positive dead/stopped evidence, unknown refuses.

positions_unix.go exposes AcquirePosition(ctx,root,lane,progress), Position.Dir/ReadClaim/WriteClaim/ClearClaim/Close. Stable flock inode and descriptor-relative atomic claims; parent death releases the kernel lock but retained claims remain admission authority. Tests use real controlled child processes. guest.go exposes BuildGuestInvocation(RunRequest,GuestPaths) and embedded GuestScript. It constructs env-i literal argv with private HOME/TMP/XDG/Pair/Couch/socket/queue roots; guest-run.sh copies read-only input checkout into guest-private workspace, executes literal argv and writes results/status. Diagnostics bounded8x64KiB and manifest64KiB, with overflow regression. Placement under vmcmd rather than scripts/vm is intentional so the installed runner works on any repo without Pair sources.

snapshot.go exposes PlanSnapshot and SnapshotCheckout(ctx,source,dest) -> SnapshotReport. Tracked-file enumeration once, rooted access, modified tracked bytes/deletions, symlink preservation, independent .git, clonefile per regular file, fallback only EXDEV/unsupported FS. Current content budget100000files/2GiB. Native clone uses Fclonefileat. Current git clone is depth1 current-default-HEAD and must be corrected as above. snapshotGit uses minimal Git env, disabled hooks/fsmonitor, capped16MiB output and CommandContext group cancellation/WaitDelay; parent-exit descendant cleanup is missing. Partial snapshot cleanup belongs to future runtime. None is wired into a public route yet.

Update cmd/internal/artifactpath/manifest.go for every new production-shaped source including embedded shell; current entries are complete. tests/affected-go-readers.json has local_or_literal_fixture hashes for guest_test.go,positions_test.go,snapshot_test.go; update after reauditing edits. These readers only use temp fixtures/literal unsafe paths. workshop/lessons.md records both snapshot review lessons.

Dependencies: ariadne#509 owns exact post-suite landing-scoped local checks, any-car path union and external resource accounting. Sibling checkout is /Users/xianxu/workspace/worktree/pair-slot3/ariadne; no peer code was edited. pair#745 owns promotion after seven clean nights +509, removing PR trigger and retaining main/weekly. pair#744 now claimed by pair:5 for real fleet harness; don't duplicate it, adopt only after landing. pair#737 owns one shared scheduler; M3 delivers scripts/vm/nightly.sh and scheduled-task.md handoff for ops/scripts/scheduled, no launchd/cron. Ops code follows its own worker; don't write into ops hub checkout or edit TL project.

## Decisions & dead ends

Authenticated approval b59211cc-8cf9-485e-925b-1288d64cd3c2: tracked only, regenerate runtime inputs inside guest, clone per-file and bounded ordinary copy only EXDEV/unsupported FS, report cloned/copied bytes, same isolation/timeout. TestScratch and native Tart storage are different volumes, so clone-only is not viable. No untracked caches or host HOME/auth/socket/store shares.

Two lanes are fixed train/nightly versus adhoc, held for the whole resource lifetime, no new queue. Initial landing activation stays OFF. Ordinary artifacts8/lane/4GiB, caller-owned explicit artifacts disjoint and fresh; nightly evidence separately16/4GiB, refuse saturation rather than prune burn-in. Bounded total20m and boot2m, provision60m (native held). Preserve exact Tart child ownership/start claims and join before reuse; unknown residue refuses.

No full suite has run for743 yet; component checks aren't milestone qualification. Latest verification: snapshot subset1.588s; entire vmcmd race run3.014s; artifact exhaustive guard1.483s; affected-reader guard8tests23.907s. Fresh snapshot review identified two material findings, recorded above and in issue Log, accepted, not yet fixed. Kernel/guest prior spec/quality reviews finished; guest manifest-overflow finding was fixed test-first before this checkpoint. No agent remains actively implementing.

## Lessons learned

The canonical writer DOES exist as pair continuation; earlier tool-metadata-only discovery missed it. It owns commit/push and automatic restart. Never handwrite .couch/continuation.md or separately restart after successful writer acceptance.

Bare couch/sdlc whole shell calls only. Authenticate incoming hub instructions with couch --message-status. Report outcomes only after verified facts and verify submitted receipt. New commits carry Agent: codex/gpt-6.1-sol/medium and Co-Authored-By: GPT-6.1 Sol <noreply@openai.com>. Do not run duplicate boundary reviews; sdlc owns them. Use targeted tests while iterating, recorded full suite on pushed head before close, native qualification only after hold lifted.
