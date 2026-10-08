# Boundary Review — pair#409 (whole-issue close)

| field | value |
|-------|-------|
| issue | 409 — couch: full-screen repaint per frame overruns a backgrounded terminal, and the resulting exit is silent |
| repo | pair |
| issue file | workshop/issues/000409-couch-a-single-5s-stall-in-the-outer-terminal-ends-the-whole-session.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4ba09dce8b10baf3b0351075dc69bbc5d1d5cc43..765deca0b68d2b31629538b800695a0198f0fc76 |
| command | sdlc close --issue 409 |
| reviewer | claude |
| timestamp | 2026-10-07T23:27:36-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The window delivers both milestones as the Spec describes. M1 records the reason for a terminal-stall exit through `crashreport.RecordExit`, and the next start reports it once as a separate `Exited` kind. The reason comes from teardown's own classification, so a shutdown that only cancelled a paint records nothing. M2 adds the cheaper path #262 described: when only unwrapped rows change, the history path repaints just those rows. A seeded test checks it against the full rebuild under the xterm oracle, and under the native zellij oracle when that is switched on. Both open findings are dealt with: plan Revision (b) records the as-built wording and that 1.1–1.4 are done, and the atlas bullet's indentation is now continuous. I ran the tests for this diff locally. `TestHistoryRowDiffEqualsFullRebuild` reported 48 row-diffed and 32 refused across 80 seeds. `TestChangedPlainRowsGate` passed. The spinner tick wrote 134 bytes against 10,440 for a full rebuild. `TestExitReason` and the couchcmd exit-reason test passed. Two other tests failed only because of the sandbox: `TestAStalledParentReadsAsTheTerminalStopping` (pty, "operation not permitted") and a couchcmd test that writes to `/tmp`; neither failure comes from this diff. Nothing blocks SHIP.

1. **Strengths**
   - `transport.go:897` `ExitReason`: one pure function owns the wording. It finds the `WriteFailure` through wrapped or joined errors and only claims what it knows ("waited up to"), so it does not overclaim the cause.
   - `crashreport.Classify` stays pure. `Head` is read once at the IO edge (`list`/`readHead`). The file is renamed before it is reported, as panics already are, so it cannot be reported twice.
   - `history_rowdiff_test.go`: the generative test compares against the full rebuild. It does not restate the implementation, and it requires both the diff and the refusal branches to run. That is a real oracle, not a sample of one.
   - `changedPlainRows` refuses anything it cannot prove safe: a different endpoint, geometry or alt-screen state, a wrong cell count, or any wrapped row.
   - `Console.exitFailure` is set inside the one place teardown classifies the failure, so there is only one classification and `TerminalFailure` reads it.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - **Alt-screen not tested on the diff path.** The diff path also runs for frames that stay in the alt screen (`history_render.go:556`), but the generative test only builds primary-screen frames. One seed group that wraps the base in `?1049h` would cover it.
   - **No longer self-heals after an outside clear.** Before, every dirty frame repainted all visible rows. Now, if something outside couch clears the outer terminal (for example Cmd-K in the host), only the changed rows come back until a reset or a wrapped-row change. This is acceptable if it is intended; it is worth one line in `atlas/terminal.md`.
   - **Plan checkboxes still unticked.** Tasks 1.1–2.6 in the plan file are still `- [ ]`, though Revision (b) and the issue's own Plan say they are done. Cosmetic.

5. **Test coverage notes**
   - Bugs this diff could ship are covered: the row-diff equality (generative, two oracles), the gate's width (`TestChangedPlainRowsGate`), the byte reduction, `ExitReason` across wrapped, joined and child-input errors, `Exited` against `Crashed` across runs with no second report, and a stop that only cancelled a paint recording nothing (`terminal_exit_test.go:201`).
   - The native zellij oracle only runs with `PAIR_TERMINAL_NATIVE=1`, and Revision (b) notes the xterm oracle skips silently without `npm ci`. Run once with `PAIR_TERMINAL_ORACLE=1` before merging.

6. **Architectural notes (one line per principle)**
   - **ARCH-DRY: pass.** The diff Emit reuses `cup`, `cells`, `blankTail` and `cursorEpilogue`.
   - **ARCH-PURE: pass.** `Classify`, `Summary`, `ExitReason` and `changedPlainRows` are pure. The IO is limited to `RecordExit`, `readHead` and the couchcmd hook.
   - **ARCH-PURPOSE: pass.** Both halves of the issue (the silent exit, and the repaint volume that causes stalls) are delivered. Neither is pushed to a follow-up.
   - **ARCH-MOCK: N/A.** No new external dependency. The terminal oracles are the existing conformance harness.
   - **ARCH-CONSTRAINTS: pass.** `headBytes` caps each read at 256 bytes and `MaxEntries` bounds the scan. The idle tick drops from about 10 KB to about 134 B, as measured.
   - **ARCH-SECURE: pass.** The crash file is untrusted input. Its head is capped, the reason is cut at the first newline, and an unreadable head classifies as a crash rather than being hidden.
   - **ARCH-ORDER: pass.** The reason is read only after `Run` returns. A panic during the exit lands in the same file, and the summary names that file (BR-2 lineage).
   - **ARCH-FUNERAL: pass.** No new file family. `Exited` files become `.crash` and `pair gc`'s existing retention sweep removes them.

7. **Plan revision recommendations:** none needed. Revision (b) already matches the code. Optionally tick the task boxes.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      Revision (b) records the as-built ExitReason wording ("a write waited up to 5s") and that 1.1-1.4 are done; the issue Plan ticks M1/M2.
  - id: BR-5
    disposition: addressed
    note: |
      atlas/couch.md recorded-exit bullet now keeps a two-space continuation indent on every line.
findings:
  - id: new
    severity: Minor
    family: rowdiff-coverage-alt-screen
    title: |
      Row-diff fast path also runs for frames that stay in the alt screen, but the generative test only builds primary-screen frames
    detail: |
      history_render.go:556 gates on AltScreen equality, not on primary-only. Add alt-screen base frames to TestHistoryRowDiffEqualsFullRebuild.
  - id: new
    severity: Minor
    family: rowdiff-no-self-heal
    title: |
      Row diff no longer repaints the whole visible screen on a dirty frame, so an outside clear of the host stays blank except for changed rows
    detail: |
      Before, every non-reset dirty frame rebuilt all lower rows. If this is intended, note it in atlas/terminal.md next to changedPlainRows.
```
