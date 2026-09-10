---
id: 000221
status: done
deps: []
github_issue:
created: 2026-09-09
updated: 2026-09-10
estimate_hours: 0.81
started: 2026-09-10T10:28:19-07:00
actual_hours: 0.85
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
  through the same call, with no clear-and-replay or child repaint nudge for a
  screen that did not change.
- **The stay arm still says something.** Its original rationale, "so the badge
  clears", does not hold: `RenderStatusRow` never draws a bell on the active
  actor (`reserve.go:124`), so the acknowledgement is invisible from the actor.
  Left silent, the key would look dropped, which is the failure the
  nothing-paging arm adds a notice to avoid. So it gets a notice too. It is
  reachable only through the `focusedAtDelivery` snapshot race: `Deliver` reads
  focus at `console.go:276`, and `onChunk` marks attention later, at `:1181`.

The five outcomes, as the handler's one `switch`:

| focus | `NewestActor()` | live pane | does |
|---|---|---|---|
| panel | — | — | whatever `DecodePanelKeys` makes of the chord's own bytes, fed to `onMenuKey`. That is Return today (`decodeCSIu` drops modifiers for codepoint 13), so behaviour is unchanged and there is no second meaning. It is derived at runtime, not restated as `KeyEnter`, and it bypasses `onMenuInput` because that would consume the panel's held partial without stopping Run's escape timer |
| actor | zero | — | status notice `nothing is paging`; no switch |
| actor | set | none (child done, exit not yet reduced) | status notice `the paging thread is no longer attached`; no switch (`ctrl+backspace`'s refusal, same reason) |
| actor | set | the active one | `switchTo(target, false, arrivalNotification)` acknowledges and stays; status notice `already on the paging thread` |
| actor | set | another | `switchTo(target, false, arrivalNotification)` |

One named constant, `newestPageSequence = "\x1b[13;5u"`, feeds both the
`knownSequences` row and the panel arm. Its doc comment carries the
legacy-encoding note, beside `previousByte`'s.

**Operator docs are a consumer too.** `menuControls` (`menu.go`) is the key
inventory `TestREADMEDocumentsEveryPanelControl` walks so "a new key cannot ship
undocumented"; `Ctrl-Backspace` is already in it though it is an actor chord. So
`Ctrl-Return` joins it. README's couch section documents it, and the sweep takes
every sentence that enumerates couch's chords:
`README.md:378-381` ("`Ctrl-Space` and `Ctrl-Backspace` belong to couch… in both
encodings"), `:384-385` ("every other chord… passes through untouched"), `:390`
("one key plus `Enter`"), `:419` ("couch's third intercepted chord"). README,
not only `keys.go`, says `Ctrl-Return` is recognised only under the Kitty
protocol.

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
  `ctrl+backspace` and this issue does not change it. A page CAN arrive between
  the lookup and the landing. `attention.Mark` runs only in `onChunk`, but
  `onChunk` is reached from `flushDeferredNotifications`, which `switchTo` calls,
  and `switchTo` also runs on the operation goroutine (a queued chip-click
  switch). A page for the target is acknowledged by `switchTo`'s capture at
  landing time; a page for any other thread stays lit for the next press. Both
  are harmless. The event most likely to be mishandled is a plain Return. So
  the negative test covers every codepoint-13 encoding Pair consumes, and each
  must pass through untouched: `\r`, `\x1b[13u`, `\x1b[13;1u` (explicit no
  modifier, `wrap.go:1322`), `\x1b[13;2u` (shift), `\x1b[13;3u` (alt, Pair's own
  chord), `\x1b[13;4u` (alt+shift, `shortcut.go:375`).
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
- Current thread is the newest pager ⇒ attention is acknowledged, no switch,
  and a notice says so (the acknowledgement alone is invisible from the actor).
- Acknowledgement, switch-tracker, and `previous` behave as they do for a
  panel switch — asserted, since the whole point is that this is not a second
  switch path.
- The legacy-encoding limitation is recorded in `keys.go` beside the existing
  `ctrl+backspace` note.
- `atlas/couch.md` lists the chord.
- `Ctrl-Return` is in `menuControls` and README's couch section documents it.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module    design=0.10 impl=0.12
item: smaller-go-module    design=0.10 impl=0.16
item: atlas-docs           design=0.05 impl=0.04
item: milestone-review     design=0.00 impl=0.20
design-buffer: 0.15
total: 0.81
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. The rows are: the chord row plus the handler,
where the design weight is the route decision (direct `switchTo` or the menu
operation); the differential test, edge tests and mutation sweep; the atlas,
README sweep and `menuControls`; one close review. Frequency, per `#201`'s
lesson: this runs once per deliberate gesture, and the saving is a keystroke and
a panel paint, not milliseconds. (`sdlc estimate-source` reports the calibration
doc `[stale]`, #127.)

## Plan

- [x] Decide the two behaviour cases above; record them in `## Spec`.
- [x] `keys.go`: `newestPageSequence` (legacy note in its doc comment) +
      `seqNewestPage` declared BEFORE the `seqHotkey = seqSwitch` alias + `hit()`
      case + `knownSequences` row + `HitNewestPage` in `AllInterceptorHits`.
- [x] `console.go`: `onNewestPageHotkey` (the five-arm table above) +
      `hitHandlers()` entry.
- [x] Interceptor: the existing walkers
      (`TestInterceptorRecognisesEverySequenceAtEverySplit`,
      `TestEveryInterceptedChordHasAHandler`) cover splits and the handler for
      free. The new test is the codepoint-13 neighbour class passing through
      untouched, plus content inside a paste. Seed `FuzzInterceptorFeed` with
      the row and its neighbours.
- [x] `onNewestPageHotkey`: a differential test through the production input
      path, with ids chosen so that newest ≠ first paging ≠ first row. Compare
      `active`, the whole `SwitchTracker`, and every attention projection
      against `ctrl-space` + Return. One test per remaining arm. Proven by the
      mutation sweep, and every mutation is asserted to have applied.
- [x] Docs: `atlas/couch.md` Navigation; `menuControls`; the README sentences
      named in the Spec.
- [x] `make test` green (scrub `PAIR_SESSION_ID`/`PAIR_TAG`); operator smoke on
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
- 2026-09-10: closed — Operator smoke on live Ghostty->couch->pair after make install + couch restart: ctrl+return works. Unsandboxed `env -u PAIR_SESSION_ID -u PAIR_TAG make test` exit 0, 197 ok on 30376a0b. New tests (differential vs ctrl-space+Return comparing active/focus/SwitchTracker/attention, per-arm edge tests, codepoint-13 neighbour class) pass under -race -count=3. Mutation sweep 10/10 killed with apply-asserts (Log table). Screen jump on switch the operator saw is #209 nudge, not this issue (Log).; review verdict: SHIP

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

### 2026-09-10 — implemented (`9868976e`, `8e94de40`)

- **Code.** `keys.go` gets `newestPageSequence`, `seqNewestPage` (above the
  alias), a `hit()` case, a `knownSequences` row and `HitNewestPage`.
  `console.go` gets `onNewestPageHotkey` (the five-arm table) and its
  `hitHandlers()` entry. `menuControls` gains `Ctrl-Return`.
- **Tests.** `TestInterceptorClaimsCtrlReturnAndNoOtherReturn` covers the chord,
  eight Return neighbours (the six Pair consumes, plus the chord's key release
  and ctrl+shift) and a paste. The fuzz corpus is seeded. The existing walkers
  cover splits and the handler. `console_newest_page_test.go` has the
  production-path differential against `ctrl-space` + Return, which compares the
  whole landing (`active`, `focus`, `SwitchTracker`, three attention
  projections). It then chases the second page and returns home with
  `ctrl+backspace`, which is the "pressed again" sentence README states. There
  is one test per remaining arm. All pass under `-race -count=3`.
- **Mutation sweep.** Script `mutate221.py`: every needle is asserted to occur
  once, restores come from saved bytes, and the tree is checked clean after.
  All 10 mutations were killed:

  | mutation | killed by |
  |---|---|
  | `arrivalOrdinary` | the differential |
  | first paging actor in pane order | the differential |
  | `force=true` | the stay test |
  | panel arm dropped | the switcher test |
  | nothing-paging falls back to `ActiveAddress` | the nothing-paging test |
  | stay notice removed | the stay test |
  | not-attached arm removed | the exited-child test |
  | row removed | 4 tests |
  | `hit()` case removed | 4 tests |
  | `seqNewestPage` below the alias | the build (duplicate case) |

  The last one falsified my own comment, which claimed it would "silently open
  the switcher". It was corrected in `8e94de40` to say what the compiler
  actually does.
- **Honestly unpinned.** The panel arm uses `DecodePanelKeys` rather than a
  literal `KeyEnter`. The two produce identical keys for every input today, so
  no behaviour test can tell them apart. The switcher test pins the behaviour
  (Return on the selected row), not the derivation.
- **Suite.** `env -u PAIR_SESSION_ID -u PAIR_TAG make test`, run unsandboxed:
  exit 0, 197 packages `ok`, no failures. Sandboxed runs fail only on the known
  pty tests ("operation not permitted").
- **Doc sweep.** `git grep` for enumerations of couch's chords found the four
  README sites the gate named, and all four are updated. `atlas/couch.md:507` and
  `:589` call `Alt+n` the "third" chord *relative to the `Alt+x`/`Alt+d` grid*,
  which stays true, so they are left alone. #190's "couch intercepts only six" is
  a dated working note and is also left alone.

### 2026-09-10 — operator smoke: working

After `make install` and a couch restart, the operator confirmed `ctrl+return`
works on the live Ghostty → couch → pair stack.

They also saw the whole screen jump up one line on a thread switch, then fall
back. That is not this issue. It is `#209`'s repaint nudge: `takeOverScreen`
→ `RequestRepaint` shrinks the child one row for `RepaintSettle` (20 ms), and
zellij really renders that shorter frame. Every switch path shows it,
`ctrl+return` included, because they all land through `switchTo`. Only the
"nothing is paging" and "already on it" arms take no takeover. It is left for
its own issue.

### 2026-09-10 — close review (SHIP, 4 Minor), all four fixed in the close commit

- **Stale definitions.** `arrivalNotification`, `ExecuteConsoleOperation`'s
  switch arm and `SwitchTracker.Switch` defined a notification hop as
  "ctrl-space + Return". That was a repeat of the plan gate's doc-sweep family,
  so the class got swept, not the two sites named. `git grep` found every
  "notification hop" / "ctrl-space + Return" definition in code, atlas and
  README. Three were exclusive, and those three now define a hop by its
  PROPERTY: a landing on an actor that was paging when the operator chose it.
  The rest compare Return with a click, or are generic, and stay correct.
- **False ARCH-ORDER reason.** The Spec sentence is corrected in place; see
  Revisions. The call graph is `Mark` ← `onChunk` ← {Run loop, `drainChunks`,
  `flushDeferredNotifications` ← `switchTo` (Run or operation goroutine)}.
- **Duplicated `already` check.** `switchTo` now returns `stayed`, decided under
  its own lock, and the handler's separate `c.active` read is gone.
- **Nudge jump untracked.** The operator's disposition was *"slightly weird but
  it's ok"*. So it is not an issue: it is recorded as an accepted trade-off in
  `atlas/architecture.md` beside `RepaintSettle`, with the two unmeasured fix
  directions, and that replaces the "left for its own issue" line above.

## Revisions

### 2026-09-10 — Done-when gains the operator-docs consumer

Reason: `menuControls` exists so that no new couch key ships without README
documentation. `Ctrl-Backspace` is already in it, and this chord is its twin.
Delta: one Done-when bullet (`Ctrl-Return` in `menuControls` + README couch
section) and the matching Plan step. No behaviour changes.

### 2026-09-10 — plan-quality round 1 advisories folded in (PQ-1..PQ-4)

Reason: round 1 passed with no blocking findings, but raised four Minor ones,
and all four were right.
Delta:
- **PQ-1.** The stay arm gets a notice (`already on the paging thread`). Its
  "badge clears" rationale is withdrawn, because the row never draws the active
  actor's bell. There is a matching Done-when edit.
- **PQ-2.** The panel arm derives its keys from `DecodePanelKeys` of the chord's
  own bytes through a shared `newestPageSequence` constant, instead of
  restating `KeyEnter`.
- **PQ-3.** The README sweep enumerates `:378-381`, `:384-385`, `:390` and
  `:419`, plus a Kitty-only note.
- **PQ-4.** The test plan is compressed to function plus strategy. The negative
  class is now every codepoint-13 encoding Pair consumes (it adds `13;1u` and
  `13;4u`), and the fuzz corpus is seeded.

The ARCH-ORDER "page arriving between lookup and landing" sentence was wrong
and is replaced with the reason it cannot happen: `Mark` runs only on the Run
goroutine.

### 2026-09-10 — the ARCH-ORDER correction was itself wrong (close review)

Reason: the plan gate's round 1 said no page can arrive between the lookup and
the landing, because `Mark` runs only on the Run goroutine. I folded that in
without checking it. It is false: `switchTo` → `flushDeferredNotifications` →
`onChunk` → `Mark` also runs on the operation goroutine. The ORIGINAL sentence,
that a page for the target is acknowledged at landing, was right, just
incomplete.
Delta: the Spec's ARCH-ORDER bullet is restated with the full call graph. There
is no behaviour change, since both interleavings were already harmless.
