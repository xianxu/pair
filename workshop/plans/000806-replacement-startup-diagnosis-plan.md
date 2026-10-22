---
collaboration-mode: discussion
---

# Replacement startup diagnosis plan

> **For agentic workers:** Consult AGENTS.md Section 3. Steps use checkbox syntax. This is the bounded diagnostic phase; a production correction needs an evidence-based revision and hub approval.

**Goal:** Identify why a replacement Zellij disconnects before its first wrapper startup, retain the evidence before cleanup, and propose the smallest correction that covers the demonstrated cause.

**Architecture:** Reuse the shared disposable Couch harness and its root-confined bounded diagnostic reader. Observe existing native processes and startup records without changing admission, ownership, cancellation, or deadlines. A failed client, a server panic, and an external termination remain different observations.

**Tech stack:** Go, the existing Claude/Codex fixture personalities, installed Zellij, existing native-query observation and channel-controlled handoff seams.

## Evidence classification

The original failure is **pre-wrapper replacement transport loss, trigger unclassified**. Its exact replacement session and launch nonce differ from the retired session. The old agent was absent; Pair's exit1 preceded registration cancellation; no replacement STARTUP record was emitted. This is not #797's old-pane ownership refusal. Its private server log was deleted, so neither a panic nor the initiating query/signal is proved.

Both retained log hashes were rechecked on 2026-10-22. The successful diagnostic's private server log contains several `Server not ready` warnings and completes first-client initialization. Those warnings alone therefore do not classify the failed launch. The earlier passing run is a control, not a reclassification of the failure.

The existing birth-race instrument records prior controlled failures on Zellij0.45.1. The [upstream RemoveClient branch](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-server/src/lib.rs#L1374-L1408) unwraps optional session data. This supports a plausible pre-initialization disconnect mechanism; it does not establish that mechanism in #806. Do not run the old instrument unchanged: its shared log attribution and name-based cleanup do not meet this issue's private, exact-generation qualification boundary.

## Current query and exclusion census

| Consumer | Existing path | Current exclusion |
|---|---|---|
| Message listing | `couchcmd/message_service.go`, `launcher/zellij_listing_poller.go`, `launcher/zellij.go` | Immediate and one-second global listing; no terminal lease |
| Resource inventory seed/refresh | `couchcmd/resources_inventory.go`, `couchcore/actionableinventory.go`, `couchcore/artifactcollision.go` | Global presence before per-thread classification; native owner queries for changed rows; no terminal lease |
| Console inventory | `couchtty/console_menu.go`, `couchcmd/run.go` | Independent global inventory; no terminal lease |
| Existing standalone title poller | `titlepoller/runtime.go`, `titlepoller/run.go` | Gates its own pane birth; another session can still be initializing; Couch-hosted wrappers skip this helper |
| Birth watchdog | `launcher/birthwatch.go` | Local sidecar wait, then global liveness observation after existing ten-second threshold |
| Other handoff/lifecycle work | `launcher/session_handoff_os.go`, `session_owner_os.go`, `session_quiescence.go` | Per-canonical-socket lease on cooperating handoffs/recovery, not on all global readers |

`zellijquery.Source` (#803) owns deadlines, collection and bounded read recovery; it adds no birth exclusion. #773 owns one startup deadline and stage progress; it does not make independent readers participate in the terminal lease. The resource sampler and Git/filesystem prewarm are not Zellij query producers. Action-scoped shortcut, clipboard, layout, opener and viewport queries also use the shared collector; their existence is not proof they ran in the failed attempt.

## Alternatives and decision gate

1. **Recommended now: retain evidence and reproduce one causal ordering.** Small diagnostic-only change, then select the correction from the observed trigger. This preserves uncertainty and avoids a speculative concurrency mechanism.
2. **Coordinated birth/query exclusion:** potentially prevents Pair-owned readers from touching initializing servers, but must cover the whole census and separate processes; a process-local flag or one-poller pause cannot enforce it. Uncooperative external clients remain outside that guarantee. Consider only after causal evidence, with the synchronization owner and operating envelope approved.
3. **Upstream correction or packaged dependency change:** can cover external readers too if the upstream defect is confirmed, but changes the deployed dependency and requires operator coordination. No install, stage, wrapper exec change, or dependency pin is authorized by this diagnostic plan.

No deadline tuning, broader retry, automatic relaunch, weakened parser, relaxed ready proof, or new name-based effect fallback is proposed.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|

No new production pure entity or state transition is introduced in this phase. Existing startup identities and native query observations remain immutable diagnostic inputs. Interpretation is an evidence table in the issue, not new runtime authority. A later correction must name its own pure model and causal invariants (ARCH-PURE, ARCH-ORDER).

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Private startup server-log capture | `cmd/internal/testcouch/startup_evidence.go` | modified | Existing `evidenceRoot.read`, private Zellij log |
| Failure-before-teardown collection | `cmd/internal/testcouch/fleet.go` | modified | Existing failure cleanup, before native diagnostic queries/signals and after producer join |
| Native recovery diagnostic | `cmd/internal/couchcmd/recovery_owner_live_test.go` | modified | Existing emitted-profile public reboot and delivery scenario |

Reuse `openEvidenceRoot` and its no-follow/nonblocking regular-file reader; never read ambient machine logs. The candidate log path comes from this Fleet's private TMPDIR plus the installed Zellij log layout, not the parent's `os.TempDir`. Root escapes, symlinks, FIFOs, absent files and read failures remain explicit unavailable evidence. Render retained bytes with quoted escaping, using only the synthetic fixture environment and local test logs; never copy real HOME, account credentials, or an ambient native log (ARCH-DRY, ARCH-SECURE).

## Bounded execution and state

Actions remain inspect, retry explicitly, or stop. No product-visible state changes are introduced.

| Internal observation | Visible classification | Recovery action | Possible loss |
|---|---|---|---|
| Client exits before wrapper/ready, server evidence missing | Trigger unknown | Inspect retained evidence; stop before a correction | Historical server cause cannot be recovered |
| Private server panic plus correlated pre-ready query | Candidate causal ordering | Run one controlled contrast | Timestamp proximity alone cannot prove causation |
| Exact external signal or local cancellation precedes exit | Termination path identified | Trace that effect's ownership and lifetime | Missing signal provenance keeps the cause unknown |
| Public reboot and new-generation delivery succeed | Sample succeeded | Preserve it as a control | A pass does not erase the earlier failure |
| Diagnostic truncation, identity disagreement or cleanup uncertainty | Evidence incomplete | Stop and report | No success or absence inferred from the missing part |

Reuse existing 180s preparation, 120s operation, 15s cleanup, frame/output limits and package 96MiB write cap. One focused Go process per personality; initially one Claude and one Codex run. At most one additional paired causal contrast after a retained failure; otherwise stop and report the remaining uncertainty. No full suite is needed merely to investigate, and no repeated native loop runs until it happens to pass (ARCH-CONSTRAINTS).

Failure capture reads at most8192 bytes per server-log snapshot, at most two snapshots per failed fleet: before any teardown diagnostic query/effect and after all native producers join. Report path, observed offset/size, truncation and errors; never equate a private fleet's shared log line with a specific session without correlated session/launch/server-generation evidence. Test8192/16384-byte files for bounded retained tails and explicit truncation. Preserve the existing reader's `io.LimitReader(limit+1)` byte bound; count capture invocations and require at most two, independent of input lines or file size. Returned length alone is not claimed as proof of physical IO. The captures create no production artifact or worker; existing Go test logs own the retained bytes, private roots are removed only after exact owned cleanup, and no diagnostic copy changes teardown authority (ARCH-FUNERAL).

## Diagnostic tasks, after approval

- [ ] Add failing tests in `cmd/internal/testcouch/startup_evidence_test.go` for server-log retention before cleanup, missing/unsafe paths, bounded tails, error/truncation reporting and no extra native query. Reuse existing private root tests and reader rather than inventing a second file boundary.
- [ ] Implement the small test-harness capture in `startup_evidence.go` and failure collection order in `fleet.go`. Preserve all ownership and cleanup decisions. Capture the original failure before any diagnostic query can alter the observation.
- [ ] Use the existing query observer where a controlled in-process context reaches the actual query; preserve bounded per-attempt timing/argv/exit evidence. Enumerate observed coverage explicitly. Do not invent cross-process completeness, add an ambient trace shim, or change child exec before bringing that additional instrumentation design to the hub.
- [ ] Run `go test ./cmd/internal/testcouch -run '^TestStartupEvidence' -count=1`; run existing handoff budget/admission tests to establish the unchanged baseline. Audit any new repository-input reader certificate before broader verification.
- [ ] On the committed diagnostic head, run one `PAIR_LIVE_RECOVERY_OWNER=1 go test ./cmd/internal/couchcmd -run '^TestLiveRecoveryOwnerReboot/claude$' -count=1 -v`, then the analogous Codex invocation, each with its normal96MiB package cap and separate retained log. Do not stage on :0 or use the live fleet.
- [ ] If a failure is retained, propose and execute at most one causal contrast through the existing channel-controlled handoff and stateful Zellij seam, plus a private native control if necessary. A fabricated fake failure is not proof of the real initiating trigger; add a first-client state to the shared fake only once its native behavior is established.
- [ ] Append exact identities, raw hashes, query coverage, chronology and classification to #806. Send the concrete correction choice and revised implementation/verification plan to the hub **before production code**. If still unclassified, report that limit instead of selecting a fix.

## Completion beyond this phase

The issue remains open until its original Done when is satisfied: retained failing server evidence, causal regression through the real startup seam, an approved correction, and Claude/Codex public startup/reboot+delivery with exact cleanup. The later plan revision must enumerate affected consumers, state/effect ordering, counted cost, fake conformance and any mechanical class guard. Diagnostic logging alone does not close #806 (ARCH-PURPOSE). No persisted production format changes in this phase (ARCH-EVOLVE).
