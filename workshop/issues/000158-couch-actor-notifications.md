---
id: 000158
status: working
deps: [151]
github_issue:
created: 2026-08-31
updated: 2026-08-31
estimate_hours:
started: 2026-08-31T20:07:23-07:00
---

# couch: actor notifications and attention routing

## Problem

Couch already detects a terminal bell from an inactive actor, marks that actor
in its reserved status row, and shows a bell on the thread row. The signal has
no message, however, and the hierarchical switcher cannot explain why an actor
wants attention. Pair already interprets agent-specific output and emits
actionable OSC notifications to its outer TTY, but Couch currently treats those
bytes only as terminal output. An operator can therefore receive an outer
terminal badge without getting a Couch-local inbox or a one-keystroke route to
the responsible actor.

## Spec

### One interpretation boundary

Pair remains the only agent-specific notification interpreter. Its wrapper
recognizes native Claude, Codex, and other harness signals once and emits a
normalized Pair-owned OSC notification envelope to Pair's recorded outer TTY.
Couch does not repeat Pair's native OSC filter or learn agent-specific patterns;
it observes only the normalized envelope on each actor's existing PTY stream
(`ARCH-DRY`, `ARCH-PURE`). The initial canonical envelope is the existing
`OSC 777;notify;pair;<message>` form supported by `pair-notify`. Pair's native,
marker, idle, and hook-driven notification paths converge on the same encoder.

The byte path is singular. When Pair runs under Couch, Pair's recorded outer TTY
is the slave side of the actor PTY created by Couch; Pair's direct write bypasses
its inner Zellij but arrives at Couch's actor-PTY master. When Pair runs directly
under a terminal, the same write arrives at that terminal. Pair emits no second
copy on its ordinary stdout. For a native actionable OSC, `pair-wrap` suppresses
that recognized native notification from the inner Zellij stream and replaces
it with the one normalized outer-TTY envelope. Unknown and non-actionable OSC
continues transparently through the inner stream. Byte-faithfulness below means
the normalized envelope and all unrecognized terminal output—not preservation
of an agent-specific notification that Pair intentionally normalized.

The envelope remains an ordinary standards-compatible terminal notification.
Couch observes it as a tee and never swallows, rewrites, terminates, or
duplicates it. A focused actor's normal raw-output path forwards the bytes
exactly once. Inactive actor output remains hidden, but Couch forwards the exact
notification sequence to its own outer terminal while retaining the decoded
message. Thus cmux, terminal badges, and future outer wrappers continue to see
the signal regardless of which actor is focused. Unknown, unsupported, or
malformed OSC follows Couch's ordinary child-output rule—byte-faithful when the
actor is focused and hidden with the rest of an inactive actor's output—and
creates no Couch state.

The outer terminal is still one serialized byte stream. If its displayed actor
is mid-sequence, a complete notification from another actor cannot be inserted
safely. Couch defers that envelope until the focused stream reaches a real
boundary or a takeover resets it. The child-to-Console handoff permits at most
one unacknowledged batch per actor: Couch withholds that batch's acknowledgement,
which backpressures only its source PTY while the focused actor and other panes
continue. A safe boundary forwards the exact envelope, acknowledges the batch,
and permits that actor's pump to read again. Memory is bounded by one PTY batch
per attached actor without swallowing, coalescing, or evicting outer-terminal
events. Deferred envelopes flush in arrival-sequence order before later
Couch-owned paint bytes.

### Ephemeral per-actor inbox

Couch attributes a normalized event from the stream to that exact live actor
and thread. A focused actor's event is considered consumed immediately: it is
forwarded outward but creates no highlight or inbox row. An inactive actor
retains at most its three newest distinct messages. Repeated identical messages
collapse in place to the newest occurrence rather than adding rows. Each
accepted event receives a Console-local monotonic unread sequence so Couch can
identify the source of the newest unread notification without consulting disk
or wall time.

Pair's encoder first decodes UTF-8 with invalid input replaced, removes every
Unicode C0, DEL, and C1 control (`U+0000–U+001F`, `U+007F–U+009F`) that could
terminate or inject a 7-bit or 8-bit terminal sequence, then bounds the sanitized
message to 4 KiB without splitting a UTF-8 encoding. The limit therefore applies
after sanitization and always produces valid UTF-8. Couch applies the same
shared sanitizer defensively and uses exact equality of that sanitized,
unclipped string for deduplication. Unicode is not otherwise normalized; width
clipping happens only while rendering and never changes inbox identity.
Repeated equality moves the retained message to newest position and assigns the
new unread sequence.

Notification state is deliberately ephemeral. It is never written to thread
records, session inventory, or resume metadata. Actor exit, successful park,
or Couch exit discards that actor's messages. Restarting Couch does not restore
attention from historical OSC. This is live presentation state, not durable
thread evidence (`ARCH-PURPOSE`).

### Status row and switcher

The reserved status row keeps its current stable actor roster and spacing. An
inactive actor with unread messages is indicated only by attention color on its
existing label—no star, dot, count, or added status token. The focused actor
keeps its existing bracket treatment. Ordinary Couch operation notices may
still follow the roster after the existing separator; notification bodies do
not replace that notice area.

`Ctrl-Space` opens the switcher without reordering actors or threads. When one
or more inactive actors have unread messages, the root list initially selects
the actor whose newest retained message has the greatest unread sequence. Enter
is then one action away from switching to that actor. Every retained message is
rendered as its own indented, display-only row directly beneath the source actor
row. Up/Down navigate actor rows, not message rows. Existing filter, stable
identity, action-menu, and bounded-width rules continue to apply; notification
children never become independent work-thread identities.

Filtering continues to match the existing actor/thread row fields only. A
notification is displayed when its parent actor row is present; notification
text does not independently admit an otherwise filtered-out actor.

Merely opening the switcher, selecting an actor, or attempting a switch does
not acknowledge attention. One pure `AttentionLedger`, owned under the Console
mutex, is the sole authority for retained messages and unread sequences. Status
rendering and switcher rendering receive projections from that ledger; the menu
does not keep a parallel bell/inbox map. On switch dispatch, Console captures
the target actor and its greatest current unread sequence in the existing
operation origin. The ledger registers that attempt's exact current retained
message identities. The existing successful `forceSwitch` completion is the
only acknowledgement event: Console asks the ledger to clear exactly those
captured identities, then repaints both consumers. Dispatch failure or focus
failure cancels the capture without clearing. A message accepted or
deduplicated after dispatch therefore survives. Because the ledger owns both
retained sequences and pending captures, overflow rebase remaps them atomically
(`ARCH-PURE`).

### Operating envelope and failure behavior

Notification parsing and state transition run incrementally on the existing
child-output path. They perform no filesystem, transcript, subprocess, or
network work and start no per-notification goroutine. State is bounded by three
messages per attached actor; repeated events cannot grow it without limit. The
framer reuses the existing 64 KiB partial-terminal-sequence bound. A canonical
message is at most 4 KiB, so any candidate exceeding 64 KiB is invalid for Couch
enrichment: the scanner stops buffering, continues framing in O(1) memory until
BEL or ST, creates no inbox entry, and then resumes ordinary parsing. An
unterminated OSC has no protocol-safe recovery boundary: a later opener is
payload until the original sequence terminates. Enrichment for that actor is
therefore intentionally suspended until BEL/ST or actor teardown rather than
guessing and raising a false notification. The child read loop and raw forwarding
never wait for enrichment, memory remains bounded, and other actors' independent
scanners continue normally. Focused bytes remain on the raw pass-through path;
inactive invalid candidates remain hidden like other inactive output.
Status-row repaint and `Ctrl-Space` selection use already-resident state and
must not block on refresh or optional inventory work. The feature inherits
#151's measured switcher envelope on the operator's M2 Max: with 100 actor rows
at 120x40, opening produces its first frame within 50 ms p95 and selection,
navigation, and render computation each complete within 16 ms p95 after 20
warmups and across 200 samples. The fixture gives every actor three retained
messages, so the bound covers 100 actors and 300 notification children. Portable
tests retain allocation and no-I/O/no-goroutine assertions; target runs use the
same baseline and four-worker co-tenancy protocol as `BenchmarkMenu100`
(`ARCH-CONSTRAINTS`).

To preserve a canonical notification across a Couch screen takeover, the child
observer may withhold only a byte prefix that still exactly matches the
canonical `OSC 777;notify;pair;...` envelope. A mismatch releases the candidate
bytes immediately as ordinary output. A complete valid candidate is released
as one ordered notification part, so a switch between PTY reads cannot splice
Couch repaint bytes into the OSC. Candidate storage is bounded by the canonical
header, 4 KiB sanitized message, and terminator; exceeding that exact encoded
bound releases the buffered bytes and enters transparent passthrough through
the real terminator with no enrichment. An unterminated in-bound canonical
candidate remains withheld until terminator or actor teardown, while subsequent
non-candidate reads, the child loop, other actors, and Couch UI remain live.
This is the sole exception to immediate focused raw forwarding; it delays only
a possible Pair-owned envelope, normally until the next PTY read, in exchange
for exact-once terminal framing across takeover (`ARCH-PURPOSE`).

Each child output batch carries an absolute ring end and a replay-safe end. The
Console advances a pane's replay cutoff only after processing that batch; bytes
still withheld as a canonical candidate are not replay-safe. Switching replays
only through that cutoff and removes completed Pair notification envelopes, so
replay cannot expose a partial candidate, overtake a queued batch, or re-notify
an earlier event.

Raw byte parts use focus at Console processing time, not delivery time. Thus a
queued batch from the actor switched into is painted rather than lost, while a
queued batch from the actor switched away from stays hidden and becomes visible
through that actor's later replay. The delivery-time focus stamp is used only
to decide whether a completed notification was immediately consumed.

The ring records absolute spans for completed canonical notifications. Replay
removes intersecting spans, including when retention's oldest byte bisects an
envelope; a retained suffix can therefore never replay a body tail or
terminator as terminal input.

The stream scanner additionally has a sustained malformed-input envelope: on
the M2 Max it processes 10 MiB of 4 KiB chunks through an oversized unterminated
OSC at at least 10 MiB/s, retains no more than the existing 64 KiB bound, starts
no goroutine, and performs no allocation per chunk after entering skip mode.
Portable tests enforce the memory, allocation, independent-actor, and
terminator-recovery invariants; the target benchmark records throughput rather
than putting wall-clock assertions in ordinary CI.

Message text is untrusted terminal input. Framing must be invariant across
arbitrary PTY chunk boundaries and BEL/ST terminators, and remain memory-bounded
for long or incomplete OSC, embedded delimiter text, and control-byte attempts.
Stored/rendered text is sanitized and clipped by the existing width-aware
renderer, while the original envelope is forwarded byte-for-byte. Parser
failure affects only Couch enrichment: it must never stall or corrupt the child
read/forwarding path.

Unread ordering uses a process-lifetime `uint64` sequence. If increment would
wrap, the pure ledger rebases retained actor/message sequences to their stable
relative ranks before accepting the event; ordering remains total rather than
silently wrapping newest attention behind older attention.

### Verification strategy

Pure framing and inbox reducers cover whole/split envelopes, normalization,
deduplication, three-message eviction, unread ordering, focus-time consumption,
and sequence-qualified clearing. Console integration uses the existing stateful
PTY child/host doubles to prove inactive forwarding, focused non-duplication,
status coloring, nested rows, `Ctrl-Space` selection, failed-switch retention,
and arrival-during-switch preservation (`ARCH-MOCK`). A real-PTY conformance
case runs Pair's normalized emitter through Couch and verifies both the outer
OSC bytes and Couch-local attention state.

## Done when

- Pair's agent-specific paths converge on one normalized Pair-owned OSC
  notification envelope, and Couch contains no duplicate native-agent filter.
- Focused and inactive actor notifications reach the outer terminal exactly
  once; malformed or unsupported OSC remains byte-faithful and creates no
  inbox entry.
- The status row colors only actor labels with unread attention, while the
  switcher shows at most three distinct indented messages per actor.
- `Ctrl-Space` preserves actor order and selects the source of the newest unread
  message; only a successful switch clears the captured unread generation.
- Notification state remains bounded, in-memory, and absent after actor
  exit/park or Couch restart.
- Pure, stateful integration, real-PTY conformance, race, and bounded 100-row
  performance evidence cover the complete flow.

## Plan

- [ ] Write and approve the implementation plan.
- [ ] Normalize Pair notification emission and add chunk-safe Couch observation.
- [ ] Add bounded per-actor inbox state plus status/switcher presentation.
- [ ] Verify byte-faithful forwarding, acknowledgement races, and operating
      bounds through pure, integration, conformance, and performance tests.

## Log

### 2026-08-31

Operator selected an in-band terminal contract over a new IPC channel or a
Pair-as-library refactor. Pair filters agent output once; Couch observes the
normalized event on the actor PTY as a byte-faithful tee. The UI keeps stable
actor order, colors the existing status-row label, nests up to three messages
under the actor in the switcher, and makes `Ctrl-Space` + Enter the fast path to
the newest unread source.

## Revisions

### 2026-08-31T20:22:00-07:00 — close transport, framing, and acknowledgement ownership

**Reason:** fresh-context spec review found that the first draft did not state
how Pair's outer-TTY write reaches Couch, whether native and normalized OSC could
duplicate, how incomplete frames remain memory-bounded, or which authority
performs sequence-qualified clearing.

**Delta:** the spec now gives one exact PTY byte path and replacement rule for
recognized native OSC; caps canonical messages at 4 KiB and reuses the 64 KiB
O(1) skip-through-terminator rule; makes one Console-owned pure attention ledger
feed status and switcher consumers; defines dispatch-to-success acknowledgement,
sanitized-string equality, filter behavior, overflow rebasing, and #151's exact
100-actor latency protocol (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`,
`ARCH-CONSTRAINTS`).

### 2026-08-31T20:28:00-07:00 — close the complete terminal-control class

**Reason:** the second fresh-context review found that removing C0/DEL alone
left C1 controls such as 8-bit ST able to terminate or inject into the envelope.

**Delta:** the canonical encoder and defensive Couch decoder now share one rule:
replace invalid UTF-8, remove all C0/DEL/C1 code points, and apply the 4 KiB bound
after sanitization on a rune boundary. Framing tests cover every control class
and chunk boundary (`ARCH-DRY`, `ARCH-PURPOSE`).

### 2026-08-31T20:32:00-07:00 — define incomplete-OSC recovery honestly

**Reason:** the third fresh-context review found that a bounded skip mode still
cannot resume enrichment after an unterminated OSC because the terminal protocol
provides no safe boundary.

**Delta:** the spec now states that enrichment for only that actor waits for a
real BEL/ST or teardown while raw forwarding, the child loop, and other actors
continue. It forbids guessed resynchronization, distinguishes stream liveness
from semantic enrichment, and adds a 10 MiB sustained skip-mode benchmark with
memory/allocation bounds (`ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

### 2026-08-31T22:11:42-07:00 — make canonical framing atomic across takeover

**Reason:** implementation-plan review proved that immediate forwarding of a
canonical OSC prefix and byte-faithful Couch screen takeover are mutually
exclusive when the PTY splits the envelope across reads.

**Delta:** the observer now withholds only bytes that still match Pair's exact
canonical envelope, within its encoded bound, and releases the complete
notification as one ordered part. Prefix mismatch or overrun returns to
transparent passthrough immediately; all unrelated output remains live. The
operator approved this narrow exception to the raw-path rule. The same revision
adds processed replay cutoffs, one unacknowledged batch of per-actor
backpressure for unsafe cross-actor forwarding, absolute notification spans at
ring-retention boundaries, and ledger-owned acknowledgement captures so
overflow rebase cannot invalidate an in-flight switch
(`ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).
