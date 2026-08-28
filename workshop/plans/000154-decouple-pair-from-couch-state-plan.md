---
issue: 000154
status: approved
created: 2026-08-27
updated: 2026-08-27
---

# Pair–Couch State Decoupling Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore Pair as a standalone lower-level harness wrapper by removing every Pair-launcher dependency on Couch persistence while preserving Couch-owned name resolution and exact-tag hosting.

**Architecture:** Pair continues to own tags, scoped artifacts, Zellij bindings, and the reserved-address handshake, but its runtime and composition root no longer expose a Couch thread-index reader or standalone Couch registrar. Couch resolves its mutable names and paths directly over `ThreadRecord`, invokes Pair with the resolved exact tag, and alone promotes `ThreadStore` after observing Pair's established address claim. Couch environment remains opaque inherited process context to Pair.

**Tech Stack:** Go, injected launcher runtime, filesystem-backed Couch `ThreadStore`, table-driven unit tests, existing process/readiness integration tests.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ResolveThreadReference` | `cmd/internal/couchcore/threadmetadata.go` | modified |
| `ResolveThreadIndexReference` | `cmd/internal/launcher/thread_index.go` | deleted |

- **`ResolveThreadReference`** — Couch-owned exact-tag-first, then case-insensitive name/path resolution over in-memory `ThreadRecord` values.
  - **Relationships:** one resolver consumes N Couch records and returns zero, one, or N cloned matches; Couch owns both records and ambiguity errors.
  - **DRY rationale:** today it adapts Couch records into a launcher-owned Couch persistence projection and back. Resolving directly over `ThreadRecord` removes that circular conceptual ownership while retaining one matcher (ARCH-DRY, ARCH-PURPOSE).
  - **Future extensions:** additional Couch-only human attributes widen this resolver without changing Pair.
- **`ResolveThreadIndexReference`** — the deleted Pair-owned Couch matcher. Its
  behavior moves to `ResolveThreadReference`, where the records and ambiguity
  errors are already Couch-owned.
  - **Relationships:** formerly one resolver consumed N launcher projections;
    the retained resolver consumes N Couch records directly.
  - **DRY rationale:** deletion restores one matcher over the authoritative
    Couch entity instead of adapting to and from a shadow Pair type.
  - **Future extensions:** none in Pair; Couch-only attributes widen the Couch
    resolver.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Runtime.ReadThreadIndex` | `cmd/internal/launcher/runtime.go`, `cmd/internal/launcher/osruntime.go` | deleted | Couch manifest and addressed records |
| `LaunchOptions.RegisterStandaloneThread` | `cmd/internal/launcher/runtime.go`, `cmd/internal/launcher/createflow.go` | deleted | Couch `ThreadStore` mutation callback |
| `LaunchNativeWithStandaloneRegistrar` | `cmd/internal/launcher/runcli.go`, `cmd/pair-go/main.go` | deleted | direct Pair-to-Couch composition |
| `ScopedThreadArtifactCollisionChecker` | `cmd/internal/couchcore/artifactcollision.go` | unchanged | real Pair-scoped address marker observed by Couch |
| `ThreadStore.AdvanceStart` | `cmd/internal/couchcore/couch.go` | unchanged | Couch-owned persistent lifecycle |

- **`Runtime.ReadThreadIndex`** — deleted from the launcher effect interface and
  `OSRuntime`; after deletion no Pair runtime seam can open Couch persistence.
  The existing fake remains stateful for Pair-owned sessions, ledgers, files,
  and address claims.
  - **Injected into:** formerly `RunLaunch`; no replacement is introduced.
  - **Future extensions:** Pair-only effects; any proposed Couch-schema effect requires operator consultation.
- **`LaunchOptions.RegisterStandaloneThread`** — deleted callback which let an
  ordinary Pair create mutate Couch before the workspace child launched.
  - **Injected into:** formerly `RunLaunch`; no replacement is introduced.
  - **Future extensions:** Couch discovers threads it creates; importing
    historical Pair tags, if ever wanted, is a Couch-owned operation.
- **`LaunchNativeWithStandaloneRegistrar`** — deleted composition helper.
  `LaunchNative` still consumes one-shot Couch launch-profile input when hosted,
  but never stores `COUCH_STORE_DIR` in launcher state. Normal environment
  inheritance transparently carries opaque Couch variables to Pair children.
  - **Injected into:** `cmd/pair-go` process entry.
  - **Future extensions:** additional one-shot lower-layer invocation inputs, not Couch persistence schemas.
- **`ScopedThreadArtifactCollisionChecker`** — remains the production Couch
  observer of Pair's exact scope/tag acceptance proof. It does not prove full
  workbench readiness and does not access `ThreadStore`.
  - **Injected into:** `Couch`; direct Pair reaches the same marker format
    independently through `Runtime.EnsureThreadAddress`, while Couch polls this
    checker through its artifact-collision seam.
  - **Future extensions:** stronger Pair-owned readiness may add a separate Pair artifact without exposing Couch state to Pair.
- **`ThreadStore.AdvanceStart`** — remains the only writer of Couch lifecycle
  state after a Couch start. Existing store tests model creating → live
  promotion; the composed boundary test below supplies the real Pair marker.
  - **Injected into:** `Couch.Spawn`.
  - **Future extensions:** park/resume lifecycle stays Couch-owned.

## Chunk 1: Remove the persistence dependency and preserve hosting

### Task 1: Make Couch own thread reference resolution

**Files:**

- Modify: `cmd/internal/couchcore/threadmetadata.go`
- Modify: `cmd/internal/couchcore/threadmetadata_test.go`

- [ ] **Step 1: Strengthen Couch-local resolver tests before moving ownership**

Add a table test which passes cloned `ThreadRecord` values and proves: exact tag wins within scope; user name and path remain case-insensitive substring matches; duplicate names return sorted `[]ThreadAddress`; empty input returns `ErrThreadReferenceNotFound`; mutating a returned record does not mutate the input.

- [ ] **Step 2: Run the focused tests and establish the green refactor baseline**

Run:

```bash
go test ./cmd/internal/couchcore -run 'TestResolveThreadReference' -count=1
```

Expected: PASS against the existing adapter, proving the retained behavior before the ownership refactor.

- [ ] **Step 3: Replace the launcher adapter with direct Couch-owned resolution**

In `ResolveThreadReference`, trim the reference; filter records by optional repo scope; prefer exact `string(record.Address.Tag) == ref`; otherwise match lowercase `Name` or `WorkingPath`; sort matches by `{RepoScope, Tag}`; return cloned records and Couch's existing `ErrThreadReferenceNotFound` / `AmbiguousThreadReferenceError`. Remove the `launcher` import from `threadmetadata.go`.

- [ ] **Step 4: Rerun focused tests**

Run the Step 2 command.

Expected: PASS with no launcher thread-index types involved.

- [ ] **Step 5: Commit the Couch-local ownership move**

```bash
git add cmd/internal/couchcore/threadmetadata.go cmd/internal/couchcore/threadmetadata_test.go
git commit -m '#154: move thread reference resolution into Couch'
```

### Task 2: Pin every direct command family to Couch-independent behavior

**Files:**

- Modify: `cmd/internal/launcher/createflow_test.go`
- Modify: `cmd/internal/launcher/runcli_test.go`
- Modify: `cmd/internal/launcher/rename_test.go`
- Modify: `cmd/pair-go/main_test.go`

- [ ] **Step 1: Add the permanent process-boundary fixture matrix**

In `runcli_test.go`, add `TestLaunchNativeCommandFamiliesIgnoreCouchStore`.
Its outer table creates four `COUCH_STORE_DIR` fixtures: a valid manifest with
`legacy_migration_version`, malformed JSON, a chmod-000 unreadable directory,
and a missing path. Its inner table runs production `LaunchNative` for:

- exact-tag create/resume with `compiler-fix` and
  `couch-3dcfba1308775e82`;
- attach of a live exact tag;
- bare launch with the native picker choosing a historical tag;
- `list`, `rename`, bare and resolved `continue`, `restart`, and `quit`.

Reuse the existing temp HOME/`PAIR_DATA_DIR` setup and stub executable patterns
from `runcli_test.go` and `osruntime_test.go`; extend them only enough to make
zellij, fzf, ps, and the workspace child deterministic. For each family,
capture exit code, stdout/stderr, Pair-owned files, and launched argv/env, and
compare the result across all four Couch fixtures. Snapshot the Couch fixture
tree before and after and require byte-for-byte equality; the missing fixture
must remain absent. Restore permissions in cleanup before `TempDir` removal.

This test remains after `ReadThreadIndex` is deleted. It is the executable
process-behavior proof that no direct command can open or mutate the Couch
namespace; it does not depend on the obsolete fake seam.

- [ ] **Step 2: Add the permanent public-entry write-boundary regression**

In `cmd/pair-go/main_test.go`, add
`TestPublicPairLaunchDoesNotRegisterStandaloneCouchThread`. Build and execute
the actual `cmd/pair-go` public entry (using the existing `buildCommand` /
`runCommand` subprocess helpers) as `pair-go launch resume compiler-fix` with:

- a valid asset root and deterministic zellij/ps stubs;
- isolated HOME/`PAIR_DATA_DIR`;
- a real seeded `COUCH_STORE_DIR` containing the current manifest and records.

Snapshot the entire Couch namespace, run through `main → osLegacyRuntime →`
the production launcher composition, require a successful Pair create, then
require the Couch namespace to be byte-identical. This test fails before Task 4
because `LaunchNativeWithStandaloneRegistrar` wires
`couchcore.RegisterStandalonePair`; it stays permanently after that helper and
registrar are deleted, so the public composition—not only `LaunchNative`—is
guarded.

- [ ] **Step 3: Add focused red tests for the two current couplings**

In `createflow_test.go`, replace the old “corrupt authoritative thread index
refuses” expectation with a table over the two exact tags above. Configure the
current fake thread-index seam to return `decode thread manifest: json: unknown
field "legacy_migration_version"`; run exact resume/create; expect code 0 and
`PAIR_TAG` equal to the input tag. Change the standalone-registration-failure
test to expect that a Couch failure cannot block the Pair workspace child.

- [ ] **Step 4: Run the regressions and verify the observed failures**

Run:

```bash
go test ./cmd/internal/launcher -run 'TestLaunchNativeCommandFamiliesIgnoreCouchStore|TestRunLaunchIgnoresCouchStateForExactTag|TestStandaloneCouchRegistrationCannotBlockPair' -count=1
go test ./cmd/pair-go -run 'TestPublicPairLaunchDoesNotRegisterStandaloneCouchThread' -count=1
```

Expected: FAIL. Exact resume/create rejects the valid-forward or malformed
manifest, and the public-entry test observes standalone Couch registration.
Record matrix rows already green; they stay to prevent the dependency moving to
early returns.

- [ ] **Step 5: Commit the red boundary tests**

```bash
git add cmd/internal/launcher/createflow_test.go cmd/internal/launcher/runcli_test.go cmd/internal/launcher/rename_test.go cmd/pair-go/main_test.go
git commit -m '#154: pin direct Pair independence from Couch state'
```

### Task 3: Delete Pair's Couch read path

**Files:**

- Modify: `cmd/internal/launcher/createflow.go`
- Modify: `cmd/internal/launcher/createflow_test.go`
- Modify: `cmd/internal/launcher/runtime.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/session_index.go`
- Modify: `cmd/internal/launcher/rename_test.go`
- Delete: `cmd/internal/launcher/thread_index.go`
- Delete: `cmd/internal/launcher/thread_index_test.go`
- Delete: `cmd/internal/launcher/thread_index_conformance_test.go`
- Modify: `cmd/internal/artifactpath/manifest.go`
- Modify: `cmd/internal/threadrecord/record.go`
- Modify: `cmd/internal/strictjson/decode.go`

- [ ] **Step 1: Remove Couch inventory from the launcher decision path**

Delete `ReadThreadIndex` from `SessionNameStoreOps`, `OSRuntime`, and the fake
runtime. Remove the manifest read, history merge, live-row decoration,
human-name resolution, ambiguity branch, and `resolveResumeTag` from `runOnce`.
Preserve only Pair's existing public-Zellij-name inversion and exact
`ForcedTag`; readable and generated tags use the same exact path.

Delete the launcher thread-index implementation/tests. Update
`SessionNameEntry` to describe only Zellij socket bindings.

- [ ] **Step 2: Remove obsolete read-path scaffolding and inventory entries**

Delete fake `threadIndex` state, standalone-name picker/ambiguity/decorator
cases, and the “human Couch name is not renameable” test. Keep the permanent
process matrix and exact-tag tests. Remove deleted source paths from the artifact
manifest; update `threadrecord`/`strictjson` comments so they describe Couch's
internal record acceptance rather than a standalone Pair reader.

- [ ] **Step 3: Run the read-removal slice**

```bash
go test ./cmd/internal/launcher ./cmd/internal/couchcore -count=1
rg -n 'ReadThreadIndex|ThreadIndex|ResolveThreadIndexReference|COUCH_STORE_DIR' cmd/internal/launcher
```

Expected: tests PASS and grep has no matches. Couch-owned packages may still
set and consume `COUCH_STORE_DIR`.

- [ ] **Step 4: Commit the read-path deletion**

```bash
git add cmd/internal/launcher cmd/internal/artifactpath/manifest.go cmd/internal/threadrecord/record.go cmd/internal/strictjson/decode.go
git commit -m '#154: remove Couch inventory from Pair launch'
```

### Task 4: Delete direct Pair's Couch write path

**Files:**

- Modify: `cmd/internal/launcher/createflow.go`
- Modify: `cmd/internal/launcher/createflow_test.go`
- Modify: `cmd/internal/launcher/runtime.go`
- Modify: `cmd/internal/launcher/runcli.go`
- Delete: `cmd/internal/couchcore/standalone.go`
- Delete: `cmd/internal/couchcore/standalone_test.go`
- Modify: `cmd/pair-go/main.go`
- Modify: `cmd/internal/artifactpath/manifest.go`

- [ ] **Step 1: Remove direct-Pair mutation of Couch**

Delete `RegisterStandaloneThread`, `StandaloneThreadRegistration`, and
`StandaloneThreadRegistrar` from `LaunchOptions`/runtime. Remove the pre-child
registrar call from `runOnce`, delete `couchcore/standalone.go` and its tests,
and simplify `cmd/pair-go/main.go` to call `launcher.LaunchNative` directly.

Remove `CouchStoreDir` from `LaunchOptions`, `OSRuntime`, `newLaunchOptions`,
and `LaunchNative`; do not unset Couch variables because ordinary OS inheritance
is the opaque pass-through. Retain `CouchLaunchProfileEnv`,
`COUCH_THREAD_SCOPE`, and `COUCH_THREAD_TAG` only as one-shot invocation/address
inputs already defined by Pair.

- [ ] **Step 2: Run the write-removal slice**

```bash
go test ./cmd/internal/launcher ./cmd/internal/couchcore ./cmd/pair-go -count=1
rg -n 'StandaloneThreadRegistration|StandaloneThreadRegistrar|RegisterStandalonePair|RegisterStandaloneThread|CouchStoreDir|LaunchNativeWithStandaloneRegistrar' cmd/internal/launcher cmd/internal/couchcore cmd/pair-go
```

Expected: tests PASS and grep has no matches.

- [ ] **Step 3: Commit the write-path deletion**

```bash
git add cmd/internal/launcher cmd/internal/couchcore/standalone.go cmd/internal/couchcore/standalone_test.go cmd/pair-go/main.go cmd/internal/artifactpath/manifest.go
git commit -m '#154: decouple Pair launch from Couch persistence'
```

### Task 5: Verify the composed handshake and document every current consumer

**Files:**

- Modify: `cmd/internal/couchcore/couch_test.go`
- Modify: `cmd/internal/couchcore/artifactcollision_test.go`
- Modify: `README.md`
- Modify: `atlas/index.md`
- Modify: `atlas/couch.md`
- Modify: `atlas/session-identity.md`
- Modify: `workshop/projects/couch.md`
- Modify: `workshop/issues/000154-decouple-pair-from-couch-state.md`

- [ ] **Step 1: Add a composed Couch → production Pair address test**

Add a test-only `composedPairRunner`/handle in `couch_test.go`. It implements the
blocked-runner protocol, but on `Acknowledge` calls production
`launcher.LaunchNative` with the exact argv/env supplied by `Couch.Spawn`, a
real temp `PAIR_DATA_DIR`, and deterministic stubs for zellij/ps. Construct
Couch with production `ScopedThreadArtifactCollisionChecker` pointed at that
same Pair data dir, not `FakeThreadArtifactCollisionChecker.AutoEstablish`.

The runner records that the real scoped marker is `reserved` before ack and
`established` after `LaunchNative` accepts the exact
`COUCH_THREAD_SCOPE`/`COUCH_THREAD_TAG`, before its stub Zellij handoff returns.
Seed Couch's real temp `ThreadStore` manifest with
`legacy_migration_version`. Assert:

- argv is `pair resume <exact-generated-tag> --layout2` and all four opaque
  Couch env values are preserved;
- the real Pair path performs exactly `reserved → established` for that
  scope/tag, and no sibling/mismatched marker is accepted;
- Pair does not create or modify any Couch record or manifest;
- only subsequent Couch `ThreadStore.AdvanceStart` changes the matching record
  from creating to live after the expected helper identity is observed;
- malformed, mismatched, unreadable, and still-reserved markers remain
  fail-closed in the production-marker cases added next.

- [ ] **Step 2: Add production-marker failure and non-promotion cases**

Extend `artifactcollision_test.go` with a table over the real marker path and
`ScopedThreadArtifactCollisionChecker.Registration`:

- missing marker and a valid `reserved` marker return `RegistrationAbsent`;
- malformed JSON, wrong embedded scope, wrong embedded tag, invalid state, and
  an unreadable marker (create a directory at the marker path for a portable
  read failure) return `RegistrationUnknown` plus an error;
- a valid exact `established` marker alone returns
  `RegistrationEstablished`.

For every non-established row, create a real Couch `ThreadStore` record in the
creating state, call `Couch.awaitThreadRegistration` with the production
checker and a bounded context, and assert it errors/timeouts while the exact
record remains creating: no call path reaches `StartRegistered` or live
promotion. The established control succeeds and is then promoted through
`ThreadStore.AdvanceStart`, proving the table is connected to the real Couch
decision rather than only testing a parser (ARCH-PURPOSE, ARCH-MOCK).

- [ ] **Step 3: Run the Couch boundary regression**

```bash
go test ./cmd/internal/couchcore -run 'Test.*(Registration|AddressClaim|Spawn).*' -count=1
```

Expected: PASS, including the composed production Pair marker transition and
all production-marker fail-closed cases.

- [ ] **Step 4: Rewrite architecture documentation with a shadow sweep**

Run `rg -n 'ThreadIndex|thread index|standalone Pair|standalone registration|tag-or-thread-name|pair resume' README.md atlas --glob '*.md'` and inspect every
current restatement. At minimum:

- `README.md`: replace `pair resume <tag-or-thread-name>` and the standalone
  ThreadIndex section with exact Pair-tag semantics; mutable Couch name/path
  resolution appears only under Couch commands.
- `atlas/index.md`: replace “shared standalone registration/artifact authority”
  with the two independent authorities.
- `atlas/couch.md`: replace “Metadata and standalone Pair lookup” with
  Couch-owned metadata/resolution; opaque env passes through Pair uninterpreted.
- `atlas/session-identity.md`: replace “Durable thread index” and picker/name
  decoration with Pair address claims/scoped artifacts versus Couch
  `ThreadStore`.

Update every live contradiction or record why a remaining match is historical.
Append a 2026-08-27 scope correction to `workshop/projects/couch.md`; do not
rewrite historical #149 log entries (ARCH-PURPOSE shadow sweep).

- [ ] **Step 5: Run the complete verification suite**

```bash
go test ./... -count=1
go test -race ./... -count=1
make test-lua
bash tests/term-pane-shortcuts-test.sh
bash tests/review-toggle-test.sh
zellij --config-dir zellij setup --check
zellij setup --dump-layout zellij/layouts/main-2.kdl >/dev/null
zellij setup --dump-layout zellij/layouts/main-3.kdl >/dev/null
git diff --check
```

Expected: all commands exit 0.

- [ ] **Step 6: Update issue evidence and commit**

Tick the issue plan items, append exact verification evidence and architectural
correction to `## Log`, then commit:

```bash
git add README.md atlas/index.md atlas/couch.md atlas/session-identity.md workshop/projects/couch.md workshop/issues/000154-decouple-pair-from-couch-state.md
git commit -m '#154: document the Pair-Couch ownership boundary'
```

- [ ] **Step 7: Enter the SDLC close boundary**

Run `sdlc close --issue 154 --verified '<exact commands and observed Pair/Couch behavior>'`. Let the binary dispatch the mandatory fresh-eyes review; fix Critical/Important findings before rerunning. Do not separately dispatch a redundant code review at this boundary.

## Revisions

### 2026-08-27 — close the command, handshake, documentation, and entity-table gaps

**Reason:** the first fresh-context review found that the draft deleted its own
only Couch-IO fake instead of retaining a permanent process-boundary sweep,
used a Couch fake which bypassed Pair's real address transition, omitted live
README/atlas restatements, and mixed concepts with implemented symbols in the
load-bearing entity tables.

**Delta:** the plan now adds a four-store × every-command `LaunchNative` matrix
before deletion; splits reads and writes into separate red-green commits;
composes Couch with production `LaunchNative` and the real scoped marker;
enumerates the README/atlas shadow sweep; names one symbol per entity-table row;
and races the full Go package graph (ARCH-PURPOSE, ARCH-PURE).

### 2026-08-27 — exercise the public composition and real marker failures

**Reason:** the second fresh-context review found that `LaunchNative` bypasses
the registrar wired only by `cmd/pair-go`, and that malformed/unreadable marker
coverage used fake registration errors rather than production marker IO.

**Delta:** Task 2 now builds and runs the public Pair entry against a real Couch
namespace and requires no registration write. Task 5 now enumerates real marker
records through `ScopedThreadArtifactCollisionChecker` and a creating
`ThreadStore`, proving each adverse evidence class fails closed without
promotion (ARCH-PURPOSE, ARCH-MOCK).
