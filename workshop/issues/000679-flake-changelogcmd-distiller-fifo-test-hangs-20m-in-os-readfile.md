---
id: 000679
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'a619c38bdacc29df3607828b23bcc700615d4ac3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T17:00:30-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "0d4c0481", done: "59201f89"}
---

# flake: changelogcmd distiller FIFO test hangs 20m in os.ReadFile

## Problem

## Spec

TestManagedDistillerLeasesBeforeReadingWithoutTouchingUse (cmd/internal/changelogcmd/retention_test.go) hung until the 20m package timeout during the #674 merge integrated suite (sdlc merge, integration b153e18c, 2026-10-18). Unrelated to #674 (test-only Python fixture change).

Stack: the distiller goroutine is blocked in a raw syscall read inside os.ReadFile(cleanedPath) at changelogcmd.go:100, on the FIFO the test made with Mkfifo; the test goroutine waits on done at retention_test.go:53. The test opens the FIFO O_WRONLY and closes it immediately after seeing the lease, so the reader should see EOF; on darwin it instead stayed blocked, so the open/close rendezvous with the reader open/read is racy.

Evidence: failed log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/5bf0374293571563365a48c27b7f733c.log (copy kept in the pair:2 scratchpad as flake-changelogcmd-674.log). Recorded sdlc-local/suite on b153e18c exit 2 in 27m3s.

## Done when

- The test cannot hang: the FIFO handoff is deterministic (for example the writer writes a byte or holds the write end open until the reader has opened), and a bounded wait replaces the bare done receive.
- 50 consecutive runs of the test pass under -count.

### Proposed correction — 2026-10-23

Quick flow. Preserve the real RunWithEnv-to-os.ReadFile and storage registration path. The test must exchange an actual newline through the FIFO, not use an empty open/close as a completion event. The blank input still takes the empty-transcript path without invoking a model. Check the exact changelog-render lease before admitting the FIFO writer; after completion require release and unchanged LastUse as today.

Run the distiller in an explicitly selected self-exec test child, with the existing disposable storage owner passed literally. The parent owns the child through exactly one Wait, a10s execution context and a2s WaitDelay; every failure closes any writer, cancels only this child and joins Wait before its temp directory can disappear. Open the writer with O_WRONLY|O_NONBLOCK, retrying only ENXIO inside the existing1s admission observation bound. A successful writer open proves a reader endpoint exists; write and check the one-byte newline before closing. All other open/write/close errors fail. No detached goroutine remains to outlive t.Fatal, including early child exit or missing registration. The command receives a test-local fake agent boundary as defense against accidental model dispatch; no real agent, trust, fleet, stage or network is involved.

The correction is the payload handoff. Process containment bounds and joins a regression; merely adding a timeout to the old goroutine would leave the blocked syscall alive and is rejected. A synchronous injected read callback would also make lease ordering deterministic, but adds a production seam solely to replace a native test; retaining the real IO path is the smaller behavioral change. A sleep or longer package timeout has no causal relationship to the missing handoff and is rejected (ARCH-PURPOSE, ARCH-FUNERAL, ARCH-ORDER).

Mechanical prevention: extend Pair's existing golint-backed local catalog with test-fifo-empty-handoff, citing679. It identifies test functions that create a FIFO and directly close an O_WRONLY writer without an intervening nonempty Write/WriteString on that handle. Replay the old test's shape as RED, allow the payload variant and the scrollback sibling, cover import aliases and unrelated non-FIFO files, and run the tracked-source sweep. This is a scoped syntax guard, not a claim of general dataflow or kernel-lifetime verification; it reuses the existing catalog runner and standard suppression semantics (ARCH-DRY).

Scope census: changelogcmd/retention_test.go is the empty writer rendezvous. scrollbackcmd/retention_test.go has the same lease-before-read purpose but already writes captured bytes; it is the positive sibling fixture, with no unrelated rewrite. Other Mkfifo consumers exercise metadata/nonregular-file refusal rather than this empty-writer command handshake. No production Go behavior, persisted format, CLI, deadline, machine setting or local-gates configuration changes. Pure business logic is unchanged; this is native IO test orchestration only (ARCH-PURE, ARCH-EVOLVE). All roots are disposable and all IPC bytes are fixture-owned (ARCH-SECURE, ARCH-MOCK).

Operating envelope: one child and one one-byte payload per invocation, at most one writer handle; one parent Wait owns completion. The10s/2s limits contain a failed test rather than alter a product deadline; normal execution is expected well below1s. Polling stays bounded by the existing1s observation budget, with one owner-state read or one nonblocking open per poll; this is qualification code, no product hot path or N-growing workload. Child stdout/stderr are capped at32KiB each while excess bytes are drained; diagnostics name phase and exit result. The explicit self-exec marker is validated before entering the child path, preventing recursive test launches. No new persistent resources: child, FIFO and owner state die with the joined test and its t.TempDir (ARCH-CONSTRAINTS, ARCH-FUNERAL).

## Plan

- [ ] Implement the payload-carrying FIFO regression through an owned self-exec child, exact lease identity assertions, nonblocking bounded writer admission, and one joined completion/cleanup owner; preserve LastUse and no-model behavior.
- [ ] Pin early child exit and a deliberately stalled child as bounded failure/cleanup cases; prove the child is joined and no registration/artifact assertion is read concurrently with a running producer. Add the empty-handoff lint rule and adversarial matcher coverage, replaying the measured legacy shape.
- [ ] Run50 consecutive focused test invocations, targeted race qualification and the full changelogcmd package plus repolint. Audit any changed affected-reader certificate before its named repair; run inventory before close.
- [ ] Push the tested source, retain the recorded affected/full fallback, then binary-owned close review and sdlc merge; report each real outcome. No stage or model-native qualification is needed for this test-only fix.

## Log

### 2026-10-18


### 2026-10-22 — recurrence in #739 recorded verification

On exact pushed cc7892568e0e91d4cc74516729c5d2d82e8924f0, full recorded attempt7c7510bb47fa2dbaa737a27a12e3a9fc timed out in TestManagedDistillerLeasesBeforeReadingWithoutTouchingUse after20m. The stack matches this issue: test waits on done at retention_test.go:53; distiller blocks in os.ReadFile through changelogcmd.go:100. Remaining packages/scripts passed. Complete suite exit2 after26m13s; failed log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/7c7510bb47fa2dbaa737a27a12e3a9fc.log, SHA2565a13e9973c2201b82f217132c031fa46ea75417479152fb90010ead425a4c130.

No changelogcmd diff versus main. One unchanged focused repeat on that same head: go test ./cmd/internal/changelogcmd -run '^TestManagedDistillerLeasesBeforeReadingWithoutTouchingUse$' -count=1 -timeout=60s -v passed package0.268s/test0.10s. This is focused positive evidence, not a full-suite pass or a registered-flake bypass. SDLC classified the original package timeout as unattributable-failure, so named-test resume does not explain that saved result. Hub asked for the supported next full verification step; no timeout/production fix folded into #739. No process killed; the natural timeout released the suite token.



### 2026-10-23 — Claimed; causal native probe retained

Started the independent679 branch from main1ee96e9872ef195450d5c53f096f530ca7e95ee3 after837's original merge runner exited0. Car PR566 and train PR569 are confirmed MERGED. start-plan catch-up rebuilt local tools against ariadne56172a1009f553fa227d1085f8873c3d636da3ac; behind seeded merge-check workflow was deferred automatically and no local-gates file was edited.

Read the739 retained failure: test waits at retention_test.go:53, reader is in raw Darwin read through os.ReadFile at changelogcmd.go:100, no writer remains. A bounded throwaway Go probe with GOMAXPROCS(1) recreated the same empty open/close handoff and observed the identical raw-read stall at iteration631; its own one-second observation printed the goroutine stack and terminated its process, not a20-minute package run. Only changed the writer to WriteString newline, retaining the same open/read/close scheduling;1000 handoffs passed. This establishes the empty handoff as the causal test mechanism without claiming an unmeasured kernel implementation detail. It is not a full package pass or a passing retry over an unexplained failure.

Evidence directory /Volumes/TestScratch/tmp/pair-slot2/pair679-fifo-probe.5v_wd_ce: main.go SHA2566bd789d4f7cea1fe8675f32f8120d002e3184068c978f70aca64b765b8e51206; empty-handoff.log SHA25611c3cc9930fc33c29f21146eb269594f8e6756829303623bf66ca5f9908a2bba; payload.go SHA256d17730fc3d70ce59ad03996efc350f10931ae339ce97cc06b70e48e8f1ba18d2; payload-handoff.log SHA256d216bb48edbe9ef3a43b3a5ed526fb6dda620e618dfac68f081829c6e2b56457. Existing Go1.27.3 os.ReadFile implementation opens, stats, then reads; successful opposing OpenFile alone does not acknowledge completion of any read. No production source changed during diagnosis.

Next: send the committed plan to ops:0 and wait for approval before change-code. No queued local check, live native fleet, merge runner or stage exists for679.

## Revisions

### 2026-10-23 — Preserve native IO while replacing the empty handoff

The old Done-when's suggested byte/held-writer fix is refined to a checked one-byte payload plus child-process containment and joined cleanup. The original50-run requirement remains. A timeout around the old uncancellable in-process goroutine is explicitly insufficient. This plan is proposed to the hub; implementation has not started.
