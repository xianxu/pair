---
id: 000152
status: working
deps: [149, 155]
github_issue:
created: 2026-08-25
updated: 2026-08-29
estimate_hours: 9.90
started: 2026-08-27T12:04:27-07:00
---

# couch: verified park and resume lifecycle

## Problem

`#149` makes a durable work thread distinct from its optional live actor
incarnation. The thread menu needs a normal way to make a live thread inactive
without deleting its draft, ledger, transcript, native-session identity, name,
or description. Signalling couch's top child is insufficient: Pair hosts a
zellij server session whose panes can outlive that client, so freeing the
concurrency slot at that point would admit a second writer beside the first.

The same quiescence proof is needed by `#135` before transferring a live tag to
another agent. This issue owns that reusable lifecycle protocol; the menu in
`#151` is only a client.

## Spec

**Park is an observed all-or-fail transition.** It succeeds only after the
Pair-owned zellij session is absent and every Pair-recorded
agent/wrapper/editor PID is dead or has a mismatched process identity. Until
then the incarnation remains occupied. Unknown liveness, partial shutdown, or
timeout is a failed park; the thread remains live/unknown and its admission
slot stays consumed. Park does not wait for an LLM turn or a separate semantic
flush acknowledgement: ariadne-style work is durable in the repository, and
#155's identified native transcript tree is the reconstruction substrate.

The coordinator owns one ordered transaction: request graceful shutdown,
verify the complete recorded process set and exact Pair session absence,
preserve #155's native session-tree binding, persist the incarnation removal
plus a monotonic `last_active_at`, then publish the parked result. It never
records the timestamp or frees capacity before verification. Forced kill is a
separately named recovery action and does not silently masquerade as park.

The process inventory and shutdown seam are shared with `#135`; impossible
fake-only cleanup is forbidden. Tests use a stateful Pair/zellij/process fake
whose sessions and PIDs change only in response to modeled operations, plus a
real process-level conformance probe. Recovery after interruption is explicit:
re-running park resumes verification from durable transaction state rather than
starting a second transition or guessing success (ARCH-MOCK, ARCH-PURPOSE).

The coordinator commits verified park, `last_active_at`, and incarnation
removal through #149's locked `ThreadStore` transaction; it does not own a
parallel lifecycle file. This issue enables `couch resume <ref>` once that fact
exists. Resume accepts only a uniquely resolved, verified-parked thread. It
refuses live, unknown, partially parked, legacy-unverified, missing-path, and
ambiguous records with an exact recovery diagnostic; it never guesses that
absence means quiescence.

Resume uses the thread's latest successfully registered launch profile—the same
agent, exact argument vector, and saved native-session identity. It does not
consult the path's new-thread preference, root agent, or current repository
default. A missing/unsupported agent or invalid saved profile refuses with an
exact recovery diagnostic rather than silently changing harnesses. #153 may
later intercept a missing managed `working_path`, deterministically reprovision
and rebind it, then call this unchanged resume operation on the now-valid path;
until that integration lands, missing paths remain a refusal.

## Revisions

### 2026-08-28 — recovery uses deterministic native session trees

**Reason:** an LLM may be unavailable or overloaded during park, while the
repository and native transcripts already preserve reconstruction material.
The actual missing evidence is a reliable mapping from Pair tags to complete
agent session trees, including subagents.

**Delta:** remove the semantic flush-acknowledgement requirement. Park proves
writer and Pair-session absence before releasing capacity and preserves the
session-tree binding from #155. #152 now depends on #155's deterministic,
fail-closed forest inventory and correlation instead of treating a newest or
first transcript probe as recovery truth (ARCH-DRY, ARCH-PURPOSE).

### 2026-08-29 — Couch park is Pair full quit, not detach

**Reason:** design review exposed a noun mismatch. Pair already has the lifecycle
the operator means by “park”: Alt+x fully quits the Pair/Zellij session while
retaining the durable information needed by `pair resume <tag>`. Alt+d merely
detaches a client while the session remains live; Couch switching already covers
that useful behavior, so exposing detach would add a confusing duplicate state.

**Delta:** this revision supersedes the earlier independent process-coordinator
design. Couch does not build a second PID inventory, shutdown protocol, or
forced-cleanup implementation. Pair's existing Alt+x path remains the sole owner
of full-session teardown: quit marker, exact Zellij-session quiescence, embedded
editor cleanup, live-sidecar cleanup, durable session/config preservation, and
the established #155 native binding. The lifecycle core gains a typed,
durable completion result so direct Alt+x and Couch-driven park execute the same
behavior and Couch never infers success merely from a client disconnect.

Couch park first commits one `parking` transaction against the exact composite
thread address and expected live incarnation, then requests Pair full quit with
that transaction's stable nonce. The Couch menu's explicit confirmation replaces
Pair's local Alt+x confirmation; it does not synthesize keystrokes or show a
second modal. Because the Couch action is explicitly named park, it selects the
existing preserve-scrollback cleanup branch without another nudge; direct Pair
Alt+x retains its current optional scrollback prompt. Pair publishes exactly one
nonce-bound completion after its full quit cleanup succeeds. A client exit,
missing completion, cleanup warning/error, timeout, replacement incarnation, or
revision conflict leaves the thread occupied with an exact retryable diagnostic.
Reissuing park joins or resumes the durable transaction rather than spawning a
second teardown.

Only the matching completion may atomically remove the expected incarnation,
record a monotonic `last_active_at`, and mark the durable thread verified parked.
Every durable resumability artifact remains owned by Pair and #155; Couch stores
only lifecycle state and the latest exact successful launch profile needed to
re-enter the same address. Restart recovery consumes the same nonce-bound result
and cannot convert absence alone into proof.

Resume accepts only one exact verified-parked thread. It reuses the same
`{repo_scope, tag}`, latest successful agent and argument vector, current working
path, and established native-session binding through Pair's existing
`resume <tag>` flow. It never allocates a new tag or consults new-thread path,
root-agent, or repository defaults. Live, parking, failed/unknown, ambiguous,
missing-path, invalid-profile, unsupported-agent, and non-established native
bindings refuse before starting a child. There is no Couch detach operation
(ARCH-DRY, ARCH-PURPOSE).

The operating envelope is explicit (ARCH-CONSTRAINTS): park confirmation must
publish `parking` within 100 ms on the target development machine and must never
hold the UI beyond 1 second; full quit continues asynchronously under Pair's
existing bounded cleanup. One coordinator may exist per thread, repeated calls
coalesce by nonce, and no corpus/process-tree scan enters the interaction path.
Timeout or unavailable evidence fails closed with capacity still occupied.
Resume uses the existing local start path and adds no full native-inventory scan;
its startup tolerance is the same as ordinary `pair resume`.

The revised acceptance contract is:

- Couch park and direct Pair Alt+x share one full-quit lifecycle implementation;
  a regression in either path fails the same stateful lifecycle tests.
- The UI returns promptly with `parking`, while only a matching successful Pair
  cleanup completion releases the incarnation and records `last_active_at`.
- Interrupted, failed, timed-out, duplicated, or stale park attempts remain
  occupied and are recoverable without repeating teardown.
- Resume recreates the same composite thread through the exact saved profile and
  established #155 binding; no fallback may create a different conversation.
- Couch exposes park and resume through its shared operation schema and exposes
  no detach operation.

### 2026-08-29 — authoritative transaction and operating contract

**Reason:** fresh spec review found that the preceding semantic correction left
the original coordinator-oriented `## Done when` and `## Plan` in force and did
not fully specify crash ordering, evidence ownership, or bounded UI behavior.

**Delta:** this revision is the authoritative replacement for the original
`## Spec`, `## Done when`, and `## Plan` wherever they conflict. Pair's existing
Alt+x full-quit lifecycle is the one teardown implementation. Couch contributes
only durable orchestration around that lifecycle; it does not enumerate or kill
a parallel PID set. Direct Pair Alt+x and Couch Park enter the same typed cleanup
core. Direct Alt+x has no Couch transaction and retains its local confirmation
and optional scrollback prompt. Inside Couch, Alt+x is routed to the declared
`park` operation before Pair's modal, exactly like menu Park; Couch confirmation
selects preserve-scrollback and no second modal or synthetic keystroke is used.
Alt+d remains Pair-local detach behavior and has no Couch operation.

The shared boundary has versioned `QuitRequest` and `QuitCompletion` records.
A Couch request binds its stable park nonce, exact `{repo_scope, tag}`, expected
ThreadStore revision, expected incarnation PID and process identity, exact Pair
session, requested cleanup mode, and completion location. A completion repeats
those bindings and records success or a typed cleanup failure plus completion
time. Pair publishes completion only after the existing full cleanup returns;
publication is atomic and durable (temporary file, file sync, rename, directory
sync). Couch treats only a schema-valid completion matching every expected
binding as evidence. Direct Alt+x uses the same cleanup result but requires no
Couch nonce or completion consumer.

The pure `ParkTransaction` transition system has `requested`,
`awaiting_completion`, and `unknown` nonterminal phases. Its terminal success is
the thread's verified parked metadata, not a second lifecycle file. Couch must
commit `requested` through ThreadStore before publishing a quit request or
causing any shutdown side effect. After request publication it commits
`awaiting_completion`. A matching successful completion plus a still-matching
ThreadStore revision/incarnation permits one atomic commit that removes the
incarnation, records monotonic `last_active_at`, and records verified park. The
completion remains available until that commit succeeds, making consumption
idempotent across Couch restarts; post-commit file cleanup is best effort.

Failures are typed as `request_publish_failed`, `cleanup_failed`, `timeout`,
`completion_missing`, `stale_completion`, `revision_conflict`, or
`replacement_incarnation`. They preserve the occupied incarnation and exact
diagnostic. Reissuing a `requested` transaction republishes the same request;
reissuing `awaiting_completion` while the exact Pair session still exists may
reissue the same idempotent request/signal. If the session is absent without a
matching completion, the transaction becomes `unknown`: it does not mint a new
nonce, repeat teardown, free capacity, or infer success. Stale completion,
revision conflict, or a replacement incarnation likewise stays occupied and
requires explicit operator recovery. A retry never targets an incarnation that
does not match the original PID and process identity.

The architecture separates a pure core from thin IO (ARCH-PURE). Pure entities
are request/completion validation, park transition advancement, lifecycle-result
classification, and resume eligibility. IO seams are ThreadStore CAS/journal,
the Pair quit request/completion store and exact-session trigger, #155's exact
binding query, the runner's existing-thread launch path, and an injected clock.
The successful typed Pair cleanup/completion boundary is also the quiescence
proof consumed by #135; no second coordinator or proof vocabulary is introduced
(ARCH-DRY, ARCH-PURPOSE).

The Pair lifecycle seam has one stateful fake modeling request publication,
exact-session presence, ordered cleanup effects, completion publication, and
restart. Both direct Alt+x and Couch Park tests drive that same seam and cleanup
core. An opt-in live conformance test runs a controlled Pair/Zellij session and
checks the modeled request, cleanup, and completion behavior against the real
boundary. Neither production nor test success may be produced by a stateless
function-call mock (ARCH-MOCK).

The operating envelope is enforceable (ARCH-CONSTRAINTS): the Couch key/menu
handler renders local `parking…` on the next event-loop turn, with a target under
100 ms based on the operator's interactive-latency requirement. The durable
`requested` commit then runs without blocking input, with a 100 ms soft target
and a hard 1 second deadline before any quit side effect. If that commit has not
succeeded by 1 second, Couch renders the typed failure, publishes no request,
and leaves the thread live and occupied. Tests use injected barriers and clocks,
not wall-clock sleeps, to prove ordering and deadline behavior. After the commit,
cleanup is asynchronous and uses Pair's existing bounded exact-session cleanup;
a cleanup deadline records `timeout` and remains occupied. One transaction may
run per thread, repeated requests coalesce by nonce, and the interaction path
performs no corpus or broad process-tree scan. Resume adds no inventory scan and
inherits the ordinary local `pair resume` startup envelope. CPU, memory, network,
and workload growth add no separate material budget because the operation is one
local thread transaction with constant fan-out; concurrency is bounded to one
coordinator per thread and the existing Couch admission limit.

The authoritative acceptance and implementation plan are:

- [x] Define and unit-test the pure park phases, typed failures, binding
      validation, idempotent retry/recovery, monotonic recency, and resume
      eligibility.
- [x] Extract one typed Pair full-quit lifecycle boundary used by direct Alt+x
      and Couch Park; add durable request/completion publication without
      changing direct Alt+x semantics.
- [x] Add the stateful lifecycle fake and exercise both entry paths against it,
      plus an opt-in controlled Pair/Zellij conformance check.
- [x] Persist Couch's transaction before side effects, route Couch Alt+x/menu
      Park through it, enforce prompt UI ordering and the 1 second pre-side-effect
      deadline, and keep every non-matching/failure state occupied.
- [x] Resume only a uniquely resolved verified-parked thread through an extracted
      existing-thread launch path using the same address, exact saved profile,
      and established #155 native binding; never allocate a new tag or fall back.
- [x] Export the same typed successful cleanup proof for #135, update the shared
      operation schema with park/resume only, and document recovery and the code
      surface in `atlas/`.

### 2026-08-29 — revision tokens, durable publication, and recovery matrix

**Reason:** the second fresh spec review found that one “expected revision” could
not identify a transaction across multiple CAS commits, that rename-before-sync
left an authority window, and that timeout/recovery behavior was not exhaustive.

**Delta:** a park transaction has three distinct concurrency values. Its
`base_revision` is checked only when `requested` is first created. Its stable
identity is `(nonce, address, incarnation PID, incarnation process identity)`
and never changes. Each persisted phase has its own `record_revision`; every
advance CASes the current revision and writes the next one. Quit request and
completion bind the stable identity, not a revision that phase commits would
invalidate. Finalization CASes the current `awaiting_completion` (or timed-out
awaiting) record revision and validates that its stable identity still names the
current incarnation. Any unrelated record change is a revision conflict, never
permission to weaken incarnation matching.

Completion publication uses one per-transaction lifecycle lock shared by Pair
and Couch. Pair writes and file-syncs a temporary payload, renames it to the
final path, directory-syncs, and only then releases the lock with a committed
result. Couch acquires the lock before reading and never accepts a temporary
file. If Pair reports an error before rename it removes the temporary file. If
rename occurred but Pair failed or crashed before directory sync, the next lock
holder validates the final payload and directory-syncs it before treating it as
committed; sync failure leaves the transaction indeterminate and occupied. Thus
a visible rename is prepared evidence, not authority, and restart reconciliation
either completes its durability, rejects/removes an invalid payload, or retains
`unknown` when durability cannot be established. All cleanup and reconciliation
are idempotent under the same stable transaction identity.

The exhaustive transition policy is:

| Durable phase / evidence | Durable outcome | Occupancy | Retry and nonce | Operator recovery |
| --- | --- | --- | --- | --- |
| `requested`; request not published | remain `requested`, optionally `request_publish_failed` | occupied | automatic/manual republish of the same request and nonce; no cleanup has begun | fix the reported IO error, then Retry Park |
| `requested`; request durably published | CAS to `awaiting_completion` | occupied | same nonce only | none |
| `awaiting_completion`; exact session remains before deadline | remain awaiting | occupied | duplicate Park coalesces; no new worker | none |
| `awaiting_completion`; matching success | CAS to verified parked | released only by this CAS | nonce closes successfully; duplicate completion is idempotent | none |
| awaiting; matching `cleanup_failed` completion | record closed failed attempt | occupied | failed nonce never changes outcome; a new nonce is allowed only after the exact original session and incarnation are again proven live | Retry Park when offered; otherwise run Recover Park |
| awaiting; cleanup deadline expires while session state is known | retain awaiting with `timeout` | occupied | no automatic second teardown; a late matching success/failure remains authoritative; explicit Retry reissues the same idempotent request only if the exact session/incarnation is still live | Retry Park, or Recover Park if the session disappeared |
| awaiting; exact session absent and no matching completion | `unknown` with `completion_missing` | occupied | no new nonce and no inferred success | Recover Park asks Pair's same lifecycle core to reconcile the original request and publish the original nonce's result |
| any active phase; invalid/mismatched completion | retain phase with `stale_completion` | occupied | ignore evidence; do not retarget or mint a nonce | remove/quarantine the named stale artifact, then Retry/Recover the original park |
| any active phase; unrelated ThreadStore revision or replacement incarnation | `unknown` with `revision_conflict` or `replacement_incarnation` | occupied | no teardown or automatic retry | Abandon Park clears only the stale transaction after explicit confirmation, never the current incarnation; then park the current thread separately |
| any reconciliation; lock or sync authority indeterminate | `unknown` with `completion_missing` | occupied | no automatic retry | repair storage, then Recover Park with the same nonce |

`Retry Park`, `Recover Park`, and `Abandon Park` are recovery modes of the park
lifecycle API, not detach operations. #151 may present them only when the durable
state makes them valid. Recover delegates to Pair's same typed cleanup core in
reconciliation mode; it may re-observe and finish idempotent cleanup for the
original request, but Couch itself never turns absence into proof. Abandon
removes transaction metadata only and cannot remove, replace, or release an
incarnation.

The cleanup coordinator carries a context with a 10 second hard default budget,
a shutdown-path domain assumption that leaves headroom around the existing
authoritative 5 second exact-Zellij quiescence wait (`launcher.zjTimeout`). Every
cleanup subprocess/observation must use the remaining context; exceeding it
publishes or records `timeout`, never success. The default is configurable at the
injected lifecycle seam for deterministic tests, but production has one source
of truth. A late matching completion is accepted according to the matrix because
timeout means “not yet proved,” not “proved failed.”

Representative ARCH-CONSTRAINTS verification runs 100 feedback/commit samples
on the target M2 Max under ordinary development co-tenancy: feedback-render P95
must be below 100 ms, requested-commit P95 below 100 ms, and every requested
commit below the 1 second hard deadline. This is a smoke measurement, while
clock/barrier tests enforce correctness. The claim excludes adversarial OS CPU
starvation. The first handler action is still feedback rendering; lifecycle work
uses a bounded queue no larger than Couch's admission capacity, coalesces a
thread already queued/running, and immediately reports overload without side
effects when the queue is full. It never creates an unbounded goroutine or test
fan-out.

The real Pair/Zellij conformance probe is required at the release gate and when
the quit request/completion or cleanup seam changes; environment-enabled CI may
run the same probe more often. Ordinary hermetic CI always runs the stateful fake
suite (ARCH-MOCK).

### 2026-08-29 — append-only attempts and crash-safe request authority

**Reason:** the third review found that immutable failed completion evidence
could not be reconciled to success under the same nonce, and that requests and
the cross-process lock needed the same explicit crash contract as completions.

**Delta:** the stable park transaction nonce owns an append-only sequence of
numbered attempts. Every request/result binds `(transaction nonce, attempt,
address, incarnation PID, incarnation process identity)`. A failed result closes
only that attempt and is never overwritten. Retry or Recover CASes ThreadStore
to append the next attempt under the same stable transaction nonce after applying
the matrix's eligibility checks. Duplicate operations coalesce within the active
attempt. A schema-valid successful result from any attempt in the transaction is
monotonic quiescence evidence and may finalize the transaction while its exact
incarnation still matches; a late success therefore remains useful even if a
later reconciliation attempt was opened. Once success finalizes park, later
results are historical no-ops. Failures never outweigh an already established
success. This attempt sequence supersedes matrix wording that said Retry
“reissues” an already closed failed attempt; only an active timed-out attempt may
receive a duplicate idempotent trigger without first appending an attempt.

Quit requests use the identical prepared/committed file protocol as completions,
under the same transaction lifecycle lock: write and sync temporary payload,
rename, directory-sync, then release the committed result. Pair's exact-session
trigger is forbidden until the final request has been validated and its directory
sync has succeeded. A Couch crash after request rename but before directory sync
cannot itself trigger cleanup. On restart, Couch (or Pair only after an explicit
trigger) acquires the lock, validates the final request, completes directory sync,
and only then may trigger/consume it; an invalid or unsyncable prepared request
causes no side effect and remains occupied. Temporary request files are never
consumed.

The lifecycle lock is a stable local lock inode protected by a kernel advisory
lock whose ownership is tied to the open file description and automatically
released when the holder process exits; the inode is never renamed or removed
during a transaction. There is no timestamp-based stale-lock guessing. Supported
macOS and Linux implementations must provide these semantics behind the IO seam;
failure to acquire or verify them is an occupied typed IO failure. The stateful
fake and the live conformance probe both cover holder death after final-file
rename, acquisition by the next process, reader-assisted directory sync, and
exactly-once triggering after authority is established.

### 2026-08-29 — transaction closure and at-least-once delivery

**Reason:** the fourth review found that a late success could outlive Abandon,
and that exactly-once trigger delivery is impossible across a crash between
delivery and durable acknowledgement.

**Delta:** success finalization requires all of: the transaction nonce is still
the active nonce on the thread, its durable phase is success-eligible
(`requested`, `awaiting_completion`, timed-out awaiting, or `unknown`), it is not
tombstoned, and the exact incarnation still matches. Abandon does not erase the
transaction identity; it durably closes and tombstones the nonce before allowing
a later transaction. Every result arriving for a completed or tombstoned nonce
is a historical no-op even if the old incarnation identity still happens to
match. Tombstones live in ThreadStore's durable lifecycle history/journal so a
Couch restart cannot forget them.

Trigger delivery is at least once. Pair deduplicates cleanup effects by
`(transaction nonce, attempt)` under the lifecycle lock: an existing committed
result is returned, an in-progress attempt coalesces, and a crash-incomplete
attempt resumes the same idempotent cleanup rather than creating parallel work.
The previous paragraph's “exactly-once triggering” requirement is superseded;
the stateful fake and live conformance instead prove at-least-once delivery with
one effective cleanup outcome per attempt.

A valid success from an older attempt atomically completes the whole active
transaction. That commit prevents queued newer attempts from being triggered and
marks already-running newer attempts as obsolete. Their Pair cleanup remains
idempotent and may finish, but their results are historical no-ops: no later
attempt may mutate the parked thread, publish a second Couch authority, release
capacity twice, or change `last_active_at`.

## Done when

- A live Pair thread parks only after zellij absence and recorded-process death
  are both observed, without depending on an LLM response.
- Partial, unknown, timeout, and interrupted transitions retain the admission
  slot and give an exact retry/recovery state.
- Successful park preserves every tag-scoped artifact and #155 native
  session-tree binding, and records one monotonic `last_active_at`; resuming
  uses the same composite address and latest exact successful launch profile.
- `#135` can consume the same quiescence proof without a second coordinator.
- Stateful fake and real process-level conformance cannot claim cleanup the
  production stack did not perform.

## Plan

- [x] Model the park transaction, process inventory, and recovery states as a
      pure transition system consuming #155's session-tree binding.
- [x] Put Pair/zellij/process shutdown and observation behind one stateful seam.
- [x] Implement graceful park, verification, atomic slot release, and recency.
- [x] Exercise resume-after-park and the shared #135 quiescence boundary.
- [x] Document operator recovery and wire the lifecycle into #151.

## Log

- 2026-08-30: Operator smoke exposed that `park: done` was false: Couch wrote a
  quit intent but left Pair blocked inside its Zellij handoff, then a single
  missing-completion observation returned nil while both the Pair child and
  Zellij session remained live. Fixed the class, not only the smoke instance:
  the trigger now quiesces the exact indexed session; user operations wait
  boundedly for both matching durable completion and death of the exact
  `{pid, process identity}` before finalization; timeout retains the occupied
  transaction and completion evidence. Startup recovery stays non-blocking
  (ARCH-PURE, ARCH-MOCK, ARCH-CONSTRAINTS).
- 2026-08-30: Added the missing terminal semantic. Root Alt+x is typed `leave`:
  it parks all active/parking threads sequentially and exits Couch only after
  all are verified; non-root Alt+x parks only that actor. Inactive threads now
  render as ordinary historical rows (no `[parked]`) and reject stale local
  target binding, so their appearance and Enter/resume behavior are identical
  before and after Couch restart (ARCH-PURPOSE).
- 2026-08-30: Closed the post-Leave reachability gap found during completion
  review. `couch resume <ref>` now acquires the singleton owner and terminal just
  like `start`, then hosts the exact verified-parked thread as the new root
  console. It does not create a fresh actor first, so capacity-one admission
  cannot strand the parked root. The CLI path is covered through the stateful
  process/session fake (ARCH-PURPOSE, ARCH-PURE, ARCH-MOCK).
- 2026-08-30: close-gate round 1 returned REWORK (BR-1..BR-3). Split startup
  inspection from a context-bound serial recovery worker, so a barrier-blocked
  production trigger cannot delay construction. Generalized the lifecycle
  scenario over one driver interface and added a controlled real adapter that
  exercises crash-after-rename request/completion recovery, duplicate production
  `TriggerQuit`, real Zellij cleanup, exact OS child death, and ThreadStore
  finalization with full trace equality against the stateful fake. The live
  workflow now watches `artifactcollision*.go`. Appended an authoritative
  delivered Core Concepts table and a Go-AST contract resolving all 19 entries
  (ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK, ARCH-CONSTRAINTS).

### 2026-08-25

Split from #149 after fresh spec review showed that “stop every process that can
write and flush durable output” is a process-coordination protocol, not a field
on the durable thread record.

- 2026-08-29: unblocked after #149/#155 completion. Operator clarified that
  Couch park is Pair Alt+x full-quit semantics, while Alt+d detach has no Couch
  operation; revised the design around one shared Pair lifecycle path.
- 2026-08-29: fresh spec review rejected the first revision for competing old
  coordinator requirements and underspecified crash/failure behavior. Added an
  authoritative transaction, evidence, retry, pure/IO, fake/conformance, and
  latency contract before implementation (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE,
  ARCH-MOCK, ARCH-CONSTRAINTS).
- 2026-08-29: two further review rounds separated stable transaction identity
  from per-phase CAS revisions, made request/completion publication recoverably
  durable under an OS-released advisory lock, and made recovery append immutable
  numbered attempts rather than rewriting failed evidence.
- 2026-08-29: wrote the durable implementation plan at
  `workshop/plans/000152-couch-verified-park-resume-plan.md`; all four chunks
  passed independent plan-document review after durability, migration, cleanup,
  resume, UI-ordering, conformance, performance, and close-workflow corrections.
- 2026-08-29: `change-code` plan-quality round 1 accepted the architecture but
  raised `PQ-1` (`plan-artifact-compression`) because the task bodies enumerated
  cases and procedural diff steps. Appended an authoritative function-level
  execution map with one adversarial/mechanical guard strategy per risky
  function family while preserving the reviewed design history.
- 2026-08-29: completed implementation-plan Chunk 1. Added the shared versioned
  Pair lifecycle protocol and validated artifact paths (`473b4d4`), immutable
  crash-safe request/completion publication with stable advisory locking
  (`b85de48`), and the pure append-only park reducer plus strict thread schema
  v1→v2 migration (`f1f3b56`). ThreadStore park mutations now use journaled
  revision CAS; only matching success atomically removes the incarnation, while
  abandonment and every conflict retain admission. Focused `pairlifecycle`,
  `artifactpath`, `threadrecord`, and full `couchcore` suites passed with one
  `go test -p 20` process at a time (ARCH-PURE, ARCH-DRY, ARCH-CONSTRAINTS).
- 2026-08-30: completed implementation-plan Chunk 2. Pair now owns one typed
  full-quit cleanup path for both direct Alt+x and Couch requests (`2725d01`,
  `578cf44`). Couch adds a durable park coordinator and bounded per-address
  worker: requested CAS precedes publication, committed authority precedes the
  exact-session trigger, every eligible attempt is reconciled before newer
  delivery, and only a matching successful completion releases admission.
  Constructor recovery scans only active ThreadStore transactions; stale,
  indeterminate, revision-conflicted, replaced, failed, and abandoned outcomes
  remain occupied or historical as specified. Focused and cross-package tests
  passed with a single `go test -p 20` runner, including 999 ms acceptance and
  exact-1 s zero-effect refusal (ARCH-PURE, ARCH-DRY, ARCH-CONSTRAINTS).
- 2026-08-30: implemented Chunk 3's resume-authority boundary. Pure eligibility
  now refuses every occupied, unverified, path/profile-invalid, unsupported, or
  non-established state with stable diagnostics and returns defensive exact
  address/path/profile/native-ID values only for a verified park. The production
  resolver projects #155's bounded `QuerySession`; Pair's trusted profile carries
  saved/saved required-resume authority and rechecks the exact native root after
  address claim but before defaults, ledger, sidecars, attach, or child launch.
  Focused and full `couchcore`/`launcher` suites passed with one
  `go test -p 20` process at a time (ARCH-PURE, ARCH-PURPOSE, ARCH-DRY).
- 2026-08-30: completed exact same-address resume. New-thread Spawn and Resume
  now share one tracked blocked-helper/acknowledgement/registration/promotion
  tail; Resume atomically admits the existing verified-park record with its
  start authority, rechecks the saved native root before child effects, and
  uses a read-only exact established Pair address registration. Rollback
  restores verified park only after both helper death and exact Pair-session
  absence; ambiguous acknowledgement, missing registration evidence, and
  promotion conflicts remain occupied/unknown. Exact-profile and failure
  matrices plus full `couchcore`/`launcher` suites passed with one
  `go test -p 20` process (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).
- 2026-08-30: wired Park/Resume into the shared operation schema with no Couch
  detach surface. CLI references resolve uniquely inside the current repo;
  owner-local panel actions carry an implicit exact address. Couch derives both
  Alt+x encodings from Pair's canonical chord table, frames them across reads
  and bracketed paste, renders confirmation before lifecycle work, then uses a
  bounded single-worker queue with duplicate coalescing and immediate overload
  refusal. Child exit changes only console routing and never durable park state.
  Focused operation/interceptor/panel suites passed (ARCH-DRY,
  ARCH-CONSTRAINTS, ARCH-PURPOSE).
- 2026-08-30: added a reusable redacted lifecycle conformance trace spanning
  prepared/committed request and completion boundaries, process restart,
  duplicate delivery, and ordered idempotent cleanup. The opt-in real adapter
  reuses the controlled Zellij fixture and matches the fake's cleanup trace;
  both live lifecycle and existing session-quiescence probes passed. Stable
  lock-holder death after rename remains covered by the pairlifecycle
  subprocess conformance test (ARCH-MOCK, ARCH-DRY).
- 2026-08-30: measured 100 sequential real-ThreadStore queue/commit samples on
  the operator's M2 Max under development load: `feedback_p95=11.667µs`,
  `commit_p95=27.009417ms`, `commit_max=27.988458ms`, `overload=refused`.
  Updated README, Couch/session-identity/architecture atlas maps, and the sole
  referencing project (`workshop/projects/couch.md`). The original Plan's
  “process inventory” wording is historical: the delivered implementation uses
  Pair's shared typed cleanup/completion authority and #155 exact binding,
  consistent with the later authoritative revision (ARCH-CONSTRAINTS,
  ARCH-DRY, ARCH-PURPOSE).
- 2026-08-30: the final verification sweep reproduced the previously parked
  review-toggle shell failure. Root cause was a stale fake: production had
  changed to the bounded scalar `session-inventory --owner <tag>` contract,
  while the fake still returned the retired JSON inventory. Updated only the
  fake (`49e1f1d`); all review assertions now pass without restoring any live
  Codex fallback.
- 2026-08-30: `sdlc actual --issue 152` measured 9.16h for the claim-to-HEAD
  window. All automated close evidence now passes; the remaining plan item is
  the operator's interactive `pair-dev` Couch Alt+x → same-tag/native-root
  resume smoke.
- 2026-08-30: manual Couch smoke verified immediate confirmation and successful
  non-root Park, then exposed two UI boundary defects. Parking the last/root
  actor returned to the shell because `onExit` ended the console before the
  async result could render; a parked row correctly remained durable but its
  `·` marker did not explain that Enter resumes it. Revised the durable plan
  to keep an already-focused empty panel alive with an Escape exit and to name
  `live`/`parked` states explicitly (ARCH-PURPOSE, ARCH-CONSTRAINTS).
- 2026-08-30: added red regressions for all three smoke findings, then changed
  only the console ownership boundary and pure panel renderer. A panel-focused
  final child exit now keeps Couch alive for Resume, Escape with no actor stops
  the console, and rows render explicit `[live]`/`[parked]` states. The full
  `couchtty` package passes with one `go test -p 20` runner.
- 2026-08-30: the follow-up smoke found panel Alt+x inert after parking one of
  multiple actors. Root cause: `onExit` promoted `root` but cleared `active`,
  while panel Park intentionally targets `active`. Added red coverage for both
  root and non-root active exits, then made the remaining root the active
  fallback. Focused regressions pass (ARCH-PURE, ARCH-PURPOSE).
- 2026-08-30: deeper live inspection proved the apparent parked actors were
  still running: Couch `99657` retained two `pair resume` children plus their
  session-watch/title/Zellij descendants, and the isolated ThreadStore still
  reported both incarnations live. Root cause is a circular trigger:
  `TriggerQuit` writes an intent that Pair consumes only after its blocking
  Zellij handoff returns, while Couch intercepts the Alt+x that would end the
  handoff. `runActiveAttempt` then performs one completion observation and
  returns nil when the exact session remains, which the UI mislabels
  `park: done`. Revised the durable plan: actual trigger + bounded completion
  and child-death proof, root Alt+x as verified park-all/Leave Couch, and
  inactive rows as ordinary historical inventory (ARCH-PURPOSE, ARCH-MOCK,
  ARCH-CONSTRAINTS).
- 2026-08-30: close-gate round 2 exposed the remaining classes behind BR-1 and
  BR-2. `New` now performs no active-park session query; a barrier test blocks
  `PairSession` and proves only the context-bound serial recovery worker enters
  it. Live conformance records the exact helper process blocked in Pair's
  production Zellij attach handoff; production `TriggerQuit` releases it, the
  helper consumes the typed intent through production cleanup/completion, and
  the driver only observes committed evidence and exact death. An intent-only
  mutation remains blocked and fails, proving causality (ARCH-PURE, ARCH-MOCK,
  ARCH-PURPOSE, ARCH-CONSTRAINTS).
- 2026-08-30: the operator's post-Leave smoke exposed a host-terminal mode leak:
  cooked termios returned, but child-enabled any-event/SGR mouse reporting did
  not, so pointer movement typed escapes into the shell. Couch's single release
  path now revokes mouse encodings/tracking, focus events, bracketed paste,
  synchronized output, and extended keyboard reporting before restoring the
  shell. A regression enables mouse mode through the child stream and proves
  the reset follows it on teardown (ARCH-DRY, ARCH-PURPOSE,
  ARCH-CONSTRAINTS).
- 2026-08-30: final operator acceptance used isolated
  `COUCH_STORE_DIR=$(mktemp -d) couch start .`, avoiding all historical Couch
  rows. Leave completed, the returned shell ignored mouse movement (no encoded
  bytes), and Resume preserved the exact Pair identity. The two fresh durable
  observations were tag `couch-e9310be25fed63b0` with native Claude root
  `b7fe7062-b8d8-4433-b436-19ca2ba58d68`, then tag
  `couch-69628ced671e30e0` with native Claude root
  `c2c52d0e-9980-4c87-ba90-8c0efd6e1344`; each ThreadStore ended in one
  verified Park and its tag ledger retained that same required root across the
  smoke (ARCH-PURPOSE, ARCH-CONSTRAINTS).

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec                design=1.50 impl=0.12
item: greenfield-go-module      design=0.40 impl=0.32
item: greenfield-go-module      design=0.40 impl=0.32
item: greenfield-go-module      design=0.40 impl=0.32
item: greenfield-go-module      design=0.40 impl=0.32
item: greenfield-go-module      design=0.40 impl=0.32
item: smaller-go-module         design=0.06 impl=0.20
item: smaller-go-module         design=0.06 impl=0.20
item: smaller-go-module         design=0.06 impl=0.20
item: smaller-go-module         design=0.06 impl=0.20
item: smaller-go-module         design=0.06 impl=0.20
item: tui-screen                design=0.40 impl=0.40
item: cross-cutting-refactor    design=0.20 impl=0.20
item: atlas-docs                design=0.20 impl=0.08
item: milestone-review          design=0.20 impl=0.20
item: scope-pivot               design=0.50 impl=0.20
design-buffer: 0.15
total: 9.90
```

The five greenfield units are the lifecycle protocol/store, pure park
transaction/coordinator, stateful fake and conformance driver, exact resume
boundary, and bounded asynchronous operation path. The five smaller extensions
are artifact paths, threadrecord v2, typed launcher cleanup, quit-intent
compatibility, and operation/shortcut declarations. The TUI item covers
event-loop interception and latency measurement; the cross-cutting item covers
extracting one cleanup and one existing-address launch path without duplicating
them. One scope-pivot primitive records the operator-driven correction from an
independent Couch teardown to Pair Alt+x authority.

The plan pre-resolves the implementation decisions, so module design uses the
v2.1 thorough-spec discount and the 15% design buffer. Familiarity is 1.0 for
this established Go repository and its existing ThreadStore/launcher patterns.
The library check found the required primitives already in the current stack
(standard-library file operations and `golang.org/x/sys/unix` advisory locks),
so no novel-stack discovery item is needed. The resolved calibration document
was marked stale by `sdlc estimate-source`; these values are therefore
provisional rather than memory-derived.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*
