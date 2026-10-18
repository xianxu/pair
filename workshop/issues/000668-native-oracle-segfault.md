---
id: 000668
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '59e4c789fb2773f644782c8539fe12202423ce03' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T14:16:34-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Native terminal oracle subprocess crashes with SIGSEGV in macOS conformance

## Problem

While landing #659, macOS CI run 39111981159/job 117629869893 failed TestNativeConsoleWrapperZellij/wrapped-zellij at terminal_native_test.go:345: independent xterm oracle: signal: segmentation fault (empty subprocess output). First observed 2026-10-18 08:36:46 UTC, branch head 46e0bd0379ccf8ccabb972fa3b62aea2a9a36871; tested PR merge fb2be7a7ff1a4779bdf61884229748d808e2ae1d. Full failed-job log retained at /tmp/pair659-ci-39111981159.log; public run is https://github.com/xianxu/pair/actions/runs/39111981159/job/117629869893 .

## Spec

Identify why the independent Node xterm oracle process crashes and add a regression for the measured cause, preserving native terminal frame assertions. assertNativeOracle invokes node tests/terminal-oracle/driver.cjs with a JSON frame request via stdin; CombinedOutput returned a segmentation fault and no output. Do not retry inside the test or weaken the oracle assertion.

This is distinct from #660's missing session during reattach and #603's shell-actor discovery timeout. Direct baseline passed in 3.11s; wrapped receipts retained one BOOT=7195:e18042497f7e9a471cb2cadf and STATE with that same identity advancing 1 to 2 before the oracle crash. The wrapped subtest failed in 2.34s. The CI run predates #660 diagnostics (PR429 landed 15:19 UTC); no new socket/phase facts were available. Fixture and oracle files have no #659 diff against its integrated main. Other heads passed this conformance family, but this specific crash has one observation, not a measured flake rate.

## Done when

- Retained diagnostics identify the crashing subprocess/runtime and causal input or environment.
- A deterministic regression protects the measured cause without relaxing terminal checks or increasing deadlines.
- Native conformance passes with before/after failure counts reported.

## Plan

Classification stage, pending hub approval before implementation:

- [ ] Compare official Node22.23.3 darwin-arm64 against the installed Node26.0.0 with no global installation or configuration change. Download the CI version only into one invocation-owned TMPDIR directory, verify the archive against the official SHA256 manifest, and remove that directory after reporting. This runtime download is part of the approval request.
- [ ] Bound the headless probe to120 sequential child invocations and60s execution time:30 bare-start controls,27 complete driver requests and3 malformed-JSON controls per runtime. The one private runtime download has its own120s bound. Rotate empty, ordinary110x28, and Unicode/control-sequence frame requests; retain each exact generated input, runtime identity, return code/signal, and separate stdout/stderr. Add malformed-JSON controls to distinguish ordinary JavaScript rejection from fatal process termination. No native Zellij/PTY invocation, production data, concurrency load, retry-on-failure, or timeout changes.
- [ ] If a child crashes, stop the matrix at the first crash, preserve its exact request and process facts, and reduce that cause before choosing a fix. A bare-start failure would exclude the frame/xterm path; a driver-only failure still needs isolation of import/write/snapshot/dispose stages. Count outcomes separately by runtime and control; do not turn a later pass into a causal explanation.
- [ ] If the bounded probe stays green, report the remaining uncertainty and propose a separate diagnostic implementation stage before code: one shared subprocess boundary for all five driver.cjs snapshot consumers, exact failure-request preservation, runtime provenance, separated streams and bounded phase breadcrumbs. No speculative Node upgrade/pin, oracle skip, assertion weakening or retry. The different stream.cjs performance protocol is outside that proposed snapshot boundary.
- [ ] Once a cause and fix plan are approved, add a deterministic regression at the measured seam, retain the independent terminal assertions, and obtain a fresh native lane grant before native conformance. Close/merge only with the corresponding acceptance evidence or an explicitly approved diagnostic-stage landing/park arrangement.



## Log

- 2026-10-18: Filed unassigned under the fleet flake rule after distinguishing it from #660; #659 will rerun only this failed CI job once on the unchanged head and stop/report on a second failure. Its full integration suite passed exit 0 in 11m3s before this CI failure. Historical root cause remains unknown.

## Revisions

### 2026-10-18 — classify before selecting a fix

The hub dispatched classification first (Node, fixture or frame), with a plan before code. The historical exact request and crash stack were not retained, so the first stage is a bounded runtime/control comparison. Original causal regression and native acceptance requirements remain open; a passing sample cannot close them. Any diagnostic-stage landing needs an explicit scope/acceptance arrangement from the hub.

### Architectural lenses for the classification stage

ARCH-DRY: reuse the pinned driver/package lock and existing complete renderer fixtures; the potential shared boundary would cover all five snapshot consumers. ARCH-PURE: keep generated request data and outcome classification separate from subprocess IO. ARCH-PURPOSE: classify the crash rather than hide it with an unchanged retry. ARCH-MOCK: this stage is conformance of an external runtime, so use actual isolated Node processes; a later boundary regression can use an owned stateful child, not mocked calls. ARCH-CONSTRAINTS:120 sequential children,60s overall, no native lane or load injection. ARCH-SECURE: only fixed generated fixture inputs; official binary plus checksum; do not dump ambient environment or user sessions. ARCH-ORDER: each child is joined before the next; stop at the first fatal signal and retain that exact generation, with no automatic retry. ARCH-FUNERAL: private runtime/probe scratch is invocation-owned and removed after the bounded report; only a demonstrated crash input is preserved as explicit issue evidence.

### 2026-10-18 — retained evidence audit

Claimed unowned card and ran start-plan. The gate refreshed the ariadne substrate and explicitly requested one rerun of start-plan before creating000668-native-oracle-segfault. No code edits, install, native lane use or causal fix yet.

Full failed job:/tmp/pair659-ci-39111981159.log. Runtime:macOS26.6.2(25G83), image macos-26-arm6420261013.0424.1, Node22.23.3 arm64, npm10.9.9. The crashing child is the node driver.cjs process; SIGSEGV and empty CombinedOutput are observed. They do not identify its execution stage. The driver emits its complete stdout only after writes, snapshots and dispose finish, so empty stdout cannot distinguish startup/import, frame processing or shutdown. JavaScript exceptions normally go to its catch/stderr path. Dependency lock lists only headless6.0.0 and unicode11-addon0.9.0; no native addon is declared. Node/V8 and OS remain native surfaces.

Attempt1 retained only parent-wire prefix4096 of21043bytes and child-wire prefix4096 of4572bytes; the complete failing request is unavailable. GitHub run artifacts API returned total_count0. The transient pre/post scratch was removed by the fixture. Original temporary PR merge objectfb2be7a7 was not present locally; the known branch head46e0bd0379ccf8ccabb972fa3b62aea2a9a36871 is present and has no driver/package-lock delta against current code.

The authorized historical CI retry succeeded on the unchanged branch head: run39111981159 attempt2, job117749821939, also Node22.23.3 arm64 on macos-26-arm64. Direct baseline3.43s, wrapped23.60s, family27.03s; runner confirmed all6 required native tests/subtests executed. Retained retry log:/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair668-ci-gtapl3q4/attempt2.log(39997bytes). Attempt1 failed once, attempt2 passed; this is no measured underlying flake rate or cause.

Local headless control on installed Node26.0.0 darwin/arm64: PAIR_TERMINAL_ORACLE=1 go test ./cmd/internal/terminal -run ^TestRendererIndependentWireOracle$ -count=20 -timeout=60s passed in3.895s. This confirms only those complete generated renderer requests on the different local runtime; it does not replay the lost CI request or qualify native conformance. Read-only source audit enumerated five snapshot exec sites (three couchtty, two terminal); all use CombinedOutput and omit complete request/runtime provenance from their failure reports. A primary-source upstream search found no matching demonstrated cause; unrelated Node SIGSEGV reports are not evidence for this incident.

2026-10-18 plan clarification before approval: count malformed controls inside the120-child envelope (per runtime:30 bare +27 complete +3 malformed), and distinguish the60s execution bound from the one120s-bounded runtime download. No additional process family or unbounded download is implied.
