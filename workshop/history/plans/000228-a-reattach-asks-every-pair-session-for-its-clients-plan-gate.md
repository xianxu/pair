---
gate: plan-quality
issue: 228
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-10T23:06:38-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: The planned couchcore tests stub only the checker's zellij reads, leaving its session deleter pointed at the real host
          detail: |-
            Task 4 Step 1 builds a production ScopedThreadArtifactCollisionChecker and sets only Zellij.Path; the
            constructor also sets Sessions: launcher.OSRuntime{} (artifactcollision.go:75-77). On the red run the
            shortened resumeRegistrationTimeout drives failTrackedPostAckStart -> quiescePostAckStart (couch.go:565-585)
            -> QuiesceThreadSession -> OSRuntime.DeleteSession (thread_claim.go:277), a real zellij delete-session plus a
            SIGKILL sweep, in a retry loop. Pin checker.Sessions to the existing fakeSessionDeleter
            (artifactcollision_test.go:144-148) and assert it recorded no deletions.
          family: test-touches-real-host-state
          round: 1
        - id: PQ-2
          severity: Minor
          title: The list-clients producer enumeration names two of the three in the tree
          detail: |-
            OSRuntime.ListSessions (osruntime.go:197-199) issues one list-clients per pair session for `pair list` and is
            absent from the plan's "left alone" list, so the enumeration reads complete when it is not, and Task 6's
            stale-cost-comment sweep has no reason to visit it. Name it as a justified full scan.
          family: enumerate-the-class
          round: 1
        - id: PQ-3
          severity: Minor
          title: The failure-only PairSession calls are listed as left alone but inherit site 3's narrowing
          detail: |-
            launch_existing.go:181 and launch_existing.go:220 call the same PairSession that site 3 changes; both read
            only Name/Present so the behavior is correct, but the table's note misdescribes them and the Log's per-site
            record (Done-when bullet 3) would inherit the error.
          family: site-table-accuracy
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-10T23:08:50-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Task 4 Step 0 sandboxedChecker redirects every seam, pins Sessions to fakeSessionDeleter with a no-deletions assertion, and backstops with a PATH shim plus a stub-log-only assertion and a mutation row.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Three-of-three producer enumeration now present and verified against the tree; ListSessions named as a justified full scan with its reason.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Failure-path PairSession calls recorded as inheriting site 3's narrowing; the Name/Present claim holds by the binding type itself.
          round: 2
      blocked: false
content_hash: ad15c664d24ae5d6363cfd13403108dd9818459395ea2a1a14bbd5b83f1fd412
---

# Gate ledger — pair#228 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-10T23:06:38-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `test-touches-real-host-state` The planned couchcore tests stub only the checker's zellij reads, leaving its session deleter pointed at the real host
  Task 4 Step 1 builds a production ScopedThreadArtifactCollisionChecker and sets only Zellij.Path; the
  constructor also sets Sessions: launcher.OSRuntime{} (artifactcollision.go:75-77). On the red run the
  shortened resumeRegistrationTimeout drives failTrackedPostAckStart -> quiescePostAckStart (couch.go:565-585)
  -> QuiesceThreadSession -> OSRuntime.DeleteSession (thread_claim.go:277), a real zellij delete-session plus a
  SIGKILL sweep, in a retry loop. Pin checker.Sessions to the existing fakeSessionDeleter
  (artifactcollision_test.go:144-148) and assert it recorded no deletions.
- **PQ-2** [Minor] `enumerate-the-class` The list-clients producer enumeration names two of the three in the tree
  OSRuntime.ListSessions (osruntime.go:197-199) issues one list-clients per pair session for `pair list` and is
  absent from the plan's "left alone" list, so the enumeration reads complete when it is not, and Task 6's
  stale-cost-comment sweep has no reason to visit it. Name it as a justified full scan.
- **PQ-3** [Minor] `site-table-accuracy` The failure-only PairSession calls are listed as left alone but inherit site 3's narrowing
  launch_existing.go:181 and launch_existing.go:220 call the same PairSession that site 3 changes; both read
  only Name/Present so the behavior is correct, but the table's note misdescribes them and the Log's per-site
  record (Done-when bullet 3) would inherit the error.

## Round 2 — 2026-09-10T23:08:50-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Task 4 Step 0 sandboxedChecker redirects every seam, pins Sessions to fakeSessionDeleter with a no-deletions assertion, and backstops with a PATH shim plus a stub-log-only assertion and a mutation row.
- PQ-2 — addressed — Three-of-three producer enumeration now present and verified against the tree; ListSessions named as a justified full scan with its reason.
- PQ-3 — addressed — Failure-path PairSession calls recorded as inheriting site 3's narrowing; the Name/Present claim holds by the binding type itself.

## Open findings

(none — every finding has been disposed)
