---
id: 000833
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '22176ac8b101255a8961c0f1e8316eb2dd7b2670' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T09:26:13-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "d901db47", done: "654d9c75"}
---

# Preserve shutdown cancellation cause across presenter requests

## Problem

A console owner shutdown can be reported as a terminal failure instead of cancellation. #739 recorded suite on eafb27680f56c799d06d18e7639b15b6e5d154c4 failed only TestRunConsoleObservesStartupBeforeArmingProducers: hubs_startup_test.go:457 console startup=1, stderr "couch: terminal: terminal: presenter unavailable". Attempt598a61ed6ef5c7be79c26f6cf168a978, failed log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/598a61ed6ef5c7be79c26f6cf168a978.log, reported SHA prefix4d2073b0e476, duration11m43s. A focused20-run repeat passed before qualification, so no incidence rate is claimed.

## Spec

Preserve the causal error across presentation requests after a presenter fails during owner shutdown. A cancellation-only shutdown must remain normal; real physical failures, including joined physical failure plus cancellation, must remain failures. Do not suppress all "presenter unavailable" errors or expand deadlines.

Trace: FakeHost.Write makes bytes observable before Presenter.Panel returns. The old ordering fixture called Console.Stop immediately after observing startup bytes; Stop cancels Console.lifetime. An in-flight paint can enter Presenter.fail with context cancellation while the presenter lifetime is still live. Presenter.call rejects a failed presenter with the synthetic "presenter unavailable" error (the queued-request path similarly reports "presenter failed"), dropping the causal cancellation. Console.terminalError cannot classify that synthetic error as cancellation and latches a failure. showMenuPublishing performs further chrome publication queries after the write. The suite captured the synthetic failure; this exact internal interleaving is source-derived, not an instrumented stack capture.

A held-write test control in #739 deterministically proves that the old fixture cancels startup paint before publication completes (RED0.398s, "startup paint cancelled before release"). It did not deterministically reproduce the synthetic unavailable error itself; this follow-up must qualify that final interleaving. Existing couchtty/terminal_exit_test.go cancellation-only/physical-error controls cover a shorter command path and must stay meaningful. Console.Stop documents that it is safe from any goroutine, so this is a shared production diagnostic gap rather than just startup policy.

## Done when

- A deterministic held-paint/next-request regression reproduces loss of the cancellation cause and passes after the root fix.
- Cancellation-only shutdown exits normally, while physical failure and mixed errors remain fatal with the original cause preserved.
- No broad ignored-error text matching, timeout increase, or startup-policy change.

## Plan

Small, single-pass bugfix; expected inside the quick-flow shell (<100 added production lines, no milestones). Keep the plan here. TL requested the plan before implementation; send this committed design and wait for disposition.

### Diagnosis and design

`Presenter.fail` retains the causal error in `failureErr`, but `call` admission, the queued Failed branch in `run`, and `Present` admission replace it with synthetic errors. The `call` completion branch can also return synthetic released after a failed presenter closes. Additionally, `fail` currently publishes Failed and its cause in separate mutex transactions, permitting admission to observe Failed with no cause.

Use the existing first-failure latch as the one source of truth. Publish the Failed transition and its joined cause under the same `Presenter.mu` critical section, retaining the existing pure `Transition` owner (factor a lock-held transition helper if needed). A shared lock-held refusal constructor returns a contextual wrapper with exactly the stored cause as its wrapped error when present; healthy closing/released refusals retain their existing generic fallback. Reuse it in all four rejection sites. Never join a synthetic refusal leaf to cancellation: that would defeat the existing cancellation-only classifier. Keep drag cancellation, first-failure ownership, Failed notification, release/join and physical mode restoration intact.

Existing argument validation and `Present`'s early caller-context check keep their precedence. An already-canceled caller can still receive its own cancellation; the presenter retains any physical failure for `Failure()` and console teardown. Regression tests pin both facts. No reclassification in Console, string matching, ignored-error expansion, timeout changes or startup-policy edits.

Alternatives: returning caller cancellation on every failed admission loses physical failures; ignoring generic unavailable during Stop also hides real failures. Both are rejected. Cause-preserving wrapping fixes the information loss at its owner (ARCH-DRY, ARCH-PURPOSE).

### Concepts and operating envelope

| Concept / integration | Owner | Treatment and test |
|---|---|---|
| View transition | `cmd/internal/terminal/view.go` | Reuse pure `Transition`; no new states or enum changes. |
| Failure latch and request refusal | `cmd/internal/terminal/presenter.go` | Repair state/cause publication and rejection transport under the existing mutex; deterministic held-writer tests. |
| Cancellation-only classification | `cmd/internal/couchtty/terminal.go` | Reuse pure `shutdownCancellation` unchanged; error-tree tests plus real Console teardown. |
| Physical writer | `ttyio.Writer`, `ttyio.Fake`, `hostty.FakeHost` | Reuse stateful byte/partial-write fakes and test-only held writer wrappers; no live fleet or private native harness. |

ARCH-PURE/ARCH-MOCK: error wrapping remains tiny glue around the existing pure transition/classifier. The fake host tracks accepted bytes, raw state and closure through the real presenter and console. No external service changes. ARCH-SECURE/ARCH-EVOLVE: touches no new untrusted input, credentials or persisted format. ARCH-FUNERAL: creates no durable runtime artifacts, goroutines or growing collections; existing Release and Console teardown remain the lifetime owners.

ARCH-CONSTRAINTS: this is request admission and shutdown. Each rejection reads one retained error under one mutex, O(1) work plus one wrapper; memory per rejected call is O(1). No per-byte, endpoint or queue scan. Existing MaxPendingEvents and WriteTimeout remain the capacity/deadline bounds. At N=8 and 2N=16 rejected requests assert zero additional physical writes, no callbacks executed and unchanged first failure; count effects rather than timing. Disk/network work is absent.

### State and failure coverage (ARCH-ORDER)

Actions: continue a healthy session; stop/join; start a fresh console after physical terminal recovery.

| Internal state / event | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Healthy / request succeeds | Running | Continue | None |
| Owner canceled while paint is held; presenter fails with cancellation only; queued/later query | Normal shutdown, preserved cancellation tree | Stop/join | Incomplete frame; existing release repairs modes |
| Paint has physical failure, or physical failure joined with cancellation | Terminal failure with original cause | Start fresh after terminal recovery | Incomplete frame; release failure remains separately reported |
| Failure then repeated requests or Release racing completion | First failure stays retained; refused requests cannot write | Stop/join | No additional effects; pending frame may be incomplete |
| Healthy release / later request, without a latched failure | Existing generic closed/released refusal | Stop/join | None beyond interrupted output |
| Live caller cancellation before owner shutdown | Existing failure classification | Stop/join | Same as current contract |

Mutex publication forbids Failed with an absent cause; no observation-only proof or second latch. The actor still serializes paint and queued work. Stop may cancel the writer while producer requests exist; tests gate writer entry, queue admission, failure completion and release explicitly, and join every test goroutine. Request callbacks after failure never run. Process death, reconnect and durable retry are not introduced by this repair; no retry or rollback is added.

### Checkable work

- [ ] Add failing regressions in `cmd/internal/terminal/presenter_failure_test.go`: hold a real Panel paint at `ttyio.Writer`, admit a second request behind it, cancel the paint context while keeping presenter lifetime live, and await the failure. For guaranteed queued admission, insert a package-private presentRequest while the actor is held; separately exercise public post-failure ChromeState and background-context Present. Assert all preserve `errors.Is(context.Canceled)` and the complete causal tree. Repeat with physical and joined physical/cancellation errors using a held writer wrapper (`ttyio.Fake` alone returns ctx.Err before its configured Err on cancellation); assert errors.Is physical sentinel and errors.As WriteFailure, unchanged Failure(), no callback/write after failure, and canceled-caller precedence. Use barriers and bounded condition waits, never sleep to select ordering. Isolate the fourth refusal path with a package-private presenter: admit a healthy request, receive it from the queue, then install the failed/released snapshot under the mutex and close completion without a request result, so p.done is the only ready completion; test retained-failure and healthy-release fallback. Also retain real Release lifecycle controls.
- [ ] Run the new focused tests RED and record the actual failed assertions. Implement atomic failure publication plus one shared refusal constructor in `presenter.go`; rerun GREEN. Assert observers cannot see Failed without Failure(), and sweep all synthetic rejection constructors. Run N/2N rejection effect counts.
- [ ] Extend `cmd/internal/couchtty/terminal_exit_test.go`: use its held-paint host and a command that issues a publication query after the canceled paint has completed, before returning to Run. Signal callback completion independently because runTerminalCommand can return on Stop before the callback finishes. Call Console.Stop from the test while writer entry is held. Force the next-query path rather than relying on a timer. For cancellation-only expect Run=0, nil TerminalFailure and empty diagnostics; for physical/mixed expect Run=1, original physical cause and recorded exit reason. Keep zero/partial accepted-byte cases, host close/raw restoration and parent repair assertions; preserve existing shorter-path controls.
- [ ] Run `go test ./cmd/internal/terminal ./cmd/internal/couchtty -run 'TestPresenter.*Failure|TestPresenter.*Release|TestConsoleStopDuringPaint|TestShutdownCancellation|TestConsoleLiveCancellation' -count=1`, then focused race repetitions of the new deterministic cases. Run terminal, couchtty, couchcmd targeted startup controls and repolint preflight. Update the existing terminal atlas contract/lessons with the cause-preservation invariant if the implementation changes documented behavior. No new general lint rule: this is semantic error provenance, not a mechanically recognizable API misuse.
- [ ] Before close run `make test-affected-inventory`, audit and repair only named changed readers, push through `sdlc pr`, run recorded `sdlc local-check run --name affected`, and close through the binary's one fresh-context review. Report verification and ask hub for any required live-check disposition; do not stage or touch :0 without it. Land through SDLC after the required checks and TL disposition, then report the PR to hub.

## Log

2026-10-23: Hubf9476c31 approved separate unclaimed filing; #739 itself changes its startup-ordering fixture to normal switcher Escape with a held-write control. This follow-up owns shared shutdown error classification. No fix or full reproducer is claimed here.

2026-10-23: Claimed pair:9 and ran start-plan on an independent branch from main. Inspected pair:7's test-only startupPaintBarrierHost read-only; no stacking or runtime consumption. Its bytes-before-return control validates the issue's shutdown ordering but does not itself force the later refusal. Source audit found the three failed-request rejection sites plus call's released-completion branch, and the separate Failed/cause publication window. Existing cancellation-only/mixed-error/live-cancellation controls passed: `go test ./cmd/internal/couchtty -run 'TestShutdownCancellationRequiresOnlyCanceledLeaves|TestConsoleStopDuringPaintClassifiesJoinedFailure|TestConsoleLiveCancellationFailureRemainsLatched' -count=1` (0.379s). New regression is planned, not yet written or claimed reproduced. No production code changed. TL plan disposition pending.

2026-10-23: Independent read-only inspection by codec_review confirmed the four refusal paths and publication window. Incorporated explicit queued admission, custom joined-error writer behavior, canceled-caller precedence and separate terminal callback completion (runTerminalCommand can return on Stop before the command finishes). This inspection is planning evidence, not the binary-owned close review.

2026-10-23: TL approved the committed design (Couch delivery 30f3a179-f7ca-473e-9332-66307e71c48f); change-code admitted quick flow. Before any production edit, new presenter regressions ran RED (`go test ./cmd/internal/terminal -run 'TestPresenterFailureCauseSurvives' -count=1 -timeout=30s`, 0.169s): queued refusal is synthetic presenter failed, later ChromeState/Present refuse with unavailable, released completion returns released, all losing WriteFailure and cancellation/physical cause. The real Console final hop independently ran RED (`go test ./cmd/internal/couchtty -run '^TestConsoleStopDuringPaintNextQueryPreservesCause$' -count=1 -timeout=30s`, 0.290s): zero/partial cancellation-only cases exit 1 instead of 0 with exactly couch: terminal: terminal: presenter unavailable; mixed and physical-only next queries lose their sentinels. Tests now force the reported failure; no production fix was present in either run.
