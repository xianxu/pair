---
id: 000221
status: working
deps: []
github_issue:
created: 2026-09-09
updated: 2026-09-10
estimate_hours:
started: 2026-09-10T10:28:19-07:00
---

# ctrl+return jumps straight to the newest paging thread

## Problem

Reaching a thread that paged takes two gestures: `ctrl+space` opens the switcher
already focused on whoever paged, then `Return` lands on it. The panel is pure
ceremony in that path — the operator did not want to *browse*, they wanted to
answer the page.

Collapse it: **`ctrl+return` from an actor jumps directly to the thread
`ctrl+space` would have focused**, with no panel in between.

### Both halves already exist

**The selection logic is one call.** `onHotkey` (`console.go:1356-1370`) already
computes exactly the thing this feature needs:

```go
focus := c.attention.NewestActor()
if focus == (couchcore.ThreadAddress{}) {
    // The defined default with nothing paging: the thread being left.
    focus = c.menu.ActiveAddress
}
```

`attention.NewestActor()` **is** "the latest one with a notification". This issue
reuses that call rather than re-deriving the ordering — two answers to "who
paged most recently" would drift (`ARCH-DRY`).

**The chord is one table row.** `ctrl+space` is `\x1b[32;5u` — codepoint 32,
Kitty modifier bitmask 4 encoded as 4+1 (`keys.go:157`). **`ctrl+return` is
`\x1b[13;5u`** by the same construction. `keys.go` is already shaped for this:
one `seqKind`, one `hit()` case, one `knownSequences` row, one entry in
`hitHandlers()`. Its own comment records why that surface is small on purpose —
`intercepts()` was once a second switch over the same enum, and "two switches
over one enum agree until someone edits one".

### The one decision this needs

**What does `ctrl+return` do when nothing is paging?**

`ctrl+space` falls back to `menu.ActiveAddress` — the thread being left — which
is right for a *browser*: it opens somewhere sensible. For a *direct jump* that
same fallback means "jump to where you already are", i.e. a silent no-op that
looks like a dropped keypress.

Recommended: **do nothing, and say so** — a brief notice on the status row
(`nothing paging`) rather than a silent swallow or a pointless switch. The point
of the key is to answer a page; with no page there is nothing to answer.

Same question, second case: the newest paging thread **is** the current one.
Recommended: acknowledge the attention (which `switchTo` already does via
`c.attention.Acknowledge`) and stay put, so the badge clears and the key is not
inert.

Both are behaviour choices, not implementation details, and should be stated in
`## Spec` before the code exists.

### Legacy-encoding caveat, accepted not discovered

Under the Kitty keyboard protocol `ctrl+return` is `\x1b[13;5u` and separates
cleanly. **In legacy encoding it is indistinguishable from plain `Return`**
(both CR, `0x0d`), so it cannot be intercepted there without stealing every
Return from the child — which is unacceptable.

This is the same shape as the documented `ctrl+backspace` trade at
`keys.go:20-30`, and the resolution is the same: zellij pushes the Kitty
protocol (`support_kitty_keyboard_protocol true` in `zellij/config.kdl`), so the
chord works in practice. **Do not add a legacy fallback** — record the
limitation the way `previousByte`'s comment does.

## Spec

**A new intercepted chord, `ctrl+return` (`\x1b[13;5u`), that switches to
`attention.NewestActor()` without opening the panel.**

- Reuse `attention.NewestActor()`; do not re-implement recency ordering.
- Route the switch through the existing `switchTo` path so acknowledgement,
  the switch tracker, and the `previous` target all behave exactly as they do
  for a panel-driven switch. A second switch path is what this issue is trying
  not to create.
- **Nothing paging** ⇒ no switch, plus a status-row notice. Not the
  `ActiveAddress` fallback `ctrl+space` uses — that is a browser's default, not
  a jump's.
- **Newest paging thread is the current one** ⇒ acknowledge and stay.
- From the panel, `ctrl+return` is unclaimed by this issue — the panel already
  has `Return`. Leave it forwarded rather than inventing a second meaning.

Add the chord in the four places `keys.go`'s structure requires and nowhere
else; the file's comment explains why that count is the point.

### Design (2026-09-10)

**The handler is `ctrl+backspace`'s mirror image, not the switcher's Return.**
`onPreviousHotkey` already is "a chord from an actor, a target computed from
console-local state, a direct `switchTo`". `ctrl+return` is the same shape with
a different target: `ctrl+backspace` goes home, `ctrl+return` goes to the page.

```go
newest := c.attention.NewestActor()                // the ctrl-space rule, reused (ARCH-DRY)
target := c.switchTargetForAddressLocked(newest)   // the pane lookup ctrl+backspace uses
c.switchTo(target, false, arrivalNotification)
```

Why not dispatch the menu's `switch` operation, as a status-chip click does?
The click needs the menu because it *is* a switcher gesture on an inventory row.
This key has no row: the answer is a live pane, and the menu path would add the
operation-queue hop, a dependency on the inventory having loaded, and
`ctrl-space`'s fallback of "first visible row" when the paging thread is missing
from a stale inventory -- a browser's default again. That fallback is the one
place the two paths can legitimately disagree, and there the jump is the one
that is right.

Why each argument is what it is:

- **`arrivalNotification`.** The Return path derives this from a non-zero
  attention capture at dispatch; `NewestActor()` names only an address that has
  messages, so the capture would always be non-zero. Passing it directly is the
  same answer without the round trip. It is what makes `ctrl+return` non-pinning:
  answer the page, `ctrl+backspace` goes back to where you were working.
- **`force=false`.** Different target ⇒ identical to `force=true` (`already` is
  false). Same target ⇒ `switchTo` acknowledges, the tracker ignores a landing on
  `current`, and there is no takeover. That is the Spec's "acknowledge and stay"
  with no special case -- and no clear-and-replay plus child repaint nudge for a
  screen that did not change. No row repaint is owed either: `RenderStatusRow`
  never draws a bell on the active actor (`reserve.go:124`), so the acknowledged
  attention is visible only in the switcher, which repaints when opened.

The four outcomes, as the handler's one `switch`:

| focus | `NewestActor()` | live pane | does |
|---|---|---|---|
| panel | — | — | the panel's own Return (`onMenuKey(KeyEnter)`), which is what `decodeCSIu` already made `\x1b[13;5u` -- unchanged behaviour, not a second meaning |
| actor | zero | — | status notice `nothing is paging`; no switch |
| actor | set | none (child done, exit not yet reduced) | status notice `the paging thread is no longer attached`; no switch (`ctrl+backspace`'s refusal, same reason) |
| actor | set | found | `switchTo(target, false, arrivalNotification)` |

**Operator docs are a consumer too.** `menuControls` (`menu.go`) is the key
inventory `TestREADMEDocumentsEveryPanelControl` walks so "a new key cannot ship
undocumented"; `Ctrl-Backspace` is already in it though it is an actor chord. So
`Ctrl-Return` joins it, and README's couch section documents it next to
`Ctrl-Backspace` -- including the line that says following a page is "one key
plus `Enter`", which this issue makes false.

**ARCH notes.**
- `ARCH-DRY`: target from `NewestActor()`, lookup from
  `switchTargetForAddressLocked`, landing through `switchTo`; no new ordering,
  lookup, or landing logic.
- `ARCH-PURE`: the rules are already pure (`AttentionLedger`, `SwitchTracker`);
  the handler is glue -- read, pick one of four arms, call. It matches the other
  hotkey handlers, and pulling out a pure four-arm decision would add a type
  with no second caller.
- `ARCH-CONSTRAINTS`: a keystroke path, run once per deliberate gesture (a few
  dozen a day). Work is `NewestActor()` over ≤3 messages per actor plus one pane
  scan -- microseconds. What it saves is human: one keystroke and a panel paint,
  plus the operation-queue hop. Disk, network, memory: N/A.
- `ARCH-ORDER`: adds no state that lives between events. It runs on the Run
  goroutine like every hotkey. The interleaving that reaches it: a status-chip
  click queues a `switch` on the operation goroutine, then `ctrl+return` lands
  first. The later landing wins `active`. That is already true of
  `ctrl+backspace` and this issue does not change it. A page that arrives
  between the lookup and the landing is acknowledged by `switchTo`'s capture at
  landing time. The event most likely to be mishandled is a plain Return, so a
  negative test pins that `\r`, `\x1b[13u`, `\x1b[13;2u` (shift) and
  `\x1b[13;3u` (alt, Pair's own chord) all pass through untouched.
- `ARCH-SECURE`, `ARCH-MOCK`: N/A. The input is operator keystrokes, framed by
  the interceptor's existing exact-string match. No persisted input, no secrets,
  no external dependency; the tests use the existing `FakeHost`/`FakeChild`
  seams.

## Done when

- `ctrl+return` from an actor pane switches to the newest paging thread with no
  panel frame drawn.
- The landed thread is **identical** to what `ctrl+space` then `Return` would
  have reached — asserted by a test that drives both paths against one attention
  state and compares the result, so the two cannot drift.
- Nothing paging ⇒ no switch, and the operator sees why.
- Current thread is the newest pager ⇒ attention is acknowledged, no switch.
- Acknowledgement, switch-tracker, and `previous` behave as they do for a
  panel switch — asserted, since the whole point is that this is not a second
  switch path.
- The legacy-encoding limitation is recorded in `keys.go` beside the existing
  `ctrl+backspace` note.
- `atlas/couch.md` lists the chord.
- `Ctrl-Return` is in `menuControls` and README's couch section documents it.

## Plan

- [x] Decide the two behaviour cases above; record them in `## Spec`.
- [ ] `keys.go`: `seqNewestPage` + `hit()` case + `knownSequences` row
      (`\x1b[13;5u`) + `HitNewestPage` in `AllInterceptorHits`; legacy-encoding
      note beside `previousByte`.
- [ ] `console.go`: `onNewestPageHotkey` (the four-arm table above) +
      `hitHandlers()` entry.
- [ ] Interceptor tests (`keys_test.go`): recognised with a clean split;
      held across every read cut; content inside a bracketed paste; `\r`,
      `\x1b[13u`, `\x1b[13;2u`, `\x1b[13;3u` forwarded untouched.
- [ ] Differential test through the production input path: three live
      threads, two paging in an order where newest ≠ first-paging ≠ first row.
      Path A `ctrl-space` + `\r`, path B `\x1b[13;5u`, from identical fixtures;
      compare `active`, the whole `SwitchTracker`, and every thread's attention.
      Path B draws no switcher frame.
- [ ] Edge tests: nothing paging (notice, no switch, tracker untouched);
      current is the newest pager (acknowledged, no takeover, tracker
      untouched); paging thread's child done (notice, no switch); panel focus
      (acts as Return on the *selected* row, not the newest pager).
- [ ] Mutation sweep, each mutation asserted to apply: `arrivalOrdinary`,
      target = first paging in pane order, `force=true`, panel arm dropped,
      nothing-paging arm falling back to `ActiveAddress`. Each must turn a test
      red.
- [ ] Docs: `atlas/couch.md` Navigation, `menuControls` + README couch section.
- [ ] `make test` green (scrub `PAIR_SESSION_ID`/`PAIR_TAG`); operator smoke on
      the live Ghostty → couch → pair stack.

## Log

### 2026-09-09

Operator request: *"ctrl-return jumps to pair thread with latest notification,
same logic how to determine where to focus when ctrl-space is pressed."*

Scoped after reading the two seams rather than designing from the request:
`attention.NewestActor()` already is the selection rule, and `keys.go` already
has a four-site shape for adding a chord. So the work is small and the only real
content is the two behaviour decisions plus keeping the landing identical to the
two-gesture path.

### 2026-09-10

Claimed and designed. What reading the code turned up:

- **Two switch routes exist, and `switchTo` is where both end.** The switcher's
  Return goes reducer → `runMenuOperation` (captures attention) → operation
  queue → `ExecuteConsoleOperation` → `switchTo(target, true, how)`. `how` is
  `arrivalNotification` exactly when that capture was non-zero. `ctrl+backspace`
  calls `switchTo` directly. The Spec's "route through `switchTo`" therefore means
  the direct route, and the design says why the queued route would be worse
  here.
- **No row repaint is owed on the stay path.** `RenderStatusRow` draws a bell
  only when `a.Bell && !a.Active`, so the current actor's pending attention
  never shows on the row.
- **The panel already decodes `\x1b[13;5u` as Return.** `decodeCSIu` drops
  modifiers for codepoint 13. So forwarding the chord as `KeyEnter` keeps
  today's panel behaviour byte for byte.
- **A consumer the issue did not list: `menuControls`.** It is the inventory
  `TestREADMEDocumentsEveryPanelControl` walks. Added to Done-when (see
  Revisions).
- Nothing else binds ctrl+return: `git grep` for `13;5u`, `<C-CR>`,
  `ctrl+enter`, `ctrl+return` and friends matched only this issue.

## Revisions

### 2026-09-10 — Done-when gains the operator-docs consumer

Reason: `menuControls` exists so that no new couch key ships without README
documentation. `Ctrl-Backspace` is already in it, and this chord is its twin.
Delta: one Done-when bullet (`Ctrl-Return` in `menuControls` + README couch
section) and the matching Plan step. No behaviour changes.
