---
id: 000293
status: open
deps: [pair#292]
github_issue:
created: 2026-09-19
updated: 2026-09-19
estimate_hours:
---

# Alt+click opens a link in the right pane (browser tab or nvim)

## Problem

Coding agents' links are clickable in a plain terminal but seemed dead under
pair and couch. Diagnosis (2026-09-19), from recording what zellij 0.45.1 sends
to the terminal:

- **zellij passes hyperlinks through intact:** `ESC]8;;https://example.com …`.
- **zellij also turns plain URLs into links itself:** `ESC]8;id=1;https://…`.
- **zellij turns on full mouse reporting:** `?1000h ?1002h ?1003h ?1006h`. From
  then on Ghostty hands clicks to the app instead of opening links, and the
  mouse protocol can't carry Cmd. So Cmd+click reaches zellij as a plain click
  and does nothing.

**Shift+Cmd+click works** in zellij and under couch (confirmed by the operator).
Ghostty keeps every Shift-click for itself (`mouse-shift-capture = false`), so
it opens the link with the system default app. That settles "system default"
with no code.

What's missing is the *pair* destination: opening a link in couch's right pane
instead.

## Spec

**Two gestures:**
- **Shift+Cmd+click → system default app.** Ghostty does this already; the work
  is only to list it in Alt+h.
- **Alt+click → the right pane, handled by couch.** Shift+Alt+click can't work,
  because Ghostty keeps every Shift-click. Alt is a modifier the mouse protocol
  carries, couch already reads it (`ModAlt = 8`,
  `cmd/internal/mouseinput/mouseinput.go:46`), and it matches pair's use of Alt
  everywhere else.

**Finding what was clicked:**
- Take the link from couch's model of zellij's screen at the click position.
  zellij already sends links, including the ones it made from plain URLs.
- Otherwise, detect a URL or file path in the text of that row (`rowtext`).
- Resolve relative paths against the thread's repo root. A path that doesn't
  exist isn't treated as a file.

**Where it goes:**
- **A local URL** → a Carbonyl browser tab in the right pane (pair#292). It
  reuses the most recently used browser tab that hasn't been renamed; a named
  tab (like `local-test`) is never reused automatically. With no reusable tab,
  it opens a new one. (Moved here from pair#292, see Revisions.)
- **A remote URL** → `open`, the patched system browser, by default. Carbonyl
  bundles Chromium 111 with no upstream commit since 2023-02-26, so a click
  that opens an arbitrary site in it is the one path worth avoiding. `open` is
  also what Shift+Cmd+click already does, so the two gestures agree on remote
  links and differ only where the browser tab is the better answer.
  `browsertab.IsLocalURL` (pair#292) is the shared classifier — loopback,
  `localhost`, private and link-local addresses, `file:`, `about:`, `data:`.
  An operator who wants a remote page in the pane can still type or paste it
  into the URL field, which confirms once.
- **A text file** (markdown and the like) → a new right-pane tab running `nvim
  <file>`. `path:42` opens as `nvim +42 path`.
- **Anything else** (images, PDFs, …) → `open`, same as Shift+Cmd+click.

**Out of scope:** standalone pair without couch. There zellij is the outermost
layer, so pair doesn't see the mouse first.

## Done when

- Couch logs the modifiers it receives. Option+click arrives with the Alt bit
  set (`macos-option-as-alt` is unset in the operator's Ghostty config).
- Alt+click on an existing text file (and on `path:line`) opens `nvim` in a new
  right-pane tab. Alt+click on any other file calls `open`.
- Alt+click on a URL opens or reuses a Carbonyl tab (pair#292), and never
  reuses a named one.
- Tests cover the target resolution: a link from the screen model, a URL in the
  row text, a relative path that exists, a relative path that doesn't, and
  `path:line`.
- Alt+h lists Shift+Cmd+click (system default) and Alt+click (right pane).

## Plan

- [ ] Confirm couch receives Option+click with the Alt bit set.
- [ ] Target resolution (a pure function over the frame, the row text and the
  repo root) with tests.
- [ ] Dispatch to a Carbonyl tab, an `nvim` tab or `open`; Alt+h entries.

## Log

### 2026-09-19

Filed from a brain advisor session. The diagnosis in the Problem comes from
recording zellij's terminal output. The operator confirmed Shift+Cmd+click in
zellij and under couch, and agreed to Alt+click for the right pane.
Order set by the operator: enable Alt+click first (this issue), then the Carbonyl
tab (pair#292), which becomes the URL destination.

## Revisions

### 2026-09-19 — Alt+click's browser destination moves here from pair#292

The operator chose, in the pair#292 session, to build the browser tab first and
hand the Alt+click wiring to this issue. The dependency flips: this issue
depends on pair#292, which no longer depends on it.
- **Spec delta:** a URL goes to a browser tab with pair#292's reuse rule
  instead of `open`-until-#292.
- **Done-when delta:** the Alt+click bullet moves over from pair#292's Done-when.
  pair#292 records each browser tab's "renamed" flag, which the reuse rule reads.

### 2026-09-19 — remote links default to the system browser

pair#292's engine risk (unmaintained Carbonyl, Chromium 111) is accepted there
on the condition that a remote URL is never opened silently. That condition
lands here too, because Alt+click is the gesture most likely to hit an
arbitrary site: remote goes to `open`, local goes to the browser tab, and
`IsLocalURL` is shared rather than restated. See also pair#295 (engine
maintenance tiers), which is where a future engine swap would revisit this.
