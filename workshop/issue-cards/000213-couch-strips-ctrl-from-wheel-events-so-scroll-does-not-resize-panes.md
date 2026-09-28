---
id: '000213'
status: done
started: 2026-09-09T07:49:18-07:00
created: 2026-09-07
updated: 2026-09-09
estimate_hours: 0.71
actual_hours: 0.70
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
