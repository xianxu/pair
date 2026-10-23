---
id: 000837
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '992e629027cbc13f137438706dab9f91b3ee1981' # card fields mirrored from issue-cards; edit via sdlc
---

# Fence slash qualification journal against system deliveries

## Problem

The #162 disposable slash public check freezes an accepted-input journal baseline after current wrapper Idle, then expects exactly one appended /model entry. A delayed legitimate Couch restart notice can arrive after that baseline and make the later prose assertion fail even when /model was accepted exactly once and prose remained a draft. This is measured startup-notice interference, not evidence prose submitted.

## Spec

Harden cmd/couch/slash_keys_acceptance_test.go against asynchronous system deliveries. Select a causal baseline or identify legitimate system deliveries separately without hiding duplicate slash submissions, ordinary prose submission or journal mutation. Reuse the existing owned disposable fleet and current proof; preserve runtime/output/cleanup bounds. No production readiness/deadline change or blind retry. The TL suggested ignoring or waiting out system deliveries before baseline; settle the exact oracle with adversarial regression evidence before code.

## Done when

- A delayed system restart notice before or after the baseline does not cause false command/prose failures.
- Duplicate /model, submitted ordinary prose, changed pre-existing journal entries and unrecognized additions still fail.
- Qualification uses the existing bounded disposable public command harness and exact owned cleanup.

## Plan

- [ ]

## Log

2026-10-23: #162 source96abb5a7a4dee4c49e9a56afd8c6410c16ba605e. One disposable Claude run accepted /model once (receipt663d25eb-e060-43f0-8186-8415c53f7006), then observed the ordinary prose draft in the live pane. Journal contained /model and a Couch system restart delivery a2965831-4006-4231-910e-52b5c2a34aab; no ordinary prose entry. Frozen-baseline oracle failed at slash_keys_acceptance_test.go:187. Package34.886s, subtest27.13s. No retry and no flake registration. Evidence /Volumes/TestScratch/tmp/pair-slot2/pair162-review-regressions.pykj07ci/public-claude.log SHA256 a6269662c6e86f6ff9a18fb3fa27cd966a131391572cd48074556f6cab7b6b9d. Ops:0 deliveryc64e1031 accepts the measured behavior evidence for #162 and directs this separate unclaimed hardening issue.
