---
id: 000530
status: open
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '4e38db2ebfb2f388f6d95f6b7fc584137d6732ad' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: score stale session files and stuck slots from activity Couch already sees (baseline: 15 min without a write → ping)

## Problem

Couch sees every event across slots (deliveries and receipts, slot state changes, landings, notices, session-file writes) but uses it only for binary rules: the TL's session file 'owes a response' ping fires on any unanswered delivery, mostly for progress updates that change nothing, while on 2026-10-12/13 three slots finished their work and waited for hours overnight because their reports never reached the hub and nothing noticed.

## Spec

The operator (2026-10-13): a simple local model (not an LLM), kept simple.
1. **Baseline rule (ship first):** a session file with no write for 15 minutes while its owner has unanswered deliveries gets one ping. Anything learned later must beat this baseline on the logged data.
2. **Feature log:** at every session-file write, record the feature vector (deliveries to the hub since the last write, by kind: landed / waiting-for-operator / progress; slots whose state changed; minutes since the last write; operator submitted since); that's a positive. Negatives: the vector one minute after each write, plus random moments not followed by a write within 10 minutes. Same log for slots: moments the hub or operator acted on a slot after silence (positive) vs sampled idle stretches (negative).
3. **Stuck-slot nudge:** a slot idle on an issue branch whose last hub message is unanswered for N minutes (or whose hub delivery failed) nudges the hub; unacted, then the operator. Start rule-based; same logging.
4. Later: fit logistic weights from the log, explainable ('pinged: 3 landings, 40 min since the last write'), switched on only if it beats the baseline.

## Done when

- The 15-minute baseline replaces today's any-delivery ping; the stuck-slot nudge fires on a replay of the 10-12 overnight stall; both log feature vectors with positives and negatives; tests cover the rules and the sampling.

## Plan

- [ ]

## Log

### 2026-10-13
