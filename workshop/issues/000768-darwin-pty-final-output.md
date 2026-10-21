---
id: 000768
status: codecomplete
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours: 1.3
collaboration-mode: collaboration
card_mirror: '543405ba45f3489f6d57fce2f04e0fc364de8ef9' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T05:20:49-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 2.65
---

# Preserve short-lived command output across Darwin PTY slave close

## Problem

Public pair scribe -log PATH -- CMD [ARGS...] accepts arbitrary commands (docs/pair-scribe.md; cmd/pair-go/main.go dispatches directly to scribecmd.Run), including short-lived commands that never open their controlling terminal. On macOS these can exit and lose unread PTY output before the first master read. #766 reproduced 10 empty-stdout results in 10000 runs; an instrumented wrapper copy saw 41/5000 first reads return EOF with zero bytes while the child exited successfully. Scribe already drains before cmd.Wait; this is not fixed by waiting longer or retrying reads after EOF.

XNU's last-slave-close path can discard unread output without a controlling-terminal reference. Opening /dev/tty in the child establishes S_CTTYREF and causes exit to drain. Ruby's primary investigation documents the same mechanism and links the kernel paths: https://bugs.ruby-lang.org/issues/20682 ; Apple source: https://github.com/apple-oss-distributions/xnu/blob/main/bsd/kern/tty_tty.c . #766 fixes the test producer only, as approved by the TL; this production exposure remains.

## Spec

On Darwin, preserve final output for admitted short-lived non-terminal-aware scribe commands by retaining one parent slave through master drain, releasing it before the sole child wait. Keep direct exec, argv, child status, signals and terminal behavior; other platforms retain upstream startup. Initialization abort kills the owned direct child and closes master/slave before waiting, preserving the initialization error. Darwin session-leader exit must end the drain even with a descendant holding the slave and ignoring SIGHUP; scribe returns its direct child's result without waiting for descendants. No timeout increase or blind retry. Scope is scribe only; wrap and ptychild were audited and their different termination semantics are tracked in #769/#770.

## Done when

- A deterministic native regression forces completed writes and closure of all child-owned slave descriptors before the first master read, then verifies exact stdout/log bytes, EOF and status.
- A descendant retaining the slave, including ignored SIGHUP, cannot prevent deterministic scribe completion after direct-child exit.
- Short-lived non-terminal-aware children and normal interactive shell/agent behavior pass, including child exit status, signals and cleanup ownership.
- The design records which Pair PTY entrypoints are covered and any remaining exposure.

## Plan

Approved design: [durable plan](../plans/000768-darwin-pty-final-output-plan.md). TL approved full flow; change-code passed. One atomic review boundary.

- [x] Approve Darwin-only retained slave for scribe, scope, native acceptance clarification and full flow.
- [x] Pin delayed-first-read loss with an out-of-band child stdio-close barrier; implement scribe-local start/lease ownership.
- [x] Qualify normal drain, abort cleanup, interactive behavior, exit/signals and resource lifetimes.
- [x] Document audited remaining exposures, qualify the public built command and enqueue the recorded suite; submit the verified implementation to close review. Passing final-code suite evidence remains mandatory before landing.

## Log

Discovered while investigating #766. Public scribe command is direct evidence of production admission; the shipped documented shell hook normally wraps a long-lived /bin/zsh, whose slave reference protects its ordinary subprocesses. Explicit scribe printf and final shell exec remain exposed. File unclaimed per TL instruction; do not expand #766's approved fixture-only scope.


### 2026-10-21 — planning checkpoint (pending approval)
- 2026-10-21: closed — Pinned symlinked cwd reproduces the old lexical-oracle false failure; directory-identity fix passes the scribe package with ordinary and symlinked TMPDIR/GOTMPDIR, race (9.004s), vet and diff checks. Reader certificate regenerated. Production unchanged; recorded suite on pushed fix is queued/running, final pass required before landing per tl-worker overlap rule. (fixed-then-ship); review verdict: SHIP
- 2026-10-21: closed — Native delayed-first-read RED/GREEN and immediate-release mutation prove exact final stdout/log, EOF and exit0/7; production run returns with ignored-HUP descendant holding slave; abort-kill mutation catches hangup-only cleanup; native large-output/signals/resize/tty restoration and N8/N16 leases pass. Targeted artifact/repolint, race, vet and Linux cross-compile pass. Public clean-head binary smoke passes short/nonzero/empty/argv/Ctrl+C and exact log/status/tty restore. Per tl-worker queued-suite overlap, recorded suite lease-1672737506 on4f4cdc15 remains queued, final-code pass required before landing; hub real-shell/stage coordination requested. CI on code head green.; review verdict: SHIP

Audited all three production launch owners: scribe, wrap's runProxy/startNativePTY, and ptychild.Start (Couch/term). Recommend retaining one parent slave on Darwin through scribe's existing output drain, releasing before the sole child wait. Preserve original exec.Cmd/direct startup and synchronous executable failures; no helper protocol, retries or deadlines.

Native planning probe forced child stdio closure before first master read: unprotected baseline lost final-marker; child /dev/tty and parent retention each preserved it. The retained variant reached EOF while retained and exited7. This is feasibility evidence, not automated acceptance.

Do not generalize this patch: wrap uses EOF to stop a still-live process group, so retention changes a target that closes stdio then keeps running. ptychild exposes EOF as input-ended before reap/publication. Both remain exposed and need separate lifetime designs; file unclaimed follow-ups only after TL approves scope. ARCH-PURPOSE/DRY shaped the scribe-local choice; ARCH-STATE/FUNERAL require explicit lease release and one wait owner. Abort must close the master before joining wait, because exit may be blocked draining unread output.

## Revisions

- 2026-10-21 (proposed, pending TL approval): choose scribe-only Darwin retained-slave ownership after auditing other production owners. Original Spec remains the scope contract. Clarify the first Done when, originally “write and exit before the first master read”: require finished writes and closure of all child-owned slave descriptors before the first read, then exact stdout/log, EOF and correct reap status. A protected child may be in kernel exit-drain, so requiring waitpid completion before reading would deadlock. The durable plan records both alternatives, remaining exposures and qualification. No production/test edits yet.

### 2026-10-21 — TL approval

TL approved full flow, scribe-only retention and acceptance clarification (receipt a711dfa3-61f8-4c22-847e-25caf4a69a63). Added explicit descendant-held-slave contract to the plan: Darwin session-leader exit revokes the terminal, scribe drains to EOF and reaps its direct child without waiting for descendants. Native planning probe observed exact final-marker and EOF while an ignored-SIGHUP descendant still held the slave; direct child exited7. Qualification must prove this through production Run; failure is a stop/re-plan, not a new timeout. Separate unclaimed wrap/ptychild issues requested.

Filed unclaimed follow-ups #769 (wrap) and #770 (ptychild), each preserving probe provenance and the owner-specific termination-semantics risk.

### 2026-10-21 — plan review round1

PQ-1/PQ-2 addressed: initialization abort explicitly kills the owned direct child, closes master/slave before the sole wait, treats already-exited as benign and preserves the initialization error. The plan names startPTY, releaseSlave, run and abortPTY with adversarial test strategies rather than case inventories. Normal completion still waits for the direct child without stopping live closed-stdio targets.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; calibration source reports stale, so provisional. Two smaller Go modules cover the start/lease adapter and run/abort integration. Design bases0.2/0.3 receive thorough-spec factor0.2; implementation bases0.4/0.5 receive v3.1 factor0.4. Native Darwin qualification is one real-API discovery base0.6 ×0.4; docs and review use design0.1 ×0.2, impl0.1/0.3 ×0.4. creack/pty already supplies allocation; no greenfield allocation needed. Familiarity1.5 accounts for bounded unfamiliar kernel lifetime behavior; +15% design buffer. Computed1.301, rounded1.30 hours.

```estimate
model: estimate-logic-v3.1
familiarity: 1.5
item: smaller-go-module design=0.04 impl=0.16
item: smaller-go-module design=0.06 impl=0.20
item: real-api-discovery design=0 impl=0.24
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0.02 impl=0.12
design-buffer: 0.15
total: 1.30
```

### 2026-10-21 — implementation proof checkpoint

Full-flow change-code passed (plan round3, no open findings; estimate-quality INFO). Go-module estimate items include their native fixtures; docs/review items include audit certificates, public smoke, recorded checks and landing, with qualification/landing duration remaining uncertain per advisory.

Observed native RED before adapter: both exit0/7 lost all final bytes after the out-of-band child stdio-close barrier. Retained start passed. Production run integration now verifies exact stdout/log and child status after the same barrier. Disposable immediate-slave-close mutation failed both production cases with empty stdout/log and was restored; transcript /tmp/pair-768-immediate-release-mutation.log. Native run also returned exact bytes/status while ignored-SIGHUP descendant held a slave after leader exit. Abort tests cover live HUP-ignoring child and exit-drain, requiring reaped direct child and closed master/slave while retaining MakeRaw failure. Large concurrent output, signal results and command identity checks pass; race/vet and remaining verification in progress.

Targeted verification: go test ./cmd/internal/scribecmd ./cmd/internal/artifactpath ./internal/repolint passed; scribe race run passed, vet passed. Reader audit certificate covers only generated local log/fixture reads. Three new production files classified NonArtifactSources; inventory preflight caught and resolved those omissions before the recorded check. Interactive native fixture observes Run’s initial resize and verifies outer terminal mode restoration. N8/N16 retained descriptors remain valid before drain and are closed after join; stateful concurrent release fake requires exactly one close and stable error. No helper processes remained after the run.

Public binary smoke passed on clean source revision4f4cdc15f34ef737628ebd207869c739cb489942 (build metadata checked; binary SHA256 f21d65d5891e145aed90e907a3eb5ab742b3a4847041ce01ee4f37f9e45e2e02): short/nonzero/empty/literal argv cases preserve exact stdout/log/status and outer tty; terminal Ctrl+C reaches a shell trap, returns23, logs exact bytes and restores tty. Transcript /tmp/pair-768-public-smoke.log. Non-Darwin branch cross-compiles for Linux; this is compile coverage, not native Linux qualification.

Second mutation removed direct-child kill on initialization abort: the blocked HUP-ignoring target failed the native watchdog oracle after master close alone; exact owned child/master were cleaned, source restored byte-for-byte. Transcript /tmp/pair-768-abort-mutation.log. No native helpers remained. PR521 open; local-gates guard reports no guarded paths touched. Recorded full suite on4f4cdc15 is queued, retained runner lease-1672737506/session62442; do not duplicate it.

### 2026-10-21 — contract reconciliation

Reconciled current Spec and Plan with the TL-approved retained-slave design and clarified acceptance; original alternatives remain recorded in prior planning history/Revisions. CI on the code head is green, including macOS conformance and the local-gates guard (no guarded paths touched). The recorded full-suite runner is still queued; no replacement or duplicate run.

Final documentation sweep removed the remaining package-comment reference to the retired standalone shim. This comment-only code-file edit changes no runtime behavior; retain the existing recorded runner on4f4cdc15. Integration verification will include the final source.

### 2026-10-21 — queued-suite review overlap

Current tl-worker contract explicitly permits close review to overlap a queued recorded suite, with a final-code pass before landing and requeue on review code fixes. Entering close on that basis while retaining lease-1672737506. Suite has not passed yet. Public native smoke is complete; requesting hub coordination for a staged real-shell check per LIVE CHECKS/PERMISSIONS before touching pair:0.

### 2026-10-21 — live acceptance and SHIP

TL accepted native smoke as sufficient live acceptance for scribe-only scope and explicitly forbade staging pair:0 (receipt831cd1bc-0878-49e8-bc89-b827e7d0659a); pair:0 was not touched. Added built public-command descendant smoke: target forks a descendant that ignores SIGHUP and holds slave stdio behind a private FIFO; scribe returns exact public-descendant-marker stdout/log and exit7 while that descendant remains alive, and restores outer tty. Exact owned descendant cleaned afterwards. Transcript /tmp/pair-768-public-descendant.log. This public smoke does not force the internal first-read barrier; the native production-run unit regression separately does.

Full close review returned SHIP with zero findings (round1); independent reviewer reran native/race/vet/artifact/lint/inventory checks. Recorded full suite remains queued on4f4cdc15; enter landing flow promptly per tl-worker, with final-code/integration pass still required before merge.

### 2026-10-21 — train hold and cwd oracle correction

Train234 held this car after its combined candidate failed broadcast TestHubRandomInterleavings and the main baseline and preceding #765 prefix passed. This is registered flake #771, unrelated to scribe; no PTY regression was observed. Hub receipt7a0df4d8-b847-4930-9093-16c1814b9708 directs folding the previously deferred test-only cwd fix into this held car, narrow re-close, required suite and re-entry. A second #771-only refusal must be reported as a possible bisection gate defect.

Pinned an explicit symlinked cwd in TestDarwinStartPreservesCommand. The old lexical oracle failed deterministically: shell PWD reported the physical directory while cmd.Dir retained the symlink. The corrected oracle preserves exact literal argv/environment prefix and cmd field checks, then requires os.SameFile on the reported and requested directories. Full scribe package passes with ordinary and symlinked TMPDIR/GOTMPDIR; vet passes. Reader certificate regenerated. Production behavior and approved contract are unchanged. ARCH-PURPOSE: cwd identity is the promise; lexical cmd.Dir preservation is checked separately. No blanket path-equality lint rule fits both requirements.

### 2026-10-21 — corrected verification and train238 environmental refusal

Narrow re-close returned SHIP with zero findings, bound to evidence a273790e0965da908ccf934211be9ba9965f27a4. Corrected code b014d464b44e189062c9ca38e5764cc8feca048c passed its recorded full suite in10m29s (log SHA256 a28fab0b546973db4914fa84b3431be0431cc2e06329da102b67e6297e848635), race9.004s, vet and all20 reader-audit tests. GitHub missed the close-evidence push during its outage; the hub explicitly directed closing/reopening PR521. That resynchronized its head to348a94ce4b6e3470018ea110e0538a49db2ba922 without code changes or a replacement runner.

Train238 composed only this car; #769 and #771 were held at build. Its integration source3279e1e9ec6ccbddba0b50193f5c8f72c11f4b22 completed the full suite in10m47s with only cmd/couch.TestBareCouchInstalledCommand failing: PTY fixture creation refused terminal 📁1-1 as live/contested. All other packages, including broadcast and scribe, passed. The entire parent test passed twice on the unchanged integration source (11.305s and7.206s); second transcript /tmp/pair-768-bare-couch-rerun.log includes the source HEAD. Filed unclaimed #778. Initial subtest registration did not cover the parser's explicitly coarsened parent attribution; registered the proven parent instead and removed the redundant subtest registration. The fixed terminal basename and session-only process census are a source-level lead, not a forced causal reproduction.

The conductor had already queued a main baseline. That baseline failed immediately at runtimebundle generation because /Volumes/AgentScratch/go-build had no space (log SHA256 prefix5ccc7602a29c); it ran no tests and does not establish main health. Train238 refused at12:45, train PR529, with no #768 landing. Retained merge runner exited1; nothing was canceled or duplicated. PR521 remains open and the code is unchanged. Hub and waiting peers were informed. Next: repair scratch capacity, then re-enter the official merge flow with #778 registered; do not infer a main code regression from this baseline.
