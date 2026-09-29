---
id: 000247
status: working
deps: []
github_issue:
created: 2026-09-13
updated: 2026-09-28
estimate_hours: 4.35
card_mirror: 'c991ef7b9e7d16802fb6f526d44cc2836083d00d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-28T21:31:56-07:00
flow: {kind: full, provenance: inferred}
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

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec             design=1.0 impl=0.05
item: greenfield-go-module   design=0.5 impl=0.22
item: tui-screen             design=0.5 impl=0.26
item: smaller-go-module      design=0.1 impl=0.14
item: smaller-go-module      design=0.1 impl=0.14
item: greenfield-go-module   design=0.3 impl=0.22
item: atlas-docs             design=0.1 impl=0.05
item: milestone-review       design=0.0 impl=0.14
item: milestone-review       design=0.0 impl=0.14
design-buffer: 0.15
total: 4.35
```

Items, in order:
1. Spec and plan: brainstorm, two review rounds, operator revisions.
2. The pure idle-level and fade policy (`idle_shade.go`: level, blend, quantize, style).
3. Both renderers: the tab bar chip and the switcher's live rows.
4. The shared `threadactivity`, plus the title poller moving onto it.
5. The console activity pass, mirroring slot git.
6. The OSC 10/11 palette query and reply capture.
7. Docs and atlas.
8. Two milestone reviews.

`impl=` values are 40 % of the v2 table's midpoints (v3.1). The design buffer
is +15 % because there's a thorough plan document.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* (The calibration doc is flagged stale; the numbers are provisional per #127.)

## Plan

Durable plan: `workshop/plans/000247-couch-live-idle-shading-plan.md`.

- [x] M1 — pure policy + renderers: `IdleLevelFor` (1 day / 3 days, 3 levels), `FadeStyle`
      (blend toward the terminal background; SGR 90 / no-color fallbacks), tab bar
      and switcher live rows faded with selection/bell/placeholder precedence.
- [ ] M2 — IO seams + wiring + docs: shared `threadactivity.Latest` (the title
      poller migrated to it), a 60 s console activity pass, an OSC 10/11 palette
      query and reply capture, `couchcmd` wiring, README/help/atlas, and an
      operator smoke on dark + light themes.

## Log


- 2026-09-28: closed M1 — IdleLevelFor/FadeStyle/blend/quantize256 table tests; tab bar + switcher fade with selection/bell/attention/placeholder precedence and byte-identical level 0; mutation-checked both render guards; go test ./cmd/internal/couchtty green (unsandboxed); review verdict: SHIP
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

### 2026-09-28 — Operator answers the open idle-shading questions

- **Activity = both.** Operator input to the thread AND agent work or output
  reset idle age. Redraws, cursor blink, status refreshes and polling still
  don't count.
- **Thresholds: 1 h, 24 h, 48 h**, still four levels: under 1 h normal;
  1 h to under 24 h mildly faded; 24 h to under 48 h more faded; 48 h or more
  most faded. (Replaces the provisional 5 min / 1 h / 24 h ramp.)
- **Fade both label colors**: the normal foreground (white on a dark theme)
  and the amber label color, where the terminal can express it.
- **Precedence confirmed**: selected/focused styling and pending notifications
  keep their own emphasis. Both themes and color-disabled rendering must stay
  correct.

### 2026-09-28 — Color approach and scope

- Operator chose **blend toward the terminal's background**. Couch asks the
  terminal once for its colors (OSC 10/11) and mixes each label color toward
  the background (0/35/55/70 %). The unanswered fallback is SGR 90 grey, with
  amber unfaded.
- The recent-traffic dot (2026-09-14 revision above) is **split out to #342**.
  It is no longer part of this issue.

### 2026-09-28 — Operator shrinks the ramp to three levels

- **Thresholds: 1 day and 3 days, three levels**: under 1 day normal; 1 day to
  under 3 days faded; 3 days or more more faded. This supersedes the
  1 h / 24 h / 48 h four-level ramp recorded earlier today. Blend weights
  become 0 / 40 / 65 %.

### 2026-09-28 — M1 implementation

- `idle_shade.go`: `IdleLevelFor` (24 h / 72 h, inclusive; unknown, zero or
  future → fresh), `FadeStyle` (blend toward the reported background; SGR 90
  when the palette is unknown; `NO_COLOR` and level 0 keep today's bytes),
  `blend`, `quantize256` (cube + grey ramp, system colours skipped).
- Tab bar (`RenderStatusRow`): `StatusActor.Idle` and `StatusModel.Palette`.
  The fade is the fallback case after placeholder/bell, and never applies to
  the active chip. Amber glyphs fade with their chip.
- Switcher: live rows without attention fade from `MenuState.Activity` and
  `MenuState.Palette`. The selected row, attention rows and non-live rows
  (their age ramp) are unchanged.
- Mutation-checked (cp/cmp revert): removing the tab-bar fade case →
  `TestRenderStatusRowFadesAnIdleChip` fails; disabling the live-row branch →
  `TestSwitcherFadesAnIdleLiveRowAndItsAmberGlyphs` fails.
- `go test ./cmd/internal/couchtty` green (unsandboxed; the pty/tmp tests need it).
