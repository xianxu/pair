---
id: 000284
status: done
deps: []
github_issue:
created: 2026-09-18
updated: 2026-09-19
estimate_hours:
started: 2026-09-18T22:36:30-07:00
flow: {kind: full, provenance: inferred}
actual_hours: 2.68
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

**Direction (operator, 2026-09-18): Couch takes Alt+n and Ctrl+Alt+n from every
Pair pane.** In a Couch thread, Alt+n relaunches the thread on screen (current
binary, same conversation) whichever inner pane has focus. In the switcher it
still relaunches the highlighted row. Standalone Pair is unchanged.

This partly reverses #245 for these two chords, by operator choice. #245 took
them away from Couch so a focused agent could receive them, and assumed the draft
would keep a working Pair restart. #249 broke that assumption the same day, so the
two chords did nothing Pair-side under Couch. The agent loses Alt+n/Ctrl+Alt+n
under Couch, as #257 proposes for Alt+j/Alt+k inside Pair. Rejected
alternatives: a Pair→Couch relaunch request through a record mailbox (keeps
#245, but needs a new request lifecycle, since #147's live-owner door is punted),
and redirect-only (Alt+n still would not reload). "Ctrl+Space, then Alt+n" is not
a safe redirect either. Ctrl+Space opens on the newest *paging* thread
(`onHotkey`), not the one on screen.

Design, reusing what exists (ARCH-DRY):

1. **Couch.** `couchkeys` declares both relaunch chords `ScopeEveryPane`.
   `actorReserved` reads the scope, and `onRelaunchHotkey` already has an actor
   arm that targets `p.thread`, moves focus to the panel for the confirmation, and
   `finishOperation` re-attaches the relaunched child with focus. No new handler
   is needed. The help row's text covers both scopes. `Claimed` + `keyhelp.Layer`
   (#282) then replace Pair's Alt+n rows on the Alt+h page when Couch presents
   the client, with no edit outside the table.
2. **Pair's refusal happens before anything is destroyed (the adopted case).** Pair
   still sees Alt+n when Couch does not present the client: a Couch-created
   session attached from a plain terminal, or `pair restart` typed by hand. One
   predicate names the rule, "Couch owns this session's restarts", as session env
   hosted OR client presented by Couch (the `outer-tty` record, #282). It becomes
   a `launcher` function that both the `pair keys` hosted-wording choice
   (`keyscmd`) and the gates call, so the help derives from the enforcement
   (ARCH-PURPOSE). The presenter read goes on the `Runtime` seam next to
   `RecordOuterTTY`. The fake stores what `RecordOuterTTY` wrote (ARCH-MOCK).
   Gated sites: the class is **in-session restart-marker writers that kill a
   live session whose client refuses the marker**, and it has two members.
   - `runRestart`: after resolving the tag, before `WriteRestartMarker`.
   - `runCompaction`'s standalone path: env-hosted already routes to Couch, but
     presented-not-hosted has no thread address to route with, so it refuses.
   - `continue --retry` is excluded because it requires the source session
     already gone (`prepareContinuationRetry`).
   - An unreadable record refuses. It is untrusted input and gates a destructive
     action, so it fails closed. `pair keys` keeps failing open to Pair's page
     (ARCH-SECURE).
   - Refusal text names the working route: "relaunch it from Couch (Alt+n)".
3. **The draft stops confirming and then going silent (the class of silent
   lifecycle keybinds).** One helper runs a confirmed lifecycle command and, on
   non-zero exit, `vim.notify`s its output at ERROR. That is the pattern
   `PairConfirmAgentRestart` already uses. Its four users: `PairConfirmQuit`
   (`pair quit`), `PairConfirmDetach` (`zellij action detach`),
   `pair_confirm_restart_impl` (`pair restart`) and `PairConfirmAgentRestart`
   (`pair agent restart`). On success the first three end the session, so the
   notify fires only on failure.

ARCH notes. PURE: the predicate takes values, and the presenter read is the seam.
CONSTRAINTS: keystroke path, one table lookup more, no IO on the input path.
ORDER: no new state held between events, since relaunch's outcome machine
(`relaunch.go`) is reused unchanged. The focus-after-prefix ordering is pinned by
existing tests, and Alt+n joins their chord set. FUNERAL: nothing created.
Considered and rejected: a `pair restart --check` preflight so the draft could
skip the dialog in the plain-terminal case. That case is rare and now fails
visibly, so the preflight is not worth a flag.

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

- [x] Alt+n in a Couch-hosted Pair pane either works or visibly says why not and
      where to go instead; it never confirms and then does nothing.
- [x] Alt+n in a Couch-presented thread Couch did not create never ends the
      thread; a test drives that case.
- [x] Alt+h's hosted wording and README agree with the new behavior.
- [x] With a Pair pane displayed, every encoding of Alt+n/Ctrl+Alt+n opens Couch's
      relaunch confirmation for the thread on screen (even while another thread
      pages), and no byte of the chord reaches the child. Alt+d/Alt+x/Alt+Shift+N
      still pass through.
- [x] `pair restart` and in-session compaction refuse before writing any marker
      when Couch presents the client or its record is unreadable.
- [x] A failed quit/detach/restart/agent-restart in the draft shows its error.
- [x] Operator smoke on a rebuilt binary: Alt+n from the agent pane and from
      the draft of a Couch thread relaunches it with the conversation kept.

## Plan

- [x] Decide the direction with the operator. Couch takes Alt+n (2026-09-18).
- [x] Couch: relaunch chords → `ScopeEveryPane`, help row covering both scopes.
      Tests (red first): actor-focus Alt+n/Ctrl+Alt+n on every encoding → relaunch
      confirmation for the on-screen thread with another thread paging, zero
      chord bytes to the child, surrounding bytes forwarded. Update
      `TestActorLifecycleCandidatesPassThrough` (Alt+d/Alt+x only),
      `TestClaimedIsEveryPanePairChordsOnly`, and run
      `TestNoPairRowSharesAnEveryPaneCouchKey` on the layered page.
- [x] Launcher: the ownership predicate plus the `Runtime` presenter read. Gate
      `runRestart` and `runCompaction`. `keyscmd` calls the predicate. Tests:
      presented-not-hosted, hosted, unreadable record → exit 1, no marker, no
      quit intent, no kill. Standalone still restarts. Update
      `TestPresenterAndHostingAreIndependent` for the claimed rows.
- [x] Draft: the shared notify-on-failure helper for the four confirmed lifecycle
      commands, with a Lua test that a non-zero exit notifies.
- [x] Docs: `HostedHelp` for Alt+n/Ctrl+Alt+n, README (the table row + couch
      section), `atlas/couch.md`, `atlas/architecture.md` hosting note.
- [x] `go test` on touched packages, then `TMPDIR=<scratchpad> make test` in
      full. Rebuild and ask the operator to smoke test.

## Log



- 2026-09-19: closed — make test green (exit 0; 212 Go pkgs + shell/Lua incl. tests/lifecycle-command-nvim-test.sh). Mutation-checked: reverting ScopeEveryPane reds the couchtty confirmation test; removing either launcher gate reds its test; reverting a draft call site to vim.fn.system reds the wiring test; restoring the --retry advice reds the recovery-route test. Operator smoke 2026-09-19: Alt+n from agent pane and draft in a Couch thread relaunched it, conversation kept.; review verdict: SHIP
- 2026-09-19: flow upgraded quick → full — 136 added lines in code files (limit 100); an earlier round of this close already ran the full review
### 2026-09-18

- Filed from `pair#282` scoping (code reading, not a live repro). Confirmed that
  the draft nvim in a Couch thread carries `COUCH_THREAD_SCOPE`/`COUCH_THREAD_TAG`
  (`ps eww` on the draft pid), which is what the launcher's hosted check reads.
- Claimed. Traced the paths:
  - `couchkeys.go:89` declares relaunch `ScopeSwitcher`, so `actorReserved`
    is false and `dispatchInputCandidate` forwards the chord raw.
    `onRelaunchHotkey` (`console.go:1517`) already handles actor focus
    (pre-#245 behavior), so the dead code path is the fix.
  - #245's plan (§Scope) deliberately passed the lifecycle chords through. It
    expected the draft to keep Pair's restart, which #249 refused the same day.
  - #147 (live-owner routing) is punted. The only child→Couch door is the
    continuation record mailbox, polled every 500 ms (`watchContinuations`).
  - `continue --retry` needs its source session gone, so it is not in the
    adopted-case class.
  - #246 (adoption) is still open, so the adopted case is latent today. Once
    #246 lands, Couch's interception covers Alt+n, and the launcher gate covers
    a hand-typed `pair restart` and compaction.
- Operator chose "Couch takes Alt+n" over the mailbox and redirect-only
  options.
- Implemented:
  - `couchkeys` puts both relaunch chords on a shared `pairChord(scope, …)`
    constructor, replacing `switcher`.
  - `launcher.CouchOwnsRestart` and `couchRestartGate` gate `runRestart` and
    `runCompaction`. `Runtime.OuterPresenter` is the new seam. The fake reads
    back its own `RecordOuterTTY`.
  - `keyscmd` derives its hosted wording from `CouchOwnsRestart`.
  - `nvim/lifecycle_command.lua` is registered in the artifact manifest and in
    `test-lua`.
  - Red first: the new Couch test timed out waiting for the confirmation before
    the scope change. Mutation-checked: removing either launcher gate fails its
    test.
- Two misses caught by the full `make test`:
  - `init.lua` hit Lua's 200-local ceiling (E5112), which broke every later
    definition. `workshop/lessons.md` already has this rule (#66), and I added
    two top-level locals anyway. Moved the helper into a `do` block shared as
    `_G._pair_lifecycle`.
  - The #245 cross-layer test `wrapcmd` `TestConsoleWrapperShortcutPassthrough`
    still sent Alt+n unpasted through Couch to the agent. My sweep had covered
    only the packages I changed. Alt+n now sits inside its paste span, where it
    must stay literal through both layers. A tree-wide sweep for `110;3u` /
    `110;7u` / `ChordAltN` / `ChordCtrlAltN` found no other test assuming
    passthrough.
- Verification:
  - `make test` is green (exit 0): 212 Go packages plus the shell and Lua
    suites, run unsandboxed with the scratchpad `TMPDIR` and the five-var
    retention scrub.
  - Operator smoke, 2026-09-19, on the rebuilt pair and Couch: Alt+n from the
    agent pane and from the draft of a Couch thread relaunched it with the
    conversation kept. Smoke setup hit a separate leave bug, filed as
    `pair#291`.
- Close round 1: FIX-THEN-SHIP, 2 Important + 5 Minor, all fixed before the
  close commit (the flow upgraded quick → full at 118 added code lines).
  - BR-1 compaction's refusal stranded a validated checkpoint and named an
    operation that discards it. Fixed as its CLASS — a refusal arm names what
    it retained and the route onward — which covers the sibling Minor on the
    unreadable-record message. The restart gate needs no such line: it refuses
    before anything is written.
  - BR-2 the four draft call sites had no failing test, so reverting one to
    `vim.fn.system` kept the suite green. Added
    `tests/lifecycle-command-nvim-test.sh` (wired into `make test`): it drives
    the real `init.lua` headlessly, asserts each entry point routes through
    `_G._pair_lifecycle.run`, and fails on any direct `vim.fn.system`.
    Mutation-checked by reverting the detach site.
  - Minors: pass `opts.Env.CouchHosted()` rather than a literal false; garbled
    comment in `keyscmd`; the Lua module takes `error_level` as a dep so it
    reads no global; `atlas/architecture.md`'s confirm-modals entry names the
    new rule and seam; the help row reads for both scopes and still fits the
    page (86 chars of the 104-col budget).
- Close round 2: 7 findings disposed, BR-1 not addressed, 2 new Minors.
  - BR-1 again, and rightly: the route I named, `pair continue --retry`, needs
    a retained restart marker, and this arm returns before writing one. The
    reviewer ran it. The route that survives the refusal is the checkpoint doc,
    so the message names `pair continue --checkpoint <path>`.
    `TestCompactionRefusalNamesARouteThatRuns` pins the class: no marker was
    written, the named route parses, and its path resolves and validates.
    Mutation-checked by restoring `--retry`.
  - A comment in `lifecycle_test.go` still described the pre-#284 behavior.
    Fixed, and swept all 25 `#284` mentions in cmd/nvim/tests for tense.
  - The Spec claimed Alt+n joined the focus-after-prefix tests; it had not.
    `TestLifecycleCandidateUsesFocusAfterPrefix` now runs both chords and
    pins the arm where they differ: after navigating back to an actor, Alt+x
    reaches the child and Alt+n confirms against the thread on screen.

## Revisions

### 2026-09-18: direction chosen; Done-when made concrete

**Reason:** the operator picked "Couch takes Alt+n from every pane".
**Delta:** the Spec's candidate list became the design above. Done-when gains
four rows: the Couch routing contract, the gate before any mutation, the draft
error surfacing, and operator smoke. The original three rows stand.

### 2026-09-18: the adopted case is destructive, not benign

**Reason:** #282's third plan review showed that Alt+n has a client-side gate
as well. The earlier text said an adopted session's Alt+n "is not refused" and
the help was right there. In fact the thread is torn down and not relaunched.
**Delta:** the Spec paragraph above is corrected in place, and a Done-when row
is added for the adopted case.
