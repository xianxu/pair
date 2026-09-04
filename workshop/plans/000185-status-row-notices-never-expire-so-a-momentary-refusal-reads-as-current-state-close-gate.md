---
gate: boundary-review
issue: 185
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-04T15:33:30-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: A transient notice now expires unseen on an idle console — nothing repaints when one is pushed
          detail: |-
            setNotice does not repaint and no other paint trigger is guaranteed, so on
            an idle pane a ctrl+backspace refusal is pushed, the 12s timer is armed, and
            the timer's own repaint renders the row WITHOUT the sentence — the operator
            never sees it. Probe on a fresh console after one lifetime painted only
            "[brain]". This is the outcome reportPrevious's comment at console.go:1258
            says must not happen. Fix: have syncNoticeExpiry repaint when the row body
            changed (it already reads the row under the lock every iteration), and pin it
            with a console test that writes previousByte and asserts the sentence reaches
            the host with NO explicit repaint() — red today. ARCH-PURPOSE: the retirement
            half shipped, the appearance half did not.
          family: row-must-track-the-feed
          round: 1
        - id: BR-2
          severity: Important
          title: The timer dedup is documented as correctness-critical in code, Spec, atlas and commit message; it is not
          detail: |-
            "Without this an event-heavy console would re-arm on every iteration and the
            notice would never actually go" is false: the re-arm uses row.Standing
            (expires - now), which shrinks, so the absolute deadline is preserved either
            way. Verified by deleting the guard — the whole couchtty package stays green,
            so it is also untested. It is a real churn-avoidance optimisation; restate it
            as one in console.go:658, the Spec's consequence 3, and atlas/couch.md so the
            codebase map stops recording an invariant the code does not rely on.
          family: overstated-guard-rationale
          round: 1
        - id: BR-3
          severity: Minor
          title: syncNoticeExpiry is the fourth near-identical arm-a-timer closure in Run
          detail: |-
            armInputEscape, armPanelEscape, syncSpinner and now syncNoticeExpiry all
            repeat stop/reset/assign-channel. A shared arm(&timer, &ch, d) helper is owed
            (ARCH-DRY). Following the established shape was the right local call.
          family: repeated-timer-arm-shape
          round: 1
        - id: BR-4
          severity: Minor
          title: NewFeed's nil-clock and lifetime<=0 defaults are unreachable and untested
          detail: |-
            All three call sites pass a real clock and a positive lifetime. A future
            caller passing 0 to mean "no expiry" silently gets 12s.
          family: unreachable-constructor-default
          round: 1
        - id: BR-5
          severity: Minor
          title: The noticeC fire branch leaves noticeExpiry stale, relying on the bottom-of-loop sync
          detail: |-
            console.go:702 clears noticeC but not noticeExpiry. Correct today because an
            expired row always reports a zero Expires, but the invariant is implicit.
          family: implicit-timer-state-invariant
          round: 1
        - id: BR-6
          severity: Minor
          title: README documents the reserved status row but not that transient notices now retire
          detail: |-
            One sentence at README.md:~305 — transients go after ~12s, an exit notice
            stands — would document the new user-visible timing. No new command, flag or
            keybinding, so the README gate otherwise passes.
          family: readme-behavior-timing
          round: 1
      blocked: true
---

# Gate ledger — pair#185 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-04T15:33:30-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `row-must-track-the-feed` A transient notice now expires unseen on an idle console — nothing repaints when one is pushed
  setNotice does not repaint and no other paint trigger is guaranteed, so on
  an idle pane a ctrl+backspace refusal is pushed, the 12s timer is armed, and
  the timer's own repaint renders the row WITHOUT the sentence — the operator
  never sees it. Probe on a fresh console after one lifetime painted only
  "[brain]". This is the outcome reportPrevious's comment at console.go:1258
  says must not happen. Fix: have syncNoticeExpiry repaint when the row body
  changed (it already reads the row under the lock every iteration), and pin it
  with a console test that writes previousByte and asserts the sentence reaches
  the host with NO explicit repaint() — red today. ARCH-PURPOSE: the retirement
  half shipped, the appearance half did not.
- **BR-2** [Important] `overstated-guard-rationale` The timer dedup is documented as correctness-critical in code, Spec, atlas and commit message; it is not
  "Without this an event-heavy console would re-arm on every iteration and the
  notice would never actually go" is false: the re-arm uses row.Standing
  (expires - now), which shrinks, so the absolute deadline is preserved either
  way. Verified by deleting the guard — the whole couchtty package stays green,
  so it is also untested. It is a real churn-avoidance optimisation; restate it
  as one in console.go:658, the Spec's consequence 3, and atlas/couch.md so the
  codebase map stops recording an invariant the code does not rely on.
- **BR-3** [Minor] `repeated-timer-arm-shape` syncNoticeExpiry is the fourth near-identical arm-a-timer closure in Run
  armInputEscape, armPanelEscape, syncSpinner and now syncNoticeExpiry all
  repeat stop/reset/assign-channel. A shared arm(&timer, &ch, d) helper is owed
  (ARCH-DRY). Following the established shape was the right local call.
- **BR-4** [Minor] `unreachable-constructor-default` NewFeed's nil-clock and lifetime<=0 defaults are unreachable and untested
  All three call sites pass a real clock and a positive lifetime. A future
  caller passing 0 to mean "no expiry" silently gets 12s.
- **BR-5** [Minor] `implicit-timer-state-invariant` The noticeC fire branch leaves noticeExpiry stale, relying on the bottom-of-loop sync
  console.go:702 clears noticeC but not noticeExpiry. Correct today because an
  expired row always reports a zero Expires, but the invariant is implicit.
- **BR-6** [Minor] `readme-behavior-timing` README documents the reserved status row but not that transient notices now retire
  One sentence at README.md:~305 — transients go after ~12s, an exit notice
  stands — would document the new user-visible timing. No new command, flag or
  keybinding, so the README gate otherwise passes.

## Open findings

- **BR-1** [Critical] `row-must-track-the-feed` A transient notice now expires unseen on an idle console — nothing repaints when one is pushed
- **BR-2** [Important] `overstated-guard-rationale` The timer dedup is documented as correctness-critical in code, Spec, atlas and commit message; it is not
- **BR-3** [Minor] `repeated-timer-arm-shape` syncNoticeExpiry is the fourth near-identical arm-a-timer closure in Run
- **BR-4** [Minor] `unreachable-constructor-default` NewFeed's nil-clock and lifetime<=0 defaults are unreachable and untested
- **BR-5** [Minor] `implicit-timer-state-invariant` The noticeC fire branch leaves noticeExpiry stale, relying on the bottom-of-loop sync
- **BR-6** [Minor] `readme-behavior-timing` README documents the reserved status row but not that transient notices now retire
