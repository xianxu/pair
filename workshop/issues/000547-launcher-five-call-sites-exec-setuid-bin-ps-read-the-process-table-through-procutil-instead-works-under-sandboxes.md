---
id: 000547
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-14
estimate_hours: 3.59
card_mirror: '2a468a35224412ddcce31958cdfe6960ac8ed1d2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T00:00:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
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

- [x] Implement and verify the native process inspection plan: [plan](../plans/000547-native-process-inspection-plan.md).

## Log

### 2026-10-13

### 2026-10-14 — design

Claimed and started from fresh main. Hub approved sweeping all production ps calls and full flow. Plan above records native argv/executable distinction, per-row uncertainty and snapshot identity fencing (ARCH-DRY, ARCH-ORDER). Prior continuation commit remains on its own branch; untracked pensive preserved. #528 overlap coordinated with pair:2.

## Revisions

- 2026-10-14: enumerated the full no-ps contract beyond the five launcher sites: procutil helpers/unsupported fallback and diagnostic runtime inventory are in scope, approved by the hub.

- Plan-quality round 1 requested named risky functions and mechanical test strategies (PQ-1), plus explicit non-goals (PQ-2). Appended concrete interfaces, strategy table, composition seams and scope boundaries to the plan. Baseline seatbelt probe: TestCommand skips because ps is unavailable, while TestTableMatchesIdentityAndParent passes; replacement must remove that skip on supported OSes.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only; calibration is marked stale and provisional. Native adapters reuse x/sys/unix and existing procfs/sysctl patterns; no greenfield library is needed. Thorough reviewed plan applies 0.2 design discount to implementation primitives, implementation values use v3.1's 0.4 scale, familiarity 1.0. Two API items represent Darwin and Linux adapters plus conformance; two refactors represent launcher and diagnostic/composition migrations. Spec item accounts for the actual cross-consumer shaping rather than discounting it away.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.70 impl=0.08
item: api-integration design=0.20 impl=0.60
item: api-integration design=0.20 impl=0.60
item: cross-cutting-refactor design=0.12 impl=0.20
item: cross-cutting-refactor design=0.12 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 3.59
```

### 2026-10-14 — implementation and focused verification

- Replaced all nine production ps executions with procutil native metadata/argv observations; AST guard failed on the nine original calls and now passes. Launcher ancestry uses one table and cycle detection; server projection retains snapshot identity and refuses unreadable possible servers. Diagnostic inventory distinguishes full executable identity from argv and short names. Both cleanup proof paths share an injected Inspection; the public apply fixture verifies a legacy writer blocks both, then removing it permits collection.
- Procutil native parser/snapshot implementation delegated as bounded work. Initial API tests failed before implementation. Additional red/green tests enforce Darwin argc allocation bound and reject numeric PIDs that would truncate into a live PID at Darwin's int32 syscall boundary.
- Full targeted package runs passed: procutil, launcher, diagnosticlog, gccmd, gcruntime. Artifact source inventory test passed. A real argv-spoofed helper was found, classified and authorized through native launcher observations under sandbox-exec (version 1)(allow default), with all six selected tests passing and no skips.
- Estimate advisory allocation: Darwin API item includes shared snapshot/error/fencing fake tests and seatbelt conformance; Linux API item includes portable procfs fixture/cross-build/native CI verification. Consumer refactors include targeted race/full-suite execution and fresh-main conflict integration; review item covers the single close boundary. Spec hours describe completed shaping, not additional unresolved design.

- Acceptance limitation approved by ops:0 (receipt e1e6c492-6689-4734-a050-582a546a575f): Linux: portable procfs fixture and cross-build pass; native Linux procutil is untested (pair has no Linux test runner, and pair runs on macOS in practice). No new runner and no colima. Actual workflow inspection found Ubuntu merge-check only runs duplicate-ID checks, plus local-gates-guard; no native Linux test execution is claimed.
- make -k build passed. Targeted race tests in procutil, launcher and diagnosticlog passed. Fresh Darwin binary passed TestNativeProcessInspectionWithoutPS, TestCommand, TestIdentity, TestTableMatchesIdentityAndParent, TestDarwinArgCountBound and TestPIDRejectsTruncationToLiveProcess under seatbelt, without skips. Full suite is running before close.

- First full make -k test finished with one failure: embedded-runtime smoke still faked ps ancestry, so native Table correctly detected the developer's Zellij parent. Replaced that fake with a bounded, test-only reparented command helper (tests/without-ancestors.py); the real native ancestry now sees init as parent. Targeted make -k test-pair-embedded-runtime passed. Direct script execution retains live Couch environment and is not a valid substitute for the make target's environment sanitization.
- Extended the fixture sweep to cmd/pair-go/main_test.go: TestPublicPairCommandFamiliesIgnoreCouchStore failed in all launch families before switching its ancestry fixture to the same bounded reparented helper; it now passes. Its test-owned sidecar cleanup uses procutil Snapshot, so sandbox verification does not depend on a hidden ps call. Remaining test ps uses are negative assertions or an independent niceness oracle.
- Added a red/green runtime-inventory regression for Linux's unlinked-executable suffix: /proc/PID/exe may end in " (deleted)" after a binary replacement, but the live generation remains a possible diagnostic writer. Normalize that marker for executable-name classification; never silently omit the writer. The inventory tests now pass.
- Additional real TestSessionQuiescenceLive seatbelt probe exposed persistent possible-zellij EINVAL after deletion (PID 44367): investigate exited/unreaped native process status instead of treating argv failure as absence. Paused close for a native terminal-state regression and rerun of the live acceptance probe.
- Confirmed the live failure with a controlled exited-but-unreaped child: Darwin P_stat=5 (SZOMB), argv EINVAL. Red regression retained it as unknown; fix adds native Process.Zombie (Darwin 5, Linux Z/X/x), keeps Table rows and excludes only confirmed terminal rows from Snapshot. Unreadable live rows remain errors. Procutil race, Linux cross-build and five seatbelt conformance repetitions passed. Fresh TestSessionQuiescenceLive under seatbelt passed in 1.93s after the fix.
- Full make -k test rerun passed (all shell/editor and Go packages). Repeating the final full suite after the terminal-state delta; no further production edits planned.

- Final verification after b541927de: make -k test passed with exit 0 (/tmp/pair547-test-verified.log), including all shell/editor tests and go test ./... -count=1. make -k build passed (/tmp/pair547-build-verified.log). Fresh main merged with only #549 issue documentation, no code conflicts. Ready for the single SDLC close review.
