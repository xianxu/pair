---
id: 000778
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9d8f9b1d406856908db6dee43a406e6e15e5833e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T18:23:34-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/couch.TestBareCouchInstalledCommand]
flow: {kind: quick, provenance: inferred, spec: "6653e200", done: "855df120"}
---

# flake: TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/couch.TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub: Exact source3279e1e9ec6ccbddba0b50193f5c8f72c11f4b22: train238 suite lease-1293856745 failed cmd/couch TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub after15s with live/contested terminal 📁1-1; log .git/sdlc/test-stats/inflight/de3d7e12cd48155a874dbd8db94ac3f6/run.log. On the unchanged source, env -u PAIR_TEST_BINARY go test ./cmd/couch -run ^TestBareCouchInstalledCommand$ -count=1 -timeout=90s -v passed11.305s (PTY2.12s). This fixture has private socket roots but fixed terminal name; SessionServers filters only session basename, a lead not yet a forced reproduction. Production scribe is unchanged. Track this intermittent fixture refusal separately from #768.

## Spec

Fresh private identity authorities each start at C=1; their first terminal is therefore the same 📁1-1 even when socket roots differ. The launcher deliberately uses a host-wide name census. Preserve that refusal and give fixture authorities distinct high-range store IDs instead (ARCH-PURPOSE). Both installedEnv and testcouch.allocateRoots construct this class of authority; initialize both through one testpair helper (ARCH-DRY).

The helper creates a new private registry exclusively, seeds its first C from the live test-process PID plus a bounded atomic fixture sequence, and leaves enrollment and N/M allocation to IdentityStore. Each fixture owns one store. PID supplies disjoint concurrent-process namespaces; sequence distinguishes fixtures in one process. Reject invalid PID, sequence exhaustion and existing registry rather than overwrite. This is concurrent-fixture uniqueness, not permanent identity: crashed residue/PID reuse still encounters the unchanged fail-closed census. Existing high-range fixture precedent stays disjoint. Do not weaken launch admission, add retries or change readiness budgets.

Alternatives: socket-only isolation is insufficient because census is host-wide; serializing suites would hide the shared-name defect and reduce capacity. Decimal temp suffixes alone do not guarantee uniqueness across temp parents. The OS process incarnation owns the namespace for the lifetime of its fixtures, whose existing cleanups join children before releasing their roots (ARCH-ORDER/FUNERAL). The private seed file is one bounded record in the already-owned identity root and is removed with that root; no new shared state or production format changes. Work is constant per fixture: one sequence reservation and one exclusive seed write; normal allocation is unchanged (ARCH-CONSTRAINTS/PURE/EVOLVE). Existing stateful native fixture and shared disposable harness remain the external seam (ARCH-MOCK).

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.
- Two actual independent installed environments allocate distinct C/session/tag through the real allocator; first N/M=1 and later N/M advance under stable C. Equivalent shared-fleet authorities also remain distinct.
- The existing installed PTY launch passes with an expectation derived from its allocated C. Seed validation, no-overwrite and concurrent fixture coverage pass; an AST catalog rule catches omitted seeding at both constructors.
- Recorded full suite and fresh close review pass; retain original failures and subsequent passes without treating a single native pass as proof of absence.

## Plan

- [x] Reproduce independent installed-environment name equality deterministically; retain the saved #775/#757 failure references and source trace.
- [x] Add one exclusive fixture-authority initializer, apply it to installedEnv and testcouch roots, derive the installed launch expectation, and enforce both constructors in Pair's lint catalog with positive/negative tests.
- [ ] Run targeted allocator/constructor/PTY/lint checks and inventory preflight, and record a passing full suite. The SDLC close/landing receipts then own closure and retirement of the flake registration.

## Log

### 2026-10-21

### 2026-10-21 — recovered evidence and bounded plan

#775 landed in train254 through PR534; its retained installed-fixture-778.log and peer757-installed-fixture-778.log under workshop/evidence/000775 preserve the original fixed-name refusal. Peer #757 unchanged-head exact-package rerun passed7.556s (receipt4a5871e0); original failure and pass remain distinct evidence. Read-only tracing confirms private host authorities default NextC=1, first M=1, and native census selects matching basename across all socket roots. The old failure does not identify its competing process; the new deterministic allocation regression will establish the collision precondition without depending on scheduler timing. Bounded independent audit found the same constructor class in testcouch and the cross-parent weakness of decimal suffixes. Quick-flow fixture repair, no public product state or gates change; post plan to hub and proceed unless objected under tl-worker rules.

## Revisions

### 2026-10-21 — concrete fixture-isolation acceptance

Expand the original root-cause Done when into deterministic allocation, preserved admission, both constructor consumers, exclusive seed safety, lint enforcement and recorded verification. Preserve the known-flake history; closing retires its existing parent registration.

### 2026-10-21 — deterministic root-cause RED

Added regressions through the actual installedEnv and testcouch.allocateRoots constructors, then real IdentityStore.Allocate. Both fail on the intended condition: independent authorities each yield C1/N1/M1 and duplicate native name 📁1-1. Installed regression exits1 in0.207s; shared-fleet regression exits1 in0.224s. This reproduces the precondition independently of native scheduler timing. Existing admission checks remain unchanged. Helper implementation and pure lint rule are delegated as separate bounded file-owned tasks; main owns consumers and integration verification.

### 2026-10-21 — constructor and executable GREEN

Shared initializer and its tests now cover exclusive creation, preserved preexisting residue, permissions, directory errors, PID/sequence bounds, permanent exhaustion, 8/16 concurrent fixtures and real allocation with stable C/advancing N/M. Reviewed the pure maximum and corrected one edge: the MaxUint64 seed must refuse because the real allocator cannot enroll from it; the focused regression went RED then race GREEN. Both constructors now seed their private authority. installedEnv returns its seed C so the existing executable PTY assertion derives the exact expected Pair tag via FormatPairTag instead of hardcoding C1. The combined installed allocation/PTY parent tests PASS13.415s; fleet factory tests PASS0.247s. Pure constructor catalog cases PASS, with exact helper-import/alias resolution and shadowing negatives. No production census or timeout changed; full-package checks and inventory remain due.

### 2026-10-21 — package and inventory verification

Full targeted packages PASS: testpair5.666s, testcouch20.105s, repolint6.239s. Staged tracked-source inventory correctly named the two existing changed reader audits; their inputs/ownership are unchanged (main_test remains local/literal fixture; environment_test remains non-Go full fallback). Regenerated only those two per-file certificates using the printed record commands, then fast inventory PASS. No .sdlc or workflow edits. The shared harness now receives longer high-range session identities, so run its existing opt-in Claude/Codex native matrix on the committed source before the full suite and close; this qualifies the actual socket/actor paths as well as pure allocation. No duplicate harness, timeout adjustment or live-fleet operation.

### 2026-10-21 — native two-personality qualification PASS

Committed sourcebf7be2e2e6b1371df6182287c83f3ddd2aa948b1 passed make test-couch-fleet in75.199s (Claude34.43s, Codex31.36s). Each genuine native fleet registered its steward and3workers, delivered/submitted the ordinary message with accepted-body proof, detached/restarted with all4actors returned, and proved owned generations absent plus joined child/IO cleanup. Retained completion tail and pinned manifest are workshop/evidence/000778/native-*; completion-tail SHA256c983b14df2c7a3ccc64e003aa97486480b7b12d67299c3d82af03c151cb6e739. This qualifies the high-C identity/session paths through both personalities without any live-fleet effect or retry. PR536 opened; full recorded suite and close/landing remain required.

### 2026-10-21 — final-source recorded suite queued

Recorded suite lease275330665 is retained on pushed sourceec3c6714 (PR536), attempt0afdfcfe7e70e8b1e2cd8bd438e8dc1e; fresh checkout /Volumes/TestScratch/local-checks/sdlc-local-4115531898. Initial queue position5, no execution yet. Native matrix and affected-package/inventory checks are green. Keep this runner; wait for its complete result, finish the final Plan checkbox, then close and merge without chasing newer main. No continuation/restart while this check is queued or running.

### 2026-10-21 — complete suite failure retained and inventory correction

Lease275330665 acquired after2792s and finished11m57s, exit2. Complete log confirms exactly one failing test: artifactpath.TestProductionArtifactReferencesAreExactlyClassified, because the new non-test testpair/couchidentity.go was absent from its exhaustive source inventory. All other packages and non-Go groups passed, including installed cmd/couch, testpair, testcouch and repolint. The post-run flake-registry fetch separately failed DNS resolution for github.com, so SDLC recorded known-flake-rule-unavailable; this does not explain or waive the deterministic test failure. Do not run resume/register this as a flake. Raw log and pinned manifest are suite-inventory-failure.*; SHA256a662f00e110fac06295153cc8cf469dddfefb51834bbfc5c16ac79aec825decc.

Audit: the helper writes only its private test identity authority, just like the already-classified couchidentity store, not a Pair artifact family. Add it to the ordinary NonArtifactSources inventory; preserve exhaustive scanning and the known-violation quarantine. Verify the failing guard directly, then all relevant packages and inventory before recording a fresh suite on the new pushed head. No behavioral change to the qualified native helper; the native source evidence remains valid for this inventory-only correction.

### 2026-10-21 — ordinary inventory correction verified; DNS blocks publication

After the one-line NonArtifactSources classification, all relevant packages PASS: artifactpath16.042s, testpair11.379s, testcouch27.901s, repolint11.911s. Reader inventory and diff checks PASS. Git diff against the native-qualified source confirms no changes in the helper or either constructor, so the Claude/Codex native evidence still measures the same behavior. The full failed run remains retained; require a new recorded suite on the corrected pushed head. A separate git fetch also failed with Could not resolve host: github.com; an outside-sandbox fetch is checking whether access policy explains it. No passing suite or publication is claimed.

### 2026-10-21 — implementation checklist clarification

The original final Plan row included "close, land, and retire the flake registration through close" after full verification. Those are subsequent SDLC receipts, not pre-close implementation tasks. The active row now ends at the passing full suite and explicitly leaves closure/registration retirement to the binary's close/landing receipts. No acceptance or implementation scope changes; the original text is retained here to preserve the revision.

### 2026-10-21 19:40 — hub-directed DNS hold

Authenticated TL receipt65adc077 confirms machine-wide github.com DNS failure from a stuck Mac resolver; the operator owns recovery. Hold corrected sourcefa165827a45bacaa8954ea19d8a90d544c8f4c25 locally until the hub says Git works. The outside-sandbox fetch had already completed with the same DNS failure before this direction; no machine settings or resolver changes were made. All own recorded/targeted/native runs are finished; no live runner or landing ticket is retained for #778. PR536 remains at the earlier pushed head.

Resume: confirm the hub's recovery notice, publish the corrected local head with sdlc pr, run a fresh recorded suite (the old suite was a deterministic inventory failure, not a flake), tick the final Plan row only after its pass, close and merge. The native Claude/Codex pass still qualifies unchanged behavior. Do not start #772 on this branch; it follows #778 landing. #775 is done/archived after train254/PR534, reports delivered and grant revoked. Previous recovery screen remained unavailable; issue/Git/tool evidence supplied continuity.

### 2026-10-21 19:58 — DNS recovery and fresh suite

Authenticated TL receipt9b51cb83 reports operator DNS-cache recovery and instructs resumption. sdlc pr successfully published corrected checkpointf025dba400e49af9c6a6d6a4beda4cbc8aba5cf7 to PR536. Fresh recorded suite lease2286202273, attempt54cda992757ffb884f353a200db8f185, is retained on that head; checkout /Volumes/TestScratch/local-checks/sdlc-local-2794522815, initial position2. Old failure remains preserved and no flake resume was used. Wait for this complete result; do not duplicate/cancel or restart while it is queued/running. Close after a full pass; then enter merge promptly.

### 2026-10-21 — next-item priority updated

Authenticated TL receipta5d4365f directs returning immediately to owned #722 (fleet drain, PR477) after #778 lands, and reporting its recovered state. This supersedes the earlier next-item order #772/#740. Keep finishing #778 and its retained fresh suite first; do not switch branches during this check or landing.
