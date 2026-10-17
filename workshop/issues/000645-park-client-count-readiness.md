---
id: 000645
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'c8f5e2efdfd1a31a6536625a8e5c4010c0bb4ba0' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T10:19:44-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "d2af5684", done: "bca0b514"}
---

# Park fixture accepts empty client baseline as attachment readiness

## Problem

Read-only #636 audit found that cmd/internal/couchcore/park_lifecycle_live_test.go zellijClientCount directly invokes list-clients, counts every nonempty stdout line (including the header), and accepts successful empty stdout as count0. startProductionParkHelper snapshots that baseline, starts a helper, and accepts any later count greater than baseline as attachment. If baseline is an empty successful reply and the next response is only CLIENT_ID / ZELLIJ_PANE_ID / RUNNING_COMMAND header, the count moves0 to1 even with no attached client. Source behavior establishes this false-readiness sequence; no native reproduction or causal link to #636/#640 is claimed.

## Spec

Make the park live fixture preserve unknown query results and count actual clients consistently with the production boundary. Do not interpret exit0/empty stdout as zero clients or a header as a client. Preserve lifecycle assertions and setup/deadline separation; no blind retries, sleeps, longer deadlines or production transport change. Use the existing shared query/parser seam where practical (ARCH-DRY/ARCH-ORDER).

## Done when

- A stateful fake regression feeds empty baseline then header-only response and proves helper attachment cannot be established by that sequence.
- A complete header-only response represents known zero clients; a real added client can prove attachment against a known baseline.
- CLI failures and empty/whitespace replies remain uncertain, fixture callers share the rule, and existing lifecycle tests pass.
- Native validation is serialized with #640 and uses a disposable session after the shared native race is classified; do not claim this fixture weakness explains #636 without evidence.

## Plan

- [ ] Add stateful disposable CLI regressions for empty/whitespace/error baselines, header-only zero, real client rows, and refusal before helper startup.
- [ ] Expose the existing canonical one-query ClientCountContext boundary and route every park counter call through it (ARCH-DRY); preserve production parsing, deadlines and query count.
- [ ] Run targeted and recorded full-suite checks; coordinate any native acceptance with #640's diagnostic owner.

## Log

2026-10-17: Filed unclaimed from #636 read-only audit. Relevant source: cmd/internal/couchcore/park_lifecycle_live_test.go zellijClientCount and startProductionParkHelper; canonical production semantics in cmd/internal/launcher/zellij.go and zellijparse.go reject empty stdout before counting rows. No code, test fixture or native environment changed.

### 2026-10-17 — Approved quick design

Authenticated TL dispatch255c075c and approval42415423. #636 is explicitly held behind pair:1's #640 native-trigger investigation; #645 is independent fixture counter work. Claimed and started on its own branch from main. The first start-plan refreshed the slot-local substrate and compiled, then the required repeat started the branch. Refreshed weave's package policy was read first: slots only check packages with HOMEBREW_NO_AUTO_UPDATE=1; the observed refresh used brew bundle check, with no package install or auto-update.

Use the existing launcher.ZellijSource clientCountContext as exported ClientCountContext, preserving its existing query/error/parser implementation. Both fixture baseline and polling already call zellijClientCount, so replacing that adapter removes the duplicated header-inclusive/raw-empty parser in one place. No new authority state, transport, retry or timer is introduced. Existing native table grammar is unchanged; the acceptance envelope is CLI failures, empty/whitespace replies, complete header-only zero and complete client rows. Strict table grammar would be separate work requiring native format evidence.

Tests use stateful shell CLI fixtures in temporary allowlisted environments, never the real Zellij binary. A subprocess exercises startParkLifecycleHelper's real baseline guard: first empty reply and queued header-only second reply must fail before any helper starts/second query. Direct counter controls cover valid zero and actual added rows. Targeted checks precede the pushed recorded suite. Native runs stay held until pair:1 releases the shared diagnostic slot; no pair:0 staging or native dependency change is requested.

## Revisions

- 2026-10-17 (TL42415423): narrowed the filed broad malformed-response wording to CLI failures plus empty/whitespace replies. Preserve existing canonical production parsing; strict nonempty table grammar stays outside this counter-readiness bug. No relaxation of the empty-baseline attachment invariant.
