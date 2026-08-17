---
gate: plan-quality
issue: 131
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-16T20:16:28-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Release tag transaction omits the Pair commit that must be tagged.
          detail: The plan edits CHANGELOG.md at workshop/plans/000131-homebrew-formula-stale-cmd-list-plan.md:50, then tags and pushes v1.24 at line 52, but never says to commit/push the Pair repo state or verify that the tag points at the commit containing the changelog and current single-binary source. That is a hard-to-reverse publication step and under-delivers the issue's release purpose (ARCH-PURPOSE).
          round: 1
        - id: PQ-2
          severity: Important
          title: Required clean Homebrew verification is made optional.
          detail: The issue's Done when requires a clean-prefix source build/install and installed binary smoke checks at workshop/issues/000131-homebrew-formula-stale-cmd-list.md:75, but the separate plan only says to run a source-build install/test "when practical" at workshop/plans/000131-homebrew-formula-stale-cmd-list-plan.md:54. The plan should either require that evidence or name the exact fallback evidence and why it is acceptable.
          round: 1
        - id: PQ-3
          severity: Important
          title: Test surface is not stated in the required shape.
          detail: The plan lists verification commands but does not name the functions being unit-tested or explicitly state that this release/formula change adds no new pure functions and is covered by live formula validation instead. Compress this to the risky surfaces, for example PairFormula#install/Homebrew test block via clean-prefix brew install, and existing Go source via the repo's named build/test command.
          round: 1
        - id: PQ-4
          severity: Minor
          title: Non-goals are missing.
          detail: 'The plan has future extensions, but no explicit non-goals. Add what this release deliberately will not do, such as bottles, release automation, preserving old helper binaries, or solving #132 beyond correcting the stale caveats line.'
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-16T20:17:34-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: The plan now commits and pushes the Pair release commit before tagging, then verifies v1.24 points at that exact commit.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Clean-prefix Homebrew source install/test is now required, with stop-and-report behavior if the environment cannot provide it.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: 'The plan states no new pure functions are added and names the risky surfaces: PairFormula#install, formula test do, and existing Go source via go test ./...'
          round: 2
        - id: PQ-4
          disposition: addressed
          note: 'Non-goals now explicitly exclude bottles, old helper binaries, solving #132 beyond caveats wording, and release automation.'
          round: 2
      blocked: false
content_hash: 3decc5306a174d87f98143f60bbda9336e3f491027eb76c827e6648dd67434b1
---

# Gate ledger — pair#131 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-16T20:16:28-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] Release tag transaction omits the Pair commit that must be tagged.
  The plan edits CHANGELOG.md at workshop/plans/000131-homebrew-formula-stale-cmd-list-plan.md:50, then tags and pushes v1.24 at line 52, but never says to commit/push the Pair repo state or verify that the tag points at the commit containing the changelog and current single-binary source. That is a hard-to-reverse publication step and under-delivers the issue's release purpose (ARCH-PURPOSE).
- **PQ-2** [Important] Required clean Homebrew verification is made optional.
  The issue's Done when requires a clean-prefix source build/install and installed binary smoke checks at workshop/issues/000131-homebrew-formula-stale-cmd-list.md:75, but the separate plan only says to run a source-build install/test "when practical" at workshop/plans/000131-homebrew-formula-stale-cmd-list-plan.md:54. The plan should either require that evidence or name the exact fallback evidence and why it is acceptable.
- **PQ-3** [Important] Test surface is not stated in the required shape.
  The plan lists verification commands but does not name the functions being unit-tested or explicitly state that this release/formula change adds no new pure functions and is covered by live formula validation instead. Compress this to the risky surfaces, for example PairFormula#install/Homebrew test block via clean-prefix brew install, and existing Go source via the repo's named build/test command.
- **PQ-4** [Minor] Non-goals are missing.
  The plan has future extensions, but no explicit non-goals. Add what this release deliberately will not do, such as bottles, release automation, preserving old helper binaries, or solving #132 beyond correcting the stale caveats line.

## Round 2 — 2026-08-16T20:17:34-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — The plan now commits and pushes the Pair release commit before tagging, then verifies v1.24 points at that exact commit.
- PQ-2 — addressed — Clean-prefix Homebrew source install/test is now required, with stop-and-report behavior if the environment cannot provide it.
- PQ-3 — addressed — The plan states no new pure functions are added and names the risky surfaces: PairFormula#install, formula test do, and existing Go source via go test ./...
- PQ-4 — addressed — Non-goals now explicitly exclude bottles, old helper binaries, solving #132 beyond caveats wording, and release automation.

## Open findings

(none — every finding has been disposed)
