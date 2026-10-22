# Boundary Review — pair#796 (whole-issue close)

| field | value |
|-------|-------|
| issue | 796 — Couch: --tab-style collapse (one row; fold quiet slots and inactive hubs to +N, attention always shown) alongside multiline |
| repo | pair |
| issue file | workshop/issues/000796-couch-tab-style-collapse-one-row-fold-quiet-slots-and-inactive-hubs-to-n-attention-always-shown-alongside-multiline.md |
| boundary | whole-issue close |
| milestone | — |
| window | 651e30092e7ca245e5537ba1450b7ea7eb36ddae..bccaa4617a2fa49cf6a530cfab72fc04bb581b04 |
| command | sdlc close --issue 796 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T14:28:29-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The launch wiring, shared renderer, scoped switcher, and geometry tests are sound. One verified policy defect blocks shipping: numbered hubs do not receive the anchor preservation and combined folding promised by the plan.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: collapse-hub-role-policy
    title: |
      Numbered hubs violate the approved collapse policy
    detail: |
      cmd/internal/couchtty/status_collapse.go:13-39 uses legacy chipGroups runs and preserves quiet anchors only when SlotNumber == 0. A temporary regression overlay at width 100 produced "repo:1 +1  [:2]" for a quiet current hub at slot 1 with active worker 2, folding the current hub despite ample space. Making both members quiet under an inactive hub produced "repo:1 +1  repo:2 +1", with two count spans for the same (OwnerHub, GroupKey). The plan at lines 21-23 requires preserving the current hub when space allows and folding quiet inactive hubs with their workers into one count per identity. Preserve anchors by hub role and aggregate collapse counts by exact identity without changing multiline grouping; add primary/numbered × current/inactive regression cases.
```

1. **Strengths**

   - CLI validation and actual Console configuration are tested, including opaque message arguments.
   - Count spans come from the shared drawing pass and use the existing confirmed-revision mouse fence.
   - Scoped-menu tests cover owner collisions, disappearing membership, nested return, and Escape recovery.
   - Native tests retain kernel PTY-size and physical bottom-row assertions across both styles. README and atlas describe the new surface.

2. **Critical findings**

   - **Numbered-hub policy:** [status_collapse.go:13](/Volumes/TestScratch/review-checkouts/0802444ba0bdba70/cmd/internal/couchtty/status_collapse.go:13), with the anchor predicate at line 27. **ARCH-PURPOSE:** legacy display runs are insufficient for the promised collapse identity. Preserve the numbered current hub and consolidate inactive members sharing the exact owner/group.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Targeted collapse, scope, parser, and runtime-wiring tests passed.
   - Status, broadcast, privacy, annotation, chrome, and native wrap selections passed.
   - A temporary Go overlay reproduced both numbered-hub failures; it was removed and the repository remains clean.
   - Existing inactive-hub coverage uses slot zero, missing the numbered-hub cases. No prior findings required disposition.
   - Recorded-suite success and staged operator acceptance remain outstanding acceptance gates.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared packing, drawing, and presentation authority.
   - **ARCH-PURE — pass:** collapse selection and scope filtering remain pure.
   - **ARCH-PURPOSE — flag:** numbered-hub behavior underdelivers the approved policy.
   - **ARCH-MOCK — pass:** existing terminal/process doubles and native conformance harness are reused.
   - **ARCH-CONSTRAINTS — pass:** production-loop work counters exercise bounded planning.
   - **ARCH-SECURE — pass:** shared sanitization and confirmed-geometry fencing remain in place.
   - **ARCH-ORDER — pass:** scope recovery and refresh reconciliation use the existing menu model.
   - **ARCH-FUNERAL — pass:** no new durable artifacts or background owners.
   - **ARCH-EVOLVE — pass:** no persisted-format changes.

7. **Plan revision recommendations**

   Add a `## Revisions` entry specifying numbered-hub coverage and distinguishing legacy multiline display runs from collapse’s exact owner/group aggregation. Retain the approved anchor and folding contract.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"862aa34a-53af-4ca3-babd-4f4ccbcf6019","repository":"github.com/xianxu/pair","issue_id":"000796","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/0802444ba0bdba70","reviewed_head":"bccaa4617a2fa49cf6a530cfab72fc04bb581b04","slot":"pair:4","requested_at":"2026-10-22T21:26:26.57925Z"},"phase":"finished","pid":68160,"started_at":"2026-10-22T21:26:27.096314Z","finished_at":"2026-10-22T21:28:28.174404Z","session_id":"01a16582-1b94-7642-af75-73c61154c955","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 5.360233666s
head: "bccaa4617a2fa49cf6a530cfab72fc04bb581b04"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
