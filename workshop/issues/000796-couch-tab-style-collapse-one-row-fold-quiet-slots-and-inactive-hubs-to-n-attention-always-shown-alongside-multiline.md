---
id: 000796
status: codecomplete
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours: 2.593
collaboration-mode: collaboration
card_mirror: '57cea296eb0d539a384988ce331aee78a247fbd8' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T13:53:50-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 1.78
---

# Couch: --tab-style collapse (one row; fold quiet slots and inactive hubs to +N, attention always shown) alongside multiline

## Problem

The tab bar must remain useful across a large fleet without changing child height as chips or notices arrive. The operator prioritizes collapse as the default; multiline remains an explicit choice. Wrap-induced chrome-height transitions are implicated in #810's resize-without-repaint idle-admission stalls.

## Spec

Operator (10-22, after accepting pair#779's two-line wrap): "another way to deal with the width limit is to keep it at
one line but have some intelligent collapsing. for example, the inactive hub's worker slots can be collapsed. slots
without either notification (amber) or blocking (red) can be collapsed to just a number, e.g.
`pair  :1 (red)  :4 (amber)  +4` (indicating 4 more slots hidden). can we make such a mode and selectable from command
line? maybe --tab-style collapse | multiline."

Design:
- `couch --tab-style multiline|collapse` at launch (default `collapse`; `multiline` retains pair#779 behaviour); a runtime toggle
  key is a possible follow-up, not in scope.
- **collapse:** one tab row. Within each repo group, show the active slot, every slot needing attention (amber
  notification, red blocked), with launch-hub (primary or numbered) and useful :0 anchors independent of focus.
  Known parked/non-live rows are excluded; queued/loading placeholders remain explicit. Fold eligible quiet live
  members only AT three per exact owner/repository group: one or two stay explicit, exactly three folds.
  Quiet inactive hubs may fold with quiet workers when that threshold is met. `+N` means hidden live members;
  selecting one legitimately removes it from the count. Every character of the complete count opens the switcher
  exactly scoped by owner hub and group; no in-row expansion. At impossible widths retain active/attention
  priority and mark hidden attention in `+N !`; the active chip is never folded.
- Reuse pair#779's measured packing, hit-map and chrome-row publication: collapse is a different input to the same
  packer, not a second renderer. The status-text cap (pair#789) applies in both modes.

## Done when

- Bare launch defaults to collapse; explicit multiline remains selectable, with CLI validation and actual Console wiring tested.
- Collapse maintains one chrome row across actor/attention updates and width changes whenever terminal height permits one chrome row; child-height and no-wrap-induced-resize regressions cover #810's mechanism. Physical terminal size changes still resize PTYs.

- With `--tab-style collapse`, a fleet of N slots per group renders in one row: active plus attention slots shown,
  eligible quiet live members form `+N` only at three or more (two stays visible, exactly three folds). Known
  parked/non-live rows never contribute. Counts and launch-hub anchors stay unchanged across outside-tab focus
  changes; selecting a folded member legitimately changes the hidden count. Render/projection regressions cover these cases.
- Clicking the plus and every digit of `+N` opens that exact group, using physical terminal cells after emoji aliases as the click oracle.
- Primary and numbered hubs obey the same role policy: launch hub retained independently of focus when space allows; eligible quiet inactive hub/workers produce at most one count per exact owner/group when the threshold is met, independent of legacy display-run splits.
- `multiline` is unchanged (existing tests).
- Staged for an operator look.


## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only; calibration is currently stale/provisional per sdlc estimate-source. Current-stack reuse is already available: measured renderer, PresentThreads, menu reducer, FakeHost/FakeRunner and native PTY harness; no greenfield external API or library discovery. Familiarity1.0 (same #779/#789 chrome area). Thorough approved plan earns design x0.2 for implementation primitives and15% design buffer; v3.1 implementation values are40% of v2 table units.

Derivation, before buffer: issue/spec shaping design0.30+impl0.08; style parser/plumbing smaller module raw design0.30 x0.2=0.06, impl0.50 x0.4=0.20; collapse packing TUI raw design1.50 x0.2=0.30, impl1.00 x0.4=0.40; scoped menu TUI raw design1.00 x0.2=0.20, impl0.80 x0.4=0.32; geometry/caller integration cross-cutting raw design0.50 x0.2=0.10, impl0.50 x0.4=0.20; docs design0.10 x0.2=0.02, impl0.15 x0.4=0.06; close/stage review design0.20 x0.2=0.04, impl0.40 x0.4=0.16. Design subtotal1.02; implementation1.42; design buffer0.153; total2.593 focused hours. Machine queue time is excluded; native/default compatibility and operator iteration are the main uncertainty.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.30 impl=0.08
item: smaller-go-module design=0.06 impl=0.20
item: tui-screen design=0.30 impl=0.40
item: tui-screen design=0.20 impl=0.32
item: cross-cutting-refactor design=0.10 impl=0.20
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.04 impl=0.16
design-buffer: 0.15
total: 2.593
```

## Plan

- [x] Review and approve [durable plan](../plans/000796-collapse-tab-style-plan.md).
- [x] Implement and verify CLI/style policy, measured collapse packing and exact scoped count navigation.
- [x] Verify native geometry, safety guards and multiline compatibility; submit recorded verification and request close review.

## Acceptance

Recorded suite must pass before landing. Stage for the TL-sequenced operator look after #753; operator acceptance precedes landing. Close and merge remain gated. No acceptance result is implied by the implementation checklist.

## Log

### 2026-10-22
- 2026-10-22: closed — BR-4 fixed: existing pending-reattach regression expanded to collapse and multiline; collapse red0.317s reproduced zero placeholder candidates, corrected queued/loading admission plus parked exclusion/two-focus/click/threshold selection PASS0.343s. No policy change: placeholders remain explicit and parked attached rows excluded. Prior full reviewer independently passed whole tty50.579s/cmd114.246s/textwidth/rowtext/artifactpath and disposed BR-1/2/3. Inventory/diff checks pass. TL9df07fa8 relayed operator second look works. New exact-source suite submitted after placeholder fix; prior suite is older snapshot only and not claimed as final. Landing requires corrected-source full suite and this full review.; review verdict: SHIP
- 2026-10-22: closed — BR-1 accepted round2: exact identity aggregation and numbered/current hub-role regressions plus old-code negative control. BR-2 addressed by complete concepts-table audit against actual production declarations: new opener/config files, modified projection/menu/renderer, unchanged geometry/menu reuse correctly named. Documentation-only delta; source/stage/suite snapshot unchanged. Whole couchtty PASS52.969s, couchcmd121.386s; native and guards independently passed round2. Corrected stage93b8b8b6 all4 binaries modified=false; suite lease-521073252 and operator acceptance pending, both required before landing.; review verdict: SHIP

Claimed and started from main at TL6a768f10 while #789's isolated integration suite waits. No #789 commits are stacked here. #789 remains branch3306f2d16 with check session61444/lease-3428295058/attempta71b7262e10d47a252d630372095d2ff; return there for full-suite result, fixed review and merge. start-plan hit ariadne#557's generated workflow dirt: saved /tmp/pair796-seeded-merge-check.diff, restored only .github/workflows/merge-check.yml per TLc4e804fc, retried successfully; no seeded gate changes committed.

Read-only survey found existing chipKept keeps every hub and existing aggregate overflow has no click spans; collapse needs mode-specific candidate policy and count spans from the same clipping pass. A fuzzy switcher Filter cannot express exact owner+group identity. Proposed typed scope and count navigation are in the durable plan, awaiting TL approval. No implementation started.

Plan-quality round2 CLEAN: PQ-1 addressed; no open blocking findings. Estimate derived only after this pass.

Plan-quality round1 REWORK/PQ-1: scenario inventories needed named production-function test strategies. Refined parser/extractor/CLI, collapse preparation/packing/drawing, scope filtering/reducer and revision-fenced opener coverage with adversarial inputs and mechanical guards. No policy change; implementation remains gated.


### 2026-10-22 — implementation checkpoint

change-code passed plan quality (PQ-1 disposed) and estimate-quality INFO, then admitted full-flow implementation. Estimate2.593 uses idle-removed ship wall-clock; INFO notes integration/stage optimism. The two TUI primitives include their unit/integration iterations; geometry primitive covers additional native caller wiring. Issue/spec design0.30 is a pre-resolved shaping allowance (raw1.50 x0.2); no implementation scale is applied twice. Operator iteration remains uncertainty rather than a passing gate.

Implemented one typed launch default and both CLI forms, actual Console configuration before adoption, collapse candidates through the existing measured packer/drawer, weighted per-group counts and exact drawn spans, shared one-row birth/paint geometry, confirmed-revision count clicks, exact scoped menu filtering/reconciliation/Escape and visible scope recovery. Explicit multiline remains the compatibility policy. No new timer/process owner or persisted setting. Quiet inactive hubs fold; active and Bell/Blocked members precede anchors/counts; impossible widths preserve clipped active and attention-marked global counts when those fit. README/help/atlas updated.

TDD: initial collapse tests failed because quiet members/hubs stayed visible and width1 wrapped; now collapse/priority/count-scope/conservation/N-and-2N work tests pass0.357s. CLI/runtime/help sweep passed tty0.275s/cmd3.044s. Exact scoped-menu tests observed same-name/nested-return red then passed0.430s; visible scope header test red then passed0.380s. Confirmed/stale count input test passed0.348s. Native content-change geometry test passes and forcing multiline via a test overlay fails at the old wrap boundary (110.394s); no deadlines changed. Expanded existing native wrap test to collapse/multiline x Claude/Codex, retaining selected/background kernel size and physical bottom-row checks: PASS0.658s. Artifact source classification PASS4.162s; reader inventory pending final check.

Compatibility sweep: first whole couchcmd failed one additional default invocation literal; all ten launch literals now explicitly include DefaultTabStyle, nonlaunch literals stay zero. Corrected whole couchcmd PASS121.386s. Existing multiline fixture helpers and direct native wrap test now explicitly exercise that option. An earlier whole tty run compiled before these corrections and exhausted its4m diagnostic limit after the old native two-row expectation; its later shutdown/pointer failures occurred with that exhausted package budget and are not classified as flakes. Corrected known multiline activation/two-row cases PASS0.588s, capability recovery PASS0.471s and drive PASS0.429s. Fresh whole tty verification is running on the corrected snapshot, session66232; no package/full-suite pass claimed for it yet.

#789 integration suite session61444/lease-3428295058 remains snapshot-isolated and queued, last observed position15. No #789 merge runner is active. Return to that branch on its result for fixed review/full-suite recovery and merge; keep #796 independent. Staging #796 awaits green checks and TL's first operator window after #753; never overwrite pair:0 without the go.


### 2026-10-22 — package green and recorded verification

Corrected whole couchtty PASS64.889s; whole couchcmd PASS121.386s. The previous native test's implicit multiline assumption was replaced by an explicit collapse/multiline x Claude/Codex matrix, retaining kernel winsize, selected/background PTY geometry and physical bottom-row output. No fixture deadline changed. Collapse count display conservation additionally passes across widths1–120: drawn actors plus per-group/global folded counts equals original actor cardinality whenever a count is shown. Count/work/native selection PASS0.357s; final conservation test PASS0.398s; inventory/diff checks pass. Existing native test isn't classified as a repository reader; its attempted certificate record correctly refused, and complete inventory passes without a new certificate. Other new/current reader certificates cover only local literal argv/private PTYs, no repository reads.

PR547 opened through sdlc. First recorded #796 suite on e6ccdbd36 was queued as session80684/lease-3415236468/attempt0326d146a188c676013f15328aba6dbe, initial position20. Canceled only that exact lease before execution to include the final conservation assertion on the tested snapshot; no result claimed, #789's separate lease preserved. Replacement recorded suite will run on this final test head. The whole-package evidence applies to unchanged production code; the additional assertion passed separately. Implementation is complete; recorded suite, close review, staged operator acceptance and landing remain workflow gates, not completed evidence.


### 2026-10-22 — final recorded suite submitted

Final #796 recorded suite on pushedab4055c12b80f7ee9e4172f71b332d284aa19b53 is queued, session45335, lease-3791587181, attemptb1e4be1f0838dcda8c7443d6729136f7, isolated checkout /Volumes/TestScratch/local-checks/sdlc-local-788450635. First confirmed FIFO position20. The older #796 lease was canceled before execution and exited1/context canceled, with no test result. #789's distinct lease-3428295058 remains preserved; the multiple-owned-runs warning is expected because TL authorized overlapping independent snapshot suites. No source changes are pending. Close review may overlap this queued check under tl-worker; landing still requires its passing result and staged operator acceptance.


### 2026-10-22 — close round1 BR-1 numbered-hub class

Review REWORK/Critical BR-1 correctly found that legacy chipGroups deliberately splits after a numbered hub. Reusing those runs both folded a quiet current numbered hub (only slot0 was preserved) and emitted two counts for one inactive owner/group. Regression matrix primary0/numbered1/numbered4 x current/inactive reproduced all four numbered cases (targeted0.305s). Corrected the class: collapse aggregates by exact StatusGroupID in first-presentation order, with hub-role or slot0 anchors retained for the current owner; inactive quiet hubs fold with workers into one count. The existing multiline chipGroups definition is restored unchanged. Shared measured packer, drawing/spans and scope routing remain the sole mechanisms. Preparation is two linear actor passes plus group visits, with O(N) transient storage; counted N/2N bounds pass unchanged.

Collapse/status/scope/native selection PASS0.490s on the fix, including all six role cases and actor conservation. Fresh whole tty run session71345 is pending. The previous #796 suite lease-3791587181 was canceled while queued before execution, retaining #789's independent lease. No suite pass claimed. Staged bccaa4617a2fa49cf6a530cfab72fc04bb581b04 had all4 binaries verified modified=false, but TL was immediately told to hold that operator look until corrected restage; review did not finalize close. Prior released main e1636e71 differs from our base only in two workshop issue files, so accepted operator code remains present.


### 2026-10-22 — BR-1 corrected stage and recorded suite

Whole couchtty PASS52.969s on93b8b8b655c07853c6bc3a1f3523903863680b40; exact role/scarcity/scope/native selection PASS0.490s. Reader inventory and diff checks pass. Restaged on pair:0 through sdlc stage --build: couch, pair, pair-launch-helper and pair-slug all verify that exact revision with vcs.modified=false. TL received the corrected head and two-minute default/narrowing/count/Escape/multiline operator check; no acceptance claimed yet.

Replacement #796 recorded suite on93b8b8b655c07853c6bc3a1f3523903863680b40 is queued: session38831, lease-521073252, attempta278fb06545111abc7fe0a1a1410829a, isolated checkout /Volumes/TestScratch/local-checks/sdlc-local-1119877122; initial/latest observed FIFO position18. The canceled prior lease had not executed; #789's distinct snapshot remains queued. Source frozen for close round2 and this suite. Next: review BR-1 disposition, receive operator look, require passing final recorded suite before landing.

### 2026-10-22 — close round2 BR-1 addressed; BR-2 plan mapping

Reviewer accepted BR-1, independently passed focused CLI/scope/style/native/stale-input/broadcast/privacy/anchor checks, and verified the old-code mutation fails all four numbered cases. One new Critical BR-2 is documentation-only: the concepts table still named predicted files as modified although geometry/menu machinery was reused and the opener landed in a new file. Audited the entire table and recorded actual locations/new-versus-modified status, with unchanged reuse listed separately and a plan Revision. No code or tests changed; corrected stage93b8b8b6 and queued suite lease-521073252 remain valid and untouched. Re-close on this documentation correction; operator look may proceed on the same corrected stage.


### 2026-10-22 — close SHIP and priority admission

Close round3 EXIT0/SHIP, no open blocking findings; evidence commit844928d2ba0b145122c018026098fb1de0c920fb pushed. Reviewer accepted the corrected concept table and independently checked focused native/geometry/menu/privacy guards. TL granted #796 priority until16:13 (delivery031af089); retained suite lease-521073252 now reports fleet-blocking priority, position1 next to run. No duplicate or replacement runner started. Source/stage93b8b8b6 remains unchanged, all4 binaries previously verified modified=false. Full-suite pass and operator acceptance are still outstanding before landing. #789's separate integration snapshot/lease-3428295058 remains queued and preserved; return for fixed review and merge when its result arrives.

### 2026-10-22 — stage refinement verification

Operator's first look was not accepted; follow-up covers threshold, live source and the full count hit target. Threshold red reproduced four 1/2-member cases; physical click red reproduced final digits of both +4 and +12 outside the old emoji-shifted span. Live projection red reproduced a parked attached pane counted as live; isolated old-anchor mutation still fails after parked filtering (bench focus changed brain +3 to +4). Corrected focused collapse suite PASS0.424s. New projection fixture proves aliased ops-echo hub and four echo slots are distinct repositories, one actual echo aggregate, exact scoped membership.

Corrected whole couchtty PASS64.736s; textwidth PASS0.063s; rowtext PASS0.068s. An intermediate width implementation broke TestMenu100Bounds (2462 allocations >1400); root allocation eliminated with original-prefix slices/ASCII fastpaths, existing budget unchanged, targeted performance pin green and included in whole tty pass. An earlier in-flight targeted run compiled before updated threshold fixtures exhausted its existing count wait and failed obsolete expectations; no pass/flake claim for that superseded snapshot. Inventory and whitespace checks pass. Whole couchcmd still running. Next commit/push corrected source, restage all4 binaries and submit a replacement recorded suite under TL's retained priority, then fixed review; operator second look remains required.

### 2026-10-22 — full review requested after stage refinements

Fix-only close round4 REWORK/BR-3: no functional findings, independently passed focused grapheme/fit/collapse tests; the new threshold/live policy must receive a full close review. Main Spec/Done-when and plan decision paragraphs reconciled with recorded operator revisions, preserving the revision history. Corrected staged source435148c06892bc8db5d30aef95c70b1ca005ed93 remains unchanged, all4 binaries verified modified=false. Whole couchcmd PASS130.577s. Recorded suite session90507/lease-1801226722/attemptb4b13e2a5d8973807f06c92257f9db8e, snapshot /Volumes/TestScratch/local-checks/sdlc-local-3536115088, admitted15:03 and running after restart. #789 session61444/lease-3428295058 still queued; no duplicates. Operator second look pending. Next full close, then require suite and operator acceptance before merge.

### 2026-10-22 — operator accepted; BR-4 pending-state admission

TL9df07fa8 relayed the second operator look: "works". Acceptance applies to the approved threshold/live count/click policy on staged435148c0; #819 hidden-state glyphs is separate, after #796 and #789 land, before #811/#812. Full close round5 independently passed whole tty50.579s, cmd114.246s and shared packages but found BR-4/Critical: known detached rows with legitimate queued/loading placeholder candidates were removed by the live-only filter. This violated the already explicit placeholder contract. Expanded the existing reattach regression to both styles; collapse reproduced zero placeholders (red0.317s). The narrow filter exemption preserves queued/loading placeholders while parked attached rows remain excluded. Both-style regression plus parked/two-focus/physical-click/threshold selection PASS0.343s; reader inventory and diff checks pass. No new lifecycle or policy; only admission matches the existing contract. Original running suite lease-1801226722 remains its exact older snapshot, not a pass on this fix. Next: corrected source commit, full close rerun, restage and record suite on the corrected pushed source before landing.

### 2026-10-22 — final SHIP, accepted stage, retained suite

Full close round6 EXIT0/SHIP; BR-1/2/3/4 disposed, zero new findings, evidence4f5add0941ba2bed4552c5a7a42467068a1cc168 pushed. Operator second-look acceptance already recorded. Corrected source/stage22b6e77554789861539120c0e5ca28d4c180ddc5 verified all4 binaries modified=false after placeholder fix. Prior source435148c0 recorded suite PASS14m21s, log sha256 prefix2fceb69bdaa0 (complete log under .git/sdlc/local-checks/435148c06892bc8db5d30aef95c70b1ca005ed93/suite.log); this is prior-source evidence only. Corrected exact-source suite retained session9797, lease-3545135467, attempt71aeab195c9a59de7e5016e7ffb4fa08, snapshot /Volumes/TestScratch/local-checks/sdlc-local-2859257579, priority queued position2. No active merge runner. Next: require this exact-source suite pass, then sdlc merge; preserve #789's separate queued lease-3428295058 and return for its fixed review/merge when available. After both land, claim #819 before #811/#812. No new feature code while waiting.

### 2026-10-22 — inactive-owner count styling correction

Operator TL5ca1126d requires restored inactive-owner dimming and a new quick stage look before landing. The synthetic count draw path bypassed all ownership styling; ordinary chips retained SGR2. Render regression red0.320s measured missing50% explicit shade in physical cells. Reused existing blend/quantize logic: quiet inactive chips, glyphs and counts share at least50% background blend with stronger idle fade retained; preserve themed attention/red and placeholder cues, unknown-palette faint fallback, NO_COLOR and hub reverse. Active-owner chips/counts stay unchanged. Focused collapse/fade selection PASS0.359s; expanded dark/light/unknown/NO_COLOR/256-color/cue and hit-map tests pass in whole couchtty45.582s. Canceled only obsolete queued22b6e775 suite through lease-3545135467 before execution, with no result; #789 remains untouched. Earlier operator works acceptance is recorded, but this styling revision requires the requested quick look. Next corrected-source suite, close review and restage; do not land before new look and green gates.

## Revisions

### 2026-10-22 — operator priority and default (TL0afd9a82)

Collapse becomes the default launch style; multiline remains available. Add #810's wrap-induced resize-without-repaint mechanism to the Spec and acceptance tests. This supersedes the initial multiline-default sentence. The single-row policy removes content-driven chrome-height transitions, while real terminal-width changes still resize PTYs; it does not claim to repair every possible idle-admission stall.

### 2026-10-22 — detailed policy proposal, awaiting TL approval

The durable plan selects the original Spec's scoped-switcher alternative rather than in-row expansion, and makes owner+group scoping exact rather than fuzzy label matching. It proposes folding quiet inactive hubs themselves and preserving attention-first scarcity with +N !; these refine the earlier all-:0-visible wording and acknowledge physical width limits. Policy state/recovery, work bounds, multiline compatibility and native geometry tests are specified in the plan. No code until approval.

### 2026-10-22 — TL approval (3fa9a5f6)

TL approved default one-row collapse, explicit unchanged multiline packer, quiet inactive hubs folding, active/attention priority, +N ! fallback at impossible widths, and +N opening the switcher exactly scoped to owner hub and group with Escape clearing scope. Proceed to change-code; the staged operator look remains required before landing. This accepts the detailed policy proposal above.

### 2026-10-22 — checklist separates implementation from acceptance

The implementation and verification-submission checkboxes are complete. Recorded-suite success, close verdict, staged operator look and landing remain explicit acceptance gates outside those checkboxes, preventing circular claims that close/landing already happened before requesting close. Done-when is unchanged.

### 2026-10-22 — BR-1 clarifies role and aggregation identity

Pin primary and numbered hubs under the same approved current/inactive policy; collapse aggregation uses exact owner+group rather than multiline display runs. This corrects implementation coverage, not approved behavior or scope.

### 2026-10-22 — operator stage refinement (TL02345065)

Collapse only when at least three members of the exact owner/repository group are eligible to fold. One or two quiet members remain individual chips; active/attention/current-hub anchors remain protected and do not count toward that threshold. Preserve one aggregate per canonical repo and owner as displayed; investigate the observed echo pair rather than merging aliases or ownership domains. Add threshold and identity regressions, then restage for a second operator look. This supersedes the earlier any-positive-count folding policy; the one-row geometry, scarcity overflow and exact scoped-menu contracts remain unchanged. Prior stage was not accepted. The old queued suite on93b8b8b6 was canceled through its exact lease before executing to avoid spending priority capacity on the superseded policy; #789's independent lease is untouched.

### 2026-10-22 — whole count hit target (TL458efcf0)

The entire count chip, including every digit, opens the exact scoped menu. Root cause is shared textwidth's hardcoded rune ranges: sofa+VS16 measured one column while terminal.StyledRows renders two, shifting following spans left. Replace that incomplete metric with the existing terminal grapheme metric and make rowtext fitting share grapheme-safe traversal. Pin actual terminal-cell columns rather than deriving test coordinates from the same faulty width helper, for +4 and +12 after an emoji alias. Shared textwidth/rowtext APIs retain their callers and sanitizer ownership; no hitbox padding or timeout change. The threshold is exactly AT three eligible members (two stays, three folds), confirmed by TLf6e4f6f0.

### 2026-10-22 — live source and focus-independent anchors (TL22b772d4)

Counts continue to mean hidden live members, not total inventory. Parked/non-live inventory rows never enter collapse candidates; the shared menuRows view remains authoritative (a completed reattach may legitimately mark a row live). Unknown-inventory attached panes retain their existing attachment fallback; queued/loading placeholders never fold. Collapse anchors derive from the launching hub/owner, not the selected actor's owner; focus may change dimming but cannot change anchor policy. Selecting a hidden member legitimately removes it from +N, while switching between outside tabs preserves membership/counts. Pin a parked attached member and the same live group under two active hub tabs. Switcher inventory remains complete. The earlier OtherHub-derived anchor policy is superseded.

### 2026-10-22 — BR-3 full-boundary validation

Reconcile main contract prose with the approved stage refinements recorded above: threshold at three, live-only membership, launch-hub anchors independent of focus, hidden-live count semantics and whole-chip clicks. Run full sdlc close rather than the fix-only recipe because these are operator-directed policy changes after the earlier SHIP. No source delta or test rerun is needed for this documentation reconciliation.

### 2026-10-22 — inactive-owner styling regression (TL5ca1126d)

Restore visible inactive-owner styling in collapse, including synthetic counts, and retain it in multiline. The count branch accidentally bypassed OtherHub styling; this was not an intentional ownership-policy change. Quiet inactive-owner labels/counts/glyphs use a 50% blend toward the known terminal background, retaining stronger existing idle fade. Preserve attention/theme-red/placeholder cue precedence with existing dim fallback; unknown palette uses existing SGR2 dim, NO_COLOR disables owner fading. Current-owner chips/counts remain unmodified. Reuse the existing blend/quantize logic through one shared helper; pin actual rendered cell styles and hit-map stability in both styles, dark/light/fallback/no-color/attention cases. Cancel only obsolete queued796 lease before it executes, then record corrected-source suite and restage for quick look; #789 stays untouched. This supersedes the accepted stage's styling only; #819 remains separate.
