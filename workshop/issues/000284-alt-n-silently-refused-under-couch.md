---
id: 000284
status: open
deps: []
github_issue:
created: 2026-09-18
updated: 2026-09-18
estimate_hours:
---

# Pair's Alt+n in a Couch-hosted thread confirms, then silently does nothing

## Problem

Found while scoping `pair#282`, by reading code rather than by pressing the key.
Two changes from the same day compose into a dead key:

- `#245` (`df2a8897`, 2026-09-14) passes Alt+n / Ctrl+Alt+n through to Pair
  while a Pair pane is displayed. Couch only acts on them in its switcher
  (`couchtty/console.go` `dispatchInputCandidate`: non-`actorReserved` hits are
  routed raw to the focused child).
- `#249` (`d87fe67c`, 2026-09-14) made `pair restart` refuse when Couch hosts
  the thread (`launcher/runcli.go`, the `args.Command == "restart"` branch):
  *"pair: hosted inner restart is unsupported; use Couch relaunch (Alt+n)"*.

So in a Couch-hosted Pair pane, Alt+n opens `PairConfirmRestart`'s "Reload
pair?" dialog. Confirming runs `pair restart`, which refuses. The refusal goes
to stderr, and `pair_confirm_restart_impl` (`nvim/init.lua`) calls
`vim.fn.system(argv)` without checking `shell_error`, so the operator sees
nothing. The refusal text also points at "Couch relaunch (Alt+n)". From a Pair
pane that is the same key that just failed; the working route is Alt+n in the
switcher.

`README.md`'s Alt+n row still says it reloads pair, with a trailing note about
the switcher.

## Spec

To be designed. The candidate directions, none chosen:

- Surface the refusal. At minimum the draft notifies on a non-zero exit, and the
  message names the switcher route.
- Skip the confirm dialog under Couch, and say where relaunch lives instead.
- Route Pair's Alt+n to Couch's relaunch of this thread, if Couch has a door a
  child process may knock on (`RequestCouchContinuation` is the existing
  precedent for a hosted Pair asking Couch to act).

`pair#282` documents the current truth in Alt+h ("unavailable while Couch hosts
this thread"). Whatever this issue changes must update that `HostedHelp` and the
launcher test that pins the refusal.

## Done when

- [ ] Alt+n in a Couch-hosted Pair pane either works or visibly says why not and
      where to go instead; it never confirms and then does nothing.
- [ ] Alt+h's hosted wording and README agree with the new behavior.

## Plan

- [ ] Decide the direction with the operator.

## Log

### 2026-09-18

- Filed from `pair#282` scoping (code reading, not a live repro). Confirmed that
  the draft nvim in a Couch thread carries `COUCH_THREAD_SCOPE`/`COUCH_THREAD_TAG`
  (`ps eww` on the draft pid), which is what the launcher's hosted check reads.
