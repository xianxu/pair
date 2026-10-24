# Couch broadcast

`cmd/internal/broadcast` streams the composed Couch screen, view-only, to remote
browser viewers (#395). Status: built (M1–M5): frame tap, hub, server, viewer page, session, Couch
control, viewer theme and font, and `cloudflared` tunnels.


Couch's status groups can expose **⏎** for feedback supported by the existing
draft submission action. Chrome uses its Board's current open notes and finished
labelled sketches, excluding exactly delivered versions; LIVE terminal notes do
not participate. See [cast submission](webcast-submission.md). The status button
and Shift+Ctrl+Alt+Return share one declared workbench action, available from any
Pair pane without moving focus.

## Source: the Presenter tap

`terminal.Presenter.SetTap` installs a `terminal.Tap`. It is called on the
Presenter goroutine after a frame is fully written to the parent **and**
admitted as presented, with an owned clone and a `terminal.FrameClass`.
`Select` makes the class public; `Panel(ctx, f, class)` takes it. Couch's
switcher (`couchtty.Console.showMenu`) is the one private panel: it lists the
whole fleet. Taps see exactly what the operator saw. They don't see parent bytes,
which are diffs against the operator's previous frame and carry parent-mode
controls.

## Hub

`broadcast.Hub` turns tapped frames into the broadcast. One goroutine owns the
`Stream` (the last frame sent), the subscribers and the grace timer, so a join
and a frame are totally ordered.

- **Withholding:** a frame is forwarded only if `IndicatorShown` holds. That
  means the first compositor-declared chrome row starts with `LiveLabel`
  drawn in `LiveSGR` (red background). Missing/invalid chrome metadata or a
  clipped label fails closed; child pixels cannot grant broadcast authority.
  The status row draws the same constants. Invariant: viewers get only frames the operator saw marked LIVE.
- **Fail-safe:** a tagged watch, `off | shown | hidden`, is driven by the
  indicator on the most recent frame. After `Activate()`, an indicator that
  stays hidden for `Grace` (1s) ends the hub with `ErrIndicatorHidden`.
  `Activate` comes after the tap is installed, never while the tunnel is
  opening. It arms the timer only if the last frame lacked the indicator, so a
  quiet screen already showing LIVE keeps broadcasting.
- **Privacy:** `ViewerFrame` replaces a private frame with a placeholder that
  keeps every declared tab-bar row and its row metadata, unless `ShowSwitcher`
  is set.
- **Shared diffs:** each frame is rendered once with `terminal.Render`, the
  same renderer the xterm oracle checks, and the diff is shared. Late joiners
  get `Join()`, a full render of the current frame.
- **Slow viewers:** a viewer whose 8-deep queue overflows is marked for resync.
  It gets `Join()` as soon as it has room (on a frame or the 100ms tick), and
  never a diff against a frame it missed.
- **Backpressure:** `Offer` never blocks the Presenter; the latest frame wins.

The hub keeps no history and writes nothing to disk; the only frame it retains
is the current one, for late joiners.

## Server and viewer page

`broadcast.Server` is read-only by construction:
- **GET only:** every other method gets 405, and no request body is read.
- **Token gate:** everything lives under `/<token>/`, and the token is
  compared in constant time. Only the page's fixed assets (`assets` in
  `server.go`) and `events` are served; any other path, or a wrong token,
  gets 404.
- **Code lock (#495):** see [Code lock](#code-lock-495) below. Before a
  session the Host serves the lock page instead of any of the above.
- **Headers on every response:** `Cache-Control: no-store`,
  `Referrer-Policy: no-referrer` (the token is in the path), `nosniff`, and a
  CSP that confines the page to its own origin. Scripts are strict. Inline
  styles are allowed because xterm.js's DOM renderer inserts `<style>`
  elements.
- **`events`:** a Server-Sent Events stream.
  - `event: frame` carries `{cols, rows, chrome_rows, b: base64}`. The trusted
    reservation accompanies initial, diff, metadata-only and late-join frames;
    client point/note bodies cannot override it.
  - `event: end` carries `{reason}`.
  - A `: ping` every 15s keeps Cloudflare from closing an idle stream (it does
    at 100s). A failed ping ends that viewer, which frees its slot.

## Code lock (#495)

Every cast link is locked behind a short numeric code, delivered out of band:
the operator says it on the call.

- **Core.** `castlock.Lock` holds one link's code and its wrong-try budget
  (8 per code, counted across every client), plus the sessions it issued.
  - **Spent budget:** the code rotates and the existing sessions are kept.
  - **Operator rotation** (`Session.RotateCode`): revokes every session and
    closes the generation's `Revoked` channel.
  - **Ordering:** one mutex orders unlocks and rotations.
- **Gate.** `Mountable.LockFor(segment)` names a link's lock, and the
  `Host` gate (`lockgate.go`) runs before any mount sees a request.
  - **Without a session:** `GET /<token>/` gets the lock page, plus
    `lock.js`, `lock.css` and `POST unlock`. Everything else gets `401 locked`.
  - **With a session:** the request reaches the mount under a context that
    is cancelled on revoke, so live streams end with the session.
  - **`GET session`:** the gate's own probe, `204` for a live session and
    `401` otherwise. A page whose stream closed uses it to return to the
    lock page.
- **Session cookie.** It is `HttpOnly`, `SameSite=Strict`, scoped to
  `/<token>/` and has no expiry. It is `Secure` on https links.
  - The page's own fetches send it with same-origin credentials.
  - **Trade-off:** a viewer who re-opens the link from a chat app
    (cross-site) is asked for the code again.
- **Unlock attempts:** each link shares a burst of 5 then 1/s across clients, with 8 wrong codes per forced rotation. The third forced rotation (24 wrong codes) locks out every new unlock, including the correct code, until a successful explicit operator rotation. Successful unlocks do not reset this lifetime count; already-unlocked sessions survive forced rotations and the cap. Failed rotation randomness leaves the lock refusing. Couch notices name the cast/link on each forced rotation; the third combines “code changed” with “link locked until you rotate,” without exposing the code.
- **Terminal broadcast.** Each link has its own 4-digit code:
  - `view`: the lock exists before `Mount`;
  - `pointer`: the lock is minted with the pointer token.
  - Both locks end in `teardown`, after the hub's `end` event, so viewers
    still see why the broadcast ended.
- **Lock page** (`web/lock.*`): asks for the code and the viewer's name; the
  name is stored as `couch.viewerName`.
  - `URL#NNNN` pre-fills the code, and the page strips it from history at
    once.
- **Couch.** Every link's codes reach the operator through the presenter's
  operator-only layer (`terminal.OperatorLayer`), which is applied after the
  tap takes its frame, so no broadcast frame carries a code.
  `console_castcodes.go` draws three things there:
  - **Hover tooltip:** hovering a cast's cell (`LIVE ⏸` or `Chrome ⏸`, the
    main entrance) shows all of its codes, for example
    `view 4821 · pointer 7730`. The link controls beside it show none.
  - **Codes menu:** Alt+click on the cell (only there) opens it.
    - `codes_menu.go` is a pure model: layout, a hit target per word,
      selection and drawing.
    - It has one row per link (`view 3926 · rotate · copy-link`), then
      `rotate all`. Each action word is a click target drawn in the link blue
      (`notesMenuLink`).
    - Keyboard: Up/Down pick the row, Left/Right the word, Enter runs it, and
      Escape closes the menu.
    - Running a word keeps the menu open: the word is drawn pressed for
      `codesPress` (180 ms), and a rotation rebuilds the menu in place with the
      new code, keeping the selection (`refreshCodesMenu`). Only Escape or ×
      closes it.
    - A rotation disconnects that link's viewers.
    - The screen pointer link's row has no actions until the link is minted.
      Right-clicking 👆 calls Session.EnsurePointerLink to mint a locked link
      while staying off; copy-link never turns pointing on.
  - **The cast row's rule** (operator, at the #495 smoke test): a left click
    acts or toggles, a right-click copies that control's link, and Alt+click
    opens the Codes menu. The README has the per-control table. `LIVE ⏸`'s
    left click still stops the broadcast; pause is #518.
  - **Flash:** a new code flashes three times.
  - **Notices:** the status row and the notice line are broadcast, so a
    notice says that a code changed and never what it is.
    - Every current guess rotation raises a notice; the third includes the operator-rotation recovery action.
    - Rotations apply in `Gen` order.
- **Registry.** `broadcast.LinkLocks` is one per cast: a lock per link token,
  minted with the token (`Mint`), routed by `LockFor`, listed by `Codes`, and
  rotated and ended together. A new link is one `Mint`.

### Threat model

- **What the code protects against:** a cast link that leaked.
  - A link pasted in a chat log, kept in browser history or a proxy log, or
    fetched by a chat app's link preview is still only half the credential.
  - The code travels out of band: said on the call.
  - A `#code` fragment is never sent to a server, and preview fetchers don't
    run the page's script.
  - Guessing is bounded by the per-code budget, per-lock rate and per-link lifetime cap.
  - Rotating a code revokes every session that used it.
- **What it does not protect against:**
  - a viewer who has the code: they can share what they see, or pass on
    both link and code;
  - a stolen session cookie, which stays good until rotation or the cast's
    end (it is `HttpOnly` and scoped to the link's path);
  - anyone who can read the operator's screen, including the operator's own
    local diagnostic capture.
- **LIVE viewers see the tab bar**, but never the codes: these draw only in
  the operator-only layer.
- **Replay share links** are not locked yet (pair#509).

The page (`web/`), a vendored `@xterm/xterm` 6.0.0 with `@xterm/addon-unicode11`
(Unicode 11 widths, so emoji are two columns as in Couch; #412),
(`web/vendor/xterm/VENDOR.md`, the same version as the headless oracle) are
embedded in the binary. The viewer:
- keeps the sender's grid and scales the font to fit (`nextFontSize`,
  node-tested via `TestViewerNode`);
- sends nothing back and uses no storage on a view link (a pointer link posts
  points and remembers one value, the viewer's color pick, #446);
- on `end`, resets the screen and shows the reason, and never retries;
- never leaves a dead connection looking live. `connect(deps)` is a state
  machine, node-tested against a fake EventSource. A lost connection dims the
  screen and says so; one closed for good is retried at 2, 4, 8, 15 and 30s,
  and then the screen shows "Disconnected".

`TestManualViewerServer` (`BROADCAST_MANUAL=1`) serves a sample broadcast on
loopback for checking the page in a real browser.

## Session and tunnels

`broadcast.Session` (`Start(ctx, Config)`) owns one broadcast:
- **Start:** mints a 32-byte base64url token, serves the Server on the
  tunnel's listener, opens the tunnel, and returns only once the link answers
  200.
- **Stop:** ends viewers at once. The listener (`Shutdown`, 2s) and the tunnel
  close in the background, and `Done` closes when they have.
- **Ends on its own** when the hub ends (indicator hidden), the tunnel exits
  (`ErrTunnelExited`), or its HTTP server stops serving (`ErrServerFailed`).
  The hub publishes its end reason before closing any viewer queue, so every
  viewer is told the real reason.
- **Cancelling a start** closes whatever was opened, including a tunnel that
  finishes opening after the cancel.

A `Tunnel` owns both its `Listen` and its `Open`, because the listener's kind
depends on what exposes it. There are three:
- `LocalOnly`: loopback TCP, a link that works on this machine only.
- `FakeTunnel`: stateful, for tests in other packages too.
- `Cloudflared`: coming in M4.

`TestSessionPersistsNoFrameData` walks home, temp, XDG and the working
directory after a session and finds no frame content.

## Couch control

`couchtty/console_broadcast.go` owns the console's side as a tagged phase:
`off | starting | live | stopping`. The transition table is in the file's
header comment; all transitions happen on the Run loop.

- **Start:** Ctrl+Alt+b (`couchkeys.ActionBroadcast`). It is enhanced-encoding
  only, because the legacy `ESC ^B` is also nvim's Esc-then-page-up.
  `broadcast.Start` runs off the loop (`GoTracked`). A start that completes after
  it was cancelled or superseded is stopped, not adopted (the `attempt` counter).
  Who owns a finished start's session, the loop adopting it or the start
  goroutine stopping it, is decided by one CAS (`startClaim`). The start
  context is released as soon as the start ends; a tunnel must outlive it.
- **Going live:** the tap goes in before the chrome shows `LIVE ⏸`, so the
  first LIVE frame streams. `Activate` comes after, so the grace watch starts
  with the indicator on screen. The link goes to the clipboard (OSC 52) and is
  never drawn; the notice only says "link copied".
- **Status row:** `RenderStatusBar` draws `LIVE …` (starting) or `LIVE ⏸`
  (live) in `broadcast.LiveSGR` before the `REC` badge, through the same
  clipping pass. Its span is `RenderedStatusRow.Control`. A click anywhere in
  it (`routeMouseEvent`, checked before the actor chips) toggles.
  `TestStatusRowLiveCellSatisfiesIndicator` feeds the drawn row to
  `broadcast.IndicatorShown`, so the drawer and the checker can't drift.
- **Stop:** removes the tap (ordered with paints) and stops the session, which
  tells viewers at once. The phase is `stopping` until the listener and
  tunnel are down, off the input path. A toggle during `stopping` gets a
  notice.
- **Ends on its own** (indicator hidden past grace, tunnel exit, server
  failure): back to `off`, with a notice drawn from the same vocabulary
  viewers see (`broadcast.EndReason`).
- **Shutdown:** `teardown` calls `endBroadcastForShutdown` before the presenter
  is released, and waits up to 6s. Background waiters also end on console
  stop, so a tunnel whose close hangs cannot hold Couch's exit beyond that
  bound.
- **Options** (`couchcmd/broadcast.go`): `COUCH_BROADCAST_SWITCHER=show` and
  `COUCH_BROADCAST_TUNNEL=off`. An unknown value refuses startup, and both
  are parsed before anything opens. Without a configured broadcaster the cell
  never shows, and the key explains why.

## Theme and font

Viewers see the operator's colours and font.

- **Palette:** Couch's startup `paletteQuery` asks for OSC 10/11 (default
  fg/bg, also used by idle fading) and OSC 4 for colours 0–15. ultraviolet
  returns OSC 4 replies as `UnknownOscEvent`; `parseOSC4Reply` reads them, and
  the decoder classifies them as replies, never child input.
  `Console.broadcastTheme` builds a `broadcast.Theme` (`#rrggbb` strings,
  empty for anything unreported). The session's server sends it as
  `event: theme` to each viewer before its first frame. The viewer
  (`xtermTheme`) accepts only `#rrggbb` values and applies them as xterm.js's
  theme and the page background.
- **Font:** JetBrains Mono 2.304 (OFL) is vendored in `web/vendor/fonts/` and
  served under `/<token>/fonts/`. That is what the operator's Ghostty draws,
  as its built-in fallback when the configured family isn't installed. The page
  waits up to 3s for **all four faces** before xterm.js measures its cells,
  then falls back to the system monospace stack. A face left to load lazily
  draws in a fallback with a different advance, and xterm.js's in-flow row
  layout then shifts the rest of the row (#395 M5 smoke). Only fonts whose licence allows redistribution
  are vendored.
- **Symbols (#415):** symbols JetBrains Mono lacks (`⏺ ⏸ ✳ ✔`, braille
  spinner frames) come from "Couch Symbols", a subset of Noto Sans Symbols 2
  listed after JetBrains Mono. Without it, iPad Safari draws several of them as
  colour emoji. Its `unicode-range` is exactly `web/vendor/fonts/symbols.txt`,
  and every glyph has JetBrains Mono's cell advance. It is still **preloaded**
  with the other faces (`SYMBOL_SAMPLE`): xterm.js's DOM renderer measures
  each character on first draw and keeps a per-character `letter-spacing`
  fix-up, so a face fetched lazily on first use is measured as its fallback,
  and the real glyph is then squeezed to no width (#415 iPad smoke).
  `symbols.py` derives the list from recorded agent output and builds the
  subset. The tests hold the font, the list and the CSS range in agreement.
  Wide (emoji-presentation) characters are left out on purpose and stay
  emoji, as in the operator's terminal.

## Tunnels (cloudflared)

`broadcast.Cloudflared` is the production `Tunnel`
(`couchcmd/broadcast.go` picks the mode from the environment):

- **Named tunnel** (`COUCH_BROADCAST_TUNNEL=<name>` with
  `COUCH_BROADCAST_HOSTNAME`): `Listen` makes a private 0700 directory under
  `os.TempDir()` with a unix socket `s` in it. There's a TCP fallback when the
  path would exceed `sun_path`. `Open` writes a per-broadcast `config.yml`
  there (hostname to `unix:<socket>`, everything else `http_status:404`) and
  runs `cloudflared tunnel --config … run <name>`. It's ready when stderr
  reports a registered connection, and cloudflared finds the credentials
  itself in `~/.cloudflared`. The link is `https://<hostname>/<token>/`.
- **Quick tunnel** (unset): a TCP loopback origin, because quick tunnels
  refuse sockets in every form (spike, issue Log). It's ready when stderr
  prints the `*.trycloudflare.com` URL.
- **Probe:** `Session.Start` probes the link before returning, resolving through
  `PublicResolve` (1.1.1.1, then 1.0.0.1). Their "not found" stands, and the
  system resolver is used only if neither is reachable. Asking the system
  during a fresh hostname's ~1.5s propagation would cache a negative answer
  for a minute or more. Probe errors drop the `*url.Error` text, which carries
  the token.
- **Guard** (`couch __broadcast-guard`, `RunGuard`): cloudflared runs in its own
  process group under a guard that holds a pipe from Couch. When the pipe
  closes (Couch exits, crashes or is SIGKILLed), the guard stops the group
  (TERM, then KILL after 3s) and removes the private directory. Couch's own
  `Close` only closes the pipe, because signalling the guard would orphan the
  tunnel.
- **Stderr lifetime** (#585): `Open` owns an `os.Pipe` assigned to `cmd.Stderr`,
  closes its parent writer after `Start`, and joins process wait with the
  scanner drain before publishing `Exited`. This preserves a fast child's final
  error even when wait completes before reading. The scanner closes its reader;
  forced `Close` at the existing five-second budget kills the owned groups and
  interrupts that reader before joining, including a retained inherited writer.
  The owned process group can outlive its reaped leader, so group termination
  remains necessary while completing a pending drain.
- **Run records** (`<couch store>/broadcast/*.json`): owner, guard and
  cloudflared PIDs with `procutil.StrictIdentity`. A named tunnel's record has
  a fixed name, `named-<tunnel>.json`, and is its lock (`ErrTunnelBusy`): two
  connectors of one tunnel are replicas, and Cloudflare would split viewers
  between two tokens. Reaping and claiming run under one `flock` on the
  records directory, so two Couches starting together can't both take a
  stale lock. `ReapOrphans` (at Couch startup and before each open) acts only
  on records whose owner is dead; a record without an owner identity is
  cleared when its owner PID is gone, with nothing killed. It kills a process only if its
  identity and command match. It removes a directory only if it sits directly
  in the run directory, carries the `couch-broadcast-` prefix, and is a real
  0700 directory, not a symlink. The quick-tunnel URL pattern requires the
  hyphenated random host, so cloudflared's API host in a failure line never
  passes for a tunnel, and the probe gives up as soon as the tunnel exits.
- **Conformance:** `TestCloudflaredLive` (`BROADCAST_LIVE_CLOUDFLARED=1`,
  `BROADCAST_LIVE_NAMED=<tunnel>@<hostname>`) runs the real binary end to end.
  Rerun it after any cloudflared upgrade. Measured with 2026.7.3: named link
  ready in about 1s, quick in about 6s, both 530 after stop.

## Pointer link (#412)

A broadcast can carry a second capability link that lets a helper point at the
operator's screen. It is the only input a broadcast accepts from the internet,
and it is kept narrow.

- **State:** `PointerState` (`pointer.go`) holds a 256-bit token, minted on
  first `Session.EnablePointer` and kept for the broadcast's life, plus a
  `pointing` switch. `DisablePointer` downgrades the link to view-only; enabling
  again restores the same link. Its mutex is a leaf: callbacks run after it is
  released.
- **Routes:** `/<pointer-token>/…` serves the same page and stream as the view
  link, plus `POST /<pointer-token>/point` (and the note routes below), the only routes that read a body.
  Everything else is GET-only, as in #395: the view link answers a POST with
  405.
- **The POST**, in order:
  1. pointing must be on (403);
  2. `Content-Type: application/json` (415);
  3. an in-flight cap of 4 (429);
  4. a token bucket of 30/s per link (429);
  5. a 2s read deadline for this request only;
  6. a body capped at 4 KB (`MaxBytesReader`, 413);
  7. `ParsePointBatch`: strict JSON, known fields only, 1..64 integer cell
     pairs inside a stated grid of at most 1000×1000, an optional integer
     `color` 0..5 (strings, fractions and anything outside the palette are
     refused), fixed errors that echo nothing (400).

  Accepted and dropped batches both answer 204.
- **Landing:** `acceptPoint` passes a batch to `Config.OnPoints` only if
  pointing is on and the hub's `Current()` frame matches the batch's grid, is
  public, and shows the active pointer marker. Otherwise it is dropped.
  Nothing else consumes points.
- **Visible capability:** the hub's second watch (`ArmPointer`,
  `DisarmPointer`, `OnPointerHidden`) checks `PointerShown`. That means
  `PointerLabel` on `PointerSGR`'s amber background, right after `LIVE ⏸ `.
  Hidden past the grace, the watch turns pointing off (`Config.OnPointerOff`)
  without ending the broadcast.
- **Pointer pages** get `event: caps` (`{"pointer":bool,"color":n,"palette":[…]}`)
  on join and on every flip; view pages never do. Stopping the broadcast ends
  both links.
- **Pointer colors (#446):** six Okabe–Ito-based colors (`markPalette` in
  `marks.go`, the single source; orange, index 0, is the old amber). `caps`
  carries them as `#rrggbb` (`PointerPalette`), so viewer.js hard-codes none.
  Each pointer stream claims the color fewest live streams hold
  (`PointerState.claimColor`, lowest on a tie) and releases it when it ends.
  The page posts that color until its viewer picks a swatch; the pick is kept
  in `localStorage` (`colorStore`, try/catch, so blocked storage only stops the
  remembering) and outranks the assignment. Couch never learns picks, so
  assignment counts streams, not picks.
- **Pointer rail (#449):** the swatches are a column (`#rail`) overlaid on the
  screen's right edge, vertically centered; `fit()` docks it to the drawn
  screen (`railAt`). It sits outside `#stage`, and only its buttons take
  pointer events, so a swatch tap never starts a stroke and a point anywhere
  else reaches the screen. The mode buttons (`#modes`: `#mode-point` 👆 and `#mode-comment` 💭, both shown,
  the current one pressed, as the web cast's; #447, #488) sit below the swatches. There is
  no hint bar (#447 M3): the swatches and the mode buttons explain themselves
  in tooltips (`title`, `MODE_TIPS`), so the screen keeps the whole window;
  touch devices show no tooltips.
- **Server timeouts:** `ReadHeaderTimeout` 10s, `IdleTimeout` 60s, 8 KB
  headers, and a read deadline only in the POST handler. There is no
  server-wide `ReadTimeout`: it would also time out the disconnect watch on
  long-lived event streams.
- **Marks** (`marks.go`): pure. A batch's points are joined by line fill; an
  off-grid point is dropped before any line reaches it. The broadcast's controls
  (`StatusGuardCols`, the width of `LIVE ⏸ 👆` at the left of the first
  declared chrome row) are never marked or tinted: the overlay refuses them whatever put a mark
  there, including wide-cell overlaps. The rest of the tab bar, including its
  second row, can be marked. Current trusted chrome metadata determines note
  admission, pin placement, outlines and halos after resizing; stored note grids
  remain unchanged. At most 1/8 of the grid is marked, oldest first. A cell keeps its batch's color. A mark holds 1.5s at
  its full color (`MarkHold`), then fades fast over 0.5s (`MarkFade`, 10
  steps): with truecolor it blends into the operator's real background,
  otherwise it walks that color's five-step 256-colour ladder. The Presenter's overlay hook draws marks (terminal.md).
- **Tests:** a method × route × token table; rejections that echo nothing; drops
  for a stale grid, a private frame and a hidden marker; rate and in-flight
  limits, including slow bodies; re-entrant callbacks; `caps` on pointer streams
  only; `FuzzParsePointBatch`; the hub property test models the pointer watch.

### Couch side of the pointer

`couchtty/console_pointer.go` owns the pointer phase: `none | on | off`,
under `c.mu`, with transitions on the Run loop; the header comment has the
table.

- **Status row:** while live it reads `LIVE ⏸ 👆`. `👆` is dim when off and
  on `PointerSGR` amber when on (`StatusModel.Pointer`). Its span
  (`RenderedStatusRow.Pointer`) takes a left click (toggle) and a right click
  (re-copy the link). A right click on `LIVE ⏸` re-copies the view-only link.
  The `👽` placeholder for remote control (#407) was removed until that is
  built (#503); it comes back with its own span and a guard one control wider.
  `routeMouseEvent` checks these spans before anything else on the row.
- **Toggling:** the first click calls `Session.EnablePointer` and copies the
  link (the notice never shows it). Later clicks flip pointing for the same
  link. The watch's `OnPointerOff` reaches the loop as `pointerOffByWatch`.
  Turning off clears every mark at once.
- **Points:** `Config.OnPoints` → `onPoints` → `runTerminalCommand` →
  `applyPoints`, which re-checks on the loop that the phase is on, the
  broadcast is live and the switcher isn't open, then calls `Marks.Add`.
  Coordinates go nowhere else, and no byte reaches a child.
- **Drawing:** `markOverlay` is installed with the tap when the broadcast goes
  live and removed when it ends (`detachBroadcastScreen`, which also forgets
  the pointer). It skips private frames. A timer driven by
  `Marks.NextChange` calls `Presenter.Refresh` while marks live, so the fade
  advances with no child output.
- **Locks:** `pointerMarks.mu` is a leaf. It guards only the marks and their
  timer, and is never held across a call into the Presenter, the session or
  the loop. `TestPointerStressNoDeadlock` paints, posts and toggles together
  under `-race`.

## Sticky notes (#447)

Notes are the pointer link's second collaboration layer. A viewer leaves a
plain-text note on a box of the screen (#488): a drag draws the box, and a tap
gets the default box (`collab.js` `gestureBox`, shared with the web cast). Couch
outlines the box and draws `💭` (`NoteGlyph`) inside its top-right corner. The box
is what a submission crops (#455, `NoteCrop`). A note stays where it was placed,
whatever scrolls underneath (no anchoring). Notes live
in memory and are deleted whenever pointing turns off (click or the 👆-hidden
watch) or the broadcast ends.

- **Model** (`notes.go`, pure): `Note{ID, Col, Row, Box, Color, Text}`.
  - `Box` is a `CellBox` `[col,row,w,h]` on the grid, at least the pin's two cells wide.
  - `Col, Row` is the pin. `NoteBoard.Add(n, cols, rows)` places it: `firstPin`: the first spot (top-right inside
    the box, then leftward, then the rows below, never on the controls) that no open pin covers.
  - `NoteBoard` holds the limits: 8 open notes per pointer color, no two pins sharing a cell. A box with no free
    spot is refused.
  - `NoteCrop(n, cols, rows, margin)` is the box plus margin cells, kept on the grid: the seam #455 crops through.
  `ParseNote` is strict like `ParsePointBatch`: 1–1024 bytes of valid UTF-8, no
  control characters except `\n`/`\t`, no format characters (bidi, zero-width);
  the request is `{cols, rows, box, color, text}` (a page never sends the
  pin), and the box must be on the grid with a pin spot off the controls
  (`NoteVisible`, which reuses the marks' `guarded`). `DrawNotes` tints the
  box's edge cells in the author's color (`noteOutline`, through the glow's
  tint merge; text kept, controls never), puts the glyph on two cells, blanks any wide character it cuts in half (keeping
  its style), and widens the row's used columns, since the renderer draws text
  only that far. `FuzzParseNote`; a property test over random wide rows.
- **Chip, glow, flash** (`notehalo.go`, M3): the 💭's cells get its author's
  color as a chip; a round glow around it (rows ±2, columns scaled by the 2:1
  cell aspect, stronger in the inner ring) tints backgrounds only, never the
  controls or another note's chip, the stronger glow winning where two meet.
  It flashes at full color on arrival (`Note.At`, stamped at save with the
  console's clock via `Config.Now`, not sent to pages) and, for every open
  note together, on one shared beat every `NoteFlashEvery` (15s), holding
  `NoteFlashHold` (300ms) then stepping down over `NoteFlashFade` (500ms, 20ms
  steps) to rest, where only the inner ring keeps one very light shade. One beat bounds the repaints at about 26 per beat
  plus 26 per arrival, whatever the number of notes. `NoteStyle.Hover`
  holds a note at full glow. `paletteTint(c, t, bg, blend)` is the one place a
  pointer color is weakened, for marks and notes alike (blend into the real
  background with truecolor, else the color's 256-colour ladder; `blendable`
  is the one predicate for that), and
  `tintCell` tints both halves of a wide character. The console arms one
  timer from `NoteNextChange` and stops it on pointer-off and broadcast end.
- **State** (`pointer_notes.go`): the board lives inside `PointerState`, under
  its one lock, so a flip to off and the clear are one critical section.
  `notesVersion` only increases. `addNote(n, cols, rows, gen)` refuses unless pointing is on
  and hasn't flipped since `gen`, which `Session.acceptNote` reads before its
  unlocked screen check (`landsNow`: same grid, public frame, 👆 shown).
  Nothing called under `p.mu` takes it or waits on another goroutine.
- **Routes:** `POST /<pointer-token>/note` and `/resolve` share `point`'s
  admission (`readPointerBody`: pointing on, JSON, in-flight slot, the route's
  rate budget, then a prompt body), with their own budget (2/s, burst 5) and an
  8 KB body cap. A refusal is one of a closed set of sentences
  (`refuse*` in `server.go`) that the viewer shows as-is. Resolving an unknown
  id answers 204. The view link still reads no body.
- **Stream:** every page, view and pointer alike, gets
  `event: notes` (`{"guard":StatusGuardCols,"notes":[…]}`) on join and after
  each change, so a view-only viewer can open and read notes.
- **Couch side** (`couchtty/console_notes.go`): `Config.OnNotes` delivers
  `(session, notes, version)`; `onNotes` hands it to the Run loop without
  waiting (a 👆 click runs there and reports the emptied board). `applyNotes`
  keeps it only from the live session while pointing is on and only if newer;
  every pointer-phase change resyncs from `Session.Notes()`. One snapshot under
  the leaf lock `pointerNotes.mu` feeds both `screenOverlay` (notes, then marks,
  public frames only) and the status row's `💭N` count after `👆`
  (`StatusModel.Notes`, outside the guarded columns).
- **Notes menu** (`couchtty/notes_menu.go` pure, `console_notes.go`, M4): a left
  click on `💭N` (`RenderedStatusRow.Notes`) toggles a box over the bottom right,
  above the status row (`RenderNotesMenu`: width ≤ 48, each note a chip of its
  author's color via `broadcast.NoteChipColor` plus its text wrapped to ≤ 3
  rows with `…`, "+k more" when they don't fit, a `×` close mark). It is drawn
  last in `screenOverlay` (`menuOverlay`, after the marks), so viewers see it.
  Its layout on the frame now on screen is the only thing the mouse is
  hit-tested against, and it is cleared on any frame that doesn't show the
  menu (a private panel), so a hidden menu takes no clicks. `routeNotesMenu` runs first in `routeMouseEvent`: over the open
  box it takes press, motion, wheel and a press's release, so nothing there
  reaches a child that tracks the mouse. Motion over an item sets
  `NoteStyle.Hover` (row highlight; its 💭 held at full glow), a click copies
  the note's text (OSC 52, "Note copied."), a right click copies and resolves
  it (`Session.ResolveNote`, as a viewer's Resolve), `×` or another `💭N` click closes
  it. It closes when the last note goes, pointing turns off or the broadcast
  ends (`settleMenuLocked`, `forgetNotes`). Esc is not taken: agents use it.
- **Viewer page** (`web/notes.js`): keeps the list from `event: notes`
  (`parseNotes`) and hit-tests a tap against both cells of each visible 💭
  (`noteAt`, using the event's `guard`), so a tap on a 💭 opens it on any page:
  read-only on a view link, with Resolve on a pointer link. `pointerMode`
  routes taps:
  - 💭 → open.
  - Pointing off → nothing.
  - Comment mode → a box gesture. Press draws nothing. The shared `collab.js`
    `noteGesture` latches a drag at 8 CSS px for mouse or 16 for touch on either
    axis; a dashed `.drag-box` follows it, even after moving back below the
    threshold. On release `gestureBox` opens a draft with the dragged cell box,
    or the default box if no drag began. A box with only the controls under it
    (`boxHasPin`) gets no draft.
  - Pointer mode → a stroke. Turning pointing off resets to pointer
  mode and closes the dialog. `noteDialog` is one fixed-size sticky note: the
  draft exists only in its text box until Save (a byte counter mirrors the
  1 KB cap), a refusal shows Couch's sentence, and a viewed note that leaves
  the list closes. Text is only ever set as `textContent`. All requests go
  through `viewer.js`'s single `postJSON` (`POST_ROUTES`: point, note,
  resolve); the page test pins that and bans HTML sinks in both scripts.
  `BROADCAST_MANUAL_POINTER=1` makes `TestManualViewerServer` serve a pointer
  link that draws notes, for checking the page in a real browser.

### Note anchors (#488)

A terminal note retains its box position/size, slot and draw-time text from
Couch's published frame. Its internal `Anchor` has `kind`, `path` and `text`:
`pane pair:3` for the slot's screen, `tab bar › tabs: pair:1, pair:3` for
Couch's own chips, or `status line` for the rest of Couch's last row. Text under
the box is bounded to 1 KiB. Existing frame/PNG capture paths are independent
and unchanged.

`Config.AnchorNote` invokes `couchtty.anchorNote` synchronously at acceptance.
It uses the last public frame (`keepNoteFrame`) and Couch-owned layout through
the pure `AnchorFor`/`TextUnder` functions. Notes cause no native Zellij query,
file read, role/filename/line/section inference or asynchronous enrichment.
The production composition and broadcast tests enforce that boundary.

The operator chose this deterministic capture contract in #647. Native pane
rectangles include other tabs and hidden floating layers, while captured frames
carry no native tab/client provenance. Matching snapshots or matching disk text
cannot supply that provenance. Internal anchors are excluded from browser note
JSON and the terminal notes menu renders the authored note text; removing the
guesses does not remove a displayed breadcrumb. DOM web-cast element anchors
use their separate browser-owned interfaces.

## Listing (`couch --broadcast-list`, pair#413)

`couch --broadcast-list [--json]` prints the running couch's broadcast from any
shell:
- its state (starting, live or stopping);
- when it started;
- its mode (`tunnel` or `local-only`);
- the viewers connected now.

With no broadcast it prints `no broadcast`.

While recordings are shared for replay (pair#477, [webcast.md](webcast.md#replay-471)), a second line names them:
`replays shared until Couch exits: <id>, …`. `--json` adds `replays`. The ids come from `Response.Replays`, which
`broadcast-status` fills from the console's `SharedReplays`. It never carries a share's link.

**It never prints the link or its token** (operator decision: the token is the
broadcast's only credential). `broadcast.Status` and
`couchmessage.BroadcastStatus` have no field that could carry either, and
`TestBroadcastListNeverPrintsTheLink` checks both formats against a real
session's token.

The path:
1. The CLI sends a `broadcast-status` request to the broker socket, found
   through the store, not a slot's identity.
2. `messageService` answers it before any caller check, from
   `Console.BroadcastStatus`.
3. The viewer count is `Hub.Viewers`, read on the hub's loop, so the answer
   comes from memory.

It is the one identity-free broker op. The socket's directory is `0700` and
owned by the uid, and the socket is `0600`, so only the operator can ask.

## Capability controls (#523)

`couchtty/cast_control.go` shares click intent across terminal 👆 and web 👆/👽:
right copies, left enables-and-copies or disables, and a retryable cleanup uses
the same left-click as explicit retry. Web authority comes from Cast snapshots;
Couch owns no duplicate enabled state. Its completion callback checks cast
identity, generation and attempt before copying or showing a result. Failure
notices precede actor chips so the recovery action remains visible under clipping.
Terminal EnsurePointerLink mints token and code under the existing pointer lock,
without changing admission generation or arming the visibility watchdog.


### Own-note editing (#537)

Both cast viewers use `collab.js`'s pure `noteDialogTransition` for draft, edit,
read-only, pending and held states. Clicking one's note opens its current text;
Save changes text and adds an `edited` marker, while Cancel makes no request.
Other authors' notes remain readable and resolvable by annotators. Annotation
OFF and view-only links permit reading only.

Ownership reuses the existing HttpOnly unlock cookie. The Host stamps the
request context; `GET session` exposes `X-Couch-Note-Owner` only for UI equality.
Refreshing keeps ownership, but unlocking again, rotating the code, or using
another browser/link makes earlier notes read-only. Public attribution stays
unchanged. Empty legacy owners are read-only. `note_edit.go` enforces the owner,
current pointer generation and live note identity before changing text.

Terminal resolution records a typed `NoteResolution` before deleting the note.
`Session.Resolutions()` returns the latest 128 events for this cast lifetime;
ending the cast clears them. This is a bounded in-memory record, not a journal
or a new history/reopen UI. Operator resolution records `Operator`.
