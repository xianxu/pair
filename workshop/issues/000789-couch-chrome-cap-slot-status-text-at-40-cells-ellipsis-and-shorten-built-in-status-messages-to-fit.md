---
id: 000789
status: codecomplete
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'b82aaceb3d61a64c79cc597f4caee0b61f3db0e2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T06:59:35-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 3.19
---

# Couch chrome: cap status text at 40 cells and preserve full copying

## Problem

## Spec

The operator's layout requirement is a maximum of 40 display cells for chrome status text, with an ellipsis when clipped. One pure function sanitizes terminal text and applies that cap, shared by Notice and CapabilityNotice in the legacy and composed status renderers. Source feed and diagnostic text remain complete.

The existing switcher Ctrl+Y action recovers full text in this order: held copy-retry payload; selected operation diagnostic/error; current capability diagnostic; last retained feed notice. This preserves selected failure/retry behavior while making a clipped capability diagnostic recoverable.

TL receipt58326574-1d4c-4767-b222-cfb4d5adddce splits the original request. Built-in wording cleanup is now the separate low-priority #802, limited to a handful of common messages with focused tests and no total-catalog proof. #789 contains no producer rewrites or exhaustive producer-catalog enforcement.

## Done when

- A 200-character ASCII status renders as 40 cells ending in `…`; short/exact-boundary, wide-rune, combining and control inputs remain bounded and terminal-safe, through both renderers, with the existing layout unchanged (tests).
- Full feed text remains available to the existing copy action after clipping/expiry; Ctrl+Y copies the complete current capability diagnostic before an unrelated feed notice, while selected operation/error and held retry payload keep precedence (tests).
- The final diff contains the display cap, full-copy behavior, their tests and documentation. Producer wording and exhaustive catalog proof are excluded by the authorized split and tracked in #802.

## Plan

- [x] Implement one pure 40-cell chrome projection and keep full source diagnostics.
- [x] Preserve full-text copying with selected-error and retry precedence; document both README surfaces.
- [x] Re-run focused display/copy tests and submit the corrected recorded check; close and merge remain gated on review and check results.

## Log



- 2026-10-22: closed — TL58326574 cap/full-copy-only scope; producer/catalog removed and #802 filed unclaimed. BR5 screen-assertion family swept: both known cases red→green, bounded screen + unmodified feed + actual clipboard assertions retain security and exact-handle invariants. Whole couchtty package PASS41.445s, count1, no#788; targeted fixes PASS1.042s. Reader inventory/diff checks pass; no source wording or fixture deadline changed. Recorded final suite352190cb queued lease-823433472, not claimed passing; landing awaits pass. No stage approved.; review verdict: SHIP
- 2026-10-22: flow upgraded quick → full — an earlier round of this close already ran the full review
### 2026-10-21


### 2026-10-22 — drain checkpoint and recorded-suite failure
- 2026-10-22: closed — Prescribed fixed review after main integration: only generated tests/affected-readers/cmd/internal/couchcmd/run_test.go.json conflicted, both sides fixed_scanner; preserved all merged source/test changes, regenerated certificate against merged source, startup3 PASS1.466s and reader inventory/diff checks passed. Exact integrated head b2ed415e recorded full suite PASS11m26s. Display cap/full-copy-only scope unchanged; no producer rewrites, new deadlines or bypasses. Existing TL no-stage approval retained; #796 independently landed, #819 remains separate after789. (fixed-then-ship; full suite: Recorded suite on integrated b2ed415eae449b0ef4d93c108e2520355c2430f7 PASS11m26s, attempt a71b7262e10d47a252d630372095d2ff, log SHA256 prefix7a1202907b41, saved .git/sdlc/local-checks/b2ed415eae449b0ef4d93c108e2520355c2430f7/suite.log)
- 2026-10-22: closed — Caller-package assertion class: original three couchcmd failures reproduced61.851s; independent bounded paint literals plus full dispatched cause/root/diagnostics retained; actual native fixture Ctrl+Y preserves complete borrowed warning. Targeted3 PASS2.204s; whole couchcmd PASS114.700s; reader inventory/diff checks pass. Replacement recorded suite queued on8727e94da; previous suite failure retained, no flake/bypass. No production or deadline changes in fix. (fixed-then-ship)

Retained suite session55007 completed with exit1: attempt f9b3e2af722df79bce43bf1cf59e89dd on352190cbc25d79b54e6ad240f6bcfe9e64af8415, 12m13s after queue admission, log SHA256 prefix b45a2de39b81. Retained log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/f9b3e2af722df79bce43bf1cf59e89dd.log. Three unknown failures, all in couchcmd: TestHubStartupLaterFailureLeavesFirstUsable and TestHubStartupLaterFailureAfterWarmWorkerLeavesFirstUsable timed out waiting for a >40-cell refusal prefix; TestInitialHubBorrowedAttachmentNoticeAppearsOnFirstPaint expected the entire warning while captured paint shows the bounded borrowed-attachment warning. Whole couchtty passed40.244s in this attempt. This is not a suite pass and these failures are not registered as flakes.

Drain requested an atomic checkpoint; it was lifted while inspecting these completed results. No queued work was restarted or reordered. Merge session46377/pid83174 remains retained, last position1, blocked on the missing passing local record (and the #803 fleet grant); it cannot land this failed head. Next step: reproduce the three couchcmd assertions, check their whole screen-assertion family, retain complete diagnostic recovery checks, then submit a corrected pushed head to the recorded suite and close review. #796 remains unstarted until #789 lands; #802 remains the separate unclaimed low-priority wording item.


### 2026-10-22 — caller-package screen assertions corrected

The three recorded failures reproduced unchanged in a targeted couchcmd run (61.851s). Read-only audit of all couchcmd terminal-write and decoded-screen assertions found two incompatible sites covering all three failures; other long path strings belong to switcher content or uncapped notification transport. Hub-startup tests now expect an independently written 40-cell ellipsis literal while retaining first-hub input/output usability, warm-worker counts, and the complete dispatched failure's injected cause, root and diagnostic-path facts. Initial borrowed-attachment paint expects its independent bounded literal, retains absence on failed adoption and warm-pass arming, and sends real Ctrl+Y through the existing Console input path to a private native pbcopy fixture to prove exact full start.Warning() recovery.

An attempted hub-notice clipboard assertion correctly exposed that the asynchronous warm-worker borrowed notice can become the latest notice before the copy; it was removed rather than forcing production ordering or weakening latest-notice semantics. The stable initial-paint fixture owns the full-copy check; hub failure's original full dispatched error remains asserted. No production or fixture deadline changed. Corrected targeted three tests PASS2.204s; whole couchcmd package PASS114.700s, count1. Affected reader inventory and diff checks pass. Full repository suite remains pending on the forthcoming corrected pushed head; the previous failed suite is retained as failure evidence, never relabeled as a flake/pass. The existing merge runner is preserved awaiting the replacement local-check record.


### 2026-10-22 — fix review SHIP and replacement suite queued

Fixed-then-ship close round5 returned SHIP/high confidence, zero new or open findings. Reviewer independently passed the three corrected cases, whole couchcmd104.454s, reader inventory and whitespace checks; production behavior, Spec and Done-when are unchanged. SDLC published codecomplete bound to evidence commit6f4c3d3403bc283b40ac676aa610fcce9fddc58c and adopted measured actual2.37h.

Replacement recorded full suite is already queued on pushed8727e94da57d888272eaa1a855a12e6b334f87ff: session80343, lease-40475742, attempt247075fa5a5ea07c61206b544f4d06d4, isolated checkout /Volumes/TestScratch/local-checks/sdlc-local-1857757870. Initial and latest observed normal FIFO position9; no tests have executed yet. Existing merge session46377/pid83174 remains retained first waiting for the passing local-check record, with no duplicate runner or queue changes. Next step is to consume that result, satisfy any real failure, and let the retained merge integrate/main-test/land. No stage is required. #796 remains unstarted pending this landing.


### 2026-10-22 — replacement full suite passed; retained landing ready

Replacement attempt247075fa5a5ea07c61206b544f4d06d4 on8727e94da57d888272eaa1a855a12e6b334f87ff completed EXIT0 in12m7s, recorded as sdlc-local/suite. Passing log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/8727e94da57d888272eaa1a855a12e6b334f87ff/suite.log, SHA256 prefix0ff964e18e4e. Retained merge session46377/pid83174 reports ready at6c95e1692 for PR543; latest hold is #773's fleet-blocking grant until14:32:08 local. No duplicate runner or queue change.

TL authorized starting #811 from main only if the checkout switch cannot disturb #789. The suite is finished and was snapshot-isolated, but the existing unqualified merge runner reads branch selection after admission (per sdlc merge contract), so switching this checkout could select the wrong issue. Keep #789 checked out and preserve its runner. Next step: consume admission/integration results and land #789; then claim #811 (Ctrl+Return opens the attention list like the bell; Ctrl+Backspace remains previous tab), followed by #812 and #796.


### 2026-10-22 — admitted integration certificate conflict

Retained merge session46377 survived restart and acquired admission after #773. It exited1 on integration of main651e30092e7ca245e5537ba1450b7ea7eb36ddae: only tests/affected-readers/cmd/internal/couchcmd/run_test.go.json conflicted, with both sides retaining fixed_scanner but different source hashes. The automatic merge was aborted by sdlc; reran the prescribed manual main merge. Production and test source merged without conflicts. Re-audited the merged caller test's repository-reading seam and regenerated that certificate through make affected-readers using its unchanged fixed_scanner disposition, preserving both sides' test changes. No test/source content was chosen away to resolve the certificate.

Next: finish merged-head targeted checks/inventory, commit and push this integration, record its authoritative full suite, then follow the requested fixed-then-ship close with full-suite evidence and re-enter sdlc merge. The prior runner has exited; no duplicate remains. #796 (default collapse, multiline option, #810 mechanism recorded in Spec) is still next after landing; #811 and #812 follow it.


### 2026-10-22 — integration suite queued; authorized parallel planning

Integrated-head startup three-test selection PASS1.466s, reader inventory and diff checks pass. Recorded integration suite onb2ed415eae449b0ef4d93c108e2520355c2430f7 is queued: session61444, lease-3428295058, attempta71b7262e10d47a252d630372095d2ff, isolated checkout /Volumes/TestScratch/local-checks/sdlc-local-3878503325. Initial normal FIFO position16; no full-suite result claimed. The prior merge runner has exited, so none can select or mutate this checkout on admission.

TL6a768f10 explicitly authorizes starting #796 from main while this immutable snapshot suite waits. Preserve #789's branch and check session; return here when the result arrives, perform requested fixed-then-ship review with passing full-suite evidence, then re-enter merge. #796 plan first: collapse default, multiline available, and #810 resize/no-repaint stall mechanism documented. No stacked branch development.

### 2026-10-22 — retained integration suite PASS; resume landing

Retained session61444/lease-3428295058/attempta71b7262e10d47a252d630372095d2ff completed EXIT0/PASS11m26s on integrated headb2ed415eae449b0ef4d93c108e2520355c2430f7, log SHA256 prefix7a1202907b41, retained .git/sdlc/local-checks/b2ed415eae449b0ef4d93c108e2520355c2430f7/suite.log. No cancellation, duplicate or unrecorded retry. #796 landed as PR547 and its stage clear completed with pair:0 atmain09c8964de39cdb5e6bd9db20601c232446af6949; pair:9 has the753 staging go. TL9536d81d directs789 next, then819 before811/812. Returning to this independent branch: run prescribed fixed-then-ship review with this passing full-suite evidence for the sole generated-reader-certificate integration conflict, then promptly sdlc merge so its turn owns further main integration. No stage needed for789 per existing TL waiver; no producer wording work or819 scope added.

### 2026-10-22 — fixed review SHIP; newer-main atlas conflict

Prescribed fixed close round6 EXIT0/SHIP, no findings, evidence58c2bc3852362edc3c0c6ba7737cf63edf78e23c published. Reviewer independently checked certificate/source hash, fixed_scanner coverage, all21 inventory tests and whitespace. Entered merge promptly; retained merge session4425 vouched ready, admitted, then EXIT1 on sole atlas/couch.md conflict integrating main09c8964de39cdb5e6bd9db20601c232446af6949. SDLC aborted that attempt and prescribed manual merge, full suite and fixed review.

Reran one manual merge of that main. All source/test changes auto-merged; only the atlas introduction overlapped #796's new tab-style section with789's cap/copy description. Retained the complete collapse/multiline/geometry/scope description and added both-style40-cell/full-copy semantics; removed the obsolete unconditional two-row claim. Reader inventory PASS; integrated cap/collapse/owner/pending/copy selection PASS0.539s. Diff against incoming main passes whitespace; cached whole-merge check reports one existing main-owned trailing space in797's archived review, preserved unchanged rather than rewriting unrelated history. No code conflict or new design/source fix. Next commit/push this integration, record required exact-head suite, fixed review with pass, retry merge; no active merge runner remains. #819 waits for789 landing.

## Revisions

### 2026-10-22 — implementation scope

The cap counts display cells including the ellipsis, after row sanitization. It applies to Notice and CapabilityNotice in the shared renderer, never to actor labels, menu details, feed storage or error records. Catalog enforcement counts built-in literal/template text; dynamic labels/errors may exceed 40 and are bounded only in their chrome projection. Ctrl+Y retains full feed text and copies the currently displayed capability diagnostic when no selected operation/error overrides it.

### 2026-10-22 — design and producer inventory

ARCH-DRY/ARCH-PURE: reuse rowtext.Sanitize/Fit and textwidth.Width behind one pure ChromeStatusText function; both legacy and composed status renderers share renderStatusCells. Alternatives were truncating producers/storage (loses detail), or relying on terminal-width clipping (does not enforce 40); neither meets the contract. Work is O(B) in input bytes per projection, with no IO, timers, retained state or persisted changes. Existing status packing counted tests retain their bounds; counted input-size tests pin the shared projection's scan bound at N/2N. ARCH-SECURE retains the existing sanitizer before width measurement. ARCH-MOCK/STATE/FUNERAL/EVOLVE add no external effects, lifetime owner or persisted format; copy reuses the current clipboard seam.

Producer inventory: Console.setNotice and ExitNotice; cast capabilityNotice and driveNotices; broadcast/pointer/webcast/cast-code instructions; action/lifecycle refusals. Couchcmd also forwards lifecycle/supervision/stale-session/archive/renice/crash diagnostics as arbitrary full-text Notice values. Menus and detailed diagnostic records remain unrestricted. A source-derived test covers direct setNotice/Notice.Body templates plus named helper producers, with negative controls; runtime substitutions use the display cap. This is presentation policy, not a lint ban on diagnostic length.

Catch-up generated a separate merge-check.yml change (SIGPIPE-safe metadata grep); preserved in stash 'pair789: preserve catch-up generated merge workflow separately', excluded from this issue.

### 2026-10-22 — implementation and targeted evidence

The shared projection and Ctrl+Y capability copy path are implemented. ASCII 200-cell, exact-boundary, CJK, combining, control-sequence, legacy/composed renderer, layout, retained feed, and clipboard tests first exposed the missing cap/copy behavior and now pass. N/2N scan-accounting bounds admit at most three linear text passes. Artifact source classification passed (2.675s); affected reader inventory passed. Catalog negative controls detected 25 old long templates (up to 235 cells); the final source catalog covers direct/local/formatted/helper/range/map notice production in couchtty and couchcmd. It measures fixed template contribution; dynamic names/errors retain their full values and receive the display cap. Shared automatic-input-stall hub/desktop diagnostic evidence is explicitly retained whole upstream, not treated as a concise status template.

Fresh producer/catalog selection passed 9.629s; upstream producer tests passed 11.012s, and root independently verified supervision/renice/lifecycle/archive selection in 15.930s. An earlier concurrent selection compiled old wording assertions and timed out waiting for that removed text; assertions were updated, with no production or fixture deadline changes. A further test pins supervision's precise hub/user-input boundary and actionable peek instruction instead of ambiguous 'quiet'.

Final root status/notice/capability-copy/catalog/linear-work selection passed 3.006s after the final wording changes; source-classification and final affected inventory passed.

### 2026-10-22 — acceptance mode and recorded check

TL receipt 99cf6f2f-0eb6-4d4b-ac27-4f406820728f approves landing on cell-level pins and tests with no stage window; the operator looks at the next restart onto main. No primary checkout changes are authorized or needed for this issue. Correction: #737 does not currently hold pair:0; its future live run remains in the operator queue.

Recorded affected run on pushed 552007c635eeadaa74cef34467d589457f4d8815 selects the full suite because scripts/test-affected.py changed to register the fixed source-catalog preflight. Retained session63160, lease-1866031189, attempt966250dd27cfef1ce823807b98fe8242; queued normal FIFO position7 at submission. This is not a suite pass. Close review may overlap the queued check per tl-worker; landing still requires its passing record. Implementation checklist wording now distinguishes submitted verification from those remaining workflow gates.

### 2026-10-22 — close round 1 correction (BR-1 through BR-4)

REWORK found that the source catalog discards fixed selector/format substitutions and misses callback returns. The previous claim that source-template enforcement alone covered built-in messages was incomplete. Sweep capability glyph/action variants; webcast view-only/pointer/drive copy and not-ready variants; cast/code rotation variants; and status callbacks. Runtime matrices now form the enforcement boundary for finite program-owned substitutions; AST coverage complements them for direct/local literals. Arbitrary diagnostic substitutions remain unrestricted upstream as planned. The second Done-when clause is reconciled with that distinction and now explicitly requires full capability-copy precedence/recovery. This supersedes the former unconditional 'No built-in status message exceeds 40 cells' shorthand and its incomplete source-only interpretation.

BR-2 retains the lifecycle count invariant while updating its remaining stale assertion. BR-3 updates both README copy contracts. The original queued run lease-1866031189 was canceled through sdlc before admission (session63160 exited1 with lock interrupted/context canceled), with no test result claimed; a corrected head receives a new recorded check. Close round1 did not finalize or publish codecomplete. All four findings remain pending verification in the next review.

### 2026-10-22 — round 1 findings addressed, pending re-review

BR-1: actual runtime producer matrices first failed on 41–50-cell capability/link/rotation variants and the 72-cell pointer callback; a real unrecorded-cast callback additionally failed on the 51-cell drive notice. Three small pure formatters are used by production and tests; all finite variants now fit. The matrix covers annotation/drive × waiting/retryable/broken, all three cast links × copied/not-ready/missing, LIVE/Chrome × link × ordinary/forced/capped/all-code rotation, and real non-nil unrecorded callback states. Complete dynamic suffixes are tested for both capability kinds and failure classes. AST negative controls now cover global missing-link callbacks too; runtime tests are registered in the fixed source preflight. Broad tty regression passed 14.296s; final producer/catalog/dynamic tests passed 0.425s, and root independently passed chrome/copy/catalog/runtime variants in 0.577s.

BR-2: exact TestLifecycleDrainReportsOnlyUntargetedFacts failed on the stale wording and passed 0.344s after retaining its one-untargeted-observation invariant with the new text. No fixture deadline changed. BR-3: both affected README passages now state selected-error/capability/feed precedence and full-text recovery. BR-4: Done-when expressly covers finite built-in substitutions, dynamic diagnostic exemptions, full capability copying and selected-error/held-payload precedence. Root added selected-error and held-copy priority checks to the capability clipboard regression (PASS0.535s). Artifact/reader inventory and diff checks pass. Round2 must verify all dispositions; no SHIP or suite pass is claimed yet.

Corrected recorded affected→suite submitted on pushed 797d86d9108817b9a8d5374460a96a53f96133be. Retained session63594, lease-1433186204, attempt454779b8b84a38b0d1f4264bc9cdea6b; initial normal FIFO position6. Suite is pending, not passing. Close round2 overlaps this queued check under tl-worker; code remains fixed while review/check run.

### 2026-10-22 — authorized scope split (TL58326574)

TL directs simplification: #789 lands the 40-cell display path plus full copying only. The layout invariant is already protected by the cap; exhaustive producer enforcement is removed from Spec, Done-when, Plan and final source diff. The original wording cleanup and its global proof attempts above are superseded. Low-priority unclaimed #802 now owns shortening a handful of common wait/lifecycle statuses, with focused tests and explicitly no total-catalog proof. This authorized change disposes BR-1 by removing its requirement and implementation rather than claiming the incomplete catalog passed.

Restored producer source/wording tests, affected source-preflight registration and catalog-only certificates to the exact branch base; removed AST/variant catalog tests. Retained only ChromeStatusText, its shared renderer calls, full-copy selection, focused tests, artifact-source classification and their docs. BR-2's obsolete producer assertion change is absent in the final diff; baseline wording and its assertion are restored together. BR-3/BR-4 copy documentation and acceptance clauses remain correct. The second queued check lease-1433186204 was canceled through sdlc before execution; no result claimed. No new broad sweep or durable plan is needed after the split.

Split-scope verification: focused display/status/notice/full-copy/capability selection passed 3.916s after restoring all upstream wording. Affected reader inventory and diff checks pass. The existing broken-capability screen assertion now checks its actionable prefix within the cap; the full original diagnostic is pinned through Ctrl+Y. No producer content is shortened by #789.

Final-scope suite submitted on pushed 2ab91b2179ebafb427398c5fde81ff1d2789397a: retained session40889, lease-4050460370, attempted4231a2c8ecb6c217b97ffe37913c11, initial normal FIFO position5. Suite is pending; not evidence of a pass. Close round3 reviews the TL-authorized cap/copy-only contract while the suite waits. #802 creation is verified on origin and unclaimed.

### 2026-10-22 — round 3 screen-assertion family (BR-5)

Review accepted the authorized scope split, withdrew the producer-catalog requirement, and confirmed cap/full-copy regressions fail without their implementation. Only BR-5 remains: existing screen assertions sometimes require complete notices longer than 40 cells. Sweep screen-level notice assertions, check bounded displayed text independently, and verify full diagnostics via retained data/copy; preserve security-generation/stream and borrowed-attachment invariants. Known sites are guessing-rotation feedback and borrowed-attachment feedback. Whole couchtty package verification is now required before resubmitting, not only a name-based focused selection. The suite lease-4050460370 was canceled through sdlc before execution; no test outcome claimed. No production behavior or fixture deadlines change for this test correction.

BR-5 evidence: unchanged guessing-rotation screen test failed in6.02s and exact-returned-handle resume test failed in0.01s. Both now assert independently written40-cell ellipsis text, retain complete feed/menu payload assertions, and exercise actual full-copy recovery through the shared fake-pbcopy fixture. Rotation flash/secrecy/old-generation and exact-handle/claim/projection invariants remain intact. Targeted correction passed1.042s; whole couchtty package passed41.445s with count1/3m diagnostic test limit and no#788 occurrence. No fixture deadline or producer changed. Reader audit: only chrome_status_copy_test.go reads its own temp clipboard fixture; its local/literal certificate refreshed. The other two changed tests perform no repository reads and need no new certificate. Final affected inventory and diff checks pass.

BR-5 corrected recorded suite submitted on pushed352190cbc25d79b54e6ad240f6bcfe9e64af8415: retained session55007, lease-823433472, attemptf9b3e2af722df79bce43bf1cf59e89dd; initial normal FIFO position6. Suite is pending, not a pass. All implementation/verification-submission plan items are complete; close and merge still require their gates.
