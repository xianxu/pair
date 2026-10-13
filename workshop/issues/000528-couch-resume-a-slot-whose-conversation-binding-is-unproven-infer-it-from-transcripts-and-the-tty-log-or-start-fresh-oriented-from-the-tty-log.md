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
- [ ] M1 — Prevent split terminal incarnations and show attachable-agent liveness.
- [ ] M2 — Calibrated native recovery with tty context and safe fresh fallback.


## Revisions

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
