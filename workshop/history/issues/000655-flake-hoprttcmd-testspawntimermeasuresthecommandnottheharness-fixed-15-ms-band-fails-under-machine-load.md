---
id: 000655
status: done
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'c4900f8b0014ac95eb5c7691e69157151c87b310' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T17:35:46-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "e237cd10", done: "90c1f2dc"}
actual_hours: 0.36
---

# Flake: hoprttcmd TestSpawnTimerMeasuresTheCommandNotTheHarness fixed 15 ms band fails under machine load

## Problem

Flake in the recorded pre-merge suite, 2026-10-17: `cmd/internal/hoprttcmd` `TestSpawnTimerMeasuresTheCommandNotTheHarness` failed on #620's integrated head 7c96d30f (PR #407):

    hoprtt_test.go:69: fork+exec median 16.935 ms is outside the plausible band; the harness is measuring itself, not the command

- The test gives a fork+exec of /usr/bin/true a fixed 15 ms ceiling. Its comment budgets for the suite's own spawn load (about 4.5 ms), not for whole-machine load. At the time the machine's load average was about 10 now, 11.5 over 5 min and 18 over 15 min: two full suites plus fleet slots running.
- #620 doesn't touch hoprttcmd, and the main baseline passed, so this is the band failing under machine load, not a code change.
- Log retained: /Volumes/AgentScratch/tmp/pair-618-share/suite-7c96d30f-hoprtt-flake.log (sha256 prefix f1d958641097; original at .git/sdlc/test-stats/failed/e9a569fffa42f6450a9b09fb08569c4b.log).

## Spec

Make the positive control independent of machine load while still catching the bug it pins (a harness that times a clock subprocess, about 18.7 ms against about 1.9 ms). Options:
- Compare against a same-run baseline: time a trivial in-process operation, or measure the harness overhead directly, and assert the ratio rather than an absolute ceiling.
- Or assert on the harness structure (no subprocess clock read) with a fake clock, and keep the live timing only as a soft log.

Selected approach: extract only the command-preparation/run boundary from `spawnRTT` into a shared internal sampler. Keep its existing `time.Now`/`msSince` timer and real `exec.Command(...).Run` adapter. Replace the fixed-band positive control with `testing/synctest`: generated preparation delays (outside the sample) and command durations (inside), exact per-sample durations, invocation order and failure counts. This tests the actual production loop without a machine-speed assumption (ARCH-DRY/PURE/ORDER). Retain a shipped-binary spawn smoke that requires positive, ordered measurements but no absolute latency ceiling. A separate same-run real-command baseline still has scheduling skew; a source-text check would be brittle and weaker than the controlled timer boundary.

## Done when

- The test can't fail from machine load alone. Checked with a load generator running alongside it, for example 20 parallel `yes > /dev/null` processes.
- Reverting to a subprocess-clock harness still fails it.

## Plan

- [x] Add the exact virtual-time boundary regression and real-binary smoke; demonstrate timer-around-preparation, omitted-command and external-clock mutants fail.
- [x] Run focused/race checks, the bounded owned load check and full suite before close; complete status publication through the normal merge gates.

## Log

### 2026-10-17
- 2026-10-17: closed — Virtual-time regression asserts exact command-only samples across 0/5s/1h preparation delays, order and failure counts; setup-timing, omitted-command and subprocess-clock mutants fail. Package race passed 5.087s; 1000 runs under 20 owned CPU workers plus deliberate child panic proved cleanup, zero workers left, 0.688s total. Full pre-close suite attempt 1f9b1fc91cdd4c2f914efebc9cf760db passed all tests on ad7bc8a9eca6c5f141ecca3a3495d90528e1a5c8 in 14m16s, retained log SHA 7a660a7d1e9c; only GitHub success-status posting failed. Hub directs normal sdlc merge rerun/record, no hand-post or gate bypass; ariadne#431 tracks recovery. PR417 CI green; atlas updated; no operator UI change. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

- Verified dispatch `a9e1b2e9-32c9-4479-bc2c-911d5eae55c8`, claimed before design, and ran start-plan. First attempt refreshed the base binary and refused before starting; reran as directed. Verified retained failure log SHA256 `f1d9586410976a53db635894d2fdb3894b3ada0b30f16338409f627f4cf3113c`: the unmodified command reports median 16.935ms against the fixed 15ms ceiling. The production timer directly surrounds `c.Run`; the faulty inference is the test treating scheduling delay as a clock-process bug.
- Quick-flow plan: expose a local prepare-to-run function seam, with no public API, new state or global hooks. A deterministic virtual-time fake tracks preparation/run events across samples and failures. Assert exact command-only durations for varied preparation delays, including command durations above the old ceiling. Negative controls move the timer before preparation, omit execution, and replace its clock with a subprocess wall clock. Keep native shipped-binary reachability/output smoke, without speed certification. No runtime behavior, CLI, docs surface, persistent artifact or environment authority change; existing `testpair` owns native binary isolation.
- Proposed load verification: compile the targeted package test once, then run the deterministic regression repeatedly while 20 owned CPU workers run for at most 3 seconds; a scoped helper terminates and waits for every owned worker on all exits. This is a brief load check, not a timeout increase or passing flake retry. Required full suite remains machine-gated. No operator-environment live smoke or :0 staging applies to this internal measurement-test seam.

- Verified Submitted approval `385f99bc-af71-416f-bccb-5ddd666ec98c` accepts the virtual-time plan and 3-second/20-worker check, requiring cleanup even on test panic. Quick-flow change-code admitted implementation. New regression first failed because the sampler seam was absent (`/tmp/pair655-red.log`); then the existing loop was extracted without changing timer scope or failure semantics.
- `TestSpawnTimerMeasuresTheCommandNotTheHarness` now uses actual `time.Now`/`msSince` under synctest, not a replacement clock callback. Across setup delays 0/5s/1h it requires exact 0.25/17/100ms command samples, one sequential prepare/run per sample, and the controlled failure count. The production adapter still supplies `exec.Command(...).Run`. Real shipped-binary smoke asserts positive ordered percentiles without a speed ceiling; output was 4.773/7.155/8.399ms in the race run.
- Negative controls all failed as required: timer including preparation reports 5000.25ms instead of 0.25ms; omitting Run breaks the observed event order; Python subprocess clocks report their own real 29–44ms instead of the controlled durations (`/tmp/pair655-mutant-{setup,omit,external-clock}.log`). Original production source restored before green.
- Package race check PASS 5.087s (`/tmp/pair655-package-race.log`). Approved load check: precompiled test ran 1000 times under 20 live owned CPU workers and PASSED; a subsequent deliberately forced child test-timeout panic exercised the same cleanup path. All 20 workers killed and reaped, zero live, total elapsed 0.688s (under the 3s cap). Every worker also carried its own 3s alarm in case the helper died; helper SIGINT/SIGTERM exit through finally. Logs `/tmp/pair655-load-{green,panic}.log`. No full-suite failure/retry occurred; this was an expected negative cleanup control.

- Required recorded pre-close suite attempt `1f9b1fc91cdd4c2f914efebc9cf760db` queued on pushed ad7bc8a9eca6c5f141ecca3a3495d90528e1a5c8. Verified fresh Submitted hub approval `18f9d853-9b1a-444f-8a07-c5d2e2a100eb` authorizes an early draft PR for CI overlap, and makes this a standing allowance while suite, close review and merge gates still precede landing.

- Early draft PR https://github.com/xianxu/pair/pull/417 has all server checks green. Pre-close suite attempt `1f9b1fc91cdd4c2f914efebc9cf760db` completed every test successfully on pushed ad7bc8a9eca6c5f141ecca3a3495d90528e1a5c8: exit 0, not timed out, 855.752s execution (14m16s), 950.755s queue, log SHA256 `7a660a7d1e9cbe7fb9b2ba16f35fb021722024b835954e5d6e52afad6fb8c3ff`. Rehashed retained log against the saved attempt. hoprttcmd passed in 0.577s. No test failed. The runner returned an error only because GitHub was unreachable while posting the success status; readback shows it was not recorded.
- SDLC exposes no status-publication recovery and saves the full attempt before PostStatus. Proposed narrow manual status replay to the hub; verified Submitted instruction `13ed420d-07be-4f8e-8847-b92701f2f4f4` refuses hand-posting and directs normal `sdlc merge` to rerun. Existing ariadne#431 owns publication replay, so no peer issue filed. No status was posted manually. Close reviews the actual passing pre-close test evidence; merge must obtain a normally recorded suite before landing.

## Revisions

- 2026-10-17 — Select virtual-time command-boundary assertions over a relative live baseline: independent scheduling skew can invalidate the latter. Preserve the original failure evidence and both original Done-when requirements; brief owned load check and external-clock negative control remain required. Plan posted before code; awaiting hub response.

- 2026-10-17 — Status-publication network failure: the full suite passed but its GitHub status did not publish. Per hub instruction, retain this attempt, close against its real test evidence, and let normal merge rerun/record the required suite. Checklist distinguishes completed verification from pending landing publication; no gate bypass or manual status.
