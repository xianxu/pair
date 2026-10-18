---
id: 000668
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'bc12087889b5a38161ebb3fd2d0f03650faf1b3b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T14:16:34-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "15ca87b6", done: "2b4c3d0c"}
actual_hours: 0.66
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

Parked pending the next failing CI run. Diagnostic code is already on main via PR439; a new causal implementation needs a fresh plan and branch/PR.

- [ ] Read the next failure's complete oracle request, separate stderr phases, process outcome and optional Node report; preserve missing evidence as unknown.
- [ ] Reproduce or classify the measured cause, then propose a narrow fix and deterministic regression to the hub before code.
- [ ] Run ordinary verification/review/merge gates for that causal change; never retry inside the oracle or relax frame assertions/deadlines.

## Log


- 2026-10-18: closed — Diagnostic stage: helper/terminal/artifactpath packages, four Node controls, seven Python retention tests, and native CI 39168539412 with all six required tests/subtests passed. BR-1 root README invocation coverage is fixed across retention, replay and controls; docs-only fix preserves queued suite code. Recorded suite da0de6c5364a3ef25592d24cebaa199c remains queued, not claimed passing. tl-worker permits review overlap; merge requires its final-code pass. Only unchecked plan row is pending suite/close/merge/reopen chronology. Historical cause unresolved; restore causal contract and reopen/park after landing. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
- 2026-10-18: Filed unassigned under the fleet flake rule after distinguishing it from #660; #659 will rerun only this failed CI job once on the unchanged head and stop/report on a second failure. Its full integration suite passed exit 0 in 11m3s before this CI failure. Historical root cause remains unknown.

### 2026-10-18 — reopened after diagnostic landing

Reopened per the hub-approved diagnostic-stage contract: PR439 merged as c5245132b9eb26588c56b05a19f733261368f69e; archive9950fa540423ab382e682437fc0564fadc104f29. Branch suite attemptda0de6c5364a3ef25592d24cebaa199c passed on77952363b5474fc6381b3d0f610cdce9974ebde4 in11m35s (log9d767ecb5296); authoritative integrationc0d87c90496de9a1e73a733df576c861 passed on5bdac877594e096ad135498b2d51547dbcb81531 in11m47s (log58ea061349ca), all5 final CI checks green. Final review SHIP, BR-1 root README coverage disposed; no code findings remained. The historical SIGSEGV has no retained exact failing request/stack and no measured cause. Restore the causal acceptance criteria, then unclaim/park pending the next CI hit; inspect complete request, phase stderr and optional Node report before choosing a fix. A green diagnostic run is not a causal resolution.

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

### 2026-10-18 — hub-approved diagnostic-stage revision

Verified hub receipt0ada5fa1-a48d-4d23-9c84-acaf5facf536 directs skipping the120-run probe and Node download, landing small diagnostics, then reopening/parking for the next CI hit. The active Plan and temporary diagnostic Done-when reflect that direction. The causal contract to restore after landing is:

#### Causal acceptance to restore

- Retained diagnostics identify the crashing subprocess/runtime and causal input or environment.
- A deterministic regression protects the measured cause without relaxing terminal checks or increasing deadlines.
- Native conformance passes with before/after failure counts reported.

The Node report flags do not guarantee a report on raw SIGSEGV: report-on-signal defaults to SIGUSR2 and report-on-fatalerror covers internal runtime fatal errors. Exact request plus last observed phase remain necessary even when no report exists. Official API: https://nodejs.org/api/report.html (exclude-env available since22.13, exclude-network since22.0). Go1.26 ArtifactDir is temporary without -artifacts; native CI currently uploads no artifacts and removes scratch, so the runner must copy failure evidence to its existing retained job log before removal. Manual -artifacts/-outputdir runs can retain the original files. No protected workflow edit is required.

ARCH-DRY: one snapshot subprocess owner across five callers; stream.cjs has a different long-lived protocol and is unchanged. ARCH-PURE: request/report data stay distinct from the thin synchronous subprocess shell. ARCH-PURPOSE: preserve the next failure, never claim that diagnostics fix this cause. ARCH-MOCK: controlled owned subprocesses model report/exit/stream outcomes through the actual helper; real Node checks driver/report compatibility and native CI supplies native conformance. ARCH-CONSTRAINTS: one existing Node process per call, no new probes or retries; phase writes grow2 per chunk plus constant setup/dispose, failure artifact bytes grow linearly with exact request/streams, counted atN/2N. Normal success adds only the unique report-directory setup and removes it. ARCH-SECURE: request is synthetic fixture data; exclude report environment/network data, preserve separate quoted streams, use private invocation paths and never inspect arbitrary user reports. ARCH-ORDER: cmd.Run joins before cleanup observes outcomes; later assertion failure retains that exact call, success removes it; no effect outlives its owner. ARCH-FUNERAL: Go owns artifact lifetime; CI reports failures before removing scratch, manual retained artifacts belong to the explicit output directory, no persistent automatic cache.

Diagnostic actions: inspect bundle, replay exact request, or wait for the next CI hit. Complete request + phase/report permits replay; missing report is reported as missing and still permits request replay; absent startup marker leaves execution stage unknown; evidence-write failure is explicit and calls for fixing that IO before relying on a later run. No such observation authorizes an automatic retry.

### 2026-10-18 — diagnostic implementation and targeted red/green

The shared helper regression first failed on missing Node report flags, then passed with actual controlled children (exit7, killed signal, cancellation before spawn and later caller assertion failure). Exact request budgets are15,395 and30,755bytes, including Unicode/control characters; saved bytes equal submitted bytes, stderr is independent, missing reports remain missing, each call owns a distinct directory, success removes it and Go's default outer artifact lifetime removes unpreserved files. Path/Dir/Env and existing context are preserved. All five driver callers now use this helper; stream.cjs is unchanged. The artifact manifest classifies the new test-only helper.

Real headless Node tests first failed3/4 because phase records were absent; after implementation all4 passed in1.209s. Ordered phase count is2N+3 atN=4/8; stdout still decodes to the original frame response. Node's actual report control checks flags enabled, runtime identity present, and environment/network fields absent. This is a report compatibility control, not reproduction of the historical SIGSEGV.

Native CI retention tests failed on absent complete request output before implementation; after adding the qualification failure boundary all7 Python tests passed in0.141s. A36KB request and report final markers survive in the published job log before scratch removal, ordinary exit7 and successful-but-incomplete execution are rejected, and success does not publish a bundle. Existing malformed-output coverage still joins its owned child. The Go -artifacts flag is boolean, with a separate -outputdir under owned scratch; no workflow/gate configuration changed.

Initial integrated targeted command PAIR_TERMINAL_ORACLE=1 go test ./internal/testoracle ./cmd/internal/artifactpath ./cmd/internal/terminal ./cmd/internal/couchtty -run Test(Run|RequiredOracle|RendererIndependent|HistoryOracle|Manifest|NonArtifact) -count=1 -timeout=120s passed all4 packages (0.818s,0.157s,1.743s,0.318s). Broader affected-package guards are running next. No local native lane spent, no runtime download or version change, no retries or deadline changes.

### 2026-10-18 — early draft CI and review overlap

Early draftPR439 opened on77952363b, using the standing overlap exception. Broader verification PAIR_TERMINAL_ORACLE=1 go test ./cmd/internal/terminal ./internal/testoracle ./cmd/internal/artifactpath -count=1 -timeout=120s passed (75.697s,0.713s,10.397s). Artifact inventory/dead-symbol/consumer guards therefore ran, not only matching selectors. Native CI run39168539412/job117796542047 passed all4 Node controls and confirmed execution of all6 required native tests/subtests; direct2.26s, wrapped22.22s, family24.47s. Other CI checks passed. Historical observation remains one failing CI attempt followed by its authorized passing retry; this new passing diagnostic run establishes no cause or fix.

Recorded suite attemptda0de6c5364a3ef25592d24cebaa199c on77952363b is queued (machine-wide gate, position2 last observed), not yet a pass. Under tl-worker's explicit permission for a close review to overlap a queued recorded suite, request the boundary review now. The final unchecked Plan row contains that pending suite plus close/merge/post-landing reopen actions; --no-plan-check acknowledges only those chronological gate actions, not omitted implementation. A recorded suite pass on the final code still precedes landing, and any review code fix requires a fresh recorded suite. No main chasing or extra baseline is requested. This diagnostic change does not assert a mechanical root cause suitable for a Go lint rule; the unknown Node failure is still the causal card to reopen.

### 2026-10-18 — close review BR-1 disposition

Round1 found no code defect and returned FIX-THEN-SHIP, but its blocking ledger left close unfinalized: BR-1 root-readme-surface-coverage requires root README coverage of the new developer commands. Verified the root testing entry points were unchanged; added retention usage and links covering replay, headless controls and report/lifetime limits at both oracle/native guidance. The class sweep covers all three new invocation surfaces, not just the one link. Added the prevention rule to lessons. Documentation-only fix; the queued exact-code suite remains applicable. Per the gate's explicit error, rerun close to have the reviewer dispose BR-1; no manual ledger edit or waiver.

### 2026-10-18 — restore causal contract and park

Diagnostic PR439 is verified merged; the temporary diagnostic acceptance is fulfilled. Restore the original causal Done-when and replace the diagnostic implementation checklist with a next-failure investigation plan. Keep the historical diagnostic contract below as evidence of exactly what landed. No pending test, merge runner or native lane remains owned by this slot. The current branch is a handoff only; do not reuse the merged PR or claim a passing diagnostic run fixed SIGSEGV.

#### Completed diagnostic-stage contract and plan

##### Diagnostic Done when

Diagnostic-stage landing only (hub-approved2026-10-18); the causal bug stays open afterward:

- Every driver.cjs snapshot invocation enables Node fatal-error/signal reports in its owned artifact directory and preserves complete request JSON, separate streams and runtime/phase evidence on failure. Success still returns only the unchanged JSON oracle result.
- Deterministic owned-child regressions show input beyond4KB survives byte-for-byte, stderr never corrupts JSON output, failed artifacts remain available through reporting, and successful evidence directories are removed. Real headless Node verifies phase order/report configuration without changing frame expectations or deadlines.
- The native CI runner emits the failed oracle bundle into its retained job log before removing invocation scratch; its regression proves complete final input/report markers survive. No workflow/gate configuration change.
- Recorded suite, native CI and ordinary close/merge gates pass. Then restore the causal acceptance contract below, reopen and unclaim/park this issue for the next failing CI evidence. A diagnostic pass does not prove the historical SIGSEGV fixed.

##### Diagnostic Plan

Approved diagnostic stage; the proposed runtime replay/download is withdrawn.

- [x] Add failing subprocess/evidence regressions in internal/testoracle/run_test.go: actual controlled owned children, independent stdout/stderr, complete Unicode/control-byte JSON over4KB, abnormal exit, post-return assertion failure, report flags/directory, success cleanup and failure retention under Go's artifacts flag. Count retained bytes atN and2N; no timing ceiling.
- [x] Implement one small internal/testoracle.Run(t, cmd, request) IO helper for all five snapshot callers (three couchtty/terminal_native_test.go sites and terminal/render_oracle_test.go + history_oracle_test.go). Use unique subdirectories of t.ArtifactDir; enable --report-on-fatalerror, --report-on-signal, --report-directory, --report-filename, --report-exclude-env and --report-exclude-network. Keep each caller's command/context/deadline and every existing frame assertion. Capture stdout/stderr separately; save exact request and streams when the owning test fails, including a later frame/decode failure. Remove successful call directories. Register the new Go helper as a non-artifact source in the existing manifest.
- [x] Add synchronous stderr phase markers in driver.cjs for startup (runtime identity), import, write, snapshot and dispose. Keep stdout exclusively the existing JSON response. Regressions use the real headless driver, check ordered markers and valid frame JSON, and prove flags configure report creation with an owned signal/report control. No Node download, version pin, exception masking or retry.
- [x] Extend tests/native-terminal-ci.py to invoke Go with -artifacts and -outputdir under its owned scratch, then publish the failed oracle files into the existing retained CI log before scratch cleanup. Extend native-terminal-ci-test.py with full request/report final-marker and cleanup-order controls. Update terminal-oracle README and atlas map for replay/evidence limits.
- [x] Run targeted helper, Python, renderer-oracle and repository guard tests, preserving red/green evidence. Open early draft for existing macOS native CI. Run the recorded suite and close/merge through ordinary gates; no local native lane is spent without a fresh hub grant. After landing restore causal acceptance, reopen/park/unclaim and report exact merge plus unresolved-cause status.

