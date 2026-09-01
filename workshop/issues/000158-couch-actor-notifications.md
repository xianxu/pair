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
normalized Pair-owned OSC notification envelope to the recorded outer TTY.
Couch does not repeat Pair's native OSC filter or learn agent-specific patterns;
it observes only the normalized envelope on each actor's existing PTY stream
(`ARCH-DRY`, `ARCH-PURE`). The initial canonical envelope is the existing
`OSC 777;notify;pair;<message>` form supported by `pair-notify`. Pair's native,
marker, idle, and hook-driven notification paths converge on the same encoder.

The envelope remains an ordinary standards-compatible terminal notification.
Couch observes it as a tee and never swallows, rewrites, terminates, or
duplicates it. A focused actor's normal raw-output path forwards the bytes
exactly once. Inactive actor output remains hidden, but Couch forwards the exact
notification sequence to its own outer terminal while retaining the decoded
message. Thus cmux, terminal badges, and future outer wrappers continue to see
the signal regardless of which actor is focused. Unknown, unsupported, or
malformed OSC remains byte-faithful terminal output and creates no Couch state.

### Ephemeral per-actor inbox

Couch attributes a normalized event from the stream to that exact live actor
and thread. A focused actor's event is considered consumed immediately: it is
forwarded outward but creates no highlight or inbox row. An inactive actor
retains at most its three newest distinct messages. Repeated identical messages
collapse in place to the newest occurrence rather than adding rows. Each
accepted event receives a Console-local monotonic unread sequence so Couch can
identify the source of the newest unread notification without consulting disk
or wall time.

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

Merely opening the switcher, selecting an actor, or attempting a switch does
not acknowledge attention. Switch dispatch captures the actor's current unread
sequence. Only successful focus transfer clears messages at or before that
captured sequence. A failed switch clears nothing, and a notification arriving
during the switch remains unread. After successful clearing, a later event
highlights the actor again.

### Operating envelope and failure behavior

Notification parsing and state transition run incrementally on the existing
child-output path. They perform no filesystem, transcript, subprocess, or
network work and start no per-notification goroutine. State is bounded by three
messages per attached actor; repeated events cannot grow it without limit.
Status-row repaint and `Ctrl-Space` selection use already-resident state and
must not block on refresh or optional inventory work. Representative tests use
up to 100 actor rows and require the keystroke-to-render path to remain inside
the existing interactive switcher budget (`ARCH-CONSTRAINTS`).

Message text is untrusted terminal input. Framing must survive arbitrary PTY
chunk boundaries, BEL and ST terminators, long or incomplete OSC, embedded
delimiter text, and control-byte attempts. Stored/rendered text is sanitized
and clipped by the existing width-aware renderer, while the original envelope
is forwarded byte-for-byte. Parser failure affects only Couch enrichment: it
must never stall or corrupt the actor stream.

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
