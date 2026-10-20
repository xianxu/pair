---
id: 000708
status: done
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours: 1.78
collaboration-mode: collaboration
card_mirror: '4958e4242caa2c0cbd07211db7344fb6c2327a7e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T10:47:24-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 2.94
---

# couch: put slots' temp folders and Zellij session metadata on the TestScratch RAM disk

## Problem

The operator's write-only fs_usage sample with the machine quiet (2026-10-19, ops side doc 2026-10-19-06) showed two steady
sources of main-disk file churn outside full suites:
- **Slot temp folders.** Codex and Qoder slots run with the system default TMPDIR (`/var/folders/.../T`, the main SSD), so
  every targeted `go test` they run creates and deletes its temp folders there (~3,700 operations in 30 s). Claude slots use
  `/Volumes/AgentScratch/tmp` (also the SSD).
- **Zellij session metadata.** Every live Zellij session (29 slot panes) rewrites
  `~/Library/Caches/org.Zellij-Contributors.Zellij/contract_version_1/session_info/<session>/session-metadata.kdl` every
  second (~1,840 operations in 30 s), regardless of `session_serialization false`; each write is also journaled by
  fseventsd. The folder also holds 73 session directories for 29 live sessions (stale ones never pruned).

The operator approved a 16 GB RAM disk at `/Volumes/TestScratch` (ops `teams/devinfra/tools/testscratch`, login item;
contents vanish at reboot). ariadne#468 moves sdlc suites, suite checkouts and review checkouts onto it. This issue moves
the slots' own churn. Operator: top priority, for speed first; SSD writes are one bottleneck.

## Spec

When `/Volumes/TestScratch` is mounted (a separate device from `/Volumes`) with room (same floor as ariadne#468, 2 GiB):
- **Slot temp:** Couch starts every slot's agent (Claude, Codex, Qoder) and its shells with `TMPDIR` (and `GOTMPDIR`) set to
  `/Volumes/TestScratch/tmp/<repo>-slot<N>/`, created 0700 at start. Otherwise today's TMPDIR, unchanged. Each slot's
  folder is removed when the slot is torn down; a stale-folder sweep at Couch start removes folders of slots that no longer
  exist. Share the "is the RAM disk usable" rule with ariadne#468 in spirit (same path, same floor); pair owns its copy.
- **Zellij session metadata:** Couch keeps `.../Zellij/contract_version_1/session_info` as a symlink to
  `/Volumes/TestScratch/zellij-session-info/` when the RAM disk is usable, created before the first Zellij session starts;
  when the RAM disk is absent, it restores a real directory. Never delete a live session's metadata (Zellij needs it for
  `zellij ls`/attach); migrate live sessions' folders when switching. Prune session folders with no live Zellij session.
- Neither change touches durable state: transcripts, pair's data dir and the Go build cache stay where they are.
- Respect ARCH-EVOLVE: no format change; the fallback is today's behavior.

**Approved Revision R1 (2026-10-19):** the hub replaced live metadata migration with reboot-only setup. With no native Zellij server running, rename the real session_info to session_info.disk (retain forever) and create the RAM symlink. If any server is live, change nothing and log once `Zellij metadata stays on disk until the next reboot`. If RAM is absent with no server and session_info is our link, replace it with an empty real directory; do not restore .disk. No atomic exchange, layout linking or metadata cache pruning. The original migration/pruning clauses above are superseded. The slot-temp half remains approved, including CLAUDE_CODE_TMPDIR and no-holder cleanup.

**Approved Revision R3 (2026-10-19, supersedes all earlier cleanup clauses):** keep slot folders while Couch runs; RAM empties at reboot. At startup and hourly remove ordinary top-level entries only when their own and every descendant's mtime are strictly older than 24h. A job leaving regular temp files untouched for a day accepts ordinary age-cleaned /tmp loss risk; no process census, holder proof or FD exemption. Never remove sockets, FIFOs or other special files, or a top-level tree containing one. Manager bookkeeping stays outside swept interiors. Keep 2 GiB admission fallback. Pin the pre-overlay ZELLIJ_SOCKET_DIR in every slot and Couch's own native calls so moving TMPDIR does not move the session namespace. R1 metadata setup remains approved.

## Done when

- With TestScratch mounted, a new slot's agent and shells report TMPDIR under `/Volumes/TestScratch/tmp/<slot>/`, and a
  targeted `go test` in a Codex slot leaves nothing in `/var/folders/.../T` (live check).
- `session_info` resolves to the RAM disk; `zellij ls` and attach keep working across a Couch restart (live check).
- Without TestScratch, both behave exactly as today (fixture tests for both branches).
- A 30 s write-only fs_usage sample on a quiet fleet shows the `/var/folders` and Zellij lines gone or near zero (the
  operator runs it; command in the ops side doc).

- R1 metadata acceptance supersedes the Couch-restart migration check: isolated-HOME fixtures prove no-server RAM setup, live-server no-op and absent-RAM restoration from our link, including retry/failure preservation. The real metadata switch and write-only sample happen at the operator's next reboot. Existing panes need relaunch for their inherited temp settings; no live cache migration/pruning is required or allowed in this issue.
- R3 fixtures prove a fresh entry survives, a 25h-old ordinary tree is removed, and an old tree containing one fresh file survives. Socket/FIFO/special-containing trees, slot roots and external manager bookkeeping survive regardless of age. Startup/hourly sweeps are serial and joined on shutdown; counted N/2N metadata work stays linear with zero cleanup process inspections. With a slot TMPDIR on RAM, resolved Zellij socket directory/spelling stays unchanged for slot commands and Couch calls.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md, using Method A and the v2/v2.1 primitive table. estimate-source marks this calibration stale; values are provisional. Library check: standard os.Root, stat/statfs and existing runner/worker/native-process seams cover the integrations; no novel runtime/library build is required. For the two new single-concern modules, midpoint design 1.25h is halved for library availability and multiplied by 0.2 for the approved thorough plan (0.125h each). Other design midpoints receive the same 0.2 spec multiplier. Implementation values are v2 midpoints multiplied by 0.4 once; familiarity is 1.0.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.125 impl=0.22
item: greenfield-go-module design=0.125 impl=0.22
item: smaller-go-module design=0.03 impl=0.14
item: smaller-go-module design=0.03 impl=0.14
item: cross-cutting-refactor design=0.12 impl=0.14
item: real-api-discovery design=0 impl=0.18
item: atlas-docs design=0.025 impl=0.05
item: milestone-review design=0.02 impl=0.14
design-buffer: 0.15
total: 1.78
```

Rows respectively cover temp manager/ownership, age sweep/worker, socket resolver/pin, reboot-only metadata setup, constructor/runner/service wiring, owned child/native qualification, README/atlas and one close boundary. Design subtotal 0.475h plus 15% = 0.54625h; implementation subtotal 1.23h; total 1.77625h rounded to 1.78h. Machine-capacity waits and the operator's later reboot are not active implementation time.

## Plan

- Durable design: [Couch RAM scratch plan](../plans/000708-couch-ram-scratch-plan.md).
- [x] Implement shared scratch policy, pinned socket environment and R3 startup/hourly age cleanup.
- [x] Implement R1 startup-only metadata routing and the three isolated-HOME fixture branches.
- [x] Complete counted budgets, documentation, recorded suite, close review and priority landing.

## Log

### 2026-10-19
- 2026-10-19: closed — BR-3: the real special-node retention test now sets an overlong inherited TMPDIR; observed RED before the owned short-root fixture and GREEN after, using the existing justified tmp-literal lint exception. Scratch race tests and both source-guard regressions pass. Slot naming reuses EnvName; noon main integrated cleanly with no conflicts. Final corrected-code suite is queued or running and must pass before landing. Compound final Plan item remains unticked until suite and landing; no-plan-check acknowledges only this sequencing. Real smoke remains approved operator reboot/relaunch, no live edits or pair:0 staging. (fixed-then-ship); review verdict: SHIP
- 2026-10-19: closed — BR-1 addressed: TryLock/nonblocking flock and one probe precede lazy identity; paused sweep and unavailable regressions prove no resolver call, caller cancellation bounds identity. BR-2 addressed: injected private native observation/clock/ticks exercise enabled and fallback constructor-to-launch, actual startup and maintenance callsites, and service cancellation/join. Both production callsite omission mutations fail as required; restored targeted race/source guards pass. Fresh recorded suite queued on 613439241486df66548977b33c25e9d2d6b4f0c9. --no-plan-check acknowledges only the compound final suite/close/landing row; final suite pass required by merge. Real smoke remains approved operator reboot/relaunch, without live edits or pair:0 staging.; review verdict: SHIP

Verified the hub's canonical assignment receipt dcc7fdf7-95c8-43b1-aad6-fcf150ad0053 (submitted to this pair:3 incarnation), ran tl-worker, claimed and entered start-plan. Start-plan refreshed the slot's ariadne substrate and created the branch from main at e51a0e6eda6910f209e68fde2df8f9ea494ec07f. Read the complete Spec and ops second-machine measurement document. No live session/cache/volume changes made.

Read-only exploration identified both production StartBlocked sites (launchTrackedThread and attachListedTerminalOnce), Claude's CLAUDE_CODE_TMPDIR override, and the need to retain detached servers/background jobs across attachment exit. Exact Zellij v0.45.1 source confirms unlocked pathname-based metadata writes and unconditional parsing of an existing resurrection layout before live attach. A rename-then-symlink gap or incoherent layout copy is unsafe. Proposed atomic exchange retains originals; existing layouts/external files keep their original inode via links while metadata moves to RAM. Native fixtures must use private HOME because XDG_CACHE_HOME alone does not isolate macOS ProjectDirs.

Read-only native cache sizing counted 73 session directories, 72 metadata files, 10 layouts, zero other files/symlinks and 308,437 bytes. This is not liveness or deletion evidence. ariadne#468's in-flight scratchdisk helper uses TestScratch, a separate-device check and 2 GiB floor; pair owns an analogous injected policy. Plan names the remaining approval boundaries: foreign/unproved cache rows remain protected, and the real fleet rollout belongs to the operator's restart. No code or tests run before plan approval.

Hub receipt a7a22cfc-b15d-478f-bac2-c9a1089f683f was verified submitted with exact body/sender/current recipient. It approved temp setup and directed R1: defer metadata switch until no Zellij server is running, keep a single permanent .disk backup, no live migration/exchange/layout links/cache pruning, and fixture-only metadata acceptance before the operator's next reboot. Appended the revised executable design and retained the original proposal as history. No production code has been written.

Plan-quality round 1 accepted the simplified metadata design and raised one Important PQ-1: current ancestry/open-file snapshots do not prove a detached job's lifetime. Appended R2 with ObserveScratchUsers using one complete native table and PID/birth-fenced initial-environment reads (existing NativeResourceReader), exact temp-root/thread attribution independent of ancestry, and explicit conservative retention of empty/unavailable/changed evidence. The stateful regression controls parent exit, reparenting, closed FDs, delayed scratch use and exact child exit; unknown evidence keeps the folder. Busy/opaque hosts may retain all candidates until a later complete sweep or RAM reboot; no process is stopped to improve collection.

Plan-quality round 2 kept PQ-1 open: a nonempty changed environment does not prove no future scratch dependency. Stopped before a third delta and asked the hub. Verified submitted receipts 060f7bf3-3388-4042-a976-a66bf5598d6b and 8e09b3e8-1302-4ed1-8ca9-ff8068370141 explicitly replace lifetime collection with the conventional 24h age contract, preserve slot roots, approve special-tree exclusions and require socket-root pinning. Added R3 as the sole executable contract. No holder/env census, retirement cleanup hook, atomic migration or layout-link code will be built. Only metadata setup still observes native server presence.

Plan-quality round 3 returned CLEAN, disposing PQ-1 as addressed by the explicit R3 contract change; no open findings. Derived the estimate only after that verdict, against the configured v3.1 source and baseline (stale calibration noted), and set card estimate 1.78h through sdlc. The mandatory implementation gate is being rerun with the reconciled block; no production code before it clears.

Implementation gate passed at 72795a6e61ab29f732719a827ea539eddba0ff56. Estimate-quality INFO called the 1.78h estimate optimistic for guarded cleanup, qualification and integration/landing attention; the nonblocking warning is retained here. Recovery peek reports previous recovery screen unavailable; branch, issue, plan and accepted gate supplied continuity. No historical peer instruction was treated as new operator authorization.

TDD observed missing APIs before manager, metadata, socket and launch integration implementations. Private targeted tests pass for policy/age/special-file retention, owner evolution and conflicts, missing RAM roots, metadata retries and exact backup preservation, fresh/warm launch overlays, retained primary/numbered identities with missing checkouts, and unchanged sockets. Initial manager verification includes race/count=20; final qualification still pending. Source inventory and declaration-comment guards pass. No live metadata, slot environment, or pair:0 binary was changed.

Implementation uses the retained ThreadRecord at the two existing launch boundaries, rather than re-reading an address through a checkout-dependent backend in the runner decorator. This is necessary because warm reattachment can have a missing checkout; the runner still owns the one socket pin for both Start and StartBlocked. A primary's first thread without enrolled authority may call the existing bounded verified workspace resolver once; malformed/ambiguous evidence never falls through. Startup metadata/sweep finish synchronously before launch; one joined maintenance worker performs hourly sweeps without repeating startup. Metadata observation is separate from age cleanup. Aggregate cleanup notices are bounded and quiet for unchanged interiors.

Cleanup counters distinguish entry visits (<=2F+O(S)), all stat/fstat inspections (<=4F+O(S)), owner declaration reads, and zero temp-payload reads. No process inspections participate. Descriptor confinement and revalidation are best-effort against concurrent filesystem mutation, not atomic tree deletion: old siblings may be pruned before a changed child aborts; the final compare-to-unlink window cannot be closed portably. This is recorded as the ordinary temp-cleaner tradeoff, without claiming lifetime or atomicity proof.

Final targeted race checks passed across cmd/couch, couchscratch, couchcmd, couchcore and launcher; artifact/source inventory, dead-symbol and whole-tree duplicate/stolen-comment guards passed. Private owned child wrote its file under the injected slot directory and reported all three temp variables plus the unchanged socket root. Socket resolution now matches directories-rs 6.0.0 and Zellij v0.45.1 on Linux as well as Darwin; absolute XDG_RUNTIME_DIR is retained, relative/absent values fall back. Implementation commit 20e2e96244b0c2adc439e2f15da10136d2b6ecb4 is pushed in draft PR 463. Recorded suite attempt 5206c8256520442f515e6abdba4518ef is queued on that exact head. Close review may overlap this queue under tl-worker; the last compound Plan row remains unticked until the suite and landing gates finish. No passing-suite claim is made yet.

A bounded read-only real-root qualification found /Volumes/TestScratch mode 0775 and its existing shared tmp parent mode 0755 (current account). The private-directory helper incorrectly demanded 0700 on that shared parent, so every real slot would have fallen back. Stopped the owned close parent (41321, exact verified command) with SIGINT; the binary canceled its reviewer and admitted no verdict. Canceled only this checkout's obsolete queued suite through local-check cancel, preserving other checkouts. This is a code correction before execution, not a flake retry. Add a fixture with the observed parent mode, preserve its inode/mode, keep slot/bookkeeping 0700, and qualify writable ownership without imposing private permissions on shared containers. The interrupted review's disposable checkout cleanup warned about a nonempty diagnostic marker directory; the binary's next review sweep owns retry, no manual process/path intervention.

Shared-container regression observed RED on the original helper and now passes with the actual 0755 shape. Its inode/mode remain unchanged; owned slot/bookkeeping children remain 0700. Unsafe writable, foreign-owned and symlink containers refuse both preparation and sweep. A held owner lock now produces immediate fallback instead of blocking optional admission. Final targeted race checks passed across all five affected packages; manager package race/count=20 and counted budgets passed; source/declaration guards passed. README now states the age-loss and reboot contracts alongside the atlas map. The first implementation's macOS conformance/merge-check/local-gates CI passed; new source changes require a new pushed-head suite/CI and full close review.

Boundary round 1 returned REWORK with two Important findings: BR-1 optional-maintenance-blocks-admission, BR-2 production-composition-regression-coverage. No Critical/Minor findings. The broader reviewer run hit socket-path/production-registration fixture restrictions; its focused checks passed. This is not the authoritative suite. Canceled only the obsolete queued attempt e10643e6213f2f1f471c3268ea51e1d3 before changing source; no flake retry or status republish.

BR-1 correction enumerated the entire optional admission path: the manager mutex is TryLock, the owner flock is nonblocking, and PrepareResolvedSlot performs availability checks before lazy identity resolution. Busy/unavailable scratch never resolves identity. Caller cancellation and one five-second context now cover retained-manifest lookup plus the existing workspace fallback. A paused-sweep regression failed before the fix, then proved admission returned before sweep release, with no resolver call; manager race/count=20 passes. All fallback notices are bounded to512 bytes.

BR-2 adds private runtime injection for native server observation, clock and ticks. Constructor-enabled/fallback tests exercise real NewCouchWith, retained membership, both temp outcomes, and socket authority. The test exposed isolated TMPDIR defaults masking a legitimate RAM overlay: isolated runtimes now accept only a complete overlay under their injected, confined scratch root and keep all three fallback temp variables private. Foreign/incomplete overlays and escaped backend roots refuse. The actual run.go startup path and actual messageService lifecycle are exercised; future fixture time makes clock injection observable, unbuffered ticks prove a completed sweep, and service.Close must cancel/join a held maintenance return.

Omission mutations were run and restored sequentially: disabling run.go's ScratchStartup call failed TestScratchStartupRunsBeforeMissingResumeFailure (observer calls0); disabling message_service.go's ScratchMaintenance worker failed TestScratchServiceDrivesMaintenanceAndJoinsOnClose (worker did not start). Restored targeted checks passed. Targeted race tests passed for scratch, Couch CLI/core/executable and launcher; source/declaration guards passed. New source still requires a fresh recorded suite and close disposition.

Boundary round 2 returned SHIP (medium confidence), disposed BR-1 and BR-2 as addressed, and raised no new findings. The reviewer independently repeated all three omission mutations and restored their regressions. Close evidence is 8882b30b03eba4e3d83e048267b8925b401c8e13; tested source is 613439241486df66548977b33c25e9d2d6b4f0c9; actual 2.61h was measured/adopted by sdlc. All CI checks on that tested code passed. Recorded branch suite 003733a04266285b135f734f5c2554bd is pending. Verified canonical submitted hub receipt 63e5b639-681b-48fe-902f-08eec4db6c0c grants six-hour operator priority and directs merge after suite records; the ticket reports fleet-blocking position 1. The final Plan row remains unticked until landing. Read-only verification found no couch-scratch bookkeeping directory on the live TestScratch volume: qualification created none there.

**Approved operator live check after landing:** at the next operator-owned reboot, ensure TestScratch is mounted before starting the rebuilt Couch. Confirm native session_info is the RAM link and the retained session_info.disk still exists; inspect the one deferred-metadata line if a native server was already live. Relaunch selected slots to inherit the new environment, then inspect TMPDIR/GOTMPDIR/CLAUDE_CODE_TMPDIR against /Volumes/TestScratch/tmp/<repo>-slot<N>; a targeted Go command should leave its temporary files there. Compare ZELLIJ_SOCKET_DIR before/after relaunch, verify zellij list-sessions and couch actors still see the same native namespace, and perform the normal attach/Return smoke. The operator's 30-second write-only trace checks temp/metadata churn after reboot/relaunch. Do not unmount RAM or edit a live metadata route for this check; absent/full/unknown branches are private fixtures. No pair:0 staging or operator process intervention by this slot.

### Recorded suite source-guard corrections

Attempt 003733a04266285b135f734f5c2554bd ran the complete suite on 613439241486df66548977b33c25e9d2d6b4f0c9 and failed in 13m16s, retained log SHA256 prefix fd083a03a869. Its only failures were deterministic branch defects: TestEverySlotStateSiteIsAResource rejected the duplicated slot basename spelling in couchscratch/policy.go, and TestTrackedGoFilesPassCatalog rejected a literal /tmp socket fixture root. All other package and preceding integration results passed. This is a corrected-code run, not a flake retry. Scratch naming now delegates to the existing couchcore.EnvName authority (ARCH-DRY); the socket fixture uses os.MkdirTemp with the inherited temp root. Removed the now-unused production number-format wrapper. Both failed checks and the special-node regression pass directly; the scratch package race check passes. A new recorded suite and close review are required on this correction before landing. Hub receipt 72640f1e-8ea3-4fdc-b6fa-35072b27ebd0 verified submitted: #705 landed and the issue-publication hold is lifted.

Fix-review round 3 returned REWORK with BR-3: inherited long TMPDIR makes the native socket fixture exceed its address limit. Added that exact long-root condition to TestSpecialTopLevelNodesRetained and observed RED (invalid argument), then used the existing transport fixture's explicitly justified tmp-literal exception for an owned short root. The test still binds real socket/FIFO nodes and checks their retention. The queued corrected-code attempt 6b90c9f971421e7ec695e05707851430 on 19fb9c730fe0a2a760350df2f2c84ab371860a5c was canceled through sdlc before running because this test correction supersedes it; no retry result was recorded. Noon freshness required a clean main integration before round 3; no conflict was resolved. BR-1 and BR-2 remain disposed as addressed.

### Verified landing

2026-10-19: PR 463 merged as 537d37028aaae4a6a839d08acb10234e52af63b0, confirmed reachable from freshly fetched origin/main. The corrected branch suite f9ccb6d8b702a2ea104e5384dbd6c667 passed on d7593dc70df0e65f516ef7baa1b45e4b1c632514 in 16m40s (log SHA256 prefix 913f421a01c1). The authoritative integrated suite 8e041ed891c4b2dcee614123fff17698 passed on c98dd14495236bfd54e3ea8f21bb0b74a75a9e7f in 16m03s (log SHA256 prefix c76c60546e36); all five merge checks were green. No flake retry was used. Fix-review round 4 was SHIP, all three findings addressed, no new findings. SDLC marked the card done, archived the issue/plan/reviews, removed the issue branch and returned this checkout to main-slot3. The operator-owned next-reboot/relaunch smoke remains the approved live arrangement above; no live metadata route, slot environment or pair:0 build was changed here. The final Plan row is now ticked on actual evidence rather than anticipation.

## Revisions

### 2026-10-19 10:56 PDT — R1, hub-approved simplification

Replace the initial live metadata migration/pruning contract with no-server startup setup and permanent session_info.disk retention. Scope now excludes atomic exchange, layout linking and cache pruning. Done when uses three isolated-HOME fixture branches and the operator's next reboot for real metadata routing. Temp environment, ownership and cleanup remain approved. Detailed final execution contract is Revision R1 of the durable plan.

### 2026-10-19 11:04 PDT — R2, plan-quality cleanup clarification

Name the existing native environment API and detached-job evidence lifetime; retain on incomplete attribution instead of inferring no holder from ancestry or closed files. Add the delayed-use sequence/mutation test and linear observation budget. This specifies the approved no-holder rule without expanding scope or changing R1 metadata behavior.

### 2026-10-19 11:05 PDT — R3, hub-approved age contract and socket stability

Supersede R2 and all no-holder cleanup with startup/hourly whole-entry 24h age cleanup. Keep special-containing trees and slot roots until reboot, keep bookkeeping outside sweep interiors, and document the accepted old-regular-file loss risk. Pin today's socket root across temp changes and all native Zellij calls. R1's reboot-only metadata setup remains; real switch waits for the operator's reboot. See final R3 core-concept, strategy and execution tables in the durable plan.

### 2026-10-19 — Review corrections within R3

Keep the same age/fallback/reboot contract. Nonblocking admission includes the manager mutex and availability check before optional identity IO, with caller context covering the one identity budget. Qualify RAM-enabled production composition with injected native observation/time/ticks and omission-sensitive startup/maintenance tests. Private runtime overlays remain confined to private storage; this adds no public config or live rollout action.
