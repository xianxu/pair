# Boundary Review — pair#573 (whole-issue close)

| field | value |
|-------|-------|
| issue | 573 — Agentic documents: Couch registry so session-sync verbs work for any submitted file (universal-agent-input-interface S1, Couch side) |
| repo | pair |
| issue file | workshop/issues/000573-agentic-documents-couch-registry-so-session-sync-verbs-work-for-any-submitted-file-universal-agent-input-interface-s1-couch-side.md |
| boundary | whole-issue close |
| milestone | — |
| window | d6da7d4b809729db8a8766bd71258ca8f3326d9a..52a0ff84621e6a41f057af5a3b8d1c4d03c0d560 |
| command | sdlc close --issue 573 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T12:59:40-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The registry, CLI, documentation, and queued-submit attribution largely match the approved plan. All four focused package suites pass. Two reproduced correctness defects block shipping: repeat registration can bypass an operator lock through a symlink alias, and successful operations can publish shadows that subsequent sync commands refuse to read.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: canonical-document-lock-authority
    title: |
      Repeated registration discards additional aliases and permits writes under their operator locks
    detail: |
      cmd/internal/sessionsync/registry.go:256 returns from existing-registration handling without retaining the requested spelling; documentLockPaths at line 529 can inspect only the canonical path, current request, and stored spellings. Reproduced by registering FILE, registering a symlink ALIAS to FILE, creating ALIAS.lock, then writing FILE: the write succeeds despite the operator lock. Preserve all admitted live spellings through the registry transition model, with bounded growth, and use them consistently for lock checks and legacy handoff. ARCH-ORDER / ARCH-PURPOSE.
  - id: new
    severity: Critical
    kind: defect
    family: persisted-writer-reader-envelope
    title: |
      Shadow writers exceed the registered reader limit and strand successful operations
    detail: |
      cmd/internal/sessionsync/sync.go:417 bounds only the resulting document; line 432 can publish a larger section-preserving shadow, while registry.go:496 refuses shadows over 4 MiB. Reproduced with an approximately 2 MiB baseline section, an operator deletion elsewhere, and an approximately 2 MiB section write: Write succeeds, then Read fails with registry-limit. Independently, registry.go:287-311 imports an oversized legacy shadow without validation, returns registration success, and leaves subsequent reads failing. Validate complete prospective metadata before document mutation or registry publication; sweep every metadata writer against its reader limits and add regressions for both paths. ARCH-CONSTRAINTS / ARCH-FUNERAL.
```

1. **Strengths**

   - Registry transitions preserve stable identity, reject conflicting owners and destinations, and detach returned history from input state.
   - Forget retains central authority until legacy metadata export succeeds; fault-injection coverage exercises that handoff.
   - Broker admission time survives queue promotion. The broker/ledger regression controls late completion across rename and old-path reuse.
   - README, embedded skill, atlas contract, and artifact inventory cover the new surface.

2. **Critical findings**

   - **Alias lock authority:** `registry.go:256`. Retain additional registered spellings without resetting identity or the owed horizon. Add a regression asserting canonical writes refuse under a subsequently registered alias’s lock.
   - **Writer/reader bounds:** `sync.go:432` and `registry.go:311`. Preflight the prospective shadow before changing the document, and validate imported metadata before admitting registration. Preserve existing authority and snapshots on refusal.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   ```text
   go test ./cmd/internal/sessionsync ./cmd/internal/couchmessage ./cmd/internal/couchcmd ./cmd/internal/artifactpath
   ```

   Three temporary Go test overlays failed on the pinned head, reproducing the alias-lock bypass, oversized legacy-shadow import, and oversized generated shadow. The overlays were removed; the checkout remains unchanged. Existing tests do not cover these cases. `git diff --check` also passed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** reuses the existing ledger, CAS, ownership parser, and durable publication.
   - **ARCH-PURE — pass:** registry decisions are separated from filesystem effects; model tests run without IO.
   - **ARCH-PURPOSE — flag:** canonical registration does not preserve lock authority across repeated alias registration.
   - **ARCH-MOCK — pass:** temporary filesystem backends and controlled broker endpoints exercise production seams.
   - **ARCH-CONSTRAINTS — flag:** shadow producers do not enforce the reader’s declared envelope.
   - **ARCH-SECURE — pass:** registry decoding validates persisted authority and refuses malformed records; registered file reads check type and size.
   - **ARCH-ORDER — flag:** admitted alias knowledge is absent from authoritative registry state. Rename/forget serialization and uncertain-publication handling otherwise have meaningful coverage.
   - **ARCH-FUNERAL — flag:** the shadow cap currently bounds readers without matching writer enforcement. Explicit forget, orphan cleanup, and bounded history otherwise provide lifecycle rules.

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying:

   - How repeated registrations retain bounded live alias spellings, and how those spellings govern locks and legacy handoff.
   - A prepublication invariant covering every registered metadata producer, including legacy import and section-preserving shadows, with refusal before document mutation when the prospective result exceeds its envelope.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"15ce5ed6-357a-4e41-8c31-1db418eb5595","repository":"github.com/xianxu/pair","issue_id":"000573","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/7ef127e48f5f4414","reviewed_head":"52a0ff84621e6a41f057af5a3b8d1c4d03c0d560","slot":"pair:5","requested_at":"2026-10-15T19:57:01.859517Z"},"phase":"finished","pid":98632,"started_at":"2026-10-15T19:57:02.10891Z","finished_at":"2026-10-15T19:59:40.001749Z","session_id":"01a14123-b8c4-76a1-b0ee-e416704ea4bc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.817890708s
head: "52a0ff84621e6a41f057af5a3b8d1c4d03c0d560"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
