# Reboot caller listing loss — diagnostic implementation plan

> **For agentic workers:** Follow AGENTS.md §3. This plan authorizes diagnostic coverage and one native sample, not an authentication correction or issue close.

**Goal:** Retain the exact listing and selected projection judged by an accepted reboot's failed status authentication, then distinguish actual caller loss from native discovery omission without resending the mutation.

**Architecture:** Carry bounded observation data alongside the existing liveness read and selected binding. The exact same raw bytes feed parsing and evidence; no second read substitutes for the judged frame. The CLI keeps the existing unavailable/uncertain outcome. Stateful tests force accepted -> omitted/exited -> status refusal. The shared disposable harness retains the caller explicitly instead of selecting only the latest four workers.

**Tech stack:** Go, existing ListingPoller, scoped artifact checker, slot-operation CLI, stateful Zellij fixture, testcouch and fixture Claude.

## Approval, limits and interpretation

Verified submitted ops:0 receipt 967ef946-fe21-42f9-9a25-a412cfe8a2da approves this diagnostic phase, the deterministic no-resend regression, and ONE headless disposable Claude run through worker5. Run nothing alongside that native sample. No foreground window, live fleet, profiler, spinner, full native matrix, timeout increase or authorization relaxation. Stop after the sample and report before any behavioral fix. Existing captured N10 failed while starting worker5; this experiment needs five provisioned workers, within main's eight-worker bound, and claims no scale threshold.

The existing error proves only a syntactically valid cached listing missing/EXITED the caller. Positive cache freshness is not proof of instantaneous server truth. Current retained data cannot choose a correction. A successful sample does not classify the original failure and does not authorize retries until failure.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| ListingEvidence | `cmd/internal/launcher/listing_evidence.go` | new |
| SelectedListingEvidence | `cmd/internal/couchcore/listed_session_evidence.go` | new |
| PairSessionBinding | `cmd/internal/couchcore/artifactcollision.go` | modified |

ListingEvidence contains observation time, original byte count, full SHA256, at most8192 raw bytes and explicit truncation. Byte evidence uses JSON base64 for exact non-UTF8 prefixes and a bounded wire size; hash/count refer to the complete parsed input. A pure constructor accepts bytes/time, clones the retained prefix and computes the hash. No file or clock access in the constructor. It is a value attached to one observation, not a retained history.

SelectedListingEvidence combines that exact listing with selected address/name, selected row state (missing/live/exited), claim count and projected presence state. PairSessionBinding carries an optional pointer to it; existing Name/Present/Owner semantics stay identical. No census or alternate session observation runs merely for diagnostic detail. Do not treat missing diagnostics from an old/custom observer as evidence of absence.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| coherent listing read | `launcher/zellij_listing_poller.go`, `launcher/zellij.go` | modified | cached raw/time read and existing full parser |
| selected proof | `couchcore/artifactcollision.go` | modified | existing census and presence projector |
| failed caller diagnostic | `couchcmd/message_service.go` | modified | existing resolver error and public CLI refusal |
| explicit caller capture | `testcouch/startup_evidence.go`, new `testcouch/caller_evidence.go` | modified/new | private files, native read-only listing/pane, selected PID identity |

Expose a single liveness-with-evidence read sharing the existing snapshot/parser implementation. Poller raw and its completion timestamp are captured under its existing mutex; formatting/hashing occurs outside the lock. A direct read records completion time after the existing native query. Existing callers still use LivenessContext and do not pay new evidence/hash costs. Evidence never changes stale/parse/error decisions. A later poll cannot mutate the returned bounded evidence or relabel its timestamp.

The checker returns evidence from the same successful liveness read and census used for ProjectSessionPresence. The message resolver includes JSON evidence only in its existing failed-positive-list Cause, retaining RelaunchRequiredError and all identity/ownership checks. The enclosing unavailable response, request ID and uncertain/no-resend text stay unchanged. Keep this bounded evidence below the existing frame/output limits; no protocol bypass or new public mutation.

The harness starts from the real public Actors response and freezes the caller binding before worker launches. Explicit caller capture prioritizes that bound address over latest-worker sources while preserving the existing four-source/file/pane bounds, private descriptor/path checks, native socket confinement and shared diagnostic deadline. A separately timestamped native listing and selected wrapper/fixture identity are later evidence, never the judged listing or authorization. Missing pane/file/process/receipt remains explicitly unavailable. Do not impersonate another actor to read a receipt. No environment dump or broad new process census.

Reuse stateful zellijtest faults and the real ListingPoller/resolver for deterministic omission/exited/recovery sequences. Reuse portable testcouch roots, real binaries, ActorCLI and exact-generation cleanup for native conformance. No custom fleet owner, query shim or second process-lifetime mechanism.

## State and recovery

Actions: inspect retained proof; inspect later caller evidence; report classification to hub.

| Internal state/failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Accepted, caller remains listed | normal polling/result | inspect retained proof if needed | none added |
| Accepted, later judged missing/EXITED | existing uncertain outcome plus exact bounded evidence | inspect later caller evidence | effect may have completed; no resend |
| Listing stale/invalid or census contested | existing distinct refusal | inspect retained proof | missing raw evidence is explicit, never inferred empty |
| Listing changes after refused proof | immutable earlier evidence and separately labelled later sample | report classification | truncated bytes omitted but full count/hash retained |
| Native sample times out or cleanup uncertain | explicit bounded incomplete result | report classification | no root-cause claim; existing owner retains cleanup responsibility |

No new durable state or background worker. Evidence values die with requests/results; test output is retained only by its existing owner. No journal/ledger format changes. ARCH-DRY keeps one parser/projection, ARCH-PURE keeps evidence construction testable, ARCH-ORDER preserves uncertain outcomes, ARCH-FUNERAL creates no new artifact family, ARCH-EVOLVE leaves persisted formats unchanged.

## Tasks

- [ ] Write red evidence tests: exact bytes/time/hash/count; >8192-byte prefix truncation; empty legitimate listing; copied bytes survive source mutation; cached refresh after read cannot change evidence; malformed/stale controls retain their existing errors. Run focused launcher tests, implement the shared coherent read, rerun. Count one poll refresh at8/16 rows and zero subprocess reads during repeated evidence reads. Hashing O(B) only on requested evidence; parsing remains one existing pass.
- [ ] Write red checker/resolver tests with the real stateful fixture: selected live -> missing and EXITED, conflicting claimant, malformed refresh, and subsequent restored listing. Assert diagnostics retain selected address/name/claim/state and exact listing identity after restoration. Preserve all existing positives/negatives; no extra native reads per status. Implement evidence propagation, then run focused couchcore/couchcmd tests.
- [ ] Add accepted-then-lost CLI integration regression: exactly one reboot admission, all later requests operation-status with the same ID, failure remains uncertain and no mutation retry. Drive the actual resolver-backed refusal rather than a handwritten error string. Assert evidence survives response/CLI formatting within MaxFrameBytes and fixture output limits. Run focused tests and focused race variants sequentially.
- [ ] Add bounded explicit-caller harness capture and tests for caller older than four worker records, malformed/missing identity, cross-root/symlink refusal, shared deadline/cancellation, truncation and failed native queries. Retain existing capture behavior for unrelated startup failures. New native test in `cmd/couch/caller_listing_diagnostic_test.go` uses public Actors binding, shared testcouch with5 workers, existing setup180s and runtime120s bounds, per-process write96MiB, no workload/profile, one Claude sample. It logs exact candidate build and later observations before deferred exact cleanup. If120s expires before worker5, report that limitation without raising it.
- [ ] Commit/push and run the targeted diagnostic tests (count1; never >10) before native execution. Run the new opt-in native test once, headless, alone in this slot, with an overall test timeout covering existing build/setup/runtime/cleanup bounds. Preserve all failures and cleanup evidence. Stop after this sample and report to the hub. Do not close, merge, rerun native, select a fix or revise Done-when without further instruction.

Targeted commands: `go test ./cmd/internal/launcher -run 'ListingEvidence|ListingPoller|SessionListingCost' -count=1`; `go test ./cmd/internal/couchcore -run 'SelectedListing' -count=1`; `go test ./cmd/internal/couchcmd -run 'MessageThreadResolver|SlotOperation.*LostListing' -count=1`; `go test ./cmd/internal/testcouch -run 'CallerEvidence|StartupEvidence' -count=1`. Run one at a time. Derive the exact opt-in native invocation from the added test and record it before use. No queued full check is needed merely to take the approved diagnostic sample; normal recorded checks and close gates still precede any eventual landing.

## Revisions

2026-10-24 — Diagnostic phase approved; original issue still requires measured classification and any approved correction. This plan cannot establish that root cause in advance. Return the one native result before changing behavior.

2026-10-24 — Corrected a budget attribution before native execution: the existing96MiB guard is the internal/couchcmd TestMain self-process write counter, not a native-child cap, and cmd/couch has no such TestMain guard. Preserve actual existing fixture byte/file limits, setup180s/runtime120s, CLI deadlines and exact cleanup; no new budget mechanism is authorized or needed. This supersedes the per-process write96MiB phrase in the task above. Report that qualification limit plainly.

2026-10-24 — The approved single native Claude sample passed through worker5 on f24dc287ba445bb177a119519049f4005b676770,124.74s, exact cleanup confirmed. No original refusal recurred; original cause remains unclassified. Diagnostics and deterministic uncertainty/no-resend controls are implemented and targeted/race verified. Execution is stopped pending the hub's disposition; the original issue's root-cause Done-when has not been met and no behavior correction or additional sample is authorized.
