---
id: 000265
status: working
deps: []
github_issue:
created: 2026-09-15
updated: 2026-09-16
estimate_hours:
started: 2026-09-16T09:04:38-07:00
---

# couch crashes on switch to brain: no admitted endpoint

## Problem

Switching couch to the `brain` repo crashes couch.

Repro (from operator report):

- Trigger: switch to `brain` (coding harness not remembered — not the usual harness variant couch expects).
- Observed: `brain` appears in the tab bar, but the main viewport is fully blank. No content is rendered.
- Then: pressing a key (possibly `Esc`, uncertain) prints a raw escape sequence at the top-left corner instead of being handled.
- Then: couch exits with:

  ```
  couch: terminal: terminal: no admitted endpoint
  ```

Source of the inner error is `cmd/internal/terminal/presenter.go:494` inside `Presenter.Input`:

```go
if v.State != Ready || v.Admitted == "" || p.selected == nil {
    return errors.New("terminal: no admitted endpoint")
}
```

That path is the non-mouse input path — any key event when the presenter has no admitted endpoint. The double `terminal:` prefix in the fatal log indicates the error is wrapped once on the way out of the terminal package and again at the couch top-level.

Current behavior turns a blank/unadmitted pane into a fatal couch exit on the next keystroke. Expected behavior: the pane should either render (admit the endpoint) or degrade without taking down the whole couch; unhandled input on an unadmitted/blank view should be dropped or surface a non-fatal diagnostic, not crash.

Open questions to resolve during fix: what harness `brain` is actually running (and why its endpoint never reaches `Ready`/`Admitted`), why the viewport stays blank while the tab exists, and why the key event leaks as a raw escape sequence instead of being consumed.

## Spec

- Preserve the report verbatim in `Problem` (switch-to-brain → blank viewport → escape sequence leak → crash with `terminal: no admitted endpoint`).
- Trace the switch path for `brain`: endpoint creation/selection, `Presenter.Select` → `View` transition, and what leaves `v.Admitted == ""` / `v.State != Ready` / `p.selected == nil` after the switch. Identify why `brain`'s harness produces an endpoint that never admits.
- Make `Presenter.Input` (and any other `no admitted endpoint` throw site) non-fatal for the couch process: unadmitted input should be ignored or return a handled error without exiting couch. If the error is still surfaced, it must be non-fatal and actionable (which endpoint/view state, which harness).
- Fix the blank-viewport side: either ensure the `brain` endpoint is admitted and paints, or show an explicit empty/error state instead of a blank screen with a leaked escape sequence.
- Keep the fix in the terminal/presenter seam; do not add a `brain`-only workaround in the viewer layer.

## Done when

- Switching to `brain` no longer crashes couch; a key press (including `Esc`) on a blank/unadmitted pane does not exit the process.
- `brain` either renders its content after switch, or shows a coherent empty/error placeholder instead of a blank screen with raw escape sequences leaking to the top-left.
- The `terminal: no admitted endpoint` condition is handled without a fatal couch exit and, if logged, includes enough context to diagnose the endpoint/view state.
- Regression coverage exists for the input-on-unadmitted-endpoint path (key event when `Admitted == ""` / not `Ready` / `selected == nil` does not panic/exit).

## Estimate

```estimate
# refined estimate pending plan approval
```

## Plan

- [ ] Reproduce / narrow: log the `View` state (`State`, `Admitted`, `Selected`) after switching to `brain`; confirm which guard fails at `presenter.go:494`.
- [ ] Identify why the `brain` endpoint never admits (harness/mode mismatch, attach failure, or selection without publication) and why paint is blank.
- [ ] Harden `Presenter.Input` (and callers) so unadmitted input is non-fatal; decide on drop vs. diagnostic.
- [ ] Fix or surface the blank-viewport case for unadmitted endpoints.
- [ ] Add test for key input on unadmitted endpoint; verify switch-to-brain no longer crashes.

## Log

### 2026-09-15

- Recorded from operator report: switch to `brain` → tab appears but viewport blank → key/Esc leaks escape sequence at top-left → `couch: terminal: terminal: no admitted endpoint` and exit. Source pinned to `cmd/internal/terminal/presenter.go:494`. Issue created as `000265`.
