---
gate: plan-quality
issue: 458
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-12T00:59:15-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Recorder inside Chrome via LaunchOptions/raw CDP events reverses browser.go's stated seam (no CDP crosses Browser) without saying so; Cast to Launch wiring unclear
          detail: browser.go:5-6 says the recorder will use only Browser. The plan forwards raw CDP events and a Caller instead, but CastConfig.Launch is func(ctx) (Browser, error) (cast.go:13), so how the Recording(id) sink reaches LaunchOptions.Recorder is unspecified, and FakeBrowser casts can't exercise it. State the new seam, update the doc, and specify the wiring.
          family: unstated-seam-change
          round: 1
        - id: PQ-2
          severity: Important
          title: recording.FrameWriter does not exist; PutFrame is a method on the concrete *DirWriter (recording/dir.go:127)
          detail: 'Adding an interface to #457''s contract package is a contract change. List it with the M1 format additions coordinated through the TL.'
          family: unbacked-existing-claim
          round: 1
        - id: PQ-3
          severity: Important
          title: No handling for stale queued DOM events across synchronous re-snapshots, or for active-session detach / frame detach mid-seg
          detail: getDocument invalidates node ids, so events Chrome sent before the snapshot response but queued behind it will be applied to the new tree on every seg, documentUpdated or resync. Name the discriminator (a sequence stamp), the policy for the uncancellable events, and a sequence test with injectable ordering against the fake.
          family: order-enumeration-missing
          round: 1
        - id: PQ-4
          severity: Important
          title: Risky functions (mirror translate, path resolver, password rules) not named with an adversarial input class and mechanical guard
          detail: 'For example: random mutation sequences on the fake, then reduce(output) must equal the fake DOM and ValidateDir must pass; stale or malformed paths fuzzed; no event sequence lets a password value past the reducer.'
          family: test-strategy-per-risky-function
          round: 1
        - id: PQ-5
          severity: Minor
          title: Serial CDP calls (getResponseBody, callFunctionOn per sheet change, screenshots) on the recorder goroutine have no latency budget for live record emission
          family: operating-envelope-latency
          round: 1
        - id: PQ-6
          severity: Minor
          title: Id map kept for the whole recording grows with every node ever seen; no bound stated
          family: unbounded-growth
          round: 1
        - id: PQ-7
          severity: Minor
          title: A value recorded while an input was type=text, before it flips to password, is already recorded; say whether that is accepted
          family: secret-leak-path
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T01:01:13-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Seam stated (recorder on the CDP side, Browser unchanged, browser.go comment corrected); Launch signature change and Output wiring specified.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: FrameWriter is a consumer-side interface in package recorder; contract package untouched.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Conn sequence numbering as the discriminator (Seq < reply number dropped); policies for unknown node, detach, and pending resync, with injected-ordering tests.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Property test vs fake DOM plus ValidateDir; fuzz target for the path resolver; planted-secret property test.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Side workers with bounded queues; 250 ms p95 budget measured in M3.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Live nodes plus a 50k graveyard per session, dropped on document replacement.
          round: 2
        - id: PQ-7
          disposition: addressed
          note: Explicitly accepted, with the reason.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-12T01:01:58-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: 'Seam stated in "Plug-in (M2)": recorder inside *Chrome, Browser unchanged, browser.go comment corrected; Launch takes recorder.Output.'
          round: 3
        - id: PQ-2
          disposition: addressed
          note: FrameWriter is a consumer-side interface in package recorder, satisfied by *recording.DirWriter.
          round: 3
        - id: PQ-3
          disposition: addressed
          note: 'Ordering section: Conn sequence numbers, snapshot reply discriminator, session/frame detach and unknown-node cases with sequence tests.'
          round: 3
        - id: PQ-4
          disposition: addressed
          note: One strategy line each for mirror translate (property), path resolver (fuzz), password rules (property with planted secrets), segments.
          round: 3
        - id: PQ-5
          disposition: addressed
          note: Side workers for asset bodies and screenshots with bounded queues; 250 ms p95 budget measured in M3.
          round: 3
        - id: PQ-6
          disposition: addressed
          note: Live nodes plus a 50 000-entry graveyard per session, dropped on document replacement; counter never reused.
          round: 3
        - id: PQ-7
          disposition: addressed
          note: 'Explicitly accepted: a value typed while type=text is recorded and not erased on the type flip.'
          round: 3
      blocked: false
content_hash: ca37d6cd533ba6f714a7d5351b035da2c92db0da722d5c45718f60cd9d9e53b1
---

# Gate ledger — pair#458 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T00:59:15-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `unstated-seam-change` Recorder inside Chrome via LaunchOptions/raw CDP events reverses browser.go's stated seam (no CDP crosses Browser) without saying so; Cast to Launch wiring unclear
  browser.go:5-6 says the recorder will use only Browser. The plan forwards raw CDP events and a Caller instead, but CastConfig.Launch is func(ctx) (Browser, error) (cast.go:13), so how the Recording(id) sink reaches LaunchOptions.Recorder is unspecified, and FakeBrowser casts can't exercise it. State the new seam, update the doc, and specify the wiring.
- **PQ-2** [Important] `unbacked-existing-claim` recording.FrameWriter does not exist; PutFrame is a method on the concrete *DirWriter (recording/dir.go:127)
  Adding an interface to #457's contract package is a contract change. List it with the M1 format additions coordinated through the TL.
- **PQ-3** [Important] `order-enumeration-missing` No handling for stale queued DOM events across synchronous re-snapshots, or for active-session detach / frame detach mid-seg
  getDocument invalidates node ids, so events Chrome sent before the snapshot response but queued behind it will be applied to the new tree on every seg, documentUpdated or resync. Name the discriminator (a sequence stamp), the policy for the uncancellable events, and a sequence test with injectable ordering against the fake.
- **PQ-4** [Important] `test-strategy-per-risky-function` Risky functions (mirror translate, path resolver, password rules) not named with an adversarial input class and mechanical guard
  For example: random mutation sequences on the fake, then reduce(output) must equal the fake DOM and ValidateDir must pass; stale or malformed paths fuzzed; no event sequence lets a password value past the reducer.
- **PQ-5** [Minor] `operating-envelope-latency` Serial CDP calls (getResponseBody, callFunctionOn per sheet change, screenshots) on the recorder goroutine have no latency budget for live record emission
- **PQ-6** [Minor] `unbounded-growth` Id map kept for the whole recording grows with every node ever seen; no bound stated
- **PQ-7** [Minor] `secret-leak-path` A value recorded while an input was type=text, before it flips to password, is already recorded; say whether that is accepted

## Round 2 — 2026-10-12T01:01:13-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Seam stated (recorder on the CDP side, Browser unchanged, browser.go comment corrected); Launch signature change and Output wiring specified.
- PQ-2 — addressed — FrameWriter is a consumer-side interface in package recorder; contract package untouched.
- PQ-3 — addressed — Conn sequence numbering as the discriminator (Seq < reply number dropped); policies for unknown node, detach, and pending resync, with injected-ordering tests.
- PQ-4 — addressed — Property test vs fake DOM plus ValidateDir; fuzz target for the path resolver; planted-secret property test.
- PQ-5 — addressed — Side workers with bounded queues; 250 ms p95 budget measured in M3.
- PQ-6 — addressed — Live nodes plus a 50k graveyard per session, dropped on document replacement.
- PQ-7 — addressed — Explicitly accepted, with the reason.

## Round 3 — 2026-10-12T01:01:58-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Seam stated in "Plug-in (M2)": recorder inside *Chrome, Browser unchanged, browser.go comment corrected; Launch takes recorder.Output.
- PQ-2 — addressed — FrameWriter is a consumer-side interface in package recorder, satisfied by *recording.DirWriter.
- PQ-3 — addressed — Ordering section: Conn sequence numbers, snapshot reply discriminator, session/frame detach and unknown-node cases with sequence tests.
- PQ-4 — addressed — One strategy line each for mirror translate (property), path resolver (fuzz), password rules (property with planted secrets), segments.
- PQ-5 — addressed — Side workers for asset bodies and screenshots with bounded queues; 250 ms p95 budget measured in M3.
- PQ-6 — addressed — Live nodes plus a 50 000-entry graveyard per session, dropped on document replacement; counter never reused.
- PQ-7 — addressed — Explicitly accepted: a value typed while type=text is recorded and not erased on the type flip.

## Open findings

(none — every finding has been disposed)
