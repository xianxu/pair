---
gate: boundary-review
issue: 422
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-09T23:35:12-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: bin/couch-dev duplicates pair-dev's symlink-resolving loop verbatim (ARCH-DRY)
          detail: bin/couch-dev:17-23 copies bin/pair-dev:23-29. These two are the only instances in the window. A resolve_here helper in bin/lib/ would make it one copy.
          family: launcher-script-shared-helpers
          round: 1
        - id: BR-2
          severity: Minor
          title: bin/lib/dev-rebuild.sh header and messages still name only pair-dev as the caller
          detail: 'Instances: the header comment (pair-dev then bin/pair is described as the caller), and the three echo lines prefixed "pair-dev:" with a "fix, then Alt+n" hint. couch-dev now calls the hook too, so the prefix and hint are misleading on a Couch launch.'
          family: shared-hook-doc-names-single-caller
          round: 1
      recipe: small-diff-review
      reviewed: 9e1c0d0040b505f285a320dd6d156af464622aee
      blocked: false
    - "n": 2
      timestamp: "2026-10-09T23:41:20-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: bin/couch-dev:17-23 still copies bin/pair-dev:23-29 verbatim; no bin/lib resolve helper landed. Minor, non-blocking.
          round: 2
        - id: BR-2
          disposition: addressed
          note: '5924e039/4d898f76: header names both callers; all three echo lines now "dev-rebuild:" with a "relaunch" hint; grep shows no consumer of the old prefix.'
          round: 2
      recipe: small-diff-review
      reviewed: c2f5ce78b305ee9d7c676b8208ab54e3152959f4
      blocked: false
---

# Gate ledger — pair#422 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-09T23:35:12-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `launcher-script-shared-helpers` bin/couch-dev duplicates pair-dev's symlink-resolving loop verbatim (ARCH-DRY)
  bin/couch-dev:17-23 copies bin/pair-dev:23-29. These two are the only instances in the window. A resolve_here helper in bin/lib/ would make it one copy.
- **BR-2** [Minor] `shared-hook-doc-names-single-caller` bin/lib/dev-rebuild.sh header and messages still name only pair-dev as the caller
  Instances: the header comment (pair-dev then bin/pair is described as the caller), and the three echo lines prefixed "pair-dev:" with a "fix, then Alt+n" hint. couch-dev now calls the hook too, so the prefix and hint are misleading on a Couch launch.

## Round 2 — 2026-10-09T23:41:20-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — bin/couch-dev:17-23 still copies bin/pair-dev:23-29 verbatim; no bin/lib resolve helper landed. Minor, non-blocking.
- BR-2 — addressed — 5924e039/4d898f76: header names both callers; all three echo lines now "dev-rebuild:" with a "relaunch" hint; grep shows no consumer of the old prefix.

## Open findings

- **BR-1** [Minor] `launcher-script-shared-helpers` bin/couch-dev duplicates pair-dev's symlink-resolving loop verbatim (ARCH-DRY)
