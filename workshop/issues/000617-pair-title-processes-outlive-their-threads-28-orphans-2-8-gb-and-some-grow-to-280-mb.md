---
id: 000617
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours: 3.33
card_mirror: 'b625859c279abf3fe676c7bda8b9f1864aa0c763' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T12:54:47-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# pair title processes outlive their threads (28 orphans, 2.8 GB) and some grow to ~280 MB

## Problem

`pair title` processes (one per agent thread) outlive their threads: on 2026-10-16 there were 28 of them, all reparented
to launchd (ppid 1), **2.8 GB resident** in total. Some belong to threads long gone (`1-parli-12`, `1-ducks-11` from
Oct 6; `1-xianxu-dev-22` from Oct 9; `couch-9d6fdf5e…` from Oct 13), and several of the live ones have grown to
~280 MB (`1-ops-33`, `1-pair-8`, `couch-9d6fdf5e…`), so there may also be unbounded growth inside one. Couch's resource
monitor now flags them (orphan-rss alerts for ops:0 and pair:2).

## Spec

- **Lifetime:** a title process lives exactly as long as its thread's wrapper. Prefer the lock/lease shape: it holds
  or watches something its owner holds for life (the wrapper's pid with its start stamp, or a pipe/flock the wrapper
  owns) and exits when that goes, rather than a reaper deciding later by observation.
- **Growth:** measure why one grows to ~280 MB (heap profile on a live one); fix the cause (bounded buffers/caches).
- **Existing leaks:** a one-time cleanup for title processes whose thread has no live wrapper, identity-fenced (pid +
  start stamp), consistent with pair#583's rules; never by name alone.

## Done when

- A test: kill a thread's wrapper; its title process exits within seconds.
- A title process's memory stays bounded over a long-running thread (a soak test or a measured bound).
- After the fix and the cleanup, live: one title process per live thread, none with ppid 1 for a dead thread.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A primitives plus Method B on memory attribution. Calibration marked stale by estimate-source; numbers provisional. Existing os.Pipe/exec.CommandContext/procutil remove any novel library implementation. Owner lease design uses the resolved spec discount (1.0h x0.2); legacy cleanup retains mid-density design (0.8h x0.5) until #583 answers. Memory attribution has three unresolved decisions at 0.15h each (retention versus spike, which scan dominates, and the resulting work bound), with code-heavy implementation 0.45 x1.5 x0.4=0.27h. Other implementation values below use v3.1's 40% scaling once. Familiar stack, 15% design buffer for this durable plan.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.40 impl=0.32
item: real-api-discovery design=0.00 impl=0.24
item: method-b-decisions design=0.45 impl=0.27
item: smaller-go-module design=0.06 impl=0.20
item: cross-cutting-refactor design=0.10 impl=0.20
item: atlas-docs design=0.05 impl=0.08
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 3.33
```

Design subtotal 1.30h plus15%=1.495h; implementation subtotal1.83h; total3.325h, rounded3.33h. Respectively: lease, fenced cleanup, live-profile discovery, memory attribution/fix, cancellation integration, launcher ownership sweep, docs, single close review.

## Plan

See [implementation plan](../plans/000617-title-process-lifetime-plan.md). TL approval required before code.

- [x] Attribute Go heap/allocations and reconcile #583 cleanup contract.
- [ ] Red tests, then wrapper-owned title lease and cancellation.
- [x] Measured memory fix and bounded soak.
- [ ] Identity-fenced legacy cleanup, live acceptance, normal SDLC close/landing.

## Log

### 2026-10-16

### 2026-10-16 — planning investigation

Claimed in pair:6 and started planning from main; initial start-plan refreshed the ariadne substrate and refused until rerun, then created the issue branch. No implementation changes. Detached launcher ownership and 60s/five-miss session probes do not bind title lifetime to the wrapper. PPID 1 alone is not dead-thread proof: live ops wrapper 43773 still corresponds to title 25522. Native heap/vmmap succeeded: 24.5 MiB current physical footprint, 271.8 MiB peak, about 283 MiB ps RSS, only 65 KiB native malloc. Go heap attribution remains open; native heap is not a Go pprof capture. Repeated uncached inventory/usage scans are an allocation candidate, not an established retained leak. Private evidence paths/hashes and proposed red tests are in the durable plan. Sent #583 coordination to pair:4 (receipt 6a93aca7-49d2-4fff-8a4f-07677803c112), still queued at the last read. ARCH-DRY favors existing activity caches; ARCH-ORDER/SECURE require EOF lifetime and exact generation-fenced cleanup. Await TL plan/profile-method decision before code.

## Revisions

### 2026-10-16 — plan-first investigation

Added the durable implementation proposal and explicit evidence gates. Original Spec and Done when remain the contract; memory cause, Go-profile method, #583 cleanup seam and live rollout are recorded as unresolved rather than assumed.

### 2026-10-16 — plan handoff unavailable

Plan committed at be96916a6. Couch socket disappeared before handoff. The send to hub returned uncertain outcome with receipt 15b2288f-f3ba-415e-bc7b-192ccf543f9c; status lookup also could not connect. No automatic resend, no alternative channel, no implementation. TL has not been confirmed to receive this plan; query that receipt before any later send. Earlier pair:4 coordination was queued, not confirmed read. Await transport recovery and TL review.

### 2026-10-16 — TL approved plan

After restart the plan report was resent (09bd705d-e7a6-48a7-b2a2-e9f6b1804a90). Verified TL approval receipt 72aed150-a96e-46b8-8e68-8bb2dcbaa16f: proceed to change-code, implement wrapper pipe lifetime with red tests first, choose controlled production-path replica profiling using real ledger/catalog, no diagnostic rollover of live titles. Cleanup coordination with #583 remains pending. Full flow requested for this cross-process ownership and memory change.

### 2026-10-16 — plan gate round 1

Plan-quality REWORK: PQ-1 requires cancellation to cover in-process inventory/usage, not only sleep and commands; PQ-2 requires named function-level testing strategies. Revised plan compresses test inventories and specifies cooperative context throughout query/range IO plus a proposed bounded CLI-process exit fallback for indivisible decoding. Requested TL approval of that fallback (submitted receipt d4707560-510f-42d6-a747-eb62e6e2dcd0). Profiling must copy the real ledger/catalog first: QuerySession can advance catalog state, so a live PAIR_DATA_DIR would not be read-only. Real ops ledger is 10,323,719 bytes, catalog 37,403 bytes. No implementation changes.

### 2026-10-16 — first implementation checkpoint, incomplete

Full-flow change-code exited0 after plan-quality CLEAN and estimate-quality INFO (3.33h optimistic for cancellation and operational acceptance; conditional memory allocation). Wrapper owns sole CLOEXEC pipe writer and a reaped title child; launcher create/attach detached spawns removed. Actual title command checks wrapper PID/start and pipe descriptor, observes EOF independently, cancels the RunContext loop and joins cooperatively. Three production SIGKILL/inheritance/replacement regressions were RED (no wrapper title, then title survived death) before GREEN (~25–38ms observed exit). Corrected an initial test oracle mixing Identity and StrictIdentity, then reran the behavioral RED/GREEN. Native child ignores HUP and remains alive to prove writer non-inheritance; sentinel proves no global cleanup. Removing EOF cancellation makes SIGKILL regression fail; removing cancellable range adapter reads10MiB versus one64KiB range, then restoration passes. Cancellable shared context projection and query/range boundaries are wired; indivisible decoding fallback remains pending TL review.

Controlled replica profile baseline:~644MB allocated/poll pair, postGC296KB;66% ledger read/decode,34% full usage scan. Existing OwnerLaunchCache+ActivityCache reuse passes real-file RED/GREEN and cuts unchanged warm cost to~218MB/poll. No usage cache added. Actual wired replica under active native writes exposed full proof fallback as73.54% of8,552.73MiB sampled allocations, with individual polls allocating~2.78GB. Native transcript records/event construction are transient; postGC319KB, HeapSys259.6MB mostly released. Stop further memory changes and re-plan this cause with TL; an unchanged-usage cache would miss it. Original data files unchanged by replica; catalog writes occur only in copied Pair scope. See plan revisions/private profile paths.

Validation so far: titlepoller/contextcmd/launcher package tests pass (launcher33.769s); focused race tests titlepoller2.548s, contextcmd1.148s, wrapcmd8.222s; artifactpath production inventory passes1.486s; diff check passes. Full suite, complete fallback/blocked-command tests, memory bound/soak, legacy cleanup, live rollout and SDLC close/landing are outstanding. #583 coordination messages are still queued; inspected its committed ProcessTable/identity fence as read-only evidence, no nvim/cleanup changes. No live title rollover, restart or cleanup performed.

### 2026-10-16 — test isolation incident

The new wrapper fixture initially inherited live PAIR_SCROLLBACK_EVENTS_PATH despite temporary HOME/data/tag. After adding a real temporary --scrollback-log for positive owner admission, openWitnessedCapture used the inherited live events path and truncated this slot's /Users/xianxu/.local/share/pair/repos/50f39cf60172c338/scrollback-1-pair-55-codex.events.jsonl around14:58 (observed362bytes, mtime1792187898.8101048). Affected run had already finished when stopping was considered. No speculative reconstruction/restart was attempted; prior capture bytes are unavailable. Reported to hub as incident receipt617dbf80-34ef-43c9-a3db-7bef61156bb7. Native fake agents do not access native transcripts; no native transcript loss observed. Corrected the fixture to a minimal PATH/TMPDIR/locale/TERM allowlist and explicitly supplied only disposable Pair paths. Added integration sentinel plus positive local capture-write control, and recorded the prevention rule in lessons. This capture gap is distinct from the original unavailable previous-screen evidence for #544.

### 2026-10-16 — isolation correction and additional admission guard

Sentinel isolation regression now passes with a positive disposable witness-write control. Mutation that restores inherited events-path authority fails by overwriting only the test's foreign sentinel; restoring the minimal environment passes. Generic inherited Pair environment without an exact selected scrollback path also had a RED regression (spawned a title); the title spawn now requires the wrapper's explicit scrollback path to equal artifactpath's selected owner capture. This preserves explicit in-pane restart/legacy launch support while preventing a generic wrapper from borrowing owner metadata. Title-process source is registered as a resolved scrollback consumer. No live data was used by the corrected fixtures.

Final checkpoint checks after fixture isolation/admission guard: focused race titlepoller5.221s, contextcmd1.265s, wrapcmd14.034s; artifact production inventory6.344s; all passed. No full-suite or completion claim.

### 2026-10-16 — awaiting revised memory decision

Incomplete implementation checkpoint4863c21d457f6a21a0362f54f9a261d47177c740 committed; clean branch. Sent the active-write full-proof revalidation finding and revised streaming-projection proposal to hub, receipt d01d89b6-8321-472c-8512-12943be9bfbb (queued position2 at admission). This supersedes the scalar usage-cache proposal; do not implement that proposal merely on a delayed approval. Pending: TL revised memory scope/design decision, bounded CLI teardown fallback decision, pair:4 cleanup seam response; all final acceptance remains outstanding. Earlier no-delivery transport failure and later test-capture gap stay recorded, with no automatic resend/reconstruction.

### 2026-10-16 — structural isolation directive and read-only backup check

Verified TL receipt e6510de0-274d-4237-bf31-8f086d531634: replace per-fixture sanitization with one shared empty-environment helper used by all Pair/wrapper subprocess tests and a guard like #602. Added corresponding plan revision; implementation underway with behavioral RED/GREEN. Pending memory design/fallback remains separate.

Read-only inspection of configured /Users/xianxu/.local/share/pair/archive/catalog.jsonl found no cataloged backup of repos/50f39cf60172c338/scrollback-1-pair-55-codex.events.jsonl. Covered all complete records: 40 bundles, 6,993 file entries, no numbered catalog shards, no matching basename or Pair-data source root. Kinds: Claude9, Codex transcripts23, Qoder2, Codex DB6; verification times 2026-10-16 09:10–11:22 PDT. Catalog SHA256 d91381c469a0cedfaa4acaea2c7f19fd9f72aabfa38561ef79e0099c2bcbaa24. No backup, restore, decryption or live writes performed. This establishes absence from the configured completed catalog; independent backups and unverified staging were not searched.

### 2026-10-16 — structural subprocess isolation checkpoint

Implemented the TL-directed class fix: tests/lib/pair-test-env.sh is the sole empty-environment owner for Go/shell/Python; testpair.Environment/EnvironmentFrom and ApplyEnvironment consume it. Only PATH, a disposable HOME, the explicit helper path and fixture-authored entries cross the boundary. Migrated direct Pair commands, generated launcher stanzas, wrapper selfexecs, native hook commands, nested terminal fixtures and shell/Python adapters. Indirect owner-wrapper fixtures use the same outer boundary; audited all seven AddOwnerWrapperPane call sites. Shell entry uses a fixed shebang/bootstrap prelude and always cleans before running its trusted local body; public argv cannot skip isolation. The source guard checks actual Go/Python provenance and shell entry contracts, with no blanket helper-string exemption. A read-only audit ran it against 38 original complete source bodies (11 Go including native hook, 27 shell including nested shared-builder contract); all were detected. Durable synthetic controls cover the original recipes and review findings without depending on Git history.

Ad-hoc fresh-eyes review found unsafe conditional aliases, late assignments, mapping updates/alias mutation, exception/finally flow, literal wrapper omissions and a public bootstrap-marker bypass. Verified negative controls, then corrected the class using ordered Go provenance, a pure Python AST analyzer with conservative joins, and unconditional shell isolation. Review is now clean within the documented supported recipes. Test boundary itself preserves foreign sentinels while proving disposable writes; ApplyEnvironment replacement has mutation proof. No production Pair behavior changed in this checkpoint.

Final verification: full testpair package with race detection passed14.174s; production artifact inventory passed1.758s; diff check passed. Actual wrapper SIGKILL/noninheritance/replacement and capture sentinel race tests passed14.700s; generated layout wrapper, protocol skew and selected-root race tests passed20.898s; nested wrapper/stand-in race tests passed5.291s. Continuation writer acceptance passed24.234s, public Pair/route tests29.867s, shipped probe tests7.580s. Notification PTY conformance passed (native-console test compiled but remained opt-in/skipped); no live native rollout claimed.

Shell sweep retained all logs at /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-617-isolation-checks.S2Vwcq. Initial final-body sweep: 24/27 passed; nested builder fixture failed entry-path lookup (fixed to original $0, standalone rerun passed); watcher timed out240s downloading cold Go modules (replaced go run with existing shared candidate, causal-round rerun passed); pair-rename failed T7 legacy pair- prefix expectation. Original pre-change rename script fails identically in the same empty environment; remaining T1–6/T8–9 pass57 assertions. Thus26/27 scripts pass after the two branch fixes; the rename baseline failure remains recorded, not hidden by a retry. Initial logs/final-results.json and followups.json remain separate. No leftover matching watcher shell/go-run process was observed after timeout; no global cleanup performed.

This is an incomplete #617 checkpoint: full declared integrated suite, native live rollout/cleanup, CLI indivisible-decode fallback and measured memory bound remain outstanding. Revised memory-design and fallback decisions still await TL; nothing in the test-isolation directive approves those changes. No SDLC close, PR or landing.

Structural checkpoint 7a39140d80b126fea2df78048c3663256dfe7081 committed with clean tree. Outcome report to hub accepted15:26:48PDT as receipt57990322-b41f-4858-928e-a276592c5e01, still queued position5 when inspected. Queued is not submitted/accepted work: retain this delivery obligation, inspect receipt before any resend, and do not resend blindly. Pending memory/fallback/cleanup decisions remain separate.

### 2026-10-16 — measured one-admission re-plan, awaiting approval

Verified TL36649870-8348-4568-b299-c02445643d9d: third memory delta means re-plan; immediate os.Exit on EOF/owned command groups supersedes pending teardown fallback; allowlist+guard already complete. Measured before production code: default loop three60s sleeps/three active activity+usage pairs (configured1 poll/min, actual60s+work); isolated unchanged full validator cost1.908s and1,025.6MiB cumulative allocations reading83.0MiB, cold owner lookup separately185ms/203MiB. Post-GC measurement-process live heap316KiB, not yet a title bound. Go1.27.1/GOMAXPROCS3/GOFLAGS -p=3, shared host load~33. Private overlay/stats/hash and exact metrics are in plan revision; only copied Pair ledger/catalog and read-only native data used.

Sent one-paragraph plan ffff4ada-6afe-41fb-b745-4eb23b6bf8eb (queued position7): await initial binding under lease, prove once for exact wrapper invocation/ordinal, seed compact usage/activity and discard full records/events, then bounded appended-tail projection without shared-query fallback. Pin inode/offset and existing provider rules; exit on replacement/shrink/parse/identity failure; cheap owner-ledger fence handles real legacy-conflict/supersession possibilities. Explicit approval needed for native append-only assumption (hidden overwrite/regrow is not detectable by inode+offset) and proposed32MiB partial-record ceiling. No production changes this turn; no immediate-exit implementation yet, no new change-code gate, suite, close or landing.

### 2026-10-16 — consolidated decisions and separate rename filing

Verified TL556ee0bd-57cc-4a3c-a0cc-62ef9bad3d61: no bounded fallback or added context plumbing; immediate EOF process exit, owned command groups, atomic durable writes. Appended the removal/atomic-write requirement to the pending plan; no production changes. Measured one-paragraph memory plan ffff4ada-6afe-41fb-b745-4eb23b6bf8eb now reads submitted, so it was not resent; approval remains pending.

No matching rename T7 issue was listed. Filed #621 Rename shell test fails legacy prefix case via sdlc issue new --no-claim, published card/details on origin and verified status open/unattributed/no branch. This slot continues only #617; no rename implementation started.

Direct cleanup predicate request to pair:4 is5ebc255a-774d-4f4c-9775-bc628b5c0610, delivering while prior automatic input clears. Earlier coordination receipt is unavailable, not evidence of a failed effect. With no answer, inspected TL fallback ClassifyUnattributed as it stands: it is an unproven RSS alert classifier, explicitly grants no ownership, accepts agents/zellij/nvim--embed, excludes Pair title binaries, and does not establish wrapper absence. Kept it unchanged. Read-only ps confirms25522 is PPID1 with comm ending bin/pair, hence excluded. No legacy kill predicate or cleanup is inferred from PPID1/RSS; the missing compatible owner predicate is being reported to hub.

### 2026-10-16 — revised plan approved

Verified TL approval253d1a36-2b1d-4ebd-b1ca-232f683bd8ca of measured re-planffff4ada. Added required 4KiB trailing hash before every tail read and explicit terminal mismatch behavior. Cleanup now uses exact title TAG plus absent exact-TAG live wrapper in one sample, legacy cutoff and PID/start fencing; earlier ClassifyUnattributed fallback is superseded. Report-only dry run must reach TL for comparison before signals. Full code gate will be rerun before red-first implementation. Historical capture gap remains documented; no reconstruction or restore.

### 2026-10-16 — revised code gate and red-first implementation

Full change-code completed at a5c9adf4 with plan-quality CLEAN. Estimate-quality INFO identifies the original3.33h derivation as predating isolation, one-proof tails and revised teardown; it remains a historical forecast, not remaining work or measured actual. Actual will be measured by close. Projection RED observed unchanged native rereads (1→4) and failure to exit on shrink. Atomic-publication RED observed old PID/cmux inode modification; GREEN uses existing WriteAtomic. Singleton RED demonstrated a legacy pidfile suppressing the leased generation; removal passes. Launcher cleanup RED demonstrated both ports shelling kill from numeric metadata; GREEN leaves process and title-owned metadata untouched, with lifecycle cancellation preserved. Focused launcher tests pass. EOF mechanism question361802ad-df9a-4087-96eb-2daabd2384a3 submitted to TL: owned process-group SIGKILL is race-free but supersedes literal os.Exit; dependent effect remains pending that clarification while independent tests/code progress.

### 2026-10-16 — bounded tails verified; immediate EOF effect still pending

Implemented one-admission exact-ordinal TitleProjection with unchanged existing authorization, compact provider state,64KiB appended reads,32MiB partial-record ceiling,4KiB SHA-256 boundary checks before every chunk, and terminal authority/artifact contradictions. Six-provider parity, delayed binding, malformed/split tails, replacement/shrink, per-chunk mutation, admission drift and growing-state tests pass. Ad-hoc fresh review found missing post-tail owner fencing and profiling of stopped no-ops; both corrected. Supersession inside ReadAt now prevents publication; profile requires nonterminal positive activity/usage. Hash removal and final-owner-fence removal each make their regressions fail. Added lessons. Atomic PID/cmux writes, strict ordinal, no legacy singleton suppression and bounded frame cache are verified. Launcher numeric PID metadata confers neither signal nor removal authority.

Frozen-tree verification: projection race checks sessioninventory11.887s/titlepoller8.137s; entire titlepoller race excluding the explicitly pending EOF tests7.609s; wrapper Title/Poller race11.565s; focused launcher race1.293s; exact artifact classification3.375s after fixing stale binding declarations; environment source guard4.210s (earlier full testpair race84.425s). Two concurrent compilation attempts observed intermediate API/helper transitions and were rerun after files froze; they were not behavioral failures. Immediate EOF tests remain intentionally RED: blocked proof fails after1.66s and owned command descendant survives EOF (1.40s). No full suite, close, PR, deployment or live cleanup claimed.

Final controlled real-source production adapter replica: copied Pair ledger/catalog; native transcripts read-only; exact current ops launch ordinal;10 immediate polls, positive activity/count346k each. Startup3.377345875s,1,308,577,072B cumulative allocation above baseline. Nine warm unchanged polls allocate29,904–31,952B and take161.875µs–1.130834ms after the final double owner fence. PostGC334,920B while keeping runtime/projection reachable. This replaces the earlier17–19KB pre-final-fence result. Private profiles/stats: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-617-admitted-verified.MX0xD5, stats SHA256cb6dac640e5a91c250e57188d8e5024dc380ac196236a94891976c8b5d2563b7. Startup still pays one full proof; no live title rollover or original RSS composition claim.

Accelerated production-adapter real-file growth soak uses a disposable10MiB ledger and validated Codex fixture:100 polls,100,000 appended usage records,10,600,000 appended bytes, no warm ledger/catalog reads; native IO bounded by appended bytes plus boundary windows. Runtime/projection remain reachable. PostGC heap306,168B baseline,414KB around poll10,416,960B at poll100 (417,360B after recording); cumulative warm allocation166,367,992B. Ran1.77s, not a100-minute live soak. Private overlay/profiles/stats: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-617-growth-verified.5jB66T, stats SHA256fdfd69fb78e617972464f7e9e33041366e37ebf27b42770f26482b752ad3a7c2. Shared initial validators and unsupported usage rules remain unchanged.

Legacy report-only implementation uses one complete native sample, exact executable/title argv and reconstructed exact wrapper artifact-owner TAG; unknown preserves processes. Approved execution helper resamples and intersects approved PID/start/TAG/agent before identity-fenced TERM/wait/KILL. Linux observation deliberately fails closed because current procutil.Table can skip unreadable rows; no shared classifier was widened. On Darwin the22:48:54Z sample had977rows,29 production titles:5eligible,24kept for live exact-TAG wrappers. Eligible generations:16985/1792013619.751720 (1-pair-47),17402/1792013624.472303 (1-ariadne-32),27555/1791936005.707366 (couch-9d6fdf5eecace248),96593/1791306631.272585 (1-parli-12),96914/1791306635.674554 (1-ducks-11). Another worktree-binary title excluded. Disclosed pre-deploy cutoff is report-helper start1792190934.318226; lease build not rolled out. Report SHA256900b665f1a758c9ae6b58fd02fd6e57fc93447b2cdd22e02307b183c8b8520a0 at /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-617-legacy-report.KILlHq/report.json. Submitted to hub as93d4945d-42bd-48e1-b5fc-2874157f73d1 for comparison with original28; NO live signals or metadata writes.

Remaining dependency: TL answer to submitted361802ad-df9a-4087-96eb-2daabd2384a3. Literal immediate os.Exit cannot kill a concurrently-starting command without a registry publication race; killing the title's already-owned process group is immediate and race-free but terminates title before literal os.Exit. Await this narrow interpretation before EOF effect/context removal, then make both red tests green, integrated verification/close, and TL-coordinated rollout/cleanup. Existing screen/capture gaps remain as recorded, no reconstruction. #544 already landed; do not reopen it.

### 2026-10-16 — immediate owned-group EOF teardown approved

Verified submitted TL receipt2ed371a2-2749-401b-a235-03c3f7121ed4, exact ops:0 to current pair:6. TL accepts SIGKILL of the title's own Setsid process group on lease EOF as the immediate teardown effect, superseding literal os.Exit. Commands inherit that group; one kernel effect kills title and commands even when Start races EOF. No Start/register lock, context plumbing, decoder join, grace period or bounded CLI fallback. Validate title is its own process-group leader before arming this effect; never kill a borrowed group. All title-reachable durable publication remains temp+rename. Existing blocked-proof and stubborn-command descendant RED tests precede implementation. This resolves question361802ad; legacy cleanup comparison remains separate and no live signal is authorized by this teardown decision.
