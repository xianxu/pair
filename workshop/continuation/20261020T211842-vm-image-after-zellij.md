---
type: continuation
slug: vm-image-after-zellij
agent: codex
created: 2026-10-20T21:18:42
supersedes: shared-vm-runner
branch: 000743-conformance-run-couch-zellij-conformance-in-a-local-tart-vm-as-a-declared-local-check-github-job-becomes-a-non-required-main-weekly-backstop
worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
issues: [000748, 000743]
---

# Continuation: vm-image-after-zellij

## NEXT ACTION

Resume #743 M1 native image preparation, now explicitly approved after #748 lands. Read issue743 and its durable plan Revisions first, then sdlc state and tl-worker --help. Notify hub immediately before machine image work. Reinspect Tart state: the interrupted pull's tmp directory was empty at20:37, so do not invent partial state to delete. Resolve/verify the current Tahoe base digest, finish a clean pull, provision a new machine-wide image through the approved ad-hoc lane/owned-child protection, and preserve pair-test. Pin the checksum-verified official Zellij0.45.1 asset used in CI; Homebrew's same-version binary differs. Do not ask again for this already-authenticated operator authorization. No other ~/.tart deletion is authorized.

## State of play

#748 landed PR500 at15d7a02b9677d00da8df77cf88948df8d5941df1, verified MERGED at2026-10-21T04:15:57Z. The integrated recorded suite passed9m54s (log SHA prefix af84cab082f6), and final required CI was green without waivers. Landed outcome receipt08cc1c4c-ae83-41fc-ad29-0b4707044daf was verified submitted. Hub had granted priority via submitted receipt db0cf661-6d1b-4035-81a8-798399e41afc and reiterated land then image work. No suite or merge runner remains active. Its bounded product mitigation has three total idempotent-read attempts under one unchanged deadline, no sleeps/growth, strict refusal on exhaustion. Independent residual and official-asset refusals are retained honestly in issue/evidence; no universal zero-failure claim. Do not restart this issue or expand its retry policy.

#743 is still claimed pair:3/working on its preserved branch, full-flow change-code approved, estimate5.06h provisional. Latest approval checkpoint24f9fb4ba is pushed; the original portable code checkpoint remains0ea7d5bd078fb0e2c7ed4b1b4104247d561449e8. Main has advanced and #748 is now landed: integrate origin/main before final qualification as needed, preserving the issue branch and resolving inventories by audit. M1 has portable runner/CLI, safe race/fixture qualification and clean component reviews; no real VM qualification, milestone close, PR or full recorded suite. M2 strict Pair conformance and M3 nightly handoff remain. The original continuation's snapshot NEXT ACTION has already been completed; do not redo it. All previous agents finished; do not resume stale delegated work. No leftover native control process is expected.

## Thread arc & user model

The user wants causal proof and real consumer qualification, with autonomous progress after approval. #748 interrupted the VM work because its native flake held landings. The TL accepted a bounded live-product compatibility mitigation for an unreleased upstream bug, with residual failures clearly recorded; patched binaries nobody runs and unknown-as-success were rejected. This now informs the VM image's official-binary provenance requirement. Preserve the user's development VM/cache, branch separation, hosted conformance and strict lifecycle uncertainty behavior.

## Artifact map

Read workshop/issues/000743-conformance-run-couch-zellij-conformance-in-a-local-tart-vm-as-a-declared-local-check-github-job-becomes-a-non-required-main-weekly-backstop.md then workshop/plans/000743-local-vm-conformance-plan.md. Their Revisions own the rollout and two approved lifecycle replans. cmd/internal/vmcmd contains the tested generic runner, host/driver seams, strict claims/manifests, tracked-only snapshot and embedded guest bootstrap. Image provisioning scripts/lock and native qualification remain to implement; the image must publish atomically only after qualification.

#748's archived issue, workshop/evidence/000748/ and atlas/zellij-reply-safety.md retain root cause, deterministic regressions, raw native schedules, strict controls and binary hashes. Key insights are already captured there and in workshop/lessons.md; no additional pensive is needed. PR500 is the landing reference. Official aarch64 Darwin0.45.1 SHA256 ca5f9333735bdbc59a621f1d8ed8e24798845302a28ff175d253d4793d5a4a2c; Homebrew binary SHA604c684330fe097bc1363ada020d1b9b0ad17caa48d633e838869dfe9fc49e9c. Official binary was downloaded privately at /Volumes/TestScratch/tmp/pair-slot3/pair748-release.UVLqUj/zellij; installed binary unchanged.

ariadne#509 owns exact train-scoped local checks and external resource accounting; peer checkout /Users/xianxu/workspace/worktree/pair-slot3/ariadne. #744 owns shared genuine fleet/agent harness; adopt after landing, don't duplicate or infer agent coverage from installed CLIs. #745 owns promotion after seven clean nights +509. #737 owns the one scheduler; M3 supplies nightly command and declaration handoff, no launchd/cron. No ops checkout edits/project ownership.

## Decisions & dead ends

Authenticated operator image approval via hub receipt3708133b-f037-4efe-bdb2-4bd40dd5e457 supersedes the former hold: finish748, redo pull cleanly verifying digest, remove only identified state from our interrupted pull, build machine-wide image, leave pair-test intact; delete nothing else in ~/.tart without asking. Receipt was verified submitted while live; expiration doesn't revoke authorization. Hub acceptance0fe3af03-7588-42d1-9403-b7eaf6037fd8 additionally requires official checksum-pinned Zellij asset like CI.

Timeout envelope is timeout N + cleanup up to10s, not a reserved subtraction. Keep direct children unreaped until final numeric group signal, then Wait and observation-only confirmation; every secondary-group uncertainty persists a blocking cleanup claim and retained inputs. Guest-local fixed bounded result streams through guest agent, no writable host share or quota APFS mount. A THIRD lifecycle mechanism change must stop for hub options review before coding. Keep runner small and serving conformance first.

Two machine-wide lanes, train/nightly vs adhoc, held for whole lifetime; no new queue. Native image provisioning/qualification also acquires adhoc. Snapshot tracked-only with independent bounded exact-HEAD metadata; clonefile with bounded copy only EXDEV/unsupported filesystem. No host HOME/auth/socket/store shares, no credentials copied. Landing activation stays OFF; GitHub PR conformance remains required through seven clean nights and separate promotion.

## Lessons learned

Use bare whole-shell couch/sdlc calls, authenticate hub receipts, and verify submitted for outcome reports. sdlc owns mandatory fresh boundary reviews. Run sdlc agent-id and copy its current line for new commits; prior code used codex/gpt-6.1-sol/medium, but the recovered harness currently prints codex/-/-. Latest documentation commit followed that returned line. Co-Authored-By has been GPT-6.1 Sol <noreply@openai.com>. New production-shaped files require exhaustive artifact-source inventory disposition; re-audit changed reader hashes before full suite. Use bounded directory listings, not recursive output over unrelated Tart clones. Canonical pair continuation writer commits/pushes and auto-restarts; don't handwrite .couch continuation or separately restart.
