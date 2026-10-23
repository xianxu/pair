---
id: 000840
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '20e7dbb160a500159b1b5c55b9d61346df1a049d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T12:07:39-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
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

- [ ] Update `TestHubSwitcherShowsOwnershipAndFadesRelativeToActivePane` in `hub_section_test.go` to the new contract, with both active owners, selected/unselected hub and worker rows, and color/fallback/NO_COLOR cases. Assert actual parsed terminal cells have no owner faint while ownership labels, underline/selection and extents stay intact. Run the new contract RED before changing production.
- [ ] Remove the isolated switcher owner-dim mechanism, then run GREEN. Add parked/non-live controls in both ownership sections using the existing age-ramp fixtures; retain ordinary live-idle fading, blocked/attention and hub-underlining controls. The current renderer has no separate asleep styling branch; preserve its non-live path and upstream classification rather than introducing another lifecycle rule.
- [ ] Keep and run the existing tab-bar preservation tests, including `TestInactiveOwnerChipsAndCountsFadeInBothStyles`, `TestInactiveOwnerStylingKeepsCuesFallbacksAndHitMaps`, and `TestHubOtherFadePreservesStatusAndHitSpans`. Add the N/2N row/extent/cell assertions to the render regression. Run targeted switcher/idle/age/hub tests and repolint/godoc preflight.
- [ ] Update the emitted Couch skill (`cmd/internal/couchcmd/skills/couch/SKILL.md`, several-hubs paragraph) and existing hub/switcher atlas paragraph so guidance stops claiming that the switcher dims other hubs. Preserve the tab-bar contract. Audit embedded-skill assertions/readers; run `make test-affected-inventory` before close and repair only named, audited readers.
- [ ] Publish a reviewable candidate and request the hub's live-check disposition before any stage/:0 operation; this visual change can be checked on an approved build with two live hubs plus a parked row, while confirming tab-bar dimming. Run the recorded affected check on the pushed head, use the binary-owned close review, and land through SDLC after passing verification and the hub's disposition. Report the PR outcome to hub.

## Log

### 2026-10-23

Claimed pair:9 and ran start-plan on a fresh issue branch from main after #833 landed. Source tracing identified the switcher-only `activeHub`/`rootLine.otherHub`/SGR faint pass and the separate unchanged tab-bar `OtherHub`/`inactiveOwnerStyle` owner. Existing switcher test currently requires foreign-owner dim and must be changed with a RED regression. The embedded Couch skill repeats that behavior and is included in scope. No production or test code changed; plan disposition pending.

Baseline `go test ./cmd/internal/couchtty -run 'TestHubSwitcher|TestSwitcherNonLiveRowKeepsItsAgeRamp|TestInactiveOwner' -count=1` PASS (0.363s). This confirms the existing contract before the planned test change; no new-behavior pass is claimed.
