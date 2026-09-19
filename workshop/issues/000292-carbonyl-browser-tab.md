---
id: 000292
status: working
deps: []
github_issue:
created: 2026-09-19
updated: 2026-09-19
estimate_hours: 9.46
started: 2026-09-19T11:10:35-07:00
flow: {kind: full, provenance: inferred}
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

The durable plan is `workshop/plans/000292-carbonyl-browser-tab-plan.md`. This
Spec is the contract; the plan holds the design detail.

**Owner.** A browser tab is a new kind of `pair term` tab
(`cmd/internal/termcmd`), beside the shell tabs. So it works in standalone pair
as well as under couch, and it lives exactly as long as its tab in its `pair
term`.

**Opening a tab.** Alt+B in the right pane, and Shift+Alt+B from any pane,
open a URL field in the tab strip (`[url: │]`). Enter launches Carbonyl on
that URL as a new tab; Esc cancels and creates nothing.
- **Alt+B is taken in the draft** (scrollback at the previous prompt,
  `keyhelp/catalog.go:58`) and in the scrollback viewer (Alt+b/B: prompt
  navigation, buffer-local, `nvim/scrollback.lua:485-487`). Both keep their
  meaning: Alt+B is a right-pane role chord, and the viewer's buffer-local
  Shift+Alt+B outranks the global there.
- **A shell in the right pane loses Alt+B** (backward word): `pair term` takes
  the chord before the shell, as it already does Alt+t/w/r.
- **Shift+Alt+B moves focus to the right pane**, unlike Shift+Alt+T, because
  the URL field needs the keyboard.
- **A browser tab owns `pair term`'s tab keys.** Alt+t/w/r/b/←/→ are not
  passed through to Carbonyl as they are to vim or htop (#227): Chromium has no
  use for them, and Alt+w and Alt+r are how you close and rename the tab.
- **Both bindings appear in Alt+h.**

**Each tab runs** `carbonyl --remote-debugging-port=0 --user-data-dir=<profile>
--fps=15 <url>`:
- **The binary** is `$PAIR_CARBONYL` or `carbonyl` on PATH. If it's missing,
  the strip shows a notice and no tab opens.
- **Carbonyl older than 0.0.3 gets a strip notice.** 0.0.2 spins a CPU core
  when idle (measured, see Log); 0.0.3 idles at 0%. The tab still opens.
- **A throwaway profile per tab,** outside Pair's data root (under the user
  cache dir), so storage GC never walks a Chromium profile. It never touches the
  operator's Chrome profile.
- **15 fps cap.** Output scales linearly with the cap, and `pair term`, zellij
  and couch each re-parse it (measured: `pair term` alone costs 21% of a core
  at 60 fps and 6% at 15 on a full-motion page). M2 measures the whole chain and
  confirms or revises the value.

**Name and label.** A browser tab's name is its handle.
- It gets an automatic default (`web`, `web-2`, …), unique across the pair
  tag, so a handle always exists. Alt+r renames it (for example `local-test`),
  and a renamed tab is marked *named*. #293's Alt+click reuse rule reads that
  mark.
- The strip label is `name · host[:port]`, e.g. `local-test · localhost:1111`.
  `pair term` reads the url and title over the DevTools protocol, not from the
  screen, and the label follows navigation.
- Page titles and URLs are page-controlled, so they are sanitized before they
  reach the strip or the record.

**Setting the URL.** Carbonyl's own bar can't be cleared quickly (see Log).
Clicking an already-active browser tab opens the strip's URL field, prefilled
with the current URL (Ctrl+U clears it, and paste works). Enter navigates
through the DevTools protocol. `localhost:1111` becomes `http://localhost:1111`.

**Shared with the agent.**
- **A record per tab** under the tag's `browser-<tag>/` directory, written
  through `cmd/internal/artifactpath`. It holds the name, the named flag, url,
  title, the DevTools HTTP and websocket addresses, the Carbonyl pid and birth,
  the owning `pair term`'s pid and birth, and the profile dir.
- **`pair browser`** prints JSON. With no argument it lists the current tag's
  tabs; `pair browser <name>` and `pair browser <tag>:<name>` print one. An agent
  passes `cdp_http` to Playwright's `connectOverCDP` (or a DevTools MCP server)
  and drives the page the operator is watching.
- **The record is a pointer, not the truth.** Readers check the Carbonyl
  process's identity (pid and birth time) before trusting the address, and skip
  the record if it fails. Only the owner writes or deletes a record; a crashed
  owner's records are swept when the owner is proved dead.

**Lifecycle.** Everything created names its end.
- **Closing the tab (Alt+w), or `pair term` exiting** (last tab, session quit,
  park or archive via SIGHUP/SIGTERM) SIGKILLs the whole Carbonyl process group,
  then deletes the record and the profile.
- **`pair term` dying without cleanup (SIGKILL, crash):** closing the pty
  master SIGHUPs Carbonyl's group, and the tree dies (measured: gone within
  0.5 s). The record and profile left behind are swept on the next browser
  launch, once their owner is proved dead.
- **A couch crash ends nothing.** The zellij session and `pair term` outlive
  couch, so the tab and its browser stay live and reattach with the thread.
  This is not a leak: the owner is alive.

**Alt+click** on a URL moves to pair#293 (operator, 2026-09-19). That issue now
depends on this one.

**Security.** The DevTools port gives full control of that browser to any local
process (it listens on 127.0.0.1 only). That's acceptable *because* each tab
uses a throwaway profile that holds nothing sensitive.

**Engine risk, accepted by the operator 2026-09-19.** Carbonyl is
unmaintained: last upstream commit 2023-02-26, last release v0.0.3, bundling
Chromium 111. Nothing maintained does what it does (a real engine rendered as
terminal *text*, with CDP for the agent); Browsh tracks current Firefox but
can't be driven over CDP. Two conditions follow:
- **The engine stays swappable.** It sits behind "launch a binary, speak CDP",
  so a replacement keeps the record, `pair browser`, the chords, the label and
  the lifecycle.
- **A remote URL is never opened silently.** `IsLocalURL` classifies loopback,
  `localhost`, private and link-local addresses, `file:`, `about:` and `data:`
  as local. A remote URL in the URL field flashes `Chromium 111, unpatched
  since 2023 — Enter again to open <host>, or Esc` and needs a second Enter.
  pair#293 should prefer `open` (the patched system browser) for remote
  Alt+clicks.

**Limit.** Chrome 111 (early 2023) is fine for a development preview of your own
dev servers, and for agent-driven checks. It doesn't stand in for testing against
current Chrome. Carbonyl displays one page, so an agent should drive the existing
page, not open new ones.

## Done when

- Alt+B (right pane) and Shift+Alt+B (any pane) open the URL field, and Enter
  opens a Carbonyl tab on that URL. Alt+h lists both.
- The tab label shows `name · host` and follows navigation.
- Clicking the active browser tab opens the URL field, and Enter navigates
  through DevTools.
- `pair browser <tag>:<name>` returns a live DevTools address. A test connects
  over the DevTools protocol, navigates, and sees the label change. The same
  runs against real Carbonyl in the live conformance probe.
- Closing the tab, `pair term` exiting, and SIGKILL of `pair term` each leave no
  Carbonyl or Chromium processes. The first two also leave no record or
  profile; after the third, both are swept once the owner is proved dead. A test
  covers each.
- A Carbonyl older than 0.0.3 gets a strip notice naming the idle-CPU bug.
- A remote URL in the URL field needs a second Enter; a local one doesn't. A
  test covers both, and `IsLocalURL`'s local set is closed (fuzzed).
- The chosen frame-rate cap, and the measured CPU of Carbonyl, `pair term`,
  zellij and couch on an idle and a full-motion page, are recorded in the Log.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

**Derivation:**
- **Design hours:** v2 ranges, with the ×0.2 spec-quality discount on every
  code primitive. The durable plan resolves their decisions, with code and
  tests written out.
- **Undiscounted design:** `issue-spec`, which is the spike and design already
  spent since the claim, and `ux-rename-iteration`: three operator smoke rounds
  plus the chain CPU measurement.
- **Library check (Step 2.5):** `coder/websocket` halves the CDP client's
  discounted design range; 0.3 was picked.
- **Implementation hours:** 40% of the v2 ranges (v3.1), picked at the upper
  part of each range for items that carry fuzz or pty tests.
- **Familiarity 1.2:** the Carbonyl/CDP stack is novel but bounded, inside a
  familiar codebase, so the multiplier sits between ×1.0 and ×1.5.
- **Design buffer +15%:** thorough plan doc (v2.1).

```estimate
model: estimate-logic-v3.1
familiarity: 1.2
item: issue-spec               design=1.0 impl=0.12
item: smaller-go-module        design=0.05 impl=0.2
item: greenfield-go-module     design=0.2 impl=0.32
item: smaller-go-module        design=0.05 impl=0.12
item: smaller-go-module        design=0.05 impl=0.16
item: smaller-go-module        design=0.05 impl=0.16
item: cross-cutting-refactor   design=0.1 impl=0.2
item: tui-screen               design=0.2 impl=0.4
item: api-integration          design=0.3 impl=0.4
item: tui-screen               design=0.2 impl=0.32
item: tui-screen               design=0.2 impl=0.4
item: smaller-go-module        design=0.05 impl=0.2
item: real-api-discovery       design=0.0 impl=0.24
item: ux-rename-iteration      design=0.5 impl=0.08
item: smaller-go-module        design=0.05 impl=0.16
item: smaller-go-module        design=0.05 impl=0.2
item: smaller-go-module        design=0.05 impl=0.08
item: skill-or-dispatcher      design=0.1 impl=0.16
item: atlas-docs               design=0.1 impl=0.04
item: milestone-review         design=0.1 impl=0.16
item: milestone-review         design=0.1 impl=0.16
item: milestone-review         design=0.1 impl=0.16
design-buffer: 0.15
total: 9.46
```

**Revision 2026-09-19 (post-gate):** one `smaller-go-module`
(design=0.05 impl=0.08) added for `IsLocalURL` plus confirm-on-remote, after
the operator accepted the unmaintained engine on that condition.
9.31 → 9.46.

**Item order**, top to bottom:
1. Spike and design.
2. M1: pure core, fakecarbonyl, `ptychild` group kill, profile store, strip
   field, chords.
3. Controller and lifecycle.
4. M2: CDP client, state machine, DevTools phases and URL field, conformance
   probe, real-API discovery, operator rounds.
5. M3: record, artifactpath, `pair browser`, atlas.
6. Three milestone reviews.

## Plan

Detailed tasks: `workshop/plans/000292-carbonyl-browser-tab-plan.md`.

- [x] Spike: URL bar, terminal modes, process tree, crash path, CPU by `--fps`
  (Log, 2026-09-19).
- [ ] M1 — Browser tab lifecycle in `pair term`: tab kind, URL field for new
  tabs, launch (binary lookup, version notice, profile), group kill on close and
  exit, crash sweep of profiles, Alt+B / Shift+Alt+B plus Alt+h, tab keys owned
  by browser tabs, default names and the named flag.
- [ ] M2 — DevTools: CDP client, the browser-tab state machine, the `name · host`
  label, click-the-active-tab URL field → navigate, live conformance probe, and
  the chain CPU measurement that settles the cap.
- [ ] M3 — Record and `pair browser`: artifactpath family plus GC registration,
  owner-only writes and the dead-owner sweep, the subcommand, the end-to-end
  DevTools test, atlas, operator smoke test.

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

## Revisions

### 2026-09-19 — Spec rewritten after the spike (pre-change-code)

Reasons: the spike (Log) and the operator's two answers this session.
- **Owner:** couch → `pair term`.
- **Opening:** Alt+B now opens a URL field; Enter launches.
- **URL setting:** "native bar or floating input" → the strip field plus a
  DevTools navigate.
- **Version:** Carbonyl ≥0.0.3 recommended, with a notice below it.
- **Profile location:** outside the data root.
- **Record:** a per-tag directory, owner-only writes.
- **Lifecycle:** "couch crash kills the browser" → ownership by `pair term`
  (couch's crash leaves the session and the browser live).
- **Alt+click:** moved to pair#293.
- **Done-when:** follows every change above, adds the version notice, and
  names the chain CPU measurement.

### 2026-09-19 — plan approved; handoff

- **Plan approved.** `sdlc change-code` passed. Plan-quality needed 3 rounds:
  - PQ-1…6 were addressed in round 1.
  - PQ-7 was advisory and fixed structurally with `ptychild`
    `Options.KillGroup`.
  - The gate's round-3 note, `Start()`'s `initTerminal`-failure kill as a
    fourth site, is folded into Task 1.3.
- **Branch** `000292-carbonyl-browser-tab` is in place in `~/workspace/pair`.
  The unrelated dirty files (Makefile typechange, `bootstrap.sh`,
  `merge-check.yml`, `scripts/issue-sync.sh` deleted,
  `scripts/merge-checks.d/40-duplicate-issue-id.sh`) are NOT ours; leave them
  unstaged.
- **Estimate-quality was info (non-blocking).** It judged 9.31 h likely low,
  by about 2–3 h:
  - one `ux-rename-iteration` item for three operator rounds;
  - no item for Task 3.3, or for the fake's DevTools half (Task 2.2);
  - `familiarity` 1.2 where the table's novel-but-bounded row is ×1.5;
  - the `coder/websocket` veto branch not costed.

  Not revised: the estimate stands as derived at the gate, and the close ledger
  will measure the gap.
- **Open operator decisions:**
  - Accept or veto `github.com/coder/websocket` (plan header table).
  - Install Carbonyl ≥0.0.3. The npm `latest` tag is 0.0.2, which spins a core
    when idle. Use `npm i -g carbonyl@next`, or the v0.0.3 release zip with
    `PAIR_CARBONYL=<path>`.
- **Spike harness** (scratch, not committed):
  `/tmp/claude-501/spike/{spike.py,termcost.py,site/}`, plus the v0.0.3 zip
  unpacked at `/tmp/claude-501/spike/c003/carbonyl-0.0.3/carbonyl`. Pty tests
  and the probe need the sandbox off.
- **Next:** M1 Task 1.1 (`cmd/internal/browsertab` pure helpers + fuzz), per
  `workshop/plans/000292-carbonyl-browser-tab-plan.md`. Run `sdlc state` first.

### 2026-09-19 — operator decisions on the engine and the dependency

- **`github.com/coder/websocket` approved** (ISC, zero transitive deps, last
  commit 2026-06-15). It supplies the CDP client and the test fake's server
  half; the in-tree alternative was ~250–350 lines of framing, masking,
  continuation and close-handshake code plus its own fuzzing.
- **Carbonyl accepted, on two conditions**, after its maintenance state was
  measured: last commit 2023-02-26, last release v0.0.3 (2023-02-18), 19.5k
  stars, 90 open issues, bundling Chromium 111.
  - The engine stays swappable behind "launch a binary, speak CDP".
  - Remote URLs are confirmed once before opening (`IsLocalURL`).
- **Alternatives checked.** Browsh (last commit 2025-07-05) drives your
  installed, patched Firefox and also renders real text, but Playwright can't
  attach to Firefox over CDP, so the agent-sharing half of this issue would be
  lost, and it costs more CPU. A current headless Chrome streaming screenshots
  is not viable in-pane: downscaled text is unreadable, and zellij doesn't pass
  terminal image protocols through.
- **Estimate** 9.31 → 9.46 for the local-URL guard.
