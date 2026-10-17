---
id: 000627
status: codecomplete
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '6f766ddadf7093a4eaa7261a32186dcfc4fac407' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T18:58:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 2.22
---

# Live shortcut fixture races Zellij cache removal during TempDir cleanup

## Problem

First observed 2026-10-17T00:28:29Z on pair#604 PR #381 integrated head f8e3e1c7970f0ab6505478e74ec823007d2c3a5e, GitHub Actions run 38986306428, conformance job 117258474451. TestAgentShortcutInputConformanceLive passed every input subtest, then failed during Go TempDir cleanup: unlinkat .../002/Library/Caches/org.Zellij-Contributors.Zellij: directory not empty.

Failed log: /tmp/pair604-integrated-conformance-failure.log. Durable CI evidence: https://github.com/xianxu/pair/actions/runs/38986306428/job/117258474451 . Previous PR-head conformance run 38983105918 passed unchanged clipboard implementation. No repeat of the failed job yet.

## Spec

Investigate and eliminate the isolated real-Zellij fixture cleanup race. TestAgentShortcutInputConformanceLive installs a private HOME and registers fixture.Close, currently ignoring its error. A surviving Zellij writer racing TempDir removal is a hypothesis, not measured causality. Preserve input assertions; do not hide cleanup errors or increase timeouts without evidence. Independent from #604 clipboard behavior; no test fix folded into it.

Capture server PID + kernel start identity from exact --server argv/session ownership before delete-session. After requesting shutdown and joining the client, wait using procutil's complete native table until that generation is absent, replaced or zombie; unknown reads fail rather than imply exit. Only then remove fixture-owned stub files and permit caller TempDir cleanup. A bounded cleanup context replaces today's unbounded subprocess wait. Missing resurrection-cache exit 2 is tolerated only with its exact native diagnostic and confirmed terminal server state; all other shutdown errors propagate. Native daemon reparenting to PID 1 prevents waitpid; the exact kernel process generation is the lifetime resource. ARCH-ORDER/FUNERAL: shutdown before cache deletion; ARCH-DRY: fix the shared fixture rather than duplicate a native process census in each caller. Production behavior and shortcut assertions remain unchanged.

## Done when

- A reproducible cleanup regression identifies the resource still writing and passes after the fixture owns its teardown.
- Repeated isolated/headless runs retain all shortcut assertions and clean their private cache reliably.

## Plan

- [x] Record a real-server red control and terminal-state evidence.
- [x] Make the shared fixture wait for its exact daemon generation before removing owned files, retaining teardown errors.
- [x] Add delayed-cache-writer/error/generation controls; qualify repeated live/race/package runs and recorded suite. Mechanical review and landing remain pending in Log.
- [x] R3: preserve Linux enumeration uncertainty, verify native-reader and both-consumer controls, then qualify a fresh pushed-head suite before re-close.

## Log



- 2026-10-16: closed — Native old-daemon red and original-Close mutation fail; delayed-cache-writer regression passes. Linux actual procfs stat-read and parse regressions fail before correction, pass after; both fixture consumers preserve unknown rows and owned files. Changed packages pass with race detection; final shortcut race count=10 and complete integrated native target pass. Original-fixture native query failure filed unassigned #636 with exact differing actions and controls, retained failure plus one unchanged native-target rerun pass. Recorded integrated-source suite 4ffba844d2cccf57f0fc6eb601d0e8be59f99a94 PASS 11m14s, log sha256 b1cc17b8c66e; all CI including native conformance green. Atlas updated; TL owns project records. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
- 2026-10-16: flow upgraded quick → full — 106 added lines in code files (limit 100)
All 13 shortcut input subtests passed; other live launcher/core tests passed before this teardown failure. The landing CI gate correctly refused #604. No retry or bypass has been used for this failure.

2026-10-16: Verified submitted dispatch b7c6c549-80ff-4fa5-b3cf-2dad5662be74 from ops:0. Loaded tl-worker rules, claimed, started branch from main. Initial start-plan catch-up failed because installed weave lacks --defer-settings; canonical weave refresh advanced the private ariadne substrate and rebuilt its tools, and the retry started successfully (nonblocking old-weave settings-check warning). No implementation changes yet.

Investigating the real daemon lifetime: ControlledZellij.Close deletes the session, closes the PTY and reaps the client, but has no explicit daemon exit join. This is a hypothesis until measured. A scratch overlay logs exact unique-session process rows before/after Close. First 3-run probe could not start: inherited ZELLIJ_SOCKET_DIR points under macOS's long /var/folders path and Zellij 0.45.1 rejected the session name as exceeding its computed limit. Qualification reruns explicitly unset that inherited override, preserving the fixture's existing TMPDIR=/tmp policy; no source workaround added.

2026-10-16: Native 20-run immediate existence probe found the daemon PID after Close; a separate 10-run Darwin kernel-state probe found state 2 (running, not zombie) immediately after Close in all 10. A pre-delete SIGSTOP control was inconclusive: the native CLI blocks on its session probe while the daemon is stopped, so this is not used as regression evidence. Upstream v0.45.1 zellij-utils/src/sessions.rs delete_session sends KillSession on Unix without a daemon exit join, then removes the resurrection directory; missing directory reports exactly Session: NAME not found and exit 2. Source: https://raw.githubusercontent.com/zellij-org/zellij/v0.45.1/zellij-utils/src/sessions.rs . Scratch controls/logs: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i . Small shared-fixture plan posted to hub receipt a8f13a8a-71dc-4340-8cc5-62fd453c1784, proceeding unless objection per tl-worker quick-flow rule.

2026-10-16: Native red assertion FAIL: original Close returned with daemon Darwin state=2. Deterministic portable delayed-writer regression also FAILS with the original Close body replayed byte-for-byte through a temporary overlay: Close returned before observing the cache writer's exit. Current helper passes delayed-writer, terminal/absent/reused/unknown-generation, exact-session, census-error and native-diagnostic controls; owned files are retained when shutdown evidence is unavailable. Missing-cache exit 2 allows exactly the observed native missing-cache line, optionally preceded by the exact active-session diagnostic; arbitrary exit 2 and extra diagnostics remain failures. Cleanup now has a 5s bound instead of the previous unbounded command.

Qualification: original native daemon assertion now passes 10 runs (server absent after Close); corrected live shortcut test passes 50 runs in 53.060s, every input assertion retained. Shared fixture -race count=10 PASS (9.550s); after the final failure-path retention change, -race count=1 PASS (3.028s). Full affected packages PASS: pairlifecycletest 2.388s, couchcmd 142.590s, launcher 97.728s, layoutcmd 0.281s. Exact make -f Makefile.local test-couch-zellij-live PASS in a private HOME/config/data/cache/runtime/socket namespace, using unchanged shared Go caches; namespace manifest native-private-env.json. The original fixture's isolated TestSessionDetachLive control passes 5 runs in 9.002s.

Qualification is NOT complete: first broad native target overlapped the 50-run shortcut qualification in the default /tmp Zellij socket namespace and failed TestSessionDetachLive before initial screen render (10s). Its first three launcher tests passed; subsequent full private-namespace target ran all three packages successfully. A subsequent private-socket shortcut -race count=10 failed two layout-state assertions (alt-left and alt-l), before teardown, with no race detector report or cleanup failure. These observations do not yet prove pre-existing flakes. Per tl-worker's second-unrelated-flake rule, stop for hub classification; propose same checks with original fixture before more suite/retry work. No assertion or timeout has been weakened.

Upstream cache writer is write_session_state_to_disk in https://raw.githubusercontent.com/zellij-org/zellij/v0.45.1/zellij-server/src/background_jobs.rs : it creates session-info directories and metadata from the daemon's background jobs. Native running-daemon evidence and the held late-writer regression identify the lifetime gap; waiting for kernel terminal state prevents further daemon writes.

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/red.log: SHA256 2f24817936b9f4a7967925630b7f6cd2f9a9cb01e26e3aaf13bf460c913a572d

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/old_close.go: SHA256 7a50d5abcce8c807702c47493899440fc2dbcdf4d2ff5fbae1a7ec78e2dbae30

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/old-close-overlay.json: SHA256 edeacc6148e14de95db4ba5f673b2410fe8527c290c82fad58e9a91fe81f05bd

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/green.log: SHA256 b4523d14dc0b2e61e08757199383f35d8d21f9777205a3b1df35c28620f687f3

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/live-target.log: SHA256 bb3718a11b8fc9287b5db3e6c9e8ec90a58c4d5b93a14adc18995ba8db2eba12

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/live-target-private.log: SHA256 7c615e398b576b9b1ede46e5a6e827c18456095d9be9b44048b26650f3e5549b

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/shortcut-race.log: SHA256 5f892cbc106d1a23cfca77163c7da978042e302049ecace10e0dfd7288e39841

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/detach-baseline.log: SHA256 c6a35987499b3a8b56eb41c37f997589021c627adbe9f67368de18ffe72237e6

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/native-private-env.json: SHA256 87f4e679195670ec0950c17b1d7fe5e99188d3023cc284b0a9d6f4f3d0b6fa44

## Revisions
2026-10-16 R1: TL receipt a8c8d7a3-348e-46eb-84ea-386652e8c0aa authorized original-fixture controls and fixing candidate-only failures in #627. Original helper AND original shortcut body, identical private socket root and -race count=10 PASS (30.387s); separately isolated original detach count=5 PASS. Candidate diagnostic race run reproduces the shortcut assertion: initial layout is valid, AFTER is empty despite native exit 0. Extend the test fixture's snapshot readiness to reject empty observations with the existing context/helper, retaining the exact full-layout comparison for the first nonempty response. Never wait for equality or retry input. Add deterministic empty/whitespace-read and immediate-nonempty-change controls. This is a fixture-only qualification refinement; no product behavior or deadlines change. The initial overlapping detach timeout remains unclassified until same-root isolated candidate control completes.

2026-10-16 R1 qualification: original shared fixture plus original shortcut body -race count=10 PASS (30.387s), same private socket root /tmp/p627-race.k8gep9fw. Separately isolated original detach count=5 PASS, then candidate same private HOME/config/cache/runtime/socket roots isolated detach count=5 PASS (6.015s). The first detach startup timeout is not reproduced in either isolated control; its earlier run overlapped other native probes and is retained as a non-comparable qualification attempt, not evidence of a pre-existing flake. TL classification report receipt fbd7c097-f39d-4395-bc56-e14d9a195352.

Candidate diagnostic -race count=10 reproduced the layout assertion with an entirely empty successful AFTER response, not a nonempty layout change. Treated as ours per TL policy. Extracted the existing snapshot read without changing its behavior; the empty/whitespace observation regression FAILED (returned empty after 1 read). The corrected helper uses existing waitShortcutCapture/context only for unavailable observations. It returns the first nonempty response unchanged; a separate control proves a nonempty changed layout is not retried. Both tests -race count=10 PASS (1.633s); final isolated live shortcut -race count=10 PASS (16.426s), preserving every input and exact layout assertion. No deadline increased, input replayed or layout difference ignored.

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/original-fixture-overlay.json: SHA256 af6d30ccf489028e361cf8eca181d52e917ff42da4ef2cb20b579bc89402cb9e

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/shortcut-race-original.log: SHA256 4f167a53c249489752c4243f5a4174ea26b91cc8344140013aa08dceb5e4ddbe

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/detach-original-isolated.log: SHA256 94ede1113b8981b188eb018c859715958964996ca3c989069aa249d30b1989a2

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/detach-candidate-isolated.log: SHA256 4ad95575fc18aa1a29f1d68402fe1b7dfb86b1890541bddefcaaf1131611826b

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/layout-diagnostic.log: SHA256 12c1f4615624bcce6713b2e82c7033d4aa54453ac905d5c475df00788c6763e2

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/shortcut-race-final.log: SHA256 178fc7926103434a5216bf4486b8c5b3db7607e621dc77ba0b84bc725fe9b139

2026-10-16: Final shared-fixture native fullscreen conformance PASS (8.754s package, every named focus/fullscreen scenario executed), log /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/fullscreen-live.log, SHA256 058544d72902f615e0374ca5516fcdeceb457c375472d0424c232e7cff7493fb. Hub classification report fbd7c097-f39d-4395-bc56-e14d9a195352 confirmed submitted. PR https://github.com/xianxu/pair/pull/386 opened on code head b76223d9bd95ab8eabd8a8391cf99e0d5bf615da (Git verified). Recorded suite started via sdlc local-check run --name suite and is waiting for a machine slot, position 8 initially; inflight log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/3ed46c3e7f2ad30891148471a946b560/run.log . Close/review/landing remain pending.

2026-10-16: Recorded full suite PASS on b76223d9bd95ab8eabd8a8391cf99e0d5bf615da, make PAIR_GO_TRIMPATH=-trimpath test via sdlc local-check run --name suite. Execution 14m33s, queue 1413s; full log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/b76223d9bd95ab8eabd8a8391cf99e0d5bf615da/suite.log, SHA256 prefix ae725ec82b5e. couchcmd 154.979s, pairlifecycletest 1.455s, couchcore 522.866s. No failures. All PR #386 CI checks PASS, including native Zellij conformance. Close review and landing pending.

2026-10-16 R2 (administrative): Final plan checkbox now records completed qualification only; close review and landing are explicitly pending gate actions in Log. No design or scope change.

2026-10-16 R3 (proposed; awaiting TL approval): Close review REWORK, Critical BR-1, family unknown-process-read-is-not-exit. Verified Linux Table silently skips non-disappearance stat-read errors and malformed stat rows. Both initial capture and shutdown polling therefore lose uncertainty despite correct handling of returned errors. Proposed correction: preserve listed unreadable PIDs as PID-only unknown rows; omit only positive ENOENT/ESRCH. Existing SnapshotNamed and waitServers contracts then refuse capture or retain liveness respectively. Extract the Linux table reader behind a portable procfs-root test seam; test read/parse failures and disappearance at the reader, and unknown rows at both consumers. Shared procutil changes expand the original fixture-only scope, so implementation waits for TL approval. Simplest alternative, failing the whole host census on any unreadable process, needlessly aborts unrelated discovery; preserve existing per-row uncertainty instead (ARCH-PURPOSE). A fixture lease cannot own an external daemon's cache-writing lifetime. Existing full-suite/native successes remain recorded but do not satisfy the next code head. Mandatory review artifacts retained; no close/landing yet.

2026-10-16 R3 approved: TL receipt 0868ff15-0ae9-46ce-a457-99bcbd613b90 is submitted and verified. Expand into shared procutil because its native Linux enumeration discards the uncertainty required by both fixture consumers. Preserve PID-only rows on non-disappearance read and parse failures; no new row state or public API. Proceed with portable native-reader regression controls, both consumers, fresh suite and ordinary re-close.

2026-10-16 R3 implementation: extracted existing Linux table loop without behavioral change into the existing portable procfs adapter; new native-reader regressions FAILED for both deterministic EISDIR stat-read failure and malformed stat data (Table and SnapshotNamed both lost the listed PID). Corrected enumeration to preserve PID-only unknown rows except ENOENT/ESRCH. Reader regressions now PASS, including disappeared numeric entries and retained zombie generation. Both fixture capture and shutdown-poll controls retain owned files on unknown rows. Focused regressions -race count=3 PASS (procutil 1.549s, pairlifecycletest 1.647s); both full packages -race count=1 PASS (6.904s, 2.156s). Final isolated live shortcut -race count=10 PASS (10.089s), r3-shortcut-race.log in the existing scratch root. Linux amd64 procutil test binary cross-compiles successfully; the actual portable procfs reader and named capture run on this Darwin host, not a claimed Linux execution. Native full target is running; fresh suite/re-close pending. Atlas native-process contract now documents retained unknown rows.

2026-10-16 R3 native qualification: initial private-root target passed launcher (4.692s) and park, then warm recovery failed with ErrZellijNoReply from list-panes before cleanup. Original helper byte-for-byte verified against fc899cd5d40530707ffe2577a2e0d392c8e7dd31. Warm-only original count=10 PASS (34.375s); matching original park+warm sequence count=10 FAIL (47.705s), one warm failure at list-clients with the same ErrZellijNoReply, while immediate diagnostic panes query succeeded. Candidate matching sequence count=10 PASS (48.288s). Actions differ, so evidence establishes the pre-existing unanswered-native-query family, not an identical CLI action or its root cause. Filed unassigned #636 with exact commands, same private roots, sequential controls and hashes. Hub classification receipt 23cc401d-fdd9-4c3e-b1a6-af9c7a36d933 confirmed submitted.

One unchanged full native-target rerun PASS on source head 8847a4c0501a8ea19daf142445f555c29a433ff2: launcher 17.085s, couchcore 4.673s, couchcmd 1.684s; every shortcut case retained. Recorded full suite on that pushed source head remains queued (initial position 7). Review BR-1 fixed; re-close waits for the new suite.

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-live-target.log: SHA256 89d52bd636f2c6694c47f0e3e279ae05bed9efe23d2bc49504513a69e9bd19cf

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-recovery-original.log: SHA256 362b4eeaed6676e8839b52930f1e72fab8bf21aac26932e91cf7925b37d09676

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-core-original.log: SHA256 a4700a0f4aada38e80878549dfd9b03be429ba55509cc1194a3e665d562ec7e8

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-core-candidate.log: SHA256 8c28e75af317e1a7f9064f98d35515133c65fa617252227710052a9a47cbdc46

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-shortcut-race.log: SHA256 54f57c0bd67c9fbe90e48cda2164ca39859a0f5740e2f9ed871a10d2eb979ec9

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/r3-live-target-rerun.log: SHA256 7c20dfffaade908a9aef752674002783aa449d63d48b209ad6c23f09b8d6153a

2026-10-16 integration: GitHub marked PR #386 DIRTY, preventing its ordinary merge/native CI jobs. Cancelled only our queued (not executing) source-head suite after 332s waiting; exit 130 records intentional cancellation, not a test failure. Merged origin/main cleanly (no manual conflict resolution), Git-verified integrated source head 4ffba844d2cccf57f0fc6eb601d0e8be59f99a94, pushed via sdlc pr. Updated main includes test environment isolation and title lifetime work; #627 code unchanged. Environment guard and both changed packages PASS (testpair 5.706s, procutil 0.731s, pairlifecycletest 1.825s). Complete integrated native target PASS: launcher 12.110s, couchcore 4.161s, couchcmd 1.761s, all shortcut cases retained. GitHub merge/native jobs now run. New recorded full suite pinned to integrated source head queued at initial position 10; inflight 5827c51405d91edee78048b1e686ae7c. No second full suite was executed for the superseded head.

- /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i/integrated-live-target.log: SHA256 f655d265417da705f5ae94c9a7f39cceff4f1d752d0001eb5f0c4b1b0556ad31

2026-10-16 R3 final verification: recorded integrated-source full suite PASS on 4ffba844d2cccf57f0fc6eb601d0e8be59f99a94 in 11m14s, exit 0, log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/4ffba844d2cccf57f0fc6eb601d0e8be59f99a94/suite.log, SHA256 b1cc17b8c66e1cd138bbf1868cfb5957d93dd80c77bbc62a52304a8285e1a14a. All GitHub CI checks PASS on that source head, including native conformance (2m22s) and merge-check (1m26s). Current subsequent branch change is issue evidence only. Re-close review and landing remain pending.
