---
id: 000533
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours: 2.199
card_mirror: 'fa987f74c0d57634421ae045c361d1ddacd78d69' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T14:01:25-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# draft: show progress while a DOM submission prepares, and hold the draft read-only until it's sent or fails

## Problem

After Shift+Ctrl+Alt+Return the draft looks idle for seconds while Couch prepares note evidence. The operator cannot tell whether the chord worked or what text is being sent.

## Spec

Show immediate visible submission progress in the draft status line, then update from actual work: preparing replay/context, settling and capturing crop i/N, saving the bundle, sending to the pinned slot, and sent or the failure reason. Progress is informational, never proof of delivery.

The initiating draft is read-only only while this operation is active. Success releases the lock before applying today's successful-send clearing rules. Every failure, exception, missing response, or timeout releases it immediately and preserves the authored draft, with a visible reason. A new press after completion creates a fresh submission as in #524; duplicate delivery remains acceptable. While an operation is active, another press must not replace its ownership or extend its deadline. No persistent freeze, discard binding, or confirmation dialog.

Measure capture, animation settlement, bundle preparation/publication, and pane-send phases before implementation; record measured numbers and limits here. Identify the dominant measured phase and whether it can safely be accelerated.

## Done when

- [x] Phase measurements and their reproducible commands are logged, with dominant-phase assessment.
- [x] The production chord visibly renders progress before external work and reports actual per-crop and pinned-slot progress.
- [x] The initiating buffer is read-only only until success/failure/timeout; all terminal paths restore editability and preserve failure text, and stale callbacks cannot clear or lock a successor.
- [x] Pure model, controlled integration, and headless production chord tests cover progress, lock lifetime, timeout and recovery; affected checks and full suite are recorded.

## Plan

See `workshop/plans/000533-submission-progress-plan.md`. One atomic close boundary.

- [x] Measure existing phases and finalize design.
- [x] Carry preparation progress through existing status polling.
- [x] Render progress and enforce operation-scoped editing/timeout ownership.
- [x] Verify, document, review, and land.

## Log

### 2026-10-13

- Received operator requirements through ops:0, verified Couch receipt c9b8646e-a7f4-49b3-8397-7e45fbdaa509. #533 was already claimed to pair:6. Started planning after SDLC refreshed this slot and its ariadne dependency.
- Investigation: server preparation has a 30-second context; client preparation currently counts 360 polls with 100ms delays and up to 4 seconds per RPC, so this is not an absolute client timeout. Synchronous list-panes, append/commit and Zellij actions use unbounded vim.fn.system. The plan must bound those DOM-only operations as well as the UI timer (ARCH-CONSTRAINTS, ARCH-ORDER).

- Initial isolated measurements before implementation: calm two-crop capture 194.759ms (settlement 1.198/0.563ms; screenshots 51.050/42.383ms). With 800ms animation, capture 857.531–873.928ms and settlement 780.195–795.264ms; with 900ms animation, capture 952.945–1007.452ms and settlement 886.534–933.857ms. Durable Store.Prepare for a tiny PNG 39.216–78.110ms; supersede plus publish 63.498–75.225ms. Private real Zellij bracketed paste of 1.7KB, 100ms settlement and AltEnter 179.683–233.568ms. These fixture measurements exclude production Couch begin validations and do not claim to time the operator's exact page. Animation settlement dominates the animated crop samples (about 91–94%); shortening it would violate #524's natural-completion capture guarantee. Full command/sample record follows when measurement completes.

- Plan 84e3ca051 approved by ops:0, receipt e5054a62-e77f-434b-a431-597d7c6d9c3d. Additional display requirements: “waiting for the page to settle, crop i/N” and “already sending…” on repeated chord; no deadline renewal.
- Completed measurement: three 320×140 crops (12,060 PNG bytes) published durably in 51.132/53.176/58.706ms. Real private Zellij sends including two list-panes identity checks took 416.850–629.031ms (median550.851ms); each list36.119–214.894ms, write52.902–140.078ms, submit55.398–138.440ms, deliberate sleep100.285–105.657ms. All8,570 expected bytes received by isolated raw PTY. Missing measurements: recording read/build, replay opening, initial pin, Couch begin RPC and session-log append/commit; do not sum these fixtures into claimed operator end-to-end latency. Repeated CLI startup deserves investigation if latency remains high; retain recipient identity guards and paste delay.
- Reproduction: disposable git-archive tree `/tmp/pair533-phase-measure` with monotonic timing wrappers around CaptureCrops, waitForCapture, CaptureScreenshot and Store.Prepare. Commands: `WEBCAST_LIVE_CHROME=1 GOMAXPROCS=2 go test -p 1 ./cmd/internal/webcast -run '^TestLiveSubmission(Crop|CaptureReady)$' -count=1 -timeout 60s -v`; `GOMAXPROCS=2 go test -p 1 ./cmd/internal/submission -run '^TestStoreLifecycle$' -count=3 -v`; `GOMAXPROCS=2 go test -p 1 ./cmd/internal/submission -run '^TestMeasurementThreeCropBundle$' -count=1 -v`; `python3 /tmp/pair533-pane-measure.py`. Evidence `/tmp/pair533-phase-measure-chrome.log`, `/tmp/pair533-pane-measure-results.json`. All passed; no operator pane/browser touched.

- Remaining isolated measurements: 4-record/1-note recording read0.085–0.201ms and dom.Build0.011–0.061ms (5 samples, fake capture excluded); local-only replay first mount5.435/5.603/5.604ms with fixture5ms polling, already-ready lookup<0.001ms (production polling50ms, public tunnel startup unmeasured). Real session-log append+commit subprocesses first/cold326.119ms, next4 totals48.281–55.467ms. Reproduction patch `/tmp/pair533-measurement-instrumentation.patch` applies to base7efc4f904ece07c6fa2b554e9002e16dafde11df; additional script `/tmp/pair533-session-log-measure.py` and its results JSON, logs `/tmp/pair533-phase-measure-build.log` and `/tmp/pair533-phase-measure-replay.log`. Measurements satisfy phase decomposition without attributing synthetic fixtures to the operator's exact multi-second experience.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration marked stale, so provisional. Existing Go callback/polling, Neovim APIs and subprocess library suffice; no greenfield library search needed. Backend is one smaller Go extension; UI ownership and DOM deadline propagation are two focused Lua units. Thorough approved plan applies0.2 design discount; v3.1 applies0.4 to implementation; familiar #524 code uses familiarity1.0. Base values: Go0.2/0.5, each Lua1.5/1.0, docs0.1/0.1, review0/0.5, discovery0/0.5 (design/impl hours before discounts).

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.04 impl=0.20
item: lua-neovim design=0.30 impl=0.40
item: lua-neovim design=0.30 impl=0.40
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0 impl=0.20
item: real-api-discovery design=0 impl=0.20
design-buffer: 0.15
total: 2.199
```

- Implemented informational progress over existing polling; authenticated slot label; 36-second draft operation budget and bounded DOM-only external calls. TDD REDs: manager missing initial replay progress, reducer allowed active replacement, adapter lacked deadline, production buffer remained editable. GREEN: affected Go packages (submission/couchmessage/couchtty/couchcmd/recorder/webcast), focused couchtty+recorder race tests, Lua reducer/adapter/transport/sender and production headless chord tests. Embedded Neovim UI verifies actual status frames precede held pin/send, then settling/sent/failure and buffer release.
- Estimate clarification: discovery covers completed Chrome/Zellij measurement; both Lua primitive implementation allowances include controlled and headless verification. Residual design covers choosing budget propagation and UI ownership seams. Full-suite runtime can exceed the provisional primitive allowance; it will be measured rather than back-fit.

- Committed implementation bff86d35c. Stable-source verification: complete `tests/submission-transaction-nvim-test.sh` PASS (including embedded UI); live `TestLiveSubmissionCaptureReady` PASS18.05s and `TestLiveSubmissionCrop` PASS12.82s. These loaded-machine test durations include fixture lifecycle and are not replacements for the isolated phase measurements. Full suite launched to `/tmp/pair533-full-suite.log`, currently waiting under the two-suite machine lock.

- First full suite stopped in the new embedded UI fixture (`/tmp/pair533-full-suite.log`). Captured UI diagnostics reproduced the cause: missing PAIR_AGENT_PICKS_PATH made TextChangedI completion call io.open(nil), entering a hit-enter prompt and blocking the next RPC. The fixture now supplies private picks/output files, strips inherited live-session variables, waits for explicit startup readiness, and triggers TextChangedI to exercise the regression. Full DOM headless/embedded suite passes after this test-only correction; restarting the full suite.

- Boundary-review scheduling: affected checks are green, first full-suite result and fixture correction are recorded, and the corrected full rerun remains queued under the machine limit. Run close review concurrently with that already-launched full suite; landing remains contingent on inspecting its final result. No full-suite pass is claimed.

- Close round1 REWORK (BR-1, operation-ownership-must-guard-mutators): review independently reproduced queued-text overwrite from partial navigation while the buffer is locked. Accepted; appended plan revision to guard all owned-draft mutators before state/file effects and test failure/timeout persistence. Stopped corrected full-suite attempt2 while still queued (exit130, no suite result) before edits; it must restart on stable corrected sources.

- BR-1 correction verified RED→GREEN: production Alt+Right initially raised nomodifiable under held preparation. A buffer-scoped action guard now precedes navigation, autosave, queue/delete/reset, send/finish, quote/image, pending scrollback, slug updates and late doctor clear. Existing image poll defers before consuming evidence files. Four production cases cover draft/queue origin × failure/timeout; assert logical position, both stored contents, no prompts/sends and later navigation/delete/queue/reset recovery. Separate scratch remains editable. Complete DOM headless+embedded UI, queue-send regression, sender tests and diff check PASS.

- Close round2 REWORK on BR-1: navigation regression passes and mutation-check fails as intended; FocusLost scheduled padding still bypassed lock, raising nomodifiable and a hit-enter prompt. Accepted; expanding enumeration to every direct/indirect draft write including focus pad cleanup with embedded event sequences. Full-suite attempt3 stopped while queued, exit130; no full-suite success claimed.
