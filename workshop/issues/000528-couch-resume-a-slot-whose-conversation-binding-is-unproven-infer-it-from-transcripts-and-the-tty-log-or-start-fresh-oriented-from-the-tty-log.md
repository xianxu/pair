---
id: 000528
status: working
deps: []
github_issue:
target: slot-continuity
created: 2026-10-13
updated: 2026-10-13
estimate_hours: 6.3535
card_mirror: 'fb4c11eee48daa5228e9b370920ddf2f84d2321c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T08:57:28-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log

## Problem

On 2026-10-13 pair:6 could not resume: relaunch reported
`resume-binding-provisional`, claiming no completed turn proved the conversation.
The operator could identify its Codex rollout (2026-10-12T20-13-14,
01a1333f-…) by worktree cwd, codex-tui originator and 07:54 last activity,
matching the tty's “Worked for 19m 19s • 7:54 AM”. Another rollout at 20:15:55
(last active 07:37) shared the cwd but belonged to a boundary review. After
Couch restart, Zellij resurrected empty “Waiting to run” panes while Couch
still reported the slot live. The operator called this a weak spot in the
Couch/Pair stack and requested robust recovery with a few visible states,
each offering one primary action.

## Spec

Measure Claude Code, Codex and Grok transcript metadata first: worktree cwd,
interactive provenance, last activity and extractable tail text. Recovery ladder:

1. Proven binding resumes through the existing path.
2. Unproven binding gathers interactive transcript candidates with exact slot
   worktree cwd and modification after the slot's last launch. Rank by activity
   alignment with the tty and transcript-tail/tty-tail text overlap. Exactly one
   strong candidate offers an explicit confirmation: resume the session with
   cwd, last-active time and tail-match percentage shown. Never resume inferred
   identity silently or treat an unconfirmed heuristic as proven authority.
3. Ambiguous or absent candidates offer fresh startup oriented from preserved
   tty evidence, branch and issue. Orientation names the exact previous screen
   path and `couch --peek repo:N --previous --all` as the read route.
4. Dead recorded processes or resurrected panes with no running commands show
   “needs relaunch”, never “live”. Unknown observations retain their uncertainty.

Design before code; report the durable design to the TL for approval. Observe
pair:6 read-only while measuring; no live recovery/reboot is authorized here.

## Done when

- pair:6's exact case offers the correct inferred Codex session with evidence
  and resumes it only after operator confirmation.
- Ambiguous/no candidates select a fresh tty-oriented session as the fallback.
- Resurrected/dead slots show “needs relaunch”.
- Tests cover every rung using measured native transcript shapes and stateful
  lifecycle doubles, including reviewer exclusion and uncertain observations.

## Plan

- [x] Measure native metadata, exact pair:6 evidence and current recovery owners.
- [x] Approve durable design: workshop/plans/000528-slot-recovery-plan.md.
- [x] Historical first implementation — terminal policy and native recovery (prior M1/M2 reviews retained; superseded by operator revision).
- [x] M1 — Always-start preview-only Couch, listed-live attach, policy removal and three-outcome projection; separate pre-upgrade operator acceptance.
- [x] Split remaining revised M2 — Resume/Fresh execution, person-or-hub confirmation and always-reachable Fresh — into #556; not implemented or accepted as part of #528.
- [x] Split remaining revised M3 — new-wrapper endpoint repair without child restart/input replay — into #557; not implemented or accepted as part of #528.


## Revisions

- 2026-10-13 — Operator direction 15876eb2 supersedes 3cd0aa805: Couch always starts; terminal liveness is Zellij list-sessions (listed/non-EXITED live). At selected attach only, live attaches normally; not live never attaches, deletes the exact stale saved session without force and offers Relaunch. Remove policy/custom socket admission; retain attach lock, new-session serialization off and resume ladder. Exact pre-upgrade restart compatibility and honest list→attach death-race measurement are required. PLAN ONLY, operator approval before code/build; see latest durable-plan revision.

- 2026-10-13 — TL/operator correction 6bf45548 supersedes policy-based attach admission: answering live server/socket attaches as before, old or new, with no session-policy requirement; absent answering server never resurrects and offers Relaunch. Re-plan recorded in durable plan; M1 prior SHIP does not certify this new compatibility rule. Await TL approval before code.

- 2026-10-13 11:39: BR-5 extends the bounded persisted-input rule to the complete M1 consumer set through shared osfs IO; config is bounded at 1 MiB and capture events keep their existing whole-file 8 MiB limit.

- 2026-10-13 11:10: M1 boundary REWORK tightens uncertainty, exact readiness child generation, bounded metadata IO, and the fixed 256-stripe lease lifecycle; see appended plan revision. Scope and explicit-only M2 remain unchanged.

- 2026-10-13 10:21 PDT — TL reports the operator is recovering pair:6. It is no longer a test bed: no further split investigation or process/cache changes there; use scratch setups for legacy/conflict conformance. This changes test location, not the approved design.

- 2026-10-13 09:03 PDT — TL receipt a9307d75-b068-482d-8738-e8d6a344640a
  approves two measured corrections: the cutoff is the launch producing the
  retained tty evidence, not a later empty/recovery helper start; classify
  subagent/tool execution by source/parent metadata, not originator alone.
  The original Spec above retains the dispatched wording as provenance; these
  corrections govern implementation.

- 2026-10-13 09:12 PDT — TL receipts 08e69ac9 and f0c32c35 supersede the
  initial ladder priority and unconditional-confirmation policy. The original
  pair:6 agent remains alive; a second same-name Zellij server claimed the
  operator-facing socket. Prevent split incarnations first, with shared
  ownership observations and a Pair process lock; measure serialization-off
  configuration on isolated scratch sessions before considering a Zellij fork.
  Liveness means the agent the operator can attach to. After safe ownership is
  established, calibrated strong inference auto-selects, close calls ask, and
  resumed native conversation receives tty-tail context. Pair:6 is read-only;
  its live recovery stays with the operator. See the authoritative revision in
  workshop/plans/000528-slot-recovery-plan.md.

- 2026-10-13 09:21 PDT — TL receipt baf2f67b approves no-fork safety:
  serialization disabled for new servers, exact launch-policy attestation,
  Pair-level effect lock, and unsafe legacy/contested attach refused into one
  visible Relaunch action. Optional legacy cache cleanup is a separate machine
  operation with exact paths and operator approval; no deletion is authorized.
  Full implementation design and calibration are in the plan's final revisions.

- 2026-10-13 09:27 PDT — TL receipt bea98157 approves 4cedc4baa, requiring
  expanded independently proven calibration before change-code and a successful
  automatic-resume evidence notice with one fresh-from-previous-screen action.
  The three-pair threshold is provisional until the broader corpus is measured.
  No deletion or live pair:6 recovery is authorized.

- 2026-10-13 09:30 PDT — TL receipt 4c8b05d4 replaces the single correction
  action with ranked resume alternatives (default plus up to four), each with
  evidence, in both slot actions and programmatic candidate listing/selection.
  Alternatives persist after automatic resume. Read-only socket investigation
  confirms the current pathname is a replacement born at 07:55:48; causative
  deletion remains unknown. See plan revision for exact evidence and guards.

- 2026-10-13 09:31 PDT — TL receipt 6f74a1a9 authorizes change-code now:
  M1 includes forward producer witnessing; M2 delivers explicit best-candidate
  and alternatives with UI/CLI evidence. Automatic choice stays OFF pending
  a future recorded witnessed-corpus calibration. No historical hand-labelling.

## Log


- 2026-10-14: closed M1 — Operator accepted round2 restart14:33 on fc145480: Continue auto-attached, pair:1 recovered, all20 wrappers message independently of panes. Pair:1 rev648 empty incarnation is intended graceful dead-helper retirement; regression passes. BR11-13 and15 disposed; BR14 parent warm-only FIFO/context/journal regressions pass. Downstream helper now skips unrelated migration/index/orphan work, bounds local title inference without native lookup, regular draft open/poller cleanup/quit-restart admission; production RunLaunch/runOnce FIFO REDs now GREEN. Final launcher package PASS39.361s, guards4.578s atdb08fb647; prior core334.436s/cmd61.139s verified by round15. M2/M3 published unclaimed #556/#557. Final recorded suite required after settled review and before landing.; review verdict: SHIP
- 2026-10-13 10:57 PDT — Complete make test passed with exit 0; /tmp/pair528-full-suite-final.log contains the shell/packaging checks and go test ./... -count=1 results. Worktree was clean. Final make build is run on efb53841e (production changes through e4e880353); M1 now proceeds to the mandatory boundary review. Hub checkpoint delivery was refused because the recipient is not live; no live staging or recovery was attempted. M2 preview/candidate execution remains a separate unimplemented boundary.

- 2026-10-13 — Final verification continuation: all implementation and fixture fixes are committed through e4e880353. The full cmd/pair-go package passed after correcting its fake terminal admission expectations. A fresh complete make test is running with output at /tmp/pair528-full-suite-final.log; inspect its final exit/result before claiming success. Next: fix any failures, make build on the final committed HEAD, then M1 milestone-close (no review dispatched yet). M2 is still unimplemented and automatic candidate choice must remain off. Read-only M2 seam mapping: reuse sessioninventory stable observation/native normalizers, scrollbackcmd.RenderOwnedCaptureLines, and existing capture readers; an exact witnessed-ordinal native binding lookup must be added inside sessionledger rather than substituting CurrentLaunch or PreviousEstablished. No agents are editing; no live slot or legacy cache cleanup is authorized.

- 2026-10-13 — Additional M1 verification: real scratch Zellij contested cleanup test passed (1.078s), with no effect before authorization and exact-snapshot cleanup afterward. Production cold/warm integration uses real fixture server → wrapper → agent processes and socket, verifies cold policy publication then detached warm attach to the unchanged server/socket/policy/readiness, with one create and one attach. Two independent concurrent runs passed (6.018s/6.021s); globally unique fixture names are required because the production ownership scan is host-wide. Policy fuzz smoke passed 104,989 inputs; targeted admission race suite passed (2.781s).
- The completed second make test run passed core (358.507s), UI (42.071s), launcher (26.002s) and the other reported packages except couchcmd and pair-go. Couchcmd's corrected cold-resume/switch fixtures subsequently passed the full package (145.322s). Pair-go's public command independence fixture still assumes a no-op terminal can pass admission and is being corrected. A separately observed PTY race reproduced 2/20 before a FIFO handshake and passed 20/20 after; its assertion was preserved. Full green suite and milestone review remain outstanding. Latest make build succeeded again on current production code.

- 2026-10-13 — M1 integration checkpoint: make build succeeded in pair:2; not staged for live acceptance. Full launcher suite passed after preserving readiness across nonce preparation, checking live agents across the registry under the admission lease, and rejecting mismatched policy socket identity. Selected-action detached eligibility now shares the same usability rule as inventory. Couchtty full suite passed (32.817s): unsafe terminal rows offer confirmed Relaunch and cannot be overwritten live by the reattach overlay. Artifactpath full suite passed, including explicit GC protection for policy and stable lock files. Policy-retirement test proves unknown records and the coordination inode survive cleanup.
- The first full couchcore run exposed a production regression: the one-primary startup guard still keyed the retired orphan reason, allowing a second primary beside a needs-relaunch row. Extended the guard to all unsafe terminals; the unchanged spawn regression and added metadata-free unsafe-row test now pass. Other failures required explicit positive terminal evidence or ended-session cleanup in fixtures; obsolete automatic orphan recovery expectations now assert held/no effects. Full couchcore rerun is pending in /tmp/pair528-couchcore-integrated-final.log.
- Full make test reached embedded-runtime packaging and stopped because its no-op Zellij client correctly fails the new admission contract. The packaging fixture now expects that refusal while retaining every embedded-asset/path assertion; its isolated rerun passes. Full suite rerun remains required. M1 review is not complete; candidate preview/execution remains M2, and the UI Relaunch entry currently reaches the existing proven-binding lifecycle.

- 2026-10-13 10:21 PDT — Recovery checkpoint after the network interrupted compaction. Change-code passed at 295be1754 (plan-quality CLEAN); M1 implementation is in progress, not accepted. Implemented strict terminal policy and per-socket lease, admission before releasing the lease, exact-owner quiescence without saved-layout deletion, capture generation/launch witnesses, and shared usable-agent liveness. Targeted policy/lease/quiescence/liveness/witness tests passed; storagegc and wrapcmd package suites passed. Full launcher currently fails the selected-root fake's unrecognized pane signature; couchcore full run hit its 60-second timeout and exposed a production-launch fixture that exits before admission. UI Relaunch integration, those fixture corrections, full verification, build and M1 review remain. No M2 automatic choice is enabled. All implementation files checkpointed as work in progress so failed network compaction cannot lose them.
- The isolated real-Zellij argv probe used config serialization=true plus the explicit launch options session-serialization=false; it produced no saved layout, and attach after exit failed without creating a server/socket. Control without the override produced a layout. Evidence: /tmp/p528-argv-2lahzl_h/results.json.

- Measurement: pair:6 scope 50f39cf60172c338, tag 1-pair-42. Root
  01a1333f-ffcb-7760-a1b1-50f52544102d is source=cli, originator=codex-tui,
  parent absent, exact worktree cwd. The 20:15:55 rollout is its subagent,
  not a second interactive root. Ledger rows 17/18 identify the original
  Oct 12 20:13:11.331 launch (ordinal 18, Pair-log offset 0); root begins
  3.426 seconds later. Last activity Oct 13 07:54:11.154 precedes the latest
  recovery helper by 93.180 seconds, but differs from tty mtime by only 15ms.
- Existing RenderBoundedLines with 100 history rows returned 139 rows including
  viewport. Whitespace/Markdown-decoration normalization gives a full final
  assistant-message match: 28 words, 21/21 eight-word windows (100%). Overlay
  measurement passes in 0.902s. No live slot state changed.
- Claude 2.1.296: cwd/entrypoint/isSidechain/message timestamp exist; root cli
  versus sdk-cli and subagents are distinguishable, but cwd may change within
  a conversation. Grok 1.0.50: 42/42 summaries have info.cwd; only 4/42 declare
  session_kind, all headless. Missing kind and relationship=primary do not
  prove interactive. Scanner chronology is first activity, not last; existing
  causal event normalization omits assistant text and cannot be used unchanged
  as a display-tail renderer.
- Liveness audit: inventory trusts hosted/recorded Pair helpers, detached
  projection trusts server/client count, and reattach overlay promotes attach
  completion to live. All three consumers need the same agent-usability fact.

### 2026-10-13
- 2026-10-13: closed M1 — 10 review rounds; zero open findings at e690b4d3; reviewer states the revised M1 contract is implemented. Host evidence recorded at 8667a13e: full couchcore PASS 363.825s, couchcmd PASS 44.835s, launcher PASS 22.019s; reviewer-blocked process checks PASS 6.376s (TestSelectedRootNativeCreateListAndListedResume, TestPreparingReadinessRetainsPreviousAgentEvidence, TestColdLaunchUnknownReadinessRefusesUnderLease, TestColdLaunchAcceptsProvenDeadRetainedAgent), ps -p 1 -o pid=,comm= exit 0; make build PASS. Review environment denies /bin/ps and lacks generated runtime assets. operator-approved --no-judge; ariadne review-environment issue filed. Operator live M1 acceptance remains pending; do not stage until TL schedules it.; review verdict: not-run
- 2026-10-13: closed M2 — BR6 and BR7 disposed by prior review; BR8 adds production recovery parser to existing opt-in live native conformance with bounded reads, explicit skip outcomes and independent authored-tail witness. Targeted tests PASS0.165s and race PASS1.299s; provenance and tail-drop mutations killed; scratch OS fixtures pass. Clean build5be717b9e preceded this conformance-only delta. Installed scan and full suite remain outstanding; TL holds capacity (b70dbf44), previous make test stopped at #514 capture prerequisite before Go. Operator live acceptance outstanding; boundary review requested by TL, no bypasses.; review verdict: SHIP
- 2026-10-13: closed M1 — BR-5 full added-reader sweep uses shared osfs IO; config/capture consumer FIFO, size and cancellation regressions pass; raw-only optional-sidecar fallback and strict oversize refusal pass. Complete make test passed all behavior packages; sole godoc guard fixed and rerun green. Main merged: build 0a0369285, merged-package full tests and DOM integration pass. Logs /tmp/pair528-read-family-final.log, /tmp/pair528-integrated-tests.log, /tmp/pair528-integrated-build.log; targeted race green. No live slot changes.; review verdict: SHIP
- Verified dispatch receipt 65b09489-edf7-445a-8c0d-32e0c934eb62.
  Filled the skeleton from the operator/TL requirements; claim already belongs
  to pair:2. Measurement and design only at this stage.

- Scratch Zellij 0.45.1 measurement: all tested disabling configs still read
  preexisting serialized layouts; live attachment preserves the original
  server. New nonserializing sessions refuse dead attachment. Cache env
  overrides tested did not redirect the macOS cache. All scratch servers
  cleaned. No live Pair process, record, capture or cache was changed.
- Advisory spec review found and the revised design addresses the second
  Pair binding gate, pre-ledger durable target intent, absence-backed capture
  provenance (not a forged park), and the missing historical launch/capture
  join. One advisory pass only; mandatory plan review remains change-code.

- 09:27 PDT — Expanded offline corpus audit and scoring delegated separately:
  labels must come from native binding/park evidence independently of overlap;
  score every same-cwd candidate with a capture-time native event cutoff.
  Production code remains unchanged pending this pre-change-code requirement.

- 09:30 PDT — Read-only ps/lsof/lstat: original PID 41843 and replacement
  PID 9682 remain alive with distinct Unix socket objects under the same name.
  Current socket inode 466359361 was born at Oct 13 07:55:48.344482, matching
  replacement startup. The current entry exists; historical unlink cause is
  not observable from this snapshot. No pair:6 process or socket was changed.

- 09:32 PDT — Expanded provenance audit: 92 unique retained thread records,
  157 parked capture manifests, 294 typed native-binding rows (Claude 211,
  Codex 70, Qoder 3, Muse 10), and 88 readiness records. Owners having both
  binding records and captures: Claude 30, Codex 13, Qoder 2, Muse 4; no Grok
  label. These are evidence counts, not 49 proven capture/conversation pairs.
  Certified labels: **0**. Capture manifests retain raw/events identity and
  timestamp but no producing launch ordinal/nonce; native binding rows prove
  conversation identity independently, without that historical capture join.
  Neither overlap nor temporal proximity was used to fabricate a label.
- Three possible single-native-history cases remain excluded: you-decide13's
  latest resume requests lack binding confirmation; xianxu18 lacks a witness
  linking its one chosen launch to its one capture; 42shots30 includes earlier
  Qoder captures under the same tag. Therefore right-versus-best-wrong margins
  are **not measurable on a certified corpus**, and no expanded automatic
  threshold is justified. This is a pre-change-code evidence limitation.
- Audit manifest/exclusions: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair528-provenance-2hqx1nyh/.
  Offline scorer: /tmp/pair528-offline/ (run.sh, overlay.json, score_test.go,
  scoring-manifest.json, metrics.json, README.md). Uses production VT rendering
  and equal native capture-time timestamp cutoff for every candidate; cutoff
  is explicitly an approximation, not a generation witness. Empty-corpus
  plumbing test PASS (0.118s); it validates no ranking or threshold. No native
  transcript, capture, slot record, process or cache was mutated.

- TL resolved the calibration block: proceed with explicit selection and
  forward witnesses. Proposed activation floor is 20 witnessed pairs per
  enabled agent/parser version across five threads, five held out, all
  same-cwd negatives, zero wrong automatic picks; a reviewed recorded policy
  is required rather than a count-triggered enablement.


### 2026-10-13 11:10 — M1 boundary fixes

M1 review against 2efe103cf returned REWORK: BR-1 unknown evidence promoted to live/relaunch, BR-2 missing exact agent process generation, BR-3 blocking/unbounded metadata reads, BR-4 permanent per-name lock accumulation. Accepted all four and appended the corrected contracts and actual function owners to the durable plan. The implementation now distinguishes unknown/usable/unsafe, pins readiness child identity, uses bounded nonblocking metadata reads, and caps permanent lock inodes at 256 stable stripes. Regression checks include behavioral red for unknown projection/attach overlay and a mutation removing the agent identity comparison; neither defect is waved away by prior suite success. Review artifacts preserved. Focused checks are in progress; M1 remains open and M2 has not begun. No live Pair slots or saved layouts changed.


- 2026-10-13 — Review-fix verification: make build passed on f28a76e82; targeted race checks and full launcher/artifactpath/readiness/wrapcmd packages passed. Complete make test found only the cold/warm production-terminal fixture failures: its synthetic pair-wrap still emitted readiness without AgentIdentity. Updated that producer to observe the real child PID/start identity; both previously failing tests now pass (3.953s, /tmp/pair528-review-fixture.log). Production checks were not weakened. A new complete suite follows this fixture correction.

- 2026-10-13 — M1 review-fix verification complete: make test passed exit 0 on f7783da40 (/tmp/pair528-review-full-final.log); make build passed exit 0 on the same HEAD (/tmp/pair528-review-build-final.log). Targeted race checks passed across launcher/readiness/core/UI (/tmp/pair528-review-race.log). All BR-1 through BR-4 fixes and regression evidence are committed; resubmitting M1 to the mandatory boundary review. No staging or live recovery performed.

- 2026-10-13 11:39 — Round 2 accepted BR-1..BR-4 with independent mutation confirmation, but returned BR-5 (Important): capture-witness events and effective config remained blocking inputs. Audited all new M1 persisted byte readers, moved one shared bounded regular-file reader into osfs, wired the full consumer set and preserved whole-sidecar 8 MiB refusal. Consumer-level FIFO/oversize regressions failed old implementations and now pass; focused launcher/storagegc/osfs/artifactpath checks passed, and full storagegc passed (15.766s). Updated plan with the enumerated consumers/bounds/context limits. M1 remains open for final suite/build and review.

- 2026-10-13 — Whole-suite integration caught a fallback regression from the new shared error type: an optional non-regular events sidecar must still preserve raw-only history without witness certification. InvalidMetadataError now carries a typed violation kind; only non-regular sidecars join the existing unreadable-events fallback, while oversized/malformed authority still refuses. Added missing/directory/FIFO/oversized capture cases; full launcher/storagegc/osfs packages pass. The suite also exposed an existing 30ms test preflight race: caller expiry before observation legitimately makes zero calls. The test retains exact inner-bound/no-effects assertions and adds deterministic expired-parent coverage; no production recovery logic changed.

- 2026-10-13 — Final read-family verification on 25e0de547: make build and targeted race passed; complete make test passed all behavior packages, including couchcore, launcher and storagegc, but its termcmd godoc guard found the InvalidMetadataError comment attached to the newly inserted MetadataViolation declaration. Moved only that comment; the exact guard now passes. Full output /tmp/pair528-read-family-final.log retains the single guard failure transparently; build /tmp/pair528-read-family-final-build.log and race /tmp/pair528-read-family-final-race.log. Integrating current main before the next boundary attempt.

- 2026-10-13 — Integrated main cleanly at 0a0369285 before the noon integration boundary. make build passed on that exact HEAD (/tmp/pair528-integrated-build.log). Full tests for merged artifactpath/couchcmd/couchtty/submission/webcast/termcmd packages and DOM submission integration passed (/tmp/pair528-integrated-tests.log, /tmp/pair528-integrated-dom.log). M1 behavior packages had passed the preceding full make test; its only remaining documentation guard was corrected and passed both targeted and integrated termcmd runs. BR-5 is ready for re-review; M1 still open pending verdict.

### 2026-10-13 — M2 implementation checkpoint

M1 closed SHIP at 3ef0ebf54; TL authorized M2 in receipt b15bc8c6. Parser/scorer, bounded contextual native collector, explicit preview/key codec, schema-3 recovery intent/context, one-shot orientation builder and UI/CLI wiring are implemented in the working tree. Integration is still in progress: archived-source preservation, both launch gates, legacy/live ownership retirement, post-start alternatives and full acceptance have not been verified. Automatic inference remains disabled; no live slot was changed. New witness timestamp and retained-message count bounds are recorded in the plan Revisions. Targeted key codec, orientation, parser/collector, threadrecord, protocol and menu checks have passed independently; no M2 full-suite or boundary claim yet.

M2 integration checkpoint (12:45 PDT): explicit executor and both authority gates now pass targeted recovery tests. Full sessioninventory, couchcmd, couchmessage, couchtty and artifactpath suites passed independently; latest couchtty run (32.105s) includes all pre-existing recovery tests restored unchanged. Capture preservation tests cover same-size/restored-mtime replacement; collector and terminal-retirement mutation checks killed missing ownership/snapshot/cutoff guards. A defaults regression found by integration tests was fixed: successful recovery does not save agent startup preferences. Original tty activity is kept separate from archive publication time (including unknown legacy activity). Terminal adapter and live-source/post-start alternatives tests remain in progress. No full M2 suite, build, boundary acceptance or live acceptance is claimed.

M2 integration follow-up: regression tests now cover live settled-source preservation before park, choosing a second alternative after a successful launch while retaining the original screen, and pending/indeterminate orientation without automatic resend. Legacy terminal recovery is wired and tested to allocate a new terminal name; saved layouts remain untouched. Both DeleteStart rollback branches now retain exact recovery context (red→green), fixing loss after a failed pre-exec launch. Contextual bounded VT replay passes the full scrollbackcmd suite (1.460s); ongoing native-owner guard implementation temporarily makes broad test compilation incomplete, so final suite awaits that atomic change.

M2 verification checkpoint (2026-10-13 12:58 PDT): main 360b2881a merged cleanly at 7efce6fcd. First make test stopped at the unchanged capture-performance prerequisite (12s versus 6s); log /tmp/pair528-m2-full-suite.log. A standalone retry under rising contention measured 19s; no budget override or unrelated performance change. TL receipt 36431188 confirms two ariadne full suites (~12 shards each) and directs no new full suite until uptime load is under 40. Other verification: latest artifact inventory full suite PASS31.775s; final native-owner/core guards and UI/CLI timezone tests pass. Four overlay mutants were killed: skip final candidate validation, skip final capture validation, repeat uncertain orientation polling, and relaunch on uncertain orientation. Reproducer /tmp/pair528-core-mutations.py; existing wrapper partial-write/no-retry and submit-failure tests PASS0.367s. Added permanent raw/events identity mutation cases to the workspace-preparation second-gate regression (PASS1.181s). M2 full suite/build, boundary review and operator acceptance remain outstanding; no live slot changed.

M2 build/race checkpoint (2026-10-13 13:16 PDT): make build PASS; /tmp/pair528-m2-build.log. go version -m confirms pair, couch and pair-launch-helper all embed d485dda0e1a6265e32032e8218fbef37fc0c215f with vcs.modified=false. Serialized targeted race checks passed for couchcore (7.492s), launcher (1.374s), scrollbackcmd (1.241s): /tmp/pair528-m2-targeted-race.log. Host load remained above the TL threshold through 13:16 (latest one-minute 91.64), so no further full suite was started. Waiting for full-suite capacity, then full verification and TL/operator live check before M2 closure.

2026-10-13 — TL receipt 4b2480f6 requests boundary review now if its gate permits, while suite capacity remains unavailable. The milestone-close dry run accepts the prepared evidence and selects the M2 interdiff since M1 rebased onto main360b2881 (95 files), without bypass flags. #514 documents the same unchanged capture-performance failure on loaded main. This is the observed first failure, not proof that the entire full suite has only that failure: make stopped before the Go suite. Review proceeds with this limitation explicit; remaining full verification and operator scratch acceptance still block whole-issue closure/landing.

2026-10-13 13:26 PDT — M2 boundary returned REWORK, BR-6 (wrong latest tag-level launch attributed to old live wrapper) and BR-7 (primary unsafe Relaunch bypasses recovery preview). Both reproduced by reviewer tests; fixes underway as family sweeps recorded in plan Revisions. Review also reports TestProvisionRecoveryMarkerPublicationExcludesCleanup failure in its checkout; reproduce locally before classifying it. Report-to-hub delivery a4d8ef36-7bbb-4e19-b7a9-0d750e53407e was uncertain because the supervisor socket disappeared; receipt lookup also failed, so no automatic resend. No full suite or live action started.

2026-10-13 13:33 PDT — BR-6/BR-7 corrections verified. Native ownership now joins the observed wrapper to the exact live readiness agent PID/start and launch ordinal, refusing missing or changed association instead of borrowing the latest tag-level launch; source exemptions and the stateful fake use the same rule. Unsafe Enter, mouse, explicit Relaunch and restart shortcuts now share the recovery chooser, preserving healthy relaunch confirmation. Serialized targeted race tests PASS (couchcore 8.516s, launcher 1.309s; /tmp/pair528-m2-review-fixes-race.log), full couchtty PASS43.893s (/tmp/pair528-m2-br7-couchtty.log), diff check clean. The exact-generation regression kills the prior latest-launch mutation. Review's marker-publication test passes locally isolated and count=3; no unrelated fix made. No new full suite started; #514 remains the observed prerequisite failure and the rest remains unverified pending TL capacity. Resubmit fixes for boundary review; operator live acceptance still outstanding.

2026-10-13 13:40 PDT — Second M2 boundary REWORK: BR-6 and BR-7 addressed, independently mutation-tested by the reviewer; new Important BR-8 requests recovery-specific native conformance. Existing conformance indeed calls legacy scanners only. Extend that read-only path and add drift regression per plan revision. Build at 5be717b9e passed with all three binaries vcs.modified=false. Reviewer targeted suites passed; full-suite and live acceptance still outstanding.

2026-10-13 — BR-8 verification: existing opt-in native-shape test now calls bounded production recovery parsing via RunRecoveryConformance, with redacted provenance/cwd/activity/authored-tail coverage and explicit unsupported/absent outcomes. Independent native-event normalization detects lost authored tails instead of silently skipping them. Drift, vanished-sample, skip outcomes and scratch real-OS reads pass targeted tests (0.165s) and targeted race (1.299s). Provenance-skip and dropped-authored-tail mutations both fail the regressions. Installed native scan was not run to preserve the live-slot investigation restriction; scratch files only. TL b70dbf44 still holds full-suite capacity at load274. No full suite or live slot action started.

### 2026-10-13 — Reviewer flake retained for full-suite follow-up

The final M2 reviewer observed unchanged `TestProvisionRecoveryLostSuccessPublication` fail in its broader couchcore race selection before the intended injection (`weave=0`); an isolated race rerun passed. This is a recorded intermittent failure, not evidence that the broader run passed or that its cause is established. The SHIP review and evidence are in the M2 review sidecar (2026-10-13 13:49 PDT). Per TL receipt 03a3a089-1fce-424f-9bb7-f55e62430275, file a separate issue if this failure recurs in our full run. Full-suite execution remains queued for the TL's capacity signal; no suite was started for this note.

### 2026-10-13 — Full verification and transition guard correction

Per TL 4e054fd4, `make -k test` entered the machine gate and acquired after 312s. All Make prerequisites passed, including the previously load-sensitive capture budget. The complete Go run reached every package and failed only `TestArbitraryLifecycleMutationHasNoDoor`: ResumeCandidate directly supplied a callback to updateExistingThread. Log: /tmp/pair528-full-20261013-final.log. The recorded TestProvisionRecoveryLostSuccessPublication flake did not recur, so no follow-up filed. Corrected the authority boundary with typed ThreadStore.PreserveRecoveryBeforeRetirement, validating the selected intent and retaining clone/CAS semantics. The unchanged architecture guard plus all TestResumeCandidate cases and DeleteStart recovery checks pass (4.117s, /tmp/pair528-transition-fix.log); diff check clean. This is a failed full run followed by passing focused correction verification, not a claimed green rerun. Staging follows for the TL's scratch live acceptance before close.

2026-10-13 — `sdlc stage --build` succeeded on pair:0 at e1876500ec010e3d2fc59e8361d6641798ed701b; pair:2 retains the issue branch. couch, pair, pair-launch-helper and pair-slug embed that revision; the stage command warns auxiliary pair-hoprtt remains 2284b412 (modified). TL handoff fd62e544-2de3-4f84-b571-e2435422cb28 names the exact stage, full-run failure/focused correction evidence, scratch acceptance procedure in the durable plan and compile-checked policy fixture source. Awaiting TL/operator acceptance after their Couch restart; no scratch/live mutation executed here and whole-issue close not yet run.

2026-10-13 — Emergency rollback requested after staged policy admission hid ~20 legacy live slots and blocked Couch startup. sdlc stage --clear found pair:0 already on main with no stage. make build there passed; couch/pair/pair-launch-helper each embed local main9ea3bdcbdcd360324337d910ed419dc6a65443db with vcs.modified=false, clean checkout. Local main is270 commits behind origin/main; no pull performed during urgent rollback. TL notified f1aa0ef3. Appended operator-directed live-socket admission re-plan; no code change/restage.

2026-10-13 — Recorded operator replacement design from verified TL receipt15876eb2. This is plan-only: no code, test process, layout cleanup, rebuild or restage. pair:0 remains on main after prior rollback. The list→attach race is explicitly unresolved for old saved sessions pending scratch measurement after approval.

- 2026-10-13 20:20 PDT — PLAN ONLY, operator frame via TL receipt 70291ee7-93d3-4b62-b027-9f8162a08357. Appended code-verified state manifest and ordered reconciliation table to the durable plan: exactly Continue / Resume from disk / Start fresh. Audited registry/thread journals, park/start, channels, process generations, native/tty evidence and Zellij. Found ordinary thread reads can replay journals; startup currently reconciles interrupted starts and parks; candidate reads acquire retention leases. These cannot be hidden inside observation-only startup. Existing broker reconnect works, but an unlinked wrapper listener cannot be recreated by old wrappers; Continue by terminal remains valid. Installed Zellij reports 0.45.1; matching upstream source shows list-sessions can unlink connection-refused stale sockets and attach can resurrect legacy saved layouts after a list-to-attach death race. Both explicitly surfaced for operator review, not silently “fixed.” No implementation, build, test, stage or live recovery performed. Plan also closes missing-tty fallback and identity-bound live-stop confirmation gaps. Awaiting operator/TL approval. One fresh-eyes editorial pass identified two gaps, both folded into the plan: effect-time usable-agent evidence for Continue, and exact confirmation keys across proven, inferred and fresh recovery routes.

- 2026-10-13 — Added the operator-approved slot-continuity target verbatim from ops draft, per verified TL receipt 7a3c544d-15c1-4c7a-b01c-5f36479f40f7; linked issue frontmatter and atlas index. Checked the manifest/table against every commitment and appended a target-conformance review to the plan. Conflicts include indefinite pending/retry states and complete-scan prerequisites for Fresh, the proposed native listing write exception, and the old-wrapper reconnect gap; hub authority is now explicit. Target copied byte-for-byte; no implementation authorized or changed.

- 2026-10-13 — Verified TL decision cb59774f-021f-480a-8d44-a7a344416088 and appended revised plan: one bounded observation retry then Continue / Resume / Fresh, Fresh always offered, unknown/foreign remnants left intact with fresh-name fallback, person-or-hub confirmation. Native listing cleanup, legacy attach death race and old-wrapper messaging limitation explicitly accepted. Replaced execution sequence with three separately reviewed/operator-accepted milestones: M1 startup/attach/projection; M2 recovery execution; M3 endpoint repair. Historical M1/M2 closes do not satisfy these revised boundaries. Documentation only; awaiting TL confirmation of revised plan before M1 code.

- 2026-10-13 — TL confirmed plan 1de34a1a3 in receipt e09b7c38-e67f-4961-906e-6419ec146e81, authorizing M1 only and no staging. Added explicit M1 constraints: failed listing never Continue; attributed live transcript owner never executable Resume. change-code passed (plan-quality CLEAN, estimate-quality INFO) at 303756eba. Estimate advisory says original derivation covers previous scope; keep it as historical whole-issue baseline, reconcile replacement/M3 increment before the next milestone rather than presenting it as remaining effort. Active Plan rows now represent the replacement M1/M2/M3; previous closes/review evidence remain in Log/sidecars. TDD implementation split by source ownership (startup; terminal admission; projection/menu); full suite and boundary review remain outstanding.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against
baseline-v3.1.md. Method A only; source marked stale by estimate-source, so
calibration is provisional. Derived after plan-quality CLEAN (round 2).

Decomposition: issue/spec (1); greenfield single concerns (3: terminal
admission, capture witnessing, recovery ranking); extensions (6: three native
parsers, policy schema, recovery persistence, orientation); recovery executor
and CLI integration (1); recovery TUI (1); shared liveness refactor (1); atlas
(1); M1/M2/final review boundaries (3). Existing unix.Flock, atomicfile,
VT renderer, native parsing, start transactions and retention cover standard
mechanics; no library supplies Pair's novel authority relationships, so no
additional library discount. Thorough-plan design discount x0.2 on code
primitives; familiar Go stack x1.0. Implementation values are table values
already scaled x0.4, not scaled again. Issue shaping design stays 1.0.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=1.0 impl=0.08
item: greenfield-go-module design=0.2 impl=0.24
item: greenfield-go-module design=0.2 impl=0.24
item: greenfield-go-module design=0.2 impl=0.24
item: smaller-go-module design=0.04 impl=0.16
item: smaller-go-module design=0.04 impl=0.16
item: smaller-go-module design=0.04 impl=0.16
item: smaller-go-module design=0.04 impl=0.16
item: smaller-go-module design=0.04 impl=0.16
item: smaller-go-module design=0.04 impl=0.16
item: api-integration design=0.4 impl=0.48
item: tui-screen design=0.2 impl=0.32
item: cross-cutting-refactor design=0.1 impl=0.16
item: atlas-docs design=0.05 impl=0.06
item: milestone-review design=0.04 impl=0.16
item: milestone-review design=0.03 impl=0.16
item: milestone-review design=0.03 impl=0.16
design-buffer: 0.15
total: 6.3535
```

Design subtotal 2.69 x1.15 = 3.0935; implementation subtotal 3.26;
central estimate 6.3535 focused ship-hours. Main uncertainty is integration
across attach/create and existing recovery admission, not future calibration
collection (automatic enabling is deferred).

## Revisions — 2026-10-13 20:20 PDT

The operator's three-outcome frame supersedes incompatible policy-based and narrow attach revisions. The current design and source-verified state manifest/reconciliation table are the final appendix of `workshop/plans/000528-slot-recovery-plan.md`. Startup observes slots without recovery effects and never blocks Couch; broken messaging does not justify teardown of an attachable live agent. Recovery requires evidence, preserved context and explicit confirmation before stopping a running agent. This is a plan for operator/TL review, not authorization to implement.

## Revisions — 2026-10-13 operator-approved recovery decisions

The final revision in `workshop/plans/000528-slot-recovery-plan.md` supersedes pending-only projection and the prior gap decisions. Its three unchecked milestones replace the implementation sequence for remaining work; prior checked milestones/review records are historical. TL confirmation of this revised plan is required before M1 implementation; each revised milestone needs its own operator acceptance session.

### 2026-10-13 — Revised M1 implementation checkpoint

- Implementing only the confirmed replacement M1 (plan 1de34a1a3; change-code checkpoint 303756eba). Startup no longer recovers interrupted starts, active parks or journals, provisions slots, or automatically reattaches. Preview inventory and explicit listed-live attach replace session-policy authority; the existing attach lock and serialization-off launch remain.
- Targeted startup, launcher and three-outcome projection regressions pass. Candidate execution now refuses a live/unproven attributed source before lifecycle effects; incomplete candidate evidence retains Start fresh. Menu tests exercise primary Fresh confirmation and ordinary Continue. Broader package tests are in progress; this is a checkpoint, not a completed milestone.
- Accepted legacy race measured with installed Zellij 0.45.1 on isolated scratch names/HOME/config/socket root /tmp/z528-4yd9qd4d. Legacy session z528-fa3324e10f-legacy was listed live with a saved layout, then its exact server (1710) was killed; plain attach resurrected server 4020. Session z528-2e515bbb9a-off was created with options --session-serialization false, listed live, and had no saved layout after two serialization intervals; after exact server 4412 died, plain attach exited 1 ('No session with the name ... found!') and created no replacement server. Only scratch owners were stopped; final ownership scan found no scratch servers/panes. Script /tmp/pair528-zellij-race.py and report /tmp/z528-4yd9qd4d/report.json retain raw measurement. This confirms the documented accepted legacy list-to-attach race and the no-layout behavior for new servers.
- No staging: operator acceptance is separately scheduled by the hub after the M1 boundary review.

- M1 integration: path-based candidate previews previously entered workspace Ensure/enrollment; new read-only resolution plus byte-snapshot regressions prove previews neither repair an existing slot nor create an absent store. A failed listing at the observation deadline now retains known rows and projects Fresh without claiming Continue. Existing global session-name indexes disambiguate same-name claims across scopes before attach; no custom socket probe or policy authority returns.
- Milestone execution limit: the new Fresh projection is visible on busy/unknown/needs-relaunch rows, while existing destructive reboot guards remain closed. Replacement M2 supplies the person-or-hub confirmation/retirement execution for these cases. The action-agreement regression explicitly verifies these exact M1 guards remain closed, rather than widening stop authority to make the new offer executable prematurely. Sent this phase-boundary clarification to hub (receipt 5d077934-f8ae-4ab0-921f-87f58d5cbe05); acceptance staging remains withheld.
- Broader core run exposed obsolete startup/projection expectations, a real cross-scope terminal claim gap, and missing transition refusal coverage; targeted reruns of all failures pass (4.294s), including timeout preservation. Removed dead startup layout-conflict helpers and the obsolete policy fixture; dead-symbol guard passes. Old wrapper endpoint dial failures now name messaging unavailable until next relaunch, preserving underlying errors and Continue by terminal.

- First full replacement-M1 make -k test completed: shell/editor checks and most Go packages passed; failures were old startup/attach fixture expectations (corrected and targeted green), the live fleet's new staged reason (already fixed by landed #539; integrate main rather than copy it), and TestSpawnTimerMeasuresTheCommandNotTheHarness (19.524 ms under suite load; isolated rerun passed2.861s). Corrected installed-command + full couchcmd rerun passed4.062s/76.909s. Full suite log /tmp/pair528-m1-full.log; next run will include main's fleet compatibility.

- 2026-10-13T21:34:22-0700: TL corrected the M1 action boundary (receipts 5e7063f1 and 46c487bc). Removed guaranteed-refused Fresh offers and their test exception; uncertain rows retain working Relaunch/recovery preview, while ineligible native bindings remain evidence rather than selectable resume actions. Positive live listing always gives Continue. The rare nonlisted unfinished StartClaim displays Starting with read-only details; M2 must bound its lifetime and add confirmed retirement. Appended plan Revision and interrupted the stale-window boundary review; no milestone finalized.

- 2026-10-13 21:48 PDT — M1 review of 0d4565a9 returned REWORK: BR-9 found
  listed-live Continue still hitting cold profile/path executor gates; BR-10
  requested README alignment. Fixing both before rerunning the boundary.
  The post-main full `make -k test` completed with only the old menu action
  sweep failing; the corrected sweep and actual blank-key Fresh admission edge
  passed the full couchtty package (40.867s). Build passed. Evidence:
  /tmp/pair528-m1-full-after-main.log and /tmp/pair528-m1-ui-sweep-fixed.log.

- 2026-10-13 21:54 PDT — BR-9 correction separates exact warm attach from
  cold launch prerequisites throughout core and Pair. Warm claims carry no
  native profile; unavailable original cwd uses the existing host root only
  for the helper, preserving the saved path and hosted artifact scope. Exact
  terminal ownership checks and the final positive listing remain. Focused
  independent rerun passed core 1.816s / launcher 0.107s; tests cover four saved
  metadata/path cases, unchanged cold refusal matrix, changed-owner refusal,
  and unchanged persisted profile/path. Existing public-binary harness now
  covers neutral-cwd scope routing and wrong hosted scope/name refusals.
  Full suite running at /tmp/pair528-m1-br9-full.log; no staging.

- 2026-10-13 21:56 PDT — Broader warm-resume tests caught
  an existing continuation receipt's need for retained incarnation agent
  metadata. Correction to the preceding note: a warm claim retains the actual
  optional saved profile when present; it never requires or invents one and
  sends no cold launch profile to Pair. Successful warm registration skips
  default/preference edits. The continuation regression plus warm executor and
  cold eligibility matrix passed independently (1.583s). No new state/schema.

- 2026-10-13 21:57 PDT — Full suite caught the new
  direct artifactpath import in the command entry point. Reused existing
  NewScopedPaths.Validate/ScopeDir instead of widening the artifact manifest;
  ownership check passed (1.151s) and public real-binary attach regression
  passed (4.839s). Full-suite remainder still running.

- 2026-10-13 22:03 PDT — Full `make -k test` completed
  with exactly two failures: artifact ownership (corrected and rerun above)
  and legacy slot subdirectory root presentation. Restored shared checkout
  normalization as optional metadata for warm attachment; cold failures still
  refuse. Added nil path-observer coverage. Targeted legacy/warm/continuation
  tests passed (7.135s); full affected core/artifactpath packages are rerunning
  at /tmp/pair528-m1-final-packages.log. The full run passed couchcmd 34.926s,
  couchtty 39.185s, launcher 21.032s and wrapcmd 40.136s. No staging.

- 2026-10-13 22:08 PDT — Final affected-package rerun
  completed exit 0: couchcore 284.517s, artifactpath 14.262s
  (/tmp/pair528-m1-final-packages.log). Combined with the completed full suite
  and corrected public-entry regression, all observed failures are resolved.
  Final `make build` passed (/tmp/pair528-m1-final-build2.log) at
  259a826bebe5d7e0393becd761102735baced345. Rerunning the corrected M1 boundary;
  operator acceptance remains pending and nothing is staged.

- 2026-10-13 22:14 PDT — Boundary round 8 REWORK:
  BR-10 addressed and BR-9 metadata/path regressions verified, but a positively
  listed StartClaim still reached lifecycle cleanup and refused. Recorded the
  class-level correction in plan Revisions: public helper-only attach preserves
  thread records; explicit continuation retains its adoption path. The prior
  full suite and affected-package green results remain evidence for that head,
  not a pass for the forthcoming correction. No staging or M2 work.

- 2026-10-13 22:24 PDT — BR-9 executor shape sweep
  now covers all 13 accepted listed-live fixtures across live/dead/unknown
  owners, immutable records, held foreign lifecycle operations, and exact
  helper-only cleanup. Old production fails the shape sweep. Independent
  focused core/menu/artifact checks passed (3.863s/0.236s/2.364s). Console
  regressions cover actual local-pane reuse and listed row without local pane
  (warm-only fallback), red before fix and green after (0.354s). Explicit
  lifecycle compatibility and upstream resume routing remain in progress.

- 2026-10-13 22:36 PDT — Core helper-only progress checkpointed at
  d2360a4f0a8a0838b16fc3dbe4d618b00d054f39; integration verification pending.
  New production selected-list tests pass (0.848s), including attached clients,
  no owner metadata, recorded-name precedence and failed/ambiguous listing.
  Independent combined run caught stale detached-query fault-injection hooks;
  migrate those tests to the actual listing seam. Inspection also found hidden
  owner-probe requirements after helper acknowledgement and inside Pair attach;
  remove them only from Continue, preserving lifecycle ownership requirements.
  No stage or live-slot mutation.

- 2026-10-13 22:39 PDT — Final Continue correction checkpointed: core ec74f9061,
  launcher 6bc51034ff5ef683cb84656d3fa7c2793a81b70a. Production checker
  regressions first reproduced post-ack owner-probe refusal and false success
  for an exited helper; both pass after the correction. Existing tracked
  adoption failure coverage is retained on its private route. Actual public
  Continue → Park → cold resume preserves the conversation identity and creates
  the next terminal identity. Independent combined attachment-path tests pass:
  core 17.740s, console 0.317s, launcher 1.062s, artifact inventory 2.298s
  (/tmp/pair528-final-attach-path.log). Startup/details regressions pass
  (core 3.012s, console 0.200s). make build passed
  (/tmp/pair528-m1-continue-build.log). Full make -k test is still running;
  no final suite or review claim yet. No staging.

- 2026-10-13 22:53 PDT — Full make -k test completed exit 2
  (/tmp/pair528-m1-continue-full.log), with failures confined to couchcore,
  couchcmd and launcher. Corrections preserve existing continuation publication
  and execution after nonowning Continue, missing-slot adoption routing, explicit
  retention use and recovery-report convergence. Old warm-adoption assertions
  now verify unchanged records; explicit lifecycle ownership tests remain.
  Targeted core continuation passes 27.000s, command continuation 18.273s,
  routing/switch 28.245s, recovery menu 5.595s, retention/report 48.937s, and
  restored-registry ownership 1.183s. Complete launcher package passes 31.589s.
  Final affected-package rerun is /tmp/pair528-m1-corrected-packages.log; the
  corrected make build already passed. No new failure is attributed to load.

- 2026-10-13 22:55 PDT — Final affected-package rerun completed:
  couchcmd passed 44.835s, launcher passed 22.019s. All prior M1 regression
  failures cleared in couchcore; its 296.305s full run failed only
  TestOpenRefusesWithTheHandoffWhenSetupCannotConverge and
  TestReconcileAgreesWithRealGit/branch=broken/elsewhere with
  `workspace host creation is busy`. Both passed unchanged together in isolation
  (20.384s, /tmp/pair528-isolated-workspace-failures.log). The unchanged
  provision_lock_unix.go documents #436's inherited-descriptor fork/exec window
  and 100ms settle bound; that is a plausible cause, not a measured attribution
  for these two runs. No timeout was raised and no failure is hidden or claimed
  as a clean full-suite pass. Final build passed; git diff --check is clean.
  Rerun the corrected M1 boundary with this exact evidence. Operator acceptance
  remains pending, with no staging and no M2 implementation.

- 2026-10-13 23:06 PDT — Boundary round 9 at
  73f877e9121fb3eb5773cac06b2798c21e7f1c93: all code findings addressed,
  zero new findings, but REWORK solely for verification environment: the fenced
  reviewer could not execute /bin/ps. No code or design change was requested.
  Host reruns of TestSelectedRootNativeCreateListAndListedResume pass 5.972s;
  TestPreparingReadinessRetainsPreviousAgentEvidence,
  TestColdLaunchUnknownReadinessRefusesUnderLease and
  TestColdLaunchAcceptsProvenDeadRetainedAgent pass together 0.745s.
  A direct ps read succeeds here. Logs are
  /tmp/pair528-review-host-process-verification.log and
  /tmp/pair528-review-retained-process-verification.log. The milestone is not
  finalized; no gate bypass or review-fence/config change was made. Report the
  environment blocker to TL for a process-capable review. A final full-core
  confirmation run is still in progress. No staging.

### 2026-10-13 23:09 PDT — Round 9 environment counter-measurement

At HEAD `ebc09ad6b143d334fd8900bb15e05369edfbbb56`, reran unsandboxed
on this machine (unchanged production code from reviewed 73f877e9):

```sh
go test ./cmd/internal/launcher -run '^TestSelectedRootNativeCreateListAndListedResume$|^TestPreparingReadinessRetainsPreviousAgentEvidence$|^TestColdLaunchUnknownReadinessRefusesUnderLease$|^TestColdLaunchAcceptsProvenDeadRetainedAgent$' -count=1
```

Exit **0**, launcher **6.376s**, complete output:
`ok github.com/xianxu/pair/cmd/internal/launcher 6.376s`. Log:
`/tmp/pair528-ebc09ad6-host-countermeasure.log`. A direct
`ps -p 1 -o pid=,comm=` also exited **0**, returning `1 /sbin/launchd`.
The process-observation restriction is specific to the fenced review, not
the tested production behavior or the host verification environment.

The round issued **no finding ID** for its environment blocker and disposed
all BR findings. `sdlc finding dispute --help` refuses settled or unissued IDs,
so no ID is fabricated and the generated ledger is not edited by hand.
TL approved recording this counter-measurement and re-closing normally; if
the next round returns only the same sandbox blocker with zero findings,
report and stop for the operator rather than loop. No gate waiver.

Final full-core confirmation: `go test ./cmd/internal/couchcore -count=1`
exited **0**, **363.825s** (/tmp/pair528-core-final-confirmation.log), without
changes or raised deadlines. Together with the corrected full command and
launcher package runs, every failed package from make -k test has now passed
in full. Earlier transient workspace-lock failures remain recorded above.

- 2026-10-13 23:13 PDT — Boundary round 10 at
  e690b4d30098c471422f0e0254563c9af78717f6 again returns REWORK with
  **zero new/open code findings**. It explicitly finds the revised M1 contract
  implemented, but refuses verification because its fence denies /bin/ps and
  its disposable checkout lacks generated runtimebundle/assets/runtime/files
  needed to build console/public-binary tests. The host counter-measurement
  was read but the reviewer says it cannot replace its own blocked inspection.
  No design or production correction requested. Per TL's verified instruction
  1d3d9b89-19fc-4a04-97c6-94c3bd22e43e, report this environment-only blocker
  and stop rather than loop or waive gates. All failed packages from the host
  full make run now pass in full; M1 is still not finalized or accepted.
  No staging, no M2 implementation, and no review configuration changes.

### 2026-10-14 — main integration before operator M1 acceptance

Refreshed on main-slot2: ariadne substrate 407e48a39057503fab7cdecaf6c69ce338ce7631;
returned to #528 and merged origin/main 962f3c0478ef4b46766f6e29b4f28092b909735f.
Conflict files: README.md; atlas/couch.md; couchcmd message_service.go, messages.go,
run.go, run_test.go, skills/couch/SKILL.md, slot_operations.go; couchcore
continuation_conformance_live_test.go, recovery_conformance_live_test.go,
transition_refusal_test.go; couchmessage operation.go/protocol.go; couchtty
action_agreement_test.go, console_menu.go, menu.go, menu_actions.go,
menu_reattach_test.go, menu_switchagent.go; launcher args.go, createflow.go,
fresh_nonce_test.go, launch_args_policy.go; readiness/record.go; wrapcmd/wrap.go.

Kept #546 hub ownership, scoped routing/guards and queued extra-hub operations,
including candidate guards at queue preparation and effect. Kept #528 positive
listed-live Continue, immutable lifecycle records for borrowed attachments,
recovery candidates/evidence and policy removal. Adopted #522 Prompt and its
legacy decode-only tombstone; removed recovery-only delivery/status APIs. Fresh
starts point at retained evidence; native resumes receive no prompt. Exact
readiness agent identity and recovery authority/attempt nonce remain pinned.

TL clarified hubs are explicit startup requests (including bare couch cwd);
only non-hub slots are preview-only. No per-slot startup reattach/recovery sweep.
Adjusted native-live fixtures to retain identity without delivery callbacks;
removed a stale test for a transition main deleted. Next boundary covers this
merge. Targeted candidate/startup/attach/launcher tests pass (core 11.338s,
launcher 7.509s); orientation/menu and candidate hub-guard tests pass. Full
local-check suite and staging remain pending at this checkpoint.

Merge regression checks: selected-hub tests pass (22.795s), including proven
cold resume, legacy live attach without saved launch profile/layout, fresh
new-tag startup beside an unchanged orphan, and unrelated retained slots left
untouched. Multi-hub focus regression initially failed because its first fake
hub child registered without publishing Zellij liveness: inventory showed ops
unusable/needs-relaunch, and Escape truthfully reported no live focus target.
The fixture now publishes the exact acknowledged session’s positive listing;
production projection remains unchanged. Added candidate mutation guard tests
for both queue preparation and effect. Full suite follows this merge checkpoint.

Full local-check suite on 1c1c3b5ce395a8260885a624299728bf682b7bfd acquired
the machine gate after 835s and ran for 12m4s, exit 2. All shell/editor checks
and all Go packages except cmd/couch and couchcore passed. Failures were
TestBareCouchInstalledCommand/pty_observes_current_directory_without_launching
(old preview-only startup fixture with a Pair stub that never acknowledges the
launch helper), and TestExplicitOpenRepairsPartialSlotWithoutCreatingConversation
(implicit hub fallback incorrectly widened explicit StartOpen). Full log:
/Users/xianxu/workspace/pair/.git/sdlc/local-checks/1c1c3b5ce395a8260885a624299728bf682b7bfd/suite.log.

Restored explicit action admission: args.Action != empty delegates to the existing
managed operation; only implicit hub startup selects fresh when no native proof
exists. The unchanged explicit-open regression and new hub cases pass (6.363s).
Installed public-command smoke is being adapted to actual hub launch/attach.
No staging; another full local-check pass is required.

Installed-command smoke now uses real Pair launch/registration and the existing
stateful terminal fixture (socket/server, wrapper/agent, readiness). It asserts
exactly one hub launch and attached PTY output, waits for readiness before
cancellation using the original bounded deadline, and keeps pipe refusal.
Targeted cmd/couch run passes (6.416s); explicit-open + hub regressions pass
(6.363s). The next full local-check includes both corrections.

Second local-check on 9988efdc72af3c9c19d33fe09549beb112e4f9a1 acquired
the gate after 1772s and ran 10m36s, exit 2. Installed smoke passes 8.860s;
full core passes 336.992s; launcher passes 24.093s. Warm recovery acceptance
failed its no-native-query assertion: its fake SetDetachedSession(empty) deletes
positive session presence, while SetPairSession restores only the binding. This
models false absence after acknowledgement. A synchronous inventory probe at
that point reproduces the original failure (3.08s). Preserving the surviving
session’s presence fixes it; probe retained and assertions unchanged. Warm
count10 passes 30.332s, reboot passes 5.062s; production unchanged.

The run also reports TestDriveActionShutdownJoinsAndRejectsClicks (2.05s),
console_drive_test.go:616, shutdown did not finish after writer settled. Targeted
repeat investigation is pending; no timeout adjustment or assumed runtime fix.
Log: /Users/xianxu/workspace/pair/.git/sdlc/local-checks/9988efdc72af3c9c19d33fe09549beb112e4f9a1/suite.log.

Independent shutdown check: go test ./cmd/internal/couchtty -run
'^TestDriveActionShutdownJoinsAndRejectsClicks$' -count=10 -v reproduces once
(first iteration 2.14s), then nine passes (0.05–0.08s), package 3.052s.
Drive implementation/test, teardown, cast and host code are byte-unchanged
from origin/main; #528’s console changes are outside this path. The one-second
completion assertion covers cast/HTTP shutdown as well as writer acknowledgement.
The exact blocking stack is not yet measured; do not attribute a root cause
or increase a timeout. Another full run will include the warm-fixture fix.

### 2026-10-14 — approved final M1 acceptance delta

Recorded TL decisions 379f4f5d/4ad3f4a8/b6a98c4d in the plan and atlas:
explicit attachment retry, pane-independent authenticated broker admission,
and guarded replacement of a proven-dead client helper only. Real pair:1 stays
untouched for operator acceptance. Runtime changes need a new final-head suite;
queued local check 5594eb85 does not cover them.

Shutdown counter-measurement: temporary stack probe (restored afterwards),
go test ./cmd/internal/couchtty -run '^TestDriveActionShutdownJoinsAndRejectsClicks$'
-count=10 -v, reproduced one 2.04s failure and nine passes. At one second,
Console.teardown waited in endWebcastForShutdown -> Cast.End -> Mount.Unmount
-> Host.beginCloseLocked -> net/http.Server.Shutdown; the writer ACK worker had
already exited. Existing HTTP shutdown budget is two seconds; this test waits
one second. Log /tmp/pair528-drive-shutdown-probe-2.log. This is unchanged-main
HTTP shutdown evidence, not an ACK regression; no timeout or production edits.

Third local-check attempt pinned 5594eb85a12fec01e8edb73f57082d45d2987b5f
was still waiting at 1173s, with no suite execution, when the approved M1 runtime
delta superseded it. Cancelled the exact owned local-check process 16430
(SIGINT, exit 130); process cwd verified as pair-slot2/pair and no child/check
process for worktree sdlc-local-4103953598 remained. No test-lock manipulation.
Final runtime head will receive a new recorded suite. Parent independently ran
the subprocess reconnect and resolver regressions: couchcmd 1.347s, exit 0.

### 2026-10-14 — final M1 runtime regression evidence before recorded suite

Parent verification of broker/slot authority group: go test
./cmd/internal/couchmessage ./cmd/internal/couchcmd -run
'Test(Message|Lifecycle|Hub|SlotOperation|CandidateResume|ResumeCandidates|Notify|SessionWrite|SessionSync|Supervision)'
-count=1 -timeout=120s passes (message 0.177s, cmd 13.632s). Real subprocess
wrappers reconnect and exchange via actual Unix endpoints without pane events.
Production preserves process/ready/launch/workspace proofs and uses exact native
listing at use time; a detached client alone no longer disconnects its wrapper.

Parent core verification: go test ./cmd/internal/couchcore -run
'Test(WarmAttachment|WarmListing|DeadClientRepair|PublicWarmAttach|AttachmentNotice|StartupAttaches)'
-count=1 -timeout=90s passes 20.051s. Includes narrow dead-client replacement,
open claims and current parked metadata unchanged, producing launch metadata
and closed history preserved, changed proof at atomic publication, missing
store reported without creation, bounded read/helper retries and cancellation.
The shape matrix uses independent explicit expected outcomes. An earlier new
fixture wrongly combined verified-current-park with a live incarnation; it was
corrected into closed-history preservation and valid current-park borrowing,
without weakening record validation. First-row Enter/click via real operation
dispatch passes 1.097s; exhausted attempts show a visible reason. No background
reattach sweep and no real pair:1 writes. Ephemeral attachment notices are
excluded from persisted registry JSON and surfaced in console/hub receipts.

Initial hub attachment uses the same ephemeral warning, queued through the
existing standing console notice so it survives slow startup until first paint.
Parent TestInitialHubBorrowedAttachmentNoticeAppearsOnFirstPaint passes 0.314s;
failed initial attachment shows no success/borrowed warning. It does not arm
background attachment. Final runtime checkpoint follows; full recorded suite
remains required and acceptance remains with the operator/hub.

### 2026-10-14 — recorded final suite integration corrections

Local-check on ef04492b462c60cefffd1b60a95822cde3d1b350 entered after
1581s queued. Shell/editor/runtime integrations passed, including capture
performance (5s against 6s budget). Installed Couch smoke passed 8.503s.
The Go phase found two integration omissions: exhaustive artifact source
inventory lacked warm_attach_repair.go, and the real-helper-death warm-menu
acceptance still asserted unconditional record equality. Classified the file
as NonArtifactSources alongside warm_attach.go/threadstore.go: it constructs
no artifact filename and uses existing ThreadStore-owned paths. The ratchet
exception list is unchanged. Its exact classification check passes 4.507s.
Warm acceptance now independently permits only helper PID/Identity/State and
Revision replacement after the fixture's actual old-process death proof,
requiring all other metadata and the exact terminal unchanged. Whole actual
helper-death acceptance passes 9.825s. No production behavior changed. The
recorded suite's remaining packages are still running; a corrected-head full
check follows its result.

### 2026-10-14 — final recorded suite passed; staging awaits pair:0 release

The ef04492b recorded run finished exit 2 in 11m22s, with only the two
integration omissions above. Core 405.571s, tty 48.294s, message 7.273s,
launcher 36.667s and wrapper 38.127s passed; the independent shutdown flake
did not recur.

Corrected final runtime/test head 35ec94d0169cc2438dcefb70c378a31545bce0ed:
sdlc local-check run --name suite PASSED exit 0 in 8m30s, recorded as
sdlc-local/suite, log sha256 prefix ed69b6e867e0. Complete shell/editor checks
and go test ./... passed. Core 252.238s, cmd 44.431s, tty 35.789s, message
6.962s, launcher 26.913s, wrapper 40.935s, installed Couch 8.004s and artifact
classification 3.909s. Log:
/Users/xianxu/workspace/pair/.git/sdlc/local-checks/35ec94d0169cc2438dcefb70c378a31545bce0ed/suite.log.

Before any staging write, fleet inspection found pair:0 now holding another
stage: ec7a395a609a57d75ba58785f5b762b2671b1bcf (#553: omit processes that
vanish during native inspection), clean but staged/unlanded. The previous
free-slot authorization no longer describes its current hold. Did not replace
that stage or build there. Hub send and actors inspection refuse because this
pair:2 incarnation is not registered with Couch; no message receipt was issued.
Waiting for the hub's release/reassignment of pair:0 and restored messaging.
Real pair:1 remains untouched. Operator acceptance and the next M2/close review
cover the merge and runtime delta; neither is claimed complete here.

### 2026-10-14 — merge main after #553 acceptance and stage release

Verified TL a9a15486: pair:0 free, #553 landed, final suite must cover stage.
Merged origin/main 918341db2be241260af23dc1fb42d35f821cd649, including #553
625957fd. Three conflicts: launcher/session_quiescence.go and
launcher/session_servers.go keep #528's exact attributed generations, socket
and argv effect rechecks, two absence observations and saved-layout retention;
adopt main's native procutil snapshot and #553's gone/zombie omission while
unknown/permission rows fail closed. No ps fallback or policy-file authority.
tests/pair-embedded-runtime-test.sh combines controlled native-ancestry
reparenting with the unverified-client-exit admission refusal, not a fake
successful session. Other main changes include creation-lease retry (#545),
Qoder receivers (#550) and raw-open CLOEXEC (#551).

Parent launcher targeted native/server/quiescence checks pass 0.226s.
make test-pair-embedded-runtime passes (five reparenting tests 1.781s, embedded
runtime smoke). A direct shell invocation initially retained this slot's live
Couch scope and refused its isolated PAIR_DATA_DIR; the sanctioned Make target
uses the declared session-environment isolation and passes. No runtime change
was made to hide that fixture environment failure. No unmerged paths or
diff-check errors remain. Fresh recorded full suite follows this merge;
previous passing record 35ec94d0 does not cover these changed runtime paths.

### 2026-10-14 — native recovery-reader integration correction

Recorded merge-head a49a5911831d9f18589290eaffe1169baed33b1d suite exited 2
in 8m31s after 366s gate wait. Only TestProductionDoesNotExecPS failed: #528's
recovery_terminal_os.go still executed ps outside the two conflicted files.
Core 261.115s, cmd 45.056s, tty 37.812s, message 6.871s and all other packages
passed; source inventory and embedded native ancestry passed. Log:
/Users/xianxu/workspace/pair/.git/sdlc/local-checks/a49a5911831d9f18589290eaffe1169baed33b1d/suite.log.

Migrated that reader to procutil.Snapshot plus the final native Table bracket.
A pure projection keeps whole-row identity/parent stability, withholds missing
or changed descendant command evidence, refuses a moved exact named server,
and retains the aggregate 8MiB command bound. Shared serverProcesses preserves
#553 gone/zombie omission and unknown/permission refusal. No ps fallback, guard
exception, new recovery mechanism or M2 execution change. Parent regression
command go test ./cmd/internal/launcher -run
'TestRecoveryNative|TestRecoveryTerminal|TestProductionDoesNotExecPS|TestServerProcesses'
-count=1 -timeout=60s passes 0.553s. New tests independently cover changed
generation/parent, gone vs unknown rows, duplicates, bounds and cancellation.
Fresh recorded corrected-head suite is required before staging.

### 2026-10-14 — merged-head suite passed and acceptance build staged

Corrected runtime head 2808a71a98c4a8994ac34a54587ae64558c1df31 passed
sdlc local-check run --name suite: exit 0 in 10m23s after 103s gate wait,
recorded as sdlc-local/suite, log sha256 prefix bf540f71cbeb. Whole shell/editor
suite and go test ./... passed, including the native no-ps guard. Core
329.197s, cmd 46.506s, tty 37.789s, message 7.175s, launcher 35.850s and
wrapper 56.439s. Log:
/Users/xianxu/workspace/pair/.git/sdlc/local-checks/2808a71a98c4a8994ac34a54587ae64558c1df31/suite.log.

Under TL a9a15486 release, verified pair:0 clean on main, then ran
sdlc stage --build successfully. pair:0 HEAD and bin/couch, bin/pair,
bin/pair-launch-helper and bin/pair-slug all report staged revision
2808a71a98c4a8994ac34a54587ae64558c1df31. The tool also reported an existing
bin/pair-hoprtt from 2284b412 (modified), which the standard build did not
rebuild; no claim is made that this separate binary matches the stage.
The acceptance binaries above match the exact successful suite head.

Sent built hash and evidence to hub: receipt
6623411b-8923-42f3-b8b9-f8d26f7316af, queued. Operator restart and M1 live
acceptance remain pending; named pair:1 case remains untouched. This later
log-only checkpoint does not alter the staged commit or require a restage.
Keep pair:0 staged until the hub directs otherwise. No M2/M3 work started.

### 2026-10-14 — live acceptance evidence and operator startup correction

TL ec749007 reports 13:33 restart on 2808a71a: supervisor starts, three outcomes
render, pair:1 Continue/source-live, all 20 wrappers admitted without focus.
Read-only checks found ops hub attachment present: Couch32496 -> helper35242
resume1-ops-33 -> Zellijclient36717 attach📁1-233, registry warm-reattach, original
server38324/wrapper38785 retained. Current on-screen focus is not persisted.
Bare thread inventory unavailable is the initial RefreshStarted placeholder,
not a named failed read; successful inventory never clears it. Actual failed
reads include an error suffix. Current crash log is empty, no event trace open.
Reported findings to hub, receipt c86e0e8b-8f5e-47a6-9d3a-1f795c3b3ff6 queued.

TL d62e0729 conveys operator's approved M1 delta: auto-attach every Continue
row at startup with bounded concurrency and one retry; preserve failed rows
with visible reason/retry, no startup recovery execution. Hubs unchanged.
Include guarded pair:1 dead-client repair in automatic attachment and the
inventory notice correction. Appended the authoritative plan Revision before
code. Recorded suite and new staged build precede repeat operator acceptance.

- 2026-10-14 M1 operator delta verification: inventory loading placeholder was
  reproduced by `TestInventorySuccessClearsOnlyInventoryNotice` (RED: successful
  populated inventory retained the bare unavailable notice). Inventory notices
  now carry their own transient owner; successful refresh clears only that
  owner, and initial loading does not overwrite borrowed/error/progress notices.
  AST inventory-read guard remains unchanged. Continue-only pass seeding,
  every failure reason, current outcome rendering, exact warm-only retry and
  reducer map-copy tests passed. Full `go test ./cmd/internal/couchtty -count=1`
  passed in 33.975s after the final UI edits. A run compiled while a renderer
  fixture was still being updated saw its old exact substring expectation;
  the settled fixture and complete package rerun passed. No live slot was touched.

- Startup arming exposed a second ordering regression: generic menu reduction
  painted before Run and before the hub's queued borrowed-attachment notice.
  The existing first-paint regression failed. ArmReattachPass now configures the
  same reducer under its mutex without rendering; this pre-Run event emits no
  operations. The first-paint and successful/failed adoption arming tests passed
  together (0.304s). This is startup configuration, not a new scheduler.

- Mixed startup integration passed in 1.899s through the real dispatcher and
  Console with controlled helper barriers: UI paints and processes input before
  attachment completion, failed Continue exhausts exactly two attempts with a
  visible reason and a successful third explicit retry, dead client repair
  changes only helper identity/revision, and live/claim/binding records remain
  identical. Exited rows cause no helper, native launch or teardown. The pass
  keeps the existing bounded lifecycle pool; retention locking may serialize
  external helper entry, so the test asserts its upper bound, not forced overlap.

- TL 23c5d13c narrowed notice ownership to the bare loading placeholder only.
  Updated test first failed because a real corrupt-manifest error disappeared;
  removing the loading-owner flag from real errors fixed it. Focused notice tests
  PASS0.303s. TL 9d117536 added rendering-only pane status: exact row snapshots
  for Continue · attaching…, Continue, and Continue · pane failed: reason
  (retry), plus queued/sanitized width cases, PASS0.361s after RED against the
  old labels. No lifecycle changes. Recorded suite on 8493ccecf is still
  running and cannot certify this later head; run a new recorded suite before
  staging the final head. No native recovery or live-slot actions performed.

### 2026-10-14 — final auto-attach stage ready; repeat acceptance pending

Recorded `sdlc local-check run --name suite` on 8493ccecf passed11m21s,
log sha2563a2dcab807be (core402.485s/cmd70.758s/tty46.041s). That head
preceded the final notice/render clarification and was not staged.
Final runtime head fc14548007f69b3cc6f9802a96b119ebb54b7ac5 passed its own
recorded suite8m20s after45s queue wait; log sha2566d7e009dae36, stored at
.git/sdlc/local-checks/fc14548007f69b3cc6f9802a96b119ebb54b7ac5/suite.log.
Core246.537s, cmd44.351s, tty36.475s, messaging6.892s, launcher22.710s,
wrapper43.782s. Exit0, recorded sdlc-local/suite. No gate bypass.

`sdlc stage --build` replaced pair:0's2808a71a stage with exact fc14548007f6.
Build exited0; Couch, Pair, launch-helper and slug all report exact full staged
revision. Target HEAD rechecked exact. Diagnostic pair-hoprtt remains old2284b412
(modified), already disclosed and outside standard build target. Sent final
built hash and evidence to hub, receipt757592ad-3082-447f-b128-b77d7aed61f1
queued. Operator restart/repeat acceptance, including pair:1 narrow live record
repair, remains pending. Keep this stage; no live slot was attached, relaunched
or mutated by this agent. This log-only checkpoint does not change the tested
or staged runtime commit. M2/M3 execution remains out of scope.

### 2026-10-14 — operator accepts M1 round 2; graceful shutdown explains rev648

TL receipt84fbcea7-c720-4263-82cf-4e7c3702c1ab (submitted, exact ops:0→pair:2)
reports operator restart14:33 onto fc145480 ACCEPTED: Couch started, Continue
rows attached quickly, pair:1 returned Continue and its #542 report reached the
hub after four earlier not-live failures; all20 wrappers message independently
of panes. This closes live acceptance, not M2/M3 authorization.

Read-only explanation of pair:1 rev647→648 with no incarnation: thread.json
mtime14:33:32 precedes new Couch16699 birth14:33:35. Its owned Pair attach
helper17541 started14:33:46 and Zellij client17693 at14:33:47. Thus the
removal predates the new attachment. The old2808a71a LeaveDetach path snapshots
all records, including an unfocused slot with no locally owned helper:
park.go leaveOne proves exact retained PID/start Dead, clearLifecycleDebris
calls RetireIncarnation, which removes that sole dead client and adds one
revision while preserving binding, LastActiveAt and park history. This is the
intended explicit old-shutdown cleanup, not a startup recovery write or gap.
The new attach sees zero incarnations and correctly borrows the listed terminal;
its local handle/registry owns the helper. RepairDeadClientHelper only replaces
an exact retained dead identity; it deliberately never invents one from empty
state or writes a wrapper/agent PID. A non-graceful restart retaining the dead
incarnation still takes the separately tested replacement path. No runtime fix
or pair:1 mutation is needed; add a stateful graceful-exit/restart regression.

Choose TL's recommended landing of accepted M1 now. Remaining revised M2/M3
contracts move to #556 (confirmed disk/fresh recovery) and #557 (new-wrapper
endpoint repair), respectively; preserve their exact acceptance requirements.
No follow-up implementation starts on this branch.

- Graceful-shutdown regression added in warm_attach_repair_test.go: old
  LeaveDetach removes only the exact dead client (+1 revision); a true new
  Couch then WarmOnly-attaches the listed terminal as borrowed, with a local
  owned helper and no further durable/native/teardown effects. Targeted attachment
  group PASS4.028s. No runtime change was necessary.
- Pre-close main merge95997fa92 retains #552's automatic-input reporting and
  #528's pane-independent admission. Sole conflict was NonArtifactSources:
  retained both recovery classifications and main automatic-input classifications.
  New follow-up skeletons #556/#557 came in with main; publish filled contracts
  before releasing their claims. The accepted stage stays fc145480 until landing.

- #552 merge fixture correction: pane disappearance no longer disconnects an
  authenticated live wrapper. Its stall tests now close/reopen the actual
  wrapper connection, preserving last-activity/sequence and reporting safeguards.
  Added unfocused-wrapper positive reporting/dedup case. Targeted stall group
  PASS0.876s, shared AutomaticInputStall/HubMessages/Message/Lifecycle group
  PASS7.154s. Only tests changed. #556/#557 filled contracts published together
  on main c07a31921 and merged; both claims released. Updated atlas pointers and
  the intended empty-incarnation attachment explanation. Full post-merge suite
  and normal M1 review are next; accepted stage remains fc145480 until landing.

- Normal M1 re-close round11 at95a2b9fe returned REWORK, one new finding BR-11:
  mixed startup test times out at its combined failed-attempt-count/rendered-
  reason assertion, reproduced twice including isolated31.542s. Core/UI/launcher
  and other focused checks passed. Split observations and investigate actual
  cause; no timeout widening or gate bypass. Appended plan Revision. The
  post-merge recorded suite remains queued; no successful M1 re-close or landing
  is claimed. Generated sidecars checkpoint this failed round for the next review.

- Post-merge host suite on95a2b9fe passed, but status publication failed:
  gh could not connect to api.github.com. Log SHA256
  2b19da93ed19daa219a22a753a1da867e35bf5df23acb4f0b62cdae8a24b6905;
  this is host evidence, not recorded gate evidence. Final corrected head needs
  a recorded rerun. GitHub rate-limit read later succeeded (remaining5000).
- BR-11 correction is test-only. Original reviewer timing remained unreproduced
  across30 host repetitions/probes. Controlled incremental repaint demonstrates
  raw writes can omit a contiguous phrase visible in VT cells. Replaced the
  raw-render oracle with existing VT projection, explicit menu-open/filter
  barriers, separate retry-count/reason waits and bounded diagnostics. No
  deadline increased. Raw-oracle mutation RED1.396s; focused startup/hub group
  PASS9.231s; corrected integration+oracle10 repetitions PASS18.800s. Complete
  repair/explicit-retry assertions pass. Append-only plan Revision records the
  measured scope without inventing the original cause. Normal re-review next.
- Hub progress sends abbe2e55 and0c60f68a were uncertain (context deadline);
  follow-up receipt reads reported unavailable. No automatic resend was made.
  The final landed outcome still needs a fresh report.

- Normal re-review round12 addressed BR-11 (ten repetitions and mutation checked)
  but found Critical BR-12: automatic broker GetThread calls replay pending
  journals. Reviewer overlay observed recordChanged=true/journalRemoved=true
  without lifecycle action. Confirmed three calls in thread/record/agent
  authority callbacks; families already preview-only/contextual. Appended the
  correction design before production work. Fix shared preview/context routing
  and all automatic authority consumers with global/local pending-journal and
  cancellation regressions, then normal re-review and recorded final suite.
- Canceled only this agent's obsolete9a167 suite process31870 (exact process
  birth16:22:19, parent4470 verified) via INT; exit130. It acquired the machine
  gate after478s and began early script checks before cancellation. No pass or
  recorded status is claimed. Other slots' suite holders were left alone.

- BR-12 regression RED: the previous writable getter replayed valid pending
  journals in global/local/routed-local layouts (nil error, changed payload).
  Shared PreviewThreadContext now uses the same decoder and stamped-read hook
  with preview-only locks; contextual routing covers manifest, legacy and local
  backend reads. Broker thread/agent/record callbacks all use it, and tail
  metadata carries request context. Families were already preview/contextual.
  Preview tests PASS4.555s: pending-journal preservation across3layouts, absent
  store/retention non-enrollment across2, cancellation/contended lock bounds and
  exactly-once unlocked originating read hook across3. Broker resolver pending-
  journal regression passes; explicit lifecycle GetThread afterward proves the
  retained fixture journal remains replayable. Full core/cmd packages pending;
  no final success claim yet. No lifecycle/protocol/agent restart change.

- BR-12 complete core/cmd checks passed on99035: core296.499s, cmd70.737s.
  Round13 verified the journal fix/mutation but returned new Critical BR-13:
  global persisted-file byte reads escape preview deadlines on a FIFO. The
  complete reachable read graph and shared OS reader correction are recorded
  in the appended plan Revision; do not claim lock bounds also bound byte IO.
  No skip/dispute. Canceled own obsolete99035 suite process83401 (birth16:40:15,
  parent4470 verified), INT exit130; no complete pass/recorded status claimed.
  Only that run was stopped; the final suite will follow the source correction.

- BR-13 RED confirmed on actual PreviewThreadContext: a FIFO blocked global
  record201.224ms and routed legacy202.304ms beyond a25ms context. Swept all
  reachable persisted-byte reads: manifests, global/local records, legacy
  routing, damaged-local salvage, archive fallback, aliases and inspection.
  Reused osfs regular metadata admission and extracted its opened-descriptor
  core; local reads retain descriptor-relative no-follow parent opens. Preview
  metadata retains the existing4MiB bound; explicit journal migration limits
  and writable global compatibility remain. No detached read goroutine.
  Nine FIFO branches plus symlink/oversize/cancellation/journal/inspection tests.
  A second RED caught local inspection reads losing authority cancellation;
  one shared context merge now covers global and local inspection reads.
  Final focused metadata/inspection/preview/broker/fastpeek group PASS:
  osfs0.090s, core11.815s, cmd1.691s; diff check clean. Normal re-review next;
  final recorded full suite follows the settled source, before landing.

- Round14 addressed BR-13 with mutation evidence, but returned Critical BR-14:
  automatic dead-client repair routed without context and used writable global
  ReadFile;50ms warm attachment exceeded207–227ms in four FIFO reproductions.
  Important BR-15: production-symbol guard identifies four orphaned adapters.
  Extend the existing bounded-read invariant over the whole automatic attach
  graph and refuse absent WarmOnly before lifecycle work; remove obsolete
  adapters. Appended plan Revision before source. No cap demotion used to
  ignore BR-15, no review bypass; final suite follows the correction.

- BR-14 actual-consumer RED: post-ack FIFO214.041ms despite50ms context;
  preflight lock1.003s despite25ms context; absent WarmOnly replayed a valid
  journal arriving after preflight. Corrected preflight to contextual preview,
  moved absent WarmOnly refusal before hold/retention/lifecycle reads, and
  routed repair through the readonly contextual backend for both routing and
  reread, with context forwarded to the local publication gate. Revision,
  exact-dead old helper and locally owned new helper guards remain unchanged.
- Automatic ThreadStore read inventory: broker record/agent/use-time callbacks
  use PreviewThreadContext; families use RepositoryNamesContext; warm resume
  preflight uses PreviewThreadContext; listing gates retain caller context;
  attachment registration reads only in-memory registry then saves it; repair
  uses readonly contextual routing/reread and no journal replay; worker inventory
  uses PreviewSnapshotContext through local backends. Selected hub observation
  uses a readonly local clone, while explicit hub setup/lifecycle authority stays
  unchanged. Generic Store registry/description reads predate M1 and remain
  outside this per-thread correction; no claim that all filesystem IO is bounded.
- BR-15 removed all four obsolete adapters (readOptionalFile, readThreadLocked,
  localMembership, readLocalPayload), updating two test consumers. Combined
  symbol/artifact classification guard PASS3.236s. Focused automatic warm/startup,
  preview/inspection, inventory-deadline, broker/fastpeek plus symbol guard PASS:
  artifactpath1.403s/core44.923s/cmd4.434s. Includes cancellation immediately before
  repair publication and global/local/routed-local inventory deadline tests.
  Normal re-review and final recorded suite remain required.
- FIFO fixture cleanup also bounds its own writer wait and skips it if the
  acknowledgement hook never ran; focused final rerun PASS1.526s. No production
  edits after the combined passing group; diff check clean.

- Round15 addressed BR-15, verified the parent BR-14 repair regression/mutation
  and complete core334.436s/cmd61.139s plus launcher/tty/message/readiness/osfs
  packages. BR-14 remains open for downstream Pair InferAgent optional metadata:
  actual helper probes wait201.947/200.358ms on ledger/agent FIFO. Parent callgraph
  ended too early. Appended downstream helper design before code: skip unrelated
  selected-attach migration/hygiene, bound optional inference without native
  lookup, poller cleanup, draft open and post-handoff marker admission. Healthy
  presentation and explicit lifecycle semantics stay intact; no gate bypass.

- BR-14 helper RED: production RunLaunch/runOnce real metadata consumers blocked
  on FIFO ledger/agent/index/poller/quit; selected attach incorrectly ran migration
  and orphan sweep; oversized quit object accepted (group4.531s). Separate
  no-reader draft FIFO RED0.605s. Correction skips only selected-attach migration/
  orphan-index housekeeping, bounds optional local inference without native
  dereference (ledger4MiB, agent4KiB), preserves regular draft bytes with a
  nonblocking/no-follow open, bounds poller PID4KiB, quit64KiB and existing restart
  marker cap. Empty legacy quit remains valid; explicit cleanup semantics remain.
- Production helper graph/caps/symlink/healthy-title/final-listing checks PASS
  0.875s; complete launcher package on final source PASS39.361s; combined artifact
  and production-symbol guards PASS4.578s. Four existing runtime files changed
  plus one regression file. Plan matrix and atlas identify each reachable branch,
  its contract and consumer regression. Diff check clean. Normal review and
  final recorded full suite still required; accepted stage remains untouched.

- M1 normal round16 SHIP atdb08fb647, no open findings; evidence commit8cd630492.
  Recorded suite on8cd63049 found a fixture publication race: couchcmd116.844s
  failed TestMessageUnfocusedWrappersReconnectAndExchangeWithoutAttachment at
  line202, unexpected end of JSON input. Ready binding uses non-atomic WriteFile
  while the reader treats existence as complete publication. Plan Revision
  records atomic fixture publication correction; runtime stays unchanged. The
  suite continues other packages; this run is not a pass.

- Ready-publication root cause verified: disposable Go overlay stretches the
  existing empty-file window100ms, reproducing unexpected end of JSON input
  deterministically (RED0.712s). The child now publishes Ready with the existing
  osfs atomic writer; no reader retry/deadline expansion. Same stateful reconnect
  test ten repetitions PASS16.546s. Only test import/publication line changed;
  runtime remains the M1-reviewed db08fb647. Final recorded suite rerun required.

- First recorded suite8cd63049 completed exit2 in11m19s, recorded failure,
  log SHA256prefix f542fcc9ab5d. Only failure is the corrected Ready publication
  fixture; core425.441s, launcher20.412s and all other packages passed.
  Canceled only own queued c0d89560 check (PID45257, parent4470, birth17:38:57
  verified), INT exit130 before machine admission, because origin/main advanced.
  Integrated main4780a5528 cleanly: #555 generated-TL-skill .gitignore entries,
  published issue details, archived #555 artifacts and one lesson; no runtime
  source or code-path conflict. New recorded suite must cover the merged head.
  Normal whole-issue close still follows, with no review skip.
