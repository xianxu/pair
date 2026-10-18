# Boundary Review — pair#622 (milestone M2)

| field | value |
|-------|-------|
| issue | 622 — Couch server spends ~1 core emulating every hosted client's output in a vt parser, visible or not |
| repo | pair |
| issue file | workshop/issues/000622-couch-server-spends-1-core-emulating-every-hosted-client-s-output-in-a-vt-parser-visible-or-not.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | fcc03aa5c56197583ccf4a25546a0ffc19b1c54c..568f5476283ab7170f3ae6cc6bd8a007982e060a |
| command | sdlc milestone-close --issue 622 --milestone M2 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T11:20:59-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned M2 changes preserve eager terminal authority while avoiding general grapheme segmentation for narrowly proven ASCII cases. Independent tests and retained performance evidence support landing. No blocking findings; production CPU and console/peek/waiting acceptance remain explicitly pending for whole-issue closure.

```findings
{}
```

1. **Strengths**

   - `third_party/vt/utf8.go:21` retains provisional cluster amendment, existing limits and the normal rendering transition.
   - `cmd/internal/terminal/frame.go:118` preserves text, link, width and neighbor validation around the shortcut.
   - The VT differential reference exactly matches the pinned pre-optimization method. Tests compare state, replies and effects across chunking, Unicode continuations and resize.
   - Independently verified all 28 retained benchmark logs, binary hashes and load guards. Recomputed complete-redraw CPU reductions match the report: ASCII 13.68–15.57%; mixed 2.84–6.30%.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Fresh full VT module tests passed.
   - Fresh `go test -race ./cmd/internal/terminal ./cmd/internal/ptychild -count=1` passed.
   - Inspected the recorded full-suite log and confirmed its tested head differs from the review head only in issue/plan prose.
   - Exhaustive ASCII, invalid-cell, generated Unicode and stateful differential tests exercise the changed decisions. Existing tests cover hidden replies, effects, synchronization, resize, input modes and diagnostic-ring eviction.
   - `git diff --check` passed.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** existing Unicode and rendering paths remain authoritative.
   - **ARCH-PURE — pass:** frame validation and local ASCII decisions remain deterministic; emulator mutation stays within its existing integration boundary.
   - **ARCH-PURPOSE — pass:** complete-ingest measurements qualify M2; live acceptance remains a tracked deliverable.
   - **ARCH-MOCK — pass:** benchmarks and consumer tests reuse the stateful Child/Endpoint seams and real parser.
   - **ARCH-CONSTRAINTS — pass:** no added workers, buffers or per-byte clocks; representative complete workloads were measured.
   - **ARCH-SECURE — pass:** safety checks remain ahead of the validation shortcut.
   - **ARCH-ORDER — pass:** existing state transitions retain ownership; differential sequence tests check ordering-sensitive state.
   - **ARCH-FUNERAL — pass:** no new product artifact family or persistent resource is introduced.

   Atlas documents the mechanism and qualification limits. No README update is needed because M2 introduces no command, flag, keybinding or configuration surface.

7. **Plan revision recommendations:** None. The inventory matches the implementation, and existing revisions distinguish M2 landing from pending production acceptance.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"2cf4e200-630a-485a-9b33-a29786a35599","repository":"github.com/xianxu/pair","issue_id":"000622","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/34ed979eb906ba79","reviewed_head":"568f5476283ab7170f3ae6cc6bd8a007982e060a","slot":"pair:5","requested_at":"2026-10-17T18:19:11.012194Z"},"phase":"finished","pid":66932,"started_at":"2026-10-17T18:19:11.419446Z","finished_at":"2026-10-17T18:20:58.834973Z","session_id":"01a14b16-dca6-7a53-9e4d-4171c9b5bd9a","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.021497709s
head: "568f5476283ab7170f3ae6cc6bd8a007982e060a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
