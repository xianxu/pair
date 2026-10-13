# Boundary Review — pair#500 (whole-issue close)

| field | value |
|-------|-------|
| issue | 500 — roll out ariadne#309 workflows (local-gates-guard, merge-check on push) to pair |
| repo | pair |
| issue file | workshop/issues/000500-roll-out-ariadne-309-workflows-local-gates-guard-merge-check-on-push-to-pair.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3f11a481abd83091971f3c8d921a8aaec874d157..73fc04ce0c455153a07e6bdb8ee55d446695c209 |
| command | sdlc close --issue 500 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T14:34:40-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This change does what the Spec asks. It adds `.github/workflows/local-gates-guard.yml` and updates `.github/workflows/merge-check.yml`, and both files at the head commit are byte-identical to ariadne's `origin/main` copies and to the pair-slot6/ariadne checkout they were seeded from. I checked that with `cmp`; the commit message was not my evidence. Nothing in the window goes beyond those two files. Nothing blocks shipping.

1. **Strengths**
   - **The Spec's "byte-identical, no edits" promise holds.** Both files match ariadne `origin/main` and `worktree/pair-slot6/ariadne` exactly. The `../ariadne` checkout and the other slots still have the older `merge-check.yml`, which is consistent with the catch-up only having reached pair-slot6 so far.
   - **The guard keeps PR data away from code it runs.**
     - It uses `pull_request_target`, which runs main's copy of the workflow, and never checks out the PR's code.
     - PR values reach the script through `env:` rather than being pasted into it, and the PR number and head SHA are checked before use (`local-gates-guard.yml:72-73`).
     - Status writes are a pending-then-trap pattern: the status is set to pending first, and any exit before a decision turns it into a failure.
   - **The push branch of `merge-check.yml` handles a brand-new ref.** When the push has no previous commit (an all-zero `before`), it exits cleanly instead of failing on `merge-base` (`merge-check.yml:53-56`). It also moved the old `${{ }}` text pasted into the script over to `env:` variables.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - The guard's comment points readers to `atlas/workflow/ci-merge-check.md`, which doesn't exist in pair's tree. The doc lives in ariadne, so pair readers follow a dead path. This is ariadne's content and the Spec forbids edits, so don't change it here.
   - After this merges, any pair PR that touches `.github/workflows/*` will need the operator's `local-gates-verified` label. That is intended, but pair's issue doesn't mention it as an operational consequence.

5. **Test coverage notes:** The change is copied workflow config with no logic owned by pair, so it needs no pair-side tests. The behavior is ariadne#309's to test. The "a slot's catch-up leaves its tree clean" outcome follows from the byte-identity, but I didn't rerun a catch-up.

6. **Architecture pass/flag**
   - ARCH-DRY: pass. The files are seeded copies of ariadne's single source; nothing was hand-edited to diverge.
   - ARCH-PURE: not applicable (workflow config).
   - ARCH-PURPOSE: pass. Both seeded files are committed, which is the whole goal.
   - ARCH-MOCK: not applicable to pair; any fakes for the `gh` calls are ariadne's.
   - ARCH-CONSTRAINTS: pass. The concurrency group cancels older runs per PR.
   - ARCH-SECURE: pass. PR values go through `env:`, are validated, and PR code is never run.
   - ARCH-ORDER: pass. A newer run cancels an older one, and the moved-head and pending/trap cases are handled explicitly.
   - ARCH-FUNERAL: pass. The only things created are commit statuses, which GitHub owns.

7. **Plan revision recommendations:** none.

```findings
findings:
  - id: new
    severity: Minor
    family: cross-repo-doc-reference
    title: |
      local-gates-guard.yml comment cites atlas/workflow/ci-merge-check.md, absent in pair's tree
    detail: |
      The referenced doc lives in ariadne; per Spec this file must stay byte-identical, so fix (if any) belongs upstream in ariadne.
```
