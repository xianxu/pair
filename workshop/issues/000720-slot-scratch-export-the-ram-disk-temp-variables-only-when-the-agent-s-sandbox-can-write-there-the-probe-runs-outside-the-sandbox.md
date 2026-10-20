---
id: 000720
status: working
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours: 1.56
collaboration-mode: collaboration
card_mirror: 'bb27176e8fb0279d34bc5d4040df5b9736e860a3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T21:04:58-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Slot scratch: export the RAM-disk temp variables only when the agent's sandbox can write there (the probe runs outside the sandbox)

## Problem

pair#708 exports `TMPDIR`, `GOTMPDIR` and `CLAUDE_CODE_TMPDIR` pointing at `/Volumes/TestScratch/tmp/<slot>` into every
slot (couchscratch/temp.go, `Prepare` → env at ~line 189). It does probe the folder with a real write first, but the
probe runs in Couch's own process, outside the agent's sandbox. On 10-19 after the reboot the probe passed while the
Claude sandbox denied the path, so every sandboxed `go build` in every slot failed ("creating work dir … operation not
permitted") until the operator added `/Volumes/TestScratch` to `sandbox.filesystem.allowWrite`. A probe that can pass
while the consumer can't write proves nothing about the consumer.

## Spec

**Superseded by the approved V2 persistent-opt-in contract in Revisions below.** The original requested inference is retained here as decision history, not the current implementation target.

Export the scratch variables only when the slot's agent can write there; otherwise leave the variables unset (the
normal temp folder) and print one note naming the missing allowance.
- Claude: read the effective `sandbox.filesystem.allowWrite` (user, project and local settings; `sandbox.enabled`) and
  require the scratch root, or a parent, to be listed. Codex: the same against its sandbox's writable roots if it has
  one, else treat it as allowed.
- One pure predicate (settings → allowed?) with table tests; the IO stays at the existing seam.
- `couch --show` / slot diagnostics show "scratch: RAM disk" or "scratch: normal temp (sandbox doesn't allow
  /Volumes/TestScratch)".

## Done when

- A slot whose agent sandbox lacks the RAM disk gets no scratch variables and a note; with it allowed, it gets them
  (tests over settings fixtures, including an absent settings file and sandbox disabled).
- The 10-19 failure mode is a test: probe passes, sandbox denies → no export.

## Plan

- [x] Implement the shared pure sandbox predicate and bounded standard settings projection with red-first table/fixture tests.
- [x] Gate scratch exports with the actual selected launch profile; share the read-only selection with show diagnostics and update the atlas.
- [ ] Run focused checks and recorded full suite, close review, land through SDLC and report to the hub.

Durable design: [sandbox-aware slot scratch](../plans/000720-sandbox-scratch-plan.md). One atomic close boundary; full flow because settings and diagnostic plumbing exceed the quick production-line shell.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; source is marked stale. Plan-quality cleared before costing. Policy is one smaller Go module; the settings reader is one greenfield concern using a maintained TOML decoder (design midpoint 1.25h halved for library availability, then spec discount 0.2). Launch plumbing and show each extend an existing small module. Two native settings contracts get discovery allocation; review and atlas are separate. Familiar Go stack 1.0; implementation values are 40% of v2.1 values; design discounts reflect the approved concrete plan. Includes tests, excludes background suite/queue waiting. The moderate native-policy uncertainty remains documented rather than hidden in an invented full resolver.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.04 impl=0.20
item: greenfield-go-module design=0.125 impl=0.22
item: smaller-go-module design=0.03 impl=0.14
item: smaller-go-module design=0.03 impl=0.14
item: atlas-docs design=0.025 impl=0.05
item: milestone-review design=0.02 impl=0.14
item: real-api-discovery design=0 impl=0.36
design-buffer: 0.15
total: 1.56
```

## Log

- Sibling: ariadne#477 (landed) makes sdlc override an unwritable inherited temp folder; this issue stops exporting
  one in the first place. Setup step for the allowlist line documented in ops `teams/devinfra/tools/testscratch/README.md`.

### 2026-10-19 — design approved

Claimed in pair:7 and started from fresh main. Root seam: scratchSlotEnvironment delegates to a host write probe with no consumer-policy admission. A read-only research subagent checked official Claude settings/managed-policy semantics and the absence of an existing sandbox reader. Hub approved the conservative Claude projection and narrow Codex TMPDIR check, then the compact full-flow shape (6cfb6a23-fb86-4013-93fd-1047d7a7b73c). Required positive control: enabled Claude user settings with allowWrite containing /Volumes/TestScratch must stay on RAM. No settings/credential writes or native agent startup. Current Codex tool calls are filesystem-unsandboxed (danger-full-access); they are not sandboxed-conformance evidence. Codex's source explicitly defaults TMPDIR exclusion false and adds TMPDIR to workspace-write roots; links and trust envelope are in the durable plan. Live follow-up is the next natural relaunch; no staging on pair:0.

### 2026-10-19 — red/green and bounded implementation

Plan-quality round 2 CLEAN, PQ-1 addressed; estimate-quality INFO accepted, then change-code exited 0. Predicate subagent owned only sandbox.go/test: permissive-stub red followed by literal policy green. Root-owned settings projections and reader first failed their required positive controls, then passed. The public composition regression TestScratchHostProbeCannotAuthorizeSandboxExport failed with all three RAM exports despite a denying Claude config (0.330s); after admission wiring it passed (0.329s). It mutates portable settings between deny, allow and malformed states, proves the host probe really succeeds, and checks the one explanatory note and shared diagnostic. TestScratchEnvironmentReachesFreshAndWarmLaunches first failed because fresh admission received nil profile (0.306s), then passed with a call-local selected-profile projection (0.359s). The existing callback API and durable profile semantics are retained.

Focused scratch/launch/switch/show checks passed in couchscratch/couchcmd/couchcore (0.148s/0.599s/0.341s), and the complete artifactpath package passed 11.647s. Settings fuzz passed 40,866 executions in 2s (2.181s wall), no panic or corpus failure. Current real user settings were inspected without content output: sandbox enabled, ten concrete allowWrite paths including TestScratch, no denyWrite and no filesystem-disabled key; managed directory absent. The checked-in positive fixture preserves that relevant shape without native configuration writes. New TOML dependency is pelletier/go-toml/v2 v2.4.3; parsing is typed for consumed fields and tolerant of unrelated additions, never a regex or settings rewrite. Settings reads are fixed-count/bounded and cannot hang on special files. Full recorded suite and close review remain due; no live sandboxed agent check is claimed.

## Revisions

### 2026-10-19 — qualification checkpoint

Implementation Git-verified at ad1fc8519b2cfb691a8f96030e22f82200bae7c8, published as PR #470. Final focused policy/reader/launch/switch/show checks passed (couchscratch 0.153s, couchcmd 0.468s, couchcore 0.312s); the full couchscratch package including regular/special/oversized read-boundary controls passed 0.382s. Counted settings test fixes reads at three for 8 and 16 allowances. Recorded full-suite attempt 8cffba3e5ad21df2e70eca2667f643f3 is queued on that exact code head, initially position 4; exec session 93238 owns it and must be preserved. Its private checkout is /Volumes/TestScratch/local-checks/sdlc-local-3990507660. No tests have run in that attempt yet. Per tl-worker's explicit overlap rule, close review may run while queued; --no-plan-check acknowledges only the pending recorded qualification, and --no-project preserves TL ownership. A passing final-code suite still precedes landing; any review code correction requeues it. No bypass of CI or recorded evidence is authorized.

### 2026-10-19 — approved projection envelope

Hub approved standard local Claude settings with conservative unknown-layer fallback, and Codex's narrow TMPDIR rule (b02f4c9f-ac9d-49c5-bcdc-7ff5f673caa9; 6cfb6a23-fb86-4013-93fd-1047d7a7b73c). Profile/CLI config resolution and detected managed policy remain native-owned: use normal temp instead of predicting them. Show labels the current next-launch selection and does not claim to observe an existing child. No pair:0 stage; acceptance uses headless fixtures and source evidence, with live follow-up at the next natural relaunch.

### 2026-10-19 21:30 PDT — V2 persistent declaration replaces native inference

Close round 1 returned REWORK, not finalized: BR-1 (permission-deny inputs) and BR-2 (native local-settings locations) expose a resolver-fidelity spiral; alias handling was another uncertainty. Hub explicitly stopped that direction and approved V2 (53c2a54b-e92a-4bd9-959d-0091c50793e1): persistent regular marker XDG_CONFIG_HOME/pair/agent-sandbox-allowed, fallback HOME/.config/pair/agent-sandbox-allowed, contents ignored. No marker/unreadable/non-file means no optional scratch export and one note naming it. Marker plus existing host/owner checks permits all agents, without parsing or claiming to verify their sandbox settings. The operator creates it together with machine allowances, and owns removal; it survives RAM reboot. A stale/false declaration can still produce build refusal, an explicit accepted limitation. Delete native config readers/projections, TOML and selected-profile plumbing rather than extending them. The durable plan's final V2 section is authoritative for its concepts, tasks and Done-when.

Current Done-when: missing declaration despite a passing host write probe produces no three-variable overlay and a named note; regular declaration permits the original overlay; isolated HOME and explicit XDG routing, missing/non-file/error cases and effect-free show diagnostics pass. Full final-code suite and close review precede landing. No real marker or machine settings are changed by this worker. Hub will create the marker and update ops README after landing and before any relaunch. The superseded recorded suite was safely cancelled through local-check cancel before execution, leaving two other checkout runs untouched; reviewer artifacts are retained. V2 waits for revised change-code before code changes.

### 2026-10-19 — V2 implemented; declaration basename clarified

V2 plan-quality round 3 CLEAN and change-code passed before implementation. Native settings/TOML policy code and selected-profile plumbing are deleted. The pure table failed against a permissive stub, then passed; the public integration failed on a missing marker note before implementation (0.388s), then passed (0.362s). All scratch-related couchcmd checks passed (0.462s), and core scratch/show controls passed (0.234s). Explicit XDG, isolated HOME fallback, marker lifecycle, non-file/symlink refusal and effect-free show are covered.

The source guard caught the proposed agent- basename as a reserved thread-artifact family. Hub approved pair/sandbox-allowed instead (3827d91b-8eff-4aec-9ef2-9802731551e7); this basename supersedes the earlier V2 proposal everywhere, with unchanged XDG/HOME routing. The existing guard remains intact. BR-1 and BR-2 are addressed by removing native-policy interpretation, not by extending its fidelity claims. Updated README/atlas explain that the declaration is a promise, not a sandbox proof. The operator owns creating it and configuring allowances after landing and before relaunch. Final renamed-source guards, recorded suite and close review remain due.

Final renamed-source qualification passed: complete couchscratch 0.237s and artifactpath 5.525s, all scratch/show controls in couchcmd 0.467s and couchcore 0.296s; git diff --check clean. Native settings files and TOML dependency have no remaining production references. Queue a new recorded suite only on the pushed V2 code.

### 2026-10-19 — BR-3 diagnostic target coverage

Close round 2 disposed BR-1/BR-2 as addressed, but did not finalize because BR-3 identified missing scratch selection on a valid slot with no thread. Current-selection diagnostics apply to both thread targets and slot-only targets. Admission is machine-wide and does not require a ThreadRecord; retain the existing callback and invoke it with an empty record for the slot-only case. Extend the real slot-only show regression rather than inventing a thread or preparing scratch. Cancelled queued V2 suite through SDLC; final corrected code gets a new recorded run. No sandbox admission or setup contract changes.

BR-3 regression TestShowReportsASlotWithoutAThread failed with an empty scratch diagnostic (1.805s), then passed with the effect-free thread show control (1.875s) after adding the slot-only projection. Review round 2 explicitly disposed both native-fidelity findings. The prior queued suite ended cancelled before execution; no passing evidence is claimed from it.
