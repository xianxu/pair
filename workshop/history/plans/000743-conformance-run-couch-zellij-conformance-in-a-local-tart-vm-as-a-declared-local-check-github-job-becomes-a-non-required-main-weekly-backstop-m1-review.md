# Boundary Review — pair#743 (milestone M1)

| field | value |
|-------|-------|
| issue | 743 — conformance: run couch-zellij-conformance in a local Tart VM as a declared local check; GitHub job becomes a non-required main/weekly backstop |
| repo | pair |
| issue file | workshop/issues/000743-conformance-run-couch-zellij-conformance-in-a-local-tart-vm-as-a-declared-local-check-github-job-becomes-a-non-required-main-weekly-backstop.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e30ea629818a733f65741bfc95c11de8e1868f34..784562d8fa7168ae60e053935f6dc59d3af47290 |
| command | sdlc milestone-close --issue 743 --milestone M1 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T22:03:33-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 delivers substantial isolation, ownership, bounded transport, tests, and documentation. The targeted tests pass. Two correctness gaps block SHIP: cancelled provisioning can activate an image, and publication can replace the active manifest with one its reader rejects. The core-concepts table also needs correction under the requested consistency gate.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: activation-requires-success-authority
    title: |
      Cancelled provisioning can still activate the new image
    detail: |
      cmd/internal/vmcmd/provision.go:87-100 decides sealing and activation from collected/executed/stopConfirmed flags, without the RunState cancellation outcome. Cancel during successful Stop: runtime.go:131-133 records cancellation, but Release receives the detached cleanup context and can still SaveImage and publish the manifest. This violates the failed-provision-preserves-active-image contract and ARCH-ORDER. Make activation eligibility an explicit state-owned decision; add a controlled cancellation regression asserting nonzero exit, unchanged active manifest, and no activation.
  - id: new
    severity: Critical
    kind: defect
    family: persisted-writer-reader-bounds
    title: |
      Image publication can install an unreadable active manifest
    detail: |
      cmd/internal/vmcmd/provision.go:212-229 writes and renames encoded image metadata without enforcing the 64 KiB limit imposed by readMachineImage at lines 113-119. An individually valid active manifest with large preserved unknown metadata, combined with an individually valid new tool report, can exceed that limit. Provisioning then succeeds while subsequent runs reject image.json. Enforce the shared size limit before publication, preserving the previous manifest on overflow; test exact-limit and over-limit publication with preserved unknown fields. ARCH-CONSTRAINTS.
  - id: new
    severity: Critical
    kind: defect
    family: core-concept-code-traceability
    title: |
      M1 core-concepts locations contradict the delivered files
    detail: |
      workshop/plans/000743-local-vm-conformance-plan.md:59-60 names runtime_darwin.go, runtime_other.go, and positions.go, which do not exist at the pinned head. Native runtime responsibilities are implemented in host.go and driver_darwin.go/driver_other.go; positions are in positions_unix.go/positions_unsupported.go. Correct the greppable table and add a Revisions entry mapping these M1 entities to their actual files. This is Critical under the requested Core concepts consistency rule; the functionality itself is present.
```

1. **Strengths**

   - Git and Tart share subprocess ownership that retires signalling before reaping, with controlled lifetime tests.
   - Guest results use bounded transport and rooted host writes rather than a writable host share.
   - Stateful fakes cover partial failures, cleanup uncertainty, and cancellation barriers.
   - README and atlas document both public commands, isolation, resource limits, and outstanding M2/M3 work.

2. **Critical findings**

   The three findings above require correction and review before crossing M1. For cancellation, extend `TestProvisionRuntimeSealsOnlyQualifiedStoppedVM` with cancellation during export/Stop and verify that activation never occurs. For publication, extend `TestPublishProvisionRequiresExactToolsAndAtomicManifest` with combined metadata overflow.

3. **Important findings**

   None additional.

4. **Minor findings**

   None additional.

5. **Test coverage notes**

   Passed:

   - `go test ./cmd/internal/vmcmd ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
   - `python3 -B scripts/vm/test_provision.py` — nine tests

   The existing provisioning tests omit cancellation-dependent activation and oversized final manifests. Native qualification artifacts were inspected; real VM operations and mutation testing were not performed during this read-only review. No prior findings require disposition.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** shared subprocess ownership and strict JSON helpers are reused.
   - **ARCH-PURE — pass:** request, manifest, transition, and admission decisions have pure tests; IO uses runtime/driver seams.
   - **ARCH-PURPOSE — pass:** M1 supplies the generic runner/image; strict conformance and nightly handoff remain explicitly assigned to M2/M3.
   - **ARCH-MOCK — pass:** stateful runtime/native doubles share production boundaries, supplemented by retained native qualification.
   - **ARCH-CONSTRAINTS — flag:** final manifest publication lacks the reader’s size bound.
   - **ARCH-SECURE — pass:** inspected paths use private roots, literal argv, clean guest environments, and bounded exports.
   - **ARCH-ORDER — flag:** provisioning activation bypasses the reducer’s cancellation authority.
   - **ARCH-FUNERAL — pass:** artifacts, snapshots, claims, and image candidates have cleanup or explicit saturation/refusal policies.
   - **ARCH-EVOLVE — pass for codec paths:** unknown fields survive manifest/claim round trips and retained result encoding. Publication must also respect storage limits.

7. **Plan revision recommendations**

   Add a `## Revisions` entry correcting M1’s runtime/position file mapping. Extend the canonical provisioning strategy to require cancellation-safe activation and exact/over-limit final manifest publication tests. Keep M2/M3 unchecked until their separate deliverables are verified.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"12905d93-bd0d-4524-a31e-02c22afb0b59","repository":"github.com/xianxu/pair","issue_id":"000743","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/a14ad98fe2ac673c","reviewed_head":"784562d8fa7168ae60e053935f6dc59d3af47290","slot":"pair:3","requested_at":"2026-10-21T05:00:12.090035Z"},"phase":"finished","pid":26028,"started_at":"2026-10-21T05:00:12.248648Z","finished_at":"2026-10-21T05:03:33.021964Z","session_id":"01a15cd4-cdc5-7e83-b5cf-b71fdc577ede","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.284756208s
head: "784562d8fa7168ae60e053935f6dc59d3af47290"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T22:16:38-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 743 — conformance: run couch-zellij-conformance in a local Tart VM as a declared local check; GitHub job becomes a non-required main/weekly backstop |
| repo | pair |
| issue file | workshop/issues/000743-conformance-run-couch-zellij-conformance-in-a-local-tart-vm-as-a-declared-local-check-github-job-becomes-a-non-required-main-weekly-backstop.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e30ea629818a733f65741bfc95c11de8e1868f34..ca1e57116e68be45336ef22d0b7f8b80f1681234 |
| command | sdlc milestone-close --issue 743 --milestone M1 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T22:16:38-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range satisfies the revised M1 scope. All three prior findings are addressed, with executable regression evidence for both behavior fixes. No new blocking findings emerged. M2 conformance integration and M3 nightly delivery remain explicitly outstanding.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      provision.go:85-119 now requires reducer success authority and checks original-caller cancellation before sealing/publication. TestProvisionCancellationNeverActivates passes at head; removing those guards in a disposable copy made all six cancellation stages fail with activation and changed manifest bytes.
  - id: BR-2
    disposition: addressed
    note: |
      provision.go:239 enforces the shared 64 KiB bound before replacement. TestPublishProvisionManifestSizeBoundary verifies readable exact-limit publication and unchanged prior bytes on overflow; removing the writer guard reproduced publication of an unreadable 65,537-byte manifest.
  - id: BR-3
    disposition: addressed
    note: |
      The plan's corrected integration table at lines 59-67 names the existing runtime/host, driver, platform positions, snapshot, process and guest files. The 22:07 Revisions entry explains this mapping; the pinned before/after diff and delivered files agree.
```

1. **Strengths**
   - Provisioning activation now consumes reducer authority and retains cancellation checks through publication.
   - Git and Tart subprocesses share termination-before-reap ownership.
   - Guest results use bounded transport without a writable host share.
   - README and atlas document the new commands, resource limits and recovery behavior.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Passed VM, dispatcher and public CLI package tests.
   - Passed the complete VM race suite, nine provisioning-script tests, shell syntax checks and targeted source-inventory guards.
   - Both behavior regressions failed when their fixes were removed in a disposable copy, subsequently deleted.
   - Inspected retained native qualification evidence; no live VM provisioning or boot was performed during this review. Repository status remained clean.

6. **Architectural notes**
   - **ARCH-DRY — Pass:** shared subprocess owner and manifest codec.
   - **ARCH-PURE — Pass:** request, codec, invocation and transition decisions remain separate from IO.
   - **ARCH-PURPOSE — Pass:** revised M1 delivered; later milestones remain visible.
   - **ARCH-MOCK — Pass:** stateful doubles exercise production runtime/driver seams; native qualification evidence is retained.
   - **ARCH-CONSTRAINTS — Pass:** snapshot, export, artifact and manifest bounds have adversarial coverage.
   - **ARCH-SECURE — Pass:** isolated guest roots, literal argv, restricted sharing and strict boundary decoding.
   - **ARCH-ORDER — Pass:** controlled cancellation tests preserve cleanup ownership and prevent late activation.
   - **ARCH-FUNERAL — Pass:** normal cleanup, bounded retention and explicit uncertain-residue recovery.
   - **ARCH-EVOLVE — Pass:** additive metadata preservation and strict validation have regression coverage.

7. **Plan revision recommendations:** None required for M1.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"b1c631a8-52f9-437e-b610-74e7845ca2a0","repository":"github.com/xianxu/pair","issue_id":"000743","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/a7e0afb2d3bfa270","reviewed_head":"ca1e57116e68be45336ef22d0b7f8b80f1681234","slot":"pair:3","requested_at":"2026-10-21T05:14:12.208796Z"},"phase":"finished","pid":84505,"started_at":"2026-10-21T05:14:12.369278Z","finished_at":"2026-10-21T05:16:37.816462Z","session_id":"01a15ce1-9f7f-7122-b7c3-17e314a7ca75","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.573867s
head: "ca1e57116e68be45336ef22d0b7f8b80f1681234"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409
