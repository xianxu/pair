# Boundary Review — pair#777 (whole-issue close)

| field | value |
|-------|-------|
| issue | 777 — session-sync: serialize operator acquisition with the writer commit boundary |
| repo | pair |
| issue file | workshop/issues/000777-operator-acquire-write-boundary.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1ed39dfdbbe0182b72c4b3a62b5504b2e725ce5d..c5f0eecea31e78eee9a9071e3d7e8898638b51a0 |
| command | sdlc close --issue 777 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T15:17:47-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The shared registry transaction implements the intended acquisition boundary, with convincing coverage of writer ordering, ownership, uncertain effects, and move recovery. No Critical correctness defect was found. One Important documentation gap remains: README does not list the new public commands.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: public-cli-readme-coverage
    title: |
      README update appears missing for operator acquisition and release
    detail: |
      cmd/internal/couchcmd/session_sync.go:63-64 adds operator-acquire and operator-release with required --id, --turn and --json arguments, but README.md is unchanged in the pinned range and its session-sync reference at lines 508-531 omits both commands. Add their invocation forms and a brief pointer to the acknowledgment, retry and release contract in atlas/couch-agentic-documents.md to satisfy the README docs gate.
```

1. **Strengths**

   - Acquisition and release reuse the existing writer registry flock rather than introducing another exclusion mechanism.
   - `operator_lock_test.go:68` deterministically tests contention after the writer guard and through final round publication, then verifies protected Write/Apply refusals preserve document and metadata.
   - The stateful filesystem fake distinguishes visible effects from durable effects, covering lost acknowledgment, failed directory synchronization, and partial alias removal.
   - Ownership parsing rejects ambiguous known fields while preserving exact proof bytes and unknown lines on retries.

2. **Critical findings**

   None.

3. **Important findings**

   README command coverage, detailed above. Add both commands near `README.md:528`, including required arguments and a link to the atlas contract.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - `go test ./cmd/internal/sessionsync ./cmd/internal/couchcmd ./cmd/couch`
   - `go test -race ./cmd/internal/sessionsync -run 'TestOperatorLock' -count=1`
   - `make test-affected-inventory`
   - Pinned-range `git diff --check`

   Pure model tests use no IO. Integration tests exercise the production transaction through portable storage and injected filesystem effects. No prior findings required disposition; no regression mutation was performed.

6. **Architectural notes**

   | Principle | Assessment |
   | --- | --- |
   | ARCH-DRY | Pass: reuses registry exclusion, canonical routing, alias enumeration, and durable publication. |
   | ARCH-PURE | Pass: proof parsing and decisions are separate from filesystem execution. |
   | ARCH-PURPOSE | Pass: covers Write, Apply, acknowledgment ordering, and move recovery; consumer freshness remains explicitly assigned to Parley. |
   | ARCH-MOCK | Pass: production and stateful tests share the filesystem seam; executable tests exercise the real backend. |
   | ARCH-CONSTRAINTS | Pass: bounded reads and existing registry limits remain; counted alias tests exercise linear effects. |
   | ARCH-SECURE | Pass: validates identity/token, refuses unsafe sidecars, and preserves foreign protection. |
   | ARCH-ORDER | Pass: controlled ordering and failure sequences cover uncertainty, process death, competing turns, and partial progress. |
   | ARCH-FUNERAL | Pass: exact-turn release owns sidecar removal; deterministic stages have a documented reclamation path. |
   | ARCH-EVOLVE | Pass: ownership metadata is additive and existing proof text survives retries unchanged. |

   Core-concept entries match their stated files and classifications. Atlas coverage is present.

7. **Plan revision recommendations**

   Add a `## Revisions` entry extending the documentation task to include README’s public command reference for both new verbs.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6c9f4a66-73c4-4e2c-82a6-bd8e01a365bc","repository":"github.com/xianxu/pair","issue_id":"000777","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/cc617d1395f64682","reviewed_head":"c5f0eecea31e78eee9a9071e3d7e8898638b51a0","slot":"pair:2","requested_at":"2026-10-21T22:15:59.944625Z"},"phase":"finished","pid":89253,"started_at":"2026-10-21T22:16:00.037421Z","finished_at":"2026-10-21T22:17:47.226024Z","session_id":"01a16089-1a27-7c73-9670-a580ce3302e1","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.714367375s
head: "c5f0eecea31e78eee9a9071e3d7e8898638b51a0"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T15:23:25-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 777 — session-sync: serialize operator acquisition with the writer commit boundary |
| repo | pair |
| issue file | workshop/issues/000777-operator-acquire-write-boundary.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1ed39dfdbbe0182b72c4b3a62b5504b2e725ce5d..5620eb1d7a9592f06992466f17083a784d97ecb7 |
| command | sdlc close --issue 777 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T15:23:25-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation fulfills the producer-side exclusion contract: operator acquisition shares the registered writer transaction, preserves exact turn ownership on retries, and acknowledges only after durability confirmation. BR-1 is addressed. No new findings.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:532-537 adds both required invocation forms and the acknowledgment/retry/release pointer. atlas/couch-agentic-documents.md:87-155 describes the contract implemented by session_sync.go and operator_lock.go. This is a prose-only correction; behavioral conformance tests also pass.
```

1. **Strengths**

   - Acquisition and release reuse `withRegistry`, covering the same exclusion boundary as registered Write and Apply (`operator_lock.go:71`).
   - Deterministic ordering tests pause writers after their guard and at final round publication, proving acquisition cannot acknowledge early.
   - Recovery preserves exact proof bytes and handles uncertain publication, partial alias removal, process death, and canonical witness reconstruction.
   - README, atlas, atlas index, and embedded skill consistently document the new commands and legacy-client limits.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed:

   - Full `cmd/internal/sessionsync` suite.
   - Selected operator CLI and real executable tests in `couchcmd` and `cmd/couch`.
   - Race-enabled `TestOperatorLock` tests.
   - `make test-affected-inventory`.
   - Pinned-range `git diff --check`.

   Pure ownership tests use no IO. Integration tests use disposable folders and a stateful filesystem fake. Native editor freshness/input behavior remains the explicitly documented consumer deliverable in parley.nvim#408.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Existing registry exclusion, path authority, and durable publication are reused.
   - **ARCH-PURE — pass:** Proof parsing and ownership decisions are pure; filesystem effects remain in the transaction shell.
   - **ARCH-PURPOSE — pass:** Write, Apply, and move recovery participate in the documented acknowledgment contract.
   - **ARCH-MOCK — pass:** Production and uncertainty tests share the filesystem seam; real-backend and executable tests provide conformance evidence.
   - **ARCH-CONSTRAINTS — pass:** Existing registry/alias bounds, bounded sidecar reads, and counted alias tests support the operating envelope.
   - **ARCH-SECURE — pass:** Identity, token, file type, and ownership validation precede destructive effects; foreign protection is preserved.
   - **ARCH-ORDER — pass:** Tagged decisions and serialized transactions preserve uncertain outcomes; controlled ordering and crash tests exercise recovery.
   - **ARCH-FUNERAL — pass:** Exact-turn release and existing move ownership govern sidecars; deterministic stages have bounded residue and reclamation.
   - **ARCH-EVOLVE — pass:** New sidecar fields are additive; retries and canonical recovery preserve raw unknown lines.

7. **Plan revision recommendations:** None. The core-concept paths, classifications, and implementation status match the inspected code.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a79e1ab8-4d83-42c8-b6ff-984d94baf599","repository":"github.com/xianxu/pair","issue_id":"000777","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/1eb60932f33eb070","reviewed_head":"5620eb1d7a9592f06992466f17083a784d97ecb7","slot":"pair:2","requested_at":"2026-10-21T22:22:08.42904Z"},"phase":"finished","pid":80408,"started_at":"2026-10-21T22:22:08.540777Z","finished_at":"2026-10-21T22:23:24.403074Z","session_id":"01a1608e-b9e5-7a43-bd12-10e577e9ed2c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.213661125s
head: "5620eb1d7a9592f06992466f17083a784d97ecb7"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
