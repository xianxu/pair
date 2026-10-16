# Boundary Review — pair#600 (whole-issue close)

| field | value |
|-------|-------|
| issue | 600 — Fresh reboot onto a new address fails recovery authority: SourceRevision names the old record |
| repo | pair |
| issue file | workshop/issues/000600-fresh-reboot-onto-a-new-address-fails-recovery-authority-sourcerevision-names-the-old-record.md |
| boundary | whole-issue close |
| milestone | — |
| window | c9f60c6aced376d7120df4bdc2c4e48e6d621d67..ae4f827e4fd4cb1e6d9cd23ca77ce35b0d4105e1 |
| command | sdlc close --issue 600 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-15T23:18:31-07:00 |
| verdict | SHIP |

## Review

I've read the diff and the replace invariant. Next I'm checking that `next.Revision` inside the Commit callback is the revision that actually gets persisted, then I'll run the tests.

Tests pass at head. Now I'm reverting the fix in this disposable checkout to confirm the regression tests go red without it.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** The fix does what the Spec asks. In the fresh-onto-new-address path, the intent's `SourceRevision` is now set to the new record's revision. The global store's replace check in `threadstore_replace.go:72` was updated the same way, so the producer and its checker still agree. The launcher's validation rule (`recovery_authority_os.go:37`) is unchanged, and no new refusals were added.

I checked the regression evidence myself in the disposable checkout. With both one-line fixes reverted, `TestFreshRecoveryOntoNewAddressPassesLauncherValidation` (slot-local store) fails with the exact live error. So does `TestRebootPrimaryUsesSharedFreshJournalAndNotice` (global store):

- slot-local: "explicit recovery source revision, time or agent changed (source revision 5, record revision 1)"
- global: "… (source revision 4, record revision 1)"

With the fixes restored, `couchcore`, `launcher` and `threadrecord` all pass, and the checkout is back to a clean tree. Nothing blocks SHIP; there is one Minor duplication note.

**Strengths**
- The tests run the real pure check, `launcher.ValidateRecoveryRecord`, on the authority Couch actually builds, read back from the record file on disk (`recovery_execution_test.go:131-157`). This is the gap the issue blamed for the miss, and it is now closed for both the slot-local and global store paths.
- The change is minimal and consistent: it touches only the fresh-address Commit callback (`recovery_prepared.go:478`) and the matching replace check (`threadstore_replace.go:72`). Resume and relaunch paths still use `SourceRevision: record.Revision` from line 243 and are unaffected. The inferred-path `source.Revision = intent.SourceRevision` (line 370) is never reached for Fresh.
- `next.Revision` really is the persisted value. `claimFreshRecord` builds the record with `Revision: 1` (`fresh_claim.go:52`), and later fences only raise `RecordRevision`, so `SourceRevision ≤ RecordRevision` holds.
- The old record's revision is still kept for audit in `SelectedKey` (`Fresh:<tag>:<rev>`), so provenance isn't lost.

**Critical findings:** none.

**Important findings:** none.

**Minor findings**
- ARCH-DRY: the rule for moving a Fresh intent onto its new address is written out in two places, `recovery_prepared.go:478` and `threadstore_replace.go:72`. They must stay identical or the replace fails with "fresh replacement selection changed". A small helper, e.g. `freshIntentOnto(intent, next ThreadRecord) *RecoveryIntent`, used by both would make it one source of truth.
- `requireLauncherAcceptsRecovery` takes its PID and identity from the record it validates, so the helper-identity part of the check proves nothing. That's acceptable here, since the bug was in the revision, time and agent part.

**Test coverage notes**
- I confirmed both regression tests fail without the fix and pass with it (above).
- The slot-local replace (`replaceSlotCurrent`) has no intent cross-check, so the producer change alone covers it there. The slot-local test covers that path.

**Architecture**
- ARCH-DRY: flag (Minor, above).
- ARCH-PURE: pass. The exported validator is pure, and the tests call it without IO fakes.
- ARCH-PURPOSE: pass. Both store paths are covered, and "drop the fake if cheap" was handled by running the real pure check inside the fake runner's acknowledge hook.
- ARCH-MOCK: pass. The fake `ValidateRecoveryAuthority` remains, but its blind spot is now backed by the production pure check.
- ARCH-CONSTRAINTS: N/A. No change to the runtime envelope.
- ARCH-SECURE: pass. `SourceRevision` now names the record the intent is actually committed into, and the launcher's guard against a source newer than that record is unchanged.
- ARCH-ORDER: pass. The intent is rewritten inside the same journaled Commit, so a crash between replace and launch leaves a consistent new record. A retried reboot rebuilds the intent from the current record (line 243 and `PreserveRecoveryBeforeRetirement`).
- ARCH-FUNERAL: N/A. Nothing durable is added; only one field's value changes.

**Docs:** `ValidateRecoveryRecord` is exported inside `internal/` only, so there's no user-facing surface and no atlas or README change is needed.

**Plan revisions:** none. The plan matches the code.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: duplicated-intent-rebinding-rule
    title: |
      Fresh-intent move to the new address is duplicated in recovery_prepared.go:478 and threadstore_replace.go:72
    detail: |
      ARCH-DRY: the rule (address := next.Address, source revision := next.Revision) is written in two places that must stay identical or replace fails with "fresh replacement selection changed"; extract a shared helper used by both.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"726521d2-b275-4075-9b11-a12bdff62a39","repository":"github.com/xianxu/pair","issue_id":"000600","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/f0d73105afb893e6","reviewed_head":"ae4f827e4fd4cb1e6d9cd23ca77ce35b0d4105e1","slot":"pair:2","requested_at":"2026-10-16T06:12:21.830076Z"},"phase":"finished","pid":37825,"started_at":"2026-10-16T06:12:21.927433Z","finished_at":"2026-10-16T06:18:31.407083Z","session_id":"0fc5e4f3-6896-40a0-a96d-77fd7c792bc8","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.277757833s
head: "ae4f827e4fd4cb1e6d9cd23ca77ce35b0d4105e1"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
