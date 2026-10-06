---
id: 000397
status: open
deps: []
github_issue:
created: 2026-10-06
updated: 2026-10-06
estimate_hours:
card_mirror: 'cea61f84ab7ab6a87597617f513cdf40781269af' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: capture panics to disk via debug.SetCrashOutput

## Problem

Couch crashed on 2026-10-06 sometime between 13:56 and 14:02 (battery at 2%, CPU throttled) and left no record of why. A Go panic writes its stack trace only to the controlling terminal's stderr. Relaunching `couch` in the same tab redraws over it. Go doesn't produce a macOS `.ips` report, the system log showed no memory kill or sleep, and pair's per-thread logs (`wrap-events`, `zellij-actions`) record only that activity stopped. The root cause is unrecoverable. The next crash will be the same unless couch captures its own panic.

## Spec

- At process start, couch calls `runtime/debug.SetCrashOutput(f, ...)` (Go ≥1.23; the module is on 1.26) with `f` a crash file under pair's data dir (e.g. `~/.local/share/pair/couch/crash/<timestamp>-<pid>.log`). Fatal panics and runtime fatal errors are then written there **in addition to** stderr.
- The file is opened eagerly (SetCrashOutput needs an open fd). To avoid leaving an empty file per launch, either delete it on clean exit or use one append-only file per day. Pick whichever retention/GC handles more simply.
- Retention: register the crash files with the existing diagnostics retention (`diagnosticlog` / `pair gc`) so they age out like other diagnostics, not forever.
- On the next startup, if a crash file from a previous incarnation exists and hasn't been acknowledged, show a single status-row notice ("previous couch crashed — see <path>"). Without it, the file sits unread.
- Consider the same for `pair` (the per-thread wrapper) if it shares the entry-point plumbing. Couch is the priority.

## Done when

- [ ] A test-only forced panic in couch (env-gated or a test binary) leaves a crash file containing `panic:` and a goroutine stack at the expected path, and still prints to stderr.
- [ ] A clean exit leaves no stray empty crash file (or the documented daily file stays bounded).
- [ ] Crash files are covered by `pair gc` retention.
- [ ] The next startup surfaces the previous crash once.
- [ ] `atlas/couch.md` names the crash-file path next to the `COUCH_TRACE` section.

## Plan

- [ ]

## Log

### 2026-10-06

- Filed from a brain session after the 13:56–14:02 couch crash (see Problem). Related finding from the same investigation, to be filed separately: startup reattach is serial (~17 threads × 5–13 s ≈ 1m45s when throttled). It wasn't the `go build` in the `couch` shell function (≈2 s of cache writes).
