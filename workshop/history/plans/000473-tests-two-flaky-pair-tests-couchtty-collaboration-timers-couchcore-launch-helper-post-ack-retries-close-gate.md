---
gate: boundary-review
issue: 473
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T03:52:56-07:00"
      dispatched: "2026-10-12T03:51:46-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Done-when "repeated full-suite runs" has no evidence in the Log
          detail: The Done-when requires 200 -race runs AND repeated full-suite runs. The Log Proof entry records only the 200 -race runs, yet Plan item 3 is checked. The couchcore flake was only ever seen in a full run. Run make -k test several times and log the count and result. This is the only instance in the window.
          family: done-when-clause-unevidenced
          round: 1
        - id: BR-2
          severity: Minor
          title: Couch.after defaults at its use while Couch.sleep defaults in the constructor
          detail: sleep is set to time.Sleep in the constructor (couch.go:197) and also nil-checked at use (couch.go:958). after is only nil-checked in newHandleCleanup (couch.go:973). Pick one idiom for injected clocks.
          family: injected-seam-default-idiom
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T03:54:32-07:00"
      dispatched: "2026-10-12T03:53:38-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Log now records three `go test ./...` runs plus `make -k test` (exit 0), with the four target tests green in every run.
          round: 2
        - id: BR-2
          disposition: addressed
          note: after is set to time.After in New (couch.go:198) and nil-checked at use (couch.go:974), matching sleep (couch.go:197, 959).
          round: 2
      findings:
        - id: BR-3
          severity: Minor
          title: Full-suite Log entry says each Go run had an unrelated failure, then says run 2 was fully green
          detail: 'Issue Log, Full-suite runs bullet: change to "Runs 1 and 3 each had one unrelated failure". Only instance in the window.'
          family: doc-claim-unverified-count
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: 6ac70d87b1b177ea3849f16b118b8117de9f6e1f
      blocked: false
---

# Gate ledger — pair#473 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T03:52:56-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `done-when-clause-unevidenced` Done-when "repeated full-suite runs" has no evidence in the Log
  The Done-when requires 200 -race runs AND repeated full-suite runs. The Log Proof entry records only the 200 -race runs, yet Plan item 3 is checked. The couchcore flake was only ever seen in a full run. Run make -k test several times and log the count and result. This is the only instance in the window.
- **BR-2** [Minor] `injected-seam-default-idiom` Couch.after defaults at its use while Couch.sleep defaults in the constructor
  sleep is set to time.Sleep in the constructor (couch.go:197) and also nil-checked at use (couch.go:958). after is only nil-checked in newHandleCleanup (couch.go:973). Pick one idiom for injected clocks.

## Round 2 — 2026-10-12T03:54:32-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Log now records three `go test ./...` runs plus `make -k test` (exit 0), with the four target tests green in every run.
- BR-2 — addressed — after is set to time.After in New (couch.go:198) and nil-checked at use (couch.go:974), matching sleep (couch.go:197, 959).

### Raised

- **BR-3** [Minor] `doc-claim-unverified-count` Full-suite Log entry says each Go run had an unrelated failure, then says run 2 was fully green
  Issue Log, Full-suite runs bullet: change to "Runs 1 and 3 each had one unrelated failure". Only instance in the window.

## Open findings

- **BR-3** [Minor] `doc-claim-unverified-count` Full-suite Log entry says each Go run had an unrelated failure, then says run 2 was fully green
