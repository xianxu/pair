---
id: 000340
status: working
deps: []
github_issue:
created: 2026-09-28
updated: 2026-09-28
estimate_hours:
card_mirror: '0e09bc509411578aa6f50588394de785a997e1f0' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-28T15:14:24-07:00
flow: {kind: full, provenance: inferred}
---

# Review pane: show Alt+c return hint, and Esc returns to the agent

## Problem

With the Alt+c review pane open, nothing on screen says how to get back to the
agent pane. The operator had to guess Alt+c again. The review pane's bottom bar
(`statusline_text()` in `nvim/review.lua`) shows only
`🪄 <mode> • <file>  L<n>/<N>`: no way out.

## Spec

- **Hint in the bottom bar.** Add `Alt+c → agent` (wording to settle) to the
  review statusline, in both its idle and awaiting (spinner) forms. It must
  name the key that actually works. Alt+c is routed to `PairReviewToggle`
  (`nvim/init.lua`), which hides the floating review pane.
- **Esc returns to the agent pane.** In the review buffer's NORMAL mode, Esc
  does what Alt+c does from there: hide the review pane, and focus lands on
  the agent pane. Insert/visual-mode Esc keeps its vim meaning (leave the
  mode). Esc in normal mode is otherwise a no-op in stock vim, so nothing is
  lost. Floats opened from the review (diagnostic float, definition float)
  keep closing on Esc first, before the pane itself hides.
- Check where focus lands after hiding the floating pane. If it doesn't land
  on the agent pane, focus it explicitly by pane ID (ID-based, never relative;
  see lessons).
- **Document the review buffer's own keys.** The operator also forgot the
  accept/reject keys. They exist (`nvim/review.lua`, buffer-local, normal mode)
  but appear in neither README's key table nor the Alt+h help page:
  - `Alt+a` / `Alt+r`: accept / reject the 🤖 suggestion at the cursor
    (also `<leader>a` / `<leader>r`)
  - `Alt+Shift+A` / `Alt+Shift+R`: accept / reject every 🤖 suggestion in the
    paragraph, up to the cursor
  - `Alt+q`: insert a human comment marker (visual mode: quote the selection)
  - `Alt+Enter`: finish the human turn
  - `]m` / `[m`: next / previous marker
  The full list goes in the Alt+h help (`pair-help`, `cmd/internal/keyscmd`)
  and README's key table, derived from the keymaps' `desc` fields where
  practical, so the list can't drift from the mappings. The bottom bar is
  narrow, so it carries only the most-needed hints: the way out, plus
  `Alt+a/r accept/reject`. Width budget and wording are to settle at design time.
- **Alt+n / Alt+Shift+N step through markers** (operator request): in the review
  buffer, Alt+n does `]m` (next 🤖 marker) and Alt+Shift+N does `[m` (previous),
  both wrapping, both buffer-local, normal mode.
  - Alt+n is free here. Pair's Alt+n reload is draft-only since #333, and the
    review pane installs only non-draft globals
    (`workbench_route.install_global_maps(false)`).
  - Alt+Shift+N is the global "restart the agent conversation" shortcut, and
    the review pane installs it. The buffer-local mapping overrides it in review
    only (the agent restart stays reachable from the draft). The Alt+h help must
    say so: the row for Alt+Shift+N names the review exception.
  - Check under Couch that the review pane actually receives Alt+n
    (Couch's routing may replace Pair's, #284), and with Pair alone.
  - `]m` stops at every 🤖 marker, the operator's own `[H]` comments included.
    Skipping to agent proposals only (`pending` in `review/markers.lua`) is a
    possible refinement, not asked for.
- Update help/README/atlas prose for the review mode's keys (lessons: UI text
  is a public contract).

## Done when

- The review pane's bottom bar shows the return key (and the accept/reject
  hint) in the idle and awaiting states.
- Alt+h help and README list the review buffer's keys (accept/reject,
  paragraph accept/reject, comment marker, finish turn, marker jumps). A test
  fails if a review keymap is added without a help entry.
- Normal-mode Esc in the review pane hides it and leaves focus on the agent
  pane. Insert-mode Esc still just leaves insert mode. A test covers both,
  and fails if the mapping is reverted.
- Alt+n / Alt+Shift+N in the review buffer move to the next / previous marker,
  under both Pair alone and Couch. Alt+Shift+N still restarts the agent from
  the draft pane.
- Live smoke by the operator: open review with Alt+c, read the hint, Esc back to the agent.

### Revised acceptance after operator smoke feedback

The following supersedes the agent-return destination above; the remaining
marker-navigation, mode-preservation and help requirements still apply.

- Draft nvim displays Alt+c review, including while a review is active.
- The review bar advertises Alt+Return submission in both idle/waiting states.
- Esc in review normal mode and Alt+c hide review and focus draft nvim; an
  internal popup closes first on Esc. Insert/visual Esc keeps its Vim meaning.
- Closing scrollback or changelog with Esc or :qa emits annotations before
  hiding the floating layer and focusing draft. A retained review must not
  become visible underneath, and remains available through Alt+c.
- Headless controls, draft status and stacked-viewer regressions cover those
  paths; operator confirms the revised smoke behavior.

## Plan

- [x] Pin idle/waiting status hints, normal-mode Esc, insert/visual Escape,
  diagnostic float dismissal, and wrapped marker jumps in headless review tests.
- [x] Add compact status hints and buffer-local return/jump bindings, reusing
  pane-ID discovery for explicit agent focus after hiding the review.
- [x] Derive review help from mapping descriptions, cover every review map with
  a drift test, and document the review exceptions in README and atlas.
- [x] Teach Couch's existing focus probe to preserve review Alt+n, test the
  real role classification and console forwarding, retaining draft/menu relaunch.
- [x] Run review and affected Go checks, build pair:0; operator smoke: Alt+c
  hints, Esc to agent, insert/visual Esc, Alt+n/Alt+Shift+N wrapping, Alt+h.

Design: keep the existing review mappings and marker navigation authoritative
(ARCH-DRY); add no durable state or processes. The statusline puts exit and
accept/reject before the truncatable filename. Escape dismisses an internal
floating window first, otherwise hides the review and focuses the existing agent
by its absolute pane ID. The existing Couch focus observation gains the review
role instead of another observer (ARCH-PURPOSE). The approved Spec is the scope;
this is one atomic implementation/review boundary.

## Log

### 2026-09-28
- 2026-09-28: closed — Operator smoke passed and close/land authorized. Full review/statusline suites, affected Go race tests, help/drift checks and viewer/routing Lua tests passed. BR-1 addressed: both reordered-pane tests assert body and submit target agent 7, pass with production, and fail with old title selector mutation; source restored. No production code changed after smoke.; review verdict: SHIP
- 2026-09-28: flow upgraded quick → full — 193 added lines in code files (limit 100); an earlier round of this close already ran the full review

- Filed from operator feedback (screenshot of the review bar with no exit hint).
  They found Alt+c by guessing.
- Added: Alt+n / Alt+Shift+N → `]m` / `[m` in the review buffer. Operator's
  choice; the Alt+Shift+N override of the global agent restart is noted in Spec.
  Operator confirmed: while the review pane has focus, Alt+Shift+N means `[m`
  (previous marker). Everywhere else it keeps the agent restart.
- Added: document Alt+a / Alt+r (and the rest of the review keys). The
  operator had forgotten those too.

### 2026-09-28 — implementation

- Added idle/awaiting return and accept/reject hints; normal Esc dismisses a
  diagnostic float first, otherwise returns by agent pane ID. Insert/visual Esc
  remains native. Alt+n/Alt+Shift+N wrap through markers in review normal mode.
- Review help derives from map descriptions with drift coverage; Couch's existing
  focus observation now distinguishes review and preserves Alt+n there. Ctrl+Alt+n
  still relaunches from review, and draft/agent/switcher keep their behavior.
- Fresh-eyes review caught ambiguous non-draft-title agent discovery. A reordered
  right-terminal fixture reproduced it; positive command identity fixed both
  return and poke. No other Important/Critical findings in the ad-hoc review.
- Updated an existing review-toggle fake to delegate retention calls to the real
  binary; otherwise draft initialization failed before the toggle test ran.
- Focused controls tests demonstrated missing hints and focused-popup Escape
  failures before implementation. Full affected Go packages passed with race
  detection; generated help and embedded-source drift tests passed.
- `make test-review` passed all review suites; `make build` plus `make pair`
  rebuilt Couch and Pair with the current embedded sources. Operator smoke pending.

### 2026-09-28 — final acceptance

- Operator confirmed "worked" after the revised hints, draft return, and
  scrollback/changelog overlay fixes, then explicitly requested close and land.
  Automated evidence is recorded above and in the revision validation below.

### Boundary review round 1 — BR-1 addressed

- REWORK identified a regression oracle weakened when return destination became
  draft: reordered panes no longer asserted the separate agent-poke destination.
- Poke fixture now puts terminal 4 before agent 7 and asserts both body and
  submit target 7. Stateful controls host also checks destination and records
  both successful operations. Both tests pass with production code and fail
  with the old non-draft-title selector restored; original source restored.

## Revisions

- 2026-09-28: implementation planning after operator approval; Couch focus
  ownership needs a review exception for Alt+n to reach the local mapping.

- 2026-09-28: routing/help integration exceeds the small-diff code envelope;
  expanded implementation record: [plan](../plans/000340-review-controls-plan.md).
  Operator's implementation authorization covers the existing Spec unchanged;
  close must use the full review if the measured diff remains outside the shell.

- 2026-09-28 — operator smoke revision (supersedes the agent-return wording
  above): hiding review with Esc or Alt+c must focus the draft nvim, including
  the draft-side toggle path. The draft bar must advertise Alt+c review, and
  the review bar must advertise Alt+Return submission in idle and waiting states.
  Existing insert/visual Esc semantics and popup-first dismissal still apply.
  Regression checks now assert draft pane ID 3 rather than agent ID 7; draft
  status tests cover both ordinary and active-review hints.

- 2026-09-28 — additional smoke finding: closing scrollback or changelog reveals
  a still-live review underneath. Extend the shared return-to-draft policy to
  those viewer exits, hiding the floating layer after annotations are emitted.
  Keep the review alive so Alt+c can reopen it; do not discard its buffer.

- Revision validation: new viewer test reproduced all four stacked-overlay
  failures (scrollback/changelog × Esc/:qa) before the shared exit hook; now
  passes and asserts annotations precede draft focus. Revised controls/status
  tests also failed first. Full `make test-review`, `make test-statusline`,
  workbench routing shell/Lua tests, changelog/scrollback Lua tests and keyhelp/
  keyscmd Go tests pass. Fresh-eyes delta review found no Important issues.
  Rebuilt Pair/Couch in pair:0; live smoke of these corrections remains pending.
