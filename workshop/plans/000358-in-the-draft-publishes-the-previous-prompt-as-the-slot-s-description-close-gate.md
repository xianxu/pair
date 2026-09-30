---
gate: boundary-review
issue: 358
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-30T11:56:31-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: README still says "There is no `!!` escape" right after documenting `!!`
          detail: 'README.md:273. The sentence meant "no way to send a literal leading `!`". Next to the new `!!` paragraph it reads as a contradiction. Reword it. This is the only instance in the window: the atlas bullets are consistent.'
          family: doc-claim-stale-after-change
          round: 1
        - id: BR-2
          severity: Minor
          title: Couch publish argv built twice and normalization.lua loaded with dofile twice in init.lua (ARCH-DRY)
          detail: init.lua:803 and init.lua:826 build the same publish-description argv. init.lua:790 and init.lua:1008 each dofile normalization.lua. Extract one argv helper and hoist a single normalization load above both users.
          family: shared-helper-not-extracted
          round: 1
        - id: BR-3
          severity: Minor
          title: The `!!` publish's ENOENT and timeout paths have no integration case
          detail: Only describe-nonzero exercises a failure. A describe-missing case (PATH without couch) and a timeout case would pin the pcall branch and the 5 s bound. The missing and slow cases exist only for `!`.
          family: failure-path-untested
          round: 1
        - id: BR-4
          severity: Minor
          title: one_line drops invalid UTF-8 bytes when it cuts, and can leave a trailing space before the ellipsis
          detail: nvim/bang_tag.lua:20. The char pattern skips 0xC0/0xC1/0xF5-0xFF lead bytes and stray continuation bytes. A cut at a space yields "word …". Cosmetic.
          family: utf8-truncation-edges
          round: 1
      recipe: small-diff-review
      blocked: false
---

# Gate ledger — pair#358 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-30T11:56:31-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `doc-claim-stale-after-change` README still says "There is no `!!` escape" right after documenting `!!`
  README.md:273. The sentence meant "no way to send a literal leading `!`". Next to the new `!!` paragraph it reads as a contradiction. Reword it. This is the only instance in the window: the atlas bullets are consistent.
- **BR-2** [Minor] `shared-helper-not-extracted` Couch publish argv built twice and normalization.lua loaded with dofile twice in init.lua (ARCH-DRY)
  init.lua:803 and init.lua:826 build the same publish-description argv. init.lua:790 and init.lua:1008 each dofile normalization.lua. Extract one argv helper and hoist a single normalization load above both users.
- **BR-3** [Minor] `failure-path-untested` The `!!` publish's ENOENT and timeout paths have no integration case
  Only describe-nonzero exercises a failure. A describe-missing case (PATH without couch) and a timeout case would pin the pcall branch and the 5 s bound. The missing and slow cases exist only for `!`.
- **BR-4** [Minor] `utf8-truncation-edges` one_line drops invalid UTF-8 bytes when it cuts, and can leave a trailing space before the ellipsis
  nvim/bang_tag.lua:20. The char pattern skips 0xC0/0xC1/0xF5-0xFF lead bytes and stray continuation bytes. A cut at a space yields "word …". Cosmetic.

## Open findings

- **BR-1** [Minor] `doc-claim-stale-after-change` README still says "There is no `!!` escape" right after documenting `!!`
- **BR-2** [Minor] `shared-helper-not-extracted` Couch publish argv built twice and normalization.lua loaded with dofile twice in init.lua (ARCH-DRY)
- **BR-3** [Minor] `failure-path-untested` The `!!` publish's ENOENT and timeout paths have no integration case
- **BR-4** [Minor] `utf8-truncation-edges` one_line drops invalid UTF-8 bytes when it cuts, and can leave a trailing space before the ellipsis
