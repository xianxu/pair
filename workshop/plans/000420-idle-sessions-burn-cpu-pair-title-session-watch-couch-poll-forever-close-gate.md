---
gate: boundary-review
issue: 420
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-10T18:42:07-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: OSRuntime.StatFile has no direct test; the os.ErrNotExist contract callers depend on is pinned only by the fake
          detail: 'ActivityForSession and ObserveNamedArtifacts branch on errors.Is(err, os.ErrNotExist). Add a temp-dir test: regular file gives the same FileEntry as ListFiles, a missing file is ErrNotExist, and symlink or non-regular entries are rejected.'
          family: real-fake-seam-conformance
          round: 1
        - id: BR-2
          severity: Minor
          title: IdleExpiry counts only Pair sends, but a chosen-id launch binds via the filename handshake with no send
          detail: An operator typing directly into a chosen Claude pane after more than 24h idle never binds. Counting ScrollbackRaw movement as activity would keep the funeral and those bindings at about one stat of cost.
          family: idle-expiry-activity-signal
          round: 1
        - id: BR-3
          severity: Minor
          title: A final-component symlink in StatFile now errors in ActivityForSession, where the walk returned (false, nil)
          family: walk-to-stat-behavior-parity
          round: 1
        - id: BR-4
          severity: Minor
          title: The sessionwatch loop adds quietAt, quietLogMod, outputSeen and turnPending flags without an explicit phase enum (ARCH-ORDER)
          detail: Already deferred in the issue Log; the epoch, quiet, pending and bound enum should lead the next change to the watcher.
          family: loop-state-flag-constellation
          round: 1
        - id: BR-5
          severity: Minor
          title: OperationCountForRoot stays vacuous for ListFiles; only a lesson was recorded, not a fix to the helper
          family: vacuous-count-helper
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-10T18:47:02-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'TestOSRuntimeStatFileMatchesListFiles (runtime_os_test.go) exercises real OSRuntime on a temp dir: entry parity, ErrNotExist, non-regular, symlink, unknown root.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: wait() counts ScrollbackRaw mtime movement after launch; TestRunAgentOutputDefersIdleExpiry requires about 44h, which fails at 24h without the fix.
          round: 2
        - id: BR-3
          disposition: withdrawn
          note: The old walk also returned listErr when the transcript was a symlink (not false,nil); parity holds and the new test pins both refusals.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: Still deferred, as logged in the issue; the loop now carries 6 cross-pass vars, so the phase enum should lead the next watcher change.
          round: 2
        - id: BR-5
          disposition: addressed
          note: OperationCountForRoot adds the bare-root key; TestOperationCountForRootIncludesRootLevelOperations fails without it.
          round: 2
      findings:
        - id: BR-6
          severity: Minor
          title: The v3 epoch-acquisition branch walks the store each SlowPoll and bypasses wait(), so idle expiry never applies there
          detail: 'When launchBoundaries stays incomplete (any unreadable or invalid-path artifact in the store), run.go''s observationEpoch==nil branch Observes the whole store every SlowPoll for the agent''s lifetime with no IdleExpiry. Rule: every unbound pass that sleeps goes through wait(); replace the rt.Sleep plus deadline check there with wait().'
          family: funeral-covers-every-wait-path
          round: 2
      recipe: milestone-review
      reviewed: 53f20aded2b8d5cd83cc0e440389cfd3bee59b1d
      blocked: false
---

# Gate ledger — pair#420 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-10T18:42:07-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `real-fake-seam-conformance` OSRuntime.StatFile has no direct test; the os.ErrNotExist contract callers depend on is pinned only by the fake
  ActivityForSession and ObserveNamedArtifacts branch on errors.Is(err, os.ErrNotExist). Add a temp-dir test: regular file gives the same FileEntry as ListFiles, a missing file is ErrNotExist, and symlink or non-regular entries are rejected.
- **BR-2** [Minor] `idle-expiry-activity-signal` IdleExpiry counts only Pair sends, but a chosen-id launch binds via the filename handshake with no send
  An operator typing directly into a chosen Claude pane after more than 24h idle never binds. Counting ScrollbackRaw movement as activity would keep the funeral and those bindings at about one stat of cost.
- **BR-3** [Minor] `walk-to-stat-behavior-parity` A final-component symlink in StatFile now errors in ActivityForSession, where the walk returned (false, nil)
- **BR-4** [Minor] `loop-state-flag-constellation` The sessionwatch loop adds quietAt, quietLogMod, outputSeen and turnPending flags without an explicit phase enum (ARCH-ORDER)
  Already deferred in the issue Log; the epoch, quiet, pending and bound enum should lead the next change to the watcher.
- **BR-5** [Minor] `vacuous-count-helper` OperationCountForRoot stays vacuous for ListFiles; only a lesson was recorded, not a fix to the helper

## Round 2 — 2026-10-10T18:47:02-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — TestOSRuntimeStatFileMatchesListFiles (runtime_os_test.go) exercises real OSRuntime on a temp dir: entry parity, ErrNotExist, non-regular, symlink, unknown root.
- BR-2 — addressed — wait() counts ScrollbackRaw mtime movement after launch; TestRunAgentOutputDefersIdleExpiry requires about 44h, which fails at 24h without the fix.
- BR-3 — withdrawn — The old walk also returned listErr when the transcript was a symlink (not false,nil); parity holds and the new test pins both refusals.
- BR-4 — not-addressed — Still deferred, as logged in the issue; the loop now carries 6 cross-pass vars, so the phase enum should lead the next watcher change.
- BR-5 — addressed — OperationCountForRoot adds the bare-root key; TestOperationCountForRootIncludesRootLevelOperations fails without it.

### Raised

- **BR-6** [Minor] `funeral-covers-every-wait-path` The v3 epoch-acquisition branch walks the store each SlowPoll and bypasses wait(), so idle expiry never applies there
  When launchBoundaries stays incomplete (any unreadable or invalid-path artifact in the store), run.go's observationEpoch==nil branch Observes the whole store every SlowPoll for the agent's lifetime with no IdleExpiry. Rule: every unbound pass that sleeps goes through wait(); replace the rt.Sleep plus deadline check there with wait().

## Open findings

- **BR-4** [Minor] `loop-state-flag-constellation` The sessionwatch loop adds quietAt, quietLogMod, outputSeen and turnPending flags without an explicit phase enum (ARCH-ORDER)
- **BR-6** [Minor] `funeral-covers-every-wait-path` The v3 epoch-acquisition branch walks the store each SlowPoll and bypasses wait(), so idle expiry never applies there
