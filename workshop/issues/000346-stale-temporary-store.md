---
id: 000346
status: working
deps: []
github_issue:
created: 2026-09-29
updated: 2026-09-29
estimate_hours: 4.409
card_mirror: '025fd590586521464a04d863675f5cdbfb5227da' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-29T08:41:28-07:00
flow: {kind: full, provenance: inferred}
---

# Stale temporary store blocks Couch startup

## Problem

After a macOS upgrade/reboot, both interactive `couch` and `couch --list` fail with `registered store ".../scratchpad/store.tYjN" unavailable: lstat /private/tmp/claude-501: no such file or directory`.

The persistent Pair retention registry contained the normal `~/.local/share/pair/couch` store plus a temporary Claude scratchpad store from a Pair slot1 development session. The temporary directory no longer exists; the normal store remains present. The path suggests a development/smoke-test store leaked into the production inventory; the exact originating invocation is not yet verified. `/tmp` resolves to `/private/tmp` on this machine, and registration canonicalizes symlinks.

`storagegc.StoreRegistry.validate` checks every registered path. `RegisterStore` validates the existing inventory before registering even the normal store; `NewCoordinatedThreadStore` invokes registration during construction. One vanished auxiliary store therefore prevents startup and read-only listing. `UnregisterStore` also reads the fully validated registry and requires an existing canonical directory, leaving no normal recovery route for a missing store.

## Spec

Prevent development/test stores from contaminating the operator's durable retention inventory. Trace the source of the scratchpad registration and ensure smoke tests isolate both Couch namespaces and Pair retention roots.

Define supported recovery for vanished registered stores and allow unaffected stores to remain usable where safe. Preserve retention safety: an unavailable store is not proof that it had no references; do not silently enable collection or discard real thread data. Distinguish missing temporary stores from unreadable or temporarily unmounted durable stores and make the recovery action explicit.

Relevant code: `cmd/internal/storagegc/stores.go`, `cmd/internal/couchcore/retention.go`, and `cmd/internal/couchcmd/run.go`. ARCH-SECURE: tests must not write operator state. ARCH-FUNERAL: registration must define the lifecycle and removal of ephemeral stores.

### Binding and launch observation (agreed 2026-09-29)

Binding means the association `(Pair tag, native root transcript UUID)` within the existing repository/agent scope. Native UUIDs identify conversations; device, inode, generation, file size and parser-cache continuity are not conversation identity. UUID-named transcripts are normally append-only. Do not require a full body reread to retain a recorded resume target after filesystem metadata changes. An internal field that appears to name a different UUID may reflect changed producer semantics; it is diagnostic evidence, not by itself grounds to revoke the filename/recorded association.

Fresh launches and resumed launches share one current-launch observation/correlation process. Fresh launch discovers a root transcript. Resume starts immediately with the recorded UUID A as a provisional target (probation), then observes which root actually receives the current interaction. Candidate selection must include post-launch activity in pre-existing transcripts as well as newly created files; the current fresh-start exclusion of all baseline paths cannot be reused unchanged. Historical matching messages cannot confirm a new launch.

Use the existing causal message/progress matching thresholds initially, with current-launch boundaries. If sufficiently distinctive evidence confirms A, confirm this launch; if it establishes D, trust the observation and update the binding to D, retaining A, launch identity and the reason/evidence for the change for debugging. A-to-D fallback/fork is a scenario to test, not an observed incident finding. Silence or ambiguity means not yet confirmed, not failure: normal interaction and early Alt+n remain available, retrying the best available target (usually A). A historical target must belong to the same intended conversation/agent; do not fall back to unrelated old launches such as pair:2's prior Claude sessions.

Treat agent-owned extensible field values as open world. Unknown `source` values must not automatically reject an otherwise evidenced root; adding only `vscode` to a fixed allowlist is insufficient. Root/subagent distinction still matters, using understood positive evidence rather than assuming every unknown record is a root. Retain safe bounded file/JSON reading without making complete schema interpretation a startup gate. Where Pair supplies `--session-id X`, creation of the corresponding new root transcript named X can provide the handshake without requiring prompt matching. Distinguish malformed/unreadable input from unfamiliar valid metadata.

Separate durable resume identity, launch-specific observation confidence, and optional parsed-content/cache availability (ARCH-PURPOSE). Cache invalidation or a native parser failure must not erase the target or block Couch/Alt+n. Publication must check the current launch under the existing ledger synchronization; stale observers cannot change a newer launch. Retain prior config/launch arguments; recovery with no argv must not overwrite them.

### TTY history follow-up

Automatic startup archival, quit capture retention and printable-TTY naming moved to #349 with operator approval. The feature inventory and test requirements are preserved there. #347 separately covers permanent capture identities. Optional native telemetry remains independent of durable resume identity in #346.

The original stale-store/GC requirements above remain in scope. The durable plan at `workshop/plans/000346-stale-temporary-store-plan.md` has been reconciled against this contract; its revision history preserves the superseded file-proof design.

## Done when

- A regression reproduces registration of an auxiliary temporary store, deletion of its directory, and subsequent launch/list behavior for the intact normal store.
- A supported, actionable recovery path handles stale missing registrations without fabricating empty-store proof or weakening garbage-collection safeguards.
- Development/smoke-test invocations isolate all persistent roots; regression coverage proves the operator registry remains untouched.
- Missing, unreadable, and temporarily unavailable durable stores have tested behavior that preserves references and communicates recovery requirements.

- Persisted UUID targets remain resumable after device/inode/cache changes, unfamiliar native metadata, or unavailable optional parsing; test the production Couch and Alt+n boundaries with and without park receipts.
- Fresh root binding accepts `vscode` and future unfamiliar valid source values without abandoning root/subagent discrimination; cover Pair-supplied UUID/new-file handshake.
- Every resume is observed for its own launch: post-launch activity in existing A confirms A; evidence for D replaces A and retains diagnostic history. Old messages, silence, ambiguous candidates and stale observers cannot falsely confirm or replace it.
- Normal interaction and early Alt+n work during probation; silence preserves the best available target. Existing launch options/config are preserved during recovery.
- Every audited row has a verified outcome: available resume target with its observation state, already-live thread, empty slot, or explicit insufficient evidence. Do not fabricate pair:4's missing transcript or restart brain:0 as an experiment.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only; calibration is currently tagged stale, so values are provisional. Familiar Go/filesystem stack: familiarity 1.0. Thorough agreed spec discounts design to roughly 20% of the primitive ranges; the new capture protocol retains more design uncertainty. Existing retention/locking, ledger, correlation and VT libraries are reused; no novel external stack is assumed. Implementation values are 40% of v2.1 ranges, with a 15% design buffer.

Items in order: registry availability, GC CLI, smoke isolation; ledger v3, resume projection, observation integration, open-world scanner, repair CLI, identity-consumer sweep; capture transaction, writer/quit integration, TTY naming; docs; three real milestone reviews. These are ship-time estimates, not manual actuals.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.04 impl=0.2
item: smaller-go-module design=0.03 impl=0.12
item: smaller-go-module design=0.03 impl=0.12
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: greenfield-go-module design=0.2 impl=0.32
item: smaller-go-module design=0.03 impl=0.12
item: smaller-go-module design=0.05 impl=0.2
item: cross-cutting-refactor design=0.1 impl=0.2
item: greenfield-go-module design=0.3 impl=0.32
item: cross-cutting-refactor design=0.1 impl=0.2
item: smaller-go-module design=0.04 impl=0.16
item: atlas-docs design=0.03 impl=0.08
item: milestone-review design=0.04 impl=0.2
item: milestone-review design=0.04 impl=0.2
item: milestone-review design=0.04 impl=0.2
design-buffer: 0.15
total: 4.409
```

## Plan

Detailed implementation plan: `workshop/plans/000346-stale-temporary-store-plan.md`, reconciled with the agreed model after the operator requested starting work in pair:0. SDLC plan-quality/estimate gates precede code changes.

- [x] M1 — Keep intact stores usable, retain GC safeguards, provide explicit missing-store recovery and isolated smoke coverage.
- [x] M2 — Separate durable targets from parsed evidence; share launch-specific fresh/resume observation, open-world roots and supported incident recovery.
- Deferred to #349: former M3 TTY preservation and naming. Incident inventory and binding acceptance remain part of M2; no M3 implementation or review is claimed.

## Log

- 2026-09-29: The final termcmd rerun uses the isolated smoke environment and passes (0.384s, `/tmp/pair346-termcmd-final.log`). Interrupted an early issue-close dispatch while waiting for this verification; it produced no verdict or close. Restart close only with the completed test evidence.

- 2026-09-29: Final unsandboxed `go test ./...` completed (`/tmp/pair346-final-unsandboxed.log`): Couch core (275.932s), launcher (17.369s), wrapper (40.160s) and remaining runtime packages pass; no reviewer PTY sandbox errors. It found the known #348 artifact inventory failure and a newly misplaced Go comment above `refuseResolvedBinding`. Moved the comment to its correct declaration without behavior changes; full `go test ./cmd/internal/termcmd -count=1` passes afterward. Only the independently reproduced baseline failure remains outstanding outside #346.

- 2026-09-29: M2 re-review returned SHIP and disposed BR-14. Accepted minor state-modeling/actionability follow-ups are preserved in #350; TTY remains #349. Latest full unsandboxed Go run requested to distinguish reviewer PTY sandbox failures from product behavior before issue close.

- 2026-09-29: BR-14 fixed across consumers: positive filename admits even if unrelated listing diagnostics exist; absence authorizes fresh only after complete enumeration/explicit absent root. Unknown projection blocks standalone restart before markers/kill and picker before config removal; picker reuses the typed decision rather than probing twice. Non-ENOENT ledger failures and conflicting typed identities also refuse fresh fallback. Couch refuses before park and diagnoses incomplete storage observation. Full affected launcher/inventory/dispatcher/watcher/wrapper suites pass; final launcher suite and focused launcher/inventory/Couch race regressions pass, including errors, partial listings, conflicting identities, config preservation, and no-child/no-park effects. Evidence: `/tmp/pair346-br14-packages.log`, `/tmp/pair346-br14-launcher-final.log`, `/tmp/pair346-br14-race.log`, `/tmp/pair346-br14-couch-final.log`. No native bodies are read for chosen materialization.

- 2026-09-29: Final post-policy full verification completed: isolated `make -k test` ran all shell/Lua prerequisites and all Go packages (`/tmp/pair346-final-make.log`). The only failing test is `artifactpath.TestProductionArtifactReferencesAreExactlyClassified`, the independently reproduced baseline defect tracked in #348. Couch core passes (270.491s), launcher (16.130s), inventory (8.259s), watcher (2.653s), wrapper (39.889s), and all remaining packages pass. Focused new policy race tests pass. The full suite is explicitly not claimed green; request boundary judgment with baseline evidence.

- 2026-09-29: Implemented operator BR-5 decision: unconfirmed Pair-chosen UUID requires only a root filename before reuse; absent filename produces explicit fresh-required admission. Standalone and Couch Alt+n strip stale identity flags and launch with a new generated UUID. Confirmed and requested-resume UUIDs remain durable without native files. Couch rechecks fresh intent before child launch and refuses if the requested ID changes or its root materializes meanwhile. Claude/Qoder metadata-only tests, actual OS-ledger→restart/createflow tests, full launcher/inventory/watcher/dispatcher/wrapper packages, and focused Couch race tests pass. Full isolated make suite running after final production edits. M3 transferred to #349 by operator approval; #347 remains permanent capture identities.

- 2026-09-29: Final verification checkpoint: isolated `make -k test` completed (`/tmp/pair346-close-make.log`); all shell/Lua smoke prerequisites passed, including real review-branch restore. Go failures were the 32 baseline artifact inventory diagnostics tracked in #348 plus wrapper test-environment assumptions. Fixed wrapper tests without production changes; subsequent full isolated `go test ./cmd/internal/wrapcmd -count=1` passes (35.782s, `/tmp/pair346-close-wrap.log`), and focused tests also pass under actual HOME. Full Couch core passed (311.496s), as did binding/ledger/watcher/launcher packages. Tree committed. Closure remains pending operator disposition of unfinished M3 TTY scope and BR-5 fresh chosen-ID policy; neither waived nor silently changed.

- 2026-09-29: Final read-only incident preview: audited brain launch, prior parley.nvim launch, and pair:2 each return `repairable` with unique current-launch prompt/progress correlation; pair:4 returns `unresolved` / no unique correlation. All `applied:false`; no agent restarted. Current Couch list admits recorded targets for formerly device-stale rows, and current pair:0 is live with a confirmed new root. Preexisting full-suite artifact failure tracked as #348; all 32 residual diagnostics exactly match generated merge-base baseline.

- 2026-09-29: Close verification: isolated review-branch smoke failure was an overlong macOS Unix socket endpoint (117 bytes), not binding behavior; short owned socket plus actual editor readiness passes. Exhaustive artifact inventory now includes #346 handshake/recovery sources and both watcher log sites. Residual review-source inventory failures reproduce at merge-base `b622052f` (`/tmp/pair346-artifact-inventory-baseline.log`); do not weaken path-construction checks or claim a green full suite.

- 2026-09-29: Operator smoke passed: ariadne:0 resumed from parked; actual native conversation identity was not independently confirmed. After creating a fresh pair:0 on the updated binary at 12:11:55, watcher persisted a v3 correlation binding for root `01a0ee94-abfb-7752-9abc-73e31e8ac5e5` after the hello exchange. Previous 10:03 watcher was still old code; repair preview uniquely matched that older launch without applying changes. Final argv safety and README contract regressions pass. Full isolated make test remains blocked by review draft initialization (`PairReviewToggle` missing); diagnosis underway. TTY M3 remains unimplemented pending explicit scope disposition before closure.

- 2026-09-29: M2 review corrections verified. Shared repair intersection regression, v3 watcher restart/lifecycle regression, one-metadata-observation count, typed confirmation reason and argv-safe opaque ID tests pass. Full ledger/watcher race and focused inventory tests pass. Full launcher/dispatcher race pass; real-ledger Alt+n target preservation and picker regressions pass after final cleanup. README registry coverage test was RED for session-repair, now GREEN. BR-5 code is unchanged under the existing approved probation contract; request explicit reviewer disposition rather than silently adopting its proposed new file-presence gate.

- 2026-09-29: M2 review round 3 returned FIX-THEN-SHIP. BR-6 repair/live matching divergence and BR-7 README coverage are being fixed, plus v3 proofless migration, typed confirmation reasons, safe argv IDs and dead quarantine. Follow-up found and reproduced standalone restart's legacy ledger projection dropping requested A; it now uses the same pure target projection as the query. BR-5 is disputed against the approved admission contract, pending operator preference: `QueryResumeTarget` is ledger-only and returns the current requested UUID under probation, and early Alt+n retries the best available target. A chosen-ID request is deliberately provisional, not a confirmed binding; the filename handshake is what confirms it. The review proposes a new pre-admission filename gate and fresh fallback, but explicitly did not verify when Claude materializes its transcript. Asked the operator about this narrower unmaterialized-chosen-ID edge case; do not silently override the accepted probation policy on speculative native behavior.

- 2026-09-29: M2 implementation committed as `97f45851` and `eaa853e3`. Full ledger/inventory/watcher/dispatcher/pair-go tests passed; full ledger/inventory/watcher race tests passed, with final source-shape and optional-parsing additions rechecked under race. Full Couch passed (233.7s); launcher/opener/review full race passed; targeted Couch/wrapper race passed. Context/title/threadactivity/slug tests passed; all command packages compile. Installed read-only `make test-session-inventory-conformance` passed across Claude, Codex, Muse, Qoder and Agy. OS-upgraded sqlite3 resets headers with `-header -csv`; reordered to `-csv -header`, verified by existing integration test. Large slug fixture now uses unknown-event padding rather than malformed session metadata so schema-rebuild coverage reads valid data. `git diff --check` passed. M2 review pending.

- 2026-09-29: M2 consumer integration in progress. RED tests reproduced rejection of a nonempty provisional UUID in full launcher and Couch admission; both now pass. Couch/launcher/review/changelog use the ledger-only target query. Saved-parameter picker resumes a durable target even with no native file; fresh Claude/Qoder launches and wrapper restarts record chosen-ID origin. Production resolver regression verifies A under probation followed by confirmed D without any transcript read. Full launcher/opener/review suites with race detection pass; focused Couch/restart/binding suite passes. Native watcher epoch/confirmation work and full M2 verification remain in progress; no milestone completion claimed.

### 2026-09-29
- 2026-09-29: closed M2 — Operator parked-resume and fresh-binding smoke pass. Final full isolated make test ran: only baseline artifact inventory defect #348 fails (matching generated b622052f). BR14 then fixed with failed/partial metadata, positive presence, config preservation, no-kill/no-park, ledger errors and conflicting identity regressions; full affected packages and final launcher suite plus focused launcher/inventory/Couch race pass (/tmp/pair346-br14-*.log). make build passes. User approved fresh UUID for absent unconfirmed chosen ID; unknown storage refuses without destruction. TTY scope transferred to #349.; review verdict: SHIP
- 2026-09-29: closed M1 — M1 full suites and race suites previously passed. BR-1–3 corrections: focused production CLI/subprocess checks passed; actual ambient registry write detected by isolation mutation; full storagegc/gccmd race rerun and focused couchcmd race passed; shell syntax/diff checks passed. Actual 3.46h comes from preceding sdlc measurement, first milestone includes design.; review verdict: SHIP

Reproduced locally: `couch --list` exited 1 with the same missing-store error as the screenshot. With the retention coordinator's exclusive flock held, backed up `stores.json`, removed only the exact missing scratchpad registration, atomically replaced the registry, and retained the original migration flag and normal store. Backup: `~/.local/share/pair/.retention/stores.json.backup-20260929-072547`. No thread payloads were removed. Afterwards `couch --list` exited 0 and enumerated existing threads. Many report lost session bindings following the reboot; those are separate from the registry startup blocker. This ticket tracks the permanent fix; no implementation changes made.

### 2026-09-29 — Full thread audit after OS upgrade and restart

Operator confirmed this was a macOS upgrade followed by a restart, not merely a reboot. Audited all 23 rows in the screenshot against current `couch --list`, the global threadstore manifest/records, numbered-slot `couch --show` addresses, and the corresponding scoped ledgers. This is an investigation, not a repair: no thread, ledger, proof, transcript, or retention-registry data was modified.

The earlier registry recovery above only made Couch inventory/startup usable. It did not recover cold-resume authority for every thread. There are **15 saved-proof identity mismatches, four current launches without bindings, one live thread also lacking a binding, and three never-started slots**.

#### A. Fifteen saved proofs rejected after device renumbering

Every proof-named transcript below exists, has the same inode as recorded, is at least as large as the recorded proof size, and has a first JSON record naming the saved native session ID. Every recorded device is `16777230`; every current device is `16777232`. These checks establish a common rejection mechanism, not complete transcript validation or authorization to rewrite the proofs. The evidence does not distinguish whether upgrade or reboot caused the renumbering.

| Thread | Agent | UI status | Native session ID | Unchanged inode |
| --- | --- | --- | --- | --- |
| 42shots:0 | qoder | binding lost | `3a5781c6-d861-4926-a067-65e443723b98` | `185512106` |
| ariadne:0 | claude | binding lost | `bee2be25-757b-4027-8573-2bdcaf857203` | `246185815` |
| ariadne:1 | claude | binding lost | `1f48235f-b720-4bcd-b5fe-e162d58f0bec` | `238895656` |
| ariadne:2 | claude | binding lost | `cf3391ab-0ac5-4464-bbb2-be53048558a6` | `240413721` |
| ariadne:3 | claude | binding lost | `b966b671-7eec-484a-93fc-930a54bdbc62` | `208706390` |
| astro:0 | claude | session gone | `f8209a7c-1cce-4325-ac6e-95b88a503560` | `160040825` |
| tools:0 (define) | claude | binding lost | `20d98988-f072-4cba-a88f-51eceeaef985` | `195338098` |
| ducks:0 | claude | binding lost | `cd295436-7ec1-426f-a851-2cdc3ca83b5f` | `233841032` |
| kbench:0 | claude | session gone | `ae4c7a3b-20f5-4dc3-9588-4fbee1370563` | `162494622` |
| nous:0 | claude | binding lost | `286b0136-54a0-40f7-8bd2-60ad0a9f08ca` | `233294735` |
| pair:1 | claude | binding lost | `4acd368e-45c2-452d-a198-49585642c8b0` | `236192803` |
| pair:3 | claude | binding lost | `04f01644-a62a-4c14-b6a8-5fccec140aa8` | `237533351` |
| parley.nvim:1 | claude | binding lost | `614d7266-98f2-4cb4-901b-cb2b908f26f6` | `230764949` |
| parli:0 | claude | binding lost | `61daadf1-9e39-44f3-a55b-fe09544fc7b2` | `166208996` |
| xianxu.dev:0 | claude | binding lost | `75bd6632-bb7b-4095-ab91-9760a61a7dd6` | `153707675` |

Source chain: `sessioninventory/filemeta_darwin.go:platformFileMetadata` builds `StableFileID` from `stat.Dev` and `stat.Ino`; incremental target validation rejects a changed stable ID; `query.go:proofAllowsFullRevalidation` also refuses the mismatch. `QuerySessionContext` leaves failed proofs provisional. `couchcore/actionableinventory.go:ClassifyThread` reports `binding-lost` when a verified park receipt exists, otherwise `session-gone`. Thus **astro and kbench do not demonstrate missing transcript data**: they hit the same proof mismatch but have no verified park receipt. Their UI label must not be interpreted as evidence that their conversation is irrecoverable.

#### B. Missing current-launch bindings

| Thread | Agent | Evidence | Interpretation |
| --- | --- | --- | --- |
| pair:0 | codex | Two typed launches; latest physical ledger line 3; zero binding rows; legacy session ID empty | Cold resolver returns provisional before checking file identity. Device renumbering does not explain the absent binding. |
| parley.nvim:0 | codex | One typed launch at line 2; zero binding rows; legacy session ID empty | Same missing-binding shape as pair:0. |
| pair:2 | codex | Five typed launches across agents; current Codex launch at line 13 has no binding; four historical bindings belong to Claude generations; latest legacy ID empty | Old bindings cannot authorize the current launch. Do not restore an older conversation just because its ID is available. |
| pair:4 | claude | One typed launch at line 2; zero binding rows; legacy session ID `d75674b7-7278-40af-b4d3-62203aaf190f` | No transcript filename matching that legacy ID was found under `~/.claude/projects`; recovery remains unproven. |

All four display `binding lost` and have park receipts. `sessioninventory/query.go:QuerySessionContext` returns provisional for a current launch without a binding before consulting transcript proofs. The two current-launch bindings on ariadne:1 name the same root and are not a conflict. Why the producer failed to commit each missing binding remains a separate investigation; the reboot cannot be assigned as the cause from these ledgers alone. Pair:4 also has an empty Pair send log and only 84 retained wrapper events, so no sent/completed round is evidenced there. Binding publication calls `ObserveAndPersist` after completed-round correlation (`sessionwatch/lifecycle.go`); the retained Codex wrapper events supply no watcher/binding diagnostic that establishes why publication did not occur.

#### C. Live and never-started rows

- **brain:0:** live, recorded PID 41885 at audit time; current Codex thread `2e51fcf9799b1d8f/couch-6b111ea230c149dc`. One typed launch at line 2, no binding rows, empty legacy session ID. Live status proves current execution, not cold resumability. This is an additional instance to cover when investigating Codex binding publication; do not restart it as a diagnostic experiment.
- **brain:1, parli:1, tools:1:** `never started`; `couch --show <slot>` reports no readable current thread. These are empty-slot cases, not evidence of transcript loss. The CLI listing currently mislabels all three as `tools:1`; their distinct working paths and the screenshot identify the intended slots. This rendering discrepancy is separate from proof validation.

#### Recovery/design implications

- Preserve the existing stale-store recovery requirements. Add upgrade/restart recovery coverage for persisted native-session authority; a mount device number cannot serve as an indefinitely stable identity by itself (ARCH-PURPOSE, ARCH-FUNERAL).
- A fix must revalidate the exact saved conversation safely when filesystem identity changes, including rejection of replaced files or conflicting conversation identities. Blindly replacing `dev:` values or ignoring identity checks is not an established recovery method.
- Cover the two diagnostic branches (`binding lost` with a receipt and `session gone` without one) with the same surviving-transcript/device-renumber regression. Test the production cold-resume authorization boundary, not just metadata formatting.
- Diagnose binding publication independently for the four unbound parked launches and the live brain launch. Preserve current-launch ownership; historical or legacy IDs are not sufficient proof.
- No code fix or operational recovery has been attempted in this audit. Native transcript contents were checked only for identity evidence, not replayed or modified.

### 2026-09-29 — Agreed probation and TTY-first model

Read-only consumer audit established that changelog and orientation already use TTY as their primary text input; naming currently parses native messages, prompt history has its own exact-input log, usage and Codex lifecycle consume native telemetry, and idle activity only needs mtime. The wrapper opens raw/events with O_TRUNC; default rendering is 2,000 rows, not a capture cap. Operator agreed on automatic preservation at quit and before same-tag startup, launch-specific observation for both fresh/resume, and nonblocking probation including early Alt+n. Updated active Spec, Done when and Plan; preserved the original incident audit and earlier decisions below as historical records. No implementation or live recovery performed in this update.

### 2026-09-29 — M1 implementation and verification

Implemented structural registry validation separately from namespace availability. Intact registration and production Couch listing survive unrelated missing/unreadable stores; GC and migration still require full available inventory. Added `pair gc --forget-missing-store` with exact-path checks, coordinator serialization and migration reset. Tests isolate ambient roots and cover unsafe/permission failures without treating them as absence. Original scratchpad invocation remains unidentified; no operator registry mutation was needed in this implementation.

Verification: initial red regression reproduced the missing auxiliary store error; `go test ./cmd/internal/storagegc ./cmd/internal/gccmd ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/couch -count=1` passed. Full relevant `-race` suites passed (Couch core 270 seconds). Production CLI mutation check failed when registration's old full-availability validation was restored and passed after exact-byte restoration. Focused updated collector assertions and permission/refusal cases passed. Code commit `53be9ca8`; SDLC M1 review next. ARCH-PURPOSE/ARCH-SECURE: exercise the real list boundary and retain missing references until explicit abandonment.

### 2026-09-29 — M1 boundary review corrections

Boundary review accepted registry behavior and raised BR-1–3: the isolation sentinel was not at the real registry path, README omitted the new flag, and GC outage diagnostics lacked actionable recovery. Added a shared smoke environment wrapper, wired the recovery smoke through it, and used a real subprocess plus ambient `.retention/stores.json` sentinel. Mutating the wrapper to leak HOME/XDG/Pair/Couch roots now changes that sentinel and fails the test; restored isolation passes. Added read-only `InspectRegistry` for availability-marked listing and restore/remount/permanent-abandonment guidance, plus README documentation. Kept conservative full-inventory `UnregisterStore` behavior and documented why (review minor). Added the resulting prevention rule to lessons.

## Revisions

### 2026-09-29T07:59:14-07:00 — Expand incident evidence to all visible threads

At the operator’s request, appended the OS-upgrade context and a complete 23-row failure-mode audit. The original startup-blocker history and acceptance contract remain intact; this revision records additional recovery requirements and unresolved binding-publication cases without claiming implementation or recovery.

### 2026-09-29 — Begin the fix

Operator requested publishing the audit on main and starting implementation work. Added the initial work checklist before publication; a detailed design will be prepared on the claimed issue branch before code changes. `issue move-detail` confirmed creation was already completed, so the audit is an ordinary update to the existing main details.

### 2026-09-29 — Claimed in pair:0; concrete Codex diagnosis

Audit/checklist published on main as `b622052f`; #346 claimed and `sdlc start-plan` entered branch `000346-stale-temporary-store` in `/Users/xianxu/workspace/pair` as requested. The initial move was already complete; publication used `sdlc push` after associating the audit commit with the issue branch to satisfy the handoff ownership guard. No bypass flags used.

Read-only matching found all four audited Codex transcripts use `source: vscode`: brain `01a0eda0-b38c-7992-988a-74ad95e6925e`, pair:0 `01a0eb3e-8a92-71a2-b35d-65d20ae3e5f4`, pair:2 `01a0e914-ec6a-7e62-bf63-e73bbf5ae9cd`, parley.nvim:0 `01a0eb40-e0b1-7db0-be0d-403f75e873e5`. Each has matching normalized Pair sends and lies outside its launch baseline. This is candidate evidence, not a replacement for the production completed-round/proof checks. `codexRole` rejects vscode before correlation; [official Codex SessionSource](https://github.com/openai/codex/blob/main/codex-rs/protocol/src/protocol.rs) recognizes VSCode as a root-capable source. The earlier audit's unknown-producer cause is now narrowed to this scanner incompatibility.

### 2026-09-29 — Design revision

Expanded Spec/Done when and replaced the initial checklist with three actual review boundaries covering the original registry defect, proof device renumbering, and Codex binding/recovery. Added the durable implementation plan. All implementation remains behind plan approval and `sdlc change-code`; the active brain session stays running.

### 2026-09-29 — Identity model corrected by operator

Native conversation UUIDs make volume information unnecessary for conversation scoping/deduplication. Revised the draft plan: `(agent, native UUID)` is durable authority; file identity is a cache-continuity hint. A metadata change triggers full validation of the exact saved artifact set, not immediate loss of binding or a special device-only exception. The filename alone is insufficient; internal UUID, root role, schema, undisputed state, size bounds and read stability are checked. This replaces the earlier proposed same-inode requirement.

### 2026-09-29 — Plan review correction

Fresh-context review found one Important issue: repair must not reuse the watcher config refresh with empty argv. Changed the plan to preserve existing config bytes (and missing config), repair only ledger/catalog authority, and verify actual cold resume uses the recorded launch profile. Added old-catalog vscode regression coverage. Re-review pending; no code changes.

### 2026-09-29 — Plan review approved

Fresh-context re-review approved `ed84b793` with no remaining Important/Critical findings. Issue schema validation and diff whitespace checks pass. Plan is committed in pair:0 and awaits operator approval before `sdlc change-code`; implementation has not started.

### 2026-09-29T09:59:21-07:00 — Replace rigid file-proof admission with observed binding

Reason: operator clarified that the purpose of association is reliable conversation resumption; validating an old transcript's filesystem/parser identity does not establish what the new agent actually resumed. Delta: supersede full-reread/internal-UUID rejection and closed source allowlisting with durable UUID targets, open-world metadata, shared launch-specific fresh/resume observation, probation that permits interaction/Alt+n, and diagnostic A-to-D binding history. Add automatic raw/sidecar preservation on quit and before same-tag reuse, TTY-first text features, and feature-local native telemetry failures. Original stale-store safety scope remains. The prior plan review applies only to its historical revision; this update records the agreed design, not implementation completion or a new plan-quality verdict.

### 2026-09-29T10:22:03-07:00 — Simplify TTY scope; separate permanent identity task

Operator clarified that startup alone owns archival before path reuse. Quit leaves files intact; compaction retains its existing named-copy behavior. Removed the proposed exactly-once archive journal/two-lock transaction requirement; redundant crash-time archives are acceptable. Created #347 for permanent per-launch capture identities and stable continuation references. Continue #346 in pair:0.

### 2026-09-29 — Operator-approved TTY scope split

Reason: operator requested closing the smoke-tested binding fix and explicitly approved moving unfinished TTY work to a follow-up. Delta: transfer startup archival, quit retention, TTY naming, capture-length checks and related test plan to #349; remove M3 as a #346 review boundary. Keep incident verification and final binding acceptance in M2. Preserve original estimate as historical scope, not a revised measurement. No TTY completion is claimed.

### 2026-09-29 — Explicit fresh chosen-ID restart policy (BR-5)

Operator clarified that a UUID generated by Pair and supplied with `--session-id X` is not yet a resumable conversation if its root transcript has never materialized. On immediate Alt+n in that case, start fresh with a new UUID rather than `--resume X`. A metadata-only matching root filename is sufficient evidence of materialization; do not introduce a transcript-body/parser gate. Confirmed bindings and existing-conversation resume requests retain their durable UUID even if the native file is temporarily unavailable. This supersedes the earlier proposal to retry every requested UUID uniformly. Verify actual restart paths and ensure stale config does not resurrect discarded X.

### 2026-09-29 — BR-14 probe failure is not absence

Boundary review accepted BR-5/BR-11/BR-13 but found the new chosen-ID check treated failed or partial native listing as absence. Clarification: only complete absence (including an explicitly absent storage root) authorizes fresh replacement. Unknown materialization must refuse restart before removing config, stopping the existing session, or creating a new conversation. A positive matching root filename remains sufficient; established and resume-requested identities remain independent of this probe. Add runtime error/partial-list and real restart-admission regressions; keep native bodies unread.
