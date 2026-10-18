---
collaboration-mode: discussion
---

# Warm recovery no-reply diagnostic plan

> **For agentic workers:** Consult AGENTS.md Section 3. Use bounded source/test tasks where context is capturable; the hub owns approval and native-lane admission. Steps use checkbox syntax.

**Goal:** Reproduce and locate the warm-recovery unanswered-query boundary, then correct its demonstrated cause while retaining recovery/ownership assertions and unknown observations.

**Architecture:** Reuse strict zellijreply parsing, ScopedThreadArtifactCollisionChecker, zellijtest's shared stateful CLI generator and WithZellijQueryObserver. The first approved phase adds deterministic cross-layer refusal coverage, explicit disposable fixture isolation and bounded deferred measurement; it does not guess a product/readiness correction. Native measurements require a separate hub grant. A cause-level correction is designed from that evidence before implementation.

**Tech Stack:** Go tests, private file-backed Zellij CLI fake, installed Zellij0.45.1 only in a separately authorized disposable lane.

## Scope and alternatives

Recommended: production-shaped deterministic fault tests plus direct-CLI observation around the existing local detach boundary. This distinguishes decoder/adapter failure from native stdout failure without a new transport. Alternative1, switch KillClient to graceful detach or add a settle poll now, changes the stimulus before its cause is established. Alternative2, repeat the old transparent IPC proxy, has known scheduling perturbation and a different trust/lifecycle surface. Neither is part of this proposal.

Historical evidence proves an empty native CLI reply with a later healthy same-session observation. It does not prove the original query's IPC order or detach completion. Current source proves missing explicit private socket/runtime roots after fixture setup; fix that test isolation before any native experiment, but do not label it the no-reply cause.

## Concepts and integration points

No new pure production entity or recovery state is proposed. Existing pure decoders and recovery decisions remain unchanged.

| Existing concept | Kind | Location | Status |
|---|---|---|---|
| RecoveryEvidence / DecideRecovery | PURE | cmd/internal/couchcore/recovery.go | Reused unchanged |
| ParseClients / ParsePanes | PURE | cmd/internal/zellijreply/clients.go, panes.go | Reused unchanged |
| RecoverThread / scoped native observation | INTEGRATION | cmd/internal/couchcore/recovery_execute.go, artifactcollision.go | Reused by new regression |
| Fixture model/fault/query/effect state | INTEGRATION | cmd/internal/zellijtest/fake.go, tests/lib/fake-zellij.py | Reused shared generator; no parallel serializer |
| ControlledZellij lifecycle | INTEGRATION | cmd/internal/pairlifecycletest/live_zellij.go | Existing lifetime/cleanup; measure call boundaries first |
| Disposable fixture environment | INTEGRATION | cmd/internal/testpair/environment.go, tests/lib/pair-test-env.sh | Existing empty-environment boundary reused explicitly |
| Completed-query observer | INTEGRATION | cmd/internal/launcher/zellij.go | Reused unchanged; callback stores bounded memory only |

One issue owns one branch from main. Diagnostic helpers should remain test-only unless a measured requirement justifies changing the shared ControlledZellij API. Native files are fixture data, never installed/operator state (ARCH-DRY, ARCH-PURE, ARCH-SECURE).

## Actions, uncertainty and ordering

Actions: invoke recovery once; inspect retained diagnostics; request the declared native window; submit an evidence-based design revision.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Owned server, healthy header-only clients | Existing detached warm recovery | Invoke recovery once | Existing documented recovery effects only |
| Empty/whitespace or malformed/wrong-kind query | Unknown; operation refused before repair/launch | Inspect retained diagnostics; explicitly invoke a new operation only after fresh evidence | No native/durable mutation from this refusal |
| Local client wait completed, server detach unproved | Detached state remains unproved | Inspect query/lifecycle record | No inference from local wait |
| Canceled query / capture capacity reached | Existing cancellation/unknown; diagnostics marked incomplete | Inspect retained bounded record | Uncaptured diagnostic bytes only |
| Native fixture setup or cleanup failure | Qualification invalid; exact owned resources recorded | Report to hub before further native use | Partial private fixture evidence; operator state untouched |

Do not automatically retry an unanswered query. A later healthy fake/native reply cannot retroactively authorize the failed operation. Preserve owner generation checks, confirmation reads, query deadlines and original assertion bodies. The fixture lifetime owns private resources until exact owned children/server cleanup finishes; environment restoration/removal follows that owner, never a timer (ARCH-ORDER, ARCH-FUNERAL).

## Diagnostic phase tasks

- [x] **1. Cross-layer deterministic contracts.** Add cmd/internal/couchcore/recovery_native_reply_test.go using a real temporary ThreadStore/scoped marker/index, production recovery/scoped adapter and shared zellijtest.Fixture. Use the existing process-identity fake for server ownership only; no function-call mock for reply state. Inject empty/whitespace, wrong-kind and malformed responses at first/second owned-pane reads and list-clients. Assert exact failing argv/ordinal, no further query/retry in that operation, unchanged complete durable record and no launcher/quiescence effects. Include a valid header-only detached control and keep later replies healthy to prove they are not consumed to rescue the failed attempt. Add cases to the shared generator catalog only if existing named faults cannot express them.
- [x] **2. Explicit isolation.** Add one test-only private live-environment helper used by the two existing couchcore fixtures. Allocate short /tmp roots with os.MkdirTemp and register owned cleanup; pass explicit HOME/config/data/cache/runtime/socket roots through testpair.ApplyEnvironment. Preserve TMPDIR/SHELL behavior and all original assertions. Test effective child environment using a harmless fake command, not native Zellij, and poison ambient roots to prove they cannot escape the boundary. Socket-directory plus randomized session name must fit the installed native name budget; no operator defaults are read.
- [x] **3. Deferred measurement.** Add a test-only synchronized recorder that receives existing WithZellijQueryObserver callbacks. Capture argv, exit, stdout/stderr counts and bounded copies, start/finish/duration plus call context deadline. Record startup-return, KillClient entry/return, helper-kill/join and fixture-cleanup entry/return timestamps around existing calls. Initial phase does not claim per-internal-PTY/signal/server-detach ordering; if needed, request a focused helper observation design. Keep at most64 response records/8MiB per test, mark dropped content/counts explicitly, and flush only after the measured path/cleanup. Unit-test overflow/cancellation and confirm observer-on/off equal query/effect traces through the shared fake. No synchronous file IO in callbacks, no additional native queries.
- [x] **4. Verify deterministic phase.** Run go test ./cmd/internal/couchcore -run 'TestRecoverThreadNative|TestRecoveryLiveEnvironment|TestRecoveryQueryCapture' -count=1, relevant existing recovery refusal/owner tests, complete launcher/zellijreply/zellijtest packages, and targeted races. Prove each new refusal/isolation/capture branch with a meaningful failure mutation, not wording assertions. If the patch reveals a mechanically enforceable mistake class, consult the now-shared Go lint catalog rather than inventing a provider-specific syntactic rule.
- [ ] **5. Native request, not authorization.** After deterministic verification and reviewed setup, ask the hub for one exclusive disposable installed-native window. Proposed declared controls are warm-only count10 and original park+warm count10, serial in explicit private roots, same binary/config, all original assertions retained. Bound the whole window to15 minutes with existing per-test35s bounds unchanged. Retain every attempt and failure, effective roots, installed path/version, session/server PID/start identity, logs and hashes. No operator sessions/build, OS clipboard, native rebuild, proxies or parallel native run. Do not execute this row without the hub's separate grant.
- [ ] **6. Classify and revise.** Map each failure to query stdout/process collection/parser, fixture setup/lifecycle or native emission. Empty stdout is native query failure, not proof of the native stale-ID trigger. If the cause remains unknown or both controls pass, report that gap and request the next bounded experiment; no blind repeat and no claim of a fix. Before any product/readiness change, append an evidence-based design revision and obtain hub approval. A nontrivial resulting correction gets a revised durable implementation plan and change-code gate.

## Cost and acceptance

This is a test/diagnostic path, not a new per-byte/per-tick product hot path. Existing query counts/deadlines remain fixed. Let Q be completed query records and R be control repetitions: capture work O(Q) until64/8MiB capacity; retained memory bounded; native/test work O(R) with serial ownership. A counted8/16-operation fake test must exactly double queries and effects while observer-on/off traces match; saturation tests prove capture storage stops growing. Native wall time is evidence, not a new speed assertion (ARCH-CONSTRAINTS).

Final acceptance still requires the issue's original deterministic cause regression and repeated isolated warm recovery with unchanged assertions. Diagnostics alone are not completion. Once the cause correction is verified: update atlas only for changed surface, commit/push, required recorded full suite, sole sdlc close review, SHIP then sdlc merge. Report retained native/fake evidence and any limitation to the hub. No suite or close is claimed in this planning phase.

## Revisions

- 2026-10-18 — Hub approval5a7cb535 authorizes landing tasks1-4 plus the normal recorded suite/close/merge gates now. Proposed tasks5-6 and the earlier final cause-level acceptance paragraph are deferred: after landing, ask whether the hub wants a native window based on recurrence. Current acceptance is the diagnostic/isolation contract in the issue's revised Done-when. This delivery makes no claim that the original upstream trigger is fixed. No native use is authorized by this approval.

- 2026-10-18 — Close BR-1: observer events now carry the actual query caller deadline and the effective deadline after the existing query cap. Capture consumes those event facts instead of its installation context. This corrects all inherited/shorter child callers without changing query timing, count or retry behavior (ARCH-PURPOSE). Corrected task4 selector to name the actual production-recovery regression tests.
