---
id: 000292
status: working
deps: []
github_issue:
created: 2026-09-19
updated: 2026-09-19
estimate_hours:
started: 2026-09-19T11:10:35-07:00
---

# Carbonyl browser tab in the right pane, shared with the agent over DevTools

## Problem

While developing, the operator and the agent both need to look at a web page
(a local dev server, say `localhost:1111`), and today that means leaving the
terminal. Carbonyl is Chromium rendered into terminal cells, so it can run as an
ordinary child in a right-pane tab. The operator tried it and it works well
enough for a first version. Its Chromium can also be shared with the agent, so
the human and the agent operate the *same* browser during development and
testing.

Verified 2026-09-19:
- **Version:** Carbonyl 0.0.2 (`/opt/homebrew/bin/carbonyl`) bundles Chrome 111.
- **Options:** `--fps` (default 60) and `--zoom`, and it "supports most Chromium
  options".
- **DevTools:** launched with `--remote-debugging-port=0 --user-data-dir=<dir>`,
  it writes `<dir>/DevToolsActivePort`. The DevTools protocol answers there
  (`"Browser": "Google Chrome/111.0.5511.1 (Carbonyl)"`), and `/json/list`
  returns the page's live `url` and `title`.

## Spec

**Opening a tab.** Shift+Alt+B globally, and Alt+B when the right pane has
focus, open a Carbonyl tab in the right pane.
- **Alt+B is already taken in the draft pane:** it opens scrollback at the
  previous prompt (`keyhelp/catalog.go:58`, `nvim/init.lua:3543`). The new
  binding is scoped to the right pane.
- **A shell in the right pane also uses Alt+B** (move back a word), so couch
  intercepts it before the shell.
- **Shift+Alt+B is free in couch** (couchkeys binds Alt+d/h/n/x).
- **Both bindings appear in Alt+h.**

**Each tab runs** `carbonyl --remote-debugging-port=0 --user-data-dir=<tab
profile dir> --fps=<cap> <url>`:
- **A throwaway profile per tab:** it never touches the operator's real Chrome
  profile or cookies.
- **A capped frame rate:** every repaint goes through couch's emulator and gets
  redrawn to the screen. Measure couch's CPU with a page idle and while
  scrolling, and pick the cap from that.

**Name and label.** A browser tab's name is its handle.
- It gets an automatic default (`web`, `web-2`, …) so a handle always exists,
  and the operator can rename it (for example `local-test`).
- The label shows the name and the site: `local-test · localhost:1111`.
- Couch reads the site (url and title) over the DevTools protocol, not from the
  screen.
- Accepted by the operator: this is a development workbench, not a browser for
  everyday use.

**Setting the URL.** If Carbonyl's own interface has a usable URL bar, clicking
the already-active tab focuses it. If not, clicking it opens a floating URL
input, and Enter navigates through the DevTools protocol. Clicking an
already-active tab has no action today, so this defines it for browser tabs.

**Shared with the agent.**
- **A record per tab, keyed `<pair-tag>:<tab-name>`,** holding the DevTools
  websocket address, pid, profile dir and current URL. It's written through
  `cmd/internal/artifactpath` like pair's other tag-bearing files.
- **`pair browser <tag>:<name>`** prints the record as JSON. An agent passes the
  address to Playwright's `connectOverCDP` or a DevTools MCP server, and drives
  the page the operator is watching.
- **The record is a pointer, not the truth.** Readers check the process is
  still alive, the way pair already checks session identity, before trusting
  the address.

**Lifecycle.**
- Closing or parking the tab deletes the record and kills the whole Chromium
  process group, since Chromium spawns many helper processes.
- Couch shutdown, including a crash, does the same. An orphaned Chromium would
  be a much worse repeat of parley#220's `fake_cliproxy` leak (105 orphans, 1.85
  GB).

**Alt+click destination.** With this issue, Alt+click on a URL (pair#293, which
lands first and uses `open` until now) opens a Carbonyl tab. It reuses the most
recently used browser tab that hasn't been renamed; a named tab (like
`local-test`) is never reused automatically.

**Security.** The DevTools port gives full control of that browser, and any
local process can reach it (it only listens on the local machine). That's
acceptable *because* each tab uses a throwaway dev profile that holds nothing
sensitive.

**Limit.** Chrome 111 (early 2023) is fine for a development preview and for
agent-driven checks. It doesn't stand in for testing against current Chrome.

## Done when

- Shift+Alt+B (global) and Alt+B (right pane) open a Carbonyl tab, and Alt+h
  lists both.
- The tab label shows name · site and follows navigation.
- The URL can be set (natively, or through the floating input plus a DevTools
  navigate).
- `pair browser <tag>:<name>` returns a live DevTools address. A test connects
  over the DevTools protocol, navigates, and sees the label change.
- Closing the tab, and killing couch, leave no Carbonyl or Chromium processes
  and no record. A test checks both.
- Alt+click on a URL (pair#293) opens or reuses a Carbonyl tab, and never
  reuses a named one.
- The chosen frame-rate cap and couch's measured CPU are recorded in the Log.

## Plan

- [ ] Spike: does Carbonyl have a usable URL bar, and how does it behave under
  couch's emulator (keys, mouse, CPU at different `--fps`)?
- [ ] Tab launch, the key bindings, and the throwaway profile per tab.
- [ ] DevTools client in couch: the label and navigation.
- [ ] The per-tab record plus `pair browser`, and the lifecycle cleanup.

## Log

### 2026-09-19

Filed from a brain advisor session. Design agreed in conversation. The probe
results above were verified by running Carbonyl with a throwaway profile, with
all processes cleaned up afterwards. Order: pair#293 (Alt+click) first, then this
issue.

### 2026-09-19 — spike (Plan item 1)

Ran under a real pty (Python `pty` + `pyte` screen model, scratch profile,
local `http.server`); every process tree was verified gone afterwards.

- **Idle CPU: Carbonyl 0.0.2 spins a full core.** The installed build
  (`npm` `latest` = `0.0.2-next.bacf3db`) held one process at 99.9% CPU with a
  static page, at `--fps` 60, 30, 15 and 1, with `--disable-gpu`, and on
  `about:blank`. `sample` put 1205/1704 samples in
  `RenderThread::boot → recv_timeout → Timespec::now`. Once idle, the render
  loop's frame deadline is in the past, so `recv_timeout(deadline - now)` is a
  zero timeout and the loop spins. Upstream fixed it in b4ab3a87 "fix(renderer):
  fix idling CPU usage (#126)", shipped in **v0.0.3** (2023-02-18). The npm
  `latest` tag was never moved (it's `next` = `0.0.3-next.ab80a27`). Carbonyl
  **0.0.3 (GitHub release zip) idles at 0.0%**. Measured, 10 s windows, Carbonyl
  tree only:

  | build | page | fps | Carbonyl CPU | output |
  |---|---|---|---|---|
  | 0.0.2 | static | 60/30/15/1 | ~100% | 0 B/s |
  | 0.0.3 | static | 60/30/15 | 0.0% | 0 B/s |
  | 0.0.3 | wheel scroll, 4 Hz | 60/30/15 | 10–14% | ~19 KB/s |
  | 0.0.3 | CSS animation, full motion | 60 | 5.4% | 822 KB/s |
  | 0.0.3 | same | 30 | 5.3% | 421 KB/s |
  | 0.0.3 | same | 15 | 5.1% | 214 KB/s |
  | 0.0.3 | same | 10 | 5.2% | 146 KB/s |

  Output scales linearly with the cap, and every byte is re-parsed by `pair
  term`'s emulator, then zellij, then couch. So the cap bounds the whole chain,
  not Carbonyl. The downstream CPU is still to be measured (M2).
- **URL bar exists but can't be cleared quickly.** Row 0 is Carbonyl's own
  `[❮][❯][↻][ url ]`. A click places a cursor and typed text inserts there;
  backspace-to-empty, then typing and Enter, navigates (verified via
  `/json/list`). Ctrl+U, Ctrl+A/Ctrl+K do nothing, and Alt+Backspace deletes
  one character, in both 0.0.2 and 0.0.3. → pair-owned URL input + DevTools
  navigate (operator chose the tab-strip field, below).
- **Terminal modes:** `?1049h` (alt screen), `?1003h` + `?1006h` (any-event SGR
  mouse), `?25l`. Startup sends two DCS queries, `$qm` (DECRQSS) and
  `+q544e` (XTGETTCAP "TN"). Answering them changes nothing.
- **Process tree:** the npm wrapper is `bash → node (path lookup) → carbonyl`;
  6 processes under 0.0.2's wrapper, 5 with the 0.0.3 binary directly. All share
  one process group (the pty child is a session leader).
- **Crash path:** SIGKILL of the process holding the pty master → the kernel
  SIGHUPs the foreground group → the whole Carbonyl tree was gone within 0.5 s.
  A dead `pair term` therefore takes its browsers with it. Only the files it
  wrote (profile dir, record) survive a crash.
- **DevTools:** `DevToolsActivePort` appears in the profile dir within ~1 s;
  `/json/list` carries url and title. The address is 127.0.0.1-only.
- **Carbonyl writes its profile into its install dir when given no
  `--user-data-dir`** (the npm package's `build/` holds `Local Storage` etc. from
  the operator's first run), so the flag is mandatory.

### 2026-09-19 — design corrections (operator-confirmed)

- **Owner is `pair term`, not couch.** Right-pane tabs are `pair term`'s
  (`cmd/internal/termcmd`). A browser tab is a new tab kind there, so it works in
  standalone pair as well as under couch.
- **A couch crash does not end the browser.** The zellij server daemonizes
  (atlas/couch.md, "A clean alt+d detach and a couch crash leave identical
  external state"), so `pair term` and its tabs outlive couch. The invariant is
  tied to the owner: a Carbonyl process group lives exactly as long as its tab in
  its `pair term`.
- **Alt+click moves to #293** (operator, this session). #292 records the
  "renamed" flag the reuse rule needs. The dependency flips: #293 depends on #292.
- **URL input in the tab strip** (operator, this session), reusing the Alt+r
  rename field, instead of a floating box.
