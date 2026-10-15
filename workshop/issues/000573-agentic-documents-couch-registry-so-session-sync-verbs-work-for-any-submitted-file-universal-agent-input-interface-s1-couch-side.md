---
id: 000573
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours: 1.704
card_mirror: '3a06369e2a8eaecda421654c9bc1535050e6be49' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T10:56:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Agentic documents: Couch registry so session-sync verbs work for any submitted file (universal-agent-input-interface S1, Couch side)

## Problem

Project universal-agent-input-interface (ops `teams/devinfra/projects/universal-agent-input-interface/design.md`,
slice S1): Couch's session-sync verbs (`read`, `diff`, `write`, `status`, the owed-event ledger, the CAS write and the
operator-lock refusal) work only for files with `type: session-sync` frontmatter. The operator wants every markdown
file submitted from Couch's right-pane nvim to become an **agentic document** with the same guarantees, without
writing anything into the file.

## Spec

- A Couch-side **agentic registry** keyed by canonical path: `couch --session-sync register FILE --owner SLOT` (called
  by parley on the first submit), `couch --session-sync forget FILE`, and listing. Agentic status lasts until forgotten.
- All `--session-sync` verbs accept a registered file without `type: session-sync`; the owner comes from the registry
  (or an explicit `couch-owner:`, which wins).
- The owed-event ledger works for registered files; the submit delivery names the file.
- **Forgotten lock:** if the agent needs to write and the operator has held the lock for more than 30 minutes, the
  write refusal says so (lock age), so the agent can ping the operator (or raise a Needs you item for the TL's file).
- A path move or rename updates the registry entry.

## Done when

- Tests: register, then read/diff/write/status on a file with no frontmatter; owed events; lock-age in the refusal
  after 30 minutes (fake clock); forget; rename. Existing session-sync files are unaffected.
- `couch --skill` documents the registry and the 30-minute rule.

## Plan

Detailed implementation plan: [agentic document registry](../plans/000573-agentic-document-registry-plan.md).

- [ ] Build registry model/store and stable document metadata, with idempotent register/forget/rename and read-only membership discovery.
- [ ] Route every sync verb through registration-aware ownership, CAS, owed matching and lock-age refusals, preserving legacy behavior.
- [ ] Expose the agreed Parley CLI/JSON contract and update help, skill and atlas.
- [ ] Verify targeted/race/full suites and operator live check, close review, then land on SHIP.

## Log

### 2026-10-15

Claimed on TL dispatch; read ops project's project.md and design.md before design. Scope is S1 Couch only; TL owns the project file. Sent proposed contract to parley.nvim:2, confirmed submitted, and received its integration requirements (read-only membership, canonical missing-old rename paths, explicit partial-effect outcomes). Existing ledger and CAS are reused (ARCH-DRY); stable document identity keeps rename from losing shadow/folds or old-path submits (ARCH-ORDER). Read-only design exploration independently confirmed these integration points. Concrete plan pending hub approval; implementation has not started. start-plan's attempted weave refresh refused due to the preserved unrelated untracked pensive; it explicitly admitted planning on the existing substrate, with ariadne #385/#386 in flight.

TL approved the concrete plan, with 4096 registrations / 16 MiB registry and explicit forget advice on overflow. No :0 staging for #573: land independently, TL rebuilds, and the combined plain-note live check follows parley.nvim#332 staging and operator restarts. Await Parley contract agreement before implementing the public verbs. Full flow is selected for the planned code volume; no estimate derived before plan-quality.



### 2026-10-15 — implementation checkpoint

Plan-quality round 1 found stale legacy metadata on forget, an incorrect assumption of unconditional 14-day ledger expiration, and verbose test inventories; all three were addressed. Rounds 2 and 3 returned CLEAN/high with no open findings. change-code admitted full flow after derived 1.704h estimate (calibration flagged stale). Estimate-quality INFO cautioned that durability/handoff and verification allowances are optimistic; own verification, coordination and landing are included in the module/review allocations, while the TL's combined postlanding editor acceptance is separate. Actual remains measured at close.

Parley accepted register/lookup/rename semantics and filesystem ownership; frozen caller contract committed at 432fab15c7f655d2392d823053e37b575638ee7b in atlas/couch-agentic-documents.md and sent to parley.nvim:2. The producer model was delegated as a bounded pure task; metadata/CLI integration stayed here. Core implementation supports all registered verbs, stable central state, durable legacy handoff and old-path submits. Tests captured plain-file refusal/stale legacy handoff/missing aged-lock advice before integration, then passed. A private-publication fault keeps central authority through partial legacy export. Same-identity register after uncertain publication retains registration horizon. Real symlink-parent and two-process registered CAS controls pass.

Queued-submit loss was independently confirmed: delivery recorder timestamps completion, potentially long after a rename. Requested and received TL approval for the additive AcceptedAt/AddressedAt delta; real broker queue/rename/path-reuse/late-completion test first lost the original submit, then passed after using address-time evidence. Broker focused/full/race tests passed independently (0.162s/7.043s/1.191s). Focused races passed sessionsync/couchmessage/couchcmd (4.768s/1.488s/2.816s); earlier complete sessionsync race passed 10.061s. CLI contract/status/read tests passed 2.055s; make build passed. Latest full sessionsync/artifactpath package checks remain in flight at this checkpoint; required pushed-head full suite, close review and landing remain outstanding. No :0 staging or operator acceptance claimed.

Updated embedded Couch reference passed a fresh-reader retrieval/application check: old reference lacked registration/rename/age/unsupported-binary facts; current reference supplied the safe protocol, including null-only absence and never expiring the lock. No user files edited by reference probes.


## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration source is flagged stale by sdlc, so the result is provisional.

Registry model/store is one greenfield Go concern: base design 2h, halved for existing durablefile/flock/ledger libraries then ×0.2 for the approved concrete plan = 0.2h; implementation 0.7h ×0.4 = 0.28h. Three existing-module extensions cover resolution/CAS, historical owed matching/handoff, and CLI: each base design 0.2h ×0.2 = 0.04h, impl 0.4h ×0.4 = 0.16h. Issue/spec work uses 1h ×0.2 design and 0.2h ×0.4 implementation. Docs use 0.1h ×0.2 design and 0.15h ×0.4 implementation; one review uses 0.1h ×0.2 design and 0.4h ×0.4 implementation. Familiar filesystem/Go stack = 1.0. Thorough-plan design buffer = 15%; Parley implementation and operator live-check time are outside this issue's build scope. Approximate primitive range 1.0–2.1h, selected 1.704h (0.56×1.15 + 1.06).

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.20 impl=0.08
item: greenfield-go-module design=0.20 impl=0.28
item: smaller-go-module design=0.04 impl=0.16
item: smaller-go-module design=0.04 impl=0.16
item: smaller-go-module design=0.04 impl=0.16
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.02 impl=0.16
design-buffer: 0.15
total: 1.704
```

## Revisions

### 2026-10-15 11:21 PDT — queue-safe rename attribution

TL approved immutable broker AcceptedAt and additive ledger AddressedAt for historical pathname matching only. Completion time retains its existing delivery/status/retention meaning. This fixes queued pre-rename submits completed after rename, without changing the public caller contract. The durable plan records the exact state ordering and cross-service regression.
