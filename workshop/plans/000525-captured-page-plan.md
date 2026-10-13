# Captured page identity implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to choose delegation versus warm-context execution. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Every new DOM submission identifies the page its current evidence came from, preserves structured text and useful element metadata, and emits no CSS selector fields.

**Architecture:** Read URL/title from the admitted top-frame CDP context and include that immutable page value in the existing capture fence. The submission source owns the batch's page identity; each note references its document. Privacy derives from that same URL, while recording/query selectors retain their existing independent purpose.

**Tech Stack:** Go, embedded JavaScript, CDP, existing stateful browser fixtures, Node collector fixtures and live Chrome.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| PageIdentity | `cmd/internal/submission/model.go` | new |
| Source / Note / BoxTextEntry / Validate | `cmd/internal/submission/model.go` | modified |
| PrivacyDecision integration | `cmd/internal/submission/privacy.go`, `model.go` | modified |
| Build | `cmd/internal/webcast/submission/build.go` | modified |
| ReadingGuide | `cmd/internal/submission/model.go` | modified |

One Source owns one optional PageIdentity (`url`, `title`, `provenance: "cdp"`) for its existing document number. Each Note has `source_document` referencing that number. A future multi-document submission can extend the source collection without duplicating URL authority per note. BoxTextEntry gains bounded `element` markup containing tag/classes; the legacy selector field remains decode-compatible but new version-2 output omits it. Build projects a copied anchor with `Sel` cleared; original recording notes and query selectors remain intact.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| CaptureCrops / captured page probe | `cmd/internal/webcast/recorder/submission.go` | modified | admitted CDP context and capture barriers |
| Box-text collector | `cmd/internal/webcast/recorder/box_text.js` | modified | DOM traversal in isolated world |
| PrepareSubmission | `cmd/internal/webcast/cast_submission.go` | modified | captured batch, immutable policy, bundle limits |

Reuse the recorder stateful fake CDP page/context model (extend it with title and page-probe behavior), Cast submissionBrowser fixture and portable store. Live Chrome conformance verifies the real document URL/title and collector metadata.

## Design decisions

1. Prefer a direct CDP read over the recording head or polled tab metadata. A recording-head lookup is cheap but can be stale after navigation; recorder state tracks navigation but receives title via tab polling. A bounded read in the admitted context names the actual document used for current evidence.
2. Add `source.page` with URL/title/provenance and `notes[].source_document`. Emit version 2 for new DOM bundles; continue reading version 1, including old selector-bearing bundles. Version-2 validation requires each note reference to equal source.document, and any available crop to have a valid source.page. A page-changed/unavailable batch has no claimed captured page and is private. No historical fallback is labeled as a current capture. Build's intermediate uncaptured bundle remains valid because its crops are explicitly unavailable.
3. Read URL/title from document.URL/document.title in the admitted isolated context with Runtime.callFunctionOn. Bind the page value to the start identity and compare it across existing before/after capture fences, including after each text/pixel operation and between notes. Retain document/frame/loader/tab/live/context checks and the current recorder URL fence (full URL rather than only origin). Reject changed or unconfirmed identity with the existing resubmit error; no partial bundle is published. The probe does not mutate the page. An empty title is legitimate. Bound URL to 16 KiB and title to 8 KiB (design limits), checking in JavaScript before transfer and in Go before admission; oversize identity fails clearly instead of truncating authority.
4. CapturedCrop carries PageIdentity instead of a separately authored Origin. PrepareSubmission requires all available crops to agree on page/document, installs source.page, and calls policy.Decide(page.URL). Persisted privacy_origin remains a compatibility/derived field, validated against OriginOf(source.page.url) for version 2. A missing/unavailable page cannot grant sharing. Version-1 metadata remains readable with #526's default/permission rules. Keep exact-key rejection at all permission boundaries and extend it to the new authoritative page/reference fields so aliases cannot bypass the binding.
5. Clear at.sel only in the new submission projection; preserve at.node (@N), tag/classes in element, text, semantic path/box and note context. Keep recording.NoteAt.Sel, notes.Structural selector resolution and query --sel behavior unchanged. Tests must show the original recording survives projection unchanged.
6. Remove selector-path construction from box_text.js. Carry bounded tag/class markup for the actual owning element through ordinary, shadow and slot traversal; use it for visible text and accessible names. Preserve per-text-node granularity, reading order, name kinds, clipping, bounded traversal and typed-value exclusion. Do not group by tag/classes or parent: distinct elements can share classes, and regrouping mixed text changes order. This clarifies the original spec's informal "per element" wording; hub explicitly approved preservation of current structure.
7. Keep the legacy selector field readable in version 1; new version 2 rejects nonempty selector fields and anchor Sel. Do not weaken strict decoding to accommodate old bundles. New writers omit empty selector fields via omitempty. Store retry/reopen behavior remains frozen and version-aware; there is no eager rewrite of existing bundles.
8. Update how_to_read, shipped Couch skill where needed, README schema example and atlas/webcast-submission.md. Explain source.page, per-note document references, CDP provenance, historical @N versus current element text, absent page for unavailable capture, legacy compatibility and privacy derived from the same URL. Issue externalization still sanitizes URL/text under #526. This task stores local evidence; it adds no uploader or authority to edit policy.

## Architecture checks

- **ARCH-DRY:** page identity is shared by evidence and privacy; no parallel current-origin authority. Existing capture fences, validation and delivery machinery stay owners.
- **ARCH-PURE:** PageIdentity/schema validation and Build projection stay pure. CDP probe and DOM collection remain IO adapters.
- **ARCH-PURPOSE:** sweep source identity, note references, anchor selector projection, both box-text arrays, privacy derivation, decoder compatibility, guide and schema documentation.
- **ARCH-MOCK:** extend existing stateful fake page/context/identity events rather than mock isolated function calls; run actual Chrome conformance after full suite.
- **ARCH-CONSTRAINTS:** asynchronous submission, existing 30-second batch/5-second CDP operation timeouts and 64 MiB bundle cap. URL/title limits above add bounded per-batch metadata, not a background poller. Collector retains 8192-byte group, 256-entry and 20000-node limits including element metadata. No new concurrency or network service.
- **ARCH-SECURE:** CDP metadata, persisted manifests and DOM text are untrusted inputs. Validate shape/bounds/provenance and exact authoritative keys; legacy v1 stays readable, malformed v2 refuses, unavailable capture defaults private. No credentials created, no real operator policy touched.
- **ARCH-ORDER:** reuse current capture admission → text → verified identity → pixels → verified identity loop and submission reducer. Navigation, same-document URL change, title change, context expiry, tab switch, pause/end and cancellation invalidate admission; late/failed probe never supplies identity. Existing bounded caller context owns all work; no spawned task outlives return. Persisted retry uses frozen source page without recapture.
- **ARCH-FUNERAL:** page metadata lives in the existing bounded submission.json and follows operator bundle cleanup/admission refusal at capacity. No artifact family or accumulating sidecar is introduced.

## Tasks (one issue-close boundary)

### 1. Schema, compatibility and projection

Files: `cmd/internal/submission/{model.go,privacy.go,model_test.go,privacy_test.go,store_test.go}`, `cmd/internal/webcast/submission/{build.go,build_test.go}`.

- [x] Write failing schema/projection/store tests using the strategies below.
- [x] Run `go test ./cmd/internal/submission ./cmd/internal/webcast/submission -run 'Page|Source|Selector|Build|Privacy' -count=1` and confirm intended red results.
- [x] Add PageIdentity, version-2 reference/permission validation and legacy compatibility. Project copied anchors without Sel and keep original recording data intact.
- [x] Repeat targeted command to green and commit the coherent schema change.

### 2. Capture-bound page and privacy

Files: `cmd/internal/webcast/recorder/{submission.go,submission_test.go,fakecdp_test.go}`, `cmd/internal/webcast/{cast_submission.go,cast_submission_test.go}`.

- [x] Extend stateful fake page metadata and write controlled identity-change tests; run `go test ./cmd/internal/webcast/recorder ./cmd/internal/webcast -run 'CropCapture|Submission' -count=1` for red evidence.
- [x] Add bounded admitted-context metadata read and integrate it into capture fences. Replace CapturedCrop.Origin with page evidence; derive Cast policy from its URL and enforce batch consistency.
- [x] Repeat targeted tests; run affected privacy tests; commit capture wiring.

### 3. Structured selector-free text

Files: `cmd/internal/webcast/recorder/{box_text.js,box_text_test.js}`, `cmd/internal/webcast/live_submission_test.go`, affected BoxTextEntry fixtures.

- [x] Write failing collector fixtures for element metadata and absence of selectors, preserving existing privacy/clipping tests; run `go test ./cmd/internal/webcast/recorder -run BoxText -count=1` for red evidence.
- [x] Remove selector path construction and carry owning element markup within existing byte/entry budgets. Preserve granularity and order, including mixed/shadow/slotted text.
- [x] Update live fixture to identify clipping subjects by tag/classes and assert actual page URL/title; rerun Node/Go tests and commit.

### 4. Guidance and verification

Files: `cmd/internal/submission/model.go`, `cmd/internal/couchcmd/skills/couch/SKILL.md`, `README.md`, `atlas/webcast-submission.md`, issue/plan/lessons.

- [x] Update guidance and schema examples consistently. Preserve privacy gates and @N query instructions.
- [x] Check latest main for #524 capture changes; merge main only, preserving settlement/viewport logic if it lands first. No stacking.
- [x] Run affected packages, targeted race checks, `git diff --check` and `make build`. Run full `make test` in the background before close/headless conformance; inspect completion. Known #514 timing waiver from #526 is not silently generalized to new failures.
- [x] Run `WEBCAST_LIVE_CHROME=1 GOMAXPROCS=2 go test -p 1 ./cmd/internal/webcast -run '^TestLiveSubmissionCrop$' -count=1 -timeout 60s -v`; skips are not evidence. Log results and commit.

Handoff: `sdlc close --issue 525 --verified '<evidence>'` owns fresh-context review; address findings, open PR, then merge as authorized after CI is green and report actual commit hashes/PR to hub.

## Function-level verification strategies

| Function / boundary | Adversarial class and mechanical guard |
|---|---|
| PageIdentity / Validate / manifest decode | Mutate bounded metadata, document references, privacy origins and exact JSON keys; inconsistent v2 identities never validate. Round-trip legacy v1 and new v2 through the strict decoder. |
| Build | Project real recorded anchors across selected notes; new serialization omits CSS fields while node handles, classes, text/order and original records remain intact. |
| CaptureCrops | Stateful fake CDP changes URL/title/context/navigation at controlled fences; pixels/text are returned only with the admitted page. Test failures and cancellation at metadata reads, not just success. |
| PrepareSubmission | Feed mismatched or unavailable capture identities through the stateful browser seam; source references and privacy must derive from one agreed captured URL, never recording head or viewer URL. |
| Box-text collector | Compare ordered sentinel text across mixed DOM/shadow/slot structures before/after selector removal; budgets, clipping and editable-value exclusions remain invariant while all emitted entries have appropriate bounded element metadata. |
| Store reopen/retry | Persist real temporary bundles, reopen and advance delivery; URL/title/references/privacy remain frozen and altered retries are refused. Older selector-bearing bundles still load without granting new authority. |
| Live Chrome | Navigate a local fixture, change same-document URL/title, capture actual form/clipped text, and verify reported source identity and selector-free structured output. |

## Revisions

### 2026-10-13 — hub clarification and #526 integration

Hub approved preserving existing per-text-node granularity/order and one source page bound to the batch with per-note references. #526 landed at 3b5d1ac6edb6cf156ba30d8b2a264324ca33effc before this branch; this issue now consumes its matcher and strict permission decoding. The previous apparent skeleton was a stale checkout; refreshed main contains the complete operator spec.
