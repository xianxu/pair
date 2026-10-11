---
gate: boundary-review
issue: 437
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T00:28:49-07:00"
      dispatched: "2026-10-11T00:25:48-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: release refuses unread edits telling the agent to pass --accept-edits, which release does not accept
          detail: Release calls guard(false) in sync.go, but parseSessionSyncCLI allows only --fold on release, so following the refusal is a usage error and the agent cannot release without writing. Accept --accept-edits on release (refreshing the shadow) or make the refusal text depend on the verb.
          family: refusal-names-unreachable-remedy
          round: 1
        - id: BR-2
          severity: Important
          title: ledger slotAddress regex is a second, narrower slot grammar; one out-of-grammar delivery disables status store-wide
          detail: ledger.go:54 allows only [A-Za-z0-9._-] families while couchcore.ParseWorkspaceReference/validFamily accept any UTF-8 family without separators; Append then fails, MarkFailed stamps recording.json, and every status refuses not-recording until restart. Validate with couchcore and encode the family into a safe filename (ARCH-DRY).
          family: single-source-slot-grammar
          round: 1
        - id: BR-3
          severity: Important
          title: a Couch restart clears Failed, so a lost ledger row silently reads as nothing owed
          detail: MarkRecording at startup overwrites Failed without recovering the missing row; files whose Since predates the failure report the gap as complete. Keep failure timestamps across restarts and refuse or warn for files whose horizon precedes the last failure (ARCH-ORDER).
          family: uncertainty-collapsed-to-absence
          round: 1
        - id: BR-4
          severity: Important
          title: README.md missing the new couch --session-sync verb
          detail: README.md:399-414 lists couch CLI verbs (--actors, --broadcast-list, --send-to); the new user-facing verb is absent there.
          family: docs-surface-update
          round: 1
        - id: BR-5
          severity: Minor
          title: Done-when row 5 (xx-tl switches to the verb) is unchecked and depends on ops
          detail: Get the TL's confirmation before close, or add a Revision moving it out of scope.
          family: done-when-traceability
          round: 1
        - id: BR-6
          severity: Minor
          title: a fresh file with no sidecar or shadow owes the slot's whole ledger history
          family: owed-horizon-default
          round: 1
        - id: BR-7
          severity: Minor
          title: writeAtomic is about the fifth copy of temp+rename in cmd/internal
          family: shared-atomic-write-helper
          round: 1
        - id: BR-8
          severity: Minor
          title: trim can stay above trimBytes with 512 long rows, rewriting the file on every append
          family: ledger-trim-thrash
          round: 1
        - id: BR-9
          severity: Minor
          title: Write writes the file before the shadow; a failed shadow write turns the agent's own write into unread edits
          family: write-order-self-edit
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-11T00:42:44-07:00"
      dispatched: "2026-10-11T00:40:54-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: parseSessionSyncCLI allows --accept-edits/--accept-gap on release; TestReleaseAcceptsReadEditsWithoutAWrite pins it.
          round: 2
        - id: BR-2
          disposition: addressed
          note: ValidSlot delegates to couchcore.ParseWorkspaceReference; QueryEscape filename; TestLedgerUsesCouchsSlotGrammar plus per-slot gap test.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Per-slot .gap survives recordDeliveries restart (TestALedgerAppendFailureRecordsAGapForThatSlotOnly); refuses delivery-gap until accepted per file.
          round: 2
        - id: BR-4
          disposition: addressed
          note: README.md:401 lists couch --session-sync.
          round: 2
        - id: BR-5
          disposition: addressed
          note: Round-1 Revision moves Done-when 5 to ops, citing hand-off delivery 3c7ff078.
          round: 2
        - id: BR-6
          disposition: not-addressed
          note: LoadState still returns zero Since with no sidecar or shadow, so a new file owes up to 14 days of rows; Minor, non-blocking.
          round: 2
        - id: BR-7
          disposition: not-addressed
          note: sessionsync/ledger.go:201 writeAtomic is still a local copy; durablefile.WriteAtomic is the shared helper to reuse.
          round: 2
        - id: BR-8
          disposition: addressed
          note: trim stops at trimBytes/2 newest-first; TestLedgerTrimEndsWellUnderTheBound.
          round: 2
        - id: BR-9
          disposition: not-addressed
          note: Write still writes the file before the shadow; Minor, non-blocking.
          round: 2
      findings:
        - id: BR-10
          severity: Minor
          title: readGap fabricates Last=time.Now() for an unparseable gap record, so it never expires and cannot be durably accepted
          detail: Each read gets a fresh Last, so GapAccepted never covers it and expireGap never removes it; status refuses delivery-gap forever until the file is deleted by hand. Use the record's mtime as First/Last (and l.now, not time.Now).
          family: fabricated-evidence-on-parse-failure
          round: 2
      recipe: milestone-review
      reviewed: b7c433831a5125b2d39992bc7ac223432a36c078
      blocked: false
---

# Gate ledger — pair#437 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T00:28:49-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `refusal-names-unreachable-remedy` release refuses unread edits telling the agent to pass --accept-edits, which release does not accept
  Release calls guard(false) in sync.go, but parseSessionSyncCLI allows only --fold on release, so following the refusal is a usage error and the agent cannot release without writing. Accept --accept-edits on release (refreshing the shadow) or make the refusal text depend on the verb.
- **BR-2** [Important] `single-source-slot-grammar` ledger slotAddress regex is a second, narrower slot grammar; one out-of-grammar delivery disables status store-wide
  ledger.go:54 allows only [A-Za-z0-9._-] families while couchcore.ParseWorkspaceReference/validFamily accept any UTF-8 family without separators; Append then fails, MarkFailed stamps recording.json, and every status refuses not-recording until restart. Validate with couchcore and encode the family into a safe filename (ARCH-DRY).
- **BR-3** [Important] `uncertainty-collapsed-to-absence` a Couch restart clears Failed, so a lost ledger row silently reads as nothing owed
  MarkRecording at startup overwrites Failed without recovering the missing row; files whose Since predates the failure report the gap as complete. Keep failure timestamps across restarts and refuse or warn for files whose horizon precedes the last failure (ARCH-ORDER).
- **BR-4** [Important] `docs-surface-update` README.md missing the new couch --session-sync verb
  README.md:399-414 lists couch CLI verbs (--actors, --broadcast-list, --send-to); the new user-facing verb is absent there.
- **BR-5** [Minor] `done-when-traceability` Done-when row 5 (xx-tl switches to the verb) is unchecked and depends on ops
  Get the TL's confirmation before close, or add a Revision moving it out of scope.
- **BR-6** [Minor] `owed-horizon-default` a fresh file with no sidecar or shadow owes the slot's whole ledger history
- **BR-7** [Minor] `shared-atomic-write-helper` writeAtomic is about the fifth copy of temp+rename in cmd/internal
- **BR-8** [Minor] `ledger-trim-thrash` trim can stay above trimBytes with 512 long rows, rewriting the file on every append
- **BR-9** [Minor] `write-order-self-edit` Write writes the file before the shadow; a failed shadow write turns the agent's own write into unread edits

## Round 2 — 2026-10-11T00:42:44-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — parseSessionSyncCLI allows --accept-edits/--accept-gap on release; TestReleaseAcceptsReadEditsWithoutAWrite pins it.
- BR-2 — addressed — ValidSlot delegates to couchcore.ParseWorkspaceReference; QueryEscape filename; TestLedgerUsesCouchsSlotGrammar plus per-slot gap test.
- BR-3 — addressed — Per-slot .gap survives recordDeliveries restart (TestALedgerAppendFailureRecordsAGapForThatSlotOnly); refuses delivery-gap until accepted per file.
- BR-4 — addressed — README.md:401 lists couch --session-sync.
- BR-5 — addressed — Round-1 Revision moves Done-when 5 to ops, citing hand-off delivery 3c7ff078.
- BR-6 — not-addressed — LoadState still returns zero Since with no sidecar or shadow, so a new file owes up to 14 days of rows; Minor, non-blocking.
- BR-7 — not-addressed — sessionsync/ledger.go:201 writeAtomic is still a local copy; durablefile.WriteAtomic is the shared helper to reuse.
- BR-8 — addressed — trim stops at trimBytes/2 newest-first; TestLedgerTrimEndsWellUnderTheBound.
- BR-9 — not-addressed — Write still writes the file before the shadow; Minor, non-blocking.

### Raised

- **BR-10** [Minor] `fabricated-evidence-on-parse-failure` readGap fabricates Last=time.Now() for an unparseable gap record, so it never expires and cannot be durably accepted
  Each read gets a fresh Last, so GapAccepted never covers it and expireGap never removes it; status refuses delivery-gap forever until the file is deleted by hand. Use the record's mtime as First/Last (and l.now, not time.Now).

## Open findings

- **BR-6** [Minor] `owed-horizon-default` a fresh file with no sidecar or shadow owes the slot's whole ledger history
- **BR-7** [Minor] `shared-atomic-write-helper` writeAtomic is about the fifth copy of temp+rename in cmd/internal
- **BR-9** [Minor] `write-order-self-edit` Write writes the file before the shadow; a failed shadow write turns the agent's own write into unread edits
- **BR-10** [Minor] `fabricated-evidence-on-parse-failure` readGap fabricates Last=time.Now() for an unparseable gap record, so it never expires and cannot be durably accepted
