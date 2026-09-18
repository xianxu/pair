---
id: 000282
status: working
deps: []
github_issue:
created: 2026-09-17
updated: 2026-09-18
estimate_hours:
started: 2026-09-18T09:31:36-07:00
---

# Alt+h help knows whether it runs under couch, and shows couch's keys there

## Problem

Alt+h (`PairOpenHelp`, `nvim/init.lua:3469`) shows `pair keys`
(`cmd/internal/keyscmd`), which describes pair's workbench and nothing else. Run
under couch, it is the only in-session help the operator has, and it is silent
about the layer they are actually standing in:

- **couch's own keys are absent.** Ctrl+Space (switcher), Ctrl+Backspace
  (previous thread), Ctrl+Return (newest notification), the switcher's lifecycle
  chords, and the terminal-tab keys couch reserves are documented only in
  `couch --help` — a command nobody runs from inside a thread.
- **Some of pair's own entries are wrong under couch**, and this is the part a
  simple "append couch's section" would miss. Couch intercepts chords before pair
  sees them: `couchtty/keys.go:228` keeps alt+d, alt+x, alt+n and ctrl+alt+n as
  couch's lifecycle chords, and
  `TestRelaunchChordsAreInterceptedAndAltShiftNIsNot` pins that alt+n is couch's
  while alt+shift+n still reaches pair. So pair's help currently tells a couch
  user:
  - alt+d — *"detach from the session (re-attach with `pair`)"*: under couch the
    way back is the switcher, not `pair`.
  - alt+n — *"reload pair — kill and re-launch the workbench in place"*: under
    couch this is couch's relaunch of the thread's pair process.

  The help is describing a different program than the one receiving the keys.

## Spec

Alt+h renders help for **the stack the operator is in**: pair alone, or pair
under couch.

### Detection

Two different questions, and the issue should answer both deliberately:

- **Hosted by couch** — the thread was launched by couch. The pair process
  environment already says so: `COUCH_THREAD_SCOPE`, `COUCH_THREAD_TAG`,
  `COUCH_THREAD_RESUME`, `COUCH_TREE`, `COUCH_STORE_DIR` are set in a couch
  thread's panes (verified 2026-09-17 in the agent pane; confirm the draft's
  nvim sees the same before relying on it). The launcher already branches on the
  first two (`launcher/runcli.go:117`).
- **Couch is presenting it right now** — a different fact. The env is written at
  launch and never revised. If couch exits, the zellij session survives (the
  server is PPID 1 at birth, `pair#275`), can be attached directly, and still
  carries the env — so the help would list Ctrl+Space for a couch that is gone.
  That is the same stale-witness shape as `pair#272`, in miniature.

Recommended: env decides *hosted*; the couch supervisor's singleton lease
decides *live*. `pair#278` is already adding a read of that lease to print the
supervisor's pid — reuse that read rather than inventing a second liveness
check. When hosted-but-not-live, say so in one line instead of listing keys that
will not work.

### Content — derive, don't copy

- **Couch's section comes from couch's key table.** `couchtty.CouchNavigationBindings()`
  already carries label + description per binding, and `couch --help` renders
  from it (`couchcmd/run.go:812`). Alt+h must consume the same source, so the two
  surfaces cannot drift. No hand-written copy of couch's keys in pair.
- **Pair's section already derives from its sources** (`keyhelp`, reading
  `nvim/init.lua`, `zellij/config.kdl` and the `workbenchshortcut` table, with
  drift tests). Keep that; don't fork it.
- **Overridden entries are corrected, not merely supplemented.** For each chord
  couch intercepts, pair's entry under couch shows couch's meaning (or is marked
  as couch's), so the page never lists two meanings for one chord. Derive the
  intercepted set from couch's chord table, not a list maintained in pair.
- **The switcher's own chords** (alt+d detach, alt+x park — "in an actor or in
  the switcher alike", `couchcore/park.go:111`) belong in couch's section. Note
  `pair#279`: alt+d is currently dead in the switcher; the help should describe
  the contract, and #279 makes it true.

### Layering

`pair keys` lives in pair and couch is a peer consumer of pair, so this is a
dependency question worth settling rather than inheriting: pair importing
`couchtty` just to read a key table may be the wrong direction. If it is, the
binding table moves to a small shared package both depend on, the way
`workbenchshortcut` already serves both today.

## Done when

- [ ] Alt+h under couch shows couch's keys, rendered from
      `CouchNavigationBindings()` — the same source `couch --help` uses — proven
      by a test that adds a binding and sees both surfaces change.
- [ ] Chords couch intercepts show couch's meaning under couch; no chord appears
      with two meanings. Derived from couch's chord table.
- [ ] Standalone pair's help is unchanged.
- [ ] Hosted-but-couch-not-running is detected and stated, not rendered as live
      couch keys. Decision on the detection signal recorded (env vs lease).
- [ ] `pair keys` from a shell reflects the same context as Alt+h.

## Plan

- [ ] Confirm the draft nvim's environment carries `COUCH_THREAD_*`, and what
      `PairOpenHelp` passes to `pair keys`.
- [ ] Settle the package direction for sharing couch's binding table.
- [ ] Context detection (env + lease, reusing #278's lease read); render couch's
      section; correct the overridden entries.
- [ ] Tests per Done when; atlas note on help sources.

## Log

### 2026-09-17

- Filed from a brain advisor session at the operator's request: *"it should be
  aware if we are in pair or couch, and when in couch, to also display help for
  couch."* No existing issue; nearest are #232/#233 (terminal-specific Alt
  bindings), unrelated.
- The overridden-entry problem (alt+d, alt+n) and the hosted-vs-live distinction
  were found while scoping; neither was in the request, both follow from it.
