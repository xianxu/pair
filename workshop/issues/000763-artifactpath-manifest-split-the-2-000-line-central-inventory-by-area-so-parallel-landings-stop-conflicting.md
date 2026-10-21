---
id: 000763
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'ee726e6ca790898f2592232f980675e022bc6df5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T01:47:34-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "fe4788a5", done: "6618e2de"}
---

# artifactpath manifest: split the 2,000-line central inventory by area so parallel landings stop conflicting

## Problem

## Spec

Per the TL's Revisions below, split the Go-reader audit, not artifactpath/manifest.go.
Store one versioned JSON certificate per reader at tests/affected-readers/<reader-path>.json.
Each retains its explicit disposition and exact SHA256; discovery and SOURCE_GUARDS
coverage remain unchanged. --record and --remove modify only the named certificate.
make affected-readers READER=<path> DISPOSITION=<audited-kind> regenerates that one
certificate after explicit re-audit; it never bulk-accepts changed reader inputs.
Reject malformed/duplicate keys, invalid certificate paths/schema/digests and missing,
changed or removed readers. Preserve unknown certificate fields when recording again.
Migrate existing dispositions and hashes exactly; no legacy aggregate remains writable.
ARCH-DRY/PURE: reconstruct the existing audit value in one loader; retain the auditor.
ARCH-EVOLVE: preserve unknown JSON fields; strict known fields and duplicate keys.
ARCH-FUNERAL: certificate lifetime is its reader's lifetime; explicit --remove retires it.
The batch tool scans tracked test sources once and loads one certificate per reader;
no runtime, background process or new user-action state is introduced.

## Done when

- Different reader edits touch different certificates and merge cleanly in a real Git two-branch regression.
- Migration preserves every old path/disposition/hash; stale, missing, malformed and source-scanner coverage controls still refuse.
- Named regeneration preserves unknown fields, cannot waive SOURCE_GUARDS, and never refreshes unrelated changed hashes.
- make affected-readers and atlas/README explain the audited per-reader repair; targeted tests, recorded suite and close review pass.

## Plan

- [x] Add failing per-reader loader/CLI, regeneration-isolation and real-Git merge regressions.
- [x] Migrate certificates exactly, retain audit semantics and wire named Make regeneration.
- [ ] Verify negative controls, document ownership, record suite, close and land.

## Log

### 2026-10-21

## Revisions
- 10-21 01:50 (TL ops:0): **wrong file named at filing.** pair:4 reports the actual conflicts are in
  `tests/affected-go-readers.json` (473 lines, 26 main commits in two days); `manifest.go` auto-merged both times.
  That file maps each dynamic-reader test file to a content hash, one line per file in a single JSON object, so any
  two branches touching neighbouring test files collide on adjacent lines, and every resolution costs a full suite
  plus a fix review. **Re-scoped:** make the reader audit conflict-free by construction: one small file per reader
  (e.g. `tests/affected-readers/<path>.sha256`) or another layout where independent changes never touch adjacent
  lines, with a regenerate target (`make affected-readers`) so resolving is mechanical. Same audit semantics: a
  changed reader still requires re-audit. Keep `manifest.go` as is unless it also shows real conflicts.
  Done when: two branches changing different reader tests merge without conflict (test it with git), the audit check
  still fails on a stale or missing hash, and atlas says where the entries live.

- 2026-10-21 — Design checkpoint: current Spec/Done when/Plan supersede the initial manifest.go filing, as directed by the TL revision. Inspected main local-gates-guard protected patterns: tests/** and Makefile.local are not covered; planned change touches neither .sdlc nor workflows. Keep the existing protected list unchanged. Quick-flow plan posted to hub, proceeding unless objection.

### Implementation checkpoint

CLI PASS10/3.317s after layout regressions failed against the aggregate reader. Real Git branches changing adjacent reader tests merge cleanly with exact certificate bytes and final audit. Make target regression RED (target absent), now GREEN. Exact old/new reconstructed audit equivalence PASS462 paths/dispositions/digests. Named update leaves unrelated changed reader unaccepted; duplicate/schema/path/symlink/legacy refusals, unknown nested integer roundtrip and SOURCE_GUARDS fences retained. Affected runner12 PASS7.411s. Full18-test inventory with actual affected seeded violation still running; suite/review/CI/landing pending. No runtime code or guard configuration changed.
