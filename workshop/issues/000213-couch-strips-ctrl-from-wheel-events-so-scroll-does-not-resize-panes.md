---
id: 000213
status: working
deps: []
github_issue:
created: 2026-09-07
updated: 2026-09-09
estimate_hours: 0.71
started: 2026-09-09T07:49:18-07:00
---

# couch strips ctrl from wheel events so scroll does not resize panes

## Problem

Holding ctrl while scrolling resizes a zellij pane. The operator wants scrolling
to scroll — the resize is an accidental gesture, easy to trigger and jarring
when it fires.

### Where it comes from — verified, not assumed

**Ghostty is not doing it and cannot be made to stop.** It has no ctrl+scroll
binding (font zoom is `super+=` / `super+-`), so it is not consuming the
gesture — it encodes the ctrl modifier bit into the SGR report and forwards,
which is correct terminal behaviour for an app that requested mouse reporting.
And wheel events are **not bindable triggers at all**, so there is nothing to
bind to `ignore`. Six spellings were tested and every one is rejected:
`ctrl+scroll_up`, `ctrl+wheel_up`, `ctrl+scroll-up`, `ctrl+mouse_scroll_up`,
`ctrl+wheel`, `ctrl+scroll`. This is not a naming problem — Ghostty's own docs
define the syntax as *"Trigger: `+`-separated list of **keys and modifiers**"*,
i.e. the keybind system is keyboard-only by design.

The only Ghostty lever that does exist is `mouse-reporting = false` (also
reachable at runtime via the `toggle_mouse_reporting` action), which stops **all**
mouse forwarding — selection, click-to-focus, copy-on-select, scroll. That is the
same trade `#123` already rejected on the zellij side when `mouse_mode false` was
tried and reverted, and it is far too broad for one modifier bit.

So there is no gesture to suppress and no way to add a suppressor.

**zellij does it, and 0.44.3 has no option to turn it off.** Newer zellij carries
`mouse_scroll_resize`, which when false passes such scroll events through to the
pane. It is **not in 0.44.3** — it does not appear in `zellij setup --dump-config`,
and the apparent acceptance of it in a config file proves nothing: a control test
with a deliberately bogus key (`definitely_not_a_real_option false`) validated
**identically** to a real one, so **zellij 0.44.3 silently ignores unknown config
keys**. (Worth its own issue: every key in `zellij/config.kdl` is therefore
unverified, and a typo or renamed option would fail silently forever.)

**couch is the only interception point.** The filter has to be upstream of
zellij, which consumes ctrl+wheel before it reaches any pane process:

- `pair resume` spawns `zellij attach` as a child and waits — **not** in the byte
  path.
- `pair wrap` is a pty proxy but sits **inside** the pane, downstream of zellij's
  decision. Too late.
- couch owns the host tty and its Interceptor sees every byte first.

## Spec

**Strip the ctrl modifier bit from wheel reports and forward the wheel through**
— do not swallow. Swallowing makes ctrl+scroll do nothing; stripping makes it
scroll, which is what the operator wants when their hand happens to be on ctrl.

Where it goes: `RouteMouseReport` (`couchtty/mouse.go`) is already the single
decision point for every mouse event and is a **pure function** — no IO, unit
testable directly, which is exactly where this belongs (`ARCH-PURE`). The report
is already decoded by `mouseinput.Parse` into `Event{Button, X, Y, Release}` with
`Button` as the raw SGR value.

Encoding facts the implementation needs: wheel is `64` (up) / `65` (down);
modifier bits are shift `4`, alt `8`, **ctrl `16`**. So ctrl+wheel-up is `80`,
ctrl+wheel-down `81`, and ctrl+shift+wheel-up `84`.

**Narrow the change deliberately:**

- Apply **only to wheel buttons** (`64`/`65`). Ctrl+click on other buttons may
  mean something to a child (nvim, a TUI), and stealing a modifier from every
  button is a much larger behavioural change than this issue is asking for.
- **Clear only the ctrl bit**, preserving shift and alt, so ctrl+shift+wheel
  still arrives as shift+wheel rather than as a bare wheel.

**Record that this is deletable.** Once zellij is upgraded to a version carrying
`mouse_scroll_resize`, the config option supersedes this filter and it should be
removed. Put that in the code comment with the option's name, or it calcifies
into a workaround nobody can date or justify.

### Known limitation, by construction

**This fixes couch only; standalone pair is unaffected.** For standalone pair to
filter, `pair` would have to run zellij under a pty it owns and intercept — which
is precisely couch's console, meaning two stacked proxy layers whenever running
under couch, for one modifier bit. Disproportionate. Accepted, not deferred.

In practice this covers approximately all current usage: the operator runs 10–11
couch sessions and standalone is now the exception.

**The real fix is the zellij upgrade**, which covers both surfaces with a config
line and no code. It is not folded in here because pair is calibrated to 0.44.3's
exact mouse and frame behaviour — `#123`'s tiled pivot exists because *"zellij
0.44.3 lets any floating pane be dragged off position by its frame with no config
gate"*, and `#172`/`#196`'s mouse-mode work is tuned to this version. That upgrade
deserves its own risk assessment rather than riding in as a fix for this.

## Done when

- Ctrl+scroll in a couch-hosted session **scrolls** rather than resizing a pane.
- A unit test on `RouteMouseReport` (or the strip helper) covers: ctrl+wheel-up
  and ctrl+wheel-down arrive as plain wheel; **ctrl+shift+wheel keeps shift**;
  plain wheel is unchanged; **ctrl+click on non-wheel buttons is unchanged**.
- No regression in the existing mouse routing cases — couch's row click, the
  switcher-owns-screen branch, and the `childWantsMouse` forward/swallow rule all
  behave as before.
- `#196`'s reattach test passes unmodified: this touches the same mouse seam that
  produced four rounds of defects, and the mode-belief behaviour must not shift.
- The code comment names `mouse_scroll_resize` as the upgrade that retires this.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module    design=0.05 impl=0.12
item: smaller-go-module    design=0.10 impl=0.12
item: atlas-docs           design=0.05 impl=0.04
item: milestone-review     design=0.00 impl=0.20
design-buffer: 0.15
total: 0.71
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. Rows: the `mouseinput` button splice; the
`couchtty` strip policy; atlas; one close-boundary review. The review row takes
the top of its band rather than the middle — this touches the mouse seam that
`#196` needed four rounds to settle, so a cheap review is not the expected case.
(`sdlc estimate-source` reports the calibration doc `[stale]`, ledger newer than
the doc, tracked in #127; the per-primitive hours are provisional.)

## Plan

**Design correction (see `## Revisions` 2026-09-09).** The Spec puts the strip
"in `RouteMouseReport`". That function returns only a `MouseDisposition` — the
forward path writes `hit.Raw`, the exact bytes the terminal sent, and both
`keys.go:110-115` and `console.go:1597-1608` say plainly that re-encoding from
`Event` would be a second source of truth for the wire format. A strip has to
produce *different bytes*, which a disposition cannot express. So the work
splits by ownership rather than living in one function:

- [x] **Format knowledge → `mouseinput`.** Two additions, both wire-format
      facts that belong to the one package that owns the format (`ARCH-DRY`):
      the modifier bits (`ModShift 4`, `ModAlt 8`, `ModCtrl 16`, `ModMask`)
      beside the existing `WheelUp`/`WheelDown`, with `BaseButton(button int)`
      returning `button &^ ModMask`; and `WithButton(raw []byte, button int)
      ([]byte, bool)`, which splices only the button field and leaves
      separators, coordinates and terminator byte-identical. `WithButton` is
      deliberately NOT the re-encoder the package rejects — it cannot drift on
      the parts it does not touch — and returns false on anything `Parse` would
      reject, so a caller can never splice a malformed report into a
      well-formed-looking one.
- [x] **Policy → `couchtty`.** A pure `stripWheelResizeModifier(event
      mouseinput.Event, raw []byte) (mouseinput.Event, []byte)`. **The predicate
      is on the modifier-masked base button**, not on the raw value:
      ctrl+wheel-up is `80`, so a `Button == WheelUp` test would never fire and
      the whole change would be a silent no-op. It matches
      `mouseinput.BaseButton(event.Button)` against `WheelUp`/`WheelDown` and
      clears only `ModCtrl`. Horizontal wheel (`66`/`67`) is excluded on purpose
      — zellij 0.44.3 maps the resize off vertical wheel — and the comment says
      so, so the narrowing reads as a choice. The comment also names
      `mouse_scroll_resize` as the zellij option that retires this function,
      per the Spec's "record that this is deletable".
- [x] **Apply it once, at the top of `onMouse`,** before `RouteMouseReport`, so
      routing and forwarding see one canonical event rather than two. Routing is
      provably unaffected either way — its only button test is `Button == 0`,
      and ctrl+wheel (`80`/`81`) and plain wheel (`64`/`65`) are all non-zero —
      which is why applying it early is safe and is the reason to prefer one
      event over branching.
- [x] **Wiring test — the strip must be proven to be CALLED.** Both new
      functions are pure and unit-testable, but a call whose result is discarded
      or placed in the wrong branch compiles and leaves every unit test green.
      Model on `console_mouse_test.go`'s `TestForwardPreservesRawBytes`: through
      `newMouseFixture`, have the child enable `\x1b[?1000;1006h`, write
      `\x1b[<80;7;9M` to the host pipe, and assert the child receives
      `\x1b[<64;7;9M`. Without this the Done-when rests on the manual step alone.
- [x] **Unit-test strategy, one line per risky function.** For `WithButton` the
      risky class is arbitrary bytes rather than the shapes an enumeration would
      list: a fuzz seeded with malformed and truncated forms, property =
      `Parse(WithButton(raw, b))` equals `Parse(raw)` with only `Button`
      differing and every other byte identical — which also covers absurd button
      arguments a hand-list is blind to. For `stripWheelResizeModifier` the
      risky class is the modifier cross-product: table over
      {plain, shift, alt, ctrl, ctrl+shift} × {wheel-up, wheel-down, horizontal
      wheel, left-press}, asserting ctrl is cleared for exactly the vertical
      wheel rows and every other byte is untouched. Assert on the RAW bytes as
      well as the `Event` — the bytes are what the child receives, and an
      `Event`-only assertion passes with the splice broken.
- [x] **Regression.** Existing `couchtty` routing tests and `#196`'s reattach
      test run unmodified — no edits to either, which is the evidence that the
      mode-belief behaviour did not shift.
- [x] **Atlas.** Note the strip and its deletion trigger on the couch mouse
      surface.
- [x] **Manual.** Ctrl+scroll in a live couch session scrolls and pane sizes
      hold; plain scroll still scrolls.

## Log

### 2026-09-07

Operator report. Filed after ruling out the two layers above couch by test rather
than by reading — and the Ghostty half was re-tested on 2026-09-07 across six
trigger spellings rather than left on a single negative, because one rejected
spelling is indistinguishable from a wrong guess at the name. It is not the name:
the syntax is documented as keys and modifiers only. Recorded at that strength so
nobody re-runs it: Ghostty rejects wheel keybind triggers outright, and zellij
0.44.3's config parser accepts unknown keys silently — the control test with a
bogus key is what showed that `mouse_scroll_resize` is unsupported here rather
than merely unset. Without the control, "config file well defined" would have
read as confirmation and sent this to the wrong layer.

## Revisions

### 2026-09-09 — strip cannot live in `RouteMouseReport`

**Reason.** The Spec names `RouteMouseReport` as the site because it is the
single pure decision point for mouse events. It is — but it decides a
*disposition*, and the forward path writes `hit.Raw` verbatim by explicit
design. Stripping a modifier changes the bytes, so it cannot be expressed as a
disposition. Discovered by reading the forward path, not assumed.

**Delta.** The change splits along the ownership line the codebase already
draws: the byte splice goes in `mouseinput` (the one owner of the wire format),
the wheel/ctrl narrowing goes in `couchtty` (the policy, and the part that gets
deleted on the zellij upgrade). Applied once at the top of `onMouse`. Everything
the Spec fixes — wheel-only, ctrl-bit-only, deletable, named upgrade — is
unchanged; only the location moves.

### 2026-09-09 — plan-quality round 1

**PQ-1 caught a silent no-op before it was written.** The plan said the
predicate was "wheel buttons `64`/`65`". Ctrl+wheel-up IS `80` — the modifier
bits are part of the button value — so that predicate never matches the case the
issue exists to fix, and the change would have compiled, passed a careless test,
and done nothing. The predicate is now on the modifier-masked base button, and
the mask itself moved into `mouseinput` where the rest of the wire format lives.

**PQ-2:** both new functions are pure, so a discarded call would leave every
unit test green. Added a wiring test through the real `onMouse` path, modelled
on the existing `TestForwardPreservesRawBytes` fixture.

**PQ-3:** replaced the prose enumeration of test cases with a strategy line per
risky function — a fuzz property for the splice, a modifier cross-product table
for the policy.

### 2026-09-09 — implementation

**The plan gate caught the bug this issue was most likely to ship.** The Spec
says "apply only to wheel buttons (`64`/`65`)", and written literally that is a
predicate that never fires: modifier bits live *in* the button field, so
ctrl+wheel-up is `80`. `Button == WheelUp` matches nothing, and the change would
have compiled, passed a careless test, and left the resize exactly as it was.
The predicate reads `mouseinput.BaseButton(event.Button)` instead. Reproduced
after the fact by mutation — reverting to the raw-button predicate gives
`button 80 -> 80, want 64`.

**The strip could not live where the Spec put it.** `RouteMouseReport` returns a
`MouseDisposition`; the forward path writes `hit.Raw`, the terminal's own bytes,
because re-encoding from `Event` would be a second source of truth for the wire
format (`keys.go`, `console.go` both say so). Changing bytes is not something a
disposition can express, so the work split along the ownership line the codebase
already draws: `mouseinput.WithButton` splices the button field and leaves every
other byte alone (format knowledge, in the package that owns the format), and
`couchtty.stripWheelResizeModifier` decides wheel-only/ctrl-only (policy, and
the part deleted on the zellij upgrade). Applied once at the top of `onMouse`,
which is safe because routing's only button test is `Button == 0` and every
wheel code is non-zero.

**Both halves are pure, so the wiring needed its own test.** A discarded result
or a call in the wrong branch compiles and leaves every unit test green.
`TestForwardStripsCtrlFromWheelReports` drives real bytes through `newMouseFixture`'s
input loop and asserts the child receives `\x1b[<64;7;9M` and never
`\x1b[<80;7;9M`. Deleting the call from `onMouse` reddens it.

**Evidence, and the limit of it.** `make test` green unsandboxed (EXIT=0, zero
FAIL lines). `FuzzWithButtonChangesOnlyTheButton`: 7.7M execs, property =
`Parse(WithButton(raw,b))` equals `Parse(raw)` with only `Button` differing and
every later byte identical. The strip is table-tested across the full modifier
cross-product {plain, shift, alt, ctrl, ctrl+shift} × {wheel-up, wheel-down,
horizontal wheel, left-press}, asserting raw bytes as well as the decoded event,
plus a release row.

What automation covers is that the bytes leaving couch are the plain-wheel
report. What it does not cover is zellij's response to those bytes — but that
needs no new evidence: a plain wheel report is exactly what an unmodified scroll
already sends, and unmodified scrolling demonstrably scrolls in these sessions
today. The gesture itself still wants one operator check after a couch restart,
since a running couch is on the old binary.

**Standalone pair remains unaffected**, as the Spec accepted: `pair wrap` sits
inside the pane, downstream of zellij's decision, and putting a second proxy
layer under standalone pair for one modifier bit is disproportionate. The zellij
upgrade retires both this filter and that gap.
