# Boundary Review — pair#758 (whole-issue close)

| field | value |
|-------|-------|
| issue | 758 — Couch-owned editor endpoint registry and lifecycle |
| repo | pair |
| issue file | workshop/issues/000758-remote-session-couch-remote-server-and-allowlisted-proxy-to-each-session-file-s-nvim-subscribe-edit-re-wrapped-submit-navigate.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1e70379fba4a3653258f5542ac1d1aecf0ca1b43..1ad3d3b8ad267ae72c9aaba5b773eb3c6d5dfeaf |
| command | sdlc close --issue 758 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T06:01:46-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The producer has a well-separated authority model, durable storage, private CLI/IPC integration, and substantial passing tests. Two correctness defects block shipping: stale negative probes can report false absence, and a predictable capacity refusal can poison the entire endpoint owner. Notification lifecycle qualification also falls short of the checked-off plan.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: observation-generation-fencing
    title: |
      A late negative lookup reports confirmed absence despite a current successor
    detail: |
      cmd/internal/editorendpoint/owner.go:414-424 fences retirement against the captured binding but unconditionally returns Code=ok with EditorEndpoint=null afterward. If B replaces A while A's probe is pending, a subsequent nil response for A cannot establish absence of B. Recheck current authority under the same lock as retirement and return stale or validate the successor. Extend owner_test.go:234 to assert the delayed lookup response; line 260 currently discards it. ARCH-ORDER, ARCH-SECURE.
  - id: new
    severity: Critical
    kind: defect
    family: prepublication-refusal-isolation
    title: |
      Exceeding the encoded-state limit poisons otherwise healthy endpoint authority
    detail: |
      cmd/internal/editorendpoint/owner.go:129-131 treats every Store.Publish error as uncertain publication. However, store.go:112-114 can reject encoding before invoking any writer, including codec.go:139-140 rejecting a candidate above 8 MiB. A valid near-capacity registry therefore becomes unavailable for lookup, unregister, and cleanup until restart after one oversized admission. Preflight candidate encoding and classify size exhaustion as capacity without poisoning or changing authority; reserve publication uncertainty for attempted writes. Add an owner-level regression proving no writer call and continued operation after refusal. ARCH-CONSTRAINTS, ARCH-ORDER.
  - id: new
    severity: Important
    kind: defect
    family: external-lifecycle-qualification
    title: |
      Notification doubles cannot exercise the delayed and failed lifecycle promised by the plan
    detail: |
      cmd/internal/editorendpoint/owner_test.go:69-73 only appends notices and always succeeds; couchcmd/editor_endpoint_service_test.go:124-125 is a no-op. The native callback at nvim_live_test.go:65-68 only writes a token. None can hold or fail Superseded, although plan line 168 marks held-notification qualification complete. Add controllable notification completion/failure through the production seam and verify publication precedes delivery, failure preserves the successor, and shutdown cancels and joins an in-flight notification. Model pending/current attachment identity in the editor double. ARCH-MOCK, ARCH-ORDER, ARCH-PURPOSE.
```

1. **Strengths**

   - Retry, supersession, compare-release, and process collection run through one pure transition function.
   - The retained-state codec preserves unknown fields at every owned object level, including numbers outside float64 range, while retaining strict known-field validation.
   - Private native tests use isolated environments and owned process cleanup.
   - README and atlas changes cover the new commands, authority model, lifecycle, and producer-only scope.

2. **Critical findings**

   - Fence the **negative lookup response**, as well as the retirement mutation, against successor generations.
   - Separate deterministic prepublication refusal from uncertain storage publication.

3. **Important findings**

   - Complete controlled notification lifecycle tests and correct the plan’s qualification claim.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed full package tests for `editorendpoint`, `strictjson`, `procutil`, `sessionsync`, and `couchcmd`. Focused endpoint/native/codec/process tests also passed with race detection. `git diff --check` passed.

   Existing coverage checks successor preservation but discards the delayed lookup’s response. Codec size tests do not check owner availability after size refusal. No repository-wide recorded suite result is claimed by this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared namespace, document identity, transport, durability, and JSON helpers are reused.
   - **ARCH-PURE — pass:** authority decisions and codecs are independently testable without IO.
   - **ARCH-PURPOSE — flag:** checked-off notification qualification is incomplete; the documented producer-only split otherwise matches implementation.
   - **ARCH-MOCK — flag:** notification doubles lack controllable stateful lifecycle behavior.
   - **ARCH-CONSTRAINTS — flag:** byte-cap refusal incorrectly disables healthy authority.
   - **ARCH-SECURE — flag:** evidence about a displaced attachment becomes an unsupported claim of document absence.
   - **ARCH-ORDER — flag:** negative response generation checks and publication outcome classification need correction.
   - **ARCH-FUNERAL — pass:** retained evidence has bounded storage and positive-death collection; handles are joined and editor-owned paths remain externally owned.
   - **ARCH-EVOLVE — pass:** nested unknown fields survive retained-state rewrites with strict validation.

7. **Plan revision recommendations**

   Add concrete `## Revisions` entries specifying:

   - Negative lookup completion atomically checks current authority before confirming absence.
   - Encoding/capacity refusal occurs before publication and leaves the owner usable.
   - Notification qualification includes held, failed, and shutdown-interrupted delivery; update completion claims after those tests pass.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7a1f5416-a281-4580-9698-8074dacfef93","repository":"github.com/xianxu/pair","issue_id":"000758","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/653fd1bc9ae6485d","reviewed_head":"1ad3d3b8ad267ae72c9aaba5b773eb3c6d5dfeaf","slot":"pair:6","requested_at":"2026-10-22T12:58:08.115608Z"},"phase":"finished","pid":5890,"started_at":"2026-10-22T12:58:08.24957Z","finished_at":"2026-10-22T13:01:46.159896Z","session_id":"01a163b0-b992-73d2-ba88-164aae0a719c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.738942208s
head: "1ad3d3b8ad267ae72c9aaba5b773eb3c6d5dfeaf"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
