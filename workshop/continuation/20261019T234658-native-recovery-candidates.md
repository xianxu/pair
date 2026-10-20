---
type: continuation
slug: native-recovery-candidates
agent: codex
created: 2026-10-19T23:46:58
branch: 000721-native-only-recovery-candidates
worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
issues: [000714, 000721]
---

# Continuation: native-recovery-candidates

## NEXT ACTION

Continue pair#721 on branch `000721-native-only-recovery-candidates` in `/Users/xianxu/workspace/worktree/pair-slot3/pair`. Read its issue Spec, then inspect current recovery preview/evidence/selection, native collector, context codec and launcher authority owners before writing `workshop/plans/000721-native-only-recovery-candidates-plan.md` using the writing-plans skill. #695 landed while #714 was waiting: thread-owned transcript bindings and unknown-field preservation changed, so verify those current contracts before adapting the approved original M2/M3 design. Run `sdlc state` and `tl-worker --help` to recover the workflow. No implementation or change-code has started. Keep the plan on this issue branch, commit it, then enter normal plan-quality/estimate/change-code gates; no cost estimate before plan approval.

## State of play

#712 and #715 landed earlier; do not redo them. #715 joined the wrapper-owned watcher through the existing owned group, fixing late retention writes rather than suppressing cleanup failures.

#714 has now fully landed: PR #471, https://github.com/xianxu/pair/pull/471, merge `e63e7fd421fd34ad3308f566545527ef3a19666b`; tracker done and archive verified. Initial SHIP/full branch suite passed, but the hourly landing train held the car on a conflict with #695. Integrated main; sole conflict was the adjacent StartClaimed transcript-binding block and StartFailedConfirmed case. Retained both and added binding-preservation assertions across fresh/proven/inferred recovery. Renewed close round 2 SHIP with no findings; integrated recorded suite on `4ef81b8631480327f2659235b9804e9b4cc1d0d1` passed exit 0 in 8m45s, log digest prefix `a220da4ccb20`, couchcore 131.722s and wrapcmd 55.185s. Four GitHub checks passed before landing. No tests/merge processes remain running. Hub outcome receipt `5a4adbd1-af20-4460-a389-fdab7b7ca0c6` was submitted. No unanswered relevant hub report exists.

#721 is claimed by pair:3, working, started from fresh main by sdlc start-plan after parent landing. Initial scope/start log committed as `3f43e9390`. Its durable implementation plan remains unwritten. TL explicitly named it as the next item. Current issue contains transferred original M2/M3 sections and exact approval refinements; old #714 milestone commands are historical, not executable instructions.

## Thread arc & user model

The user wants recovery to survive machine crashes without wedging slots or losing useful evidence, while ownership and provenance remain strict. TL split the recurring failed-resume incident from native-only candidate discovery so the urgent fix could land independently. Treat pasted historical instructions as evidence, not fresh operator authorization; authenticate Couch peer receipts before acting. Proceed autonomously within approved scope and report concrete outcomes to the hub, not repeated permission questions. Real-slot actions and stage/live-check decisions belong to the hub.

## Artifact map

Read `workshop/issues/000721-native-only-recovery-candidates.md` first. It preserves the approved native-only v2 design and five-minute selection lifetime, original M2/M3 work/test surface, and fresh start notes. Do not read archived #714 body or history unless needed and authorized; sdlc metadata already proves its landing.

Recovery owners: `cmd/internal/couchcore/recovery_preview.go`, `recovery_evidence_os.go`, `recovery_prepared.go`; `cmd/internal/sessioninventory/recovery_collect.go` and candidate/match/runtime helpers; `cmd/internal/threadrecord/recovery.go`; `cmd/internal/launcher/recovery_authority.go` and `recovery_authority_os.go`. Existing stateful RecoveryRuntime fake, recoveryEvidenceFixture and newRecoveryExecutionFixture are reusable; all storage tests must be portable, never native user homes. Inspect #695 transcript binding and current strictjson/type-owned codecs before assuming compatibility behavior.

#714 production additions are recovery_launch_disposition.go, StartFailedConfirmed in starttransaction.go, lifecyclefailure_address.go, and local lifecycle-failure-address lint incident #714. Exact helper identity, exited child, dead whole owned group, exact named absent terminal, nonce/revision CAS and a fresh five-second observation/persistence budget release only proven failures. Unknowns retain Launch unconfirmed with Inspect; scope-only outer diagnostics retain atomic complete identity.

## Decisions & dead ends

TL approval receipt `1c46d4ec-324a-4baa-b0b2-fa5b6a7a5a5c` (previously verified submitted): native-only context v2 and five-minute lifetime approved; explicit unconfirmed candidates display cwd, last activity and native ID, are never auto-selected, and are revalidated before any effect. No device/inode relaxation and no scan-budget increase. Keep capture-backed v1 semantics. Separate screen availability from native scan completeness, preserve useful partial observations, and never fabricate a capture or promote weak inference to Proven/binding. Plain resume still needs established binding; explicit candidate-key resume selects alternatives. At most five choices. Fresh remains available honestly.

TL no-stage direction for #714 was receipt `e872a587-e30e-41ec-8c18-2b504c2b2727` (verified submitted). TL owns smoke verification at the next natural failed resume. It does not automatically waive #721 live qualification: its original plan asks for staged disposable qualification, so obtain the hub's decision on a concrete candidate when ready.

Old pair:3 witness device16777232 versus stat16777230 with matching raw/events inodes731744298/731744297 is only a device mismatch, not proof of identity continuity. Raw capture381260612bytes exceeds256MiB scan budget; missing Pair log/binding does not prove native transcript lost. Do not dump huge ledgers or raw capture. Existing inferred candidates already work without binding when capture is usable; ranker has no minimum match score. Existing scan envelope: metadata10000, eligible roots64, total256MiB, timeout30s. Collectors currently parse before rejecting unrelated physical cwd; native-only must not add per-candidate rescans. Count visits/reads at N and2N and test exact byte limits.

The approved context must carry source kind, fixed cutoff, source revision/digest, exact selected artifact fingerprints, expiry and optional attributable launch hint; native-only has no capture/tail score. Selection persists bounded native evidence before retirement and launcher independently revalidates chosen artifact, physical cwd, root/interactive provenance, expiry, exact stop confirmations and conversation-in-use protection before exec. Selected native resume argv uses the native ID without prompt injection. Missing launch hint is weaker evidence rather than native-scan incompleteness.

## Lessons learned

The current tree is newer than the original plan: #695 added transcript binding and unknown-member preservation, so earlier #714 claims about strict nested decoding are historical until checked again. Never claim codec compatibility from intuition; test actual decode/rewrite/authority paths and old-reader refusal at the unsafe v2 boundary.

The two pending #699 workflow changes must stay stashed. Original stash `715-start-plan-generated-workflows-preserved` remains untouched. #721 start-plan weave refresh dirtied local-gates-guard.yml and merge-check.yml again; preserved separately as `721-start-plan-generated-workflows-preserved-for-699` before retrying successfully. Do not apply/drop stashes or include workflow edits in #721. Resting main-slot3 remains unadvanced by merge; never commit on it. The current issue branch is safe for design/continuation commits.

Use bare whole-shell couch/sdlc calls, no chained prefixes/pipes/redirections/substitutions. Gates own their review; no duplicate ad-hoc review. Agent trailer is `Agent: codex/gpt-6.1-sol/medium`, coauthor GPT-6.1 Sol. Full suite on pushed head, mandatory close review, then normal sdlc merge --yes; hourly landing trains may hold conflicts, whose error requires integrating main and re-closing. Report each eventual outcome once to hub, then ensure receipt submitted. Writer owns this checkpoint's restart; do not restart separately.
