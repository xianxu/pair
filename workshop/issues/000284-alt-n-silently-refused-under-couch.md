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

`pair#282` documents the current truth in Alt+h ("does not reload under Couch;
relaunch from the Couch switcher", shortened at #282's close so the row fits the
page) through `GlobalBinding.HostedHelp`, selected when Couch launched the
session or presents the client. Whatever this issue changes must update that
`HostedHelp` and the two launcher tests that pin today's gates:
`TestCheckpointHostedRestartAndRenameRefuseBeforeMutation` and
`TestCouchClientRefusesRestartMarker` (the latter added by #282).

**Worse case: an adopted thread is ended, not just left alone.** Alt+n is gated
twice. `pair restart` checks the *session* env, and the attached client refuses
the restart marker when *its* env names Couch (`createflow.go:162`), but only
after it has run the full quit cleanup (`createflow.go:129`). In a session Couch
presents but did not create (`pair#246`), the first gate passes. `pair restart`
writes the marker and the quit intent, then kills the session. Couch's client
tears the thread down and refuses to relaunch it, so one keystroke ends the
thread. Measured by #282's third plan review from code, not from a live repro.

## Done when

- [ ] Alt+n in a Couch-hosted Pair pane either works or visibly says why not and
      where to go instead; it never confirms and then does nothing.
- [ ] Alt+n in a Couch-presented thread Couch did not create never ends the
      thread; a test drives that case.
- [ ] Alt+h's hosted wording and README agree with the new behavior.

## Plan

- [ ] Decide the direction with the operator.

## Log

### 2026-09-18

- Filed from `pair#282` scoping (code reading, not a live repro). Confirmed that
  the draft nvim in a Couch thread carries `COUCH_THREAD_SCOPE`/`COUCH_THREAD_TAG`
  (`ps eww` on the draft pid), which is what the launcher's hosted check reads.

## Revisions

### 2026-09-18: the adopted case is destructive, not benign

**Reason:** #282's third plan review showed that Alt+n has a client-side gate
as well. The earlier text said an adopted session's Alt+n "is not refused" and
the help was right there. In fact the thread is torn down and not relaunched.
**Delta:** the Spec paragraph above is corrected in place, and a Done-when row
is added for the adopted case.
