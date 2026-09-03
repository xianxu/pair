---
gate: boundary-review
issue: 170
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-02T13:01:40-07:00"
      agent: claude
      boundary: M1
      blocked: false
      protocol_error: no valid findings block
    - "n": 2
      timestamp: "2026-09-02T16:05:44-07:00"
      agent: claude
      boundary: M2
      blocked: false
      protocol_error: no valid findings block
    - "n": 3
      timestamp: "2026-09-02T17:00:01-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: Detached rows are auto-selected at startup without the binding proof resume requires, so `couch` exits 1 with no way through
          detail: |-
            actionableinventory.go:238 appends detached candidates before any ResolveEstablished
            gate, while the parked branch at :243 filters on it and DecideResume/ResumeContext
            require it for both. Reproduced against HEAD: a detached row with an unbound binding
            is listed, selected by SelectUniqueResumableRoot, and refused with
            resume-binding-unbound; couchcmd renders the error and returns 1, and `couch` has no
            new-thread escape hatch. Before M3 the parked-only selector skipped it and startup
            created a new thread.
          family: listed-implies-resumable
          round: 3
        - id: BR-2
          severity: Important
          title: No couchcore test pins StartInteractive's detached wiring; only the pty-gated couchcmd acceptance test covers it
          detail: |-
            Filtering rows to parked before the SelectUniqueResumableRoot call in a scratch copy
            leaves the entire couchcore suite green. The two acceptance tests hard-fail at
            pty.Open() in any environment without pty access, so the commit's mutation claim for
            the reattach test is unconfirmable there. A ~15-line twin of
            TestStartInteractiveResumesUniqueExactParkedRoot closes it and would also catch the
            Critical.
          family: seam-untested-at-runnable-level
          round: 3
        - id: BR-3
          severity: Important
          title: Plan Step 3b, the project file and the commit credit M3 with a physicalization fix that shipped at M2
          detail: |-
            actionableinventory.go's parked-AND-detached physicalization is unchanged in this
            window; git log -L attributes it to fac153c9 (#170 M2), and M3's new test passes
            against the M2 base production source. M3 delivered the proof, not the fix. Correct
            the step wording and workshop/projects/couch.md:921-926.
          family: record-claims-unverified-delivery
          round: 3
        - id: BR-4
          severity: Important
          title: M3's stated startup envelope contradicts the M2-corrected cost model and is unmeasured
          detail: |-
            plan.md:402 says "unchanged from #167 -- one local actionable snapshot ... no
            fan-out", but the snapshot spawns 2 + N zellij CLI processes (N = every pair session
            on the host, 5s timeout each) whenever a detach candidate exists, and M3 makes that
            the normal startup case. atlas/couch.md:401-406 frames the cost as refresh-worker
            only. Measure a startup with K detached threads and correct both sentences.
          family: envelope-claim-unmeasured
          round: 3
        - id: BR-5
          severity: Important
          title: README still describes the switcher's row states and unique resume as parked-only
          detail: |-
            README.md:360 "Rows expose only proven live and exact verified parked states"
            contradicts M2's detached rows, which M3's Done-when depends on; README.md:308
            "automatic unique resume reuses the parked thread instead" is the same residue one
            paragraph below the sentence M2 correctly widened to "the sole exact resumable
            thread".
          family: readme-stale-for-shipped-surface
          round: 3
        - id: BR-6
          severity: Minor
          title: The selector's rationale is restated near-verbatim in five artifacts
          detail: |-
            startup.go:11-24, startup_test.go:13-19, atlas/couch.md:439-446,
            workshop/projects/couch.md:908-919 and the archived #167 plan all carry the same
            three-paragraph argument. Correct today, five copies to keep in sync.
          family: prose-duplication
          round: 3
        - id: BR-7
          severity: Minor
          title: The selector's ambiguity fixtures pass the same ThreadAddress twice, a shape the projector cannot emit
          detail: |-
            startup_test.go:37-38 duplicates one row; ProjectActionableThreads emits one row per
            record. The realistic distinct-address case exists two rows below, so this is
            fixture realism only.
          family: fixture-realism
          round: 3
      boundary: M3
      blocked: true
    - "n": 4
      timestamp: "2026-09-02T17:22:10-07:00"
      agent: claude
      boundary: M3
      blocked: true
      protocol_error: no valid findings block
---

# Gate ledger — pair#170 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-02T13:01:40-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 2 — 2026-09-02T16:05:44-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 3 — 2026-09-02T17:00:01-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `listed-implies-resumable` Detached rows are auto-selected at startup without the binding proof resume requires, so `couch` exits 1 with no way through
  actionableinventory.go:238 appends detached candidates before any ResolveEstablished
  gate, while the parked branch at :243 filters on it and DecideResume/ResumeContext
  require it for both. Reproduced against HEAD: a detached row with an unbound binding
  is listed, selected by SelectUniqueResumableRoot, and refused with
  resume-binding-unbound; couchcmd renders the error and returns 1, and `couch` has no
  new-thread escape hatch. Before M3 the parked-only selector skipped it and startup
  created a new thread.
- **BR-2** [Important] `seam-untested-at-runnable-level` No couchcore test pins StartInteractive's detached wiring; only the pty-gated couchcmd acceptance test covers it
  Filtering rows to parked before the SelectUniqueResumableRoot call in a scratch copy
  leaves the entire couchcore suite green. The two acceptance tests hard-fail at
  pty.Open() in any environment without pty access, so the commit's mutation claim for
  the reattach test is unconfirmable there. A ~15-line twin of
  TestStartInteractiveResumesUniqueExactParkedRoot closes it and would also catch the
  Critical.
- **BR-3** [Important] `record-claims-unverified-delivery` Plan Step 3b, the project file and the commit credit M3 with a physicalization fix that shipped at M2
  actionableinventory.go's parked-AND-detached physicalization is unchanged in this
  window; git log -L attributes it to fac153c9 (#170 M2), and M3's new test passes
  against the M2 base production source. M3 delivered the proof, not the fix. Correct
  the step wording and workshop/projects/couch.md:921-926.
- **BR-4** [Important] `envelope-claim-unmeasured` M3's stated startup envelope contradicts the M2-corrected cost model and is unmeasured
  plan.md:402 says "unchanged from #167 -- one local actionable snapshot ... no
  fan-out", but the snapshot spawns 2 + N zellij CLI processes (N = every pair session
  on the host, 5s timeout each) whenever a detach candidate exists, and M3 makes that
  the normal startup case. atlas/couch.md:401-406 frames the cost as refresh-worker
  only. Measure a startup with K detached threads and correct both sentences.
- **BR-5** [Important] `readme-stale-for-shipped-surface` README still describes the switcher's row states and unique resume as parked-only
  README.md:360 "Rows expose only proven live and exact verified parked states"
  contradicts M2's detached rows, which M3's Done-when depends on; README.md:308
  "automatic unique resume reuses the parked thread instead" is the same residue one
  paragraph below the sentence M2 correctly widened to "the sole exact resumable
  thread".
- **BR-6** [Minor] `prose-duplication` The selector's rationale is restated near-verbatim in five artifacts
  startup.go:11-24, startup_test.go:13-19, atlas/couch.md:439-446,
  workshop/projects/couch.md:908-919 and the archived #167 plan all carry the same
  three-paragraph argument. Correct today, five copies to keep in sync.
- **BR-7** [Minor] `fixture-realism` The selector's ambiguity fixtures pass the same ThreadAddress twice, a shape the projector cannot emit
  startup_test.go:37-38 duplicates one row; ProjectActionableThreads emits one row per
  record. The realistic distinct-address case exists two rows below, so this is
  fixture realism only.

## Round 4 — 2026-09-02T17:22:10-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Open findings

- **BR-1** [Critical] `listed-implies-resumable` Detached rows are auto-selected at startup without the binding proof resume requires, so `couch` exits 1 with no way through
- **BR-2** [Important] `seam-untested-at-runnable-level` No couchcore test pins StartInteractive's detached wiring; only the pty-gated couchcmd acceptance test covers it
- **BR-3** [Important] `record-claims-unverified-delivery` Plan Step 3b, the project file and the commit credit M3 with a physicalization fix that shipped at M2
- **BR-4** [Important] `envelope-claim-unmeasured` M3's stated startup envelope contradicts the M2-corrected cost model and is unmeasured
- **BR-5** [Important] `readme-stale-for-shipped-surface` README still describes the switcher's row states and unique resume as parked-only
- **BR-6** [Minor] `prose-duplication` The selector's rationale is restated near-verbatim in five artifacts
- **BR-7** [Minor] `fixture-realism` The selector's ambiguity fixtures pass the same ThreadAddress twice, a shape the projector cannot emit
