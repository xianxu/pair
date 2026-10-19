---
id: 000684
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '12257f718bf90adabacea35a6a4ba3e9cfe8427f' # card fields mirrored from issue-cards; edit via sdlc
---

# fresh fallback after a failed resume archives the old thread without tearing down its actor and session: two live actors, Couch unusable

## Problem

On 2026-10-18 the operator relaunched parley.nvim:0 (codex thread 1-parley-nvim-37, zellij session 📁1-188 since Oct 11). The park worked (18:21:47, parked-scrollback-1-parley-nvim-37-20261018T182147.raw saved, agent stopped), but the resume did not, and Couch fell back to start fresh (18:21:57, mode fresh, selected_key fresh:1-parley-nvim-37:143), archived thread 37 (18:21:58) and started 1-parley-nvim-70 in a new session 📁1-311 (18:22:01).

Archiving thread 37 did not tear down what held it: the Couch actor couch-393ac172 (pair resume 1-parley-nvim-37, pid 60305) stayed attached, and the 📁1-188 zellij server (pid 6463, with its nvim 6467/6471 and a pair title 17468 from Oct 14) kept running. Couch then had two live actors on parley.nvim:0; both tabs showed raw thread names instead of :0, the operator could not switch to any regular slot, and had to kill Couch to recover. Investigation by the operator's pair-dev session; full notes in ops qa 2026-10-18-18-couch-hang-bug.md.

## Spec

A thread is archived only after its actor and terminal session are proven gone (the same proof-of-death standard #651 uses for a stranded park). The start-fresh fallback after a failed resume must run the same teardown as a normal relaunch before starting the new thread; if teardown cannot be proven, refuse with a visible reason instead of starting a second live actor on the slot. A slot must never present two live actors; if it ever observes two, it reports it rather than mislabelling tabs. Related: #683 (failure reasons must persist; this resume's failure left no launch-diagnostics entry either), #681.

## Done when

- A regression reproduces: resume fails, fresh fallback, old actor and session still alive; today two actors result, after the fix teardown completes first or the fresh start refuses.
- Archive refuses while the actor or session is alive.
- The operator's slot switcher keeps working through a failed relaunch (live check on a disposable slot).

## Plan

- [ ]

## Log

### 2026-10-18
