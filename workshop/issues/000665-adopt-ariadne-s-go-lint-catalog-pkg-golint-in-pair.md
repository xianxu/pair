---
id: 000665
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours: 7.61
collaboration-mode: collaboration
card_mirror: 'e43e70d350aabfb4392efbacce14d8a02d1a7e5a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T05:39:24-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# Adopt ariadne's Go lint catalog (pkg/golint) in pair

## Problem

ariadne#407 added pkg/golint, a stdlib-only Go lint catalog run as a repository test, so every landing enforces it. Its seed rules come from mistakes made in both repos: test-shrunk-deadline (ariadne#434), tmp-literal (#342), test-go-build (pair#602), test-tight-wallclock and test-fixed-sleep (the 10-15 flakes). pair has the same classes (pair#659, #662, #623 were wall-clock budgets) but no catalog.

## Spec

Adopt the catalog in pair: run it as a pair repository test over pair's tracked Go files, with the same suppression convention (`//lint:allow <rule> — <reason>`). Choose how pair gets the package (a module dependency on ariadne, or base-layer files via weave) and say why. Sweep current hits: fix the real ones, give the rest reasoned suppressions.

## Done when

- pair's suite runs the catalog over every tracked Go file and fails on a new finding.
- Every current hit is fixed or carries a reasoned suppression.
- The way pair consumes the package is documented in atlas.

## Plan

Durable plan: [Go lint catalog](../plans/000665-go-lint-catalog-plan.md).

- [ ] M1 — Enforce the module catalog; resolve all78non-sleep findings; quarantine exactly103existing sleeps in one shrink-only tracked site list; qualify dependency consumers, review and land the guard.
- [ ] M2 — Remove sleep debt with explicit signals, prioritizing flake-history packages (#659/#662/#623/#485); review and land the completed sweep.

## Log

### 2026-10-18

### 2026-10-18 — Claimed and initial design

Claim and start-plan succeeded in pair:1 after the hub corrected the #636 dispatch; no changes/native runs were made for #636. start-plan refreshed the ariadne layer and required one clean re-invocation before branching. Peer MEMORY.md is absent; AGENTS.local.md was read. Chosen proposal: consume upstream golint through the existing module pin, avoiding duplicated catalog logic and a new protected weave export (ARCH-DRY). Read-only baseline found 181 findings in91files over2547tracked Go files; the durable plan records every affected path and forbids blanket migration suppressions. Module advancement also changes the existing runtime layergraph dependency, so compatibility qualification is explicit. Await hub approval before implementation; no code has changed.

### 2026-10-18 — Plan review cleared

Plan-quality round2 CLEAN disposed PQ-1; no open blocking findings. Estimate derived after that result. No implementation has begun; M1's catalog test and exact-site debt validation will start red.

### 2026-10-18 — M1 implementation and qualification checkpoint

change-code passed plan-quality CLEAN and estimate-quality INFO; no blocking finding. Module tooling resolved the approved source to v0.0.0-20261018104241-81c832d5e000; go mod tidy leaves only the new ariadne sum pair, no unrelated dependency upgrades. Runtime compatibility selection TestDeclaredDeps*/TestRunPrepare* passed, including new disposable linked-worktree primary-anchor coverage for both dependency ownership and substrate commit reporting (7.637s).

The scanner and debt controls are test-only under internal/repolint. Original103sleep sites come from the committed pre-code source anchor; current debt100after the telemetry600ms and two FIFO200ms sleeps were necessarily removed with their coupled wallclock repairs. Ordinary repository scan passed over2553tracked Go sources/five rules. Exact counts will be refreshed on the complete staged head. Original181findings remain in /tmp/pair665-catalog-initial-hits.log; the concurrent implementation's red run /tmp/pair665-m1-catalog-red.log is not mislabeled as an untouched baseline. Pure/inventory/history normal and race pass; fuzz57140executions pass; ordinary-test tracked-new-sleep and corrupt-debt mutations fail and were restored (/tmp/pair665-catalog-new-site-mutation-red.log, /tmp/pair665-catalog-corrupt-debt-mutation-red.log).

Temporary-path dispositions:24real storage roots now respect TMPDIR using compact prefixes where direct Unix sockets require headroom. Fifteen adjacent specific exceptions retain two stable cross-process UID rendezvous roots, six version/session socket trees and seven digest/PID notification socket trees. Source comments carry the per-site reasons. Six affected-package normal targets passed under the actual macOS /var/folders TMPDIR; no native gates were enabled (/tmp/pair665-tmp-targeted.log). Parser/logical namespace paths were not mistaken for actual native socket paths.

Four build findings now use one suite-owned Cache for each distinct executable bundle: Couch+registration fixture, launch helper and probe shim. Pair still uses its existing testpair binary. Probe aliases share the same executable while preserving basename, PID, argv, stdin, exit17 and trace controls. Counted real-Go control over repeated tests proves one Couch build, one fixture compile, one Pair build and one shim build; artifacts are absent after TestMain cleanup. Supplied startup helper-directory override remains; its fake-Go repeated test counts one fallback build. Logs: /tmp/pair665-buildcache-red.log, /tmp/pair665-buildcache-green.log, /tmp/pair665-buildcache-race.log, /tmp/pair665-real-build-calls.log, /tmp/pair665-counted-real-builds.log, /tmp/pair665-startup-build-sharing.log. Artifact classification and tests pass for internal/testbinary.

Non-core timing dispositions: hasPin performance is now exact one-cell search work atN/2N rather than200ms host time; canceled panebirth sleep completes through an event rather than100ms elapsed; telemetry write must complete while handler remains signal-blocked rather than winning500ms against600ms sleep; maintenance command proves shell readiness before a cued DeadlineExceeded and command join. Recorder observes Draining before firing its timeout, and repeated Close must create exactly one timer. Chrome protocol fake controls last-page grace and a replacement invalidates the previous channel before stale expiry. Two visibility-grace settings remain narrowly suppressed because readiness precedes the configured visibility behavior and explicit phase/notice events prove completion. Targeted normal/race tests for these ten findings passed; production deadlines and user behavior remain unchanged. Native fixture qualification is not claimed by these skipped-default tests.

Couchcore's25deadline/wallclock findings are under final race/mutation qualification; per-site dispositions are retained at /tmp/pair665-couchcore-dispositions.md. No final suite or milestone close has started. All source mutations must be restored and agents frozen before the recorded suite.


### 2026-10-18 — M1 source frozen before recorded suite

All78original non-sleep findings now have a disposition:24temporary-root repairs and15specific temporary-root exceptions; four test-build findings consolidated into three suite-owned once-build callbacks with exact local reasons;35deadline/wallclock findings repaired or semantically justified (the four remaining deadline exceptions are two visible-grace subjects, one fake worktree-add budget and one non-timing retention sentinel). There are no inline sleep suppressions. All original assertions/input domains are retained.

Couchcore's25-site per-function disposition follows; original line references identify the baseline, not current source positions.


| Original site | Final disposition |
|---|---|
| coldresume_birth_test.go:111 | Replace150ms with registrationDeadline/newSpendableBudget spent on third admitted pane poll; preserve >=3 observations and no session probe during wait. |
| coldresume_birth_test.go:157 | Remove unused150ms success override; warm attach zero cold observations and unchanged record/argv assertions remain. |
| couch_test.go:217 | Remove shared5ms override; use unchanged constructor production cleanup policy (500ms) instead of a fixture race. |
| couch_test.go:1215 | Remove real-descendant250ms override; unchanged production cleanup policy drives real joined child/group cleanup; all8 stdio/PTY×failure cases retained. |
| launch_timing_test.go:76 | Spend registration budget after2 actual pane polls; exact phase sequence and retained error checks preserved. |
| launchfailure_capture_test.go:155 | Remove unused1ms: injected registration observer already returns error immediately. |
| never_turned_test.go:23 | Remove unused300ms: existing assertion already proves registration factory never called. |
| operationdispatch_test.go:99 | Remove unused1ms: caller context already canceled; cause assertion retained. |
| park_wait_test.go:42 | Replace300ms elapsed delivery cue with explicit default/override selected-duration controls and signal-driven completion/expiry. Private completionDeadline seam observes actual duration passed by production await, defaults to context.WithTimeout. |
| provision_budget_test.go:71 | Narrow documented suppression: fake starts200ms only upon exact worktree-add invocation; preserved exact200ms persisted diagnostic, write-budget capture and clean retry controls. Real Git probes keep separate original budgets. |
| registration_phase_test.go:24 | Spend birth budget after2 pane polls; phase/deadline/zeroownership/per-step diagnostic assertions retained. |
| registration_phase_test.go:59 | Remove unused150ms; existing second-ownership-check spendable budget already controlled expiry. |
| relaunch_test.go:215 | Wait seam expires after missing-completion observation; remove both5ms completion/relaunch settings. All refusal/recovery/no-resume assertions retained. |
| relaunch_test.go:235 | Wait seam expires after committed-completion/live-child observation; remove both5ms settings. |
| relaunch_test.go:479,482 | Remove50ms deadline and200ms elapsed cue. Actual selected duration captured through completionDeadline; only relaunch override admits deferred quit. First-call success stays required. |
| terminal_incarnation_test.go:48 | Remove shared40ms. Exact-name failure test spends deadline when proposed name is observed after admission; old-terminal survival/cleanup-target assertions retained. |
| threadstore_preview_test.go:191 | Remove25ms/500ms race. Private context-carried contention observer spends budget only after EWOULDBLOCK and asserts exactly1 attempt + DeadlineExceeded across global/local/routed-local; pre-canceled assertion retained. |
| threadstore_preview_test.go:295 | Replace150ms-vs200ms writer race with writer-release channel. Read must finish while writer is withheld; hang guard + cancellation/release/join teardown. All FIFO refusal/unchanged-file and routed fallback assertions retained. |
| warm_attach_registration_test.go:13 | Remove unnecessary100ms around success subprocess checker; attached/detached live helper/unchanged record assertions retained. |
| warm_attach_repair_test.go:282 | Same signal-gated FIFO writer; successful BORROWED ATTACH must finish before writer release. Reader and writer joined on failure too; no50ms startup deadline. |
| warm_attach_repair_test.go:304 | Spend deadline at first actual lock contention; exactly1 attempt, DeadlineExceeded and nilhandle retained. |
| warm_attach_repair_test.go:382 | Same contention cue for all3 inventory backends; exact1 attempt + DeadlineExceeded. |
| warm_failure_test.go:262 | Narrow documented suppression for200ms sentinel explicitly asserted by retention controls. Factory no longer arms a wallclock timer; registration-timeout route explicitly spends budget on post-admission session observation. |
| warm_failure_test.go:273 | Remove unnecessary owning-start200ms override; failure injection and owning/warm cleanup matrices preserved. |


Couchcore targeted normal26.624s/race28.894s PASS, actual-default-budget mutation directly fails override/relaunch tests, ignored lock cancellation fails all3backends with175–179attempts against the exact1invariant, and restored focused normal3.631s/race4.243s PASS. Logs /tmp/pair665-couchcore-{deadline-green,deadline-race,budget-mutation-red,lock-mutation-red,mutation-controls-green,mutation-controls-race}.log. Mutation exposed worker-side t.Fatal in the new fixture; it now reports and returns an error rather than abandoning the worker. Added root mutation controls for pin-search work and page-timer invalidation: each fails its intended exact assertion, source restored, both focused tests pass (/tmp/pair665-pin-work-mutation-red.log and /tmp/pair665-page-grace-mutation-red.log).

Broader affected non-core package normal run passed: broadcast33.678s, webcast12.171s, terminalcapture0.964s, diagnosticlog47.788s, panebirth0.136s, couchcmd130.389s, couchtty59.343s, testbinary0.123s. Selected changed tests in all these packages also pass-race. Artifactpath and build-cache tests pass after explicit helper classification. No native lane use and no real operator state touched. All subagents have returned; every temporary source mutation was restored before final staging.


### 2026-10-18 — Authorized M1 review/suite overlap

PR428 is open on the M1 source commit. Recorded suite attempt fad7b8eea081451df8064a47db8affc5 is queued in the machine test line; its clone is /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/sdlc-local-2861851668. Authenticated hub deliveryc9aae4fb explicitly authorizes M1 milestone-close review overlapping this queue, with recorded PASS mandatory before landing and a new final-head suite required if review changes code; this is now a standing hub rule. Source is frozen. The review verification names targeted and mutation evidence only, never labels the pending full suite a pass. M2 preparation is read-only while M1 is outstanding.


### 2026-10-18 — M1 review round1 REWORK

Round1 inspected the complete M1 branch and reproduced BR-1's whitespace-authorized inline sleep bypass. BR-1 will be repaired across the upstream-accepted grammar, red first, and BR-2 gets README developer guidance including history prerequisites. The full staged source count is2556, independently confirmed by the reviewer and Git inventory; earlier2553was the intermediate staged scanner count. All GitHub checks on the original source passed, including native conformance, but no local full-suite PASS is claimed. Queued attempt fad7b8eea081451df8064a47db8affc5 was canceled by the owned local-check verb before admission because this fix requires a new source head; other checkouts were left running.

Authenticated hub delivery76f460c7 directs a BR-3 dispute, retaining fast disposable real Git repositories rather than implementing a second Git model. Plan R4 records the exact rationale, every-run conformance cadence and reconsideration criterion; the fresh reviewer will answer evidence. M1 remains unticked; source is editable for these findings only, M2 remains read-only.


### 2026-10-18 — M1 round1 fixes verified

BR-1 now derives rule identity with upstream's exact prefix, TrimSpace and ASCII-space token boundary. Twenty accepted whitespace/placement controls first prove upstream accepts the suppression and then require the scanner to refuse it; six lookalike/non-directive controls plus a string-literal control prevent a broader false-positive detector. RED retained at /tmp/pair665-br1-suppression-red.log. Final scanner package PASS9.236s, race PASS39.354s, suppression fuzz8074executions PASS; no new debt/parser/production helper. BR-2 README documents ordinary/focused check, exact non-sleep reason, exclusive debt file, full history/main-ref prerequisite and atlas link.

BR-3 dispute was recorded by sdlc finding dispute with fresh3.607s real-repository countermeasurement: tracked/untracked/deleted inventory and read/parser errors, exact8/16reads, candidate-vs-main authority, malformed/deleted/missing history and detached/ambiguous main refs. No fake Git or additional model was built, per hub76f460c7. The next reviewer must answer this evidence; dispute is not a fix or SHIP. Source restored and frozen; new code head must be pushed and receive its own recorded suite before landing.


### 2026-10-18 — M1 round2 and approved BR-3 correction

Round2 verifies BR-1 and BR-2 addressed and re-raises BR-3 solely on ARCH-MOCK. It reproduced the real-repository evidence but requires a shared injected fake/real read boundary. Authenticated hub42f6145f reverses its prior exception and approves the small repository-read interface with stateful index/ref/history/blob fake and every-run real-Git conformance. Plan R5 records the bounded model and named verification seams. No further dispute: address BR-3 and re-close. Canceled running e3ffaf6c suite through sdlc local-check cancel; it is incomplete, not a failed flake and not a PASS. Other checkouts were left running. Re-enter change-code for this revised design before implementation.


## Revisions

### 2026-10-18 — R1: two approved milestones

Hub delivery2b583e46 approves module consumption and layergraph qualification, and splits enforcement/78non-sleep repairs (M1) from103sleep repairs (M2). M1's exact-site JSON allowlist is temporary debt that can only shrink against the main baseline; no inline blanket exemptions. Plan R1 records matching, ratchet regressions and retirement. Implementation may proceed through change-code. Whole issue remains incomplete until M2; milestone publication will use the supported SDLC path or be escalated if unavailable.

### 2026-10-18 — R2: plan-quality verification refinement

Plan reviewPQ-1 requested named validation functions with adversarial-input strategies. Added sleepSites/decodeDebt/checkDebt/checkShrink/scanTracked/debtBaseline and the repository gate, distinguishing pure transformations from real-Git integration tests. Supersedes the prose test-case matrix without changing the approved scope or ratchet mechanism. No implementation has begun.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; the source is flagged stale, so calibration is provisional. Existing upstream checker and Go/Git avoid new-library discovery. Design uses the thorough-spec discount, implementation uses the v3.1 40% factor, familiarity1.0 and15%design buffer. Scanner/debt is one greenfield test module; M1 sweep is six bounded fixture groups and M2 sleep repair is twenty fixture-family groups (not103independent features). The grouped cross-cutting rows below aggregate those primitive counts; no queue/full-suite idle time is included. Three review boundaries comprise two milestones and whole-issue close.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.15 impl=0.32
item: cross-cutting-refactor design=0.24 impl=1.20
item: cross-cutting-refactor design=0.80 impl=4.00
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.00 impl=0.60
design-buffer: 0.15
total: 7.61
```
