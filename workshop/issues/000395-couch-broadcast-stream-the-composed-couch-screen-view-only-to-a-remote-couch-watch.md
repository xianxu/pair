---
id: 000395
status: working
deps: []
github_issue:
created: 2026-10-06
updated: 2026-10-07
estimate_hours:
card_mirror: 'd86ccb80b61184856c222e958057840daa283226' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-07T14:20:51-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Couch broadcast: stream the composed Couch screen, view-only, to a browser viewer

## Problem

There's no way to show a live coding session to a remote person. Screen sharing
apps send pixels of the whole desktop, and tools like tmate/upterm share a whole
terminal with input. The operator wants a broadcast that understands Couch: stream
what Couch shows, view-only, to a remote player, over a `cloudflared` tunnel.

## Spec

Requirements settled with the operator on 2026-10-06 and revised on 2026-10-07
(see Revisions):

- **View-only, by construction.** The viewer endpoint is an HTTP GET whose
  response is a Server-Sent Events stream. It has no inbound channel, so a viewer
  cannot send input, switch tabs or scroll into the sender's history. Remote
  control, if it comes, is a different URL with a different grant (#407); a
  view-only link can never be upgraded to it.
- **Source: Couch's Presenter frames**, not pair's per-pane recording. The pair
  recording covers only the agent pane, not the draft or terminal panes or zellij
  chrome. The Presenter owns the one writer to the real terminal and composes the
  child session with Couch's tab bar and switcher (`atlas/terminal.md`), so it is
  exactly what the operator sees. Tap point: after a successful paint in
  `Presenter.paintPublication`, handing subscribers a `Frame.Clone()`. The parent
  bytes can't be tapped, because they are diffs against the operator's previous
  frame and carry parent-mode controls.
- **Hub and transport adapters.** A transport-agnostic hub receives tapped
  frames, substitutes private frames, keeps only the current frame, and renders
  one shared diff stream: each new subscriber gets a full render of the current
  frame, then the shared diffs, in order. SSE is a thin adapter over the hub. A
  later control WebSocket (#407) reuses the hub and adds only the input direction.
- **Late joiners start from the current frame.** Every presented frame restates
  cursor, margins, SGR and modes (#262 M2), so a full render of the current
  composed frame is a complete starting point. No replay from the start.
- **Sender's grid, viewer's font size.** Full-screen programs (nvim, agent
  interfaces, Couch itself) draw for an exact grid, so generic reflow can't
  work. The sender's cols×rows is preserved exactly; the browser viewer scales
  the font to fit, at font size = min(viewport width ÷ cols, viewport height ÷
  rows), recomputed on every resize event in the stream.
- **Browser viewer.** A small page served by the local server renders the stream
  with a vendored, pinned `@xterm/xterm`, embedded in the binary with `go:embed`.
  It loads nothing from a CDN, because a page showing the operator's screen
  shouldn't load third-party scripts and should work offline. The wire bytes are
  our own `Render` output, which `tests/terminal-oracle` already checks against
  pinned `@xterm/headless`, the same parser. Check whether xterm.js honours
  synchronized output (DECSET 2026). The hub sends only finished frames either
  way.
- **Render and forget at both ends.** The viewer keeps only xterm.js's in-page
  screen. Closing the tab leaves nothing behind, and the sender's `clear` clears
  every viewer. The broadcaster keeps no history either, only the current frame
  for late joiners. This prevents accidental persistence by viewers acting in
  good faith. It does not prevent deliberate capture: anyone with the link can
  record the stream or take a screenshot. Access control is what bounds exposure.
- **Control and indicator in the tab bar's leftmost cell.** When stopped, the
  tab bar shows nothing extra. Ctrl+Alt+b starts broadcasting and copies the
  `cloudflared` link to the clipboard, for sharing through another channel.
  While live it shows
  `LIVE ⏸` on a red background with the normal foreground, ahead of the
  `REC` capture badge. Clicking anywhere in the red portion stops the broadcast.
  A second Ctrl+Alt+b also stops it. Invariant: a frame reaches viewers only
  if the same frame on the operator's screen showed `LIVE ⏸`. Fail-safe: if the
  indicator can't be drawn, the broadcast stops; it never streams without the
  indicator visible.
- **Switcher hidden by default, with an option to include it.** The switcher shows
  the whole fleet (every thread's name, path and notes), which is more than the
  session being shared. By default viewers see a placeholder ("operator is
  switching threads") while it's open. An option (a toggle or a start flag)
  includes it, for demos from a clean workspace. The switcher is not a layer:
  `Console.showMenu` presents a whole `PanelFrame`. So the call site marks its
  frame private, and the hub substitutes the placeholder (unless the option is on)
  while the operator's own terminal shows the switcher. The same mark could later
  hide individual panes.
- **Transport and access: a capability link.** A local HTTP server, bound to
  127.0.0.1, is exposed through a `cloudflared` quick tunnel (trycloudflare.com,
  no Cloudflare account setup). Access is a random per-broadcast token in the
  link that expires when the broadcast stops, or after a timeout. Requests
  without a valid token are refused. Quick tunnels can't sit behind Cloudflare
  Access; a named tunnel with Access is a possible later layer.
- **End-to-end encryption is a follow-up (#406).** Cloudflare's edge ends TLS and
  sees plaintext frames in this issue's version.

Related: #121 (remote control relay), #407 (a web front end as a remote-controlled
Couch), #406 (end-to-end encryption), #347 (permanent tty capture identity; not
used here, since render-and-forget persists nothing).

Zellij's built-in web client was checked and can't do this (see Log
2026-10-07): it draws only zellij's UI, so Couch's tab bar, switcher and LIVE
cell are outside its reach.

## Done when

- The operator starts a broadcast from the tab bar. A remote browser opening the
  `cloudflared` link sees the composed Couch screen live, at the sender's grid,
  with the font scaled to fit the window. A viewer joining mid-session gets the
  current screen immediately.
- `LIVE ⏸` is visible while broadcasting; clicking it (or the keybinding) stops
  the broadcast, and viewers see it end. A test shows that failing to draw the
  indicator stops the stream.
- The switcher is replaced by a placeholder in the broadcast frame by default, and
  included when the option is on. Tests cover both, at the frame level.
- Neither the viewer page nor the broadcaster writes frame content to disk, shown
  by a test that runs a session and checks for no persisted frame data. A
  `clear` on the sender clears connected viewers.
- The viewer endpoint is read-only: it serves only GET, a request without a
  valid token is refused, and a token stops working once the broadcast ends.
- The viewer page and xterm.js are served from the binary (vendored, pinned);
  the page makes no request to any other origin.

## Plan

Durable plan: `workshop/plans/000395-couch-broadcast-stream-the-composed-couch-screen-view-only-to-a-remote-couch-watch-plan.md`.

- [x] Spike: zellij web-client comparison; Presenter tap seam located (see Log)
- [ ] M1 — Presenter frame tap with privacy class; broadcast indicator, privacy,
      stream and hub (withholding, grace stop, resync)
- [ ] M2 — vendored xterm.js viewer page with auto-fit font; GET-only SSE server;
      session lifecycle over a tunnel seam; no-persistence test
- [ ] M3 — tab-bar LIVE cell, Ctrl+Alt+b, Console wiring and fail-safe;
      `COUCH_BROADCAST_*` options; local smoke; atlas
- [ ] M4 — `cloudflared` quick tunnel with orphan reaping; live smoke; close

## Log

### 2026-10-06

- Operator idea: "couch broadcast", stream the tty verbatim to a couch player over
  `cloudflared`. Measured for context: this session's agent raw capture was 4.8 MB
  over about 3 days (about 1.6 MB/day); the rendered transcript was 69 KB / 739
  lines. Codex output has been measured at about 53 KB/s in bursts. Bandwidth is
  not a concern.

### 2026-10-07

- Claimed; `start-plan`. Spike findings (code map + web research):
  - **zellij web client (0.43+, read-only tokens since 0.44, latest 0.45.1
    2026-08-28) can't do this.** It attaches as a zellij client and draws only
    zellij's UI, so Couch's tab bar, switcher and LIVE cell are outside its
    reach. ~48 open web-client issues (reconnect loops, freezes). Couch's value
    is the composed screen; broadcasting Couch's own frames is the only route.
  - **Tap seam:** `terminal/presenter.go` `paintPublication` (~:371) after a
    successful paint, next to `p.previous = f.Clone()`. Every paint converges
    there (endpoint, `Panel`, `UpdateChrome`, `Resize`). Hand subscribers
    `f.Clone()` (Frame holds slices; only a clone is goroutine-safe). Don't tap
    bytes: they are diffs against the operator's `previous` and carry
    parent-mode/kitty-keyboard controls. Precedent for a bounded non-blocking
    tap: `terminalcapture.Recorder` (`ErrQueueFull`).
  - **Switcher isn't a layer.** `Console.showMenu` builds a whole `PanelFrame`
    and calls `presenter.Panel`, so the broadcast needs an explicit
    "private frame" mark at that call site, not layer omission (spec's
    "layering" wording is wrong for today's code).
  - **Tab bar:** `couchtty/reserve.go` `RenderStatusRow`; capture badge is
    leftmost (no click target). Clicks: `couchtty/terminal_input.go`
    `routeMouseEvent` (last row → `ColumnToActor`). Keys: `couchkeys` table +
    `dispatchFor` + `hitHandlers`.
  - Geometry rides on every `Frame.Geometry`; resizes come free.
  - No HTTP server or network deps today; CLI is hand-parsed
    (`couchcmd/cli.go`).
- Design discussion with the operator, on four decisions:
  - Capability link on a quick tunnel instead of Cloudflare Access.
  - SSE for the view-only stream. Remote control would be a separate WebSocket
    URL with its own grant, safer than one bidirectional socket that drops
    viewer input.
  - A browser viewer (vendored `@xterm/xterm`, auto-fit font) instead of
    `couch --watch`. The browser is easier for viewers and starts a possible web
    front end: "a remote-controlled Couch" (#407).
  - End-to-end encryption deferred to #406.

## Revisions

- **2026-10-07** — after the spike and the operator discussion. Delta:
  - Viewer: `couch --watch` → browser page (vendored `@xterm/xterm`, font scaled
    to fit the sender's grid).
  - Transport: unspecified → SSE GET (no inbound channel); control later on a
    separate WebSocket URL (#407).
  - Access: Cloudflare Access + expiring link → capability token on a quick
    tunnel.
  - End-to-end encryption: optional here → #406.
  - Switcher hiding: Presenter layering → a private-frame mark at the
    `showMenu` call site, substituted by the hub, because the switcher is a whole
    frame, not a layer.
  - Done when: rewritten to match; the read-only criterion moves from "refuse
    viewer input" to "GET-only, tokens checked and expiring", and a
    "no other origin" criterion is added.
- **2026-10-07** — the durable plan settles points the Spec left open:
  - The link lives exactly as long as the broadcast; the "or after a timeout"
    clause is dropped, because an expiry would cut a live demo, and stopping
    already revokes it.
  - Fail-safe: a frame reaches viewers only if the operator's painted frame
    showed `LIVE ⏸`; if it stays hidden for 1s (resize interim, a narrow
    terminal), the broadcast stops.
  - Options: `COUCH_BROADCAST_SWITCHER=show` and `COUCH_BROADCAST_TUNNEL=off`
    (local-only). Key: Ctrl+Alt+b.
- **2026-10-07** — operator UI decisions: the idle glyph is ▶ (was ⏺); `LIVE ⏸`
  uses a red background with the normal foreground and leads `REC`; any click
  in the red portion stops; Ctrl+Alt+b toggles. The tunnel connects to a unix
  socket in a private directory, so an orphaned tunnel can't expose an
  unrelated program that later reuses a TCP port.
- **2026-10-07** — no idle glyph (▶ was ambiguous-width and odd to show all
  the time): stopped draws nothing; Ctrl+Alt+b starts; a click on the red
  `LIVE ⏸` or Ctrl+Alt+b again stops.
