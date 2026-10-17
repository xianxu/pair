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
- A due file is inspected before any command. Held operator locks and unavailable/busy owners do not consume the check allowance, so release/idle can be noticed on the next tick. Successful fresh checks, stale checks and failed command attempts each consume a 30-minute per-registration check allowance, preventing probes or repeated errors every tick. A stale result produces at most one send attempt per allowance; uncertainty consumes that allowance rather than blindly resending under a fresh ID.
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
