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
