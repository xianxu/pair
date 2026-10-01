---
gate: boundary-review
issue: 372
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-01T16:00:28-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: slugline doc claims to be "the one definition" of the slug format while nvim/slug.lua independently parses the fence
          detail: nvim/slug.lua:11-15 (plus the runtimebundle copy) re-implements the === L | R === recognition; reword the package doc to say it is the Go definition and nvim/slug.lua mirrors it.
          family: single-source-claim-overstated
          round: 1
        - id: BR-2
          severity: Minor
          title: Plan names slugline.Unfence; the exported function is Unfenced
          detail: Add a Revisions line so the plan matches the code.
          family: plan-code-name-drift
          round: 1
        - id: BR-3
          severity: Minor
          title: slugline.go const close shadows Go's built-in close
          family: builtin-shadowing
          round: 1
        - id: BR-4
          severity: Minor
          title: ApplySlugs runs on start/slot inventory calls that never show slugs
          detail: couch.go:206,265 and slotstart.go:179,322 now do one bounded read per live row; negligible cost, but the work is unused there.
          family: display-only-work-on-non-display-paths
          round: 1
      recipe: milestone-review
      blocked: false
    - "n": 2
      timestamp: "2026-10-01T16:21:26-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: slugline.go:1-4 now says "the Go definition" and names nvim/slug.lua as the mirror (verified nvim/slug.lua exists); lessons.md adds the rule.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Issue Revisions entry (2026-10-01) records that Unfence shipped as Unfenced; slugline.go:37 confirms the name.
          round: 2
        - id: BR-3
          disposition: addressed
          note: consts renamed to fenceOpen/fenceClose (slugline.go:13-15); no other built-in names in the new files; slugline and slugcmd tests pass.
          round: 2
        - id: BR-4
          disposition: withdrawn
          note: Kept on purpose and recorded in the issue Revisions; one capped read per live row on paths that are not hot.
          round: 2
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#372 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-01T16:00:28-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `single-source-claim-overstated` slugline doc claims to be "the one definition" of the slug format while nvim/slug.lua independently parses the fence
  nvim/slug.lua:11-15 (plus the runtimebundle copy) re-implements the === L | R === recognition; reword the package doc to say it is the Go definition and nvim/slug.lua mirrors it.
- **BR-2** [Minor] `plan-code-name-drift` Plan names slugline.Unfence; the exported function is Unfenced
  Add a Revisions line so the plan matches the code.
- **BR-3** [Minor] `builtin-shadowing` slugline.go const close shadows Go's built-in close
- **BR-4** [Minor] `display-only-work-on-non-display-paths` ApplySlugs runs on start/slot inventory calls that never show slugs
  couch.go:206,265 and slotstart.go:179,322 now do one bounded read per live row; negligible cost, but the work is unused there.

## Round 2 — 2026-10-01T16:21:26-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — slugline.go:1-4 now says "the Go definition" and names nvim/slug.lua as the mirror (verified nvim/slug.lua exists); lessons.md adds the rule.
- BR-2 — addressed — Issue Revisions entry (2026-10-01) records that Unfence shipped as Unfenced; slugline.go:37 confirms the name.
- BR-3 — addressed — consts renamed to fenceOpen/fenceClose (slugline.go:13-15); no other built-in names in the new files; slugline and slugcmd tests pass.
- BR-4 — withdrawn — Kept on purpose and recorded in the issue Revisions; one capped read per live row on paths that are not hot.

## Open findings

(none — every finding has been disposed)
