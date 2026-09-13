---
id: 000242
status: open
deps: []
github_issue:
created: 2026-09-13
updated: 2026-09-13
estimate_hours:
---

# couch defaults to --layout3: every thread gets pair's right-hand terminal unless --layout2 is asked for

## Problem

`couch` with no flag runs layout2 (`couchcore/couch.go:107`, `Layout:
Layout2` in `New`; the CLI overrides it only when a flag is given,
`couchcmd/run.go:266-269`; the help text says so at `:715`). The operator
runs layout3 every time, and has since #198 made it reachable: every thread
record in the store is layout3 —

    ~/.local/share/pair/couch/threadstore/records: 7 records
      7 × "layout": "layout3"     (tools, parley.nvim, pair, arc-agi-3, brain, astro, ariadne)
      0 × pre-#198 (no field)

— so the default is the one thing nobody uses. The operator has not hit a
failure here; the ask is simply that the default match how couch is used, so
`couch` means the workbench that is actually run. (A consequence worth
knowing, not the motivation: with one layout per couch process and a startup
guard that refuses to mix, a flagless `couch` next to layout3 threads refuses
to start — `atlas/couch.md:1075-1083`.)

The reasons layout2 was the default are recorded and gone: the 2026-08-22 pin
("couch owns terminal switching, so layout3's third pane is the layer couch
replaces") was an actor-cluster claim that #170's rescope to couch-lite
invalidated, and #198 reversed the pin without moving the default. The
operator's own list (2026-09-09): layout2 was kept for simplicity, doubt
about hosting a browser on the right, and the right pane's quality — all
three have since changed (#199 gave the right pane its own tab bar; browser
hosting was dropped; couch-lite is done).

## Spec

1. **`New` defaults to `Layout3`** (`couch.go:107`). `--layout2` remains the
   explicit opt-out; `--layout3` remains accepted and is now a no-op.
2. **Record normalization is untouched.** `ParseLayout("")` → `Layout2`
   (`layout.go:33-37`) is a statement about *records written before #198*,
   not about the default; it must not follow the new default, or every
   pre-#198 record would be misread as layout3 and the mixing guard would
   trust a fabricated witness. The comment at `actionableinventory.go:233`
   ("must read as Layout2 here or every existing thread conflicts with a
   default startup") is now literally true for the old records only; reword
   it so the next reader does not "fix" it.
3. **The mixing guard's message names the remedy for the new direction.**
   Today it is written for "you asked layout3 but hold layout2 sessions";
   after the flip the common case is a flagless start beside old layout2
   threads. Same refusal, same remedy (park them, or pass `--layout2`),
   worded for the flagless case.
4. **Docs.** `couch --help` (`run.go:706-715`), `README.md:281-282`,
   `atlas/couch.md:1068-1069`: `--layout3` is the default, `--layout2` opts
   out. `pair`'s own default is out of scope — `pair <agent>` with no flag
   resolves through `ResolveLayout` against the recorded session
   (`launcher/layout.go:53`) and is a separate decision; note it in the Log,
   don't change it here.

Out of scope: per-thread layouts (a couch process has one layout by design,
#198); changing what layout3 is.

## Done when

- `couch` with no flag starts threads in layout3; `couch --layout2` starts
  them in layout2; `couch --layout3` behaves as before.
- A store with a pre-#198 record (no layout field) still classifies it as
  layout2 (existing normalization test extended with the flipped default).
- Flagless `couch` beside a layout2 thread refuses with a message that names
  `--layout2` and parking as the two ways out.
- Help, README, atlas say layout3 is the default.

## Plan

- [ ] Flip the default in `New`; make the CLI test for "no flag" expect layout3
- [ ] Normalization test: `""` → layout2 survives the flip; reword the two comments
- [ ] Guard message for the flagless-beside-layout2 case; test
- [ ] Help/README/atlas

## Log

### 2026-09-13

- Filed from the brain advisor session on the operator's request — a default
  that matches usage, not a failure report; the first draft framed it as
  "forgetting the flag", which the operator corrected. Migration
  cost measured, not assumed: all 7 thread records on the operator's machine
  are layout3, none pre-#198, so the flip changes nothing for existing
  threads here. The pre-#198 normalization is the one place a careless flip
  would break something, hence Spec 2.
- `pair`'s own flagless default left alone on purpose; if the operator wants
  it to follow, that is one more line in `launcher/args.go`, a separate
  decision.
