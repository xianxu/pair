---
id: 000547
status: open
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: 'feb85bd2aef684f37596be065080e40f75c3c9e9' # card fields mirrored from issue-cards; edit via sdlc
---

# launcher: five call sites exec setuid /bin/ps; read the process table through procutil instead (works under sandboxes)

## Problem

pair#528 M1's boundary review looped ten rounds because its sandbox couldn't run `/bin/ps`. ariadne:3 measured why (ariadne#364, 2026-10-14): on macOS `/bin/ps` is **setuid root**, and seatbelt refuses to exec setuid binaries even under `(allow default)`; non-setuid `pgrep`, which reads the process table in-process, works under the same profile. pair still shells out to `ps` in five places:
- `cmd/internal/launcher/osruntime.go:903` (`ps -ww -A -o args=`), `:1113` (`ps -o comm= -p`), `:1121` (`ps -o ppid= -p`)
- `cmd/internal/launcher/session_quiescence.go:129` and `session_servers.go:147` (`ps -axo pid=,command=`)

So these paths fail under any sandboxed reviewer or agent, and they duplicate `procutil.Table`, which already reads every process in one `kern.proc.all` sysctl (ARCH-DRY).

## Spec

Route all five through `procutil` (extend it for argv/command where it lacks it: `kern.procargs2` on darwin, `/proc/<pid>/cmdline` on linux), with one snapshot per decision where several facts must describe the same moment. Keep behaviour identical; no `ps` exec remains in non-test code (add a test or lint that fails on `exec.Command("ps"`).

## Done when

- No non-test code execs `ps`; the five call sites use procutil, with tests on both platforms' table readers.
- The process-dependent tests pass inside a seatbelt profile that denies setuid exec (e.g. `sandbox-exec` with `(allow default)`), demonstrating reviewer-sandbox compatibility.

## Plan

- [ ]

## Log

### 2026-10-13
