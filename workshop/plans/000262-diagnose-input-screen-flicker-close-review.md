# Boundary Review — pair#262 (whole-issue close)

| field | value |
|-------|-------|
| issue | 262 — Screen flicker: the compositor re-emits global terminal state every frame (#255) |
| repo | pair |
| issue file | workshop/issues/000262-diagnose-input-screen-flicker.md |
| boundary | whole-issue close |
| milestone | — |
| window | a1b0a7d8e53f878b1ce6b543c2f6e4fead4c970b..27428de3a67d9a0a0e2f4d0df6e608ac6d1e2b6f |
| command | sdlc close --issue 262 |
| reviewer | claude |
| timestamp | 2026-09-17T20:36:29-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The code change is correct and well tested. `Render` and `HistoryRender.Emit` wrap every frame in exactly one DECSET 2026 bracket. The alt-screen packet stays whole inside the bracket, and no-op frames write nothing. `parentReleaseControls` closes a bracket that a failed write left open. I reverted that release fix in a scratch copy, and `TestPresenterReleaseClosesSyncAfterAnyCutWrite` failed at cut 8 in all three layouts, so the test really guards the fix. The `cmd/internal/terminal` package passes at HEAD, both plain and with `PAIR_TERMINAL_ORACLE=1` (the xterm oracle suites ran and passed). BR-4 is fixed: with the fix reverted, its regression test fails. What remains is all Minor and doc-level:
- The M2 classification rests on two claims about the code that are false, and they were copied into the atlas and a code comment.
- One checkbox in the plan is stale.
- There is still a timing gap where `sync_hold.py` can report "NOT honoured" when it simply didn't watch long enough.

None of these blocks the gate. The atlas one is worth fixing, because the atlas is supposed to stay current.

**1. Strengths**
- **Bracket in the pure renderers** (`render.go:61,112`, `history_render.go:307,409`). The bracket is added inside the pure renderers, and the constants plus `cursorEpilogue` are shared, so there's one site for the cursor-epilogue decision (ARCH-PURE, ARCH-DRY).
- **Alt-screen packet stays whole.** `e.add(syncBegin)` goes before `e.packet(...)`, so the begin marker is flushed as its own write. The `?1049h`/`l` packet still matches exactly in `Presenter.write:196`.
- **Failure-sweep test.** It measures the write layout with a probe, cuts each write, and checks the whole parent stream with a small state machine (`assertStreamBracketed`). The test gets its failure orderings from an injection seam (`ttyio.WriteStep`) rather than a sample of one (ARCH-ORDER).
- **Sole-writer premise holds.** No code in `couchtty` or `termcmd` writes to the parent outside the presenter.
- **The Revisions entry is honest.** It re-scopes Done-when bullets 2 and 5 with stated reasons instead of quietly waiving them.

**2. Critical findings**
None.

**3. Important findings**
None.

**4. Minor findings**
- **The M2 classification rests on two false code claims** (family `unverified-existing-code-claim`, 2nd finding this gate). Details are in the findings block. The outcome (keep the re-asserts) doesn't change, but the reasons written in `atlas/terminal.md`, in the Log table, and in the comment at `history_render.go:316-317` are wrong.
- **A plan checkbox is stale.** `plan:299` "Close the milestone" is still unticked, although M1 was closed in `5961cb1a` (family `plan-record-stale-after-log`, 2nd finding).
- **`sync_hold.py` can still report "NOT honoured" for a window that was too short** (family `instrument-failure-encoded-as-verdict`, 2nd finding).
- **Misplaced Log line.** `sdlc` placed the "closed M1" line at issue line 816, inside the unrelated "M3's premise is in question" section. Move it next to the M1 entries.
- **Two sentences run together in the atlas.** In `atlas/terminal.md`, the new paragraph ends on the same line as the older sentence "Product code retains shortcut and notification policy…", so that sentence now reads as part of the per-frame rationale. Add a paragraph break.
- **Imprecise atlas wording.** `atlas/terminal.md` says each frame ends "showing" the cursor. It only does that when `Cursor.Visible` is true.
- **A zellij server can outlive its temp dir** (`sync_hold.py`, ARCH-FUNERAL). If `zellij kill-session` fails, `process.kill()` stops only the client. The daemonized server then outlives the `TemporaryDirectory` that held its socket. Low stakes for a discovery script.

**5. Test coverage notes**
- The bracket contract is pinned for:
  - `Render`: first paint, one-cell change, cursor-only change, and no-op.
  - `Emit`: reset, history append, one-cell change, idle, alt enter and leave, and a frame spanning several chunks.
  - The presenter's full stream, including release.
- Failure positions are covered for all three write layouts, and the test goes red without the fix.
- The native zellij oracles were skipped in my run (they need `PAIR_TERMINAL_NATIVE=1`). The Log records them passing.
- The `ptychild`, `hostty` and `couchtty` failures in my run were pty and `/tmp` permission errors from this environment. Those packages only changed comments.

**6. Architecture notes**
- **ARCH-DRY, ARCH-PURE, ARCH-ORDER, ARCH-SECURE: pass.** SECURE is N/A: the change parses no untrusted input and handles no secrets.
- **ARCH-PURPOSE: pass.** Every frame path (`Panel` → `Render`, child → `Emit`) is bracketed. Cursor-style fidelity is a separate issue and is correctly split out to pair#283.
- **ARCH-MOCK: pass with a note.** `sync_hold.py` is a manual conformance check with no cadence. That's acceptable, because correctness doesn't depend on 2026 being honoured.
- **ARCH-CONSTRAINTS: pass.** Sync is open only inside one synchronous call bounded by `WriteTimeout`. Between a failure and release, the terminal's own sync timeout bounds the freeze, and that is documented.
- **ARCH-FUNERAL: pass**, apart from the zellij-server note above.
- **For the future row diff:** the atlas tells whoever builds it to check "one writer" before replacing the re-asserts with deltas. Correct that rationale first (finding 1), so it doesn't mislead.

**7. Plan revision recommendations**
- Tick `plan:299`, or add a Revisions line pointing at `5961cb1a` and the closed-M1 Log line.
- Correct the M2 table rows for `ESC[r` and the writer count in the issue Log with a short correction note, not a rewrite.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      main() catches instrument exceptions and exits 2 (sync_hold.py:139-146). InstrumentFailureTest fails with the try/except reverted in a scratch copy and passes at HEAD (4/4). A remaining timing sub-case is raised separately.
findings:
  - id: new
    severity: Minor
    family: unverified-existing-code-claim
    title: |
      M2 classification rests on two false code claims, repeated in atlas/terminal.md, the issue Log table and history_render.go:316-317
    detail: |
      (1) "its other writes (mode delta, effects, Copy, release) touch none of it": release writes ?6l, ESC[r, ?7h, SGR0, the OSC8 close, ESC[0 q and ?25h (presenter.go:276-277). The conclusion survives only because release is the presenter's last write, and the prose should say so. (2) "the region reset is also functional here, since history pushes set 1;2r below": Emit's push sets 1;2r at :331 and resets it itself at :364, so the preamble's ESC[r at :318 is only a convergent re-assert. Keep stays the right outcome. This is the 2nd finding in this family this gate, after PQ-1 and BR-2, so fix the class, not these two sites. Rule: a statement that quantifies over code ("none of the N writes do X", "Y is functional because Z") cites evidence for each member in the same commit: each write site's payload line, or the byte that depends on the sequence. Add the rule to workshop/lessons.md, then re-check every row of the M2 table against it.
  - id: new
    severity: Minor
    family: plan-record-stale-after-log
    title: |
      Plan Task 5 "Close the milestone" is still unticked (plan:299), though M1 closed in 5961cb1a with a closed-M1 Log line
    detail: |
      This is the 2nd finding in this family (BR-3 was the first), so fix the rule, not this instance. Rule: the commit that crosses a boundary reconciles the durable plan with the Log. Every plan row whose evidence is in the Log gets ticked, or a Revisions line says why not. Check it mechanically with grep '- \[ \]' over the plan before milestone-close or close. Better still, don't put an sdlc verb in a plan as a checkbox: the verb records itself (trailer plus Log line).
  - id: new
    severity: Minor
    family: instrument-failure-encoded-as-verdict
    title: |
      sync_hold.py still reports "NOT honoured" when the observation window was too short to see the marker
    detail: |
      The read deadline is a fixed sleep of SETTLE+HOLD+1.0 from Popen (:104), not a wait relative to t_end. If zellij takes about 1.0s to start the pane, t_end exists but the post-close marker has not been drained yet. marker_after is then None, and verdict() returns 1 (:132-134). This is the 2nd finding in this family (after BR-4, which fixed the crash case), so fix the rule. Rule: a verdict of 0 or 1 needs an observation window that could have seen the other outcome; a missing observation the window can't account for is 2. Fix: poll for t_end, then wait a fixed grace period past it before reading. Also make test_a_marker_that_never_arrives_is_not_honoured depend on the window having covered that grace period.
```
