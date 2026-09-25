# Boundary Review — pair#326 (whole-issue close)

| field | value |
|-------|-------|
| issue | 326 — Right-pane drag selection broken since clickable tabs (#311) |
| repo | pair |
| issue file | workshop/issues/000326-right-pane-drag-select-broken-by-tab-click.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8d7c86a5bf18aea13f67358819fd8e5f75929665..1c20e96dd13695efbf6674518295da2cbd16d32d |
| command | sdlc close --issue 326 |
| reviewer | claude |
| timestamp | 2026-09-24T22:32:17-07:00 |
| verdict | SHIP |

## Review

Round 1 review of pair#326 is done: all read-only inspection commands succeeded, the fix is correct, and I found no Critical or Important issues.

```verdict
verdict: SHIP
confidence: high
```

The change sets `pair term`'s mouse policy back from `AnyMotion` to `ChildRequested` (`cmd/internal/termcmd/presentation.go:72`) and leaves couch on `AnyMotion`. With a plain shell (the shell asks for no mouse tracking), the pane no longer requests mouse reports, so zellij handles drag selection again. That matches the narrowed Spec and the `## Revisions` entry. Strip clicks still work when the active program asks for mouse tracking. The pump sends every left press to `mux.clickStrip` first, whatever the policy (`cmd/internal/termcmd/run.go:549-553`), and under `ChildRequested` the host sends presses whenever the child's tracking is non-zero. The deferred part (a clickable strip over a plain shell) is filed as #327, which is open and points back to #326's Spec/Log for the options. Wheel scrolling also returns to how it was before #311: over a plain shell zellij scrolls natively, and the `RunZellijAction("scroll-*")` path from #116 still covers the tracked-but-unreported case. The only finding is one stale comment.

1. **Strengths**
   - A one-line fix at the cause, not a workaround. The Log explains why couch doesn't have this bug (its child is a zellij client, which always requests mouse tracking), which explains keeping couch on `AnyMotion`.
   - `TestPresentationLeavesMouseOffForAPlainChild` (`presentation_test.go:423`) checks for all three mouse-report modes (1000/1002/1003), not only 1003. It would fail if `AnyMotion` came back, because that policy always emits `\x1b[?1003h` (`presenter.go:286`).
   - README and atlas are updated in the same commit, and the atlas records why the two presenters now use different policies.
   - The scope cut is recorded honestly: a Revisions entry plus follow-up #327, rather than a silent subset.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `cmd/internal/terminal/presenter.go:18-20`: the `AnyMotion` doc comment still lists "pair term's tab strip, #311" as a user of that policy. After this change `pair term` uses `ChildRequested`, so that part of the comment is now wrong. It should mention couch's status row only, and could note that `pair term` uses `ChildRequested` (#326).
   - The Plan marks "Operator smoke test" as done, but the Log has no line recording its result. Add one line (date + outcome) so the first Done-when item has evidence in the issue.

5. **Test coverage**
   - The inverted policy test covers the regression class directly.
   - The strip-click tests call `clickStrip` directly, and `TestPumpStdinOffersLeftPressToStrip` uses a fake mux. Neither depends on the policy, which is right because the routing doesn't either.
   - What happens with the mouse under zellij (selection and copy) can only be checked by the operator's manual test.

6. **Architecture**
   - **ARCH-DRY:** pass. No duplication; the existing policy enum is reused.
   - **ARCH-PURE:** pass. The policy stays a constructor argument, and `desiredParentModes` is pure.
   - **ARCH-PURPOSE:** pass. The purpose was narrowed openly with a Revisions entry; the deferred clickable-strip-over-shell work is a separate extension in #327, not the point of #326 (restoring selection).
   - **ARCH-MOCK:** pass. No new external calls.
   - **ARCH-CONSTRAINTS:** pass. Fewer mouse reports reach the parent than before.
   - **ARCH-SECURE:** N/A. No untrusted input or secrets are touched.
   - **ARCH-ORDER:** pass. The presenter's gesture state machine is unchanged; only the policy input changes.
   - **ARCH-FUNERAL:** N/A. Nothing durable is created.
   - For #327: a parent-side selection would add real state that lasts across events (anchor, extent, copy). Plan it as an explicit state machine inside the presenter's `ViewEvent` model, not as a separate set of flags.

7. **Plan revisions:** none needed; the existing Revisions entry matches the code.

```findings
findings:
  - id: new
    severity: Minor
    family: stale-doc-after-policy-change
    title: |
      AnyMotion doc comment in presenter.go:18-20 still cites pair term's tab strip as a user
    detail: |
      After this change pair term uses ChildRequested; the comment should name couch's status row only (and could note pair term's ChildRequested, #326).
  - id: new
    severity: Minor
    family: verification-evidence-in-log
    title: |
      Operator smoke test is ticked in the Plan but its result is not recorded in the Log
```
