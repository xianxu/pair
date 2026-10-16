# Couch admission under load: cached listing, failure receipts, provisional codex bindings

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A `couch --send-to` lands, or says why it did not, whatever the machine load; a codex slot driven only by
peer messages becomes relaunchable; the first relaunch does not strand a recover/abandon choice.

**Root cause (measured 2026-10-16, issue Log):** every admission runs the supervisor's use-time ownership check
(`messageAuthority.current`) for the sender and, through `messageEndpoint.check`, before the recipient's Observe and
Reserve. Each run calls `ListedPairSessionContext` → `ZellijSource.LivenessContext`, which runs `zellij list-sessions`
twice. That is about six synchronous listings inside `AdmissionTimeout` (2 s). Every actor with a queued message also
retries `promote` every `PromoteRetry` (1 s), again running Observe and Reserve checks, so the live supervisor spawned
about 7 listings per second, each waking all ~80 session servers. At load average 136 one listing took 0.13–1.07 s, so
admission timed out before `Admit`; no receipt existed, and `--message-status` could only say "no connected recipient
retains this message". The CLI's own call timeout is the same 2 s, so it gives up at the moment the broker does and
never prints the broker's reason.

**Architecture:** the supervisor's message authority reads zellij liveness through one single-flight listing cache with
bounded freshness (`launcher.ListingCache`, 1 s). Only the message authority's checker copy holds the cache, so
recovery and relaunch decisions keep reading zellij directly. The per-call file checks (thread record, PID file, ready
file nonce/session, session index) stay uncached: the authority comment forbids caching live wrapper authority, and
they are cheap. A failed admission stores a `refused` receipt with its reason, which `--message-status`, the send's
last line and `couch --list` show. M2 confirms a codex binding from a delivered peer message and makes the relaunch's
park wait inside its operation.

**Tech Stack:** Go; existing couchmessage broker, couchcmd message service, launcher zellij source, sessionwatch.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ListingCache` freshness and single-flight | `cmd/internal/launcher/zellij_listing_cache.go` | new |
| `Refused` receipt status | `cmd/internal/couchmessage/model.go` | modified |
| `AdmissionRefusal` (reason text from an admission error) | `cmd/internal/couchmessage/refusal.go` | new |
| slot receive state in the list row | `cmd/internal/couchcore` list presentation | modified |
| peer-message turn correlation | `cmd/internal/sessioninventory` / `sessionwatch` | modified (M2) |

- **ListingCache** — holds the last `list-sessions --short` and `--no-formatting` outputs with their time; a read
  within `MaxAge` returns them, a stale read runs one refresh that concurrent readers share; errors are never cached.
  Clock and runner are injected, so the tests use no zellij.
  - **Relationships:** 1 cache : N `ZellijSource` copies (pointer field; copies by value share it).
  - **DRY rationale:** one listing serves every authority read; no caller grows its own memo.
  - **Future extensions:** the console and inventory can opt in by holding the same pointer once their freshness needs
    are stated.
- **Refused / AdmissionRefusal** — a receipt that records that admission never happened and why ("recipient ownership
  check timed out", "wrapper does not own the current binding", ...). It retains like other receipts, so status and
  list read it from the store that already exists; no new tracking.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ZellijSource.Listings` | `cmd/internal/launcher/zellij.go` | modified | `zellij list-sessions` |
| message authority checker copy | `cmd/internal/couchcmd/message_service.go` | modified | the supervisor's artifact checker |
| `Broker.Send` refusal record | `cmd/internal/couchmessage/broker.go` | modified | admission |
| CLI send/status | `cmd/internal/couchcmd/messages.go` | modified | the message socket |
| `couch --list` receive state | `cmd/internal/couchcmd` list path | modified | the supervisor's message socket (best effort) |
| relaunch park wait | `cmd/internal/couchcore/relaunch.go`, `park.go` | modified (M2) | the quit completion |

- **ZellijSource.Listings** — nil keeps today's behaviour everywhere; the stub-zellij tests count calls through it.
- **Message authority checker copy** — `messageThreadResolver` gets a copy of `c.Artifacts` whose `Zellij.Listings`
  is the supervisor's cache.
- **`couch --list` receive state** — `--list` reads the store directly; it asks the running supervisor over the
  message socket (as `--actors` does, short timeout) for each slot's latest receipt to it. A slot whose latest one is
  `refused` shows `can't receive: <reason>` and the one command that fixes it. No supervisor, no line.

## M1 — admission under load (items 1–3)

- [ ] M1 — cached listing for the message authority, refused receipts with their reason, receive state in `--list`

### Task 1: ListingCache

**Files:** create `cmd/internal/launcher/zellij_listing_cache.go`, `zellij_listing_cache_test.go`; modify
`cmd/internal/launcher/zellij.go` (`snapshot` reads both outputs through `s.listings(ctx)`).

- [ ] Failing tests: within `MaxAge` the runner is called once for many reads; after `MaxAge` it refreshes; 8
  concurrent stale readers cause one refresh; a runner error is returned and not cached; nil cache calls the runner
  every time.
- [ ] Implement; run `go test ./cmd/internal/launcher -run Listing`.
- [ ] Existing stub-zellij call-count tests still pass (`go test ./cmd/internal/launcher ./cmd/internal/couchcore`).

### Task 2: the message authority reads through the cache

**Files:** modify `cmd/internal/couchcmd/message_service.go` (`messageThreadResolver` builds its checker copy with
`Zellij.Listings`); test in `cmd/internal/couchcmd/message_service_test.go`.

- [ ] Failing test: a stub zellij that sleeps 1 s per listing; one admission (sender check plus recipient Observe and
  Reserve checks) completes inside `AdmissionTimeout`, and the stub counts at most 2 listings.
- [ ] Failing test: a broker actor with a queued message retrying `promote` for 5 s causes at most ~5 listing pairs,
  not two per retry.
- [ ] Implement; the supervisor constructs one `ListingCache{MaxAge: time.Second}`.

### Task 3: refused receipts

**Files:** `cmd/internal/couchmessage/model.go` (`Refused` status), `refusal.go`, `broker.go` (`Send` records a refused
receipt when `admit` fails before `Admit`), `messages.go` (CLI call timeout gets slack over `AdmissionTimeout`, the send
prints the broker's reason, status prints `refused: <reason>`).

- [ ] Failing broker test: an endpoint whose check fails with a reason → `Send` returns the error, and
  `Status(id)` returns `refused` with that reason (not "no connected recipient retains").
- [ ] Failing broker test: a deadline during admission → `refused` with "recipient ownership check timed out".
- [ ] Failing CLI test: a slow broker answer under the CLI timeout prints the reason on the send's last line.
- [ ] Implement.

### Task 4: receive state in `couch --list`

- [ ] Failing test: the list presentation with a supervisor answer naming a refused latest receipt for `pair:3`
  renders `can't receive: <reason>` and `couch --relaunch pair:3 --confirm`; no supervisor → unchanged rows.
- [ ] Implement with the existing message socket, short timeout, best effort.

### Task 5: live and close

- [ ] Live, after the TL rebuilds pair:0 and restarts Couch: under a full suite, sends to a rebooted and a resumed
  slot land; measure the supervisor's zellij listings per second and its CPU before and after.
- [ ] Atlas: couch.md messaging (authority freshness, refused receipts).
- [ ] `sdlc milestone-close --issue 618 --milestone M1`, then land (the TL wants 1–2 landed before 4–5).

## M2 — provisional codex bindings and the first relaunch (items 4–5)

- [ ] M2 — a delivered peer message confirms a codex binding; the first relaunch waits inside its operation

### Task 6: peer-message turns confirm a binding

The session watcher confirms a binding only from Pair-log turns, which only the draft pane writes. It also accepts a
native round whose user text equals the exact body of a message the broker delivered to that wrapper after
`--pid-not-before`; the wrapper records delivered bodies' hashes where the watcher reads them.

- [ ] Failing test: a codex thread with no Pair-log turns and one delivered peer message whose round appears in the
  rollout → the ledger records a confirmation, and `bindingResumeDiagnostic` is no longer provisional.
- [ ] Failing test: a rollout round whose text matches no delivered message confirms nothing.

### Task 7: the first relaunch does not strand

- [ ] Failing test: a quit that completes after 20 s → the first `--relaunch` call finishes (its operation keeps
  waiting up to a longer bound) instead of returning `ParkIncomplete` at 15 s.
- [ ] Implement in `relaunch.go`/`park.go`; `couch --operation-status` shows the wait.

### Task 8: close

- [ ] Live: reboot one codex slot and resume another; each receives a send; relaunch the rebooted one after a peer
  message only.
- [ ] Atlas; `sdlc close --issue 618`.

## Out of scope (reported to the TL)

The supervisor's own ~1 core is terminal emulation of the hosted session clients (`ptychild.(*Child).ingest` → vt
parser); a separate issue.
