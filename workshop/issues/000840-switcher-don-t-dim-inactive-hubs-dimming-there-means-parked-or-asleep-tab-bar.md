---
id: 000840
status: codecomplete
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'c8904b6877e87424c5822cb1d5027bec94984193' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T12:07:39-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "121c034b", done: "3487d723"}
actual_hours: 0.13
---

# Switcher: don't dim inactive hubs (dimming there means parked or asleep); tab bar keeps it

## Problem

## Spec

**Don't dim inactive hubs in the switcher (operator, 10-23).** The tab bar dims an inactive hub (50% background
blend, operator decision 10-22) to show where work is happening, and that stays. The switcher applies the same dimming,
which is confusing there: the switcher already uses dimming to mean a parked (or asleep) thread. Remove the
inactive-hub dimming from the switcher only; parked/asleep dimming there is unchanged.

## Done when

- In the switcher, an inactive but live hub renders like any other live row; a parked or asleep row is still dimmed
  (render test).
- The tab bar still dims inactive hubs (existing test passes).

## Plan

Quick-flow, one boundary; keep this small plan in the issue. TL requested claim and plan first; send the committed plan before implementation.

`renderRootMenuFrame` currently resolves the active owner, records `rootLine.otherHub`, and applies SGR faint to every row belonging to another hub. This is a switcher-only presentation pass. The tab bar derives `StatusActor.OtherHub` separately in `console_presentation.go` and fades through `inactiveOwnerStyle` in `idle_shade.go`.

Remove the switcher's active-owner scan, `otherHub` field/assignment and faint wrapper. Keep the existing ownership sections, per-row owner labels, hub underline, selection, red/attention cues, and lifecycle/age/idle styling. Both hub and worker rows lose only the owner-relative dim. The pure renderer owns the change; no new setting, shared presentation flag, lifecycle state or IO seam is needed. Disabling shared ownership styling would also change the tab bar, so reject that alternative (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

| Scope | Plan |
|---|---|
| Pure rendering | Modify only the owner-relative branch in `cmd/internal/couchtty/menu_render.go`; reuse existing cell parsing helpers in tests. |
| Integration | Existing menu inventory/ownership projection and terminal rendering adapters are reused unchanged; no external dependency or new fake. |
| Observable behavior | Owner focus no longer changes switcher rendition. Parked/non-live age treatment, selected-row styling and other existing status cues still compose normally. Tab-bar owner dimming stays. |
| Operating envelope | Existing per-row rendering remains; deleting one inventory scan and per-visible-row replacement adds no growth driver. Render N=8/2N=16 owned live rows and assert one extent per row, equal widths and no owner-faint cells; no timing claim or new scan. |
| State and recovery | This changes a pure projection, not an actionable state or recovery path. Existing switch/select/park/asleep actions and their lifecycle authority are untouched (ARCH-ORDER). |
| Security, formats, residue | No new input, secret, persisted format, handle, background work or durable runtime artifact (ARCH-SECURE, ARCH-EVOLVE, ARCH-FUNERAL). Stateful fake requirements do not expand because the change stays in the pure renderer (ARCH-MOCK). |

- [x] Update `TestHubSwitcherShowsOwnershipAndFadesRelativeToActivePane` in `hub_section_test.go` to the new contract, with both active owners, selected/unselected hub and worker rows, and color/fallback/NO_COLOR cases. Assert actual parsed terminal cells have no owner faint while ownership labels, underline/selection and extents stay intact. Run the new contract RED before changing production.
- [x] Remove the isolated switcher owner-dim mechanism, then run GREEN. Add parked/non-live controls in both ownership sections using the existing age-ramp fixtures; retain ordinary live-idle fading, blocked/attention and hub-underlining controls. The current renderer has no separate asleep styling branch; preserve its non-live path and upstream classification rather than introducing another lifecycle rule.
- [x] Keep and run the existing tab-bar preservation tests, including `TestInactiveOwnerChipsAndCountsFadeInBothStyles`, `TestInactiveOwnerStylingKeepsCuesFallbacksAndHitMaps`, and `TestHubOtherFadePreservesStatusAndHitSpans`. Add the N/2N row/extent/cell assertions to the render regression. Run targeted switcher/idle/age/hub tests and repolint/godoc preflight.
- [x] Update the emitted Couch skill (`cmd/internal/couchcmd/skills/couch/SKILL.md`, several-hubs paragraph) and existing hub/switcher atlas paragraph so guidance stops claiming that the switcher dims other hubs. Preserve the tab-bar contract. Audit embedded-skill assertions/readers; run `make test-affected-inventory` before close and repair only named, audited readers.
- [x] Publish a reviewable candidate, record the hub's no-stage/render-test acceptance disposition, and start the recorded affected check on the pushed head.

Remaining release gates: require passing recorded verification, use the binary-owned close review, and land through SDLC. Report the PR outcome to hub. No stage/:0 operation is authorized or needed.

## Log

### 2026-10-23
- 2026-10-23: closed — Rendered-cell regression RED before fix 0.662s, GREEN after: 8/16 live and parked rows retain cells and hit extents across both active owners, selection and palettes; parked age colors preserved and fresh live rows not owner-faint. Existing switcher/hub/idle/age/tab-bar controls PASS2.597s; targeted race PASS6.230s; emitted skill PASS0.520s; lint and reader inventory PASS. TL accepts render tests, no stage. Recorded affected check on implementation 9e51009dd5e302e716c76152ec3b14f9346e3615 is queued as lease-3944285118; close overlaps retained run per tl-worker, final pass required before landing. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

Claimed pair:9 and ran start-plan on a fresh issue branch from main after #833 landed. Source tracing identified the switcher-only `activeHub`/`rootLine.otherHub`/SGR faint pass and the separate unchanged tab-bar `OtherHub`/`inactiveOwnerStyle` owner. Existing switcher test currently requires foreign-owner dim and must be changed with a RED regression. The embedded Couch skill repeats that behavior and is included in scope. No production or test code changed; plan disposition pending.

Baseline `go test ./cmd/internal/couchtty -run 'TestHubSwitcher|TestSwitcherNonLiveRowKeepsItsAgeRamp|TestInactiveOwner' -count=1` PASS (0.363s). This confirms the existing contract before the planned test change; no new-behavior pass is claimed.

TL approval 72f66e0a-39d6-4317-abd7-896583521754 authorized this exact scope and render-test acceptance with no stage. change-code admitted quick flow. New `TestHubSwitcherOwnershipDoesNotFadeRows` ran RED before production edits (0.662s), exposing added faint cells on both live and parked foreign-owner rows. Corrected fixture assumptions before final RED: five-day age is AgeDays, and selected row/key must be explicit after ownership reordering. Removed only active-owner scan, rootLine.otherHub/assignment and owner-faint wrapper; tab-bar implementation untouched. GREEN switcher/hub/age/tab-bar controls 2.597s; targeted race 6.230s; emitted-skill controls 0.520s; repolint 9.517s and stacked-godoc preflight 0.636s. N=8/16 cell/extent fixtures preserve parked age colors, selections, owner labels and geometry. Reader inventory PASS without repairs; docs corrected. Recorded verification and close/landing remain pending. #841 is queued by TL for after this issue; no work on it is stacked here.

Close round 1 SHIP/high, no findings, evidence 51ed03b61dea1b639d8c94f2f08140c0655a9081; automatic measured actual 0.13h. Entered merge promptly; readiness correctly refused pending CI/local verification and directed retention of the existing check. Retained lease-3944285118 through its FIFO wait. Recorded affected PASS on implementation 9e51009dd5e302e716c76152ec3b14f9346e3615: 4 packages plus shell, exit 0 in 3m20s, log SHA256 prefix 57146763a732. No code changed after that tested head; CI passed on the close head. Resuming landing without stage or bypass.

## Revisions

2026-10-23: TL accepted render-test qualification and directed no stage (72f66e0a-39d6-4317-abd7-896583521754). Split completed acceptance preparation from the downstream verification/close/landing gates to avoid a self-referential unchecked close task. Acceptance scope is unchanged; a recorded pass still precedes landing. Candidate PR #565, implementation 9e51009dd5e302e716c76152ec3b14f9346e3615.
