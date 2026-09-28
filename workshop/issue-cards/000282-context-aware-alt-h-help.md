---
id: '000282'
status: done
started: 2026-09-18T09:31:36-07:00
created: 2026-09-17
updated: 2026-09-18
estimate_hours: 5.7
actual_hours: 2.27
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
