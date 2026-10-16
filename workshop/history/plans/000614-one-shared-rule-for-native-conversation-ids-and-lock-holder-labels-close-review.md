# Boundary Review — pair#614 (whole-issue close)

| field | value |
|-------|-------|
| issue | 614 — One shared rule for native conversation IDs and lock holder labels |
| repo | pair |
| issue file | workshop/issues/000614-one-shared-rule-for-native-conversation-ids-and-lock-holder-labels.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5144c2be72da66027818157f3805bf4549074975..61066dc88570e5b010925faf1ef02fcefc028231 |
| command | sdlc close --issue 614 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T07:48:16-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I've read the diff and the issue. Now I'm running the affected package tests.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

This change does what the issue asks for the conversation ID. A new leaf package, `nativeid`, holds the single rule. Both places that check an ID now call it: the lock key in `artifactpath.NativeLockFilename` and the binding notice in `sessionwatch.validBindingNotice`. The cross-boundary test `TestBindingNoticeIDRuleMatchesLockKey` fails without the fix: I reverted `binding_notice.go` and it went red on "has space", the zero-width character, "a/b" and "..". With the fix in place, the tests for nativeid, sessionwatch, artifactpath, nativelock and wrapcmd all pass. IDs that were already valid behave exactly as before. Nothing here blocks shipping. The open points are cheap:
- The holder-label rule is still a separate copy of the same character check.
- The holder parser has no test for the space or Cf (invisible format character) cases that "Done when" asks for.
- The issue's Plan section is still empty.
- The atlas owners list doesn't mention the new package.

1. **Strengths**
   - `cmd/internal/nativeid/id.go:20`: a small, pure, leaf package with no IO. Keeping it a leaf means `sessionwatch` doesn't depend on `artifactpath` and stays a non-artifact source (it is registered in `manifest.go:635`).
   - `binding_notice_test.go:74`: the test checks the property that matters ("the notice accepts exactly what the lock key accepts") instead of copying a list of inputs. It also round-trips every accepted ID.
   - The old ID check in `native_lock.go` was removed, not left beside the new one, so there is no leftover duplicate.
   - Including the 512- and 513-byte cases in both tables pins the length limit exactly.

2. **Critical**: none.

3. **Important**: none.

4. **Minor**
   - **ARCH-DRY:** `nativelock/key.go:41-45` (`validHolder`) repeats the character check from `nativeid.Validate:24-28` (`IsControl || IsSpace || Cf`). The Spec asks for "one for a holder label" too. It is a single rule today, but the shared part should be exported once, for example `nativeid.ForbiddenRune(r) bool`, and reused by `validHolder`.
   - **Tests:** "Done when" asks for space, Cf and oversize cases at each boundary. `TestHolderMetadataCannotChangeKernelDecision` (`lock_test.go:156`) only tests nil, a newline, an escape sequence and 1025 bytes. It has no space or Cf case for `ParseHolder`.
   - **Plan:** the issue's `## Plan` section is still the placeholder `- [ ]`. The "Done when" claim of "all three sites" is really two shared sites plus an existing holder rule. That scope decision should be written down.
   - **Atlas:** the owners list in `atlas/couch.md:4108` should add `nativeid/` as the place the ID rule lives.

5. **Test coverage notes:** the new tests use no IO and don't run without the fix. The sessionwatch test calls the real `artifactpath` function, so the two rules can't drift apart without it failing.

6. **Architecture**
   - **ARCH-DRY:** flagged above (character check repeated in `validHolder`).
   - **ARCH-PURE:** pass.
   - **ARCH-PURPOSE:** pass for the conversation ID. I looked for other places that check a native ID. `couchmessage` `CandidateKey` is a selector key, not this ID. Its rule is looser, so it could be future work, but it isn't in scope here.
   - **ARCH-MOCK:** pass, as no external dependency is involved.
   - **ARCH-CONSTRAINTS:** pass, since the check is bounded by the 512-byte limit.
   - **ARCH-SECURE:** pass, and an improvement. The notice coming in over the pipe is now checked against the full path and invisible-character rules.
   - **ARCH-ORDER:** pass, because the change carries no state between events.
   - **ARCH-FUNERAL:** pass, because it creates nothing that persists.

7. **Plan revisions:** fill in `## Plan` with what was delivered. Add a Revisions note that the holder label keeps its own rule (`nativelock.validHolder`, 1024 bytes, path characters allowed) and shares only the character check, if that gets extracted.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: duplicated-id-charset-rule
    title: |
      validHolder repeats the forbidden-character check that nativeid.Validate defines
    detail: |
      nativelock/key.go:41-45 copies the IsControl||IsSpace||Cf loop from nativeid/id.go:24-28 (ARCH-DRY). Export one predicate from nativeid and reuse it so the holder rule cannot drift from the ID rule.
  - id: new
    severity: Minor
    kind: defect
    family: done-when-boundary-test-coverage
    title: |
      The holder parser has no space or Cf test cases, which "Done when" requires at each boundary
    detail: |
      TestHolderMetadataCannotChangeKernelDecision covers nil, a newline, an escape sequence and 1025 bytes, but no space or zero-width (Cf) holder; add these to pin ParseHolder against the shared rule.
  - id: new
    severity: Minor
    kind: defect
    family: plan-not-recorded
    title: |
      The issue's Plan section is still an empty placeholder and the atlas owners list omits nativeid
    detail: |
      Record the delivered plan and the decision to keep the holder rule separate; add nativeid/ to the owners line at atlas/couch.md:4108.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"b5934be2-4f83-43be-859d-8d3edff5479e","repository":"github.com/xianxu/pair","issue_id":"000614","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/e70f6eae8c02f954","reviewed_head":"61066dc88570e5b010925faf1ef02fcefc028231","slot":"pair:2","requested_at":"2026-10-16T14:45:30.459853Z"},"phase":"finished","pid":48202,"started_at":"2026-10-16T14:45:30.627291Z","finished_at":"2026-10-16T14:48:04.99046Z","session_id":"b5837cc5-3baa-4d3b-97a7-4f6cef2d6bfb","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.806106459s
head: "61066dc88570e5b010925faf1ef02fcefc028231"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
