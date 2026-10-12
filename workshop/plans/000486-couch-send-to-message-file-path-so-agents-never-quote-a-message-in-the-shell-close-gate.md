---
gate: boundary-review
issue: 486
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T08:34:00-07:00"
      dispatched: "2026-10-12T08:30:48-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: README.md send-to usage lines omit the new --message-file PATH flag
          detail: README.md:435-438 lists only --message TEXT forms. run.go:1330 help, SKILL.md and atlas/couch.md:186 were updated; README is the one place restating the surface that was not. Add a --message-file PATH usage line.
          family: docs-surface-sync
          round: 1
        - id: BR-2
          severity: Minor
          title: CRLF message file leaves a trailing CR that the broker refuses with a generic error
          detail: readMessageFile strips only one trailing newline, so a CRLF file keeps a CR and the broker's control-byte check refuses it without naming the file. Strip CRLF too, or name CR in the --message-file error.
          family: message-file-line-endings
          round: 1
        - id: BR-3
          severity: Minor
          title: Live sandboxed-slot Done-when check deferred to the TL
          detail: The issue log records the deferral, which needs a Couch restart. The live check (backticks, command and process substitution, newlines via --message-file, versus --message staying sandboxed) is not verified at this boundary.
          family: live-done-when-deferred
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T08:38:50-07:00"
      dispatched: "2026-10-12T08:36:51-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: README.md:439-441 adds the --message-file PATH usage; help (run.go:1330), SKILL.md, atlas/couch.md:186 and README now all list it.
          round: 2
        - id: BR-2
          disposition: addressed
          note: messages.go:140-144 strips a trailing CRLF; the CRLF test at messages_test.go:203-208 fails when the fix is reverted (checked in the review checkout).
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: The live sandboxed-slot check is still deferred to the TL (issue Log, 2026-10-12); no live evidence at this boundary. Minor, does not block.
          round: 2
      findings:
        - id: BR-4
          severity: Minor
          title: A file with CRLF line endings throughout keeps its inner CRs, which the broker refuses without naming the file
          detail: '2nd finding in this family. readMessageFile (messages.go:140-144) strips only the final CRLF, so "a\r\nb\r\n" sends "a\r\nb" and the broker''s control-byte check refuses it with a generic error. Rule: apply one line-ending policy to the whole file, not just its end. Either turn every \r\n into \n, or refuse any CR with a --message-file error that names the file. Count in this window: 1 site (readMessageFile).'
          family: message-file-line-endings
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: 706d4e020e28799d035b482c5af04939e09c226b
      blocked: false
---

# Gate ledger — pair#486 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T08:34:00-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `docs-surface-sync` README.md send-to usage lines omit the new --message-file PATH flag
  README.md:435-438 lists only --message TEXT forms. run.go:1330 help, SKILL.md and atlas/couch.md:186 were updated; README is the one place restating the surface that was not. Add a --message-file PATH usage line.
- **BR-2** [Minor] `message-file-line-endings` CRLF message file leaves a trailing CR that the broker refuses with a generic error
  readMessageFile strips only one trailing newline, so a CRLF file keeps a CR and the broker's control-byte check refuses it without naming the file. Strip CRLF too, or name CR in the --message-file error.
- **BR-3** [Minor] `live-done-when-deferred` Live sandboxed-slot Done-when check deferred to the TL
  The issue log records the deferral, which needs a Couch restart. The live check (backticks, command and process substitution, newlines via --message-file, versus --message staying sandboxed) is not verified at this boundary.

## Round 2 — 2026-10-12T08:38:50-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — README.md:439-441 adds the --message-file PATH usage; help (run.go:1330), SKILL.md, atlas/couch.md:186 and README now all list it.
- BR-2 — addressed — messages.go:140-144 strips a trailing CRLF; the CRLF test at messages_test.go:203-208 fails when the fix is reverted (checked in the review checkout).
- BR-3 — not-addressed — The live sandboxed-slot check is still deferred to the TL (issue Log, 2026-10-12); no live evidence at this boundary. Minor, does not block.

### Raised

- **BR-4** [Minor] `message-file-line-endings` A file with CRLF line endings throughout keeps its inner CRs, which the broker refuses without naming the file
  2nd finding in this family. readMessageFile (messages.go:140-144) strips only the final CRLF, so "a\r\nb\r\n" sends "a\r\nb" and the broker's control-byte check refuses it with a generic error. Rule: apply one line-ending policy to the whole file, not just its end. Either turn every \r\n into \n, or refuse any CR with a --message-file error that names the file. Count in this window: 1 site (readMessageFile).

## Open findings

- **BR-3** [Minor] `live-done-when-deferred` Live sandboxed-slot Done-when check deferred to the TL
- **BR-4** [Minor] `message-file-line-endings` A file with CRLF line endings throughout keeps its inner CRs, which the broker refuses without naming the file
