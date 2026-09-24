---
id: 000308
status: working
deps: [pair#306]
github_issue:
created: 2026-09-22
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T17:46:01-07:00
flow: {kind: quick, provenance: inferred, spec: "23c5773a", done: "3a662ba1"}
---

# Slots v2: independent workspace preferences

## Problem

Numbered threads need the same agent and launch-parameter preferences as the primary without leaking settings across slots.

## Spec

Project: `pair/workshop/projects/couch-slots-v2.md`. Fresh task derived from the current v2 contract; historical task bodies are not prerequisites or implementation plans.

Persist preferences per durable workspace identity, including selected agent and supported launch parameters already available to the primary. Resume uses that workspace’s recorded settings. Define inheritance on first creation from existing repo/default configuration, then preserve independent values for :0/:1/:2. Preserve backward compatibility for existing primary configuration and define preference behavior on thread replacement.

Preferred-model customization is excluded from v2: add no new model picker, slot model-default field, or slot-number-to-model mapping. Existing underlying harness launch semantics remain governed by the current configuration contract. No fixed roles or permissions are implied by slot number. ARCH-PURPOSE: extend the existing preference authority rather than adding a competing settings store.

### Agreed scope — 2026-09-23

This section takes precedence over earlier conflicting layout or policy text.

Keep preferences keyed to the durable main workspace address despite its nested checkout path `/workspace/worktree/<repo>-slotN/<repo>`. Ordinary dependency clones accessed through that thread do not receive additional Couch preference records, inherited agent launches, or numbered identities. Primary and numbered slots retain the same supported preference capabilities.

### Approved implementation contract — 2026-09-23

Reuse the existing start form and Switch agent parameter editor. First use selects
an explicit agent if supplied, otherwise the current Couch root agent (falling
back to claude); parameters come from that agent's primary-repository defaults.
This preserves the current new-slot behavior. Copying :0's personal preferences
is excluded by the operator's confirmed first-use choice.

After successful registration, the workspace remembers its selected agent and
exact per-agent argv, including explicitly empty argv. Existing local values win;
an agent never used in that workspace falls back to its primary-repository
defaults. Apply that fallback consistently to create, fresh and switch-agent.
Resume retains the conversation's recorded launch profile. Fresh replacement
keeps the workspace's per-agent history. Failed/cancelled/unconfirmed launches
do not replace successful preferences; existing start transactions own recovery.

Keep numbered preferences in `<environment>/.couch/preferences.json`, with the
nested main checkout as their physical key. Keep :0's existing global path key
and decoder unchanged. Sibling dependency clones gain no Couch records merely
by existing. Couch-resolved launch arguments must never write Pair repository
defaults; direct Pair launches retain their existing default-persistence behavior.
The existing freeform parameter contract remains available, including underlying
harness flags; no preferred-model setting or model picker is added.

ARCH-DRY/ARCH-PURPOSE: reuse `ResolveLaunchProfile`, `RecordSuccessfulLaunch`,
ThreadStore routing and the existing forms. ARCH-PURE: preference precedence
remains pure; selecting a slot's primary default root belongs to the store/IO
boundary. ARCH-MOCK: use existing stateful Couch runner/artifact fakes and Pair's
fakeRuntime with temporary filesystem stores. ARCH-SECURE: preserve strict
preference decoding and fresh-argument validation before destructive switching.
ARCH-CONSTRAINTS: no new scans, external calls, timers or background work; use
existing path routing and one per-agent default lookup per resolution. Existing
4096-byte switch-parameter limit stays unchanged. ARCH-STATE: registration is
the existing commit event, stale preview refuses before park, failed registration
preserves prior settings. ARCH-FUNERAL: adds no artifact family; one revisioned
preference file per workspace remains overwritten through the existing journal.

## Done when

- :0, :1, and :2 retain independently selected agents and supported launch parameters across park/resume and process restart.
- Changing one workspace’s preferences does not mutate another workspace or repo defaults.
- Existing primary preferences migrate/read compatibly; a new slot uses the current Couch agent and primary-repository defaults, then retains its own successful settings across replacement. This behavior is documented and tested.
- Invalid settings use the existing actionable validation path; no preferred-model customization is introduced.

- Nested path resolution retains stable main-workspace preference keys; acquiring sibling dependency clones creates no extra preference or launch records.

## Plan

Single acceptance boundary; expected production change fits the quick-flow shell.

- [x] Approve the first-use inheritance contract above. Add regressions showing new-slot, fresh and switch-agent agree on primary-repository fallback while saved per-agent and explicit empty arguments win. Reuse `slotRecoveryOperationFixture`, managed-slot fixtures and switch-agent stateful fakes in `cmd/internal/couchcore/`; run the new tests red before changing behavior.
- [x] Centralize default-root selection using existing ThreadStore path routing in `cmd/internal/couchcore/threadstore_location.go`; consume it in `couch.go`, `slotrecovery.go`, and `switchagent.go` without changing preference keys, lifecycle states or schemas. Preserve errors and ordinary-path behavior. Re-run focused tests green.
- [x] Add a regression through `RunLaunch` in `cmd/internal/launcher/createflow_test.go` proving a Couch-supplied ordinary launch preserves repository defaults (including an empty profile), while direct Pair explicit arguments still persist after readiness. Fix the persistence guard in `createflow.go` using existing `AgentArgsFromCouch` provenance; run the tests red then green.
- [x] In `slotrecovery.go`, validate/build the fresh profile before `replaceSlotCurrent`. Add a regression in `slotrecovery_test.go` for remembered forbidden resume selectors proving rejection preserves current metadata, history and preferences, with no child launched.
- [x] Exercise :0/:1/:2 using temporary nested workspace stores and existing stateful fakes: choose distinct agents/argv through launch/switch APIs, park/resume, reopen Couch/store, start fresh, and verify exact profiles and byte-preserved other-workspace/default records. Verify subdirectory/address normalization and no sibling-clone preference records. Include stale preview, failed registration and invalid parameters; reuse existing tests where they exercise the same production boundary. Add menu coverage only where existing switch/start tests miss slot identity routing.
- [ ] Document first-use fallback, independent settings and fresh/resume behavior in `README.md` and `atlas/couch.md`; update the project. Run `go test ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/launcher -count=1`, affected race tests, `go test ./... -count=1`, `make pair bin/couch`, and `git diff --check`. Commit and pass `sdlc close --issue 308 --verified '<evidence>'`, then `sdlc pr` and `sdlc merge`.

## Log

### 2026-09-22 — fresh v2 task

Created from the agreed workspace/UI contract and the request for a clean task breakdown. Implementation has not started; estimates follow design approval.

## Revisions

### 2026-09-23 — Preferences belong to the numbered thread, not each dependency

Reason: operator agreed nested environments, ordinary remote dependency clones and existing per-repository publication. Delta: added the authoritative scope clarification and acceptance criteria above; original task context remains as provenance. No implementation or lifecycle-status change is claimed by this revision.

### 2026-09-23 — build preference UX on local storage from #306

Reason: `.couch` is the authoritative slot store across conversation replacement.
Delta: #306 routes the existing path preference API to
`<environment>/.couch/preferences.json` and preserves it during fresh conversation.
This issue should reuse that storage and successful-launch publication, finishing
inheritance/selection UX and restart isolation coverage rather than introducing
another store. Primary preferences remain in the global backend. No new preferred
model setting is authorized.

### 2026-09-23 — implementation proposal after #307

Claimed and entered planning. Existing storage and Switch agent UX satisfy most
of the surface. Inspection found differing default-root lookup in create versus
fresh/switch, and ordinary Pair launch persistence lacking a Couch-provenance
guard. Added the proposed contract and concrete quick-flow plan above. First-use
inheritance was offered to the operator; the proposal preserves existing behavior
until changed. No implementation has begun.

Read-only mapping also identified fresh-argument validation after current-record
replacement. Added a test and ordered validation before mutation to the plan.

### 2026-09-23 — proposal review

Fresh-context review approved the proposed contract and quick-flow sizing with no
important gaps. Implementation must retain the explicitly known primary default
root for first creation before enrollment, and retain ordinary-path defaults.
Operator approval of the proposed contract remains pending.

### 2026-09-23 — approved contract and implementation

Operator selected current Couch agent plus repository defaults; proceeded with
the reviewed quick-flow plan and passed change-code. TDD reproduced different
fresh/switch default roots and invalid fresh parameters replacing current metadata;
focused regressions passed after sharing primary-root lookup and moving validation
before replacement. The launcher regression reproduced ordinary Couch launches
rewriting repo defaults for both empty and nonempty argv; provenance now excludes
those writes while direct Pair defaults remain supported. The helper performs
read-only path routing so start previews do not initialize or recover stores.
Full-suite, race and three-workspace acceptance verification are underway.

The Done-when inheritance criterion now states the operator-confirmed default
explicitly, matching the approved Spec without expanding scope.

Focused acceptance and the final affected race suite passed (couchcore 25.931s,
couchtty 1.948s, launcher 1.327s); `make pair bin/couch`, affected `go vet`, and
`git diff --check` passed. An earlier full-suite run compiled an in-progress
acceptance fixture and failed there; the final complete run is underway after
fixing fake session publication and hosted-process cleanup. No runtime workaround
was required.
