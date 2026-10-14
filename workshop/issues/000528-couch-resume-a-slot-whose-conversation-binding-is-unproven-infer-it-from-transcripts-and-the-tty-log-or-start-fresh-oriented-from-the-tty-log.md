---
id: 000528
status: working
deps: []
github_issue:
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
- [x] M1 — Prevent split terminal incarnations and show attachable-agent liveness.
- [x] M2 — Calibrated native recovery with tty context and safe fresh fallback.


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
