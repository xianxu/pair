---
gate: boundary-review
issue: 402
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-06T22:55:25-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: '"live :0" claims about add slot are now false (README how-to, menuRowActions doc, sweep comment)'
          detail: README.md:657 says to open the repository's live :0 row to add a slot. The menuRowActions doc comment (menu_actions.go:84-87) says not-live rows get only resume and reboot. menu_actions_test.go:466 says "add slot on the live :0". A parked or unusable :0 (directory present) now offers add slot. README.md:869 and atlas/couch.md:167 (alias) are still correct.
          family: doc-claims-track-offer-table
          round: 1
        - id: BR-2
          severity: Important
          title: OnPrimary add-slot reachability with a parked :0 (Spec bullet 3) is not checked by the advice sweep
          detail: TestRowAdviceNamesOnlyReachableActions sets reach = livePrimary for every OnPrimary step (menu_actions_test.go:471-498). Also check reach against the parked and detached :0 offers (or every :0 shape whose directory exists and whose state is known), so the Spec's "reachable when :0 is parked too" fails if it regresses.
          family: spec-clause-both-states-exercised
          round: 1
        - id: BR-3
          severity: Minor
          title: expectedUnusableActions keeps a leftover bare { } block from the extraction
          detail: menu_actions_test.go:135-151; atlas/couch.md:750-751 also runs past the file's wrap width.
          family: extraction-leftovers
          round: 1
        - id: BR-4
          severity: Minor
          title: The issue Log does not record the Spec bullet 2 execution-path check or the pending live smoke test
          detail: menu.go:815 reads only menuAddSlotPath(thread), not the :0 actor. Write that in the Log, and note that the operator's live smoke test (Done-when clause 2) is still pending.
          family: issue-log-records-verification
          round: 1
      recipe: small-diff-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-06T22:57:55-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: README:657, menuRowActions doc (menu_actions.go:84-88), sweep comment (test:464-466) and OnPrimary field comment no longer claim live-only; remaining enumeration gap raised as a Minor under the same family.
          round: 2
        - id: BR-2
          disposition: addressed
          note: reachableOnPrimary requires the action in live, parked and detached :0 offers (menu_actions_test.go:471-520); len(primaryOffers)!=3 guards the derivation.
          round: 2
        - id: BR-3
          disposition: addressed
          note: expectedUnusableActions is a clean function with no bare block; atlas/couch.md:750-752 rewrapped.
          round: 2
        - id: BR-4
          disposition: addressed
          note: The Log records the Spec bullet 2 menuAddSlotPath-only check and the operator's 2026-10-06 waiver of the live smoke test.
          round: 2
      findings:
        - id: BR-5
          severity: Minor
          title: README:657-658 lists live, parked or detached and omits an unusable :0 whose directory is present
          detail: '2nd in family. Rule: state the add-slot condition (the :0 checkout is present), not a list of states. README:884-885 and atlas/couch.md:750 already do; README:657 is the only listing instance in this window. Line 658 also exceeds the wrap width.'
          family: doc-claims-track-offer-table
          round: 2
      recipe: small-diff-review
      blocked: false
---

# Gate ledger — pair#402 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-06T22:55:25-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `doc-claims-track-offer-table` "live :0" claims about add slot are now false (README how-to, menuRowActions doc, sweep comment)
  README.md:657 says to open the repository's live :0 row to add a slot. The menuRowActions doc comment (menu_actions.go:84-87) says not-live rows get only resume and reboot. menu_actions_test.go:466 says "add slot on the live :0". A parked or unusable :0 (directory present) now offers add slot. README.md:869 and atlas/couch.md:167 (alias) are still correct.
- **BR-2** [Important] `spec-clause-both-states-exercised` OnPrimary add-slot reachability with a parked :0 (Spec bullet 3) is not checked by the advice sweep
  TestRowAdviceNamesOnlyReachableActions sets reach = livePrimary for every OnPrimary step (menu_actions_test.go:471-498). Also check reach against the parked and detached :0 offers (or every :0 shape whose directory exists and whose state is known), so the Spec's "reachable when :0 is parked too" fails if it regresses.
- **BR-3** [Minor] `extraction-leftovers` expectedUnusableActions keeps a leftover bare { } block from the extraction
  menu_actions_test.go:135-151; atlas/couch.md:750-751 also runs past the file's wrap width.
- **BR-4** [Minor] `issue-log-records-verification` The issue Log does not record the Spec bullet 2 execution-path check or the pending live smoke test
  menu.go:815 reads only menuAddSlotPath(thread), not the :0 actor. Write that in the Log, and note that the operator's live smoke test (Done-when clause 2) is still pending.

## Round 2 — 2026-10-06T22:57:55-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — README:657, menuRowActions doc (menu_actions.go:84-88), sweep comment (test:464-466) and OnPrimary field comment no longer claim live-only; remaining enumeration gap raised as a Minor under the same family.
- BR-2 — addressed — reachableOnPrimary requires the action in live, parked and detached :0 offers (menu_actions_test.go:471-520); len(primaryOffers)!=3 guards the derivation.
- BR-3 — addressed — expectedUnusableActions is a clean function with no bare block; atlas/couch.md:750-752 rewrapped.
- BR-4 — addressed — The Log records the Spec bullet 2 menuAddSlotPath-only check and the operator's 2026-10-06 waiver of the live smoke test.

### Raised

- **BR-5** [Minor] `doc-claims-track-offer-table` README:657-658 lists live, parked or detached and omits an unusable :0 whose directory is present
  2nd in family. Rule: state the add-slot condition (the :0 checkout is present), not a list of states. README:884-885 and atlas/couch.md:750 already do; README:657 is the only listing instance in this window. Line 658 also exceeds the wrap width.

## Open findings

- **BR-5** [Minor] `doc-claims-track-offer-table` README:657-658 lists live, parked or detached and omits an unusable :0 whose directory is present
