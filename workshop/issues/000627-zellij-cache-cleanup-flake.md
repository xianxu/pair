---
id: 000627
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '5cf8b0570244a15da05c5d0a3d0ac6e415bdc70f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T18:58:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "283b4f95", done: "a42fcf6c"}
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

- [ ] Record a real-server red control and terminal-state evidence.
- [ ] Make the shared fixture wait for its exact daemon generation before removing owned files, retaining teardown errors.
- [ ] Add delayed-cache-writer/error/generation controls; qualify repeated live/race/package runs, recorded suite, review and landing.

## Log

All 13 shortcut input subtests passed; other live launcher/core tests passed before this teardown failure. The landing CI gate correctly refused #604. No retry or bypass has been used for this failure.

2026-10-16: Verified submitted dispatch b7c6c549-80ff-4fa5-b3cf-2dad5662be74 from ops:0. Loaded tl-worker rules, claimed, started branch from main. Initial start-plan catch-up failed because installed weave lacks --defer-settings; canonical weave refresh advanced the private ariadne substrate and rebuilt its tools, and the retry started successfully (nonblocking old-weave settings-check warning). No implementation changes yet.

Investigating the real daemon lifetime: ControlledZellij.Close deletes the session, closes the PTY and reaps the client, but has no explicit daemon exit join. This is a hypothesis until measured. A scratch overlay logs exact unique-session process rows before/after Close. First 3-run probe could not start: inherited ZELLIJ_SOCKET_DIR points under macOS's long /var/folders path and Zellij 0.45.1 rejected the session name as exceeding its computed limit. Qualification reruns explicitly unset that inherited override, preserving the fixture's existing TMPDIR=/tmp policy; no source workaround added.

2026-10-16: Native 20-run immediate existence probe found the daemon PID after Close; a separate 10-run Darwin kernel-state probe found state 2 (running, not zombie) immediately after Close in all 10. A pre-delete SIGSTOP control was inconclusive: the native CLI blocks on its session probe while the daemon is stopped, so this is not used as regression evidence. Upstream v0.45.1 zellij-utils/src/sessions.rs delete_session sends KillSession on Unix without a daemon exit join, then removes the resurrection directory; missing directory reports exactly Session: NAME not found and exit 2. Source: https://raw.githubusercontent.com/zellij-org/zellij/v0.45.1/zellij-utils/src/sessions.rs . Scratch controls/logs: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair627.34fmyl6i . Small shared-fixture plan posted to hub receipt a8f13a8a-71dc-4340-8cc5-62fd453c1784, proceeding unless objection per tl-worker quick-flow rule.
