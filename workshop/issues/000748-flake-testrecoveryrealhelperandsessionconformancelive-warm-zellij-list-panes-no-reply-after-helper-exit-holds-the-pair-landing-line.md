---
id: 000748
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '5a18e381148a906cb385cc731889dc7862fe6e1a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T17:33:54-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# flake: TestRecoveryRealHelperAndSessionConformanceLive/warm (zellij list-panes no reply after helper exit) holds the pair landing line

## Problem

`couchcore.TestRecoveryRealHelperAndSessionConformanceLive/warm` in the required `couch-zellij-conformance` job fails
intermittently on GitHub macOS: "recovery session state could not be checked; retry: observe live session owner:
zellij action list-panes --json --command returned no reply" (helper exited before recovery; the final query got no
reply). It failed on pair#706's heads twice on 10-20 (run 39501347923 attempt 1; run 39521409698 job 118965862227,
log SHA256 cd823c3894b2ef291a8c4ae4be3b0c70c062aa9ada5fa3d8b436538e393c4f69) and passed on rerun once; #706 touches
no couchcore, launcher or zellij code. Registered in pair#636. Because conformance is required, each occurrence
holds the pair landing line.

## Spec

Root-cause it (no timeout bump, no retry loop): reproduce with the exact CI geometry and timing (a fresh HOME, pinned
Zellij 0.45.1), find why list-panes gets no reply after the helper exits (session teardown racing the query? a
socket that's gone?), and fix the test or the product, whichever is wrong. Record the capture as a fixture.

## Done when

- A deterministic reproduction exists (red before the fix), the fix passes it, and 20 consecutive local runs of the
  case pass; the pair#636 entry is closed with the cause.

## Plan

- [ ]

## Log

### 2026-10-20

### 2026-10-20 — native cause investigation checkpoint

Claimed on pair:3 and started a separate branch at main `127001eae6379ce571b6097f4573deb57ca90416`; #743 remains claimed and its pushed branch preserved. No production or compiled test changes. No image, installation, operator session or machine settings changed.

Downloaded completed job 118965862227 through the GitHub job-log API (the containing workflow is still running, so the run-log command refuses). Raw downloaded log SHA256 `db34d56cac77ff292937babcfeecedc1b944a209751528a2bc34220843cd10e4`; this differs from pair:6's extracted-log hash because the downloaded representation includes the job's timestamps/escapes. The original recovery observer capture is retained at `workshop/evidence/000748/ci-failed-query-capture.json`. Local client exit/helper wait occurred at 00:30:13.694; four pane replies plus two session listings/header-only clients succeeded afterward. The final owner query at 00:30:14.004690 exited 0 after 21.787 ms with zero stdout/stderr and no cancellation. Native log shows the CLIENT's stdout-EIO panic on PTY closure at 13.694; server/plugin shutdown begins only at fixture cleanup (14.049), after the refused query. Helper death or client exit is not evidence of server death.

Fresh-HOME current-main control with installed Zellij 0.45.1: exact park→warm package selection passed once (park 1.50 s; recovery 1.84 s). This sample is not acceptance or a fix. Retained local log `/Volumes/TestScratch/tmp/pair-slot3/pair748-uv7wouog/baseline.log`.

Reconstructed #640's raw-protocol control with the current private-root/startup fixture and fresh HOME, no proxy: 2,000 independent sequential ConnStatus/ListPanes requests, 45 bad FIRST replies including 5 crossed reply kinds; test fails in 1.85 s. Every request is independent; no failed request is retried. Retained source, result and bounded first-response records are under `workshop/evidence/000748/`. Initial diagnostic setup attempt lost an output-path environment variable to the fixture scrub and failed opening an empty path; it is excluded, and the corrected diagnostic uses an explicit owned path. No product behavior was changed.

ARCH-ORDER: pinned v0.45.1 frees the lowest numeric client ID before all duplicate cleanup/queued replies have finished. A later connection can reuse it; old cleanup deletes the replacement's sender, and pending replies can go to another replacement. Source: lib.rs:655–672,1617–1619; route.rs:2676–2678,2941; os_input_output.rs:448–503 at release commit `efd8fd5a89a20c07a111d248ad7fce53848d2c18`. Raw crossed replies establish actual native misrouting, but do not identify every internal instruction in the particular CI failure. The CLI maps premature Disconnect/Unblock to successful empty output (existing #640 protocol controls and launcher empty-reply regressions).

A detach-only change cannot establish the required cause fix: stock CLI actions run ConnStatus discovery even with an explicit session, recreating the cleanup race independently. Serialized native requests already fail. No stock retirement acknowledgement exists. Keeping typed unknown/refusal is correct; replacing it with empty inventory, retries, a sleep or a timeout increase would be wrong.

Verified actual upstream PR5630 server diff (retained): monotonic client IDs and initial ConnStatus fast path before ID registration; latest published release remains 0.45.1. The mixed PR's title/body do not mention these changes, but its actual diff does. Prior #640 operator disposition declined a local Zellij build pending release. A cause fix therefore needs hub scope/dependency-policy guidance before coding: qualify a pinned patched native dependency, or explicitly revise the healthy-native success contract. Current Done-when remains unmet; no fix or deterministic forced native schedule is claimed.
