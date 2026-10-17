---
id: 000645
status: done
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '32907d3c3957b3ae3dfb6455f06a4db5b2a87cf3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T10:19:44-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "d6f20eaa", done: "ea33b815"}
actual_hours: 0.64
---

# Park fixture accepts empty client baseline as attachment readiness

## Problem

Read-only #636 audit found that cmd/internal/couchcore/park_lifecycle_live_test.go zellijClientCount directly invokes list-clients, counts every nonempty stdout line (including the header), and accepts successful empty stdout as count0. startParkLifecycleHelper snapshots that baseline, starts a helper, and accepts any later count greater than baseline as attachment. If baseline is an empty successful reply and the next response is only CLIENT_ID / ZELLIJ_PANE_ID / RUNNING_COMMAND header, the count moves0 to1 even with no attached client. Source behavior establishes this false-readiness sequence; no native reproduction or causal link to #636/#640 is claimed.

## Spec

Make the park live fixture preserve unknown query results and count actual clients consistently with the production boundary. Do not interpret exit0/empty stdout as zero clients or a header as a client. Preserve lifecycle assertions and setup/deadline separation; no blind retries, sleeps, longer deadlines or production transport change. Use the existing shared query/parser seam where practical (ARCH-DRY/ARCH-ORDER).

## Done when

- A stateful fake regression feeds empty baseline then header-only response and proves helper attachment cannot be established by that sequence.
- A complete header-only response represents known zero clients; a real added client can prove attachment against a known baseline.
- CLI failures and empty/whitespace replies remain uncertain, fixture callers share the rule, and existing lifecycle tests pass.
- Landing acceptance is the stateful boundary regressions, recorded full suite and SHIP review. Native park acceptance is explicitly TL-deferred to #640's later shared window, using a disposable session; no causal claim about #636 is made.

## Plan

- [x] Add stateful disposable CLI regressions for empty/whitespace/error baselines, header-only zero, real client rows, and refusal before helper startup.
- [x] Expose the existing canonical one-query ClientCountContext boundary and route every park counter call through it (ARCH-DRY); preserve production parsing, deadlines and query count.
- [x] Run targeted and recorded full-suite checks; coordinate any native acceptance with #640's diagnostic owner.

## Log

2026-10-17: Filed unclaimed from #636 read-only audit. Relevant source: cmd/internal/couchcore/park_lifecycle_live_test.go zellijClientCount and startParkLifecycleHelper; canonical production semantics in cmd/internal/launcher/zellij.go and zellijparse.go reject empty stdout before counting rows. No code, test fixture or native environment changed.

### 2026-10-17 — Approved quick design
- 2026-10-17: closed — BR-1 addressed by shared actual fixture readiness loop and stateful zero/one baseline, unchanged count, then growth tests. Equality and never-growth mutations both fail behaviorally; restored checks and race pass. First recorded suite after fix on pushed 69f62789 passed exit0 14m7s, following original suite pass10m33s; logs retained with SHA256 in issue. Atlas updated. Native park check explicitly TL-deferred a7364b43 to #640 window; #636 remains held. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

Authenticated TL dispatch255c075c and approval42415423. #636 is explicitly held behind pair:1's #640 native-trigger investigation; #645 is independent fixture counter work. Claimed and started on its own branch from main. The first start-plan refreshed the slot-local substrate and compiled, then the required repeat started the branch. Refreshed weave's package policy was read first: slots only check packages with HOMEBREW_NO_AUTO_UPDATE=1; the observed refresh used brew bundle check, with no package install or auto-update.

Use the existing launcher.ZellijSource clientCountContext as exported ClientCountContext, preserving its existing query/error/parser implementation. Both fixture baseline and polling already call zellijClientCount, so replacing that adapter removes the duplicated header-inclusive/raw-empty parser in one place. No new authority state, transport, retry or timer is introduced. Existing native table grammar is unchanged; the acceptance envelope is CLI failures, empty/whitespace replies, complete header-only zero and complete client rows. Strict table grammar would be separate work requiring native format evidence.

Tests use stateful shell CLI fixtures in temporary allowlisted environments, never the real Zellij binary. A subprocess exercises startParkLifecycleHelper's real baseline guard: first empty reply and queued header-only second reply must fail before any helper starts/second query. Direct counter controls cover valid zero and actual added rows. Targeted checks precede the pushed recorded suite. Native runs stay held until pair:1 releases the shared diagnostic slot; no pair:0 staging or native dependency change is requested.


### 2026-10-17 — Red/green evidence and deferred native acceptance

The stateful CLI regression ran RED with the original fixture counter: empty and whitespace returned count0/nil; header-only counted1, one real client counted2, two counted3. The actual startParkLifecycleHelper subprocess accepted an empty baseline followed by header-only output and failed with the precise message "header-only response falsely established helper attachment". This proves the false readiness without invoking installed Zellij or altering its diagnostic scheduling.

GREEN with the canonical adapter: focused core regressions pass0.557s, focused race1.508s; complete launcher package43.981s passes. Every direct query asserts exactly one CLI invocation. The real baseline guard refuses ErrZellijNoReply after one query, before helper start; the queued header-only second observation is untouched. Zero/one/two-client controls count only rows after the header. The getter body and existing production parser are unchanged; the fixture now shares the existing per-query budget/cancellation rather than defining another rule.

Authenticated TL a7364b43 approves landing on recorded suite plus SHIP and defers native park acceptance to #640's later window. That live check is not yet run or claimed. It must exercise TestParkLifecycleLiveIntentOnlyMutation on a disposable native session, preserving lifecycle/ownership assertions, serialized with pair:1. #636 stays held on the shared native investigation. No native runs, dependency changes or pair:0 staging were performed here.

### Deferred acceptance — native park check

Owner/coordinator: TL with pair:1's #640 diagnostic window. After the native slot is released and the shared trigger is classified, run the real park conformance against the landed counter adapter and record the exact build/environment/result. This is a deferred acceptance obligation, not a pre-landing gate (TL a7364b43); it does not justify retries or relaxing any native failure.

Existing recovery ownership/deadline and explicit-empty terminal-fixture controls also pass5.421s. git diff --check is clean. The adapter and source rename introduce no new production file or durable artifact family.

### 2026-10-17 — Recorded suite

The first recorded suite on pushed implementation head d2219c429adbad8b35855f36f26ae73e6aa64cf0 passed (exit0, 10m33s), including couchcore271.562s, launcher36.029s and wrapper50.705s. No rerun was needed. Log: /Users/xianxu/workspace/pair/.git/sdlc/local-checks/d2219c429adbad8b35855f36f26ae73e6aa64cf0/suite.log; SHA256 16a2e6257c28eff93bbd201862083e7a5441ac2e3ebe4e08ad607f783fb9b612. The native park check remains explicitly deferred under TL a7364b43, coordinated with #640; no live acceptance is claimed.

### 2026-10-17 — R1 readiness branch coverage

R1 returned FIX-THEN-SHIP with Important BR-1 (readiness-contract-branch-coverage): individual counter controls did not exercise the helper's unchanged-count/growth decision. Verified this gap and extracted the existing polling loop into a fixture-local helper without changing its behavior. The actual startup path and the new stateful test now share that loop. Both zero-client and one-client baselines observe unchanged counts before growth, requiring exactly three queries. Equality-accepting mutation fails both cases at query2; a never-accept-growth mutation fails both at the existing test context deadline. An initial never-growth mutation failed compilation because it removed count's use; that was discarded and the behavioral mutation rerun. Production code remains unchanged from the first suite. A fresh recorded suite follows these test changes.

The first recorded suite after BR-1 changes passed on pushed head69f62789ae83426b320fbe35dc523d0c947ba2d3 (exit0, 14m7s, after166s machine queue); no failed attempt or retry. It survived the operator Couch restart. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/69f62789ae83426b320fbe35dc523d0c947ba2d3/suite.log; SHA256 ddfcefa65d5b5634438287cc0d872119e39d1566b47b4806163c875e273ccb22. Focused readiness/unknown-baseline checks passed0.772s and race2.411s.

## Revisions

- 2026-10-17 (TL42415423): narrowed the filed broad malformed-response wording to CLI failures plus empty/whitespace replies. Preserve existing canonical production parsing; strict nonempty table grammar stays outside this counter-readiness bug. No relaxation of the empty-baseline attachment invariant.
- 2026-10-17 (TL a7364b43): native park acceptance moves to #640's later shared window; recorded suite plus SHIP authorize this fixture-only landing. Deferred acceptance remains explicit in the Log.
