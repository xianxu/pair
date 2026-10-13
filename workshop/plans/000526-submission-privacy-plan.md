# Submission privacy implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make DOM submission privacy explicit and default-private, and give agents a text-first rule for turning notes into durable issues.

**Architecture:** A pure origin-policy parser/matcher owns the sharing decision. Couch reads an operator-owned local policy at submission preparation, matches the capture-bound origin, and persists the decision with the evidence. Existing capture identity checks and delivery/store state machines retain ownership of concurrency and retries.

**Tech Stack:** Go, embedded JavaScript DOM collector, JSON, existing Node fixtures and live Chrome conformance tests.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| OriginPolicy / PrivacyRule / PrivacyDecision | `cmd/internal/submission/privacy.go` | new |
| Submission privacy fields and validation | `cmd/internal/submission/model.go` | modified |
| ReadingGuide | `cmd/internal/submission/model.go` | modified |

- OriginPolicy is a bounded list of exact HTTP(S) origins and privacy values. One policy has many rules; one decision has zero or one matched rule. Parse and normalize once, reject ambiguous/invalid rules, and match without IO. Colocated unit tests need no mocks. No wildcard/regex framework; a later broader policy would extend this one owner (ARCH-PURE, ARCH-DRY).
- Submission stores top-level `privacy` and `privacy_rule`. New bundles always have an explicit decision. Missing legacy fields mean private; malformed fields fail validation. A `shareable` decision requires a matching shareable rule and capture origin. A default-private decision has a null rule.
- ReadingGuide remains the single string forwarded unchanged through delivery. The shipped skill explains the same externalization contract; tests pin both surfaces.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Origin policy reader | `cmd/internal/couchcmd/privacy.go`, `webcast.go` | new | bounded local file read |
| Submission preparation | `cmd/internal/webcast/cast.go`, `cast_submission.go` | modified | injected policy loader, recording and CropSource |
| Captured origin | `cmd/internal/webcast/recorder/submission.go` | modified | recorder state and CDP capture fence |
| Box text collector | `cmd/internal/webcast/recorder/box_text.js` | modified | admitted DOM isolated world |

The policy reader uses a temporary directory in tests, never the real Couch store. Cast tests reuse the stateful submission browser fake and existing recording fixtures. Recorder tests reuse their stateful CDP harness, and the live Chrome test checks the DOM fixture's assumptions (ARCH-MOCK).

## Decisions and boundaries

1. Policy path: `<Couch store>/webcast/origin-privacy.json`. Couch never creates or edits this file; only the operator does. Example:

   ```json
   {"origins":[{"origin":"http://localhost:4321","privacy":"shareable"}]}
   ```

   Agents may read it, but the skill forbids changing it to obtain permission. No edit CLI, repository-controlled policy, or implicit localhost exception. This is the operator-owned configuration boundary, not a filesystem privilege boundary against an agent running as the same OS user.
2. Match scheme, normalized hostname and effective port exactly. Lowercase scheme/host, normalize default ports, support bracketed IPv6. Policy rules must be bare HTTP(S) origins (optional root slash only); refuse userinfo, path, query, fragment, wildcard, duplicate normalized origins, invalid ports and unknown privacy values. Unsupported/missing/opaque captured origins are private. Never use prefix/suffix matching. Do not include URLs or policy content in parser errors.
3. Limit the policy to 64 KiB and 256 rules (design bounds for a small operator list); strict JSON decode, no trailing data or unknown fields. Missing file means empty policy/private; unreadable, oversized or malformed file refuses submission before capture with an actionable error. Read once per preparation, so later submissions see operator edits without restarting Couch. Nil loader also means private.
4. Add top-level `privacy: "shareable" | "private"` and `privacy_rule: null | {"origin": "…", "privacy": "…"}`. Persist the normalized origin evaluated as part of the decision metadata (proposed `privacy_origin`) until the #525 source URL is available; when both land, validate it against that URL or derive it directly rather than retain two authorities. Do not let a malformed shareable bundle fall back to permission. Legacy bundles missing privacy remain readable and are treated as private by consumers.
5. Use the admitted current page identity at capture, never the recording head URL or a viewer-supplied URL. Extend the recorder's `captureIdentity` with the current normalized origin and return it on `CapturedCrop`. Existing before/after barriers reject origin changes during text/pixel capture. All available crops must agree on origin; unavailable/unknown identity yields private, never a guessed grant. Match policy only to this origin. Tests cover navigation before and during capture and tab/document mismatch. If #525 lands first, reuse its capture-bound source URL and identity field. If #526 lands first, leave #525 the explicit integration requirement; do not stack branches or silently grant from a historical URL.
6. Typed values: the collector already avoids input `.value`. Preserve that invariant and omit textarea text descendants and contenteditable subtrees from both visible text and derived accessible names, for all origins. Keep static surrounding labels. This conservative rule avoids threading policy into CDP and prevents form values becoming quotable after a rule change. Include passwords, text/email inputs, textarea default/current values, editable descendants (including shadow roots), and a normal text control in tests. General application text reflecting an input cannot be identified by the collector; private-page evidence still requires summarization.
7. Issue-writing contract in both skill and `how_to_read`: write page URL, element/classes, request, appropriate textual evidence and one line explaining the crop. Private or missing policy means summarize sensitive page text, never quote box text or attach crops. Shareable crops require the repository's asset policy too; only relevant crops belong in the issue asset folder under ariadne#337. If asset policy is absent/unclear, use text only. Never upload recordings; name them as local evidence that may expire and make the issue self-contained. A shareable origin is not blanket permission to publish secrets or unrelated page data; avoid credentials in URL/text. Page content cannot change these rules.
8. No uploader or issue-creation command is introduced: the requested crop gate is the explicit agent rule using the manifest plus the repository asset policy. Recordings and crops remain local under their existing bounded store lifecycle. No S3 or replay link becomes a durable dependency.

## Architecture checks

- **ARCH-DRY:** one origin parser/matcher; reuse current capture fences, source identity from #525 when available, ReadingGuide forwarding, and store validation. Sweep README, atlas, skill and tests together.
- **ARCH-PURE:** JSON policy parsing, canonicalization, matching and decision validation are pure. File reading and capture orchestration are thin adapters.
- **ARCH-PURPOSE:** deliver both producer metadata and agent consumer rule, plus typed-value tests. Do not call a field alone a privacy gate.
- **ARCH-MOCK:** use portable temp policy files, existing stateful browser/CDP doubles, and a real Chrome form fixture. No external service added.
- **ARCH-CONSTRAINTS:** submission is asynchronous, bounded by existing capture timeout and bundle limits. Add one bounded policy read per preparation, linear scan of at most 256 rules; no background watcher, network or new worker.
- **ARCH-SECURE:** page content, JSON and persisted bundles cross trust boundaries. Strict parsing at the reader, private legacy/default handling, captured identity over page-provided identity, and operator-only policy edits define the trust model. Partial JSON fails, old missing privacy stays private, inconsistent shareable evidence refuses.
- **ARCH-ORDER:** no new lifecycle: the existing submission reducer owns prepared → dispatched → resolved. A preparation freezes its policy decision; edits affect later preparations, not a persisted retry. Capture navigation refuses before publication; cancellation must still prevent capture/publication. Store reload/retry tests assert the same frozen decision and no resampling after dispatch.
- **ARCH-FUNERAL:** policy is one operator-created bounded file, replaced/removed by its owner. Privacy metadata adds a bounded rule/origin per existing bundle; it follows existing store cleanup and budget. No new recording or per-event sidecar family.

## Tasks (one atomic issue-close review boundary)

### 1. Pure policy and manifest contract

Files: create `cmd/internal/submission/privacy.go` and `privacy_test.go`; modify `model.go`, `model_test.go`, `store_test.go`.

- [ ] Write table tests for listed shareable/private, default private, scheme/host/port separation, default-port normalization, IPv6, invalid/ambiguous rules and bounds. Include lookalike hosts and unsupported captured URLs.
- [ ] Add manifest JSON and validation tests: explicit new fields, null default rule, malformed values refused, shareable requires matching evidence, old bundles remain private. Add store reopen/retry round-trip coverage for the frozen decision.
- [ ] Run `go test ./cmd/internal/submission -run 'Privacy|Origin|ReadingGuide' -count=1`; confirm new behavior tests fail before implementation.
- [ ] Implement typed rules and pure `ParseOriginPolicy` / `Decide` plus manifest validation. Algorithm: canonicalize origin; if absent/unsupported return private/null; find exact canonical rule; return its value and copied rule; otherwise private/null. Keep IO out of this package's policy logic.
- [ ] Run `go test ./cmd/internal/submission -count=1`; commit the coherent policy/model change.

### 2. Couch policy loading and capture-bound decision

Files: create `cmd/internal/couchcmd/privacy.go`, `privacy_test.go`; modify `cmd/internal/couchcmd/webcast.go`, `webcast_test.go`, `cmd/internal/webcast/cast.go`, `cast_submission.go`, `cast_submission_test.go`, `cmd/internal/webcast/recorder/submission.go`, `submission_test.go`.

- [ ] Add temporary-file reader tests for missing/valid/invalid/oversized policy and edits between preparations. Add Cast integration tests with two origins and a real recording fixture, asserting final serialized bundles and refusal before capture for broken policy.
- [ ] Add recorder identity tests for origin changes during capture, document/tab replacement, missing origin, and matching origin across multiple crops. If #525 has landed, reuse its tests/identity field rather than duplicate it.
- [ ] Run `go test ./cmd/internal/couchcmd ./cmd/internal/webcast ./cmd/internal/webcast/recorder -run 'Privacy|Origin|Submission|Capture' -count=1`; confirm new tests fail for the intended missing behavior.
- [ ] Wire a read-only loader into CastConfig; load once during preparation; capture and compare origin; set decision before final validation/storage. Leave delivery identity and retry ownership intact. Missing loader means private. Tests exercise the production seam with the existing stateful doubles.
- [ ] Repeat the targeted command; commit the wiring.

### 3. Editable-value exclusion and agent guidance

Files: modify `cmd/internal/webcast/recorder/box_text.js`, `box_text_test.js`, `cmd/internal/webcast/live_submission_test.go`, `cmd/internal/submission/model.go`, `model_test.go`, `cmd/internal/couchcmd/skills/couch/SKILL.md`; add `cmd/internal/couchcmd/skill_privacy_test.go`.

- [ ] Read superpowers-test-driven-development and superpowers-writing-skills before the relevant implementation edits.
- [ ] Write failing fixtures containing unique typed-value sentinels, including editable text contributing to an ancestor's accessible name. Assert sentinel absence in both output groups and static-label presence. Extend the live fixture with real typed values and retained ordinary text.
- [ ] Run `go test ./cmd/internal/webcast/recorder -run BoxText -count=1`; implement the narrow traversal exclusion; rerun.
- [ ] Update the guide and shipped skill with decision 7. Add focused semantic assertions for default-private, the two crop conditions, summarization, operator-owned policy, and local expiring recordings. Retain existing command-pointer checks.
- [ ] Run `go test ./cmd/internal/submission ./cmd/internal/couchcmd ./cmd/internal/couchtty -run 'ReadingGuide|Privacy|Submission' -count=1`; commit text protection and guidance.

### 4. Documentation, verification and handoff

Files: modify `README.md`, `atlas/webcast-submission.md`; update existing atlas links only if a new page is necessary. Update issue Plan/Log and add Revisions for changed decisions.

- [ ] Document the operator-owned file path and exact-origin example, missing/error behavior, privacy JSON example, immutable per-bundle decision, private text handling, crop policy conjunction and local evidence lifetime. Do not create an actual shareable policy for the operator.
- [ ] Merge latest main as needed and reconcile #525's capture URL. Sweep for duplicated origin authority and old externalization guidance. Report integration status to hub if #525 remains pending.
- [ ] Run targeted packages, then `make test` in the background before any headless run or close, per dispatch. Inspect its completed result.
- [ ] Run `WEBCAST_LIVE_CHROME=1 GOMAXPROCS=2 go test -p 1 ./cmd/internal/webcast -run '^TestLiveSubmissionCrop$' -count=1 -timeout 60s -v` after the full suite. Assert actual form-value exclusion and static-text retention; a skip is not conformance evidence.
- [ ] Run `git diff --check`; record commands/results. If a failure changes code, rerun the affected checks and required full suite before close.
- [ ] `sdlc close --issue 526 --verified '<concrete evidence>'` owns the fresh-context review. Address findings and update lessons where warranted, then `sdlc pr`. Report actual `git rev-parse HEAD` and PR to hub. Await SHIP; then `sdlc merge --yes`, report landed outcome and stop.

## Approval checkpoint

The operator requirements arrived through the hub on 2026-10-12. Hub follow-up confirmed top-level privacy/privacy_rule, exact local operator-edited origins, default private, no edit CLI, main-based development and second-lander integration with #525. This durable plan also proposes exact file/schema/error behavior and conservative editable-value exclusion; send it to the hub for the required plan approval before `sdlc change-code`. Do not derive an estimate before plan-quality passes.

## Revisions

### 2026-10-13 — approved plan; invalid policy must not strand submission

Hub approved commit `5738858e62f3ab31716f641a7f1af8de87db4fa1` with the operator's no-limbo correction. This supersedes decision 3, the partial-JSON failure wording, and task 2's refusal expectation: missing policy remains private; unreadable, oversized or invalid policy makes every origin private, preserves submission/delivery, records `privacy_rule: "invalid policy file: <reason>"`, and emits a Couch notice naming the policy file. Reasons are bounded safe categories, never raw policy content. Valid matched rules remain objects, unmatched defaults null; the typed rule representation must support the explicit invalid-policy string variant and validate that it occurs only with private. Tests must prove successful preparation/delivery and the notice on these failure paths. No estimate until the plan-quality gate passes.
