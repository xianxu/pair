# Boundary Review — pair#383 (whole-issue close)

| field | value |
|-------|-------|
| issue | 383 — Couch exits on stalled terminal output and fails to restore keyboard modes |
| repo | pair |
| issue file | workshop/issues/000383-couch-output-timeout.md |
| boundary | whole-issue close |
| milestone | — |
| window | b1de974ba1610ff32d633b536b482af13271d8f2..bbf2fdda4c005fb3439d49820e70040a58705bbd |
| command | sdlc close --issue 383 |
| reviewer | claude |
| timestamp | 2026-10-02T08:58:03-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

**Verdict: SHIP.** The fix removes the deadline conflict that caused the incident. `ttyio.File.WriteContext` no longer adds its own 2s cap (`cmd/internal/ttyio/writer.go:170`), and `Console.release` no longer wraps release in its own 2s timeout (`couchtty/console.go:1047`). `terminal.WriteTimeout` is now the one write budget. Presenter applies it per write and owns its release budget. I listed every production caller of `WriteContext`. The only one that had relied on the hidden cap is the start-up palette query, and it now gets `WriteTimeout` (`console.go:600`). No production code calls the now deadline-less bare `Write` on `ttyio.File` or `OSHost`. The new tests use a real PTY whose master is left unread, so the stall is genuine. A stall that resumes after 2.5s would have failed under the old 2s cap. I ran the affected packages unsandboxed (the sandbox blocks PTYs): ttyio, terminal, couchtty, hostty, termcmd and terminalqualify all pass. Nothing blocks the close.

1. **Strengths**
   - The effective deadline is tested, not the constant. `TestFileWriteSurvivesTransientHostStallWithinCallerDeadline` requires more than 2.5s elapsed, the full payload accepted and byte-exact delivery with no duplicates. `TestFilePersistentStallFailsAtCallerDeadlineWithExactPrefix` requires the failure to land between 2.9s and 4s and checks that the accepted prefix equals what the master drained.
   - A failed release is reported, not hidden: `ErrModesNotRestored` wraps the `WriteFailure` with `%w: %w` (`presenter.go:145`). The persistent-stall test checks both `errors.Is` and `errors.As`, and checks the accounting is 0/len.
   - `TestPresenterRestoresModesWhenHostResumesAfterFailure` covers recovery: after a paint failure, release still gets a fresh budget and restores the modes.
   - The Console test now measures the deadline the release write actually sees (`terminal_exit_test.go:101`), so a reintroduced short wrapper would be caught.
   - The Log separates the cause that was proven (the timeout mismatch) from the one still unknown (why Ghostty stopped reading), and names the evidence needed to find it, as the Spec required.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - Nothing tests the drag-cancellation half of the release budget (`presenter.go:140`). Removing that `WithTimeout` would leave `Release(context.Background())` unbounded while a drag cancellation is pending, and no test would fail.
   - `WriteFailure.Op` is a free-form string, and a zero value prints `terminal:  write accepted`. There are only two construction sites (`presenter.go:212` keyed, `transport.go:135` positional); they should use the same keyed style.

5. **Test coverage notes.** Every Done-when clause is covered:
   - The transient-stall case is tested at both the ttyio and Presenter level.
   - Persistent stall and cancellation are tested with exact prefixes.
   - Cleanup is covered in both states: restored after a transient stall, and reported as failed when the host stays stalled (Presenter), plus the diagnostic text and budget (Console).
   - The stall-trigger uncertainty is recorded in the Log.
   - I relied on the implementer's logged mutation checks rather than repeating them. The tests' timing assertions make those failures structurally certain.

6. **Architecture**
   - ARCH-DRY: pass. There is one budget owner. The duplicated PTY stall harness across the ttyio and terminal tests was already justified by the import cycle, so I'm not raising it again.
   - ARCH-PURE: pass. The change is confined to the IO shell, and the policy is a single constant applied by its consumers.
   - ARCH-PURPOSE: pass. I checked every consumer of the old budget (child input via `InputWriter`, Presenter paint and release, the palette query, `termcmd`'s release), and each now gets its deadline from `WriteTimeout`. The worst-case release is 2 × `WriteTimeout`, and that is documented.

7. **Plan revisions:** none. The plan matches the code.

```findings
findings:
  - id: new
    severity: Minor
    family: budget-owner-untested-branch
    title: |
      Release's drag-cancellation WriteTimeout bound has no regression test
    detail: |
      presenter.go:140 wraps cancelDrag in WithTimeout(ctx, WriteTimeout), but no test runs Release with a pending drag whose child input stalls. Removing the wrap would let Release(context.Background()) from Console or termcmd hang forever, and every test would still pass. The only other budget application in this window, the reset write via write(), is covered by the stall tests.
  - id: new
    severity: Minor
    family: struct-literal-style
    title: |
      WriteFailure is built positionally in transport.go and keyed in presenter.go
    detail: |
      transport.go:135 uses WriteFailure{"child input", accepted, len(p), err} and presenter.go:212 uses keyed fields. Op is a free-form string with no zero-value guard, so keyed literals at both sites would keep a future field reorder or omission from silently printing "terminal:  write accepted".
```

---

## Re-review — 2026-10-02T09:01:53-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 383 — Couch exits on stalled terminal output and fails to restore keyboard modes |
| repo | pair |
| issue file | workshop/issues/000383-couch-output-timeout.md |
| boundary | whole-issue close |
| milestone | — |
| window | b1de974ba1610ff32d633b536b482af13271d8f2..885ee9f0e036d656cd7c4d580e85cf51a3b733f7 |
| command | sdlc close --issue 383 |
| reviewer | claude |
| timestamp | 2026-10-02T09:01:53-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

**Verdict: SHIP.** Both findings from the last round are fixed, and I found nothing new. ttyio no longer adds its own 2s limit, so the caller's context is the only deadline. `terminal.WriteTimeout` is now the single terminal write budget: Presenter and `InputWriter` apply it, and the start-up palette query uses it too. `Console.release` no longer wraps Release in its own 2s timeout. When the reset write doesn't fully reach the host, release returns `ErrModesNotRestored` instead of reporting success. The new tests go through a real PTY and pass: `go test ./cmd/internal/terminal ./cmd/internal/ttyio` reports `ok`. I had to run them outside the sandbox, because the sandbox refuses `pty.Open` ("operation not permitted"). The couchtty package failures I saw were the same sandbox denials on PTY child processes and `/tmp`, not regressions.

1. **Strengths**
   - `ttyio/writer.go:169`: the hidden `WithTimeout(ctx, 2*time.Second)` is gone, and the `Writer` interface comment now states that the caller's context is the only deadline. That fixes the root cause, not just the symptom.
   - `presenter.go:143-146`: a reset write that fails comes back as `ErrModesNotRestored` wrapping the `WriteFailure`, so the accepted-byte count survives. `TestPresenterReleaseReportsModesNotRestoredWhenHostStaysStalled` checks it with `errors.Is`, `errors.As` and the exact 0/N count.
   - The ttyio stall tests check that the right bytes arrived, not just a count. The payload is a repeating pattern, and the test compares the host's bytes against `payload[:n]`. That covers both "no duplicated bytes" and "accurate partial accounting" from the Done-when list.
   - The Console test now reads the deadline the release write actually sees (`terminal_exit_test.go:28-31`) instead of checking the constant, which is the lesson this issue recorded.
   - Round 2 tested the premise of BR-1 by deliberately breaking the code, found the Presenter wrap was a second budget on top of `InputWriter`'s, and removed it. The Revisions entry records why.
2. **Critical:** none.
3. **Important:** none.
4. **Minor:** none new.
5. **Test coverage**
   - Done-when: transient stall checked at the ttyio and Presenter levels. Persistent stall and cancellation checked at the ttyio level. Mode cleanup checked both when the host resumes and when it stays stalled. The Console release budget and its diagnostic are checked.
   - The stall tests depend on timing (thresholds of 2.5s, 2.9–4s and similar). The margins look adequate, but they could be flaky on a heavily loaded CI machine.
6. **Architecture**
   - **ARCH-DRY: pass.** Each budget has one owner. I swept the window for every `WriteFailure{` literal: there are exactly two, at `transport.go:135` and `presenter.go:211`, and both now use field names.
   - **ARCH-PURE: pass.** The changes stay in the I/O layer, and the policy is just a context deadline.
   - **ARCH-PURPOSE: pass.** All three competing limits named in the Problem section are gone. I checked every caller that used to rely on ttyio's hidden limit: termcmd, Console, terminalqualify and the palette query all now have a bound or pass their own context. The Log keeps the original stall trigger marked as unknown and lists the evidence still needed to find it.
7. **Plan revisions:** none beyond the existing Revisions entry, which already matches the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The redundant WithTimeout wrap was removed, and TestPresenterReleaseBoundsDragCancellationWhenChildStalls guards the real owner (InputWriter); it passes when run outside the sandbox.
  - id: BR-2
    disposition: addressed
    note: |
      Both WriteFailure literals in the tree (transport.go:135, presenter.go:211) now use field names.
```
