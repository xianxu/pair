---
id: 000811
status: working
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '3b4d0353c33f53123fcf9f24ac379348cea4a3e6' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T04:07:36-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "af96eb45", done: "7a3694a8"}
---

# Couch: Ctrl+Return toggles the attention list; Ctrl+Backspace stays previous

## Problem

## Spec

Operator request (10-22, corrected the same day): rebind **Ctrl+Return** under Couch to **open the attention list**,
exactly as clicking the bell icon does (the same panel Ctrl+Alt+a opens, pair#673/#733). Today Ctrl+Return jumps to
the tab that needs attention (answers the newest page, pair#221); the operator doesn't use that, since notifications
come often and jumping into a tab is too heavy. **Ctrl+Backspace is unchanged** (it jumps to the previous tab, which
the operator uses).

- Ctrl+Return opens the attention panel (toggle: pressing it again closes it, like the bell). The jump-to-tab
  behaviour is removed from that chord; jumping stays available from inside the panel.
- Every place Ctrl+Return is recognised today keeps recognising it (encodings, focused panes); check the
  actor/page answering paths that use Ctrl+Return (pair#221, atlas/couch.md) and say what replaces them, if anything.
- README's shortcut tables, the hosted-session note and atlas/couch.md updated; Ctrl+Alt+a keeps working.

## Done when
- A test drives Ctrl+Return and sees the attention panel open (and close on a second press), with no tab switch.
- Ctrl+Backspace still switches to the previous tab (test).
- Docs name the new behaviour.

## Plan

Single-pass quick-flow change: expected well below 100 added production lines;
one issue-close review, no milestone boundary or separate durable plan.

Design: declare Ctrl+Return as another `couchkeys.ActionAttention` binding and
reuse `Console.toggleAttention`, exactly as Ctrl+Alt+a and the bell do
(ARCH-DRY). Keep all three current Kitty encodings (implicit press, explicit
press, repeat) and `ScopeEveryPane`. Remove the superseded newest-page action,
interceptor hit/sequence kind, dispatch arm and `onNewestPageHotkey`; rename
the byte constant to `CtrlReturnSequence` so its name describes the physical
chord. These internal enums are not persisted. Keep the existing switcher
selection, notification arrival and previous-thread rules.

Pure declaration: `cmd/internal/couchkeys/couchkeys.go` remains the shortcut
source, consumed by input routing and generated CLI/Pair help. Integration:
`cmd/internal/couchtty/{keys,console,console_attention}.go` routes to the existing
operator-only overlay; no child IO, inventory query or new background task is
needed to open it (ARCH-PURE). Existing list layout/expiration cost is reused;
no new growth driver, scan, cache, persisted artifact or lifecycle is introduced.

Actions: toggle attention (Ctrl+Return, Ctrl+Alt+a or bell), select item (Enter
or click), close (Escape), navigate switcher (ordinary Enter), previous tab
(Ctrl+Backspace).

| Existing state | Ctrl+Return result | Next action | Possible loss |
|---|---|---|---|
| Actor or switcher focused; list nonempty and closed | Same attention overlay opens; underlying focus, active actor and history stay put | Select an item or close | None |
| Attention overlay open | Overlay closes; items and underlying selection stay put | Toggle again | None |
| Empty or fully expired list | No overlay; existing “Nothing is waiting on you.” notice | Wait for a new item | None; existing expiration applies |

This intentionally removes both old actor/page shortcuts: no direct newest-page
jump or acknowledgement, and no modified-Return acceptance of the selected
switcher row. Ordinary Enter retains its switcher behavior; the attention list's
Enter/click retains its notification jump and Ctrl+Backspace return-home rule.
Ctrl+Space then Enter remains the existing paging route. No replacement chord
is introduced. Release events, ordinary Return, other Return modifiers and
bracketed paste keep their current forwarding behavior. Repeat events toggle,
consistent with retaining the currently recognized encodings.

- [x] Add failing production-input regressions in couchtty: all three
  Ctrl+Return encodings open then close a populated list, from both actor and
  switcher focus, without switching actor, acknowledging a page, consuming an
  item, changing the switch tracker or accepting the selected switcher row.
  Exercise empty-list behavior and Ctrl+Alt+a interoperability. Preserve the
  bell/Enter item-selection tests, including notification arrival and previous
  thread assertions.
- [x] Update the physical-key test in `console_keyboard_test.go` to generate
  Ctrl+Return from negotiated terminal state after each existing child-output
  mode/replay scenario, assert attention toggling, then close the overlay and
  prove ordinary Return still reaches the original child. Keep the split-byte,
  explicit event, release, neighboring modifier and literal-paste regressions
  in `keys_test.go`/`console_keyboard_test.go`, expecting HitAttention.
- [x] Implement the shared-action mapping and remove the obsolete jump path.
  Replace old newest-page behavior tests while retaining shared fixture/helpers
  used by other notification tests. Pin Ctrl+Backspace's legacy and Kitty forms
  through production input after a manual switch and after an attention-item
  jump; verify the overlay toggle leaves the return target intact.
- [x] Update README shortcut tables/hosted-session note, generated help wording
  in couchkeys, `couchtty/menu.go`, and atlas/couch.md's attention/navigation
  sections plus stale source comments. Explain actor and switcher behavior,
  ordinary Enter, unsupported-terminal fallback, Ctrl+Alt+a and previous-tab
  behavior consistently; sweep old newest-page chord claims.
- [ ] Run targeted couchkeys/couchtty keyboard, interceptor, attention,
  notification, previous-thread and switcher regressions, then both complete
  packages (include race coverage of the console regressions). Audit any
  affected-inventory reader inputs before its exact per-file repair; run
  `make test-affected-inventory` before close. Push and record the declared
  affected check via `sdlc local-check run --name affected`, retaining its runner.
- [ ] Request a hub-authorized real-terminal live check before landing: stage
  with `sdlc stage --build` only when authorized; open/close the list from a
  hosted actor and switcher, select an item with Enter, and use Ctrl+Backspace
  to return. Confirm Ctrl+Alt+a/bell parity. Record actual revision/outcome and
  clear the owned stage. Then SDLC whole close/review, PR and merge.

## Log

### 2026-10-22

### 2026-10-23 — claimed; plan prepared for hub

Dispatched by ops:0 (receipt ded5e61a-9733-4b5a-9f16-60aa7224469d).
Claimed #811 and ran start-plan from clean main-slot3 after #728/#832 landed.
Planning catch-up refreshed the sibling ariadne tools; seeded merge-check
workflow regeneration was explicitly deferred by weave and is outside this
issue. No product files changed and no implementation tests run yet.

Inspected the single shortcut declaration, interceptor/handler table,
onNewestPageHotkey's actor and switcher branches, attention toggle/selection,
physical-key tests, previous-thread regressions, README and atlas. The operator's
same-day correction in Spec is authoritative despite the historical filename:
Ctrl+Return changes; Ctrl+Backspace stays previous. Plan reuses the existing
attention action rather than retaining an independently named jump handler
(ARCH-DRY); all input encodings/focus contexts are covered above (ARCH-PURPOSE).
Next: send this committed plan to the hub; await its response before entering
change-code, as this dispatch requested claim and plan only.

### 2026-10-23 — approved implementation; package verification

Hub approval receipt a9d41597-7856-4a83-a994-e40721223a5d is submitted:
shared attention action, actor/switcher toggle, unchanged Enter/previous keys;
stage on pair:0 when ready, with the operator's live key check in the morning.
Entered change-code; SDLC inferred quick flow.

Before production edits the new input regression failed in all six combinations
(three Ctrl+Return encodings × actor/switcher): “Ctrl+Return navigated instead
of opening attention”; the empty-list case got the old “nothing is paging”.
Mapped both attention chords to the same existing action/handler, deleted the
obsolete jump path, and retained shared notification fixture helpers in a
renamed test file. Updated physical-key protocol tests and docs/help/atlas.

The initial implementation exposed the overlay's existing modified-Return
normalization: routeAttentionKey consumed Ctrl+Return as Enter before shortcut
recognition, so the second press selected an item. The production input seam
now recognizes the declared chord before overlay decoding and gives only
HitAttention priority; other keys keep overlay-first routing, including Escape.
Two superseded in-flight targeted builds were interrupted while correcting that
ordering; they are not passing verification or recorded-suite evidence.

Final verification: complete couchkeys PASS 0.073s, complete couchtty PASS
46.430s; targeted race console/physical-key/event/attention/previous regressions
PASS 3.440s. make test-affected-inventory PASS without any reader repair;
git diff --check PASS. No live-terminal or recorded affected-check result yet.
Next: push source, retain one recorded affected-check runner, stage the verified
head on pair:0, then obtain the morning operator key acceptance before landing.

### 2026-10-23 — staged for morning acceptance; recorded check retained

Source committed/pushed at 2f634183e290332fd1029341d49b3a8c6b4dc05e.
SDLC stage --build completed EXIT0 on pair:0: couch, pair, pair-launch-helper,
pair-slug and couch-conformance-bridge all report that exact vcs.revision and
vcs.modified=false. Stage belongs to #811 and remains held for the authorized
morning operator check; do not clear before acceptance. Hub readiness receipt
36d2ac7f-52fa-42ef-8e9a-f3f0c7b45a44 is submitted.

Opened PR556 (https://github.com/xianxu/pair/pull/556), with corrected product
wording and explicit pending acceptance. Corrected the card title through
sdlc issue set-title to match the already-approved Ctrl+Return contract; the
historical filename is unchanged.

One recorded affected check is queued on source 2f634183, lease-1126897576,
run 7bb4f6f8c2948f6ce4bc092d8b5764d3. Retain unified exec session13291;
do not restart/requeue it. Its isolated snapshot is
/Volumes/TestScratch/local-checks/sdlc-local-171488945 and log is
/Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/7bb4f6f8c2948f6ce4bc092d8b5764d3/run.log.
The declared affected plan includes eight reverse-dependent Go packages plus
the shell group; it acquired normal FIFO6 behind ariadne, no bypass.
No recorded result, hosted-CI result, live acceptance or close verdict yet.

Exact next step: read the retained affected runner's completion on its pinned
source; classify any failure before rerun. Obtain the morning operator key
check on the exact staged build using the checklist below. After acceptance,
clear the owned stage, rerun the mandatory inventory before whole close,
record actual evidence, and use SDLC close/review then merge. No continuation
writer/restart while our recorded check is queued/running.

## Operator live check

On a fresh Couch process using the staged revision (running sessions retain the
old binary), with at least one attention item:

1. From a hosted actor, press Ctrl+Return: attention opens and the actor stays.
   Press it again: attention closes and the actor stays.
2. Open the switcher with Ctrl+Space, then Ctrl+Return twice: the same overlay
   opens/closes and the switcher selection stays. Ordinary Enter still selects.
3. Open attention with Ctrl+Alt+a or the bell, close with Ctrl+Return, reopen and
   press ordinary Enter to visit an item. Ctrl+Backspace returns to the previous
   thread; also verify previous-tab after an ordinary manual switch.
4. Record actual revision and result. Worker clears its owned stage after the
   check. Empty-list behavior is already pinned by the automated regression.

## Revisions

### 2026-10-23

Expanded the empty Plan into a single-pass implementation/verification plan
using the already-corrected operator decision. Clarified that Ctrl+Return in
the switcher also toggles attention; ordinary Enter keeps row acceptance.
No change to the existing Spec or Done when contract.
