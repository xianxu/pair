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

## Open findings

- **BR-1** [Minor] `single-source-claim-overstated` slugline doc claims to be "the one definition" of the slug format while nvim/slug.lua independently parses the fence
- **BR-2** [Minor] `plan-code-name-drift` Plan names slugline.Unfence; the exported function is Unfenced
- **BR-3** [Minor] `builtin-shadowing` slugline.go const close shadows Go's built-in close
- **BR-4** [Minor] `display-only-work-on-non-display-paths` ApplySlugs runs on start/slot inventory calls that never show slugs
