---
id: 000635
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
card_mirror: 'e6b898fb6cef27fc5f2d72ba21985d106b65f265' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T06:48:43-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "6103592f", done: "1aedc7d1"}
---

# couch --message-status with a short ID reports a submitted message as unretained

## Problem

`couch --send-to` prints the message's full ID. A sender who checks it with the 8-character prefix the
fleet uses in reports (`couch --message-status 97fcc5ef --json`) gets
`{"Code":"refused","Error":"receipt unavailable: no connected recipient retains this message"}`, even while the
broker holds the receipt as `submitted`. Querying the full ID returns the real status. Observed 2026-10-16 on pair:2:
three hub reports read as lost were in fact delivered, and the TL had replied to one of them. A false
"unretained" answer leads senders to resend, or to treat delivery as broken.

## Spec

- `--message-status` with an ID prefix resolves it against the receipts the caller may see (its own sends and
  receipts addressed to it).
- Exactly one match: answer for that full ID.
- Several matches: refuse as ambiguous and name the candidates' full IDs.
- No match: refuse as unknown, worded so it can't be mistaken for an answer about a real message.
- Never answer "no connected recipient retains this message" for an ID the broker holds.

## Done when

- A test sends a message, queries its 8-character prefix and gets `submitted`.
- A test with two receipts sharing a prefix gets an ambiguity refusal that lists both.
- A test with an unknown prefix gets an unknown refusal, not "unretained".

## Plan

- [x] Reproduce the eight-character submitted lookup over the real CLI/socket path; retain the failing measurement before production edits.
- [x] Resolve caller-visible receipt/refusal prefixes under the broker lock, preserving exact-ID precedence and current waiting projections; sorted ambiguity names only visible IDs (ARCH-DRY: existing receipt ownership and retention remain authoritative).
- [ ] Regress unknown and ambiguous prefixes, exact precedence, sender/recipient visibility, hidden/expired receipts, refused/queued receipts and existing exact-ID wrapper recovery; run targeted race checks and a recorded full suite.
- [ ] Update README/atlas/help and prepare close evidence and deferred restart steps; then run fresh close review, land and report to the hub.

## Log

### 2026-10-16

### 2026-10-17 — design

Claimed in pair:3 after #634 landed; started from fresh main. StatusContext first tries an exact in-memory lookup, then asks retained-receipt holders for that literal ID, and finally reports no recipient retains it. An ID prefix therefore cannot discover an already-known full ID. Resolve only the authenticated caller's visible in-memory receipts/refusals; exact visible ID wins, one prefix yields its full receipt, multiple prefixes yield a sorted candidate list. Preserve existing exact-ID recovery and uncertainty when a holder does not answer; a no-answer lookup reports an unknown ID/prefix instead of asserting a real message was unretained. No new status, protocol operation, retention store or retry. TL approved the plan and live-check deferral to the operator restart alongside #629/#631/#634, without pair:0 staging (delivery dde33f3c-5eca-4621-aca7-9ffeea35c7bb).

Before production edits, the real CLI/socket regression delivered a message and confirmed submitted by full ID (632.084 us, exit 0), then querying 97fcc5ef returned exit 1 and the false no-connected-recipient-retains refusal (150.958 us). Evidence: /tmp/pair635-cli-prefix-red.log. The visibility/ambiguity regression also failed on original source: a shared visible prefix was reported unretained, a unique visible prefix was missed, and unknown input was described as a message-retention result (/tmp/pair635-prefix-visibility-red.log). An initial fixture compile error used json instead of the actual jsonOutput field; corrected before recording this behavioral failure.

The fixed real-CLI prefix query returned exit 0 with the full submitted receipt in 995.666 us under race (/tmp/pair635-prefix-green.log). statusLocked centralizes existing exact-read visibility and waiting projection; StatusContext resolves only that same visible set under the mutex before any recovery IO. Prefix regressions pass for ambiguity ordering, foreign IDs excluded from candidates, exact precedence, sender and recipient views, queued position/composer detail, refused receipts and refusal expiry. Existing forgotten-full-ID recovery and silent-holder uncertainty tests also passed (/tmp/pair635-prefix-regressions.log). README, atlas and CLI help now explain unique prefixes, ambiguity, unknown lookups and retaining full IDs for post-restart recovery.

Full couchmessage race passed 28.665 s (/tmp/pair635-couchmessage-race.log), all TestMessage* CLI/service tests passed under race 12.869 s (/tmp/pair635-cli-race.log), and declaration-comment checks passed 0.317 s. Added a terminal-receipt expiry case and verified it separately under race (/tmp/pair635-expired-prefix.log). The bundled agent skill's existing full-ID advice remains valid; the new CLI lookup behavior is documented in README, CLI help and atlas.

Pushed source head 0789c0e55d73633a7d40c0aa420d925c9d8a40e9; required recorded suite queued as f627daf4bcfe82d7bf2e837dd713c4fa (sixth initially). TL approved opening an early draft PR for CI overlap, with fresh close review and normal gates still required (delivery 9296ee09-223f-4970-95d7-f43adf073120). No flake attempt or retry so far.

Draft PR https://github.com/xianxu/pair/pull/400 is open; merge-check and local-gates-guard passed. Native conformance did not trigger for this patch. The suite advanced to fifth in the machine line; the queued source remains unchanged.
