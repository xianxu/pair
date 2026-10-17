---
collaboration-mode: discussion
---

# Stale session-section reminder implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3. Execute this context-warm change with superpowers-executing-plans. SDLC owns plan and close reviews.

**Goal:** Remind an idle hub of its registered session file's stale sections without changing or acknowledging the document.

**Architecture:** A single serialized worker consumes coalesced ticks from Couch's existing wait watcher. It inspects registered documents without side effects, invokes the existing tl-lead freshness command under a budget, and uses exact hub self-delivery through the existing supervision/broker path. Per-registration cooldown and failure suppression live only for this Couch instance.

**Tech Stack:** Go, sessionsync registry/document IO, Couch message broker, existing tl-lead CLI.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| DocumentInfo | cmd/internal/sessionsync/registry.go | new |
| staleReminderState / staleReminderEvent / advanceStaleReminder | cmd/internal/couchcmd/session_stale_model.go | new |
| stale reminder output classification | cmd/internal/couchcmd/session_stale.go | new |

DocumentInfo is the current registration ID, canonical file, effective owner, operator-lock holder and session-sync type flag. Its ID keys the cooldown; changing a wrapper does not create a second per-file allowance. Classification accepts the known fresh sentinel or stale section lines from tl-lead, without reproducing stamp budgets. Unknown, partial, oversized or control-bearing output is a failure, never a reminder. The complete reminder must fit couchmessage.MaxBodyBytes, including its path/prefix.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Sync.InspectRegistered | cmd/internal/sessionsync/registry.go | new | existing locked document load |
| sessionStaleWatcher | cmd/internal/couchcmd/session_stale.go | new | registered files, admitted liveness, tl-lead process, broker |
| waitWatcher / message service startup | cmd/internal/couchcmd/wait_notice.go and message_service.go | modified | existing tick and worker lifetime |

InspectRegistered uses the existing withDocument transaction, refuses a registration that disappeared, and returns current owner/type/lock without calling Read(all/frontmatter), saving a shadow, folding events or changing LastWrite. Status alone is unsuitable because it reads owed-delivery state and does not expose the session type/registration identity. The registered-file size and lock-alias rules remain owned by sessionsync.

The watcher uses the current admitted hub binding, SessionToken, Settled signal and broker ResourceIdle evidence; unknown, disconnected, replaced, picker-held or pending-message owners are skipped. It checks effective owner rather than List's fallback owner. It re-inspects registration/type/owner/operator lock and the exact admitted generation after the external command before submitting a message. The operator lock is sampled as in existing session operations; it cannot be held across tl-lead because the command itself calls Couch reads. Broker receiver admission remains the authority at actual delivery.

## Runtime contract

- The existing 15-second watcher tick offers work to a capacity-one channel without blocking. One worker owns all scan/cooldown state; no overlapping commands or per-tick goroutines. Worker lifetime is service.lifetime and service.workers owns shutdown. A coalesced tick processes current facts, not a captured stale liveness map.
- Registration enumeration retains the existing registry bounds. A pass has a 30-second deadline and advances a fair cursor if it cannot visit all registrations. One tl-lead check has a five-second deadline, bounded stdout/stderr, and bounded pipe draining; its owned process group is killed on cancellation so child Couch/git processes cannot outlive the check.
- A due file is inspected before any command. Held operator locks and unavailable/busy owners do not consume the check allowance, so release/idle can be noticed on the next tick. Completed eligible fresh/stale checks and failed command attempts each consume a 30-minute per-registration check allowance, preventing probes or repeated errors every tick. A stale result produces at most one send attempt per allowance; uncertainty consumes that allowance rather than blindly resending under a fresh ID.
- Send one ordinary exact hub-to-self generated message containing sessionsync.PingPrefix, the file path and the stale output. This reuses existing identity-based ping exclusion from owed-event recording and normal message validation/admission. No new bypass of the broker, message capacity, or wrapper input checks.
- Missing/failing tl-lead or invalid output produces no message and logs one bounded diagnostic per continuous failure episode; success clears failure suppression. Registry/file failures remain conservative skips with a diagnostic, never evidence of freshness.
- Prune per-registration cooldowns to current registry IDs each successful enumeration. The single failure flag, cursor, capacity-one tick channel and map bounded by the registry die with Couch. TL explicitly approved restart resetting the cooldown. No new durable artifact family or runtime setting is introduced.

## Tasks

### Baseline-neutral inspection and reminder logic

- [ ] Add InspectRegistered real-store tests in cmd/internal/sessionsync/registry_test.go: state-transition invariants compare document/shadow/state bytes before and after inspection, and identity/type/owner/alias-lock facts against the current registered document. Run the focused test to observe the missing API, then implement the small inspection seam.
- [ ] Add tests in cmd/internal/couchcmd/session_stale_test.go before implementation. Named strategies: output classification gets arbitrary malformed byte input with message-validation invariants; sessionStaleWatcher.tick gets fake-clock transitions against a real temporary registry plus a stateful fake check/peer endpoint, asserting per-ID allowance and no send across busy/lock/generation transitions. Use barriers around a running check to exercise changed owner/registration/lock evidence mechanically.
- [ ] Implement the worker and wire its nonblocking tick into waitWatcher. Tests hold the check at a barrier to prove wait/supervision ticking continues, commands never overlap, and service cancellation collects the worker and owned command group. Use deterministic process fixtures for output overflow, nonzero exit and cancellation; all Pair/wrapper subprocess fixtures use testpair.Environment's empty allowlist.
- [ ] Run `go test ./cmd/internal/sessionsync ./cmd/internal/couchcmd -run 'InspectRegistered|SessionStale' -count=1`, focused race tests, then both complete affected packages. Commit behavior and concrete red/green evidence.

### Conformance, documentation and one close boundary

- [ ] Exercise the real new Couch/message-service fixture and real tl-lead CLI against disposable registered session files in an isolated namespace, using an admitted disposable wrapper endpoint. Advance only the reminder clock to test the 30-minute allowance; tl-lead reads real old/fresh stamps. Verify actual receipt/body, fresh suppression, operator-lock suppression and delivery after release. The complete document and its unread baseline must stay unchanged. No live hub is messaged; installed-hub smoke is deferred to the restart by TL approval.
- [ ] Update atlas/couch.md and the shipped Couch reference/README where they explain session reminders; state cadence, idle/lock skips, diagnostic behavior, restart cooldown reset and read-only scope. Keep the existing atlas index link.
- [ ] Push the candidate and run `sdlc local-check run --name suite`. Close through `sdlc close --issue 630 --verified '<evidence>'`; resolve fresh-context review findings and record their class in lessons.

Landing protocol: open/update the PR through sdlc, integrate main, run `sdlc merge --yes`, verify the merge and archival on origin/main and report the outcome to the hub. The TL owns the installed-hub smoke after restart.

## Design choices

ARCH-DRY: tl-lead owns section freshness, sessionsync owns document metadata/locks, and the broker owns delivery. ARCH-PURE: classify CLI output separately from file/process/message effects. ARCH-CONSTRAINTS: synchronous work is confined to one budgeted worker; slow checks cannot block the existing watcher. ARCH-ORDER: all post-check guards precede send; uncertain admission is not retried blindly. ARCH-FUNERAL: no new persistent store, and all retained runtime state has a registry/service bound. Adding a second poll timer or parsing section ages in Couch is unnecessary.

## Authoritative transition model

ARCH-ORDER/PURE: advanceStaleReminder is the only writer of the reminder's cooldown, active check and diagnostic-episode state. It is a deterministic in-memory transition function without IO; the serialized worker feeds observations/results and executes returned effects. The state owns next-check times keyed by registration ID, one active check's expected registration/file/owner/admitted wrapper generation, and one continuous-failure diagnostic flag. No map is independently updated by the IO caller.

| Event | Transition and permitted effect |
|-------|---------------------------------|
| Registry snapshot | Prune IDs no longer registered; retain unchanged IDs across wrapper replacement; bounded by the existing registry limit |
| Eligible tick, due ID, no active check | Retain the exact observation as the active check and emit Check; busy/unknown/locked observations emit nothing and consume no allowance |
| Failed check | Clear active check; set its next check to now + 30 minutes; emit one Log for a continuous failure episode; emit no Send |
| Successful completion with invalidated identity, lock or idle evidence | Clear the active check without publishing a result or advancing allowance; a later eligible tick can check again after submit/idle. This aborted observation is not a completed eligible check |
| Successful eligible fresh completion | Clear active check and diagnostic suppression; advance allowance by 30 minutes; emit no Send |
| Successful eligible stale completion | Clear active check and diagnostic suppression; advance allowance by 30 minutes; emit one Send carrying the retained exact binding and validated body |
| Send result, including uncertain/refused | Retain the consumed allowance; emit no retry with a new ID |
| Cancellation | Clear active work, emit no further effects; the worker's context owns command-group termination |

The observation contains the actual DocumentInfo, SlotLiveness, verified hub membership and broker ResourceIdle evidence, rather than a precomputed 'eligible' boolean. The transition function compares the starting observation with completion evidence before a Send effect. Controlled-order sequence tests target this production function directly; integration tests exercise the same transitions through stateful stores and the real broker.

## Conformance cadence and fixture

Default suites exercise the deterministic registry/check/broker fake contract. Run the env-gated real conformance test before landing any change to this worker, Couch's session-sync CLI or tl-lead status-check output, and after changing the supported tl-lead binary. This is the recurring trigger for detecting external CLI drift, not a new CI/local-gate configuration change. The installed-hub smoke remains a TL-owned restart check.

Reuse newServiceRig and the wait-watcher admitted-session helpers, plus a disposable remote endpoint that records real broker commits. Provide the already-built current Couch and tl-lead executables to the live fixture; do not add per-test builds. Put the real status file inside a disposable Git repository nested in a disposable fleet so tl-lead's newest-landing lookup succeeds. Populate its origin/main ref and use actual old/fresh stamps; inject only the reminder clock for the half-hour allowance. The real process environment uses testpair.Environment with disposable HOME, isolated Couch roots and fixture PATH resolving the intended Couch executable. This proves the real CLI/message-service/transport contract with a controlled composer endpoint; the native installed agent composer is reserved for the deferred smoke.

## Revisions

- 2026-10-17: plan gate PQ-1: define one authoritative pure transition boundary and enumerate all retained-state changes, including invalidated evidence, send uncertainty and cancellation. Clarify that an invalidated completion consumes no eligible-check allowance so an operator submit can be noticed on the next tick. PQ-2: name repeatable real-conformance triggers and the Git-backed isolated fixture prerequisite.
