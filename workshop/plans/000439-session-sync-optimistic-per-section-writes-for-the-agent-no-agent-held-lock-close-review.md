# Boundary Review — pair#439 (whole-issue close)

| field | value |
|-------|-------|
| issue | 439 — session-sync: optimistic per-section writes for the agent, no agent-held lock |
| repo | pair |
| issue file | workshop/issues/000439-session-sync-optimistic-per-section-writes-for-the-agent-no-agent-held-lock.md |
| boundary | whole-issue close |
| milestone | — |
| window | 2d1e81f7750cf27a43cf5e225402cc21be5ec7fe..e91a498239dbeb89af6b0659677513314130a0a8 |
| command | sdlc close --issue 439 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T09:31:09-07:00 |
| verdict | SHIP |

## Review

Let me read the core files directly.

Core code read; now checking docs diff and running the tests.

```verdict
verdict: SHIP
confidence: medium
```

The change does what pair#439 asks for. Couch no longer writes, deletes, or requires a `holder: agent` lock, and a legacy lock of that kind on disk is treated as no lock (`file.go:55-65`). `read` returns a hash of one section as stored, or of the whole file. `write` requires `--if-hash` and refuses with exit 6 `stale` when the hash no longer matches, printing the current hash (`sync.go:297-306`). Every verb runs inside one `flock(LOCK_EX)` from load to its last write (`sync.go:124-145`), and a single section scanner, `section()`, serves both reading and replacing. All four Done-when test rows exist and pass: I ran `go test ./cmd/internal/sessionsync/` at the pinned head and it was green, including the two-process race. I could not run the `couchcmd` package tests: setup fails in this checkout because `runtimebundle` embeds generated assets (`assets/runtime/files`) that are missing here and that this diff doesn't touch. I reviewed those CLI tests by reading them instead, so confidence is medium. Nothing blocks shipping; the findings below are Minor.

**Inspection performed:** I ran the stat and name-status recipes, read the full code diff, and read the current `sync.go`, `file.go`, `session_sync.go` and `owed.go`, plus the README, atlas and SKILL diffs.

1. **Strengths**
   - `locked[T]` (`sync.go:124`) is one generic wrapper that every verb goes through, so the hash check and the write can't come apart. `TestAWriteChecksItsHashUnderTheFlock` checks this by holding the flock, changing the file, then releasing it, and expecting `stale`. It does not rely on timing luck.
   - `TestTwoProcessWritersExactlyOneWins` (`race_test.go`) uses real OS processes sharing a start barrier, so it covers the cross-process case that an in-process goroutine test would miss.
   - Couch deliberately never deletes `.lock`. Parley can rewrite that file as `operator` at any moment, so deleting it would break the operator's lock; the Log records this reasoning, and `TestATurnWithoutReleaseLeavesNoLock` checks that the file stays byte-identical.
   - `SectionBody` and `ReplaceSection` share `section()` (ARCH-DRY), so what `read` hashes and what `write` replaces are always the same lines.
   - Exit code 4 is retired rather than reused, and the reason is written down (`sync.go:31`).

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **`release` is a silent no-op, including with `--fold`** (`session_sync.go` release branch). It exits 0, and the only signal is a line on stderr. An older skill running `release --fold d1`, or relying on the old automatic folding of submits, will believe events were folded when they weren't. Older skills also break on `write` now that `--if-hash` is required, so the no-op only partly delivers its stated goal ("so older skills don't break"). Consider exiting non-zero when `--fold` is passed to `release`.
   - **No way to fold or accept a gap without writing.** The old `release` could fold a submit, or accept a `delivery-gap`, without changing the file. Now the only way is a non-empty `write`, so an agent with nothing to change has to rewrite a section with the same content. This should at least be documented in the SKILL.
   - **Restamp can make one of the agent's own hashes stale** (`sync.go:322`). If the `Updated YYYY-MM-DD HH:MM` stamp sits inside a `##` section rather than in the preamble, writing section A changes section B's hash, and the agent's next write to B is refused as `stale`. It recovers with a retry, but the hashes it already holds can't be trusted. The test fixtures only put the stamp in the preamble.
   - **Stray `.flock` files** (`sync.go:132`). `locked` creates `.<file>.flock` before `load` checks the file is `type: session-sync`, so running `read` against an ordinary markdown file leaves this dot-file behind (ARCH-FUNERAL).
   - **Misleading test comment** (`sync_test.go:91`). It says "A legacy agent lock beside it changes nothing", but that test never writes a legacy lock.
   - **No tracked issue for removing `release`.** The follow-up appears only in the Log, with no issue number (ARCH-FUNERAL).

5. **Test coverage notes**
   - All Done-when rows are pinned:
     - a section write succeeds over an edit elsewhere and a stale hash is refused;
     - two processes race and exactly one wins;
     - a turn with no `release` leaves no lock;
     - a write under the operator lock refuses and changes nothing.
   - The `couchcmd` CLI tests for read text/JSON output, stale exit 6 with the hash on stdout, and the `release` no-op were checked by reading them only, because their package won't build here.
   - The race between parley and Couch (the operator taking `.lock` after the guard check but before `WriteFile`) still has only the after-the-fact warning and no test that can control the ordering. This was already true before this change.

6. **Architectural notes**
   - **ARCH-DRY:** pass (shared `section()`).
   - **ARCH-PURE:** pass. `Hash`, `SectionBody` and `ReplaceSection` are pure and tested without IO; `locked` and `load` are the thin IO layer.
   - **ARCH-PURPOSE:** pass. Every listed consumer (code, CLI, SKILL, atlas, README) moved to `read` + `--if-hash`. The ops `xx-tl` draft is recorded as the TL's job, which the Spec allows.
   - **ARCH-MOCK:** not affected; no new external dependency (the `diff` binary was already used).
   - **ARCH-CONSTRAINTS:** pass. One flock per file, and `status` serialisation is cheap.
   - **ARCH-SECURE:** pass. The lock body is parsed defensively (anything unreadable counts as the operator's lock), and `.flock` is created with mode 0600.
   - **ARCH-ORDER:** pass with a note. Couch-to-Couch ordering is now serialised and tested. Parley-to-Couch ordering still depends on the operator's `.lock` plus the post-write warning; this is a known limit and should be recorded in parley.nvim#315.
   - **ARCH-FUNERAL:** pass with the Minor notes above. `.flock` is one file per session-sync file, removed along with the shadow and sidecar.

7. **Plan revision recommendations:** none required. Optionally add a Revisions note that `release` and `--accept-gap` with nothing to write now need a no-change `write`.

```findings
findings:
  - id: new
    severity: Minor
    family: retired-verb-silent-success
    title: |
      release no-op exits 0 even with --fold, so an old skill believes events were folded
    detail: |
      session_sync.go release branch prints only to stderr and returns 0; folds and the old auto-fold of submits silently do not happen. Consider exiting non-zero when --fold is passed.
  - id: new
    severity: Minor
    family: fold-requires-content-write
    title: |
      Folding a submit or accepting a delivery-gap now requires a non-empty write
    detail: |
      With release gone, an agent with nothing to change must rewrite a section with the same content to fold or accept a gap; document this in SKILL.md or add a fold-only form.
  - id: new
    severity: Minor
    family: write-side-effects-outside-target
    title: |
      Restamp can change another section's hash, making the agent's own held hash stale
    detail: |
      If the Updated stamp lives inside a section, writing section A changes section B's hash; fixtures only put the stamp in the preamble.
  - id: new
    severity: Minor
    family: residue-before-validation
    title: |
      locked() creates .<file>.flock before checking the file is type session-sync
    detail: |
      Running read/status against an ordinary markdown file leaves a stray dot-file behind.
  - id: new
    severity: Minor
    family: test-comment-drift
    title: |
      TestWriteUnderAnOperatorLockRefusesAndChangesNothing comment mentions a legacy agent lock it never writes
  - id: new
    severity: Minor
    family: untracked-removal-followup
    title: |
      Removing the release no-op is recorded only in the Log, with no tracked issue
```

---

## Re-review — 2026-10-11T09:34:56-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 439 — session-sync: optimistic per-section writes for the agent, no agent-held lock |
| repo | pair |
| issue file | workshop/issues/000439-session-sync-optimistic-per-section-writes-for-the-agent-no-agent-held-lock.md |
| boundary | whole-issue close |
| milestone | — |
| window | be20fd8d5a21101c65a0787d3a43799ae5b1db2f..89591551f1a40c6701f584ef8c1b327ac9426cda |
| command | sdlc close --issue 439 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T09:34:56-07:00 |
| verdict | SHIP |

## Review

Checking whether the old `release --accept-gap` had a lasting effect that the no-op now drops.

```verdict
verdict: SHIP
confidence: medium
```

The fix commit deals with five of the six earlier findings. One is partly open (BR-1), and it is Minor, so nothing blocks the gate.

**What I checked:**
- I ran the stat, name-status and full-patch commands on the window.
- Tests for `sessionsync` pass at the pinned head.
- I removed the BR-4 fix and ran its new test, `TestAWrongFileLeavesNoFlockBehind`. It fails without the fix, so the regression check works.
- I could not run the `couchcmd` tests here. The `runtimebundle` embed setup fails because the generated assets are missing from this checkout, as in the earlier round. I checked the new `release --fold` test by reading it.

**Scope:** The issue file changed only in its Log and frontmatter (status, flow, hours). `## Spec` and `## Done when` are unchanged. #440 is a new issue file. Nothing was added that no earlier finding asked for.

**The partial finding (BR-1):** `release --fold` now refuses, but the rule behind BR-1 is broader. That rule appears in the new lesson in `workshop/lessons.md`: "refuse those [arguments] whose effect it silently drops." Two old effects are still dropped while `release` exits 0:
- **`--accept-gap`:** the old `Release` saved the gap acceptance (`GapAccepted`). It is still accepted as a no-op flag and still exits 0 (`session_sync.go:48`). An old skill thinks the gap was accepted, and its next `status` refuses with `delivery-gap`.
- **Automatic fold of submits:** the old bare `release` folded every owed submit. It now folds nothing. Exit 0 here is the stated compatibility choice for bare `release`, so this is a note rather than a demand.

The fix is to refuse `--accept-gap` on `release` too, keeping only `--accept-edits`. That flag no longer means anything, since the unread-edits guard is gone, so a no-op is honest for it.

1. **Strengths**
   - `locked()` now calls `load()` before it creates the flock file (`sync.go:129-132`). That reuses the full check ("exists" and `type: session-sync`) instead of adding a second one (ARCH-DRY), and the test catches a regression.
   - The `release --fold` refusal names the command to use instead, `write --fold ID --if-hash H`, and the CLI test checks for that text.
   - BR-6 now has a tracked issue, #440, with a concrete Done-when.
   - The SKILL doc covers the gaps in prose: how to fold when nothing changes (write the section back as `read` printed it), and that the timestamp update can make a held hash go `stale`.

2. **Critical:** none.
3. **Important:** none.
4. **Minor:** BR-1 is still partly open; see above.
5. **Test coverage:** BR-4's regression test was checked by mutation and it works. BR-1's test covers only `--fold`; nothing shows that `release --accept-gap` refuses.
6. **Architecture**
   - **ARCH-DRY:** pass.
   - **ARCH-PURE:** pass; no logic was added to the IO layer.
   - **ARCH-PURPOSE:** flagged. BR-1 was fixed at the named `--fold` site, but the rule stated in the lesson was not applied to `--accept-gap`, a flag of the same kind.
7. **Plan revisions:** none.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Only --fold refuses; release --accept-gap still exits 0 though the old Release persisted GapAccepted (also old bare-release submit auto-fold is dropped). Apply the lesson's rule: refuse every effect flag except --accept-edits.
  - id: BR-2
    disposition: addressed
    note: |
      SKILL.md:278-279 documents folding by writing the section back as read printed it; write accepts a non-empty unchanged body.
  - id: BR-3
    disposition: addressed
    note: |
      Documented in SKILL.md:280-282 (restamp in a section makes its held hash stale; read again); by-design stale retry recorded in issue Log.
  - id: BR-4
    disposition: addressed
    note: |
      locked() runs load() before creating .flock (sync.go:129-132); TestAWrongFileLeavesNoFlockBehind goes red when the precheck is reverted.
  - id: BR-5
    disposition: addressed
    note: |
      sync_test.go:91 comment now describes the read-under-operator-lock assertion.
  - id: BR-6
    disposition: addressed
    note: |
      workshop/issues/000440 tracks removal of the release verb with Spec and Done-when.
```
