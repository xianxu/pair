---
id: 000395
status: working
deps: []
github_issue:
created: 2026-10-06
updated: 2026-10-07
estimate_hours:
card_mirror: 'c872e97f56fcc5fdccf7719c0feb334f401bd802' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-07T14:20:51-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Couch broadcast: stream the composed Couch screen, view-only, to a remote couch --watch

## Problem

There's no way to show a live coding session to a remote person. Screen sharing
apps send pixels of the whole desktop, and tools like tmate/upterm share a whole
terminal with input. The operator wants a broadcast that understands Couch: stream
what Couch shows, view-only, to a remote player, over a `cloudflared` tunnel.

## Spec

Requirements, settled with the operator on 2026-10-06:

- **View-only.** Viewers send no input. No tab switching, no scrolling into the
  sender's history.
- **Source: Couch's Presenter frames**, not pair's per-pane recording. The pair
  recording covers only the agent pane, not the draft or terminal panes or zellij
  chrome. The Presenter owns the one writer to the real terminal and composes the
  child session with Couch's tab bar and switcher (`atlas/terminal.md`), so it is
  exactly what the operator sees.
- **Late joiners start from the current frame.** Every presented frame restates
  cursor, margins, SGR and modes (#262 M2), so a full render of the current
  composed frame is a complete starting point; live frames follow. No replay from
  the start.
- **Sender's size, no reflow.** Full-screen programs (nvim, agent interfaces, Couch
  itself) draw for an exact grid, so generic reflow can't work. The player emulates
  at the sender's cols×rows and shows it letterboxed, scaled or scrolled; resize
  events are part of the stream.
- **Render and forget at both ends.** `couch --watch` keeps only the current screen;
  quitting leaves nothing behind, and the sender's `clear` clears every viewer. The
  broadcaster keeps no history either, only the current frame for late joiners.
  This prevents accidental persistence by viewers acting in good faith. It does not
  prevent deliberate capture: anyone with the link can use another client or a
  screenshot. Access control is what bounds exposure.
- **Control and indicator in the tab bar's leftmost cell.** One reserved cell shows
  ⏺ (start broadcasting). While live it shows `LIVE ⏸` on a red background;
  clicking it stops the broadcast. A keybinding does the same. Fail-safe: if the
  indicator can't be drawn, the broadcast stops; it never streams without the
  indicator visible.
- **Switcher hidden by default, with an option to include it.** The switcher shows
  the whole fleet (every thread's name, path and notes), which is more than the
  session being shared. By default viewers see a placeholder ("operator is
  switching threads") while it's open. An option (a toggle or a start flag)
  includes it, for demos from a clean workspace. The Presenter's layering lets the
  broadcast frame omit the switcher while the operator's own terminal shows it.
  The same layering could later hide individual panes.
- **Transport:** a local server exposed through `cloudflared`, behind Cloudflare
  Access, with a per-session link that expires.
- **Optional end-to-end encryption.** Cloudflare's edge ends TLS and could see
  plaintext. Frames can be encrypted with a key carried in the link's `#fragment`,
  which browsers never send to the server, so the tunnel relays only ciphertext.

Related: #121 (remote control relay, a two-way trust model; this issue is
deliberately one-way), #347 (permanent tty capture identity; not used here, since
render-and-forget persists nothing).

Before building: check how close zellij's built-in web client (0.43+) gets to
this, including its maturity and auth model. Couch's value-add is the composed,
thread-aware screen, the switcher policy and the LIVE control.

## Done when

- The operator starts a broadcast from the tab bar. A remote `couch --watch`
  connected through a `cloudflared` link sees the composed Couch screen live, at the
  sender's size. A viewer joining mid-session gets the current screen immediately.
- `LIVE ⏸` is visible while broadcasting; clicking it stops the broadcast and
  viewers see it end. A test shows that failing to draw the indicator stops the
  stream.
- The switcher is replaced by a placeholder in the broadcast frame by default, and
  included when the option is on. Tests cover both, at the frame level.
- Neither `couch --watch` nor the broadcaster writes frame content to disk, shown by
  a test that runs a session and checks for no persisted frame data. A `clear` on
  the sender clears connected viewers.
- The viewer sends no input; any input from a viewer connection is refused and has
  no effect.

## Plan

- [ ] Spike: zellij web-client comparison, and a Presenter frame tap (in-process
      subscriber, switcher layer omitted)
- [ ] Design the wire format (initial full frame, then frames and resizes) and the
      local server, then the `cloudflared` + Access setup
- [ ] Tab-bar control cell and the fail-safe indicator
- [ ] `couch --watch` player

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
