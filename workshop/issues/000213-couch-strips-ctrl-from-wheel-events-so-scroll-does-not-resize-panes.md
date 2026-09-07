---
id: 000213
status: open
deps: []
github_issue:
created: 2026-09-07
updated: 2026-09-07
estimate_hours:
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
And wheel events are **not bindable triggers**: `keybind = ctrl+scroll_up=ignore`
is rejected with `error.InvalidFormat`. There is no gesture to suppress and no
way to add a suppressor.

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

## Plan

- [ ] Add the strip in `RouteMouseReport`'s path, narrowed to wheel buttons and
      to the ctrl bit only.
- [ ] Unit tests for the four cases in Done-when.
- [ ] Verify the existing routing tests and `#196`'s test are untouched.
- [ ] Manual: ctrl+scroll in a live couch session scrolls, and pane sizes hold.

## Log

### 2026-09-07

Operator report. Filed after ruling out the two layers above couch by test rather
than by reading: Ghostty rejects wheel keybind triggers outright, and zellij
0.44.3's config parser accepts unknown keys silently — the control test with a
bogus key is what showed that `mouse_scroll_resize` is unsupported here rather
than merely unset. Without the control, "config file well defined" would have
read as confirmation and sent this to the wrong layer.
