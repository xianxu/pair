---
gate: boundary-review
issue: 498
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T14:05:22-07:00"
      dispatched: "2026-10-12T14:03:42-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: setpriority uses an absolute value, so a COUCH_SLOT_NICE below Couch's own niceness makes every slot launch fail
          detail: 'launchhelper.go:58 calls Setpriority(PRIO_PGRP, pid, nice). Non-root users cannot lower niceness, so with Couch at nice 15 (or nice 10 with COUCH_SLOT_NICE=5) every StartBlocked returns EPERM and cancels. Fix: apply max(nice, current) via unix.Getpriority, or skip when the current niceness is already at least the target; add a test where the requested niceness is below the inherited one. Same family: the README line "Couch itself keeps normal priority" assumes Couch runs at nice 0.'
          family: absolute-nice-below-inherited
          round: 1
        - id: BR-2
          severity: Important
          title: Plan Design paragraph still names a couchcore.NicedRunner decorator that does not exist
          detail: The code puts a Nice field on ExecRunner/PtyRunner (runner.go:58, ptyrunner.go:26). Only the Log records the change; the Design text and Revisions do not.
          family: plan-contradicts-code
          round: 1
        - id: BR-3
          severity: Important
          title: Done when still requires a live ps check and measured latency that Revisions deferred until after the restart
          detail: Rewrite Done when so the in-repo grandchild ps proof closes the item and the live verification follows the operator's restart, or record those results before close.
          family: done-when-stale
          round: 1
        - id: BR-4
          severity: Minor
          title: couchcmd nice_test reads the real os.Getenv and uses an unchecked type assertion
          detail: nice_test.go:43 fails if COUCH_SLOT_NICE is set in the developer's shell, and panics instead of failing on a type mismatch. Inject an empty getenv and use the comma-ok form.
          family: test-ambient-env-dependence
          round: 1
        - id: BR-5
          severity: Minor
          title: couchcore nice tests need /bin/ps, which cannot run in restricted environments
          detail: ownNice (nice_test.go:72) could use unix.Getpriority with no exec. The grandchild probe could re-exec the test binary to read its own priority.
          family: test-ambient-env-dependence
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T14:13:41-07:00"
      dispatched: "2026-10-12T14:11:34-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: EACCES/EPERM now keep inherited nice; a reviewer ps-free test (Nice -1) is green with the fix and red when the exemption at launchhelper.go:64 is reverted.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The Design paragraph now names the Nice field on ExecRunner/PtyRunner applied in startBlockedChild; Revisions records the dropped NicedRunner.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Done when splits into the in-repo grandchild ps proof and a post-restart hub-owned live check; Revisions records the move.
          round: 2
        - id: BR-4
          disposition: addressed
          note: The test injects getenv via consoleTraceConfig and uses comma-ok assertions (couchcmd/nice_test.go:33,40,46).
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: 'Reproduced here: both couchcore nice tests fail with "fork/exec /bin/ps: operation not permitted". At minimum t.Skip when ps cannot exec; ownNice could use unix.Getpriority with a per-GOOS offset.'
          round: 2
      findings:
        - id: BR-6
          severity: Important
          title: atlas and Plan checklist still say a refused setpriority cancels the launch; code now keeps the inherited niceness
          detail: '2nd finding in plan-contradicts-code. Rule: a fix round that changes a behaviour must grep and rewrite every prose restatement of it in the same round. Places: atlas/couch.md:2370-2371 ("a refused setpriority cancels the launch before exec"); Plan checklist item 1 ("a kernel-refused niceness that cancels before the target runs"); the Log mutation-check line "refused niceness returns a handle" (pre-fix, annotate as superseded). README, Design, and the startBlockedChild comment already match the code.'
          family: plan-contradicts-code
          round: 2
        - id: BR-7
          severity: Minor
          title: The non-EACCES/EPERM setpriority cancel branch (launchhelper.go:65-70) has no test
          detail: This is the only remaining path that cancels a launch over niceness; no fixture reaches it.
          family: untested-error-branch
          round: 2
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-12T14:14:52-07:00"
      dispatched: "2026-10-12T14:14:13-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: not-addressed
          note: 'ownNice (couchcore/nice_test.go:79) still runs /bin/ps; in this sandbox it failed with "fork/exec /bin/ps: operation not permitted". unix.Getpriority(PRIO_PROCESS, 0) would remove the exec for the parent; the grandchild probe still needs ps or a re-exec of the test binary.'
          round: 3
        - id: BR-6
          disposition: addressed
          note: atlas/couch.md:2367-2373 now says EACCES/EPERM keeps the inherited niceness and other errors cancel, matching launchhelper.go:61-70. Plan item 1 was rewritten with a close-round-1 note and the Log mutation line is annotated as superseded. README, Design and the doc comment already agreed. A grep of atlas/ and README for nice+cancel found no other stale statements.
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: The non-EACCES/EPERM cancel branch (launchhelper.go:63-70) is still untested. Reaching it needs an injectable setpriority seam, e.g. ESRCH on a helper that has already exited.
          round: 3
      recipe: small-diff-review
      isolation: fenced
      reviewed: 1d042f8a4552ef596fb9b5abf3defbfe149a7a76
      blocked: false
---

# Gate ledger — pair#498 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T14:05:22-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `absolute-nice-below-inherited` setpriority uses an absolute value, so a COUCH_SLOT_NICE below Couch's own niceness makes every slot launch fail
  launchhelper.go:58 calls Setpriority(PRIO_PGRP, pid, nice). Non-root users cannot lower niceness, so with Couch at nice 15 (or nice 10 with COUCH_SLOT_NICE=5) every StartBlocked returns EPERM and cancels. Fix: apply max(nice, current) via unix.Getpriority, or skip when the current niceness is already at least the target; add a test where the requested niceness is below the inherited one. Same family: the README line "Couch itself keeps normal priority" assumes Couch runs at nice 0.
- **BR-2** [Important] `plan-contradicts-code` Plan Design paragraph still names a couchcore.NicedRunner decorator that does not exist
  The code puts a Nice field on ExecRunner/PtyRunner (runner.go:58, ptyrunner.go:26). Only the Log records the change; the Design text and Revisions do not.
- **BR-3** [Important] `done-when-stale` Done when still requires a live ps check and measured latency that Revisions deferred until after the restart
  Rewrite Done when so the in-repo grandchild ps proof closes the item and the live verification follows the operator's restart, or record those results before close.
- **BR-4** [Minor] `test-ambient-env-dependence` couchcmd nice_test reads the real os.Getenv and uses an unchecked type assertion
  nice_test.go:43 fails if COUCH_SLOT_NICE is set in the developer's shell, and panics instead of failing on a type mismatch. Inject an empty getenv and use the comma-ok form.
- **BR-5** [Minor] `test-ambient-env-dependence` couchcore nice tests need /bin/ps, which cannot run in restricted environments
  ownNice (nice_test.go:72) could use unix.Getpriority with no exec. The grandchild probe could re-exec the test binary to read its own priority.

## Round 2 — 2026-10-12T14:13:41-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — EACCES/EPERM now keep inherited nice; a reviewer ps-free test (Nice -1) is green with the fix and red when the exemption at launchhelper.go:64 is reverted.
- BR-2 — addressed — The Design paragraph now names the Nice field on ExecRunner/PtyRunner applied in startBlockedChild; Revisions records the dropped NicedRunner.
- BR-3 — addressed — Done when splits into the in-repo grandchild ps proof and a post-restart hub-owned live check; Revisions records the move.
- BR-4 — addressed — The test injects getenv via consoleTraceConfig and uses comma-ok assertions (couchcmd/nice_test.go:33,40,46).
- BR-5 — not-addressed — Reproduced here: both couchcore nice tests fail with "fork/exec /bin/ps: operation not permitted". At minimum t.Skip when ps cannot exec; ownNice could use unix.Getpriority with a per-GOOS offset.

### Raised

- **BR-6** [Important] `plan-contradicts-code` atlas and Plan checklist still say a refused setpriority cancels the launch; code now keeps the inherited niceness
  2nd finding in plan-contradicts-code. Rule: a fix round that changes a behaviour must grep and rewrite every prose restatement of it in the same round. Places: atlas/couch.md:2370-2371 ("a refused setpriority cancels the launch before exec"); Plan checklist item 1 ("a kernel-refused niceness that cancels before the target runs"); the Log mutation-check line "refused niceness returns a handle" (pre-fix, annotate as superseded). README, Design, and the startBlockedChild comment already match the code.
- **BR-7** [Minor] `untested-error-branch` The non-EACCES/EPERM setpriority cancel branch (launchhelper.go:65-70) has no test
  This is the only remaining path that cancels a launch over niceness; no fixture reaches it.

## Round 3 — 2026-10-12T14:14:52-07:00 (claude) — passed

### Disposed

- BR-5 — not-addressed — ownNice (couchcore/nice_test.go:79) still runs /bin/ps; in this sandbox it failed with "fork/exec /bin/ps: operation not permitted". unix.Getpriority(PRIO_PROCESS, 0) would remove the exec for the parent; the grandchild probe still needs ps or a re-exec of the test binary.
- BR-6 — addressed — atlas/couch.md:2367-2373 now says EACCES/EPERM keeps the inherited niceness and other errors cancel, matching launchhelper.go:61-70. Plan item 1 was rewritten with a close-round-1 note and the Log mutation line is annotated as superseded. README, Design and the doc comment already agreed. A grep of atlas/ and README for nice+cancel found no other stale statements.
- BR-7 — not-addressed — The non-EACCES/EPERM cancel branch (launchhelper.go:63-70) is still untested. Reaching it needs an injectable setpriority seam, e.g. ESRCH on a helper that has already exited.

## Open findings

- **BR-5** [Minor] `test-ambient-env-dependence` couchcore nice tests need /bin/ps, which cannot run in restricted environments
- **BR-7** [Minor] `untested-error-branch` The non-EACCES/EPERM setpriority cancel branch (launchhelper.go:65-70) has no test
