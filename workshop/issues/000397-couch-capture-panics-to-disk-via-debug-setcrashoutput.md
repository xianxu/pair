---
id: 000397
status: working
deps: []
github_issue:
created: 2026-10-06
updated: 2026-10-06
estimate_hours:
card_mirror: '02d1658cbfe90831a52fc258ca524c35fd792187' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-06T16:09:24-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "9b36eef0", done: "b01aa013"}
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

Design (2026-10-06):

- **Who captures (ARCH-PURPOSE):** only the console-owning couch: the process that
  holds the singleton lease and draws the console, which is the terminal that gets
  redrawn over. CLI invocations (`couch --list` etc.) already print panics to a
  readable stderr, and capturing those too would make a file per agent query.
- **Where:** `<couch store dir>/crash/` (`crashreport.Dir(namespace)`), one store
  per singleton selection, so isolated roots get their own. The store dir already
  holds mixed metadata files. `pair gc`'s inventory excludes registered stores, so
  the files neither block collection nor go unclassified.
- **Names:** `<UTC yyyymmddThhmmssZ>-<pid>.log` while unreported, renamed to
  `….crash` once reported. The runtime writes into the open `.log`.
- **Lifecycle (ARCH-ORDER):** after the singleton lease is taken and before the
  console runs:
  1. Scan the crash dir; every existing file belongs to a previous incarnation (the
     lease proves no other owner is live).
  2. Classify, as a pure function over (name, size) (ARCH-PURE):
     - a non-empty `.log` is a crash → control notice
       `previous couch crashed — see <path>`, renamed to `.crash`;
     - an empty `.log` is an abrupt end without a panic (SIGKILL, power loss, memory
       kill) → control notice `previous couch ended abruptly (no panic recorded)`,
       then deleted;
     - a `.crash` was already reported → left for retention.
  3. Open our own `.log` (O_EXCL, 0600) and call `debug.SetCrashOutput(f, {})`.
  4. On clean exit: disable crash output, close, and delete the file if empty.
  A failure anywhere becomes a status-row notice; crash capture never stops couch.
- **Retention (ARCH-FUNERAL), a dedicated sweep:** the diagnosticlog writer protocol
  assumes a live, cooperating writer and would keep a runtime-written file forever.
  Instead, `gcruntime` Preview/Apply walk each registered store's `crash/`, matching
  only the exact name grammar, regular files only (Lstat), and skipping a `.log`
  whose pid is alive. Age is decided by `diagnosticlog.DecideSegment` (the same
  period as other diagnostics, ARCH-DRY), and rows go into `Report.Diagnostics` so
  `pair gc` renders them unchanged.
- **Bounds (ARCH-CONSTRAINTS):** one file per console run; startup scans at most a
  fixed number of entries; gc honours `limit`.

- [ ] crashreport package: name grammar, pure classification, Install/Close, gc
      sweep decision; tests including a re-exec child that panics (file holds
      `panic:` + goroutine stack, and stderr still gets it)
- [ ] couchcmd wiring after the singleton lease, notices on the status row
- [ ] gcruntime sweep over registered stores' crash dirs, with preview/apply tests
- [ ] atlas/couch.md (next to COUCH_TRACE), manifest source classification

## Log

### 2026-10-06

- Filed from a brain session after the 13:56–14:02 couch crash (see Problem). Related finding from the same investigation, to be filed separately: startup reattach is serial (~17 threads × 5–13 s ≈ 1m45s when throttled). It wasn't the `go build` in the `couch` shell function (≈2 s of cache writes).
