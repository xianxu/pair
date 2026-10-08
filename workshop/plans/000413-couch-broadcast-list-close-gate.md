---
gate: boundary-review
issue: 413
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-08T10:15:01-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: README couch command list and broadcast section omit couch --broadcast-list [--json]
          detail: README.md:378-395 enumerates every couch CLI flag (incl. --actors, --peek) and README.md:808 documents broadcasting; neither mentions the new command or that it never prints the link.
          family: readme-tracks-cli-surface
          round: 1
        - id: BR-2
          severity: Minor
          title: Spec lists broadcast states as starting/live/ending; code and atlas use stopping
          family: spec-matches-implementation
          round: 1
        - id: BR-3
          severity: Minor
          title: The predates-couch hint in broadcast_list.go never fires for a real pre-#413 couch
          detail: An older couch answers invalid-request "operation requires the calling conversation identity", not "unknown message operation", so the user sees the raw error instead of the restart hint; untested.
          family: version-skew-detection
          round: 1
        - id: BR-4
          severity: Minor
          title: handleBroadcastStatus calls Hub.Viewers without the request context
          detail: The transport contract says handlers honor their deadline; h.do has no ctx case. Bounded in practice because the hub loop never blocks.
          family: handler-honors-context
          round: 1
        - id: BR-5
          severity: Minor
          title: Mode derives from a LocalOnly type assertion with no local-only test; hub test never asserts the slow viewer resynced
          family: test-asserts-precondition
          round: 1
      recipe: milestone-review
      blocked: true
---

# Gate ledger — pair#413 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-08T10:15:01-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `readme-tracks-cli-surface` README couch command list and broadcast section omit couch --broadcast-list [--json]
  README.md:378-395 enumerates every couch CLI flag (incl. --actors, --peek) and README.md:808 documents broadcasting; neither mentions the new command or that it never prints the link.
- **BR-2** [Minor] `spec-matches-implementation` Spec lists broadcast states as starting/live/ending; code and atlas use stopping
- **BR-3** [Minor] `version-skew-detection` The predates-couch hint in broadcast_list.go never fires for a real pre-#413 couch
  An older couch answers invalid-request "operation requires the calling conversation identity", not "unknown message operation", so the user sees the raw error instead of the restart hint; untested.
- **BR-4** [Minor] `handler-honors-context` handleBroadcastStatus calls Hub.Viewers without the request context
  The transport contract says handlers honor their deadline; h.do has no ctx case. Bounded in practice because the hub loop never blocks.
- **BR-5** [Minor] `test-asserts-precondition` Mode derives from a LocalOnly type assertion with no local-only test; hub test never asserts the slow viewer resynced

## Open findings

- **BR-1** [Important] `readme-tracks-cli-surface` README couch command list and broadcast section omit couch --broadcast-list [--json]
- **BR-2** [Minor] `spec-matches-implementation` Spec lists broadcast states as starting/live/ending; code and atlas use stopping
- **BR-3** [Minor] `version-skew-detection` The predates-couch hint in broadcast_list.go never fires for a real pre-#413 couch
- **BR-4** [Minor] `handler-honors-context` handleBroadcastStatus calls Hub.Viewers without the request context
- **BR-5** [Minor] `test-asserts-precondition` Mode derives from a LocalOnly type assertion with no local-only test; hub test never asserts the slow viewer resynced
