# Boundary Review — pair#576 (whole-issue close)

| field | value |
|-------|-------|
| issue | 576 — Couch switcher: lifecycle actions run in the background per slot; the UI never blocks |
| repo | pair |
| issue file | workshop/issues/000576-couch-switcher-lifecycle-actions-run-in-the-background-per-slot-the-ui-never-blocks.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1065d70ca3796af3bb97892eaa99f5c1bcc38e87..1fc259181880f60916941eb7a09c88c6d40e9d47 |
| command | sdlc close --issue 576 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T17:27:42-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The main async path is well implemented and tested: hosted navigation remains available during lifecycle work and adoption. One blocking gap remains: temporary start reservations lose their association with a newly listed conversation before completion, allowing another job on that target and hiding its progress.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: target-ownership-survives-identity-materialization
    title: |
      A newly listed conversation bypasses its running start reservation
    detail: |
      cmd/internal/couchtty/menu_operations.go:121 reserves starts without an exact target under StartPath, but admission at lines 164–165 compares only exact keys. Row lookup and rendering likewise resolve only RowKey or ResultAddress, which is assigned at final completion. If inventory lists the new conversation while start/adoption remains running, its row has no job progress and a second action is admitted instead of returning busy: start. A temporary Go-overlay regression reproduced both failures on the pinned head. Preserve one reservation across temporary-path and authoritative-row identities, and share that resolution across admission, projection, busy selection and address lookup; cover inventory arrival during held adoption. This violates ARCH-ORDER and ARCH-PURPOSE.
```

1. **Strengths**

   - Separate bounded lifecycle/navigation lanes preserve hosted navigation under lifecycle saturation, including admission capacity.
   - Adoption preparation runs off the Console loop; installation releases `c.mu` during registration and revalidates before committing.
   - Real Console tests hold lifecycle work and adoption independently while proving input delivery to another slot and hub.
   - README and atlas document the new behavior; both documentation gates are satisfied.

2. **Critical findings**

   The finding above blocks shipping. This is a reachable production case: primary-slot creation can leave `StartResolution.Target` unset (`cmd/internal/couchcore/slotstart.go:135`), selecting the temporary-path reservation. Resolve the identity transition without releasing ownership or changing the immutable attempt origin.

3. **Important findings**

   None beyond the regression coverage required for the Critical finding.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed independently:

   - Full `go test ./cmd/internal/couchtty -count=1`.
   - Focused navigation, adoption, installation, shutdown and queue tests under `-race`, three repetitions.
   - Complete `artifactpath` and `termcmd` packages.
   - Pinned-range `git diff --check`.

   The additional overlay regression failed because the newly listed row lacked its running job and `detach` received attempt 2 while attempt 1 remained running. The overlay was removed; the checkout remains unchanged. I did not rerun the repository-wide suite.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** existing workers, operation declarations and lifecycle publisher are reused.
   - **ARCH-PURE — pass:** admission, result reduction and row projection remain pure; IO stays in Console.
   - **ARCH-PURPOSE — flag:** fresh-start identity materialization escapes the promised per-target ownership rule.
   - **ARCH-MOCK — pass:** inspected integration tests use controlled dispatchers, stateful fake children and private fixtures.
   - **ARCH-CONSTRAINTS — pass:** bounded lanes and saturation tests establish reserved navigation capacity.
   - **ARCH-SECURE — pass:** typed identities and sanitized, bounded row text are preserved.
   - **ARCH-ORDER — flag:** the transition from temporary provisioning identity to listed conversation is missing.
   - **ARCH-FUNERAL — pass:** outcomes are pruned, pending installations have cleanup paths, and workers join at shutdown.

7. **Plan revision recommendations**

   Add a `## Revisions` entry specifying how temporary start identities bind to authoritative inventory rows while execution/adoption remains running. Enumerate admission, rendering, busy-row selection and automatic-work lookup as consumers of that binding. Require a regression that materializes inventory during held adoption and verifies visible progress, `busy: start`, and exactly one admitted lifecycle job.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"cef4afb4-572f-42b9-976b-8261d302ead8","repository":"github.com/xianxu/pair","issue_id":"000576","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/426e924c03adc9c0","reviewed_head":"1fc259181880f60916941eb7a09c88c6d40e9d47","slot":"pair:7","requested_at":"2026-10-19T00:23:49.351744Z"},"phase":"finished","pid":32875,"started_at":"2026-10-19T00:23:49.461135Z","finished_at":"2026-10-19T00:26:57.732742Z","session_id":"01a1518b-0d95-7a43-b1a7-e69a735e41e1","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.930202292s
head: "1fc259181880f60916941eb7a09c88c6d40e9d47"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T17:57:04-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 576 — Couch switcher: lifecycle actions run in the background per slot; the UI never blocks |
| repo | pair |
| issue file | workshop/issues/000576-couch-switcher-lifecycle-actions-run-in-the-background-per-slot-the-ui-never-blocks.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1065d70ca3796af3bb97892eaa99f5c1bcc38e87..9f58ddf535ce1cd4c3ad09a97a30bf1c527adef9 |
| command | sdlc close --issue 576 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T17:57:04-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation satisfies the per-slot background-job contract. BR-1 is addressed with regression tests that fail without the correction. Review found no remaining blocking defects; README and atlas updates cover the new behavior.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Original reservations survive identity materialization through derived aliases shared by admission, row lookup, busy selection and address lookup. Both materialization regressions pass at head; replacing only menu_operations.go with its pre-fix version through a temporary Go overlay makes both fail, including the real Console.Run held-adoption case.
```

1. **Strengths**

   - Ownership spans execution and adoption without changing the original reservation or attempt. Symmetric overlap and negative identity controls cover the broader rule ([menu_operations.go:81](cmd/internal/couchtty/menu_operations.go:81)).
   - Navigation reserves both queue admission and worker capacity. Lost hosted proof transfers the same request to lifecycle execution or reports overload ([operation_queue.go:136](cmd/internal/couchtty/operation_queue.go:136)).
   - Adoption preparation runs outside the input loop; installation releases the Console mutex across registration and revalidates before committing ([console.go:501](cmd/internal/couchtty/console.go:501)).
   - Real Console tests hold lifecycle and adoption separately, deliver input to another slot and the hub, and verify late completion preserves focus.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed:

   - Complete `couchtty` package: 61.998s.
   - Focused operation, identity and navigation tests.
   - Focused race cohort, repeated three times.
   - Complete `artifactpath` and `termcmd` packages.
   - Pinned-range whitespace check.

   The BR-1 overlay negative control failed as expected. Temporary artifacts were removed; repository files remain unchanged. The repository-wide suite and live acceptance were not rerun during this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Reuses existing workers, lifecycle gates, scope derivation and publication seams.
   - **ARCH-PURE — pass:** Admission, identity resolution, result reduction and projection remain deterministic; Console owns IO.
   - **ARCH-PURPOSE — pass:** Background execution includes adoption, independent navigation, busy refusal and row outcomes.
   - **ARCH-MOCK — pass:** Controlled dispatchers, stateful FakeRunner and terminal fixtures share production boundaries; existing native conformance coverage remains.
   - **ARCH-CONSTRAINTS — pass:** Bounded lanes enforce the declared 15+1 admission envelope; saturation and linear projection have deterministic checks.
   - **ARCH-SECURE — pass:** Aliases derive from accepted identities rather than display names or cwd guesses; diagnostics are bounded and sanitized for rendering.
   - **ARCH-ORDER — pass:** Attempt matching, tagged phases, landing intent and completion ownership are exercised with controlled interleavings.
   - **ARCH-FUNERAL — pass:** Missing inactive outcomes are pruned; pending installation reservations and registrations have cleanup paths; shutdown joins workers and settles pending callbacks.

7. **Plan revision recommendations:** None. The recorded revisions match the implementation; live acceptance remains explicitly deferred to the approved restart window.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"4fd4dc87-33e7-497c-8cc7-a2b9742f31bb","repository":"github.com/xianxu/pair","issue_id":"000576","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/c5371658d78435ad","reviewed_head":"9f58ddf535ce1cd4c3ad09a97a30bf1c527adef9","slot":"pair:7","requested_at":"2026-10-19T00:54:39.599527Z"},"phase":"finished","pid":5496,"started_at":"2026-10-19T00:54:39.890404Z","finished_at":"2026-10-19T00:57:03.164571Z","session_id":"01a151a7-4a68-7e63-95f4-712cba8fd13f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.349403083s
head: "9f58ddf535ce1cd4c3ad09a97a30bf1c527adef9"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
