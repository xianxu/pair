---
id: 000523
status: working
deps: [pair#506]
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours: 7.02
card_mirror: '5c7a96194038bb7de50ae6825efded3e08dfe0e8' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T22:52:07-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# broadcast: 👆 and 👽 are capability switches on both casts (👆 off: no one annotates; 👽 off: no one drives from outside)

## Problem

The cast row's emoji mean different things on the two casts. On the terminal cast, left-clicking `👆` toggles pointing (first click makes and copies the pointer link; off clears marks and leaves the link view-only; README, #412). On the web cast (#506), left-clicking `👆` or `👽` only copies a link. The operator's click rule (#495) says left click acts or toggles, right-click copies.

## Spec

The operator (2026-10-12): `👆` means "marking-up, here" on both casts, and the emoji are capability switches:
- **`👆` off: no one can annotate.** Left click toggles it (amber when on). On the web cast, off means pointer-link viewers become view-only (the Board refuses their strokes and notes, as for a view-only link) and the operator's cast-Chrome overlay (#494) is unavailable; existing marks clear as on the terminal. The first click makes the pointer link and copies it, as the terminal does.
- **`👽` off: no one can take control from outside.** Left click toggles it. Off means the drive link can't ask to drive; if someone is driving when it turns off, control returns to the operator.
- **Right-click copies** that control's link on both casts, whatever the state. Hover and Alt+click on `LIVE ⏸` / `Chrome ⏸` stay as #495 made them.
- The terminal cast already behaves this way for `👆`; keep it, and share one rule in code and README.
- Server-side enforcement for both switches (never only hidden UI); viewers see a clear state ("annotation is off", "driving is off").

### Agreed interaction and defaults

- Web annotation and external driving start OFF. Enabling either capability
  copies its existing link; links and code locks survive subsequent toggles.
- Right-click is copy only, even while OFF. If the terminal pointer link has
  not been made, mint its token and code lock without enabling it, then copy.
  Preserve terminal off-to-on copying, automatic hidden-marker disable,
  geometry checks and late-request protection. There is no terminal drive button.
- Annotation OFF covers pointer-link and drive-link viewers and the operator
  overlay. Clear current strokes and notes; keep earlier recording history.
  Re-enabling starts with empty annotations and pen/watch mode, never a held gesture.
- Drive OFF prevents requests, grants and queued remote input, clears the active
  driver and every pending request, and releases held buttons/keys. Seats and
  links remain; re-enabling requires a fresh request and operator grant.
- A link's role, whether the capability is supported, and its current enabled
  state are distinct. UI and viewer state derive from server-side authorities.

### Proposed enforcement and integration

Keep annotation authority with Board and driving authority with the drive
controller. A shared pure click policy is the one left-toggle/right-copy rule;
do not replace the terminal pointer's established safety state machine.
Use generation-bound admission and ordered clear/revoke operations so an
accepted-but-delayed request cannot survive off/on. Disable closes admission
before cleanup; failed cleanup stays disabled with a visible error and a safe
retry, never a false successful on state.

Web annotation clearing needs one ordered recording operation, consumed by Go
and JS reducers and CatchUp, including pause/resume, so marks cannot reappear
on reconnect. Drive off uses one all-control reset transition rather than a
sequence of separately observable revokes/denials. Runtime capability snapshots
travel over the existing SSE connection and are replayed to late UI subscribers.

#494 is intentionally not branched into this issue. Whichever issue lands second
wires the capability into its existing operator Board and recorder admission
checks and applies `__couchcastAnnotationAllowed(bool)` to the exact couchcast
isolated context on each new document and capability change. Start unavailable
until current capability is applied; serialize/version those applications so
an older ON cannot win after OFF. No overlay implementation is duplicated here.

## Done when

- Both web capabilities start OFF. Annotation OFF refuses strokes, notes and note-state changes from pointer/drive viewers and the operator; current marks clear and never return on reconnect, re-enable or pause/resume.
- Drive OFF refuses new requests, stale grants and queued input, clears active and pending control, and releases held input; re-enable requires fresh consent.
- Left toggles and copies on enable; right copies without enabling, including making the terminal pointer link when absent. Stable links/codes, terminal pointer safety and hover/Alt-click remain intact; no terminal driving is added.
- Viewer and cast-row states stay current across reconnect, asynchronous module mount and document replacement. #494 integration is completed by the second landing issue using the agreed hook and server admission seams.
- Every capability phase projects to a clear row/viewer state and one recovery action: failed cleanup is OFF with left-click retry; lost recorder/CDP makes the cast visibly broken with an explicit end-cast action. No automatic retry or hidden enable refusal.
- Tests pin admission and delayed-operation sequences, clearing/replay, click behavior and rendering. Full suite and headless conformance pass; operator approves on pair:0 when TL schedules staging.

## Plan

- [x] M1 — Implement capability enforcement, ordered annotation clear/control reset, server state and race tests.
- [ ] M2 — Implement shared click behavior, viewer controls, overlay integration as landing order permits, documentation and acceptance.

Durable plan: [000523-cast-capability-switches-plan.md](../plans/000523-cast-capability-switches-plan.md).

## Log

### 2026-10-13 M2 implementation checkpoint

- Integrated landed #494 (PR303) from main. Exact-context permission receipts,
  default-OFF overlay construction, binding-ingress permits and ordered recorder
  checks fence replacement documents and delayed writes; held drafts survive.
- Cast-row controls and persistent one-action recovery committed at f1c26fef7;
  overlay integration at e7adbcfe9. Stale completion and transient stroke-cache
  findings from ad-hoc reviews are fixed with reproducing tests.
- Viewer controls now consume replay-latest capability state and tolerate
  unordered control/capability SSE delivery. Off cancels gestures and queued
  input; re-enable requires fresh consent across generations.
- Fresh Node suites: 229 passed. Corrected full couchtty/broadcast packages
  passed (29.982s/27.199s); focused control race checks passed. Headless operator,
  annotation and drive subset passed (6.251s), and the two updated live fixtures
  passed (10.420s). Full make test and fresh complete headless suite underway.
- README and atlas map defaults, stable links, visible recovery and overlay
  ownership. Operator acceptance remains pending; no pair:0 staging performed.


- 2026-10-13: closed M1 — All four M1 review findings addressed: replay prefix/seek regressions red then 32 tests green; actual production control adapter delayed/closed/failing/canceled recorder regressions and webcast/drive/recorder race suites pass; inventory and README corrected. Earlier make test completed with artifactpath and legacy Couch fixtures fixed and whole packages rerun green. Full fresh suite required after M2.; review verdict: SHIP
### 2026-10-12

- #506 confirmed landed as PR #302 / a5a3fa1f16174a91a647dbf1c97a5da0fdd84dab;
  claimed and start-plan in pair:7 from main, with no stacked branch.
- TL approved defaults OFF, right-copy lazy mint without enabling, stable links,
  all-pending drive revoke and no terminal drive (receipt 9e080cee-0e91-4054-ab52-d84a5fd86bd4).
- Read-only exploration identified the existing Board, Controller, SSE and
  recorder seams; code remains untouched pending design/plan review.
- #494 owner confirmed exact seams (receipt 922c3a42-8d6c-4833-91be-335b325a5eb8):
  board/operator.go annotationPageLocked; recorder/operator.go operatorAdmission
  at prepare and ordered write; chrome_operator.go startOperator/refreshOperator
  with exact session/context identity. #494 owner wires if it lands second.

- Plan-quality PQ-1/PQ-2 addressed; change-code passed after calibrated
  estimate (7.02h). Estimate review notes optimistic integration grouping:
  row 5 owns controller workers plus typed transport/release progress, row 3
  owns pure Go/JS/codec tests, row 4 owns ordered queue integration tests;
  rows 4–7 own races/full regression, row 10 headless, row 8 active staging and
  acceptance. Scheduling idle time excluded; actual remains measured.
- M1 recording tests observed missing clear/reset API (Go) and stale marks
  (browser) before implementation. Go recording suite and 13 browser-model tests
  now pass, including shared timeline data, every catch-up prefix and split-file
  validation. Capability reducer is being implemented independently.
- Ordering discovery for Task 2: recording/files.go merges equal timestamps as
  events then notes then strokes. AnnotationClear must be stamped strictly after
  the previous record (at least previous T+1), or a previously emitted note can
  replay after its clear. Subsequent records keep normal monotonic clamping.

- M1 Task 1 complete: pure capability reducer, generation/attempt fences,
  annotation-clear and atomic CtlReset, Go/JS reducers and shared recording
  timeline. Spec review compliant; quality review found a missing stale-attempt
  assertion, now fixed and mutation-verified (removing the attempt guard fails).
  Capability and recording Go suites plus all 13 model tests pass. No milestone
  boundary crossed yet; server/controller enforcement remains outstanding.

- M1 Task 2 annotation half implemented: Board default OFF, captured admissions,
  command-ordered mutations/clear, explicit retry and latest-state subscription;
  Writer and recorder acknowledge/deduplicate cleanup, including paused and
  before-live behavior. Recorder also exposes ordered ResetControl for the
  controller integration. Chrome/Cast cleanup adapter wired.
- Board spec review found sticky sink errors could preserve a nested Retryable;
  wrapped them as Broken, observed regression red then green. Re-review clean.
  Quality review clean; root ran capability/Board/recorder race suites green
  before the drive work began. Webcast compile-only green before drive edits.
- Drive authority/release implementation is in progress independently; HTTP
  admission test is written but its red run is temporarily blocked by those
  in-flight compile changes. No full-suite or milestone-close claim yet.

- M1 Cast/server integration: authoritative default-OFF snapshots, stable secrets,
  bounded single-flight cast workers, explicit retry and cross-capability broken
  fencing. Cast ad-hoc review clean; focused race tests pass. HTTP tests cover
  body-read fencing, role intersection, all four combinations, SSE OFF/reconnect.
- Drive ad-hoc review found cancellation could discard a late callback failure,
  uncertain text/click could be hidden during concurrent disable, and queued
  cancellation lost definitely-unsubmitted evidence. Recorder now claims queued
  work atomically and awaits any started callback's settled outcome; red/green
  tests plus recorder race suite pass. Drive fixes under review before M1 close.
- New multi-viewer SSE race test exposed recording.Encode mutating shared Header.K.
  Encode now stamps a shallow copy; all-kind immutability regression reproduced
  the defect. Full webcast race suite passed after the fix.
- #494 landed PR #303 (889fafd05), confirmed receipt
  f377b362-d13c-4a0c-9b99-f1ef3aae5fe7. This branch owns second-landing overlay
  permission and operator-admission integration after merging main.

- M1 Task 2/3 implementation complete. Drive spec review now clean after fixing
  late settled failures, concurrent disable with uncertain unheld input, queued
  canceled ups with existing holds, and uncertain partial control writes.
  HTTP release now uses its captured generation too; delayed-release regression
  reproduced a new grant being revoked, then passed with admission fencing.
- Verification: all webcast Go subpackages passed; race suites for Board,
  capability, drive, recorder and webcast passed together. All 13 JS model tests
  pass. Removing final Board and injector generation checks kills the targeted
  regressions; restored checks pass. Drive task quality review pending, then M1
  binary-owned boundary review. User-visible controls/overlay remain M2 work.

- Drive quality review found an uncertain control result could arrive after a
  canceled cleanup wait and be lost on retry. The controlled regression failed;
  cleanup now consumes that unobserved outcome even after the busy channel ends.
  A separate regression proves already-reported retryable failures need exactly
  one explicit retry. Quality re-review clean; focused race regressions pass.
- Full repository suite exposed the exhaustive production-source inventory gate.
  Registered the nine capability source files as non-artifact-path producers;
  artifactpath suite passes. Full-suite remainder is still running before M1.

- M1 full-suite result: make test completed; all shell/Lua checks and Go packages
  passed except artifactpath inventory and legacy couchtty note/drive fixtures.
  Both causes fixed, then complete artifactpath+couchtty package rerun passed
  (logs /tmp/pair-523-m1-suite.log and /tmp/pair-523-m1-fixed-packages.log).
  No remaining full-suite failure. Full fresh make test remains required at M2
  acceptance after controls/viewers/overlay integration.

- M1 boundary review returned REWORK with four findings. BR-1 reproduced in
  shared-fixture catch-up and seek tests (cleared notes/strokes resurrected after
  resume), then fixed; all 32 replay tests pass. BR-3 corrected the Board source
  inventory; BR-4 documents default-OFF authority versus persistent links.
  BR-2 acknowledged production control writes remains in progress. M1 is not
  closed. #494 landed at 889fafd05 / PR303; merge and overlay wiring follow M1.

- BR-2 fixed in 42ea9cdde: ordinary control writes await the actual Recorder
  reducer/sink outcome through liveGate and Chrome. Production-adapter stateful
  regressions cover delayed success/failure, closed recorder before/after first
  live, queued versus started cancellation and controller Close joining writes.
  Webcast/drive/recorder race suites and artifactpath pass. BR-1/3/4 fixes are
  committed in 570518fb2; rerunning the M1 gate with all four addressed.

## Revisions

- 2026-10-12 22:58 PDT: Added approved defaults/copy semantics and proposed
  enforcement design. Done when now enumerates the full surface rather than
  relying on a revision to supersede a stale checklist. #494 is a coordinated
  integration boundary, not an unlanded branch dependency.
- 2026-10-12 23:08 PDT: TL approved design at 2ddb90a9f conditional on visible
  recovery (receipt 6f882976-021f-41f3-833c-5a2e0c30aba4). Plan Revisions now
  defines the phase-to-row/viewer/action table, single-flight click retry and
  terminal failure/end-cast projection for both capabilities, plus tests.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md`
against `baseline-v3.1.md`. Method A only.* Calibration is marked stale by
estimate-source; these rates are provisional.

Derivation after plan-quality passed: existing Go/JS/CDP stack, familiarity 1.0.
The new pure capability reducer is repository-specific policy; no library can
replace it. Existing recorder, HTTP/SSE, controller, CDP transport and browser
fixtures provide the integration libraries, so no new-stack discovery budget.
The thorough plan gives implementation components design ×0.2 and a 15% design
buffer; issue/spec authoring itself is undiscounted. All implementation values
are v2 table rates ×0.4 once. Rows below follow this mapping:

| Row | Scope | Raw design → recorded | Raw implementation → recorded |
| --- | --- | --- | --- |
| 1 | Issue/spec and recovery design | 0.8 → 0.8 | 0.2 → 0.08 |
| 2 | Pure capability model | 1.0 → 0.2 | 0.6 → 0.24 |
| 3 | Go/JS recording clear/reset reducers | 0.3 → 0.06 | 0.5 → 0.2 |
| 4 | Board/recorder ordering integration | 2.0 → 0.4 | 1.2 → 0.48 |
| 5 | Controller/CDP cleanup reconciliation | 3.0 → 0.6 | 1.5 → 0.6 |
| 6 | HTTP/SSE authority projection | 0.3 → 0.06 | 0.5 → 0.2 |
| 7 | Viewer permission lifecycle | 1.5 → 0.3 | 1.0 → 0.4 |
| 8 | Cast row and shared interaction | 1.0 → 0.2 | 0.6 → 0.24 |
| 9 | Terminal ensure-link adaptation | 0.2 → 0.04 | 0.3 → 0.12 |
| 10 | Overlay handshake/headless integration | 1.5 → 0.3 | 1.0 → 0.4 |
| 11 | README/atlas | 0.1 → 0.02 | 0.2 → 0.08 |
| 12–14 | M1, M2, issue-close reviews | 0.1 each → 0.02 each | 0.4 each → 0.16 each |

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.8 impl=0.08
item: greenfield-go-module design=0.2 impl=0.24
item: smaller-go-module design=0.06 impl=0.2
item: api-integration design=0.4 impl=0.48
item: api-integration design=0.6 impl=0.6
item: smaller-go-module design=0.06 impl=0.2
item: api-integration design=0.3 impl=0.4
item: tui-screen design=0.2 impl=0.24
item: smaller-go-module design=0.04 impl=0.12
item: api-integration design=0.3 impl=0.4
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.02 impl=0.16
item: milestone-review design=0.02 impl=0.16
item: milestone-review design=0.02 impl=0.16
design-buffer: 0.15
total: 7.016
```

Design 3.04 × 1.15 + implementation 3.52 = **7.016 hours**, recorded as
7.02. Includes verification and review; operator scheduling wait is excluded.
