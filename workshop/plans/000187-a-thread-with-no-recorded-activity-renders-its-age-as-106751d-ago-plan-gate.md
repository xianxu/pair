---
gate: plan-quality
issue: 187
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-04T21:52:22-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: AgeBandFor guard changes nothing visible — ageColor maps AgeUnknown and AgeOld to the same escape
          detail: |-
            menu_render.go:58 already defines AgeUnknown, but ageColor (menu_render.go:447)
            has only AgeRecent/AgeDays/default, so AgeUnknown and AgeOld both render
            "\x1b[38;5;240m". The bullet's stated purpose — not colouring a no-activity
            thread as ancient — is not delivered. State whether AgeUnknown gets a distinct
            colour or the dim colour is deliberate, and name AgeUnknown rather than
            describing a new band (ARCH-DRY, ARCH-PURPOSE).
          family: unobservable-state-change
          round: 1
        - id: PQ-2
          severity: Important
          title: Two of three plan bullets name no function to unit-test and no adversarial input class
          detail: |-
            Only relativeMenuAge and the row text have a stated test surface. AgeBandFor
            and the RetireIncarnation clock change have none. One strategy line each:
            AgeBandFor returns a band distinct from AgeOld for the zero time; detach with
            a backward FixedClock (clock.go:16) does not reduce a recorded LastActiveAt,
            which is what MonotonicLastActiveAt (parktransaction.go:269) exists for. Name
            the file — retireincarnation_test.go or detach_test.go.
          family: named-test-surface-missing
          round: 1
        - id: PQ-3
          severity: Minor
          title: Plan does not say where RetireIncarnation's new time argument comes from
          detail: |-
            c.Clock.Now() (couch.go:29) is the seam; note that it should be read once
            before detach.go's revision-conflict retry loop rather than per iteration,
            so a retry does not shift the recorded activity time.
          family: unstated-injection-seam
          round: 1
        - id: PQ-4
          severity: Minor
          title: Bullets 1 and 2 install two independent zero-guards over the same expression
          detail: |-
            relativeMenuAge and AgeBandFor each compute now.Sub(lastActive); a shared
            "has recorded activity" predicate would keep the two guards from drifting
            (ARCH-DRY). Defensible to duplicate at this size.
          family: duplicated-guard-expression
          round: 1
      blocked: true
---

# Gate ledger — pair#187 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-04T21:52:22-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `unobservable-state-change` AgeBandFor guard changes nothing visible — ageColor maps AgeUnknown and AgeOld to the same escape
  menu_render.go:58 already defines AgeUnknown, but ageColor (menu_render.go:447)
  has only AgeRecent/AgeDays/default, so AgeUnknown and AgeOld both render
  "\x1b[38;5;240m". The bullet's stated purpose — not colouring a no-activity
  thread as ancient — is not delivered. State whether AgeUnknown gets a distinct
  colour or the dim colour is deliberate, and name AgeUnknown rather than
  describing a new band (ARCH-DRY, ARCH-PURPOSE).
- **PQ-2** [Important] `named-test-surface-missing` Two of three plan bullets name no function to unit-test and no adversarial input class
  Only relativeMenuAge and the row text have a stated test surface. AgeBandFor
  and the RetireIncarnation clock change have none. One strategy line each:
  AgeBandFor returns a band distinct from AgeOld for the zero time; detach with
  a backward FixedClock (clock.go:16) does not reduce a recorded LastActiveAt,
  which is what MonotonicLastActiveAt (parktransaction.go:269) exists for. Name
  the file — retireincarnation_test.go or detach_test.go.
- **PQ-3** [Minor] `unstated-injection-seam` Plan does not say where RetireIncarnation's new time argument comes from
  c.Clock.Now() (couch.go:29) is the seam; note that it should be read once
  before detach.go's revision-conflict retry loop rather than per iteration,
  so a retry does not shift the recorded activity time.
- **PQ-4** [Minor] `duplicated-guard-expression` Bullets 1 and 2 install two independent zero-guards over the same expression
  relativeMenuAge and AgeBandFor each compute now.Sub(lastActive); a shared
  "has recorded activity" predicate would keep the two guards from drifting
  (ARCH-DRY). Defensible to duplicate at this size.

## Open findings

- **PQ-1** [Important] `unobservable-state-change` AgeBandFor guard changes nothing visible — ageColor maps AgeUnknown and AgeOld to the same escape
- **PQ-2** [Important] `named-test-surface-missing` Two of three plan bullets name no function to unit-test and no adversarial input class
- **PQ-3** [Minor] `unstated-injection-seam` Plan does not say where RetireIncarnation's new time argument comes from
- **PQ-4** [Minor] `duplicated-guard-expression` Bullets 1 and 2 install two independent zero-guards over the same expression
