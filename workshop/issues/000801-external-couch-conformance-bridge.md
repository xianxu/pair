---
id: 000801
status: open
deps: [pair#758]
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'f1b694ec75ce86cf2302a23aa0e5363d664a7eaf' # card fields mirrored from issue-cards; edit via sdlc
---

# External hold-open Couch conformance bridge

## Problem

parley.nvim#405 needs a real hold-open #758 editor endpoint producer for external Lua conformance. parley.nvim#408 exercised only public adopt-store/direct-store registry commands and did not start a daemon. Pair's cmd/internal/testcouch owns genuine disposable fleets but is a Go testing.TB API, with no external driver entrypoint. Reimplementing its lifetime and teardown in Parley would duplicate authority and risk live-state leakage.

## Spec

Provide a bounded Go testcouch owner/driver and thin external wrapper, reusing Build/New/Prepare/Start/Close/AssertCleaned. Build a candidate or accept an explicit candidate Couch executable, allocate a fresh private isolated root and private PATH from an empty environment, and start only the owned disposable Couch. No live roots, operator sessions, installed agents, network or default singleton authority.

Coordinate final argv/readiness/stop contract with parley.nvim:1 before code. Proposed external shape: explicit candidate binary plus owned scratch selection; one readiness JSON line with Version, canonical Namespace, IPC Socket and complete Environment map for clients (including private store/identity/Couch selection), diagnostics on stderr. Readiness must follow actual service admission, never mere socket creation. Bound readiness/output/startup and total lifetime. Stdin EOF or TERM requests the same idempotent joined stop, returning cleanly only after Close and AssertCleaned prove cleanup. On uncertain cleanup retain exact owned evidence/root and exit failure; no guessed PID cleanup or recursive removal before descendant absence. Parent owns launch timeout/process group; document crash recovery limits.

parley.nvim#405's real-producer acceptance depends on this bridge and #758. Per TL5e43c052, #405 may land its M1 on fakes and native unit evidence; only real-producer acceptance waits. This issue is independent work from main after its producer dependency lands, not added to #758's current review.

## Done when

- External driver can start a given/built producer, read a single confirmed readiness envelope, run Parley's isolated Lua driver in the emitted environment, and stop by EOF or TERM.
- Tests cover startup failure, early EOF/TERM, repeated stop, client/child cancellation and uncertain teardown without live-state access; genuine native conformance proves exact cleanup.
- Interface confirmed with parley.nvim:1, docs/entrypoint and inventories updated, recorded checks and normal whole-issue gates pass.

## Plan

- [ ]

## Log

2026-10-22: filed unclaimed by pair:6 per TLe632079d/5e43c052 after inspecting testcouch and #408's actual direct-store probe. parley.nvim:1 provided proposed envelope and stop interface in receipt59bd1da0. TL will dispatch to a free Pair slot; #758 fixes its own review findings first.
