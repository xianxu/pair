---
gate: boundary-review
issue: 378
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-01T22:21:06-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: classifyForAction's early Unknown return bypasses ClassifyThread's Unproven precedence (live/busy beat unknown)
          detail: 'switchagent.go:97-98 returns unusable/unknown for any Unknown registry actor before gatherThreadEvidence/ClassifyThread run, contradicting actionableinventory.go:507-523 (live and busy outrank Unproven) and skipping the ErrThreadNotFound path. Route Unknown registry actors into evidence.Unproven so precedence lives in one place, and add a live+unknown mixed-actor test. Instances in window: switchagent.go:97 only; related older duplication Couch.Liveness (couch.go:1033) vs observeExactProcess (couch.go:999).'
          family: classification-precedence-single-source
          round: 1
        - id: BR-2
          severity: Important
          title: Done-when "dead actor + parked record classifies as parked" is asserted only as not-live on a non-parked fixture
          detail: TestArchiveIgnoresADeadRegistryActor uses validThreadRecord (no park) and checks state != ThreadLive. Either build a parked fixture with a dead actor and assert ThreadParked, or revise the Done-when via a Revisions entry.
          family: done-when-clause-asserted
          round: 1
        - id: BR-3
          severity: Minor
          title: atlas says every registry reader probes Couch.Liveness; several readers do not
          detail: ownsContinuationHelper (continuation.go:381), ResolveRef/knownTrees (couch.go:1073,1123) and the owned loop (switchagent.go:180) read records without probing, and slotrecovery uses observeExactProcess. Narrow the claim to readers that treat a record as hosting proof. Also, withoutDead's comment that it bounds the registry holds only after the next launch.
          family: doc-claim-overreach
          round: 1
      recipe: small-diff-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-01T22:29:26-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Unknown actors join evidence.Unproven (switchagent.go:97-112); TestALiveRegistryActorOutranksAnUnprovableOne pins live-beats-unknown.
          round: 2
        - id: BR-2
          disposition: addressed
          note: TestArchiveIgnoresADeadRegistryActor now uses createParkedThreadInCouch + established binding and asserts ThreadParked.
          round: 2
        - id: BR-3
          disposition: addressed
          note: atlas/couch.md now scopes the probe claim to readers treating a record as hosting proof (classifyForAction); withoutDead comment states the per-launch bound.
          round: 2
      recipe: small-diff-review
      blocked: false
---

# Gate ledger — pair#378 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-01T22:21:06-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `classification-precedence-single-source` classifyForAction's early Unknown return bypasses ClassifyThread's Unproven precedence (live/busy beat unknown)
  switchagent.go:97-98 returns unusable/unknown for any Unknown registry actor before gatherThreadEvidence/ClassifyThread run, contradicting actionableinventory.go:507-523 (live and busy outrank Unproven) and skipping the ErrThreadNotFound path. Route Unknown registry actors into evidence.Unproven so precedence lives in one place, and add a live+unknown mixed-actor test. Instances in window: switchagent.go:97 only; related older duplication Couch.Liveness (couch.go:1033) vs observeExactProcess (couch.go:999).
- **BR-2** [Important] `done-when-clause-asserted` Done-when "dead actor + parked record classifies as parked" is asserted only as not-live on a non-parked fixture
  TestArchiveIgnoresADeadRegistryActor uses validThreadRecord (no park) and checks state != ThreadLive. Either build a parked fixture with a dead actor and assert ThreadParked, or revise the Done-when via a Revisions entry.
- **BR-3** [Minor] `doc-claim-overreach` atlas says every registry reader probes Couch.Liveness; several readers do not
  ownsContinuationHelper (continuation.go:381), ResolveRef/knownTrees (couch.go:1073,1123) and the owned loop (switchagent.go:180) read records without probing, and slotrecovery uses observeExactProcess. Narrow the claim to readers that treat a record as hosting proof. Also, withoutDead's comment that it bounds the registry holds only after the next launch.

## Round 2 — 2026-10-01T22:29:26-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Unknown actors join evidence.Unproven (switchagent.go:97-112); TestALiveRegistryActorOutranksAnUnprovableOne pins live-beats-unknown.
- BR-2 — addressed — TestArchiveIgnoresADeadRegistryActor now uses createParkedThreadInCouch + established binding and asserts ThreadParked.
- BR-3 — addressed — atlas/couch.md now scopes the probe claim to readers treating a record as hosting proof (classifyForAction); withoutDead comment states the per-launch bound.

## Open findings

(none — every finding has been disposed)
