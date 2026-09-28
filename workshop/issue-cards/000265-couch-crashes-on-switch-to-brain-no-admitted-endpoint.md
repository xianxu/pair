---
id: '000265'
status: done
started: 2026-09-16T09:04:38-07:00
created: 2026-09-15
updated: 2026-09-16
estimate_hours: 1.70
actual_hours: 3.66
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
