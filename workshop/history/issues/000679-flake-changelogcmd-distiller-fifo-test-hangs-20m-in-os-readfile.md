---
id: 000679
status: done
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9d1651dd33ac3c379779c3d0b385cb69aa174919' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T17:00:30-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "0d4c0481", done: "59201f89"}
actual_hours: 0.20
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

Scope census: changelogcmd/retention_test.go is the empty writer rendezvous. scrollbackcmd/retention_test.go has the same lease-before-read purpose but already writes captured bytes; it is the positive sibling fixture, with no unrelated rewrite. Other FIFO uses include payload readiness barriers (couchcore/ptyrunner_test.go), deliberately held stdin lifetimes (couchcmd/reap_orphans_live_test.go), and metadata/nonregular-file refusal fixtures. This correction concerns an immediate empty writer-close handshake, not those distinct IPC contracts. No production Go behavior, persisted format, CLI, deadline, machine setting or local-gates configuration changes. Pure business logic is unchanged; this is native IO test orchestration only (ARCH-PURE, ARCH-EVOLVE). All roots are disposable and all IPC bytes are fixture-owned (ARCH-SECURE, ARCH-MOCK).

Operating envelope: one child and one one-byte payload per invocation, at most one writer handle; one parent Wait owns completion. The10s/2s limits contain a failed test rather than alter a product deadline; normal execution is expected well below1s. Polling stays bounded by the existing1s observation budget, with one owner-state read or one nonblocking open per poll; this is qualification code, no product hot path or N-growing workload. Child stdout/stderr are capped at32KiB each while excess bytes are drained; diagnostics name phase and exit result. The explicit self-exec marker is validated before entering the child path, preventing recursive test launches. No new persistent resources: child, FIFO and owner state die with the joined test and its t.TempDir (ARCH-CONSTRAINTS, ARCH-FUNERAL).

## Plan

- [x] Implement the payload-carrying FIFO regression through an owned self-exec child, exact lease identity assertions, nonblocking bounded writer admission, and one joined completion/cleanup owner; preserve LastUse and no-model behavior.
- [x] Pin early child exit and a deliberately stalled child as bounded failure/cleanup cases; prove the child is joined and no registration/artifact assertion is read concurrently with a running producer. Add the empty-handoff lint rule and adversarial matcher coverage, replaying the measured legacy shape.
- [x] Run50 consecutive focused test invocations, targeted race qualification and the full changelogcmd package plus repolint. Audit any changed affected-reader certificate before its named repair; run inventory before close.
- [x] Prepare the pushed-source recorded affected/full fallback and binary-owned close handoff; keep actual check, review and landing outcomes in the Log. No stage or model-native qualification is needed for this test-only fix.

## Log

### 2026-10-18


### 2026-10-22 — recurrence in #739 recorded verification

On exact pushed cc7892568e0e91d4cc74516729c5d2d82e8924f0, full recorded attempt7c7510bb47fa2dbaa737a27a12e3a9fc timed out in TestManagedDistillerLeasesBeforeReadingWithoutTouchingUse after20m. The stack matches this issue: test waits on done at retention_test.go:53; distiller blocks in os.ReadFile through changelogcmd.go:100. Remaining packages/scripts passed. Complete suite exit2 after26m13s; failed log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/7c7510bb47fa2dbaa737a27a12e3a9fc.log, SHA2565a13e9973c2201b82f217132c031fa46ea75417479152fb90010ead425a4c130.

No changelogcmd diff versus main. One unchanged focused repeat on that same head: go test ./cmd/internal/changelogcmd -run '^TestManagedDistillerLeasesBeforeReadingWithoutTouchingUse$' -count=1 -timeout=60s -v passed package0.268s/test0.10s. This is focused positive evidence, not a full-suite pass or a registered-flake bypass. SDLC classified the original package timeout as unattributable-failure, so named-test resume does not explain that saved result. Hub asked for the supported next full verification step; no timeout/production fix folded into #739. No process killed; the natural timeout released the suite token.



### 2026-10-23 — Claimed; causal native probe retained
- 2026-10-23: closed — Native empty-FIFO handoff reproduced the retained raw ReadFile stall at iteration 631; newline-only causal contrast completed 1000 handoffs. Real RunWithEnv/ReadFile now runs in one owned self-exec child, checked newline payload, exact child identity and changelog-render lease before writer admission, unchanged LastUse and release after the sole joined Wait. Early exit, bounded missing reader, permanent open errors, cancellation with a proven reader and held empty writer, and 32KiB-per-stream diagnostic cap are pinned. Legacy empty-handoff lint RED, 25 matcher cases plus filename catalog binding GREEN; tracked repolint PASS15.030s. Final 50 consecutive managed-distiller runs PASS6.967s; targeted race three repeats PASS9.300s plus strengthened cancellation race PASS1.501s; complete changelogcmd PASS3.629s. Atlas and lessons updated; make test-affected-inventory PASS immediately before close after one individually audited literal/AST reader certificate. PR572 pushed; recorded affected selected required full fallback lease-2867141082 on2b4a4443433cb567d0499c23ac9ab443b0b2ad26 and is queued, no suite pass claimed; landing awaits it. No production Go, model UI, product deadline, trust or local-gates change. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

Started the independent679 branch from main1ee96e9872ef195450d5c53f096f530ca7e95ee3 after837's original merge runner exited0. Car PR566 and train PR569 are confirmed MERGED. start-plan catch-up rebuilt local tools against ariadne56172a1009f553fa227d1085f8873c3d636da3ac; behind seeded merge-check workflow was deferred automatically and no local-gates file was edited.

Read the739 retained failure: test waits at retention_test.go:53, reader is in raw Darwin read through os.ReadFile at changelogcmd.go:100, no writer remains. A bounded throwaway Go probe with GOMAXPROCS(1) recreated the same empty open/close handoff and observed the identical raw-read stall at iteration631; its own one-second observation printed the goroutine stack and terminated its process, not a20-minute package run. Only changed the writer to WriteString newline, retaining the same open/read/close scheduling;1000 handoffs passed. This establishes the empty handoff as the causal test mechanism without claiming an unmeasured kernel implementation detail. It is not a full package pass or a passing retry over an unexplained failure.

Evidence directory /Volumes/TestScratch/tmp/pair-slot2/pair679-fifo-probe.5v_wd_ce: main.go SHA2566bd789d4f7cea1fe8675f32f8120d002e3184068c978f70aca64b765b8e51206; empty-handoff.log SHA25611c3cc9930fc33c29f21146eb269594f8e6756829303623bf66ca5f9908a2bba; payload.go SHA256d17730fc3d70ce59ad03996efc350f10931ae339ce97cc06b70e48e8f1ba18d2; payload-handoff.log SHA256d216bb48edbe9ef3a43b3a5ed526fb6dda620e618dfac68f081829c6e2b56457. Existing Go1.27.3 os.ReadFile implementation opens, stats, then reads; successful opposing OpenFile alone does not acknowledge completion of any read. No production source changed during diagnosis.

Next: send the committed plan to ops:0 and wait for approval before change-code. No queued local check, live native fleet, merge runner or stage exists for679.

## Revisions

### 2026-10-23 — Preserve native IO while replacing the empty handoff

The old Done-when's suggested byte/held-writer fix is refined to a checked one-byte payload plus child-process containment and joined cleanup. The original50-run requirement remains. A timeout around the old uncancellable in-process goroutine is explicitly insufficient. This plan is proposed to the hub; implementation has not started.


### 2026-10-23 — Approved implementation; focused verification

Authenticated ops:0 deliveryf91465a3 approved the proposed test-only quick flow. change-code passed inferred quick flow. No production Go, local-gates configuration or product timing was changed. The real RunWithEnv executes in an explicitly selected self-exec child with a test-local model shim. Parent observes exact child process/birth, changelog-render role and empty target before nonblocking writer admission, exchanges a checked newline, joins the sole Wait and verifies zero leases, unchanged LastUse, no model invocation and no generated output. All failure paths cancel and join that child before disposable roots are removed.

Regressions pass for early child exit, writer refusal after exit, no reader within the existing1s observation bound, non-ENXIO error propagation and cancellation of the real blocked FIFO command. An exec-copy diagnostic regression first failed with65536 retained bytes because embedded bytes.Buffer promoted ReadFrom and bypassed Write; an explicit buffer field restores the32KiB cap while draining every byte. GREEN helper coverage passed1.353s.

The local catalog matcher first ran no-op: the retained legacy shape failed RED with no finding instead of1; its25-case matrix then passed, including the scrollback payload sibling. Catalog registration separately failed RED before wiring and now passes test-only filename scope. Function-local direct syntax limits and dynamic/alias exclusions are documented in atlas/go-lint-catalog.md; the shared catalog and other repositories are unchanged.

50 consecutive real managed-distiller invocations passed4.079s. Targeted race qualification, three repeats across managed-distiller and helper tests, passed9.300s. Complete changelogcmd passed3.987s and repolint passed13.713s. git diff --check passed. Inventory named only internal/repolint/zellij_query_boundary_rule_test.go: audited the complete file, whose inputs are parsed AST/literal import and command syntax, not executed external reads; only its individual local_or_literal_fixture certificate was recorded. Stage all new test sources, rerun inventory and the tracked catalog walk, then checkpoint and push before the required recorded affected check/full fallback and binary-owned close.

Staged-source inventory PASS after all new files were added; the tracked repolint sweep passed15.030s with the new rule active. No additional reader inputs or certificates were named. This is the pre-recorded-check checkpoint; full recorded and review/landing outcomes remain pending.


### 2026-10-23 — Reader admission pinned in cancellation regression

Before recorded verification, strengthened the stalled-child regression: parent successfully opens its writer, proving an actual child reader endpoint, then keeps that writer open with no payload. Neither data nor EOF can finish the command; cancellation must join its real blocked native read. No production or approved mechanism change. Three race repeats of this strengthened cancellation path passed1.501s; the50-run acceptance passed again6.967s, complete changelogcmd3.629s, inventory and whitespace checks PASS. PR572 is open. Next checkpoint is the final test source for the recorded affected check/full fallback; no recorded check has been launched yet.


### 2026-10-23 — Gate handoff bookkeeping

The last Plan row originally required: "Push the tested source, retain the recorded affected/full fallback, then binary-owned close review and sdlc merge; report each real outcome." Clarify the checkbox as preparation/handoff for those binary-owned gates, so the close does not require falsely ticking future review/landing outcomes before their verbs execute. Product Done-when, approved scope and required verification are unchanged; outcomes are recorded explicitly below.

### 2026-10-23 — Recorded full fallback queued; review handoff

On pushed2b4a4443433cb567d0499c23ac9ab443b0b2ad26, affected selected the full suite because the individually audited reader certificate changed. Retain terminal65305, lease-2867141082, detached source /Volumes/TestScratch/local-checks/sdlc-local-2463433578 and inflight log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/7ee7a0e932192c6c2e9627e65efbd8e3/run.log. Initial FIFO8 behind live ariadne:2; no suite result or pass claimed. Later commits are prose only, no source changes. Preserve this check and FIFO; no duplicate run or continuation/restart while it is queued/running.

Required recorded verification precedes landing. Per tl-worker, the binary-owned close review may overlap this queued check; any review source fix needs fresh verification. Immediately before close run make test-affected-inventory again. Normal measured close, no typed hours or waiver. On SHIP enter merge promptly; if readiness refuses only this pending record, retain the check and retry after its pass. No manual main integration, operator stage or local-gates label is authorized or needed.


### 2026-10-23 — BR-1: qualify the FIFO census

The close returned SHIP in round1 with no Critical/Important findings; normal measured actual0.20h. Its Minor BR-1 identified a universal residual classification in the proposal that was too broad. Verified ptyrunner_test.go's checked ready payload/read and reap_orphans_live_test.go's held stdin writer, then corrected the census to name readiness, held-lifetime and nonregular-file fixtures separately. Scope and executable source are unchanged; the rule still targets only its documented direct empty-close shape. This prose disposition is verified by the named source inspection, not a wording test. The generated gate ledger stays binary-owned; no redundant review is requested for this advisory-only documentation correction.

Next: enter sdlc merge promptly on SHIP. The required recorded suite lease-2867141082 is still queued (last observed FIFO7); retain it if readiness refuses the pending record, then retry merge after the exact source passes. No suite pass or landing is claimed.


### 2026-10-23 — Recorded suite reaches known825; exact retry retained

The original affected-selected full fallback acquired capacity after1763s queued. It ran all scripts and Go packages; changelogcmd passed2.718s, couchcore297.998s and repolint17.515s. Sole named failure: github.com/xianxu/pair/cmd/internal/testcouch/agentfixture.TestCancellationRetirementAllowsIDReuseWithoutInheritedRevocation, request_lifetime_test.go:188, cycle83 leaked results:1 with nil read error. Agentfixture has no diff in this branch. Registry and issue825 identify the same prior leaked-results failure class; no second unknown failure, branch defect or679 timeout was observed.

SDLC automatically selected only that exact-package/top-level test on the same pushed2b4a4443433cb567d0499c23ac9ab443b0b2ad26 source (known issue825). Retain original terminal65305 and successor retry lease-3362002071; it is queued, last observed FIFO6. The original live log path above includes the base failure and retry admission. There is no recorded pass yet. Do not start a duplicate local-check resume: the existing owner already owns the automatic retry. No full-suite rerun, source edit, cancellation or queue change occurred.

Authenticated fleet noticebcbad09e directs affected instead of suite before close/boarding; this invocation already did exactly that and selected its declared full fallback because of the audited reader certificate. It remains running under the same owner. The close had already returned SHIP; PR572's CI passed. The attempted merge exited normally while its record was pending. Next: wait for this exact registered-flake proof and final record, then re-enter sdlc merge; report refusal if the gate cannot reuse the docs-only descendants. No continuation/restart while this check owns its FIFO place.


### 2026-10-23 — Recorded PASS with825 retry; merge ready behind852

The retained exact825 retry acquired after3160s queued and passed: test2.59s, package2.744s, immutable retry attempt07e37ba1809a33392b5df237c9cd75d1. Original terminal65305 then exited0. SDLC recorded passing sdlc-local/suite on2b4a4443433cb567d0499c23ac9ab443b0b2ad26, linked to known825; the base full coverage had exit2 in13m21s with only that registered failure. Its saved original log is /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/7ee7a0e932192c6c2e9627e65efbd8e3.log, SHA prefixc994d3f99b88. This is explicit known-flake rerun evidence, not an all-green initial suite or a full-suite repeat. No source changed and no manual duplicate resume was launched.

Re-entered sdlc merge --yes in retained terminal1512. Readiness accepted PR572 at63e131192, with CI green and the SHIP review. Landing status then showed679 first normal behind active845, but normal admissions held by fleet-blocking852 because main's couchcore park test is red. Keep this same merger; do not cancel, requeue, manually integrate main or alter the852 grant. This checkpoint is prose only and does not change the tested source. Next: let supported admission/integration finish after852 clears, inspect the retained runner and final PR/tracker result, report679 landing to ops:0, then claim and plan851 as separately dispatched (authenticated7996b2e8). No679 stage or cleanup resource remains other than the owned merge runner.
