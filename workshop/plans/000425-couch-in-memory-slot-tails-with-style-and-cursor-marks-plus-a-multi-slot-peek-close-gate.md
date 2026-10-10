---
gate: boundary-review
issue: 425
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-10T12:24:27-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Done-when 1 (live tail from wrapper memory, well under a second) never exercised end to end or timed
          detail: The Log says the live path needs a Couch restart; the only live run fell back to the recording. Restart Couch on this build, relaunch 3+ slots, time couch --peek over them, and record source live plus the dim/cursor marks in --verified.
          family: done-when-unverified-live
          round: 1
        - id: BR-2
          severity: Minor
          title: operationdispatch.go peek checks a["tag"] == "" but peek declares no tag argument
          family: dead-condition
          round: 1
        - id: BR-3
          severity: Minor
          title: ExpandPeekReferences splits any comma and expands x:1:2, changing how thread paths with those characters resolve
          family: reference-grammar-overload
          round: 1
        - id: BR-4
          severity: Minor
          title: A wrapper from before this change refusing endpoint op tail is not mapped to a relaunch hint; old-Couch detection matches error text
          family: version-skew-error-mapping
          round: 1
        - id: BR-5
          severity: Minor
          title: BoundTail single-line byte truncation can cut a markup token mid-way
          family: byte-truncation-splits-token
          round: 1
        - id: BR-6
          severity: Minor
          title: No test runs handleTail against a real RemoteEndpoint socket; ambiguous and unsupported branches untested
          family: stateful-fake-coverage
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-10T12:26:43-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Done-when 1 revised with a documented ops decision deferring the live run to the TL; every hop proven, including the timed real-socket test TestTailOverEndpointSocketIsFast (~0.7 ms). The Revision text still says the issue is not done until the TL check passes, which contradicts the close.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: operationdispatch.go:171 still checks a["tag"] == "", but peek declares no tag arg; harmless dead condition.
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: ExpandPeekReferences is unchanged; a thread path containing a comma or matching x:1:2 is still split.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: 'A pre-425 wrapper refusing op tail still surfaces as "unavailable: <raw error>" from handleTail; old-Couch detection in readSlotTail is still substring matching.'
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: BoundTail single-line cut at MaxTailBytes is unchanged; practically unreachable at real pane widths.
          round: 2
        - id: BR-6
          disposition: not-addressed
          note: The wrapper hop now has a real-socket test, but handleTail itself only runs against a fake endpoint, and its ambiguous/unsupported branches have no test.
          round: 2
      recipe: milestone-review
      reviewed: b754b12eab635306f23915fc492ee48ee50b0cef
      blocked: false
---

# Gate ledger — pair#425 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-10T12:24:27-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `done-when-unverified-live` Done-when 1 (live tail from wrapper memory, well under a second) never exercised end to end or timed
  The Log says the live path needs a Couch restart; the only live run fell back to the recording. Restart Couch on this build, relaunch 3+ slots, time couch --peek over them, and record source live plus the dim/cursor marks in --verified.
- **BR-2** [Minor] `dead-condition` operationdispatch.go peek checks a["tag"] == "" but peek declares no tag argument
- **BR-3** [Minor] `reference-grammar-overload` ExpandPeekReferences splits any comma and expands x:1:2, changing how thread paths with those characters resolve
- **BR-4** [Minor] `version-skew-error-mapping` A wrapper from before this change refusing endpoint op tail is not mapped to a relaunch hint; old-Couch detection matches error text
- **BR-5** [Minor] `byte-truncation-splits-token` BoundTail single-line byte truncation can cut a markup token mid-way
- **BR-6** [Minor] `stateful-fake-coverage` No test runs handleTail against a real RemoteEndpoint socket; ambiguous and unsupported branches untested

## Round 2 — 2026-10-10T12:26:43-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Done-when 1 revised with a documented ops decision deferring the live run to the TL; every hop proven, including the timed real-socket test TestTailOverEndpointSocketIsFast (~0.7 ms). The Revision text still says the issue is not done until the TL check passes, which contradicts the close.
- BR-2 — not-addressed — operationdispatch.go:171 still checks a["tag"] == "", but peek declares no tag arg; harmless dead condition.
- BR-3 — not-addressed — ExpandPeekReferences is unchanged; a thread path containing a comma or matching x:1:2 is still split.
- BR-4 — not-addressed — A pre-425 wrapper refusing op tail still surfaces as "unavailable: <raw error>" from handleTail; old-Couch detection in readSlotTail is still substring matching.
- BR-5 — not-addressed — BoundTail single-line cut at MaxTailBytes is unchanged; practically unreachable at real pane widths.
- BR-6 — not-addressed — The wrapper hop now has a real-socket test, but handleTail itself only runs against a fake endpoint, and its ambiguous/unsupported branches have no test.

## Open findings

- **BR-2** [Minor] `dead-condition` operationdispatch.go peek checks a["tag"] == "" but peek declares no tag argument
- **BR-3** [Minor] `reference-grammar-overload` ExpandPeekReferences splits any comma and expands x:1:2, changing how thread paths with those characters resolve
- **BR-4** [Minor] `version-skew-error-mapping` A wrapper from before this change refusing endpoint op tail is not mapped to a relaunch hint; old-Couch detection matches error text
- **BR-5** [Minor] `byte-truncation-splits-token` BoundTail single-line byte truncation can cut a markup token mid-way
- **BR-6** [Minor] `stateful-fake-coverage` No test runs handleTail against a real RemoteEndpoint socket; ambiguous and unsupported branches untested
