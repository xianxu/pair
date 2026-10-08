---
id: 000395
status: codecomplete
deps: []
github_issue:
created: 2026-10-06
updated: 2026-10-08
estimate_hours: 7.78
card_mirror: '8b53cca89acd40bdc787574ed40708569c5db4a5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-07T14:20:51-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: N/A
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

## Estimate

Derived per `estimate-logic-v3.1` (`impl=` at 40% of the v2 table). Design
×0.2 for the thorough plan (except the two smoke/UX rounds, which are
operator-bound); +15% design buffer; familiarity 1.2, because the browser and
`cloudflared` parts are novel-but-bounded while the Go side is familiar. Items,
in order: M1 tap, indicator/privacy/stream, hub; M2 viewer, server, session;
M3 status row/key, console wiring, couchcmd, atlas; M4 cloudflared; discovery
(cloudflared, xterm.js under CSP); two smoke rounds; four milestone reviews.

```estimate
model: estimate-logic-v3.1
familiarity: 1.2
item: smaller-go-module      design=0.06 impl=0.14
item: greenfield-go-module   design=0.25 impl=0.22
item: greenfield-go-module   design=0.25 impl=0.32
item: greenfield-go-module   design=0.25 impl=0.22
item: greenfield-go-module   design=0.25 impl=0.22
item: greenfield-go-module   design=0.25 impl=0.22
item: smaller-go-module      design=0.06 impl=0.14
item: tui-screen             design=0.25 impl=0.26
item: smaller-go-module      design=0.06 impl=0.14
item: atlas-docs             design=0.03 impl=0.05
item: api-integration        design=0.40 impl=0.40
item: real-api-discovery     design=0.00 impl=0.18
item: real-api-discovery     design=0.00 impl=0.18
item: ux-rename-iteration    design=0.55 impl=0.08
item: ux-rename-iteration    design=0.55 impl=0.08
item: milestone-review       design=0.00 impl=0.14
item: milestone-review       design=0.00 impl=0.14
item: milestone-review       design=0.00 impl=0.14
item: milestone-review       design=0.00 impl=0.14
design-buffer: 0.15
total: 7.78
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

## Plan

Durable plan: `workshop/plans/000395-couch-broadcast-stream-the-composed-couch-screen-view-only-to-a-remote-couch-watch-plan.md`.

- [x] Spike: zellij web-client comparison; Presenter tap seam located (see Log)
- [x] M1 — Presenter frame tap with privacy class; broadcast indicator, privacy,
      stream and hub (withholding, grace stop, resync)
- [x] M2 — vendored xterm.js viewer page with auto-fit font; GET-only SSE server;
      session lifecycle over a tunnel seam; no-persistence test
- [x] M3 — tab-bar LIVE cell, Ctrl+Alt+b, Console wiring and fail-safe;
      `COUCH_BROADCAST_*` options; local smoke; atlas
- [x] M4 — viewer theme and font: the operator's palette (default fg/bg and
      the 16 ANSI colours, queried from the terminal) applied as xterm.js's
      theme; the packed JetBrains Mono (OFL) served same-origin via
      `@font-face` (the font-file setting was dropped; see Revisions)
- [x] M5 — `cloudflared` quick tunnel with orphan reaping; live smoke; close

## Log


- 2026-10-08: closed — All five milestones closed with fresh-context reviews (M1 SHIP r2, M2 FIX-THEN-SHIP r3, M3 SHIP r2, M4 SHIP r1, M5 SHIP r2). Done-when: operator live smoke on pair:0 via named tunnel live.functeer.com in Safari (composed screen live at sender grid, font fit, late joiner, switcher placeholder, LIVE click stop ends viewers); IndicatorShown gate + grace stop tested (TestHubStopsWhenIndicatorHiddenPastGrace, TestBroadcastStopsWhenIndicatorCannotBeDrawn); switcher placeholder/ShowSwitcher frame-level tests; TestSessionPersistsNoFrameData; clear propagates (TestStreamClearClearsViewer); viewer endpoint GET-only/token-gated/revoked after stop; page same-origin with CSP (operator browser check, no CSP errors). Full verification: make -k test fails only test-pair-embedded-runtime (identical on a main archive) and test-changelog (passes with scratchpad TMPDIR); go test ./... failures (artifactpath inventory, couchcmd TestColdResume*/TestContinuationWriter*, couchcore TestSpawnComposes*, gcruntime TestCouchReferencesLocalArchive*, launcher checkpoint/scoped-data-dir tests) all fail identically on a main archive with the same scrubbed env; inventory names no #395 file. TestCloudflaredLive passes for named and quick tunnels. Actual not recorded: sdlc actual reads 1.79h for the whole issue (less than M1 alone measured at 1.75h) after the pre-move rebase and slot moves; see Log.; review verdict: SHIP
### 2026-10-06

- Operator idea: "couch broadcast", stream the tty verbatim to a couch player over
  `cloudflared`. Measured for context: this session's agent raw capture was 4.8 MB
  over about 3 days (about 1.6 MB/day); the rendered transcript was 69 KB / 739
  lines. Codex output has been measured at about 53 KB/s in bursts. Bandwidth is
  not a concern.

### 2026-10-07
- 2026-10-07: closed M5 — actual not recorded: sdlc actual remains skewed by the earlier rebase and slot moves (see Log). Review round 1 fixes: BR-21 quick-URL pattern requires the hyphenated random host (TestQuickTunnelAPIFailureIsNotAURL with the real failure line) and the probe returns on tunnel exit (TestSessionProbeStopsWhenTunnelExits); BR-22 reap+claim under one flock (TestRunRecordsConcurrentClaimHasOneWinner, 100 races; fails at iteration 62 on an -overlay mutant without the lock); minors: identity-less dead-owner records cleared without kills, private-dir removal guarded by parent/prefix/0700/not-symlink (table incl. symlink escape), record write failure fails Open, no stray temp file. go test -race ./cmd/internal/broadcast ok unsandboxed; couchcmd broadcast tests ok. Earlier evidence stands: TestCloudflaredLive both modes against real cloudflared 2026.7.3; operator live smoke on pair:0 (named tunnel to Safari, kill -9 cleanup, startup reap); xterm.js 6.0.0 and all-faces font fix confirmed by the operator.; review verdict: SHIP
- 2026-10-07: closed M4 — actual not recorded: sdlc actual remains skewed by the pre-move rebase (see Log, M3). Evidence: go test ./cmd/internal/{couchtty,terminal,broadcast} ok unsandboxed; theme: TestHex, theme event precedes the first frame and is absent without a provider, TestParseOSC4Reply (0-15 only, 1-4 hex digits, malformed refused), paletteQuery asks OSC 4 for 0-15, decoder classifies OSC 4 replies as replies at every split (input_test), TestBroadcastSendsOperatorTheme end to end from terminal replies to a viewer; viewer xtermTheme node tests accept only #rrggbb; fonts: JetBrains Mono 2.304 (OFL) vendored with size-verified source and sha256s, served only under the token (TestServerServesFontsUnderToken), unlisted vendor files 404; M3 advisories: Tunnel.Open doc fixed, TestBroadcastLateHandoverAfterAbandonIsNotAdopted fails on an -overlay mutant without the claim check. Operator visual check in Chrome with the Ghostty palette: colours, font and fit look right.; review verdict: SHIP
- 2026-10-07: closed M3 — actual not recorded: rebase onto main (#214) before the slot move plus the move itself left sdlc actual unreliable (0.79h from pair:0 where this session transcripts are absent; 3.34h from pair:3, below the 4.20h measured before M2 close); see Log. Evidence: BR-11 fixed via single startClaim CAS (TestStartClaimDecidesOnce 2000 races); reviewer repro go test -race ./cmd/internal/couchtty -run Broadcast -count=8 clean; BR-12 fixed, TestBroadcastShutdownBoundedByStuckTunnel fails on an -overlay mutant of the unbounded watcher; TestBroadcastOffHasNoClickTarget added; console broadcast tests cover key start, click stop (first/middle/last column, not past), switcher placeholder vs ShowSwitcher, clipped indicator ends broadcast, toggle while starting closes late tunnel, stop off input path, shutdown stops broadcast, unconfigured notice; drawer feeds IndicatorShown; affected packages ok unsandboxed after rebase; couchcmd TestColdResume*/TestContinuationWriter* also fail on a main archive. Operator local smoke on pair:0 (0dbe0802): works end to end.; review verdict: SHIP
- 2026-10-07: closed M2 — go test -race ./cmd/internal/broadcast ok (server token gate/GET-only/headers/same-origin lint; SSE frames, ping, end, 503 cap, 410 after end, slot freed on disconnect; session lifecycle incl. late open closed, tunnel exit, server failure, hidden indicator, no persisted frame data). BR-5: end reason published before queues close, regression test fails on -overlay mutant; reviewer repro -count=300 -cpu=1,2,8 clean. BR-6: viewer connect() state machine node-tested (dim, Reconnecting, backoff then Disconnected, no retry after end). BR-9: end reasons are a closed vocabulary; TestEndReasonIsAClosedVocabulary covers wrapped errors with paths. Operator real-browser check in Chrome: renders, refits, no CSP errors.; review verdict: FIX-THEN-SHIP
- 2026-10-07: closed M1 — go test -race ./cmd/internal/broadcast ok (indicator/privacy/stream/hub; 500-seed interleaving property test with Activate at random steps, mutation-checked via -overlay against dropped resync, missing withholding, leaked private frames, and the BR-1 always-arm Activate; real-ticker resync test); BR-1 fixed as tagged off|shown|hidden watch with regression tests; go test ./cmd/internal/terminal ./cmd/internal/couchtty unsandboxed (clean PAIR_*/ZELLIJ* env, short TMPDIR) ok; artifact inventory names no broadcast/tap file (remaining failures pre-existing); review verdict: SHIP

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
- M3 local smoke (operator, pair:0 at vcs.revision 0dbe0802, local-only
  link): it works end to end. Ctrl+Alt+b starts, the browser follows the
  composed screen, and stopping ends it. The viewer's colours and font differ
  from the terminal's (xterm.js defaults); this became M4.
- The branch was rebased onto main (#214) before the slot move, so the live
  Couch kept #214's fixes. Side effects: one commit subject starting `#395`
  was eaten as a comment by `rebase --continue`'s message cleanup and was
  repaired; M1/M2 `Review-Window` trailers name pre-rebase commit IDs; and
  `sdlc actual` became unreliable (0.79h from pair:0, where this session's
  transcripts aren't; 3.34h from pair:3, below M2's 4.20h). So M3 closes
  with `--no-actual`.
- M3 review round 1 was REWORK. Fixes:
  - BR-11: a start's session is adopted or abandoned through one atomic claim.
    The start goroutine used to read a flag that the loop could still write
    after the goroutine stopped waiting.
  - BR-12: background waits also end on console stop, so a stuck tunnel close
    can't hang Couch's exit past its 6s bound.
  The start context is released once the start ends, and the Tunnel contract
  now says a tunnel must outlive it.
- M4 visual check (operator, Chrome, `TestManualViewerServer` with the
  operator's Ghostty palette from `ghostty +show-config`, Apple System
  Colors, background `#1e1e1e`): it works. The page background and ANSI colours
  follow the theme event, the text is the packed JetBrains Mono (bold, italic,
  bold italic), and it refits on resize. In a real Couch the palette comes from
  the terminal's OSC 10/11/4 replies (`TestBroadcastSendsOperatorTheme`).
- M5 spike, `cloudflared` 2026.7.3, live:
  - **Quick tunnels can't forward to a unix socket.** `--unix-socket` alone
    is refused ("pass --url"); `--url unix:/path` is parsed as host `unix`
    (502); `--url` plus `--unix-socket` ignores the socket. A TCP origin
    works.
  - **Quick-tunnel hostnames resolve slowly on this Mac.** A new
    `*.trycloudflare.com` failed to resolve through the system resolver for
    more than 60s (curl exit 6). Through 1.1.1.1 it resolved in 1s and served on
    the first try (`curl --resolve`).
  - **A named tunnel works with a unix socket.** Operator setup: `tunnel
    login`, `tunnel create couch-broadcast`, `tunnel route dns couch-broadcast
    live.functeer.com`. Then `cloudflared tunnel --config <tmp> run
    couch-broadcast`, with ingress `hostname: live.functeer.com → service:
    unix:<private dir>/s` and a catch-all `http_status:404`, connected in 2s.
    `https://live.functeer.com/x` served through the system DNS on the first
    request, another Host got the catch-all, and `cloudflared` exited 4s after
    SIGTERM.
- M5 live smoke (operator, pair:0 at 498f7773, named tunnel
  `couch-broadcast` → `live.functeer.com`): Ctrl+Alt+b, then the link
  `https://live.functeer.com/<token>/` in Safari follows the screen live.
  - One transient glitch: a few rows were briefly offset by about 2 columns,
    with stale cells at their ends, and healed on the next update. This fits a
    frame painted half-way, since xterm.js 5.5.0 ignores DECSET 2026, and
    xterm.js 6.0.0 (2025-12-22) implements it.
  - Process chain while live: Couch, then the guard (`couch __broadcast-guard`,
    in its own process group), then cloudflared (in its own group).
  - Crash test: `kill -9` of Couch stopped the URL. Afterwards there was no
    guard or cloudflared process and the private directory was gone. The
    restarted Couch's startup sweep removed the dead owner's run record.
- M5 smoke, round 2 (xterm.js 6.0.0): the shifted rows persisted, and the
  operator pinned them to Claude's gray italic recap lines. The cause was in
  the viewer. It waited only for the regular and bold faces, so italic runs
  drew in a fallback face whose advance differs, and xterm.js's in-flow row
  layout pushed the rest of the row left (the pane border, nvim's line
  numbers). The operator also saw the view scroll as lines were added. After
  the viewer waited for all four faces (ba425fc8), the operator confirmed it
  fixed.

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
- M1 built (tap, indicator/privacy/stream, hub):
  - `⏸` and `…` are one column in both `textwidth.Width` and
    `ansi.StringWidth` (TestIndicatorGlyphWidths).
  - The hub's interleaving property test first passed against a hub with
    resync removed: its `check` drained every viewer each step, so no queue
    ever overflowed. Fixed (check only inspects; bursts of offers). Then
    mutation-checked with `go test -overlay` against dropped resync, missing
    withholding and leaked private frames; all three fail it.
  - `TestProductionArtifactReferencesAreExactlyClassified` still fails on
    pre-existing couchcmd/couchmessage files; no broadcast or tap file is
    named.
- M2 real-browser check (operator, Chrome, `TestManualViewerServer` on
  loopback): the frame renders (colours, `界面`, box drawing, `LIVE ⏸` on red,
  `⏸` one cell); the font refits on window resize with no reflow; the
  DevTools console shows no CSP errors under the shipped policy (`style-src
  'self' 'unsafe-inline'`, strict `script-src`). The only console line is the
  browser's own `/favicon.ico` request, which gets 404: it carries no token.
  xterm.js 5.5.0 ignores DECSET 2026; frames go out whole, one `term.write`
  each.
- **2026-10-07** — after the M3 local smoke, the operator asked for the viewer
  to match the terminal's colours and font. That adds M4 (theme passthrough
  and a served font file, for example "DejaVuSansM Nerd Font Mono"), and the
  tunnel moves to M5. A terminal can't report its font, so the file is named by
  `COUCH_BROADCAST_FONT_FILE`. DejaVu and the Nerd Fonts patches allow
  redistribution.
