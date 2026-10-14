# Hub-owned desktop notifications Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Swallow slot desktop notifications and let only the live Couch hub deliberately notify the operator.

**Architecture:** Keep child OSC interpretation in the existing VT/terminal endpoint and suppression in Console's effect policy. Route the new hub-only broker request to an application-owned presenter operation using the existing serialized host writer. No raw-byte filtering layer, new message queue, durable receipt, or automatic retry.

**Tech Stack:** Go; vendored VT parser; existing Couch broker Unix socket, terminal Presenter and diagnostic trace; fake terminal host and headless PTY/Zellij conformance fixtures.

---

## Approved scope

TL approved design d9c0c376 in receipt 30983ef5-6aa1-4752-9d6d-cfcd95ebf510.
All slots, including hub, lose OSC 9/777/99 desktop forwarding and notification
content in attention. Ordinary BEL retains its current behavior. Only the hub
may use `couch --notify [--title T] MESSAGE`; default title is `Couch`. Non-hub
refusal advises sending to the hub. TL owns coordinating ariadne#350's prose.

This is one atomic issue with one close review, not milestone boundaries.

## Core concepts

### Pure entities

| Name | Lives in | Status | Classification |
|------|----------|--------|----------------|
| Notification envelope / title sanitization | cmd/internal/notifyosc/notification.go | modified | PURE |
| Notify CLI invocation | cmd/internal/couchcmd/cli.go; notify.go | modified/new | PURE |
| Notify request shape | cmd/internal/couchmessage/protocol.go | modified | PURE |
| SuppressedNotificationEffect | cmd/internal/terminal/endpoint.go | new | PURE data |

- Envelope: one request yields one OSC 777 envelope. Add a general titled encoder
  beside existing Pair-specific Encode; preserve existing wrapper encoding.
  Both use Sanitize. Replace title semicolons with commas so body boundaries
  cannot change. Keep body semicolons. The new envelope uses ST termination.
- Request: add an explicit Title field, allowed only for notify. Reuse Body for
  message and existing scope/tag/session/nonce for caller identity. Notify takes
  no target, binding, ID, agent, confirmation, tail or operation overrides.
  Validate closed shapes server-side; other ops reject a nonempty Title.
- SuppressedNotificationEffect: a countable, non-forwardable effect with no
  body/title. Register OSC 99 at the existing endpoint parser seam, consume each
  complete bounded packet once, and emit this effect. Presenter acknowledges its
  sequence but never writes it, regardless of policy. This does not add Kitty
  support or multipart assembly to standalone Pair.

### Integration points

| Name | Lives in | Status | Wraps / classification |
|------|----------|--------|------------------------|
| Endpoint notification adapter | cmd/internal/terminal/endpoint.go | modified | VT parser, INTEGRATION |
| Presenter.Notify | cmd/internal/terminal/presenter.go | new | serialized ttyio.Writer, INTEGRATION |
| Console child suppression/counts | cmd/internal/couchtty/console.go; trace.go | modified | pane lifecycle and bounded trace, INTEGRATION |
| Console.NotifyDesktop | cmd/internal/couchtty/desktop_notify.go | new | presenter lifecycle, INTEGRATION |
| Notify CLI exchange | cmd/internal/couchcmd/notify.go | new | broker transport, INTEGRATION |
| Hub admission and notify wiring | cmd/internal/couchcmd/notify_service.go; message_service.go | new/modified | live registry, workspace authority, console, INTEGRATION |

Existing reusable fakes: ttyio fake writer records bytes, ordered writes and
injected write outcomes; couchtty console fixture owns panes/focus; message
service fixture owns registered caller bindings and connected workspace state.
Use their actual state transitions rather than asserting mocked call counts.
Real broker socket tests live in private temp directories. Headless PTY/Zellij
fixtures provide parser-to-parent conformance; OS desktop display requires the
operator's eventual smoke, not an automated notification sent to their terminal.

## Operating envelope, trust and ownership

- Light command: one existing broker exchange, no subprocess/git/tunnel, no
  retry. Inherit AdmissionTimeout/TransportTimeout and presenter cancellation.
  Test that valid admission does not invoke fleet/Git discovery.
- Bound raw title to 256 bytes and raw body to 4096 bytes (existing notification
  body budget); refuse oversize, then use Sanitize and UTF-8-safe output bounds.
  Reject blank message after sanitization and an explicitly blank title. Default
  title applies only when omitted. Decoder's existing frame bound caps hostile
  requests before parsing. Invalid UTF-8 becomes replacement text consistently.
- Slot output is untrusted. Reuse the chunk-safe bounded parser for ESC/C1,
  BEL/ST, cancellation and overflow; no regex/string slicing across read chunks.
  Existing notification effects cover OSC 9 and valid 777 notify envelopes.
  Malformed/oversized events are discarded; diagnostic counts count accepted
  complete events, not arbitrary prefixes or logical Kitty multipart messages.
- Server authenticates via existing current registered caller checks and cached
  connectedWorkspace/isHub authority. An environment variable or payload saying
  hub is insufficient. The socket's owner-only local-user boundary remains;
  this is coordination authority, not isolation from malicious same-user code.
- Keep one uint64 counter on each console pane; it expires with that pane and
  resets on a fresh attachment. Saturate instead of wrapping. After sequence
  deduplication, add countable effects and write one batch aggregate containing
  address and cumulative count through existing COUCH_TRACE diagnostics.
  Reattach resets are distinguishable using pane/endpoint identity in metadata.
  No payload storage, log family or growing slot map. Trace retention and lossy
  overload behavior stay owned by diagnosticlog; counts never paint the screen.
- Non-notification OSC behavior remains governed by the existing presenter and
  frame policy; this does not widen clipboard/title forwarding from hidden panes.

## Ordering and outcomes

| State/event | Result/effect | What the caller sees / recovery |
|-------------|---------------|--------------------------------|
| Invalid payload, missing/stale caller, non-hub | Refuse before presenter | Reason; correct arguments or send to hub |
| Console unavailable before submission | Refuse without write | Couch unavailable; use live hub |
| Valid admitted hub request | One presenter call in serialized writer order | Await that bounded call |
| Context cancelled before write is known to start | No new write | Failure, no automatic retry |
| Write returns success | Finished, no replay state | Written to terminal; OS display not guaranteed |
| Host write failure/partial write or missing broker reply | Unconfirmed | Explicitly do not retry automatically |
| Repaint, focus, reconnect after completion | No further notify effect | Nothing replayed |
| A second deliberate CLI call | Independent operation | Can intentionally produce another notification |

No new asynchronous state machine or queue: request stack lifetime and the
presenter's existing call/write state own progress. Do not reinterpret a timeout
as proof of no output. Validate hub authority immediately before submitting the
effect; tests disconnect or replace the binding before this point. No deferred
request may later execute after cancellation without being reported uncertain.
Child effects keep endpoint sequence identity; duplicate batches must neither
emit nor increment again. Counts consume returned acknowledged effects, including
any successfully consumed prefix when a later effect in the batch fails.

## Tasks

### 1. Terminal notification contract

Files: notifyosc/notification.go and notification_test.go;
terminal/endpoint.go, notification_mapping_test.go, presenter.go and new
terminal/desktop_notify_test.go (all under cmd/internal).

- [x] Add failing tests for titled envelope (default, semicolon, control injection,
  Unicode boundaries), non-forwardable OSC 99 packets split at every byte,
  and Presenter.Notify one write/no replay/failed or cancelled writer.
- [x] Run `go test ./cmd/internal/notifyosc ./cmd/internal/terminal -count=1`;
  capture failures proving missing behavior before implementation.
- [x] Add titled encoder using Sanitize; add count-only effect and OSC99 handler;
  presenter accepts that effect without output. Add Notify using p.call and
  writeUnlessStalled, analogous to Copy, never direct host writes or fake origins.
- [x] Rerun those packages green; commit `#540: add explicit desktop notification output`.

### 2. Swallow slot effects and count them

Files: couchtty/console.go, trace.go, desktop_notify.go and desktop_notify_test.go;
notification_test.go, effect_policy_test.go, notification_pty_test.go,
terminal_native_test.go (under cmd/internal).

- [x] Add failing assertions for zero host notification envelopes from OSC9/777/99
  in focused/hidden/hub panes; no notification attention text; per-pane count,
  duplicate batches and ordinary BEL behavior. Trace must contain count/address
  and never the test's unique secret title/body.
- [x] Run `go test ./cmd/internal/couchtty -run 'Notification|EffectPolicy' -count=1`
  and confirm behavior failures.
- [x] Disable Notifications in onChunk effect policy; replace notification
  attention marking with the counter/trace logic. Keep Bell branch and other
  effect policies. Add console's application-owned NotifyDesktop delegation.
- [x] Update existing forwarding assumptions in PTY/native tests, preserving
  their real input/focus/reattach/stress coverage rather than deleting assertions.
- [x] Run package tests; verify OSC2 titles, OSC52 clipboard and OSC8 links still
  work, plus Unicode and non-notification output. Commit suppression changes.

### 3. CLI, closed protocol and hub admission

Files: couchcmd/cli.go, notify.go, notify_service.go, messages.go,
message_service.go, cost.go, notify_test.go, notify_service_test.go;
couchmessage/protocol.go, protocol_test.go (under cmd/internal).

- [x] Add failing CLI grammar tests: exactly one opaque message argument, optional
  leading --title value, no conflicting flags; a message that begins with a dash
  remains opaque at its expected position. Validate missing values/extra args.
- [x] Add request shape tests, including Title on every other operation, hostile
  controls/oversize bodies and blank sanitized values.
- [x] Add stateful service tests for live hub emits exactly once, non-hub advises
  send-to hub, missing hub, stale nonce/session, disconnected/replaced binding,
  missing console, write error and lost reply. Assert zero output for refusals.
- [x] Run `go test ./cmd/internal/couchmessage ./cmd/internal/couchcmd -run 'Notify' -count=1`
  and capture red results.
- [x] Implement parse/exchange/admission wiring. Reuse broker caller/current and
  connectedWorkspace/isHub; extract shared naming if needed instead of copying
  switch-agent authentication. Use a synchronous console callback installed with
  the existing service wiring pattern. Mark any ambiguous write/exchange result
  unconfirmed with no automatic retry. Plain success output claims only written.
- [x] Rerun affected packages green; commit command and admission.

### 4. Documentation and verification

Files: README.md; atlas/couch.md; atlas/terminal.md; atlas/index.md if adding a
page; cmd/internal/couchcmd/skills/couch/SKILL.md and help renderer;
cmd/internal/artifactpath/manifest.go; issue and this plan.

- [x] Register new production files in the exhaustive artifact-source inventory.
- [x] Document swallowed slot notifications/counts, hub-only command and examples,
  no-retry uncertainty, preserved BEL, and light cost in help/embedded skill.
  Update help/cost contract tests. TL supplies ariadne#350's prose.
- [x] Run `go test ./cmd/internal/notifyosc ./cmd/internal/terminal ./cmd/internal/couchmessage ./cmd/internal/couchtty ./cmd/internal/couchcmd ./cmd/internal/artifactpath -count=1` (covered by package runs and full suite).
- [x] Run focused race checks on notify service/presenter/console fixtures and
  headless `TestNotificationPTYConformance` and `TestNativeConsoleWrapperZellij`
  using their documented opt-in environment after reading each fixture guard.
  No host desktop notifications from tests; use captured fake host output.
- [x] Run full queued `make -k test` to completion on the stable final source
  tree; retain all failures, distinguish known flakes with evidence, rerun only
  affected checks when warranted. Do not bypass machine queue.
- [ ] Commit evidence and docs. Send TL real `git rev-parse HEAD` and smoke:
  child notifications silent, ordinary BEL unchanged; TL hub notify produces one
  desktop notification; worker refuses with hub advice. Stage only on TL request.
- [ ] After smoke acceptance/waiver, run `sdlc close --issue 540 --verified ...`,
  resolve mandatory boundary review, merge main and use normal PR/CI gates.
  On SHIP run `sdlc merge --yes`, report landing to hub, stop.

## Architectural accounting

- ARCH-DRY: existing parser, sanitizer, caller/workspace authority, presenter
  writer and bounded trace each retain one responsibility; no parallel transport.
- ARCH-PURE: encoder and request/CLI validation have colocated pure tests; IO
  admission/writing are classified separately above.
- ARCH-PURPOSE: all child origins and all three protocols swallowed; only hub
  deliberate output reaches host; old attention-forwarding tests updated too.
- ARCH-MOCK: existing stateful terminal/registry fixtures and private Unix sockets;
  headless conformance plus explicitly scheduled operator desktop smoke.
- ARCH-CONSTRAINTS: bounded text/frame sizes, one light synchronous exchange,
  existing deadlines/backpressure, one fixed counter per live pane.
- ARCH-SECURE: current live caller and server-owned hub root; terminal-safe
  bounded envelope; no body diagnostics or new credentials.
- ARCH-ORDER: sequence identity, serialized writer, cancellation and uncertain
  outcomes explicitly covered above; no resend on lost acknowledgment.
- ARCH-FUNERAL: counters die with panes; existing bounded trace owns retention;
  request data dies on return; no new durable artifact or background process.

## Revisions

### 2026-10-13 — mixed-version CLI/supervisor follow-up

Live notification smoke passed per TL. Before close, add a private socket
regression for current CLI send/actors/status against the strict pre-#540 request
contract (ccc0ee88), with registered stateful broker callers. The observed
unavailable response identifies missing caller registration, while protocol and
caller fields are unchanged. Diagnose that accurately; never infer supervisor
age from every unavailable response or retry a possibly admitted effect.
Record the evidence and recovery choice, then clear the accepted stage and
complete the normal close/landing gates. TL authorized this scope addition.

### 2026-10-13 — full-suite integration consumer follow-up

The completed full suite found an old wrapper-to-Couch integration expectation
in wrapcmd/codex_working_test.go, outside the consumer package's test inventory.
It waited for notification text that this design intentionally suppresses and
exhausted the package deadline. Update that test to prove suppression after a
positive processing/render barrier, then rerun the regression and full wrapcmd
package so the interrupted later tests also complete. All other full-suite
packages and prerequisites passed. This is test coverage within the approved
scope; production semantics and smoke gates are unchanged.
