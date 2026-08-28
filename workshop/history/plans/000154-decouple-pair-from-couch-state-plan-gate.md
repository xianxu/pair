---
gate: plan-quality
issue: 154
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-27T18:27:57-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Critical
          title: The plan misses the existing tag-normalization seam required for exact equality
          detail: 'ARCH-PURPOSE: valid tag pair-demo is accepted by ValidatePairTag but ResumeTagFromArg passes it through NormalizeTag, which strips pair-. Revise the plan to change this parse contract and its existing tests, explicitly deciding that legacy pair- aliasing is removed.'
          family: exact-tag-identity
          round: 1
        - id: PQ-2
          severity: Important
          title: The regression matrix does not prove that Pair never opens Couch state
          detail: 'ARCH-PURPOSE: output equivalence and byte-identical snapshots detect behavioral differences and writes, but an attempted read whose error is ignored still passes. Name an open-spy or equivalent audit guard that fails on every Couch-namespace filesystem access across the enumerated command families.'
          family: namespace-io-observability
          round: 1
        - id: PQ-3
          severity: Important
          title: Task 3 requires Couch environment plumbing to be absent before Task 4 removes it
          detail: Task 3 expects no COUCH_STORE_DIR matches under cmd/internal/launcher, while runcli.go and runtime.go retain CouchStoreDir until Task 4. Move that removal earlier or narrow Task 3's verification to the read symbols actually removed there.
          family: staged-verification-consistency
          round: 1
        - id: PQ-4
          severity: Important
          title: The test plan must be compressed from case inventories to risky-function strategies
          detail: Tasks 1, 2, and 5 enumerate individual fixtures and assertions. Replace those inventories with each risky production function by name and one line stating its adversarial input class and mechanical guard, while retaining the required behavioral classes in the Spec and Done-when.
          family: test-plan-compression
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-27T18:32:35-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: ResumeTagFromArg now uses exact ValidatePairTag semantics, updates the parse test, and explicitly removes legacy pair-prefix aliasing.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: The public-entry matrix uses an unread manifest FIFO as an open tripwire and namespace snapshots across every specified command family.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Task 3 checks only deleted read symbols and explicitly defers CouchStoreDir removal and verification to Task 4.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Test work is expressed as named risky production functions with adversarial input classes and mechanical guards.
          round: 2
      blocked: false
content_hash: 5b5e9bb594ab580e7670cb8638fe765563c02ca36bc2ccbc38f4651ab7062bec
---

# Gate ledger — pair#154 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-27T18:27:57-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Critical] `exact-tag-identity` The plan misses the existing tag-normalization seam required for exact equality
  ARCH-PURPOSE: valid tag pair-demo is accepted by ValidatePairTag but ResumeTagFromArg passes it through NormalizeTag, which strips pair-. Revise the plan to change this parse contract and its existing tests, explicitly deciding that legacy pair- aliasing is removed.
- **PQ-2** [Important] `namespace-io-observability` The regression matrix does not prove that Pair never opens Couch state
  ARCH-PURPOSE: output equivalence and byte-identical snapshots detect behavioral differences and writes, but an attempted read whose error is ignored still passes. Name an open-spy or equivalent audit guard that fails on every Couch-namespace filesystem access across the enumerated command families.
- **PQ-3** [Important] `staged-verification-consistency` Task 3 requires Couch environment plumbing to be absent before Task 4 removes it
  Task 3 expects no COUCH_STORE_DIR matches under cmd/internal/launcher, while runcli.go and runtime.go retain CouchStoreDir until Task 4. Move that removal earlier or narrow Task 3's verification to the read symbols actually removed there.
- **PQ-4** [Important] `test-plan-compression` The test plan must be compressed from case inventories to risky-function strategies
  Tasks 1, 2, and 5 enumerate individual fixtures and assertions. Replace those inventories with each risky production function by name and one line stating its adversarial input class and mechanical guard, while retaining the required behavioral classes in the Spec and Done-when.

## Round 2 — 2026-08-27T18:32:35-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — ResumeTagFromArg now uses exact ValidatePairTag semantics, updates the parse test, and explicitly removes legacy pair-prefix aliasing.
- PQ-2 — addressed — The public-entry matrix uses an unread manifest FIFO as an open tripwire and namespace snapshots across every specified command family.
- PQ-3 — addressed — Task 3 checks only deleted read symbols and explicitly defers CouchStoreDir removal and verification to Task 4.
- PQ-4 — addressed — Test work is expressed as named risky production functions with adversarial input classes and mechanical guards.

## Open findings

(none — every finding has been disposed)
