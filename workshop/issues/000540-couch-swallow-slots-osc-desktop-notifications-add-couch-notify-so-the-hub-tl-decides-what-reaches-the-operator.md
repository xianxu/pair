---
id: 000540
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '5e1736cfaf2373a3e72496f9bc4ae74516af63a0' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T17:52:26-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "cafb2d9c", done: "c88d2fe0"}
---

# couch: swallow slots' OSC desktop notifications; add couch --notify so the hub (TL) decides what reaches the operator

## Problem

Agents in slots emit terminal desktop notifications (OSC 9 / OSC 777 / OSC 99 sequences, e.g. 'Claude needs your input'). With many slots under a hub they arrive at the wrong granularity (per agent turn, not per operator decision) and never worked well. The operator's attention is now managed by the TL in the hub (the tl-lead skill), which decides what's worth interrupting for.

## Spec

The operator (2026-10-13):
1. Couch swallows OSC notification sequences from every slot's terminal (the hub's included): they never reach the operator's terminal. Other OSC traffic (titles, hyperlinks, clipboard as Pair already handles it) is unaffected. Log a count per slot for diagnostics, nothing more.
2. A new verb `couch --notify [--title T] MESSAGE` (light) emits one OSC desktop notification to the operator's terminal through Couch's own output. It is meant for the hub (the TL); other callers are refused or downgraded to a hub message (decide in the plan, keep it simple).
3. Docs: `couch --help` and the couch skill say slots' notifications are swallowed and only the hub notifies; ariadne#350's tl-lead prose uses `couch --notify` for decisions the operator must see (smoke tests ready, a question, an incident).

## Done when

- A slot emitting OSC 9/777/99 produces no notification on the operator's terminal; `couch --notify` from the hub produces exactly one; tests cover both and that non-notification OSC passes through.

## Plan

- Durable plan: [hub notifications](../plans/000540-hub-notifications-plan.md).
- [x] Add terminal envelope and explicit presenter notification operation.
- [x] Suppress child notifications and retain diagnostic counts only.
- [x] Add hub-only CLI/protocol/admission with uncertain-write handling.
- [ ] Update docs, run full verification and scheduled smoke, close and land.

## Log

### 2026-10-13

- Claimed in pair:7 after #537 landed (PR #314). Refreshed the clean resting
  checkout with `weave refresh`: Pair 7f68b40238de8040e61d5fb39e3d3c09d64babb7,
  private Ariadne 86ad95c4eb409942630d1a3fc785339e9515e40c. Entered start-plan.
- Read-only exploration found existing terminal notification effects and a
  serialized presenter writer, plus live caller / connected workspace / hub
  authority already used by switch-agent. No implementation or live desktop
  notification performed during design.
- TL approved design d9c0c376 in verified receipt
  30983ef5-6aa1-4752-9d6d-cfcd95ebf510. Durable implementation plan authored via
  writing-plans, including pure/integration inventory, payload budgets,
  count lifecycle, failure projection and regression/conformance matrix.
  No code changed. TL owns ariadne#350 coordination.
- TL approved the durable plan aadffa9ec in verified receipt
  15d29993-9e03-4b2d-b011-84bdedfdd683. change-code passed, inferring quick from
  design size; implementation exceeds the shell and will require full close
  review. No gate bypass used.
- Task 1 b33504ffc: shared bounded desktop envelope, count-only OSC99 event,
  serialized Presenter.Notify. New tests failed before APIs existed, then full
  notifyosc and terminal packages passed (terminal 37.155s); focused race passed.
  Fresh spec and quality subagent reviews approved.
- CLI/protocol tests first failed on unknown notify operation. 4327d67f7 adds
  hub-only service admission, console delegation and one-shot CLI transport.
  Spec review found a wrapper replacement interleaving during workspace lookup:
  the deterministic test first returned Code:ok, then 7b2f99d85 rechecked current
  authority immediately before submission and passed. A real socket test drops
  acknowledgment after output and proves one envelope plus unconfirmed/no-retry
  advice. Targeted couchmessage/couchcmd race checks passed (1.160s/1.713s).
- First full couchmessage/couchcmd run found only the embedded-skill command
  example using optional brackets, which its executable examples parser rejects.
  Replaced with the two concrete valid forms; focused help/skill tests passed.
  Full package rerun underway. Artifact inventory package passed (17.485s).
- Full broker packages rerun passed: couchmessage 7.268s, couchcmd 49.024s.
  Admission spec re-review and quality review approved the final checks.
- Console dc428db5c passes all couchtty tests (64.179s), targeted race (4.702s),
  and private native direct/wrapped Zellij conformance (29.123s): 32 notifications,
  4096-byte UTF-8 body, hidden/focused panes, resizes, persistent agent reattach.
  Native oracle dependencies installed with locked npm ci. Native run preceded
  only the diagnostic cumulative-count adjustment, verified by targeted/full
  tests. First package run hit TestDriveActionShutdownJoinsAndRejectsClicks
  timeout; unchanged targeted rerun and final full package passed.
- Main merged at 2974107d0c45b5206facfaff6a37519820401412. Full `make -k test`
  started via weighted machine queue; evidence path /tmp/pair540-full-suite.log.
  TL reserved staging after #528's acceptance session, per verified receipt
  47ef690a-6165-499c-ad4d-30357d8e4b73; no pair:0 action until TL says clear.
- Console spec and quality reviews both approved dc428db5c; no remaining
  in-session review findings. The full-suite command remains queued, not failed
  or cancelled. Tests have not started while earlier Ariadne suites hold the
  machine gate. Implementation and targeted verification are complete; waiting
  for suite execution and TL's staging/smoke instruction.

- Full `make -k test` completed with exit 2 after the weighted queue admitted it.
  All other packages and prerequisites passed; wrapcmd's old
  TestCodexWorkingNotificationReachesCouchStatusAndSwitcher still expected
  swallowed notification text. Its 587.61s wait exhausted the 10-minute package
  deadline and interrupted later fixture tests. This is a missed test consumer,
  not an unrelated flake. Correcting the integration expectation, then running
  the focused regression and full wrapcmd package; full log remains
  /tmp/pair540-full-suite.log. No production change is indicated by this failure.

- Corrected wrapper-to-Couch integration now verifies canonical wrapper output,
  waits for the suppression trace and switcher render, then checks complete host
  history for leaked envelope/body. Focused regression passed (0.418s); complete
  `go test ./cmd/internal/wrapcmd -count=1` passed (120.249s), including the fixture
  tests interrupted in the full run. Evidence: /tmp/pair540-wrapcmd-corrected.log.
  Fresh-eyes review approved the correction. Sweep found no other stale
  wrapper-to-Couch forwarding expectations. Only tests/docs changed after the
  completed full suite; no product code changed. Waiting for TL's stage clearance.

- Readiness report attempted from eac0d99ab4cc985eb16b64326bd0c316a52fe512;
  Couch returned uncertain receipt 05d6ae41-759a-43c1-b6d7-6b91c4b85e93 because
  its broker socket was absent. The required message-status lookup also failed
  for the absent socket. No resend or alternate channel attempted; delivery is
  unconfirmed. Await restored Couch/TL stage instruction; pair:0 remains untouched.

- TL authorized staging in verified receipt aa226b2f-e50c-47cb-9102-362c05b73508
  after Couch restarted. Merged origin/main cleanly; `sdlc stage --build` placed
  18a2aa74330e6c753cf68bf5ff697cf3b4336cb5 on pair:0 and built matching couch,
  pair, pair-launch-helper and pair-slug. Existing unrelated pair-hoprtt reports
  an older revision. Target HEAD verified directly. Hub staging report receipt
  4bdd3756-c87e-4982-a37c-08e40ecf218d is submitted; desktop smoke still pending.
  Post-merge Notify/Notification/Skill/Inventory tests passed in couchcmd,
  couchmessage, couchtty and artifactpath (/tmp/pair540-postmerge.log).

## Revisions

### 2026-10-13 — acceptance and mixed-version messaging follow-up

TL receipt 9c269a35-f57f-4b22-b961-344cc16a0eb3 records live smoke PASSED:
hub notify produced the operator's alert, pair:7's exact notify command refused
with send-to-hub advice, and child notifications stayed silent. TL authorizes
stage clear, close and landing after the mixed-version follow-up.

TL observed the CLI built at 18a2aa74 receiving `unavailable: recipient is not
live` from the still-running ccc0ee88 supervisor on both send and actors;
restarting Couch recovered. Add mixed-version regression coverage and a truthful
recovery diagnostic within #540. Investigation finds unchanged caller identity,
socket derivation and session protocol; the added Title is omitted for old verbs.
The exact error means the broker had no connected caller binding. Its loss is
not yet explained; do not claim a protocol mismatch without evidence.
Verify the new CLI against the strict old request contract with registered
callers before choosing a fix. Preserve one-shot send/notify behavior and never
replay effects to diagnose a failed call (ARCH-ORDER, ARCH-SECURE).

The strict ccc0ee88 request fixture over a real private socket, using a stateful
broker, confirms actors/send/status succeed with registered callers. Disconnecting
the caller reproduces the exact unavailable message. Red tests first demonstrated
missing recovery guidance; CLI now identifies missing registration and advises
restart without claiming version skew (TL approval 90580aad-45d0-4fcc-9958-472dff9cbee2).
The unsupported notify case explicitly rejects Title before admission; it gets
unsupported-command/restart guidance. All operations remain single-exchange;
transport/lost-reply uncertainty remains unchanged.

### 2026-10-13 — proposed design for TL approval

**Approach.** Reuse terminal effects and the presenter's serialized host writer
  (ARCH-DRY). A new raw-byte filter would duplicate the existing chunk-safe VT
  parser; printing OSC in the CLI would feed the slot terminal and be swallowed.
  Refuse non-hub callers, with advice to message the hub, instead of silently
  converting a notification into a different action.

**Swallowing.** Console consumes child notification effects with notification
  emission disabled, for focused, hidden and hub slots alike. It records only
  per-slot counts; no notification text enters attention or notice history.
  Ordinary BEL attention remains. OSC 9 and 777 already map to effects; observe
  OSC 99 through the same parser as a non-forwardable notification event, without
  enabling a new standalone terminal notification protocol. Count accepted,
  bounded complete notification events; malformed/oversized strings remain
  safely discarded by the parser. Do not assemble Kitty multipart messages.
  Presenter sequence deduplication prevents replay from increasing the count.
  Aggregate counts per processed batch in the existing bounded diagnostic trace,
  identified by slot address; retain no bodies and create no new log family.

**Hub command.** Add the light verb `couch --notify [--title T] MESSAGE`, default
  title `Couch`. The CLI sends a closed notify request to the existing broker;
  the supervisor validates the current registered caller and its connected
  workspace against the hub root, never a caller-provided hub claim. Non-hub,
  missing hub, stale/disconnected caller, invalid payload or unavailable console
  yields a refusal without output. Reuse existing text sanitization/UTF-8 bounds,
  with an explicit title bound and semicolon handling for OSC 777 framing.
  Valid requests emit one OSC 777 notification through an application-owned
  presenter method, analogous to clipboard Copy, outside the child-effect path.
  No git/subprocess work, retries, new credential, queue or durable receipt.

**Outcome contract.** Success means the terminal bytes were written, not proof
  the OS displayed a notification. A transport timeout or interrupted/partial
  host write is unconfirmed and must say not to retry automatically. No effect
  is replayed on focus, repaint or reconnect. A repeat CLI invocation is a new
  intentional notification. Host serialization and cancellation stay owned by
  the existing presenter (ARCH-ORDER); request handlers spawn no background work.

**Verification.** Tests cover OSC 9/777/99 with split input and BEL/ST endings,
  focused/hidden/hub slots, no retained notification content, count deduplication,
  and unchanged title/clipboard/hyperlink/BEL behavior. CLI/protocol tests cover
  syntax, payload limits and hostile controls. A stateful fake host plus real
  broker/console seams proves hub emits once and stale/non-hub requests emit
  nothing, including cancellation/write failures. Update the existing PTY and
  headless Zellij notification tests to expect swallowing; run targeted tests
  and the complete queued `make -k test` before close. Actual desktop smoke only
  when TL schedules it.

**Documentation and next gate.** Update help, verb-cost table, embedded Couch
  skill, README and atlas. Send the command contract to TL for ariadne#350's
  tl-lead prose. After design approval, author the durable implementation plan
  with explicit entities, bounds and transitions, then run change-code; likely
  full flow given the CLI/protocol/presenter and regression-test surfaces.
