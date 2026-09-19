---
gate: plan-quality
issue: 292
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-19T12:55:30-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Task 2.5 misidentifies Ctrl+U; the Spec's "Ctrl+U clears it" is unimplemented
          detail: |-
            rename_input.go:32 maps \x1b[127;9u to RenameDeleteToStart, and
            rename_input_test.go:47 names it "super backspace" (Cmd+Backspace).
            Ctrl+U is \x15, which rename_input.go:124-129 swallows as RenameConsume,
            so the prefilled URL field cannot be cleared as the Spec promises. Add
            {"\x15", RenameDeleteToStart} in Task 1.5, or correct the Spec.
          family: unverified-existing-behavior
          round: 1
        - id: PQ-2
          severity: Important
          title: Record temp files land in the GC-inventoried browser dir with no removal path
          detail: |-
            Task 3.3 writes temps inside browser-<tag>/ and Task 3.2's member pattern
            does not cover them; storagegc/inventory.go:236-256 turns an unmatched
            path into a scope-wide blocker ("unrecognized entries in Pair root") --
            the same rule the plan cites for keeping profiles out of the data root.
            A SIGKILL mid-publish (a designed-for path here) strands one, and neither
            SweepDeadOwners nor storagegc removes it.
          family: artifact-residue-no-removal-path
          round: 1
        - id: PQ-3
          severity: Important
          title: No adversarial/fuzz strategy for the parsers that read untrusted input
          detail: |-
            DecodeRecord, ParseActivePort, NormalizeURL, ParseProfileName and the CDP
            message decode all read hand-editable or page-controlled bytes, but every
            planned test is a hand-enumerated case table, which is blind to the
            malformed class. Add one line of strategy per function (fuzz seeded with
            truncated/duplicate-key/oversized/non-loopback/control-char forms), and
            compress the line-numbered call-site inventories while there.
          family: adversarial-test-strategy
          round: 1
        - id: PQ-4
          severity: Minor
          title: Tab name/named has two writers, and the event carrying it can be dropped
          detail: |-
            browserController.Send drops on a full channel, including EvRenamed, while
            finishField sets t.named/t.name directly. State.Named feeds the record that
            pair#293's reuse rule reads, so the two can disagree. Name the authority
            and make the drop policy coalesce per kind rather than discard the newest.
          family: state-single-source
          round: 1
        - id: PQ-5
          severity: Minor
          title: Record store's scope/tag source and the no-tag case are unspecified
          detail: |-
            Task 3.3 uses artifactpath.ResolveScoped(scope, tag) without saying where
            pair term gets them (today: DataDirFromEnv + os.Getenv("PAIR_TAG"),
            run.go:731) or what publishing does when PAIR_TAG is empty -- the
            standalone-pair case the Spec explicitly supports.
          family: undefined-degraded-path
          round: 1
        - id: PQ-6
          severity: Minor
          title: New third-party dependency in a five-dep module, no alternative weighed
          detail: |-
            go.mod's five direct deps are all terminal-related; coder/websocket would
            be the first outside that. The plan states the choice (v1.8.15, ISC, zero
            transitive deps) but never compares it with a small in-tree client for the
            one browser endpoint. One line for the operator to accept or veto.
          family: dependency-justification
          round: 1
      blocked: true
---

# Gate ledger — pair#292 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-19T12:55:30-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `unverified-existing-behavior` Task 2.5 misidentifies Ctrl+U; the Spec's "Ctrl+U clears it" is unimplemented
  rename_input.go:32 maps \x1b[127;9u to RenameDeleteToStart, and
  rename_input_test.go:47 names it "super backspace" (Cmd+Backspace).
  Ctrl+U is \x15, which rename_input.go:124-129 swallows as RenameConsume,
  so the prefilled URL field cannot be cleared as the Spec promises. Add
  {"\x15", RenameDeleteToStart} in Task 1.5, or correct the Spec.
- **PQ-2** [Important] `artifact-residue-no-removal-path` Record temp files land in the GC-inventoried browser dir with no removal path
  Task 3.3 writes temps inside browser-<tag>/ and Task 3.2's member pattern
  does not cover them; storagegc/inventory.go:236-256 turns an unmatched
  path into a scope-wide blocker ("unrecognized entries in Pair root") --
  the same rule the plan cites for keeping profiles out of the data root.
  A SIGKILL mid-publish (a designed-for path here) strands one, and neither
  SweepDeadOwners nor storagegc removes it.
- **PQ-3** [Important] `adversarial-test-strategy` No adversarial/fuzz strategy for the parsers that read untrusted input
  DecodeRecord, ParseActivePort, NormalizeURL, ParseProfileName and the CDP
  message decode all read hand-editable or page-controlled bytes, but every
  planned test is a hand-enumerated case table, which is blind to the
  malformed class. Add one line of strategy per function (fuzz seeded with
  truncated/duplicate-key/oversized/non-loopback/control-char forms), and
  compress the line-numbered call-site inventories while there.
- **PQ-4** [Minor] `state-single-source` Tab name/named has two writers, and the event carrying it can be dropped
  browserController.Send drops on a full channel, including EvRenamed, while
  finishField sets t.named/t.name directly. State.Named feeds the record that
  pair#293's reuse rule reads, so the two can disagree. Name the authority
  and make the drop policy coalesce per kind rather than discard the newest.
- **PQ-5** [Minor] `undefined-degraded-path` Record store's scope/tag source and the no-tag case are unspecified
  Task 3.3 uses artifactpath.ResolveScoped(scope, tag) without saying where
  pair term gets them (today: DataDirFromEnv + os.Getenv("PAIR_TAG"),
  run.go:731) or what publishing does when PAIR_TAG is empty -- the
  standalone-pair case the Spec explicitly supports.
- **PQ-6** [Minor] `dependency-justification` New third-party dependency in a five-dep module, no alternative weighed
  go.mod's five direct deps are all terminal-related; coder/websocket would
  be the first outside that. The plan states the choice (v1.8.15, ISC, zero
  transitive deps) but never compares it with a small in-tree client for the
  one browser endpoint. One line for the operator to accept or veto.

## Open findings

- **PQ-1** [Important] `unverified-existing-behavior` Task 2.5 misidentifies Ctrl+U; the Spec's "Ctrl+U clears it" is unimplemented
- **PQ-2** [Important] `artifact-residue-no-removal-path` Record temp files land in the GC-inventoried browser dir with no removal path
- **PQ-3** [Important] `adversarial-test-strategy` No adversarial/fuzz strategy for the parsers that read untrusted input
- **PQ-4** [Minor] `state-single-source` Tab name/named has two writers, and the event carrying it can be dropped
- **PQ-5** [Minor] `undefined-degraded-path` Record store's scope/tag source and the no-tag case are unspecified
- **PQ-6** [Minor] `dependency-justification` New third-party dependency in a five-dep module, no alternative weighed
