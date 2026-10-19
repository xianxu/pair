---
id: 000701
status: codecomplete
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours: 2.24
collaboration-mode: collaboration
card_mirror: 'ca162b49431bb91f1339d59e1d08f959a19609cb' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T12:08:04-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 0.67
---

# Read closed WAL databases without sidecars on macOS SQLite

## Problem

#688's native acceptance found memories_1.sqlite (45,056 bytes, WAL format) had no -wal/-shm files, and macOS system sqlite3 3.54.0 returned exit 1 at PRAGMA schema_version with unable to open database file. The source remained unmodified by the failed read. A disposable closed-WAL fixture reproduces this independent of BEGIN order and -nofollow; installed Homebrew sqlite3 3.53.4 reads it and creates normal coordination sidecars. Active databases with existing sidecars work.

## Spec

Choose a small compatibility route for closed WAL-mode databases without restoring starvation for active writers. Hub's proposed options: (a) a verified plain copy for a database proved closed by no sidecars and unchanged size/mtime before and after, with identity and source-isolation checks; (b) a pinned known SQLite implementation. The owner must assess the closure proof and consistency contract before choosing. Do not assume immutable mode on a live database, silently open the native DB writable, or add a new runtime dependency without approval. Keep failures classified honestly and preserve earlier verified bundles.

## Done when

- A portable cold WAL fixture reproduces the system-CLI refusal and proves the selected compatibility route without changing native DB/WAL content.
- Races with a writer starting, sidecar creation and source replacement refuse unsafe publication; unknown SQL schema and rowids survive.
- Tests qualify the supported CLI(s), bound work and cleanup, and document the operator's one recovery action.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; source is tagged stale, so this is provisional. Derived after plan-quality round 2 cleared PQ-1, not from elapsed time or a chosen total.

Issue/spec work uses 0.8h design and 0.2h original implementation; the cold-copy module uses 1.0/0.8; existing orchestration 0.2/0.5; native API qualification 0/0.5; atlas 0.1/0.1; one close review 0.1/0.4. Source helpers cover filesystem/hash/SQLite integration, so no new library or novel stack. Apply the thorough-plan design factor 0.2 to implementation modules/docs/review, not the design-authoring primitive, and v3.1's 0.4 implementation scale to every row. Familiarity is 1.0 from #688; design buffer 0.15. Design sum 1.08, implementation sum 1.00: 1.08 x 1.15 + 1.00 = 2.242, rounded to 2.24h.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.8 impl=0.08
item: greenfield-go-module design=0.2 impl=0.32
item: smaller-go-module design=0.04 impl=0.2
item: real-api-discovery design=0 impl=0.2
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0.02 impl=0.16
design-buffer: 0.15
total: 2.24
```

## Plan

- [x] Qualify the closed-WAL refusal and private-copy route with disposable native fixtures and adversarial ordering controls.
- [x] Implement the bounded, descriptor-owned cold capture and share existing validation, staging and cleanup.
- [x] Verify cold and active routes, source preservation, catalog publication, cancellation and counted costs; update the archive map.
- [x] Run the recorded branch suite.

Detailed design: [closed WAL compatibility plan](../plans/000701-closed-wal-compatibility-plan.md). Hub approved and change-code passed. Delivery follows qualification: close review, SDLC landing, Git-verified report to the hub and stop.

## Log

- Hub 38daa8cd-726f-4984-aff5-c511b42bb896 explicitly keeps this out of #688's size-budget fix; file unclaimed. memories_1 is small and rarely changes, so not urgent.
- Owned private probe: Python creates a WAL database then closes it, removing sidecars. System /usr/bin/sqlite3 -readonly returns unable to open database file; /opt/homebrew/opt/sqlite/bin/sqlite3 succeeds. No machine changes or source repairs were attempted. A subsequent native readonly schema query succeeded once normal sidecars were present; who created those sidecars was not established.

### 2026-10-19 — qualification and design
- 2026-10-19: closed — BR-1 corrected across the failure family: typed uncertain source evidence/deadlines defer; proven unsafe types/size and private output create/write/close failures reject, preserving prior stage and catalog. Three classification controls failed before correction (0.190s); corrected broad SQLite/cold/continuous subset passed 11.669s and artifactpath guards 4.114s. Native system CLI conformance, real writers during both reads, transient rollback sidecars, opaque schema/rowids and counted work remain green. Old queued suite on b0e8c755 was cancelled through local-check cancel before execution because code changed; replacement recorded suite is queued on pushed 3375a20570c743da2cf33165e0ef26fc468b73c1. Per tl-worker, review overlaps the queued suite; final exact-code pass is mandatory before landing and any code correction requeues. --no-plan-check acknowledges only pending full-suite qualification, not a waiver; --no-project because the hub owns the project. No real Codex database or archive publication touched.; review verdict: SHIP

- Claimed in pair:7; start-plan completed, including its slot-owned weave refresh. Read the existing snapshot, subprocess, descriptor safety and continuous-publication seams. A read-only subagent audited those seams and challenged the consistency argument; no production files changed.
- Disposable probe under pair701-probe.hxsJxp: system SQLite 3.54.0 refuses a canonical closed-WAL path with exit 1 at schema_version and leaves its sole source file unchanged; installed Homebrew SQLite 3.53.4 succeeds and creates normal WAL/SHM sidecars. On an independent private copy, system SQLite's existing DELETE/integrity validation succeeds and returns rowid 73, unknown text and exact BLOB 0001FF. Original source still contains only its byte-identical database. The first probe used /var's symlink spelling and both CLIs refused -nofollow; canonical paths separate that different refusal from the actual closed-WAL failure. Tests must canonicalize fixture roots.
- SQLite documents ordinary read-only WAL access as depending on readable sidecars, permission to create them, or an immutable connection; raw copies of active transactions can be inconsistent. Sources: [read-only WAL](https://www.sqlite.org/wal.html#read_only_databases), [copying during a transaction](https://www.sqlite.org/howtocorrupt.html#_backup_or_restore_while_a_transaction_is_active). Immutable is not proposed on a native source.

### 2026-10-19 — implementation and targeted evidence

- Plan-quality cleared PQ-1 in round 2, round 3 CLEAN; estimate-quality INFO (allocation aggressive but derivation plausible), change-code exited 0. Estimate allocation: cold-module row includes pure and mutation tests, orchestration includes prior-stage and continuous tests, discovery covers native tool/writer fixtures, review includes foreground suite/PR/landing attention. Background queue waiting is not added to active hours. No estimate backfit.
- Production public regression before wiring: TestSQLiteClosedWALNativeConformance failed with the original unable-to-open refusal for configured sqlite3 and /usr/bin/sqlite3 (0.601s). Pure model stub tests also failed before their implementation; the first IO test build found missing helper symbols, then the implemented helper passed. Do not represent that IO compile failure as an independent behavioral red. The original user-visible failure is the public regression.
- After wiring, all new model/capture/native cases passed (1.858s). Source capture costs are exactly two complete reads plus one fixed header read and three observations per source at B/2B and N/2N. Pure header fuzz ran 28,785 executions in 2s, green. Real SQLite writer conformance passed 0.916s: committed and rolled-back transactions during either read are refused; rollback leaves DB bytes and metadata unchanged and isolates parent-sidecar-churn evidence.
- Root qualification: go test ./cmd/internal/historyarchive -run 'Test(SQLite|Cold|ObserveCold|SameCold|Continuous.*(Snapshot|Cold|Publication))|FuzzColdWALHeader' -count=1 passed 24.188s. It includes existing active-writer progress, owner-death/alarm/lease, overlap and cleanup checks plus new cold routes. Full artifactpath package passed 9.875s. Continuous cold publication independently passed 3.387s: original source identity, opaque SQL/rowid, unchanged-current behavior and failed-first/later-active success preserve the prior catalog and bundle.
- Test correction: the conformance image helper deliberately rejects nonregular entries; the unsafe-journal fixture now independently asserts unchanged source bytes and symlink target instead. This changes no production outcome. Its targeted rerun passed 0.679s. Characterization queries use a separate probe copy so normal CLI-created coordination files cannot contaminate source preservation or route eligibility.
- Final focused qualification after routing cold diagnostics through the existing bounded snapshotFailure: historyarchive passed 3.498s and artifactpath guards passed 3.982s. Implementation committed at b0e8c75592af6c67ad4e9c148138d42a3173d48d and pushed as PR #464. Recorded suite attempt 7855ff58f427e11d5d4dbdb8cbeebef9 is queued on that exact code head (initial FIFO position 6); session 46416 owns it. Full historyarchive and suite completion remain pending. Under tl-worker's overlap rule, close review may run while queued, with the final suite pass required before landing and any code correction requiring a new recorded run. The only unchecked implementation obligations are that full-suite qualification; no behavior proof is waived. Acceptance uses disposable native SQLite files, with no real Codex source or archive publication invoked.

### 2026-10-19 — close BR-1 and correction

- First close review returned FIX-THEN-SHIP but did not finalize because BR-1 was Important: proven unsafe journal paths were being described as temporary source uncertainty. Reviewer independently ran the full historyarchive and artifactpath packages (131.493s/16.838s, green) and active-writer conformance observed 280 transactions; no correctness/preservation finding beyond classification.
- Swept the failure-classification family: uncertain source evidence defers, known unsafe types/size and local output creation/writing/closing reject. Typed source/output sentinels survive the shared formatter; no phrase matching. The local writer adapter distinguishes output failures from read uncertainty during the shared digest copy.
- Red before correction: oversize, private-output collision and unsafe-journal controls failed as expected (0.190s). After correction the complete new cold/native/publication subset passed 2.760s; changed source/deadline still defer, previous candidate survives both categories. A real read-only descriptor exercises local write error provenance without changing its bytes.
- Supported local-check cancel stopped only this checkout's queued b0e8c755 run (7855ff58f427e11d5d4dbdb8cbeebef9); it never acquired capacity or ran tests. Other checkout runs were explicitly left running. This is a superseded-code cancellation, not a flake retry. A new exact-head recorded suite and ordinary close review are due; first close did not reach codecomplete.
- Corrected-head qualification passed: broad SQLite/cold/continuous/historyarchive subset 11.669s and artifactpath guards 4.114s. No existing expectation was changed except the three deliberately wrong failure categories identified by BR-1.

### 2026-10-19 13:06 PDT — SHIP; preserve the pending runner

Second close completed exit 0: SHIP, BR-1 addressed, zero new findings; evidence commit 1e32e42faa8d1066ccd1b475c49c400fa3ef2dce, published codecomplete, auto-measured actual 0.67h. Current recorded branch-suite attempt is 11b3d21450978b748de1e5049293eb39 on corrected code 3375a20570c743da2cf33165e0ef26fc468b73c1, exec session 65045, private checkout sdlc-local-1423522359. It is still queued (position 4 after about 30 minutes), no tests executed and no retry/failure. Inflight log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/11b3d21450978b748de1e5049293eb39/run.log. Preserve this runner; do not start a second suite or merge before its result. The old 7855ff58 attempt was explicitly cancelled before acquisition and is superseded. Server CI on the close's pushed head was independently green. After the recorded suite passes, complete the qualification checkboxes, commit/push only those final docs, enter sdlc merge --yes, verify merged PR #464 and the real Git merge hash on main, report to the hub and stop. Hub alias issue #711: retry once on a receipt-free refusal, then use exact ops:0; never restart Couch for it. #710 is queued after this item, not part of its branch.

### 2026-10-19 — recorded suite passed; entering landing

Corrected-code suite attempt 11b3d21450978b748de1e5049293eb39 completed exit 0 in 16m7s on 3375a20570c743da2cf33165e0ef26fc468b73c1, with no failures, registered-flake reruns or additional attempts. Saved log: /Users/xianxu/workspace/pair/.git/sdlc/local-checks/3375a20570c743da2cf33165e0ef26fc468b73c1/suite.log; log SHA256 prefix 2dff101141d5. Full historyarchive passed 211.342s and couchcore 508.961s. The suite posted sdlc-local/suite successfully. All qualification checkboxes are now supported by the recorded run; SHIP review/codecomplete is already finalized and all changes since reviewed code are documentation only. No real Codex database, Drive destination or normal archive sink was opened for acceptance. Enter the normal SDLC landing line; it owns main integration, CI and any required integrated-code suite. No new source changes or review are needed merely for these completed qualification notes.

### 2026-10-19 14:58 PDT — integrated scratch filesystem failure

Integrated Git HEAD 84f040bda18757d914cbdfa70aac15dea7bf59ab failed suite attempt f41f66b39b25f35e7381834b0c8eb3ee in 9m12s after every package ran. Saved log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/f41f66b39b25f35e7381834b0c8eb3ee.log (SHA256 prefix ba354be51442). The new TestScratch volume is HFS, with whole-second mtime/ctime, and no APFS clone semantics. Failures include same-size index/cache invalidation in couchcmd, launcher and sessioninventory, transient-sidecar detection in the new cold tests, existing continuous/archive fixtures, wrapcmd's PID timestamp bound, and physical-write budgets in couchcmd/couchcore/couchtty/sessionsync/workbenchperftest. This is not registered or accepted as a flake.

A disposable 20-repetition probe confirmed the metadata cause: TestScratch missed all 20 immediate same-size file rewrites and all 20 create/delete sidecar cycles; /private/tmp detected all of each. No production data or machine settings changed. Hub receipt 8d026d3b-57c2-4e6b-b911-4c9b553dd110 confirms the root fix: ops creates APFS with nanosecond metadata and clonefile, and ariadne#474 refuses non-APFS scratch. Hub explicitly leaves #701's approved high-resolution-local-filesystem contract unchanged. Await the operator's disk recreation and hub's go-ahead before rerunning merge.

Supported local-check cancel stopped only this checkout's queued diagnostic baseline d935b400d36278d3eed745aa82efaec3 before it executed, leaving one other checkout's run untouched. Merge session 12349 exited with the original suite refusal, and its TestScratch worktree was removed so it cannot obstruct disk recreation. All failed evidence is retained; no assertions, production code or local gates changed. This environment correction needs a new recorded integration pass, not a blind passing retry or manual status post.

## Revisions

### 2026-10-19 12:14 PDT — proposed compatibility route, pending approval

Choose a private plain copy only for a WAL-header source with all three coordination names (-wal, -shm, -journal) absent. Admission and two bounded full reads compare held file and parent identities, size, mtime and ctime; copy and reread digests must match. Parent mutation detects sidecars that appeared and disappeared between observations. Unknown evidence refuses the candidate without retry, preserving the existing staged file and verified catalog. This is an observed stable-input contract for ordinary SQLite on reliable local filesystems, not a claim of exclusive ownership. A writer beginning after the final capture fence does not invalidate the earlier snapshot. Active sidecars continue through the current pinned SQLite backup without any copy-then-compare fallback. No new dependency or format change. See the durable plan for assumptions, failure projection, costs and tests. #710 is queued by the hub after this issue; its coverage work is outside #701.

### 2026-10-19 12:16 PDT — approval and plan-quality response

Hub approved the design and clarified that uncertain cold evidence is deferred (receipt a29363a1-7c62-43e8-92bd-2bc7cc45e531). PQ-1 requested compression of prose test-case inventories; the function strategy table now gives one adversarial class and mechanical guard per risky function. Code remains untouched until the gate passes. Hub filed the separately reported hub-alias failure as #711; exact-address delivery succeeded and no restart was needed.

### 2026-10-19 12:23 PDT — actual concept map and delivery checklist

Mapped actual types/test declarations and the dedicated continuous fixture in the plan. Corrected the README's existing transaction-only summary and separated future close/landing/report instructions from implementation checkboxes; all original delivery obligations remain. Observed quiescence, no-retry deferral, native active backup and persisted formats are unchanged.

### 2026-10-19 12:35 PDT — BR-1 error categories

Corrected implementation to the existing approved operator projection: unsafe source structure/size and local output failures reject, while uncertain source quiescence/deadlines defer without retry. Added the private output adapter to the concept map and source/error controls; prior evidence, resource ownership and formats are unchanged. The original queued code head is superseded and must not provide landing evidence.
