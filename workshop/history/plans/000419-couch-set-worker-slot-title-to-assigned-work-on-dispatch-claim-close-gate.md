---
gate: boundary-review
issue: 419
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-09T21:33:33-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: couch SKILL.md receiving-dispatched-work step hardcodes pair#N instead of repo#N
          detail: SKILL.md:120-123 gives the label template as 'pair#N <short title>', but the skill is loaded by Couch agents in every repo, and the Spec and Done-when say repo#N. Use <repo>#N in the template and keep pair#300 only as the example. This is the only instance in the window; atlas/couch.md:704 already says repo#N.
          family: shared-skill-repo-agnostic
          round: 1
        - id: BR-2
          severity: Minor
          title: Skill's publish-description command breaks a line inside the quoted argument
          detail: SKILL.md:121-122 wraps 'pair#N / <short title>' across a source line inside inline code. An agent copying the raw text could include a newline and indent in the label. Keep the command on one line.
          family: skill-command-copyable
          round: 1
        - id: BR-3
          severity: Minor
          title: Spec says nvim runs publish-description detached, but the !!/clear path runs it synchronously
          detail: nvim/init.lua:847-855 runs it with vim.system():wait() and discards stdout. The conclusion that nobody saw the dump still holds; only the stated reason is imprecise.
          family: doc-claim-from-memory
          round: 1
      recipe: small-diff-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-09T21:34:43-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: SKILL.md:119-125 now says "Name the issue's own repository" with ariadne#300 as the example; no pair#N template remains.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The command is now a single-line fenced sh block at SKILL.md:124-126.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Spec now reads "detached for ! tags, waited for !! and clear", which matches nvim/init.lua.
          round: 2
      recipe: small-diff-review
      blocked: false
---

# Gate ledger — pair#419 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-09T21:33:33-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `shared-skill-repo-agnostic` couch SKILL.md receiving-dispatched-work step hardcodes pair#N instead of repo#N
  SKILL.md:120-123 gives the label template as 'pair#N <short title>', but the skill is loaded by Couch agents in every repo, and the Spec and Done-when say repo#N. Use <repo>#N in the template and keep pair#300 only as the example. This is the only instance in the window; atlas/couch.md:704 already says repo#N.
- **BR-2** [Minor] `skill-command-copyable` Skill's publish-description command breaks a line inside the quoted argument
  SKILL.md:121-122 wraps 'pair#N / <short title>' across a source line inside inline code. An agent copying the raw text could include a newline and indent in the label. Keep the command on one line.
- **BR-3** [Minor] `doc-claim-from-memory` Spec says nvim runs publish-description detached, but the !!/clear path runs it synchronously
  nvim/init.lua:847-855 runs it with vim.system():wait() and discards stdout. The conclusion that nobody saw the dump still holds; only the stated reason is imprecise.

## Round 2 — 2026-10-09T21:34:43-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — SKILL.md:119-125 now says "Name the issue's own repository" with ariadne#300 as the example; no pair#N template remains.
- BR-2 — addressed — The command is now a single-line fenced sh block at SKILL.md:124-126.
- BR-3 — addressed — Spec now reads "detached for ! tags, waited for !! and clear", which matches nvim/init.lua.

## Open findings

(none — every finding has been disposed)
