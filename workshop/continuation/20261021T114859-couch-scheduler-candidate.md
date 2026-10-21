---
type: continuation
slug: couch-scheduler-candidate
agent: codex
created: 2026-10-21T11:48:59
supersedes: couch-scheduler-one-writer
branch: 000737-couch-scheduler-run-each-hub-s-scheduled-tasks-from-its-operations-repo-due-notices-direct-script-run-skip-if-missed-schedule-status
worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
issues: [737]
---

# Continuation: couch-scheduler-candidate

## NEXT ACTION

Advance the pushed #737 M3 implementation checkpoint through its fresh recorded candidate check and full-model review: recover `sdlc state`, inspect `sdlc local-check run --help` and `sdlc milestone-close --help`, then run `sdlc local-check run --name affected` on the pushed current HEAD and `sdlc milestone-close --issue 737 --milestone M3` with honest verification/atlas evidence. No new design approval is needed: all production work in the approved replacement is implemented and targeted qualification is green. M3 is a genuinely new boundary approved by TL/operator; never reset or waive the exhausted M2 review history. A recorded check may overlap review under worker rules, but do not restart/continuation while your own recorded check is queued/running. Review may precede the live day; whole-issue close and landing still require acceptance.

## State of play

Pair#737 remains working in pair:1 under ops:0. M1 closed SHIP; M2 superseded by operator-approved option A; M3 implementation checkpoint is committed/pushed at f0685773ea65110c45548a8f232c0bfa5dacfc72. A log-only qualification commit follows at cdc651bee3cd302dc24860330955569ec5bfd083, also pushed; tree is clean. No recorded check is queued/running from this turn, no M3 review has run, and no stage, live restart, disposable acceptance, real testscratch day or merge has occurred. All child agents finished with their code/tests committed in the checkpoint. SDLC owns review; no redundant ad-hoc review was run.

Broad Schedule/Skill/HelpAndSkill/MessageSkill race selection passed 74.883s; full component packages passed couchschedule38.309s, operationalrun41.102s, durablefile0.274s, artifactpath13.069s, gcruntime22.672s, gccmd1.270s. Final growth race selection passed3.407s. Final independent root owner recovery race check passed2.102s. Full suite is NOT claimed. Native source-reader inventory/preflight qualification passes all8 tests (43.674s), including seeded refusal through the actual affected runner on detached committed HEAD containing the new writer guard. See the last issue Log; this is not a full-suite pass.

## Thread arc & user model

The operator wants the scheduler finished with one real authority and reliable evidence, not another recovery mechanism. This resumed turn executed the continuation's terminal-batch next action, then completed writer enforcement, fair dispatch, automation seam, docs and production-path qualification from the approved M3 plan. Tests exposed actual quarantine/durable-duplicate consumers missed by the initial integration; fixes stayed within the canonical-first owner model. Continue autonomously through the authorized gates. TL owns the project file, live ops task, staging/restart coordination and operator acceptance; do not re-ask settled design decisions.

## Artifact map

Read the final Log entry in `workshop/issues/000737-couch-scheduler-run-each-hub-s-scheduled-tasks-from-its-operations-repo-due-notices-direct-script-run-skip-if-missed-schedule-status.md` and final revisions in `workshop/plans/000737-couch-scheduler-plan.md`; they record the full contract, red/green evidence and sequencing. Earlier M2 plans/reviews are preserved history, not the current concurrency model. The M2 sidecars still retain BR-14 and previous dispositions for M3's fresh reviewer.

`couchschedule/owner.go` holds the existing permanent namespace flock for the service lifetime. Production StateStore.Update/WithAuthority are removed; old APIs exist only in store_legacy_test.go. Production owner turns are the sole scheduler ledger/history writer; scripts/controllers return facts. Legacy automatic/manual/coordinator orchestration now lives in *_legacy_test.go compatibility files, never live composition.

`couchcmd/schedule_owner.go` supplies canonical-first single/batched event application and refresh; `schedule_owner_batch.go` batches due/startup, terminal receipt/expiry and retained unknown publication/adoption; `schedule_owner_dispatch.go` launches bounded effect work. Receipt scaling first failed64/256 encoded rows then passed8/16 (one publication). Due remains56/112; unknown refresh9/17 includes one newly discovered idle row. History directory sync counts also qualified: due60/60, receipts144/288, unknown96/192. `schedule_owner_loop.go` interleaves eligible hubs per independent script/notice pool with one cursor; binding count is two per hub at8/16 tasks. No queue/journal/election/worker was added.

`refresh` requires durable active OR Last-only canonical evidence before releasing history quarantine; matching unchanged declaration may restore its policy. Receipt/expiry settlement can still consume terminal facts from quarantined authority without releasing new effects. `submitManual` starts with owner ViewDurable so duplicate reservation acknowledgements cannot bypass failed namespace sync. Observation stays read-only. Retained-history publication is now owner receiver methods in schedule_records.go. `schedule_writer_guard_test.go` scans production Go references, aliases/method values/globals/raw writer APIs with negative controls; scripts/test-affected.py registers all its tests and affected-go-readers.json audits new/changed readers.

The production owner qualification is in schedule_owner_durability_test.go (17 post-rename durability consumers, repeated failures, healthy peer, repair/reopen equality and resumed no replay), schedule_owner_native_test.go (actual built controller+worker+owner+private socket+CLI: quiet0/wake10/timeout/cancel130, argv/cwd, history, exact resistant-descendant cleanup), schedule_owner_recovery_test.go (manual/scheduled reserved/running/stopping actual namespace close/reopen, unknown retention protection, missing/mismatched intent, additive ledger/task/attempt/Last/record/key namespaces and immutable output after failed terminal acknowledgement), plus batch_fault/growth/fairness/manual/loop tests. M1's full ledger crash matrix remains in couchschedule store recovery tests and the independent native process/cache tests remain retained.

README, atlas/couch.md/index, CLI help, embedded Couch skill, artifact manifest and reader/preflight inventory are updated. No protected gate/workflow/.sdlc configuration was changed. Source-reader native negative-control was rerun successfully after committing, so the new scanner is qualified from committed HEAD, not merely staged source.

## Live deliberations

Required acceptance remains real ops#1 testscratch for one day. Exit10/timeout disposable qualification uses pair#744's fleet harness and scratch hub/Go tasks, never fake tasks on ops main. TL coordinates :0 staging/build/restart; do not touch :0 unilaterally. Last known #744 PR509 was still in the landing line. Once M3 check/review is ready, ask hub for current staging/harness sequence and report the concrete candidate/evidence. Do not merge or close the whole issue before the live requirement is satisfied.

Authenticated TL receipt f5a2d43e-6121-46e9-8cd1-3524b635bc0b says #739 is not landed. #737 now injects messageService.scheduleAutomationEnabled func(hubRoot)bool with nil/always-on default through service.scheduleAutomationOn to owner. Ephemeral Event.AutomationDisabled skips/advances due for both modes; scheduled pre-release authorization refuses if off; notices check before SendSystem; manual and already-running completion are independent. The second lander wires the real getter and combined broker/automation lock-order test. No cross-branch development.

No user-model questions are unresolved. If the fresh M3 review finds issues, follow the existing SDLC/TL stop/re-plan rules; no cap override or gate waiver is implied. Native test in this turn used real owner/controller/socket with disposable scratch, but it is not the live supervisor restart or real-day acceptance.

## Decisions & dead ends

Reuse the existing state lock as the lifetime fence. Canonical terminal history always settles before edit/cancel/new effect, including failed authority commits. Gate acquisition stays outside owner; gate-ready handoff joins acknowledgement before releasing the gate. Runtime wait order remains hub-effect gate -> owner turn -> broker -> automation; history lock is separate. No owner turn waits for a script or gate.

A temporary Spec.ScriptRun mutation to model automation off was rejected; an ephemeral explicit event input preserves truth without a new persisted state. Plain hub interleaving was insufficient when a hub's first row was not due; filter eligible candidates before existing interleave so an ineligible row cannot consume its hub's turn. Legacy tests remain for inherited contracts, but M3 proof uses the actual production owner.

## Lessons learned

Quarantine blocks new effects, not terminal convergence. Requalify Last-only consumers and duplicate admission after writer consolidation; readable history/reservations do not prove sync. Counter-test filesystem work as well as pure transitions: one full-ledger encode per task is quadratic, while history fsyncs must also stay linear. StateOwner copies store hook configuration at Own: install mutable closure hooks before open; compare reopened fixture structs to avoid nil/empty codec artifacts. Do not run source mutation probes against shared-tree tests; use overlays. Every reported SHA must come from Git. Commit trailers remain Agent: codex/gpt-6.1-sol/medium and Co-Authored-By: GPT-6.1 Sol <noreply@openai.com>.
