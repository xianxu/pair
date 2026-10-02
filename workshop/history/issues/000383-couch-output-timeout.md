---
id: 000383
status: done
deps: []
github_issue:
created: 2026-10-02
updated: 2026-10-02
estimate_hours:
card_mirror: 'fcf9e4c06cc14c82d1fd07b7a3e816afc0782acf' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-02T08:35:52-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "f5a3f7ef", done: "e06b4628"}
actual_hours: 0.48
---

# Couch exits on stalled terminal output and fails to restore keyboard modes

## Problem

Couch was found exited after running overnight on 2026-10-02 in Ghostty on
macOS. The screenshot showed:

```text
terminal: presenter failed
terminal: input write accepted 1024/25493 bytes: context deadline exceeded
terminal: input write accepted 0/118 bytes: context deadline exceeded
```

Keyboard escape sequences then appeared literally at the shell prompt.
Underlying agent/Zellij sessions remained running; Couch had been restarted
by the time of inspection. Despite the error's “input write” wording, the
presenter uses the same WriteFailure type for parent-terminal output.

The confirmed defect is inconsistent timeout ownership:

- `cmd/internal/terminal/profile.go:23` declares a five-second WriteTimeout,
  applied by `Presenter.write` in `terminal/presenter.go:198`.
- `cmd/internal/ttyio/writer.go:166` imposes its own two-second deadline inside
  File.WriteContext, cutting off the caller while its five-second context is
  still live.
- `cmd/internal/couchtty/console.go:1044` independently limits release to two
  seconds.
- A paint failure fails the presenter; teardown reports both the original
  failure and failed release. With zero reset bytes accepted, keyboard modes
  can remain enabled even after termios is restored.

Commit `00e95d1833ca77e1818fe1a6406747b56311a77a` (#322) extended the terminal
constant to five seconds, but left both lower limits. Its regression test
asserts the constant, not the effective transport deadline.

**Unknown:** why Ghostty stopped draining output during the original incident.
The power log contained no sleep/wake event on 2026-10-02. The restarted
Couch had COUCH_TRACE, COUCH_INPUT_TRACE, and COUCH_MOUSE_TRACE unset; that does
not prove the crashed process had the same environment. Fixing the timeout
mismatch alone does not establish that an arbitrarily long stall is solved.

## Spec

Make the intended terminal write budget effective through the production
transport and release path, with explicit ownership rather than competing
hidden limits (ARCH-DRY). Preserve bounded cancellation, accepted-prefix
accounting, and shutdown of owned workers. Never replay an entire partially
accepted packet.

Cover transient host-output stalls and terminal-mode cleanup through the real
PTY boundary. Determine and document behavior when output remains unavailable
past the recovery budget; do not claim successful mode restoration when no
reset bytes reached the host. Keep the original stall trigger separate from
the reproduced timeout defect.

## Done when

- A regression through the real ttyio/PTY path stalls output beyond two
  seconds, resumes within the intended five-second budget, and completes
  without an early timeout or duplicated bytes.
- A persistent-stall test confirms bounded failure and accurate partial-write
  accounting; cancellation remains responsive.
- Presenter/Console coverage verifies terminal-mode cleanup after a transient
  stall and explicit failure reporting when cleanup cannot reach the host.
- Write and release timeout ownership is consistent across the affected paths;
  tests exercise behavior rather than only checking a constant.
- Investigation records the original stall trigger if established, or preserves
  that uncertainty and identifies the diagnostic evidence still needed.

## Plan

Quick flow. One owner per budget (ARCH-DRY): ttyio is policy-free and honors
only the caller's context; `terminal.WriteTimeout` is the single terminal write
budget, applied by its consumers (Presenter, InputWriter); Presenter owns its
release budget instead of each caller wrapping it.

- [x] `ttyio.File.WriteContext`: drop the hidden 2s cap; the caller's context is
      the sole deadline (documented on `Writer`).
- [x] Presenter release bounds its drag cancellation with `WriteTimeout`
      (the reset write already gets its own via `write`); an incomplete reset
      write is reported as `ErrModesNotRestored` (wrapping the WriteFailure and
      its accepted count) — never as success.
- [x] `Console.release` stops imposing its own 2s; the start-up palette query
      gets `terminal.WriteTimeout` (it was implicitly bounded only by ttyio's cap).
- [x] `WriteFailure` names its direction (parent output vs child input) so the
      diagnostic stops saying "input write" for presenter output.
- [x] Tests (real PTY, master left unread to stall):
  - ttyio: stall 2.5s then drain → full write, no duplicated bytes, >2s elapsed.
  - ttyio: persistent stall under a 3s caller deadline → DeadlineExceeded no
    earlier than 3s; accepted count equals the bytes the master drains.
  - ttyio: cancel mid-stall returns promptly with accurate accounting.
  - Presenter: stalled paint resumes within budget; Release delivers reset
    controls exactly once.
  - Presenter: persistent stall → Release reports `ErrModesNotRestored`.
  - Console: release write sees the full `WriteTimeout` budget; diagnostic
    names unrestored modes.

## Log

### 2026-10-02 — incident investigation
- 2026-10-02: closed — Follow-up after codecomplete: removed redundant drag-cancel WithTimeout (mutation proved InputWriter owns the bound); new TestPresenterReleaseBoundsDragCancellationWhenChildStalls passes with and without the wrap, returning at 5.0s. Keyed WriteFailure literal. go test -race on ttyio/terminal/hostty/couchtty/termcmd/terminalqualify/ptychild green. Earlier full-suite evidence: make -k test only known test-changelog; go test ./... only the 3 known pre-existing failures. --no-done-when-fresh: the Revisions entry records an implementation detail (which layer bounds drag cancellation); the Done-when acceptance criteria are unchanged and still met.; review verdict: SHIP
- 2026-10-02: closed — Real-PTY regressions (ttyio x3, Presenter x3, Console release budget) red on main with incident signature 1024/262144 at 2.001s, green after; mutation checks confirm reverting the 2s cap or the ErrModesNotRestored wrap fails tests. make -k test: only known test-changelog fails (passes with scratchpad TMPDIR); go test ./...: only the 3 known pre-existing failures (TestBareCouchInstalledCommand, TestProductionArtifactReferencesAreExactlyClassified on console_messages.go, TestCouchReferencesLocalArchiveLocatorRoundTrip); go test -race on ttyio/terminal/couchtty green; post-review minors re-tested (ttyio/terminal/hostty/couchtty green).; review verdict: SHIP

Read-only investigation plus a temporary Darwin PTY probe reproduced the exact
screenshot counts. Open a PTY, leave its master unread, wrap the slave with
`ttyio.NewFile(slave, slave, false)`, and attempt writes of 25,493 then 118
bytes, each with a fresh five-second caller context:

```text
size=25493 accepted=1024 elapsed=2.000678s error=context deadline exceeded caller_error=<nil>
size=118 accepted=0 elapsed=2.001393917s error=context deadline exceeded caller_error=<nil>
```

This proves the hidden two-second cap and reproduces the backpressure
signature; it does not prove why Ghostty stopped reading. The temporary probe
was removed; no implementation changes were made.

Focused existing tests passed:

```sh
go test ./cmd/internal/ttyio ./cmd/internal/terminal ./cmd/internal/couchtty -run 'TestFileBlockedWritePreservesConcurrentReadAndFlags|TestWriteTimeoutAllowsFiveSecondsForTerminalIO|TestPresenterPartialFailureClosesAdmission|TestConsoleReleaseFailureIsReportedAfterRestoreAndFailsRun' -count=1
```

Those tests do not exercise real transport recovery between two and five
seconds. Filed at the operator's request; implementation has not started.

### 2026-10-02 — implementation

Budget ownership after the fix (ARCH-DRY):

- `ttyio.File.WriteContext` adds no deadline; the caller's context is the only
  one. Production callers that previously relied on the hidden cap were
  enumerated: child input (all via `InputWriter`, `WriteTimeout`), Presenter
  paint/release (`WriteTimeout`), and Couch's start-up palette query, which
  passed the deadline-less `c.lifetime` and now gets `terminal.WriteTimeout`.
- Presenter release bounds drag cancellation with `WriteTimeout` and its reset
  write with `WriteTimeout` (via `write`), so `Console.release` and
  `termcmd`'s `Release(context.Background())` both get the full budget with
  no competing caller timeout. Worst-case release is therefore bounded at
  2 × `WriteTimeout`.
- An incomplete reset write returns `ErrModesNotRestored` wrapping the
  `WriteFailure`; teardown prints it, so the operator sees
  "parent modes not restored (run `reset`)" instead of an implied restore.
- `WriteFailure.Op` names the direction: the incident's message now reads
  "parent output write accepted …", not "input write".

Red → green evidence: before the fix, the transient-stall PTY test failed with
`write accepted 1024/262144 after 2.001s: context deadline exceeded` (the
incident signature); the persistent-stall test failed at 2.0s against a 3s
caller deadline; the Console release write saw 1.99s of budget. Mutation
checks: restoring the 2s cap fails the transient Presenter test, and dropping
the `ErrModesNotRestored` wrap fails the persistent one.

**Original stall trigger: still unknown.** Nothing new was established about
why Ghostty stopped draining overnight. Behavior past the budget is
deliberate: a stall longer than `WriteTimeout` still fails the presenter and
exits Couch; release then retries the reset with a fresh budget, so a host
that comes back restores its modes, and one that doesn't is reported. Evidence
still needed to identify the trigger: a recurrence with `COUCH_TRACE` set (the
terminal trace records the failing write and release), the wall-clock time of
the failure against the macOS power/display log (`pmset -g log`, display
sleep and App Nap rather than system sleep alone), and Ghostty's own log for
the same window.

### 2026-10-02 — close review round 1 (SHIP, 4 Minor)

- `ErrModesNotRestored` split the WriteFailure method set → moved below `Unwrap`.
- Bare `ttyio.File.Write` / `OSHost.Write` are now deadline-less → documented
  on both; no production caller uses them (enumerated in the implementation
  entry), and the bounded paths use `WriteContext`.
- atlas/terminal.md line run-on → line break.
- PTY stall harness duplicated in ttyio and terminal tests → kept. ttyio's
  in-package tests cannot import a `ttyiotest` helper that imports ttyio
  (import cycle), so consolidating means moving those tests to an external
  package for about 40 lines of fixture; not worth it for two sites.
- The first close attempt was not finalized: HEAD went detached at the same
  commit (b2424cb4) 31s into the review, while sdlc had the lock released and
  no command from this session was running. Reattached to the branch and
  re-ran the close.
- Lesson added: a budget has one owner; test the effective deadline.

### 2026-10-02 — close review round 2 follow-up (SHIP, 2 advisory)

- "Release's drag-cancellation bound has no test" → added
  `TestPresenterReleaseBoundsDragCancellationWhenChildStalls`. The mutation
  disproved the finding's premise: with the `WithTimeout` wrap removed,
  Release still returned at 5.0s, because the child's `InputWriter` already
  bounds every delivery with `WriteTimeout` (a failure drops the queue and
  ends Flush). The wrap was a second, redundant budget, the pattern this issue
  removes, so it is gone. The test stays and guards the real owner. Corrects
  the implementation entry: drag cancellation is bounded by `InputWriter`,
  not by a Presenter wrap; the worst case is still 2 × `WriteTimeout`.
- Positional `WriteFailure` literal → keyed, matching presenter.go.

## Revisions

- 2026-10-02 — Plan item "Presenter release bounds its drag cancellation with
  `WriteTimeout`" is superseded: a mutation showed `InputWriter` already owns
  that bound, so the extra wrap was removed (see round 2 follow-up in the Log).
