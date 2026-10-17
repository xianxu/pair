---
id: 000633
status: done
deps: [pair#618]
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
card_mirror: '325f6f646204c201603beb4052e23e8095124711' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T07:29:55-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "f9d13e8d", done: "ef574bea"}
actual_hours: 0.89
---

# couch --list picks relaunch vs resend from a typed refusal class, not by matching error text

## Problem

`couch --list` shows a refused slot's receive state with one action (pair#618). `receiveStateText`
(`cmd/internal/couchcmd/list_refusals.go`) picks "relaunch" versus "send again" by matching five phrases ("does not
own", "conversation changed", "launch registration", "not positively listed", "no retained checkout") against the
refusal's free-text reason. Those phrases live in error strings in `message_service.go` and `registry.go`, so
rewording an error silently flips the advice from relaunch to resend. This is the second finding in the
duplicated-request-classification family (pair#618 close review, BR-14). The rule: classify a result once, at its
source, into a typed value, and have consumers read that value.

## Spec

- The ownership/registration failures that only a relaunch clears carry a typed class: a sentinel error or a typed
  error that `errors.Is`/`errors.As` can see, wrapped where they are raised.
- `refuseLocked` records that class on the refusal. The `refusals` op answers `{reason, class}` per slot: still
  reasons only, with no sender binding or body, since the op is unauthenticated.
- `receiveStateText` chooses the action from the class. No text matching.

## Done when

- A test rewords each ownership error string and the `--list` action stays "relaunch".
- A test shows an unclassified reason still gets "send again".
- `list_refusals.go` has no phrase list.

## Plan

- [x] Add couchmessage.RelaunchRequiredError (preserving cause text/unwrapping) and RefusalInfo {reason,class}; classify once in refuseLocked, with cancellation/deadline remaining retryable. Retain class beside the existing bounded refusal record.
- [x] Wrap all five source families in message_service.go, including both registration paths. Return the safe reason/class projection and make list rendering read the class only (ARCH-DRY/PURPOSE).
- [x] Red/green regressions: reworded typed reasons still relaunch; unclassified old phrases, unknown classes and timeout/cancellation still resend; source checks carry the type; broker JSON excludes credentials/body. Keep existing expiry/supersession tests.
- [x] Update atlas/README and run targeted tests and recorded full suite. Headless acceptance: unchanged visible wording/actions, typed data path exercised through broker and list fixtures; no staging requested.

The structured values replace the old refusal strings only on this best-effort diagnostic op. Mixed-version decode failure keeps the ordinary list intact and omits receive annotations, as already documented for unsupported/older supervisors; no heuristic compatibility classifier. Unknown/missing classes use send-again. Other response shapes are unchanged. No new IO, timers or retention owners (ARCH-PURE/FUNERAL); existing 250ms call, 64-entry cap and 10-minute expiry remain. Unauthenticated output stays canonical slot plus reason/class only (ARCH-SECURE). Existing ordered storage/supersession is reused (ARCH-ORDER); no new state machine. Existing fake endpoints and local protocol fixtures suffice (ARCH-MOCK); no external integration changes (ARCH-CONSTRAINTS).

## Log

### 2026-10-16

Planning: all matching strings originate in message_service.go; registry.go has no corresponding error source. Existing refusalReason normalizes context failures, so classification preserves retry advice for those even inside a typed registration wrapper.

### 2026-10-17 — implementation and targeted verification
- 2026-10-17: closed — Typed recovery class replaces all five phrase families at nine source sites; original diagnostics and error identity preserved. Reworded typed reasons relaunch; unclassified old phrases and unknown classes resend; wrapped cancellation/deadline still retryable. Phrase-matching overlay fails all 11 rendering cases, wrapper-removal overlay fails all six exercised source cases. Full affected packages and focused race pass; unauthenticated JSON privacy and existing retention/supersession green. Recorded full suite on bab5d037096002f809aa05747d3b7b8fb7424042 PASS exit 0 in 10m10s, log hash d97c81f11d79; PR #401 checks green. Headless acceptance, unchanged visible actions; mixed-version diagnostic shape mismatch omits annotations while ordinary list remains intact. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

Classified all nine source sites across the five existing ownership/registration families, retaining diagnostic text and cause identity. Broker stores the class alongside its bounded refusal and exports only RefusalInfo; list action reads that class. Existing timeout/cancellation normalization takes precedence, so slow registration checks remain resend advice. Ordinary list operation remains best effort across mixed versions.

Red: structured response fixture failed on the old string shape. A private overlay restoring phrase matching made all 11 reworded/old-phrase/unknown-class render cases fail. Source tests pass, and an overlay removing all wrappers made all six exercised source cases fail (PID mismatch, nonce/session mismatch, malformed ready record, conversation mismatch and unlisted terminal). Empty retained checkout cannot be constructed through valid ThreadStore records (absolute-path validation rejects it); its defensive source is wrapped, not reached with a fabricated invalid store. Embedded production admission closure is swept for the same three typed wrappers; source tests directly cover the send-time registration path.

Full affected packages PASS: couchmessage 24.701s, couchcmd 57.763s. Focused race PASS: couchmessage 1.266s, couchcmd 1.690s. JSON privacy, canonical targets, expiry/eviction and later-admission supersession tests remain green. No real operator environment touched. Recorded full suite and close remain pending.

### 2026-10-17 — full-suite evidence

Recorded full suite on bab5d037096002f809aa05747d3b7b8fb7424042 PASS exit 0 in 10m10s after 1919s machine-gate wait; no flakes. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/bab5d037096002f809aa05747d3b7b8fb7424042/suite.log, sha256 prefix d97c81f11d79. All shell/editor and Go targets completed: couchmessage 25.874s, couchcmd 49.972s, couchcore 260.749s. PR #401 checks green. Ready for close review, then normal landing.

## Revisions

### 2026-10-17 — separate acceptance checklist from publishing

The final Plan checkbox now covers completed docs and verification only. Close and landing remain mandatory SDLC actions after acceptance, rather than an acceptance checkbox falsely ticked before those actions run. No behavior or scope change.
