---
collaboration-mode: discussion
---

# Pair Go lint catalog implementation plan

> **For agentic workers:** Follow AGENTS.md §3; bounded independent fixture audits may delegate, while SDLC owns the review boundary. Use superpowers-executing-plans for implementation.

**Goal:** Enforce ariadne's mechanical Go mistake catalog in Pair's ordinary repository test and resolve every baseline finding without weakening test intent.

**Architecture:** Import the existing stdlib-only golint package through Pair's existing ariadne module dependency. A test-only repository scanner enumerates tracked Go files once and calls the upstream pure checker. Fix genuine mistakes at their source; give semantic exceptions local reasoned suppressions, which the catalog itself rejects when stale.

**Tech stack:** Go standard library AST, existing Git, existing Pair test helpers and ariadne golint.

## Consumption decision

Recommend a module pin at ariadne commit `81c832d5e0008578d7a03783a0d4055cc6358000` (latest catalog source at planning). Go derives the pseudo-version; no checkout replace, vendoring, local absolute dependency, or hand-copy of rules. Pair already requires ariadne for layergraph. ARCH-DRY: the catalog remains owned upstream and updates are explicit version changes.

Alternative: export source through weave. The package is not currently exported, so this would require a peer base-layer change and protected manifest changes plus copied package layout. That is broader than consuming the existing Go API. Reject an independent Pair rule implementation because it would drift.

The module update also advances runtime layergraph/workspace code (primary-checkout fallback for absent relative substrates). Qualify existing Pair consumers in slotdeps.go and slot_prepare.go, including linked-worktree fallback behavior. If the dependency introduces a behavioral incompatibility, stop and bring it to the hub rather than patching unrelated product policy.

## Core concepts

### Pure entities

| Name | Lives in | Status |
| --- | --- | --- |
| No new pure entity | Upstream golint.Check, Catalog and Finding reused unchanged | reused validation only |

Check receives one filename and source bytes; its five rules and suppression grammar remain entirely upstream. No new rule, allowlist, baseline exemption, matcher, or suppression parser is planned.

### Integration points

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| Pair catalog repository test | internal/repolint/catalog_test.go | new | Git tracked-file inventory, source reads, golint.Check |
| Ariadne module pin | go.mod and go.sum | modified | published immutable module revision |

Repository root comes from internal/reporoot.Current. One Git ls-files -z invocation returns paths, then each eligible file is read and checked once. Match upstream exclusions for workshop prose and testdata, whose intentional broken snippets are fixtures, and generated Go is skipped by the checker. Include tracked third_party and probe Go sources; do not silently exclude noisy directories or nested modules. Missing tracked files deleted in the working tree are skipped as upstream does; all other read/parser/inventory errors fail. Empty inventory fails.

Test the inventory seam against a disposable real Git repository with tracked valid and invalid files, an untracked invalid file, excluded fixture data and a generated source. Git is owned and disposable here, with no remote or operator state. Reuse upstream checker tests rather than cloning their assertions into Pair.

## Baseline and sweep contract

Read-only scan: 2,547 tracked Go files, five rules, 181 findings across 91 files, 714 ms on this host. Retained diagnostic: /tmp/pair665-catalog-initial-hits.log. Counts: fixed sleeps103; tmp literals39; shortened deadlines24; tight wall-clock11; test builds4. Exact file inventory follows below.

For every hit record fixed versus intentionally suppressed and the preserved assertion in the issue Log. No blanket file/package suppression. Keep all assertions and input domains.

- Fixed sleeps: replace readiness guesses with an existing event, completion channel, explicit clock seam or testwait condition. Sleeps whose duration is itself the subject, an intentional sample window, or controlled child behavior can retain a specific adjacent suppression explaining that role. Do not just reduce sleeps below the matcher threshold.
- Short deadlines: establish readiness before firing cancellation/deadline through a controllable seam; use existing helpers where possible. Suppress only when the test is directly checking a configured bound without racing readiness. Do not raise production deadlines or turn meaningful failures into skips.
- Tight elapsed bounds: assert the event or outcome; keep elapsed bounds only when the bound is the explicit subject and the suppression explains why it is needed. Preserve hang guards through testwait rather than guessing speed.
- Temporary paths: use owned TempDir/MkdirTemp or os.TempDir for actual temporary storage. Native Unix socket path limits require checking length before replacing compact roots. The production UID socket namespaces couchmessage.SocketPath and notifytransport.rootDirectory deliberately agree across processes with differing TMPDIR; preserve this protocol identity with documented suppressions and context tests, rather than splitting peers by environment. Literal parser/address test inputs can remain with an exact semantic reason.
- Test builds: inspect all four call sites. Reuse the Pair binary helper where the subject is Pair; genuinely different executable/instrumented builds need one suite-owned build or a reason why compilation itself is the assertion. Do not substitute the wrong binary or production build flags.

If a genuine fix requires a new product mechanism, changes user-visible behavior, needs native validation, or cannot retain its oracle without a deeper fixture re-plan, stop and send a scoped revision to the hub. Do not rubber-stamp findings with migration suppressions.

## Execution steps (one atomic close boundary)

- [ ] Resolve and pin the chosen ariadne revision with Go module tooling; inspect resulting dependency changes.
- [ ] Add the inventory integration regression first; prove a tracked violating source is reported, whereas untracked/excluded/generated controls are not. Verify errors and empty inventory refuse.
- [ ] Add the ordinary repository test and run `go test ./internal/repolint -count=1 -v`; retain the initial red baseline before sweeping.
- [ ] Sweep path/build hits and event/deadline/timing hits in bounded package groups, logging each disposition. Use TDD for fixture behavior changes: demonstrate the bad ordering or mutation first, then fix and retain the regression. Commit coherent groups with the required trailers.
- [ ] Run affected targeted packages normally and with race where concurrency changes. Native-gated tests remain gated; ask the hub for a bounded native lane only if needed, never borrow the operator's sessions or build.
- [ ] Run `go test ./cmd/internal/couchcore -run '^(TestDeclaredDeps|TestRunPrepare)' -count=1` for dependency compatibility; add a disposable linked-worktree anchor regression if existing tests do not cover the updated resolution path.
- [ ] Run the catalog clean; temporarily introduce one real tracked violation and prove the normal repository test goes red, then restore and prove green. Independently prove a reasonless/stale suppression fails via upstream API if integration coverage does not already establish it.
- [ ] Document version ownership, five seed rules, scanner scope, new-rule workflow and local suppression contract in atlas/go-lint-catalog.md; link from atlas/index.md. Update lessons from any review findings.
- [ ] Push the issue branch and run the recorded full suite with sdlc local-check run --name suite. No Makefile, CI, protected local-gate config or weave edits are planned.
- [ ] Close with the SDLC fresh review, fix genuine findings, enter the landing line after SHIP, and report the PR and Git-verified landing outcome to the hub.

## Operating envelope and state

ARCH-PURE: AST semantics remain upstream pure logic; repository discovery is a thin test-only IO shell. ARCH-PURPOSE: enumerate every tracked eligible source and every finding; no permanent baseline debt.

ARCH-CONSTRAINTS: batch test only, no interactive hot path. One Git listing plus one source read and Check per file; five bounded AST passes per source, memory scoped per file. Report file/rule counts and elapsed time, without fragile time thresholds. Inventory tests at N and2N count reads/checks and require exactly N and2N, preventing accidental repeated full scans.

ARCH-SECURE: filenames come from NUL-delimited Git output; argv is structured, parser errors fail and diagnostics are repository-relative. No credentials or operator resources are accessed. ARCH-MOCK: the only external tool seam uses disposable real Git fixtures; no service/network protocol is introduced.

ARCH-ORDER: scanner holds no state between runs; each invocation returns either all findings or an explicit inventory/read/parser failure. Fixture timing repairs will name their event ordering before code and join everything they start. ARCH-FUNERAL: temporary fixtures belong to t.TempDir; no persistent baseline, cache, daemon or log family is created. Existing module cache and SDLC evidence retain their existing lifecycle.

Actions: edit the reported source or justified local suppression, then rerun the same test. Clean -> ordinary passing test; violation -> path/line/rule/incident and failed test; unknown inventory/read/parse -> explicit failed test, repair the named input and rerun. No product data or runtime state is changed; no loss or alternate retry state is introduced.

## Baseline file inventory

Each number is a finding count, not a proposed suppression count.

| File | Findings |
| --- | ---: |
| cmd/couch/main_test.go | 2 |
| cmd/internal/broadcast/cloudflared_test.go | 4 |
| cmd/internal/broadcast/host_test.go | 1 |
| cmd/internal/broadcast/manual_test.go | 1 |
| cmd/internal/broadcast/notes_test.go | 1 |
| cmd/internal/clipcmd/admission_live_test.go | 2 |
| cmd/internal/clipcmd/cli_admission_test.go | 1 |
| cmd/internal/couchcmd/message_discovery_test.go | 1 |
| cmd/internal/couchcmd/message_lifecycle_test.go | 2 |
| cmd/internal/couchcmd/message_service_test.go | 4 |
| cmd/internal/couchcmd/messages_compat_test.go | 1 |
| cmd/internal/couchcmd/messages_test.go | 2 |
| cmd/internal/couchcmd/owned_query_test.go | 1 |
| cmd/internal/couchcmd/recovery_acceptance_test.go | 1 |
| cmd/internal/couchcmd/resources_cli_test.go | 1 |
| cmd/internal/couchcmd/resources_projection_test.go | 1 |
| cmd/internal/couchcmd/session_sync_test.go | 2 |
| cmd/internal/couchcore/coldresume_birth_test.go | 2 |
| cmd/internal/couchcore/conformance_live_test.go | 1 |
| cmd/internal/couchcore/continuation_conformance_live_test.go | 1 |
| cmd/internal/couchcore/couch_test.go | 3 |
| cmd/internal/couchcore/launch_timing_test.go | 1 |
| cmd/internal/couchcore/launchfailure_capture_test.go | 1 |
| cmd/internal/couchcore/never_turned_test.go | 1 |
| cmd/internal/couchcore/operationdispatch_test.go | 1 |
| cmd/internal/couchcore/park_wait_test.go | 1 |
| cmd/internal/couchcore/provision_budget_test.go | 1 |
| cmd/internal/couchcore/quiesce_conformance_live_test.go | 1 |
| cmd/internal/couchcore/recovery_conformance_live_test.go | 1 |
| cmd/internal/couchcore/registration_phase_test.go | 2 |
| cmd/internal/couchcore/relaunch_test.go | 4 |
| cmd/internal/couchcore/startup_detach_timing_live_test.go | 1 |
| cmd/internal/couchcore/terminal_incarnation_test.go | 1 |
| cmd/internal/couchcore/terminal_warm_fixture_test.go | 1 |
| cmd/internal/couchcore/threadstore_preview_test.go | 3 |
| cmd/internal/couchcore/warm_attach_registration_test.go | 1 |
| cmd/internal/couchcore/warm_attach_repair_test.go | 4 |
| cmd/internal/couchcore/warm_failure_test.go | 2 |
| cmd/internal/couchmessage/background_protocol_test.go | 2 |
| cmd/internal/couchmessage/drive_endpoint_test.go | 2 |
| cmd/internal/couchmessage/session_transport_test.go | 3 |
| cmd/internal/couchmessage/transport.go | 1 |
| cmd/internal/couchmessage/transport_test.go | 1 |
| cmd/internal/couchtty/console_broadcast_test.go | 3 |
| cmd/internal/couchtty/console_castcodes_test.go | 1 |
| cmd/internal/couchtty/console_live_test.go | 2 |
| cmd/internal/couchtty/console_notes_test.go | 1 |
| cmd/internal/couchtty/console_pointer_test.go | 5 |
| cmd/internal/couchtty/console_webcast_test.go | 1 |
| cmd/internal/couchtty/notification_pty_test.go | 2 |
| cmd/internal/couchtty/terminal_native_test.go | 1 |
| cmd/internal/couchtty/terminal_pressure_test.go | 1 |
| cmd/internal/diagnosticlog/proof_test.go | 1 |
| cmd/internal/historyarchive/codec_live_test.go | 1 |
| cmd/internal/hostty/hostty_test.go | 1 |
| cmd/internal/launcher/osruntime_test.go | 1 |
| cmd/internal/launcher/session_quiescence_live_test.go | 1 |
| cmd/internal/launcher/workbench_attach_invariant_test.go | 1 |
| cmd/internal/layoutcmd/fullscreen_reply_live_test.go | 2 |
| cmd/internal/layoutcmd/rightpane_rungs_live_test.go | 3 |
| cmd/internal/notifycmd/run_test.go | 1 |
| cmd/internal/notifytransport/address.go | 1 |
| cmd/internal/notifytransport/transport_test.go | 2 |
| cmd/internal/panebirth/panebirth_test.go | 1 |
| cmd/internal/procutil/resource_darwin_test.go | 2 |
| cmd/internal/termcmd/fullscreen_test.go | 1 |
| cmd/internal/terminal/presenter_stall_test.go | 1 |
| cmd/internal/terminalcapture/recorder_test.go | 2 |
| cmd/internal/ttyio/writer_test.go | 1 |
| cmd/internal/webcast/chrome_loop_test.go | 2 |
| cmd/internal/webcast/live_drive_nav_test.go | 4 |
| cmd/internal/webcast/live_drive_real_test.go | 10 |
| cmd/internal/webcast/live_drive_test.go | 9 |
| cmd/internal/webcast/live_notedraft_test.go | 2 |
| cmd/internal/webcast/live_record_test.go | 18 |
| cmd/internal/webcast/live_replaced_test.go | 4 |
| cmd/internal/webcast/live_test.go | 1 |
| cmd/internal/webcast/lock_live_test.go | 1 |
| cmd/internal/webcast/server_board_test.go | 2 |
| cmd/internal/webcast/server_drive_test.go | 2 |
| cmd/internal/webcast/server_notedraft_test.go | 1 |
| cmd/internal/webcast/viewer_annotate_live_test.go | 2 |
| cmd/internal/webcast/viewer_live_test.go | 1 |
| cmd/internal/wrapcmd/agent_drive_conformance_test.go | 1 |
| cmd/internal/wrapcmd/agent_drive_live_test.go | 2 |
| cmd/internal/wrapcmd/agent_drive_virtual_conformance_test.go | 1 |
| cmd/internal/wrapcmd/notification_output_test.go | 1 |
| cmd/internal/wrapcmd/peer_live_test.go | 1 |
| cmd/internal/wrapcmd/peer_recovery_test.go | 1 |
| cmd/internal/wrapcmd/terminal_tail_test.go | 2 |
| probes/zellijcalls/main_test.go | 2 |

## Revisions

### 2026-10-18 — R1: hub-approved staged enforcement

Reason: authenticated hub delivery2b583e46 approves module consumption and the dependency qualification, but requires two milestones so the 78 non-sleep findings do not wait for all103sleep repairs. This revision supersedes the single-close boundary and the original prohibition on a baseline file. It authorizes exactly one temporary, shrink-only sleep debt file, never blanket inline suppressions.

- M1: scanner/repository test, module pin, all78non-sleep dispositions, atlas, and explicit103-site sleep list; targeted regression/race and recorded suite, milestone review, then land the enforcing guard.
- M2: eliminate the sleep debt, prioritizing packages associated with #659/#662/#623/#485, using explicit observable signals. Preserve assertion domains, retain deterministic ordering controls and join fixture work. Each completed fix removes its debt entry. No new sleep debt or deadline changes.

Sleep-site model (new pure entity in internal/repolint/catalog_test.go or a colocated test helper): repository-relative path, enclosing function, canonical AST rendering of the Sleep call, and occurrence ordinal within that function for identical calls. It is independent of incidental source line movement. One tracked JSON file internal/repolint/sleep-debt.json lists sites. The scanner matches only test-fixed-sleep findings to those exact sites, reports every unlisted sleep as failure, and rejects duplicates and unused entries. No other rule can use the file.

Shrink ratchet: read the version of that same file at the branch/main merge base using the configured main tracking ref. Current entries must be a subset, never swapped for new sites; absence/read/JSON/Git failures refuse. During initial M1 creation only, verify the entries against the original tracked source baseline at the committed pre-implementation design anchor0f8e23d0a142f962fc011875c92da68216762058; never regenerate the admitted set from the candidate being tested. Main CI already fetches full history, so this requires no protected workflow change. After M1 is on main, the admitted set comes from main's merge base; empty list is the final state. Regression matrix: existing exact site accepted; newly added site rejected; deletion accepted only after its sleep is fixed; stale/duplicate/malformed/path-moved entries rejected; same-count replacement and addition rejected; source line-only shift keeps identity. Use real disposable Git histories for ratchet tests and pure site/set checks for matching.

The helper remains test-only, catalog logic stays upstream (ARCH-DRY). Main's committed list is the authority to retire debt; observations of current findings cannot authorize new debt (ARCH-ORDER). The file has at most103entries, shrinks with each fix, and is removed or left empty once M2 completes (ARCH-FUNERAL). No inline test-fixed-sleep exemptions are introduced while paying this debt.

M1 landing leaves the issue's M2 checkbox open, with the reviewed milestone as the boundary; use the SDLC milestone-publication path if supported. Do not close the whole issue as done with active sleep debt. If the tool cannot publish a milestone, bring that exact limitation to the hub before landing, rather than bypassing the close gate or silently treating M2 as finished.

### 2026-10-18 — R2: named verification seams (PQ-1)

Reason: plan-quality asks for function-specific adversarial strategies, rather than a prose matrix. This section replaces R1's regression-matrix sentence as the test design; the debt/ratchet behavior is unchanged.

All new functions are test-only in internal/repolint, split between catalog_test.go (repository boundary), debt_test.go (pure parsing/matching and its tests) and inventory_test.go (real-Git fixtures). Pure entities are SleepSite and Debt (new, owned by the test); the repository test owns their lifetime.

| Function | Responsibility | Adversarial strategy / mechanical guard |
| --- | --- | --- |
| sleepSites(filename, source) | Derive line-independent call identities and associate findings with exact source lines | Generated AST layout/whitespace changes must preserve identity; enclosing-function/call/occurrence mutations must change the corresponding identity; parse failures propagate |
| decodeDebt(bytes) | Decode the sole versioned JSON site set | Fuzz malformed JSON, fields and site values; reject unsupported schema, unknown fields, unsafe paths, invalid ordinals and duplicates; valid round trips preserve the set |
| checkDebt(findings, sites, debt) | Apply exact sleep-only debt and reject unused entries | Generated finding/site sets compared to independent set-difference oracle; every non-sleep finding survives unchanged |
| checkShrink(previous, current) | Require current debt to be a subset of previous | Generated set transitions against an independent subset invariant; additions and same-size substitutions never pass |
| scanTracked(root, readFile) | NUL Git inventory and one read/check per eligible file | Disposable real Git trees with generated tracked/untracked/excluded/deleted sources; injected read errors propagate; counted N/2N reads prove one pass |
| debtBaseline(root) | Resolve main merge-base debt or original source bootstrap authority | Disposable real Git histories exercise branch changes, missing refs/blobs and malformed baselines; independently known commit ancestry defines the admissible set |
| TestTrackedGoFilesPassCatalog | Fail the ordinary repository test on any unadmitted finding | Mutation control adds a tracked violating file and independently corrupts debt; clean restoration passes without changing scanner flags |

No pure function requires an IO mock. The actual Git inventory/baseline boundary is qualified with disposable repositories; fuzzing is for the JSON trust boundary, not external tools. PQ-1 is addressed for all newly introduced validation seams, not only the JSON reader.

### 2026-10-18 — R3: implementation decomposition and fixture seams

Reason: the approved non-sleep sweep exposed shared build ownership and several timing oracles; this records the chosen implementation details. Core concepts below supersede the original no-new-entity table, whose checker reuse remains true but did not yet include the staged debt ratchet.

| Pure entity | Lives in | Status |
| --- | --- | --- |
| SleepSite / Debt and exact matching/subset validation | internal/repolint/catalog_test.go | new |

| Integration point | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| Suite-owned distinct binary cache | internal/testbinary/cache.go | new | one synchronized builder callback, owned temp directory and explicit post-m.Run cleanup |
| Controllable recorder drain timer | cmd/internal/terminalcapture/recorder.go | modified | existing owned real timer by default; tests fire only after Draining |
| Controllable last-page grace | cmd/internal/webcast/chrome.go | modified | existing time.After default; protocol fake fires after observed destroy/replacement events |
| Controllable lifecycle completion deadline | cmd/internal/couchcore/park.go | modified | existing context.WithTimeout default; tests observe actual selected budget |
| Store read-lock contention cue | cmd/internal/couchcore/retention.go | modified | existing nonblocking flock; private context observer follows routed store construction |

ARCH-DRY: one testbinary.Cache shared by command/probe fixture owners, rather than three copied cache implementations; its artifact classification explicitly says test support. Cache.Directory is tested with concurrent counted builders, failure/close sequences and subtest lifetime; real executable count controls qualify the Go boundary. The remaining timing seams keep production deadlines and externally visible policy unchanged. They allow timeout/cancellation after a precise observed event instead of racing startup. Telemetry and FIFO repairs incidentally retire three original sleeps during M1, so the current list is100rather than103; this is an allowed shrink, not newly admitted debt.

Estimate INFO noted grouping/qualification uncertainty: M1's six groups are paths, executable ownership, registration/birth/rollback, lifecycle completion, lock/FIFO cancellation and non-core timer/counting oracles. The scanner module row includes debt bootstrap and dependency compatibility; M2's twenty families are a planning grouping over the affected fixture packages, not a measured per-site repair guarantee. The scalar estimate remains provisional calibration; actuals will be measured by SDLC.

### 2026-10-18 — R4: M1 review findings, directive grammar and Git boundary

Reason: round1 reproduces whitespace variants of an upstream-valid inline sleep suppression bypassing the debt authority (BR-1). The guard must recognize the exact accepted upstream prefix, trim its remainder with the same whitespace semantics and extract the same rule token; it may not equate one spelling with the grammar. Test the entire accepted whitespace class against upstream Check, then require the scanner to refuse each accepted sleep directive. Include same-line/preceding-line syntax and textual/non-directive controls; fuzz the rule-boundary normalization where useful. This is the first repair of this mechanism, not another exception or allowlist.

BR-2 adds the README developer entry beside the existing testing instructions: ordinary/focused test, tracked-file scope, local non-sleep reasons, exclusive shrink-only debt file, full-history/main-ref prerequisite and atlas link.

BR-3 is disputed per authenticated hub delivery76f460c7: do not add a fake Git or second index/ref/history/blob model. Here Git is the local deterministic source whose tracked inventory is the contract itself. Existing disposable real repositories already exercise inventory, untracked/deleted files, malformed sources, read failures, branch/history/debt transitions, missing/ambiguous authority and missing/malformed/deleted blobs. These are normal-suite integration/conformance checks on every run, not a service substitute. Keep the pure decoder/identity/subset tests separate. Reconsider only for a concrete failure those real fixtures cannot reproduce. This explicit hub instruction governs the ARCH-MOCK interpretation for this repository test; no review/gate is bypassed and the next review must answer the dispute evidence.

The queued165source suite was canceled through sdlc local-check cancel after the code defect was known, before it acquired a machine slot; cancellation is not a flake retry. Requeue the completed fix head, with recorded PASS still mandatory before M1 landing.

### 2026-10-18 — R5: approved stateful repository-read seam (BR-3)

Reason: round2 addressed BR-1/BR-2 but re-raised BR-3. Authenticated hub42f6145f reverses76f460c7 after checking that ARCH-MOCK explicitly names Git. Address BR-3, do not dispute it again or request withdrawal. This supersedes only R4's Git-boundary exception; its suppression/README fixes stay intact. The running e3ffaf6c suite was canceled through its owner because these changes require a new final-code record.

| Pure entity | Lives in | Status |
| --- | --- | --- |
| Fake repository state (index, refs, commit ancestry and immutable blob trees) | internal/repolint/repository_fake_test.go | new |

| Integration point | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| repositoryRead interface / gitRepository adapter | internal/repolint/repository_test.go | new | only the scanner's consumed read operations, including batched historical source reads |

The interface exposes typed repository facts, not arbitrary Git command execution: tracked paths; configured/detached main authority; common base; paths/blobs at a revision; whether a path existed in history; ancestry; batched source blobs for bootstrap. All Git subprocesses used by catalog scanning and debt-baseline/bootstrap lookup move behind this one adapter. Existing root-aware source reads remain the filesystem boundary and retain their error tests. Keep adapter compatibility wrappers only when needed by existing tests; no second operational path may execute raw Git around the interface.

The fake owns a bounded working index, named refs, current HEAD, commit parent links and copied per-commit blob maps. Explicit test transitions stage/remove/commit/checkout/move refs; reads derive from that state, rather than replaying a queue of canned command responses. Supported history shapes are the fixture histories needed by this scanner, including branch divergence and merge-base/ancestry refusal; this is not a generic fake Git CLI. Unknown refs/commits/blobs return errors, absence is reported only by a successful query proving it, and malformed persisted debt still reaches the same production decoder. State is private to a test and collected with its owned temporary directory; no daemon, cache or production package is added.

Named verification strategies:

- scanTrackedWithRepository and debtBaselineWithRepository: run inventory/debt decisions against injected fake state transitions; compare resulting findings, counts and refused authority against independent expected states, retaining malformed-input and counted N/2N guards.
- fakeRepository transition/read methods: assert index vs untracked/deleted worktree distinction, immutable committed snapshots and parent/ref movement; generated histories exercise the ancestor/common-base invariant, with explicit unknown-state refusal.
- gitRepository read methods: run the same repository-read contract against disposable real Git histories on every ordinary package run; normalize commit aliases, not source bytes or decisions. Retain all existing real-Git failure/history tests as conformance coverage. Real execution never needs network or a remote server.
- anchorDebtWithRepository: both backends feed the same canonical source-site derivation and fixed original103admission invariant; batched adapter IO preserves bounded spawn cost.

Test-first: add the shared injected contract and fake transitions, observe RED, then move existing logic behind the adapter and implement the bounded fake. Re-run scanner package normal/race, suppression fuzz, added-site/corrupt-debt mutations and the real-Git conformance contracts. Document the fake/real contract and every-run cadence in atlas. No fake Go build, new CLI runner, production dependency or protected config change. On the final committed/pushed code head, queue the recorded suite and overlap the next M1 review under the standing hub rule; PASS still gates landing.

R5 interface refinement (same approved shape): repositoryRead has TrackedPaths, MainTracking, RemoteMainRefs, MergeBase, PathsAt, ReadBlobs, PathExisted and IsAncestor. MainTracking returns typed configuration facts, so configured/detached main interpretation remains one common policy rather than duplicated in the fake. Fake and real contract fixtures expose transition-only Write/Stage/Remove/Commit/Checkout/MoveRef/ConfigureMain drivers. Their reads feed scanTrackedWithRepository, debtBaselineWithRepository and anchorDebtWithRepository. Historical batch reads map NUL-delimited tree paths to object IDs, then batch only IDs, preserving newline filenames without requiring a new Git minimum or per-file subprocesses.

The fixture driver also accepts explicit commit parents to qualify disconnected and crisscross histories against real Git; this adds no scanner-facing operation. Both backends refuse an absent or ambiguous merge base instead of choosing an arbitrary debt authority.

### 2026-10-18 — R6: M2 duration subjects remain explicit

Authenticated hub bf84e7de accepts retaining a sleep whose duration is the subject (the real-PTY holds past the old two-second cutoff, viewer cadence and stress/sample windows) as an individually explained entry in the exact shrink-only file. Readiness guesses become explicit signals. A reason belongs on each retained entry so the list explains itself; changing a reason cannot change its site identity or enlarge the admitted set. Historical debt remains readable. Use controlled time only where the tested code already has a clock seam, retaining separate real-timing qualification when switching to fake time. Do not add a clock merely to hide a sleep. This refines M2's completed-sweep criterion to resolved readiness debt plus reasoned duration subjects, without admitting any new site or inline sleep suppression. M2 implementation starts only on a fresh branch after M1 lands.

### 2026-10-18 — R7 proposal: concrete M2 boundaries and unresolved witnesses

Read-only package plans are preserved in [M2 implementation notes](000665-m2-implementation-notes.md). Exact current100-site accounting: priority23 (17 proposed repairs, four duration subjects, two unresolved external-agent readiness sites); utility17 (13 repairs, four duration subjects); webcast57 (51 repairs, four duration subjects, two unresolved browser-completion oracles); layout3 (three predicate-based native fixture repairs). This is84 proposed removals,12 individually reasoned duration subjects and four unresolved readiness/oracle sites, not a completed sweep.

Keep SleepSite as AST identity. Add a flat serialized debt entry that embeds SleepSite plus reason; retain version1 historical decoding with optional old reasons, and require nonblank reasons on each current retained entry. decodeDebt rejects duplicate identities even if reasons differ; checkDebt and checkShrink compare SleepSite only. The current-reason validator rejects absent/blank reasons. Adversarial JSON, shared fake/real old-baseline admission, changed-reason/new-site, duplicate-identity/different-reason and blank-reason mutations defend this metadata addition without relaxing the ratchet.

New observations are confined to their actual boundaries: messageService reports a processed Registry event after its effects/follow-ups; Console reports a decoded input event after routing; browser-policy transitions report command completion/error for that transition/session; existing viewer batch and annotation POST work expose pending completion without forcing a flush or extra input. Each named seam in the notes has a held-completion RED control, generation/order checks and joined teardown. No clock is added solely to remove a sleep, no CDP reply is claimed to fence future browser work, and no independent terminal-command queue is used as a stdin barrier. Existing product deadlines and assertions remain.

Layout fixture actions must await their deciding facts: swap-layout identity/geometry after next-swap-layout, target focus after focus-pane-id, a second terminal after new-pane, target floating state on entry, and the original geometry plus empty return record on return. Replace the initial300ms client guess with positive attached-client evidence (existing strict ClientCountContext) after registering input; malformed/unknown replies refuse rather than satisfying the predicate. Keep all rung/split/order checks and existing time bounds. Qualify the helper with stateful held-transition controls and request the hub's private native lane before running it.

Four holds require a hub decision before treating M2 as complete: wrapcmd model/picker1500ms and trust-input1s show draw-time menus but have no mounted-handler witness; webcast denied download has no qualified terminal deny event, and restored native chooser has no proved terminal completion. Options are bounded disposable investigation for an authenticated event witness, or explicit followup deferral of those exact sites. Neither is relabeled a duration subject or silently removed. All M2 code still waits for M1 landing, complete-plan approval and change-code.

R7 approval: authenticated hub42233925 approves the84 proposed removals with their bounded observations and12 reasoned duration subjects. The four unresolved sites are explicitly deferred, kept as exact debt with the reason "no completion signal exists yet" plus the missing witness. File one unassigned followup listing all four, after M1 lands; do not investigate them in M2. The M2 terminal target is16 individually explained retained sites, not zero sleeps. They remain under the same shrink-only check and none is claimed fixed. Build M2 from fresh main only after M1's recorded suite PASS and landing.
