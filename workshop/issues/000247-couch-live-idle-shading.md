---
id: 000247
status: open
deps: []
github_issue:
created: 2026-09-13
updated: 2026-09-13
estimate_hours:
---

# Shade live Couch threads by idle time

## Problem

Live Couch threads all use normal foreground even when some have seen no action
for hours or days. The operator wants active threads to stand out and idle
threads to recede through shades of gray, in both the tab/status bar and the
switcher.

## Spec

Apply one shared idle-age presentation policy to live thread labels in Couch's
tab bar and switcher. Increasing idle time progressively reduces prominence,
while retaining legibility. Lifecycle state remains live; shading does not
reorder threads or change their behavior.

Provisional four-level ramp: under 5 minutes uses normal foreground; 5 minutes
to under 1 hour mildly subdued; 1 hour to under 24 hours more subdued; 24 hours
or more most subdued. Thresholds are a proposal for operator review, not yet
approved. Selection/focused-thread styling and pending notification emphasis
take precedence, so fading never hides navigation or attention cues.

Activity semantics pending operator response: recommended meaningful user input
or agent work/output, versus user interaction only. Do not treat arbitrary PTY
bytes as work: cursor blinking, redraws, status refreshes, and polling must not
keep an idle thread bright. Inspect existing lifecycle/activity signals before
choosing the source. Existing LastActiveAt includes park/detach timestamps and
cannot be assumed to represent live work. Thread age must survive Couch restart
without making all old threads look newly active; unknown activity needs an
explicit neutral presentation, not a fabricated age.

Derive both surfaces from the same timestamp semantics, band classifier, and
theme-aware style policy (ARCH-DRY). Coordinate with #225's shared bar styling
and #217's focus dimming. Fixed dark-background grayscale values invert their
prominence on light themes; verify both themes and color-disabled rendering.
Use existing bounded refresh scheduling where possible; no per-byte durable
writes or independent per-thread timers. Specify timestamp persistence and
retirement with the thread's lifecycle (ARCH-CONSTRAINTS, ARCH-FUNERAL).

## Done when

- Live thread labels in both the Couch tab bar and switcher become progressively
  subdued according to the same approved idle-time policy.
- Relevant activity refreshes prominence; idle redraw traffic does not.
- Selection, notification, and error cues remain readable and retain precedence.
- Tests use an injected clock to cross exact thresholds and exercise activity
  delivery into both renderers, unknown/future timestamps, restart, and style precedence.
- Dark/light theme and no-color checks pass; operator docs explain the shading.

## Plan

- [ ] Settle activity semantics, thresholds, theme behavior, and refresh/persistence design.
- [ ] Author a durable implementation plan coordinated with existing bar-style work.
- [ ] Implement shared policy, verify both surfaces, and update docs through SDLC gates.

## Log

### 2026-09-13

Captured operator request. Inspected couchtty/menu_render.go (existing AgeBandFor
and non-live fading), couchtty/reserve.go (selection/notification styles but no
age), and threadstore timestamp updates. Read active #225 for theme constraints.
Asked which activity should reset idle time; response pending. No code changed.


## Revisions

### 2026-09-14 — Add a simple recent-traffic dot

The operator requested a green period after the thread name and explicitly chose simplicity over distinguishing useful work from terminal motion. This addition is independently scoped from the longer-term idle-shading proposal above; it does not require solving semantic work detection or persistent idle age.

Approved dot behavior:

- Display `ariadne.` with only the final period green when at least 15 bytes of fresh child PTY output arrived within the preceding 15 seconds. Spinners, redraws and control traffic count; no screen comparison or agent-work inference.
- Apply the same recent-traffic predicate to thread names in the status bar and switcher, including background threads. The dot is presentation only, not part of the thread's name or identity.
- Remove the dot when the rolling-window threshold is no longer met or the attachment ends. Replay and Couch's own paints do not create new activity. A new attachment starts with no observed activity; keep this transient state in memory.
- Preserve existing selection and notification cues. Use existing console scheduling for expiry; no per-thread timer, per-byte persistence, or new background process. Bound storage and per-batch work by the small threshold.
- Call the internal signal recent terminal activity. It is evidence of output, not proof of process health or useful progress. In color-disabled rendering the period still indicates activity without escape sequences.

Dot acceptance: an injected clock verifies the 15-byte/15-second boundaries, expiry without further output, fresh versus replayed bytes, background activity and attachment reset. Production-path tests reach both renderers and preserve clipping/click spans and existing emphasis. This supplies the settled dot semantics for the implementation plan; the earlier ban on arbitrary PTY traffic applies only to the separate long-term meaningful-idle shading proposal.

### 2026-09-14 — Capture status

Recorded the operator's decision; no implementation begun. The shared checkout currently carries active #250 recovery work, including staged changes, so this update publishes only #247's issue record.
