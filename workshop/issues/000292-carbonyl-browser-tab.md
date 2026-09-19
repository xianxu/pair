---
id: 000292
status: open
deps: []
github_issue:
created: 2026-09-19
updated: 2026-09-19
estimate_hours:
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
all processes cleaned up afterwards.
