---
id: 000789
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'a5dba37ce0ebc1e4e90f2cd6bbe6d1616c736532' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T06:59:35-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "5b879924", done: "e2fe8944"}
---

# Couch chrome: cap slot status text at 40 cells (ellipsis) and shorten built-in status messages to fit

## Problem

## Spec

Operator, accepting pair#779 (10-21 19:1x): "one caution is we should limit the status text to be certain length,
otherwise, a long text will wrap into 10 lines, destroy the layout? maybe limit status message to be 40 chars,
maybe another task to clean up messages showing up there to be within that limit".

pair#779 caps the tab bar at two rows, but a slot's status text in the chrome can still be arbitrarily long and
crowd out every other tab. Two parts:
1. **Display cap:** the chrome renders a slot's status text at most 40 display cells (rune-width aware), cut with
   `…`; the full text stays available where it already is (peek, the switcher or hover). One pure function,
   unit-tested, used by every place the chrome draws status text.
2. **Producer clean-up:** enumerate status-text producers (wait notices, lifecycle labels and cast actions),
   and keep every built-in template, including its finite built-in names/glyphs/variants, within 40 cells.
   Arbitrary operator names, paths, external errors and shared full diagnostic evidence remain unmodified
   upstream; the display cap handles their expansion. Test actual finite variants and callback outputs as
   well as source templates, so fixed values cannot disappear as if they were unbounded diagnostics.

## Done when

- A 200-character status renders as 40 cells ending in `…`, with layout unchanged (test); full text still reachable.
- No built-in template exceeds 40 cells, counting complete finite built-in variants such as cast/link names and capability glyphs/actions (source catalog plus runtime variant/callback tests). Arbitrary names, paths and error details, and the shared automatic-input-stall hub/desktop diagnostic, stay full upstream and receive the same 40-cell display cap.
- Ctrl+Y copies the full current capability diagnostic ahead of an unrelated last feed notice; selected operation/error and held retry payload retain precedence. Full feed messages remain copyable after display clipping or expiry (tests).

## Plan

- [x] Add a pure 40-cell sanitized/ellipsized chrome projection, used for both notice fields; retain the source text and use the existing Ctrl+Y copy action for full capability diagnostics too.
- [x] Inventory notice producers, shorten built-in chrome instructions, and enforce their literal/template contribution with an AST-derived catalog; dynamic names, paths and diagnostic suffixes retain their full source text.
- [x] Pin ASCII/CJK/combining/control inputs, shared legacy/two-row rendering, full-text copy and source-catalog negative controls; submit the required recorded check. Close and merge remain gated workflow steps, not implementation-checkbox evidence.

## Log

### 2026-10-21

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
