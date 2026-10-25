---
id: 000879
status: working
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '5c74e5497e758ccb4ae96fd56e96c1a061b06f33' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T17:15:49-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "2d363bc1", done: "02b75684"}
---

# Reusable upgrade test: candidate build adopts, lists and resumes warm sessions created by main

## Problem

## Spec

**A reusable upgrade test: the candidate build must start against warm sessions created by main (operator request,
10-24).** On 10-24 a combined stage (pair:0 at 93a57a38, with pair#866) couldn't attach to the operator's existing
warm threads: #866 refused to list any Zellij session without its new readiness marker, which every pre-#866 session
lacks, so resume failed before attach ("missing zellij-ready-v1 marker"). The operator had to revert pair:0 by hand.
Every item had passed its own tests; none started the new build against sessions from the old one.

Build a reusable native fixture: a disposable fleet started by the main build (Claude and Codex actors, hubs and
workers), then the candidate build takes over: startup adoption, listing, explicit resume and attach of every session,
plus a persisted-state read (registries, thread store, side-doc records) created by main.

## Done when

- The upgrade fixture exists and runs headless within bounds; it reproduces the 10-24 failure against 93a57a38 (RED)
  and passes against the corrected pair#866.
- Changes to live-session or persisted-state contracts are required to pass it (documented in the AGENTS.local or
  atlas test guidance and referenced by the close review).
- The combined-stage procedure (ariadne#600) can call it as the warm-start check.

## Plan

- [ ] Add one guarded build replacement operation to the existing native Fleet and portable publication controls.
- [ ] Add an opt-in two-build native upgrade acceptance case for Claude and Codex, with exact identities, public listing/resume/attach and persisted document checks.
- [ ] Qualify a same-build healthy control; capture the old-main to failed-stage RED without modifying readiness receipts; document invocation and cleanup proof.
- [ ] Pass recorded checks and boundary review, then land the fixture so #866 can run the same case against its correction.

Durable plan: [native upgrade fixture](../plans/000879-native-upgrade-plan.md).

## Log

### 2026-10-24

Claimed and entered planning on a fresh main-based branch. Pair:5 confirmed it owns only the #866 zellijquery correction and deterministic controls, and has no duplicate native helper/test. #879 owns the reusable upgrade operation and native regression. Pair:5 supplies read/build-only clean baseline /Volumes/TestScratch/pair859-oldreader.QR5mna/tree at38fe8ea272d960705134e21ad3fbf61d26a4bb00. Git independently resolves the failed stage to93a57a380d290105b497dfd620ffb9e2e5a15528. Existing BuildOwned builds coupled pair/couch/helper, but NewOwned physically copies them: assigning Fleet.Binaries alone cannot perform an upgrade. Reuse the existing Fleet lifetime, private runtime, creator cleanup and exact-generation census (ARCH-DRY, ARCH-FUNERAL).

Second coordination send to pair:5 was refused before delivery with breaker-exhausted: recipient peer-message allowance exhausted. No blind retry or alternate channel. Initial coordination receipt aa4331b5-25cb-4b25-b314-7426f205c9f1 was submitted and pair:5 replied with the required ownership seam. Hub will receive the concrete plan and this delivery limit.

Hub e14ced4c-8af4-4800-ad7b-d9623214ca65 approved the committed plan: one Fleet owner, actual triplet copy replacement under its mutex, PAIR_HOME switch, same roots/RunID/fixtures/sockets, public startup/list/resume/physical attach with unchanged bindings/PIDs/sockets, old registry reads, Claude then Codex with one hub+worker, same-main healthy control then38fe8ea-to93a57a38 RED within20m targeted bound. Hub already relayed fixture-first sequencing to pair:5. Implementation starts only after change-code gate.

change-code admitted the existing branch (quick inferred from current design; close measures the final code diff and upgrades its review if needed). Runtime replacement RED failed0.481s on the unimplemented operation. Added detached/joined-only replacement using the existing Fleet.begin and mutex, validated/staged physical copies and per-file rename. Failure after publication moves the existing OwnedRun to unknown before releasing admission mutex, with cleanup authority preserved. No new lifetime owner/state flags. Focused controls PASS0.374s, expanded partial-publication controls PASS0.518s; final focused race PASS1.881s. Cases include active/absent console, other operation, starting, canceled, missing/nonexec/symlink candidate, missing source, escaped bin, cancel after first rename and refused second rename; successful replacement verifies independent physical bytes and unchanged roots/RunID/fixtures/native tools.

Native acceptance now compiles and skips by default; pure cohort identity controls PASS0.208s. The scenario builds both coupled binary triplets from explicit clean source commits, preserves old fixtures, registers old parent/side docs, and uses a unique old draft per slot as the physical attachment screen oracle. Startup's positive native-client assertion applies to the configured hub; each worker is checked after its explicit resume/physical selection (matches production bare-start policy). A detached read-only failed-stage checkout was created at /Volumes/TestScratch/pair879-stage.OeEwVU/tree, exact93a57a380d290105b497dfd620ffb9e2e5a15528, clean status. No full suite, production change or live fleet operation.

NEXT targeted healthy control: env PAIR_NATIVE_COUCH_UPGRADE=1 PAIR_COUCH_UPGRADE_BASE_SOURCE=/Volumes/TestScratch/pair859-oldreader.QR5mna/tree PAIR_COUCH_UPGRADE_CANDIDATE_SOURCE=/Volumes/TestScratch/pair859-oldreader.QR5mna/tree go test ./cmd/couch -run '^TestDisposableCouchBuildUpgrade$' -count=1 -v -timeout=20m. Retain output /tmp/pair879-upgrade-healthy.log; only after positive cleanup run the exact same scenario with candidate /Volumes/TestScratch/pair879-stage.OeEwVU/tree and retain /tmp/pair879-upgrade-red.log. Stop and classify any different failure or cleanup uncertainty rather than accepting it as the requested RED.

Healthy-control attempt1 failed before any native fleet allocation in0.436s: BuildOwned's couch build needs generated runtimebundle/assets/runtime/files, absent from a pristine checkout. This is a source preparation precondition, not a compatibility RED. Peer baseline remains unchanged. Created a separate owned detached baseline at /Volumes/TestScratch/pair879-stage.OeEwVU/baseline, exact38fe8ea272d960705134e21ad3fbf61d26a4bb00; ran each checkout's own go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime in it and the stage checkout. Both statuses remain clean (generated assets are ignored by repository design). NEXT control uses that owned baseline path for both source variables; same saved log path, prior failure recorded here. No native descendants need reconciliation from attempt1.

## Revisions

### 2026-10-24 — fixture-first landing, hub7aa4f471

Original Done-when's corrected-#866 GREEN remains the joint compatibility goal. For #879's independent landing, the hub explicitly requires the reusable fixture plus captured RED against93a57a38 first; #866 integrates it only after #879 lands and records the corrected GREEN. This removes a circular dependency without stacking branches or duplicating the fixture. #879 must also pass a same-build native control to demonstrate that its oracle and cleanup work. No production correction, live fleet manipulation or full local suite outside the queue is in scope. Targeted runs only; hub offers a queue grant.
